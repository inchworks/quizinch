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
//
// These functions may modify application state.

package quiz

import (
	"fmt"
	"sort"
	"strconv"

	"inchworks.com/quiz/internal/models"
)

// GetScorerQuestions returns the scoring status for a round
func (q *QuizState) GetScorerQuestions(nRound int) (*dataScorerQuestions, error) {

	// serialisation
	defer q.updatesNone()()

	var sqs []*scoreQuestionData

	// round title
	rd, err := q.app.RoundStore.GetByN(nRound)
	if err != nil {
		return nil, err
	}

	// questions
	qus := q.app.QuestionStore.ForRound(int64(nRound))
	for _, qu := range qus {

		// status of question
		var a, s, b string
		status := q.scored[nRound][qu.Id]
		if status.teams == 0 {
			// start scoring
			a = "alert alert-success"
			s = "no scores"
			b = "btn-primary"

		} else if status.teams == q.app.NTeams {

			if !status.confirmed {
				// scores to be checked
				a = "alert alert-info"
				s = "scored"
				b = "btn-info"

			} else {
				// scores confirmed
				a = "bg-success text-white"
				s = "checked"
				b = "btn-secondary"
			}

		} else {
			// started scoring but missing team(s)
			a = "alert alert-danger"
			s = fmt.Sprint(q.app.NTeams-status.teams, " unscored")
			b = "btn-danger"
		}

		sq := &scoreQuestionData{
			Question: qu,
			Alert:    a,
			Status:   s,
			Btn:      b,
		}

		sqs = append(sqs, sq)
	}

	return &dataScorerQuestions{
		NRound:    nRound,
		Title:     rd.Title,
		Questions: sqs,
		dataDisplay: dataDisplay{
			Update: q.scorerUpdate,
		},
	}, nil
}

// GetScorerRounds returns the scoring status for each round
func (q *QuizState) GetScorerRounds() *dataScorerRounds {

	// serialisation
	defer q.updatesNone()()

	var srs []*scoreRoundData

	// rounds
	nTeams := q.app.NTeams
	scoring := q.quizCached.ScoringRound
	rs := q.app.RoundStore.All()
	for _, r := range rs {

		var a, s, b string

		if r.QuizOrder < scoring {
			a = "bg-success text-white"
			s = "scored"
			b = "btn-secondary"

		} else if r.QuizOrder == scoring {
			resp := q.app.TeamStore.CountResponded(r.Id)

			nq := len(q.scored[scoring])
			if nq > 0 {
				toCheck := 0
				for _, status := range q.scored[scoring] {
					if !status.confirmed {
						toCheck++
					}
				}
				if toCheck > 0 {
					// questions to be checked
					a = "alert alert-primary"
					s = fmt.Sprint("CHECK ", toCheck, " QUESTIONS")
					b = "btn-info"

				} else if nq < nTeams {
					// checked, more to score
					a = "alert alert-info"
					s = fmt.Sprint("score ", nTeams-nq, " questions")
					b = "btn-secondary"

				} else {
					// all scored and checked
					a = "bg-primary text-white"
					s = "all checked"
					b = "btn-secondary"
				}

			} else if resp == q.app.NTeams {

				if scoring == q.quizCached.ResponseRound {
					// all responses - get ready
					a = "alert alert-warning"
					s = "final answers allowed"
					b = "btn-secondary"

				} else {
					// start scoring
					a = "alert alert-success"
					s = "READY TO SCORE"
					b = "btn-primary"
				}

			} else {
				// wait
				a = "alert alert-danger"
				s = fmt.Sprint("waiting on ", q.app.NTeams-resp, " teams")
				b = "btn-secondary"
			}

		} else {
			b = "btn-secondary"
		}

		sr := &scoreRoundData{
			Round:  r,
			Alert:  a,
			Status: s,
			Btn:    b,
		}

		srs = append(srs, sr)
	}

	return &dataScorerRounds{
		Rounds: srs,
		dataDisplay: dataDisplay{
			Update: q.scorerUpdate,
		},
	}
}

// GetScorerSummary returns round headings and scores for all completed rounds, ordered by team name.
func (q *QuizState) GetScorerSummary() *scoreSummaryData {

	// Note that we must calculate totals on the fly, because they aren't stored
	// in the database until the quizmaster is ready to view the current round.
	// ## Perhaps it would have been better to extend the database for two sets of scores?

	// serialisation
	defer q.updatesNone()()

	nFull := q.nFullRounds
	completed := q.quizCached.ScoringRound - 1

	var headings []*heading
	for nRound := 1; nRound <= completed; nRound++ {

		// name for round, adding extra tie-break rounds if needed
		if nRound <= nFull {
			headings = append(headings, &heading{nRound, strconv.Itoa(nRound)})
		} else {
			headings = append(headings, &heading{nRound, "Tie " + strconv.Itoa(nRound-nFull)})
		}
	}

	teams := q.app.TeamStore.ByName()

	var tScores []*teamScores
	for _, team := range teams {

		t := &teamScores{Name: team.Name}

		// scores for all completed rounds
		nRound := 1
		scores := q.app.ScoreStore.CompletedForTeam(team.Id, completed)
		for _, score := range scores {

			// could be gaps if separate tie breaks for first and second places
			addUnscored(&t.Rounds, nRound, score.NRound - 1)
			nRound = score.NRound

			t.Rounds = append(t.Rounds, score.Value)
			if nRound <= nFull {
				t.Total += score.Value  // total for full rounds only
			}
			nRound = score.NRound + 1
		}
		// fill column for unscored teams in tie-break
		addUnscored(&t.Rounds, nRound, completed)

		tScores = append(tScores, t)
	}

	// sort the teams by decreasing score on full rounds
	sort.Slice(tScores,
		func(i, j int) bool { return tScores[i].Total > tScores[j].Total })

	// set rank, in decreasing score order
	rank := 0
	priorScore := -1.0
	for i, t := range tScores {

		if t.Total != priorScore {
			// advance rank
			rank = i + 1
			priorScore = t.Total
		}
		t.Rank = rank
	}

	// add ranks for tie-break rounds
	for nRound := nFull + 1; nRound <= completed; nRound++ {
		rankTie(tScores, nRound)
	}

	// re-sort teams by name
	sort.Slice(tScores,
		func(i, j int) bool { return tScores[i].Name < tScores[j].Name })

	return &scoreSummaryData{
		Rounds: headings,
		Scores: tScores,
	}
}

// Get scoring round (for scorers menu)

func (q *QuizState) GetScoringRound() int {

	// serialisation
	defer q.updatesNone()()

	return q.quizCached.ScoringRound
}

// PublishRound publishes a round.
// Processing dependes on whether we have response scores, or entered round scores.
func (q *QuizState) PublishRound(nRound int, fromResp bool) string {

	// serialisation
	defer q.updatesQuiz()()

	qc := q.quizCached

	// check if round already published
	if nRound != qc.ScoringRound {
		return `Round ` + strconv.Itoa(nRound) + ` already published`
	}

	// check if all rounds scored
	nRounds := q.nFullRounds
	if nRound > nRounds+q.nTieRounds {
		return `All rounds have been scored!`
	}

	if fromResp {
		// get team scores from responses
		ts := q.app.TeamStore.AllWithResponses(nRound)

		// on normal rounds ..
		if nRound <= nRounds {

			// check if all teams have responded
			if len(ts) < q.app.NTeams {
				return `Not all teams have responded yet`
			}

			// #### check if all teams have been scored (need nulls on scores)
		}

		// aggregate response scores into round scores
		for _, t := range ts {
			s, _ := q.app.ScoreStore.ForTeamAndRound(t.Id, nRound)
			if s == nil {
				s = &models.Score{
					Team:   t.Id,
					NRound: nRound,
				}
			}
			s.Value = t.Value
			q.app.ScoreStore.Update(s)
		}

	} else {
		// direct entry of round scores
		// check if all teams have been scored, on normal rounds
		if nRound <= nRounds {
			missing := ``
			scoreTeams := q.app.TeamStore.AllWithScores(nRound)
			for _, scoreTeam := range scoreTeams {

				if !scoreTeam.Value.Valid {
					missing += scoreTeam.Name + `, `
				}
			}
			if missing != `` {
				return `No score for: ` + missing
			}
		}
	}

	// get contest data, to be updated
	s, dUnlock := q.app.displayState.forUpdate()
	defer dUnlock()

	// calculate total scores and rank, if main display is ready or waiting
	// Otherwise we must wait, because displays may still be showing scores from a previous round.
	currentPage := s.CurrentPage
	if (s.CurrentRound == nRound) &&
		((currentPage == models.PageAnswers) || (currentPage == models.PageScoresWait)) {
		q.calculateTotalsAndRank(nRound, s)
	} else {
		// notify just the quizmaster
		q.app.displayState.changedQuizmaster()
	}

	// set next round to be scored
	qc.ScoringRound = nRound + 1
	q.dirtyQuiz = true

	return `Round ` + strconv.Itoa(nRound) + ` published`
}


// addUnscored sets -1 scores for unscored teams in tie-break rounds (reset to 0 later)
func addUnscored(scores *[]float64, nRound, toRound int) {
	for ; nRound <= toRound; nRound++ {
		*scores = append(*scores, -1)
	}
}

// rankTie increase rank for teams that participate in a tie-break round.
func rankTie(tScores []*teamScores, nRound int) {

	// sort the teams by decreasing score on this round
	ix := nRound - 1
	sort.Slice(tScores,
		func(i, j int) bool { return tScores[i].Rounds[ix] > tScores[j].Rounds[ix] })

	// add rank, in decreasing score order
	delta := -1
	priorScore := -1.0
	for i, t := range tScores {

		s := t.Rounds[ix]
		if s >= 0 {
			if s != priorScore {
				// advance rank
				delta = i
				priorScore = s
			}
			t.Rank += delta

		} else {
			// not in this tie - no score
			t.Rounds[ix] = 0
		}
	}
}

// Restart while serialised

func (q *QuizState) restartQuiz() {

	qc := q.quizCached

	// start at round 1
	qc.ResponseRound = 0
	qc.ScoringRound = 1
	q.dirtyQuiz = true

	q.changedAll()

	// restart displays
	s, dUnlock := q.app.displayState.forUpdate()
	defer dUnlock()
	q.app.displayState.restartQuiz(s)

	// remove all responses and scores
	q.app.ResponseStore.DeleteAll(qc.Id)
	q.app.ScoreStore.DeleteAll(qc.Id)

	// reset scoring status
	q.scored = make(map[int]map[int64]scoreStatus, q.nFullRounds+q.nTieRounds)

	// reset team totals and ranking
	teams := q.app.TeamStore.All()
	for _, team := range teams {
		team.Total = 0
		team.Rank = 1
		q.app.TeamStore.Update(team)
	}
}
