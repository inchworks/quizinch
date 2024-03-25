// Copyright © Rob Burke inchworks.com, 2019.

// This file is part of QuizInch.
//
// QuizInch is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// QuizInch is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with QuizInch.  If not, see <https://www.gnu.org/licenses/>.

// Processing related to quiz execution and scoring

package quiz

import (
	"errors"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/inchworks/webparts/etx"

	"inchworks.com/quiz/internal/forms"
	"inchworks.com/quiz/internal/models"
)

type scoreStatus struct {
	teams     int // teams scored
	confirmed bool
}

func (s scoreStatus) setConfirmed(c bool) scoreStatus {
	s.confirmed = c
	return s
}

func (s scoreStatus) setScored(nTeams int) scoreStatus {
	s.teams = nTeams
	s.confirmed = false
	return s
}

// ## Doesn't need to be an object? No state between functions. Could include mutex and dirty flag here.

type QuizState struct {
	app             *Application
	muQuiz          sync.RWMutex
	rollbackTx      bool

	publishedUpdate int // update to published scores
	responseUpdate  int // update to team responses
	scorerUpdate    int // update to unpublished scores

	// cached durable state
	quizCached    *models.Quiz
	dirtyQuiz     bool

	// cached scoring: round no -> question ID -> scoreStatus
	scored map[int]map[int64]scoreStatus

	// cached settings
	nDeferAnswers int // rounds of deferred answers and scored
	nFullRounds   int // rounds excluding tie-breakers
	nTieRounds    int // usable tie-breaker rounds
}

// Init initialises the quiz state
func (qs *QuizState) Init(qd *models.Quiz, nRounds int) {
	qs.quizCached = qd
	qs.setNumRounds(nRounds, qd.NTieBreakers, qd.NDeferred)
	qs.changedAll()
	qs.scored = make(map[int]map[int64]scoreStatus, nRounds)
}

// Begin implements the DB interface for uploader.
func (q *QuizState) Begin() func() {

	return q.updatesQuiz()
}

// Calculate totals and rank for teams, up to the specified round
//
// Note that this safe to call when stepping back, as well as forward, and also to recalculate after revising scores.

func (qs *QuizState) calculateTotalsAndRank(nRound int, s *models.Contest) {

	// ranking algorithm depends on round type
	if nRound <= qs.nFullRounds {
		qs.calcOnFullRound(nRound)
	} else {
		qs.calcOnTieBreak(nRound)
	}

	// make changes visible to quizmaster (and later to scoreboard)
	if nRound != s.QuizmasterRound {
		s.QuizmasterRound = nRound
	}

	// set revision numbers for score and quizmaster
	qs.changedPublished()
	qs.app.displayState.changedQuizmaster()
}

// calcOnFullRound performs ranking for all teams, saving the team totals.
// It also checks if we have a tie on the final round.
func (qs *QuizState) calcOnFullRound(nRound int) {

	// modifies display state
	ds := &qs.app.displayState

	// rechecks for a tie
	ds.isTied = false

	// rank teams by scores
	rank := 0
	priorScore := -1.0

	// get total for each team, in order
	teamTotals := qs.app.TeamStore.AllWithTotals(nRound)
	for nTeam, teamTotal := range teamTotals {

		if teamTotal.Value == priorScore {
			// do we have a tie on the final round?
			if nRound == qs.nFullRounds && rank <= qs.quizCached.NWinners {
				ds.isTied = true
			}
		} else {

			// advance rank
			rank = nTeam + 1
			priorScore = teamTotal.Value
		}

		// set new rank and total for team, so we needn't compute them again
		teamTotal.Team.Rank = rank
		teamTotal.Team.Total = teamTotal.Value

		// save update
		if err := qs.app.TeamStore.Update(&teamTotal.Team); err != nil {
			panic(err)
		}
	}
}

// calcOnTieBreak performs ranking for scored teams.
func (qs *QuizState) calcOnTieBreak(nRound int) {

	// modifies display state
	ds := &qs.app.displayState

	// rechecks for a tie
	ds.isTied = false

	// Note that taking part in a tie-break leaves the rank unchanged for the topmost team(s),
	// and increases the rank for lower teams. For example, four teams tied in 2nd place might
	// become ranked 2, 3, 3, 5.
	delta := -1
	priorScore := -1.0

	// get round score for each scored team, in order
	teamScores := qs.app.TeamStore.AllScoredWithScores(nRound)
	for i, teamScore := range teamScores {

		if teamScore.Value == priorScore {
			// do we have a tie for top round score?
			if delta == 0 {
				ds.isTied = true
			}
		} else {
			// advance rank
			delta = i
			priorScore = teamScore.Value
		}

		// set new rank for team
		teamScore.Team.Rank += delta

		// save update
		if err := qs.app.TeamStore.Update(&teamScore.Team); err != nil {
			panic(err)
		}
	}
}

// forEditResponses returns the data to set responses.
// We don't trust the client.
// The team UID has been validated against the access code, to prevent a team from viewing responses from other teams.
// A team is allowed to view their responses from all rounds.
func (qs *QuizState) forEditResponses(nRound int, nTeam int, token string) (*responsesFormData, error) {

	teamId := int64(nTeam)

	// serialisation
	defer qs.updatesNone()()

	// round
	round, err := qs.app.RoundStore.GetByN(nRound)
	if err != nil {
		return nil, err
	}

	// team
	team, err := qs.app.TeamStore.Get(teamId)
	if err != nil {
		return nil, err
	}

	// responses
	qrs := qs.app.QuestionStore.ForTeamRound(teamId, round.Id)

	// form, with sub-forms
	f := forms.NewResponses(make(url.Values), token)

	for i, qr := range qrs {
		f.Add(i, qr)
	}

	// template fields
	d := responsesFormData{
		NRound:    nRound,
		NTeam:     nTeam,
		Round:     round.Title,
		Team:      team.Name,
		Responses: f,
	}

	return &d, nil
}

// setNumRounds sets the round numbers, limited by the rounds that have been created.
func (qs *QuizState) setNumRounds(nRounds int, nTieBreakers int, nDeferred int) {

	if nRounds == 0 {
		// no rounds at all
		qs.nFullRounds = 0
		qs.nTieRounds = 0

	} else if nRounds <= nTieBreakers {
		// must have at least one full round
		qs.nFullRounds = 1
		qs.nTieRounds = nRounds - 1

	} else {
		// rounds as specified
		qs.nFullRounds = nRounds - nTieBreakers
		qs.nTieRounds = nTieBreakers
	}

	if nDeferred >= qs.nFullRounds {
		// must score after final round
		qs.nDeferAnswers = qs.nFullRounds - 1

	} else {
		// defer scoring as specified
		qs.nDeferAnswers = nDeferred
	}
}

// onResponses saves a team's answers for a round, and returns the round title.
// We don't trust the client.
// The team UID has been validated against the access code, to prevent a team from viewing responses from other teams.
// A team is allowed submit new responses for any prior round, as they may be catching up after a network outage.
// They are also allowed to change their responses for the current round.
func (qs *QuizState) onEditResponses(nRound int, nTeam int, rsSrc []*forms.Response) (string, error) {

	// ## should report details of client errors

	// serialisation
	defer qs.updatesQuiz()()

	// validate round number
	currentRound := qs.quizCached.ResponseRound
	if nRound > currentRound {
		return "", qs.rollbackErr("Wrong round number")
	}

	// round
	round, err := qs.app.RoundStore.GetByN(nRound)
	if err != nil {
		return "", err
	}

	// compare modified responses against current ones, and update
	teamId := int64(nTeam)
	rsDest := qs.app.QuestionStore.ForTeamRound(teamId, round.Id)

	if len(rsDest) != len(rsSrc) {
		return "", qs.rollbackErr("Corrupt form")
	}

	nResponses := 0
	for i, rSrc := range rsSrc {

		// full response record for insert/update
		// ## better to have QuestionResponse have the full Response for update?
		rDest := rsDest[i]
		response := models.Response{
			Question: rDest.QuestionId,
			Team:     teamId,
			Value:    rSrc.Value.String,
		}

		// existing response?
		if rDest.Value.Valid {
			// must be the current round
			// ## could allow organiser to fix earlier responses?
			if nRound != currentRound {
				return round.Title, qs.rollbackErr("Cannot change answers for an earlier round") // #### must explain error to user
			}

			response.Id = rDest.ResponseId.Int64 // update existing response
		}

		qs.app.ResponseStore.Update(&response)
		nResponses++
	}

	// count resposes for the scorers
	// #### ok that scores created earlier than before?
	sc, _ := qs.app.ScoreStore.ForTeamAndRound(teamId, nRound)
	if sc == nil {
		sc = &models.Score{
			Team:   teamId,
			NRound: nRound,
		}
	}
	sc.Responses = nResponses
	qs.app.ScoreStore.Update(sc)

	// refresh response displays
	qs.changedResponse()

	return round.Title, nil
}

// Get data to enter or edit scores

func (q *QuizState) forEditScores(action string, nRound int, token string) *scoresFormData {

	// serialisation
	defer q.updatesNone()()

	f := forms.NewScores(make(url.Values), token)

	// round title
	var title string
	if round, _ := q.app.RoundStore.GetByN(nRound); round != nil {
		title = round.Title
	}

	// scores with team names
	scores := q.app.TeamStore.AllWithScores(nRound)
	for i, s := range scores {

		// scores may be missing
		var st string
		if s.Value.Valid {
			st = strconv.FormatFloat(s.Value.Float64, 'f', -1, 64)
		} else {
			st = ""
		}
		f.Add(i, s.Name, st)
	}

	return &scoresFormData{
		Scores: f,
		Action: action,
		Round:  nRound,
		Title:  title,
	}
}

// Processing when scores are modified.
// ## Better in quiz manager, or keep with ForEditScores?

func (q *QuizState) onEditScores(nRound int, ssSrc []*forms.Score, edited bool) {

	// serialisation
	defer q.updatesQuiz()()

	// save modified scores
	// ## processing should be in QuizManager?
	ssDest := q.app.TeamStore.AllWithScores(nRound)
	for i, sSrc := range ssSrc {
		sDest := ssDest[i]

		if sSrc.Score != "" {

			// score specified
			// ## could check that team names match

			// full score record for insert/update
			v, _ := strconv.ParseFloat(sSrc.Score, 64)
			score := models.Score{NRound: nRound, Team: sDest.TeamId, Value: v}

			// existing score? otherwise new score will be created
			if sDest.ScoreId.Valid {
				score.Id = sDest.ScoreId.Int64
			}

			q.app.ScoreStore.Update(&score)

		} else if nRound >= q.quizCached.ScoringRound || nRound > q.nFullRounds {

			// unpublished round, or tie-break round - remove existing score
			if sDest.ScoreId.Valid {
				q.app.ScoreStore.DeleteId(sDest.ScoreId.Int64)
			}
		}
	}

	// re-rank published scores
	if edited {
		s, dUnlock := q.app.displayState.forUpdate()
		defer dUnlock()

		q.calculateTotalsAndRank(s.QuizmasterRound, s)
	}
}

// forRespondWait returns the rounds for which a response might be accepted.
func (qs *QuizState) forRespondWait(nTeam int) *dataRespondWait {

	teamId := int64(nTeam)

	// serialisation
	defer qs.updatesNone()()

	// team name
	var name string
	t, err := qs.app.TeamStore.Get(teamId)
	if err == nil {
		name = t.Name
	}

	// If the team has already responded, show them the next round too.
	// We do this in case their page does not auto-update, so that they can select the next round manually.
	nRound := qs.quizCached.ResponseRound
	limit := 1
	sc, _ := qs.app.ScoreStore.ForTeamAndRound(teamId, nRound)
	if sc != nil && sc.Responses > 0 {
		limit = 2
	}

	return &dataRespondWait{
		NTeam:       nTeam,
		Team:        name,
		Rounds:      qs.app.RoundStore.Current(nRound, limit),
		dataDisplay: dataDisplay{Index: nRound},
	}
}

// forScoreQuestion gets the data to enter or edit teams' scores for a question
func (q *QuizState) forScoreQuestion(nQuestion int, token string) (*scoreQuestionFormData, error) {

	// serialisation
	defer q.updatesNone()()

	f := forms.NewScoreQuestion(make(url.Values), token)

	// scores with team names
	rs := q.app.ResponseStore.ResponsesForQuestion(int64(nQuestion))
	for i, r := range rs {

		st := strconv.FormatFloat(r.Score, 'f', -1, 64)
		f.Add(i, r, st)
	}

	// question and round, for headings
	qu, err := q.app.QuestionStore.Get(int64(nQuestion))
	if err != nil {
		return nil, err
	}
	rd, err := q.app.RoundStore.Get(qu.Round)
	if err != nil {
		return nil, err
	}

	td := &scoreQuestionFormData{
		ScoreQuestion: f,
		NRound:        rd.QuizOrder,
		Title:         rd.Title,
		NQuestion:     nQuestion,
		Order:         qu.QuizOrder,
		Question:      qu.Question,
		Answer:        qu.Answer,
	}

	return td, nil
}

// onScoreQuestion processes teams' scores for a question.
func (q *QuizState) onScoreQuestion(nQuestion int, ssSrc []*forms.ScoreResponse) string {

	qId := int64(nQuestion)

	// serialisation
	defer q.updatesQuiz()()

	// save modified responses
	ssDest := q.app.ResponseStore.ResponsesForQuestion(qId)
	var nTeams int
	for i, sSrc := range ssSrc {
		sDest := ssDest[i]

		// score specified
		v, _ := strconv.ParseFloat(sSrc.Score, 64)

		// save modified score
		if v != sDest.Score {
			sDest.Score = v
			q.app.ResponseStore.Update(&sDest.Response)
		}
		nTeams++
	}

	// set scoring status
	r := q.quizCached.ScoringRound
	if q.scored[r] == nil {
		q.scored[r] = make(map[int64]scoreStatus, 8)
	}
	q.scored[r][qId] = q.scored[r][qId].setScored(nTeams)
	q.changedScorer()

	// path to round
	return pathToScore(r)
}

// changedAll notes that all displays should be updated.
func (qs *QuizState) changedAll() {

	t := timestamp()
	qs.publishedUpdate = t
	qs.responseUpdate = t
	qs.scorerUpdate = t
	qs.app.displayState.syncUpdate = t
}

// onConfirmQuestion processes the confirmation of scores for a question.
func (q *QuizState) onConfirmQuestion(nQuestion int, fromResp bool) string {

	qId := int64(nQuestion)

	// serialisation
	defer q.updatesQuiz()()

	// save confirmed score
	rs := q.app.ResponseStore.ResponsesForQuestion(qId)
	for _, r := range rs {
		r.Confirm = r.Score
		q.app.ResponseStore.Update(&r.Response)
	}

	// #### set scoring status
	r := q.quizCached.ScoringRound
	if q.scored[r] == nil {
		q.scored[r] = make(map[int64]scoreStatus, 8)
	}
	q.scored[r][qId] = q.scored[r][qId].setConfirmed(true)
	q.changedScorer()

	// path to round
	return pathToScore(r)
}

// changedPublished notes that published scores have changed.
func (qs *QuizState) changedPublished() {

	t := timestamp()
	qs.publishedUpdate = t
	qs.scorerUpdate = t
}

// changedResponse notes that team responses have changed.
func (qs *QuizState) changedResponse() {

	t := timestamp()
	qs.responseUpdate = t
	qs.scorerUpdate = t
}

// changedScorer notes that unpublished scores have changed.
func (qs *QuizState) changedScorer() {

	qs.scorerUpdate = timestamp()
}

// timestamp returns a 31 bit timestamp for updates, which is easier to store in Javascript than 64 bit.
func timestamp() int {

	t := time.Now().Unix() // uint64

	const low31 = (1 << 31) - 1
	return int(t & low31) // discard high bits
}

// rollback must be called on all error returns from any function that calls updatesQuiz or updatesDisplays.
// It returns an HTTP status that indicates whether the error is thought to be a fault on the client or server side.
func (s *QuizState) rollback(httpStatus int, err error) int {

	s.rollbackTx = true
	if err != nil {
		s.app.log(err)
	}

	return httpStatus
}

func (s *QuizState) rollbackErr(text string) error {

	s.rollbackTx = true
	err := errors.New(text)
	s.app.log(err)

	return err
}

// save commits changes and starts a new transaction.
func (s *QuizState) save() {

	s.app.tx.Commit()
	s.app.tx = s.app.db.MustBegin()
}

// updatesQuiz takes a mutex and starts a transaction for updates to the quiz and, possibly, displays.
// It returns an anonymous function to be deferred. Call as: "defer updatesAll() ()".
func (qs *QuizState) updatesQuiz() func() {

	// aquire exclusive locks
	qs.muQuiz.Lock()

	// start transaction
	qs.app.tx = qs.app.db.MustBegin()
	qs.rollbackTx = false

	return func() {

		// save quiz changes
		var err error
		if qs.dirtyQuiz {
			if err = qs.app.QuizStore.Update(qs.quizCached); err != nil {
				qs.app.log(err)
				qs.rollbackTx = true
			}
			qs.dirtyQuiz = false
		}

		// end transaction
		if qs.rollbackTx {
			qs.app.tx.Rollback()
		} else {
			qs.app.tx.Commit()
		}

		qs.app.tx = nil

		// release locks
		qs.muQuiz.Unlock()
	}
}

// updatesNone takes mutexes for a non-updating request
func (qs *QuizState) updatesNone() func() {

	// aquire shared locks
	qs.muQuiz.RLock()

	return func() {

		// release lock
		qs.muQuiz.RUnlock()
	}
}

// txBeginRound requests a round update as a new extended transaction.
func (q *QuizState) txBeginRound(tx etx.TxId, req *OpUpdateRound) error {

	// ## could log error
	return q.app.tm.BeginNext(tx, q, OpRound, req)
}

// txShow requests a show update as a transaction, so that it will be done even if the server restarts.
func (q *QuizState) txRound(req *OpUpdateRound, opType int) error {
	return q.app.tm.SetNext(req.tx, q, opType, req)
}
