// Copyright © Rob Burke inchworks.com, 2020.

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

package quiz

// Processing for quiz editing / setup.
//
// These functions may modify application state.

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/inchworks/webparts/etx"
	"github.com/inchworks/webparts/multiforms"
	"github.com/inchworks/webparts/uploader"

	"inchworks.com/quiz/internal/forms"
	"inchworks.com/quiz/internal/models"
)

// forEditQuestions gets data to edit questions.
func (q *QuizState) forEditQuestions(nRound int, token string) (status int, qd *questionsFormData) {

	// serialisation
	defer q.updatesQuiz()()

	// round and questions
	r, err := q.app.RoundStore.GetByN(nRound)
	if err != nil {
		status = q.rollback(http.StatusNotFound, nil)
		return
	}
	questions := q.app.QuestionStore.ForRound(r.Id)

	// start multi-step transaction for uploaded files
	ts, err := q.app.uploader.Begin()
	if err != nil {
		status = q.rollback(http.StatusInternalServerError, err)
		return
	}

	// form, with sub-forms
	f := forms.NewQuestions(make(url.Values), token)
	f.Set("timestamp", ts)

	// template for new question sub-form
	f.AddTemplate(len(questions))

	// add questions to form
	for i, q := range questions {
		f.Add(i, q)
	}

	// template fields
	qd = &questionsFormData{
		Questions: f,
		Title:     r.Title,
		NRound:    nRound,
		MaxUpload: q.app.cfg.MaxUpload,
	}

	return
}

// Processing when questions are modified
//
// There is no need to restart the quiz. Scores are not affected.

func (q *QuizState) onEditQuestions(nRound int, tx etx.TxId, qsSrc []*forms.Question) int {

	// serialisation
	defer q.updatesQuiz()()

	// compare modified questions against current question, and update
	r, _ := q.app.RoundStore.GetByN(nRound)
	qsDest := q.app.QuestionStore.ForRound(r.Id)

	updated := false
	nSrc := len(qsSrc)
	nDest := len(qsDest)
	iSrc := 1 // skip template
	iDest := 0

	for iSrc < nSrc || iDest < nDest {

		if iSrc == nSrc {
			// no more source questions - delete from destination
			q.app.QuestionStore.DeleteId(qsDest[iDest].Id)
			updated = true
			iDest++

		} else if iDest == nDest {
			// no more destination questions - add new one
			mediaName := uploader.CleanName(qsSrc[iSrc].MediaName)

			qd := models.Question{
				Round:     r.Id,
				QuizOrder: qsSrc[iSrc].QuizOrder,
				Question:  q.sanitize(qsSrc[iSrc].Question, ""),
				Answer:    q.sanitize(qsSrc[iSrc].Answer, ""),
				File:      uploader.FileFromName(tx, mediaName),
			}
			q.app.QuestionStore.Update(&qd)
			updated = true
			iSrc++

		} else {
			ix := qsSrc[iSrc].ChildIndex
			if ix > iDest {
				// source question removed - delete from destination
				q.app.QuestionStore.DeleteId(qsDest[iDest].Id)
				updated = true
				iDest++

			} else if ix == iDest {
				// check if details changed
				// (checking media name at this point, version change will be handled later)
				mediaName := uploader.CleanName(qsSrc[iSrc].MediaName)
				qDest := qsDest[iDest]
				_, dstName, _ := uploader.NameFromFile(qDest.File)
				if qsSrc[iSrc].QuizOrder != qDest.QuizOrder ||
					qsSrc[iSrc].Question != qDest.Question ||
					qsSrc[iSrc].Answer != qDest.Answer ||
					mediaName != dstName {

					qDest.QuizOrder = qsSrc[iSrc].QuizOrder
					qDest.Question = q.sanitize(qsSrc[iSrc].Question, qDest.Question)
					qDest.Answer = q.sanitize(qsSrc[iSrc].Answer, qDest.Answer)

					// If the media name hasn't changed, leave the old version in use for now.
					// We'll detect a version change later.
					if mediaName != dstName {
						qDest.File = uploader.FileFromName(tx, mediaName)
					}

					q.app.QuestionStore.Update(qDest)
					updated = true
				}
				iSrc++
				iDest++

			} else {
				// out of sequence question index
				return q.rollback(http.StatusBadRequest, nil)
			}
		}
	}

	// re-sequence questions, removing missing or duplicate orders
	// If two questions have the same order, the later update comes first
	if updated {

		// ## think I have to commit changes for them to appear in a new query
		q.save()

		qus := q.app.QuestionStore.ForRound(r.Id)

		for ix, qu := range qus {
			nOrder := ix + 1
			if qu.QuizOrder != nOrder {

				// update sequence
				qu.QuizOrder = nOrder
				q.app.QuestionStore.Update(qu)
			}
		}
	}

	// request worker to generate media versions, and remove unused images
	if err := q.txRound(
		&OpUpdateRound{
			RoundId: r.Id,
			tx:      tx,
		},
		OpRound); err != nil {
		return q.rollback(http.StatusInternalServerError, err)
	}

	return 0
}

// Get data to edit quiz

func (q *QuizState) forEditQuiz(token string) *quizFormData {

	// serialisation
	defer q.updatesNone()()

	// current data
	var d = make(url.Values)
	f := multiforms.New(d, token)
	qc := q.quizCached
	f.Set("title", qc.Title)
	f.Set("organiser", qc.Organiser)
	f.Set("nTieBreakers", strconv.Itoa(qc.NTieBreakers))
	f.Set("nFinalScores", strconv.Itoa(qc.NFinalScores))
	f.Set("nDeferred", strconv.Itoa(qc.NDeferred))
	f.Set("access", qc.Access)
	f.Set("refresh", strconv.Itoa(qc.Refresh))

	// add join URL for each team
	ts := q.app.TeamStore.ByName()
	var tus []teamData
	for _, t := range ts {
		tu := teamData{
			Team: t,
			URL:  q.app.getTeamURL(t.Access, t.Id),
		}
		tus = append(tus, tu)
	}

	return &quizFormData{
		Form:   f,
		Rounds: q.app.RoundStore.All(),
		Teams:  tus,
	}
}

// Processing when quiz is modified. No need to restart quiz.
//
// Returns true if no client errors.

func (q *QuizState) onEditQuiz(title string, organiser string, nTieBreakers int, nFinalScores int, nDeferred int, access string, refresh int) bool {

	// serialisation
	defer q.updatesQuiz()()

	// save changes via cache (conversions already checked)
	qc := q.quizCached
	qc.Title = title
	qc.Organiser = organiser
	qc.NTieBreakers = nTieBreakers
	qc.NFinalScores = nFinalScores
	qc.NDeferred = nDeferred
	qc.Access = access
	qc.Refresh = refresh
	q.dirtyQuiz = true

	// cached numbers of rounds
	q.setNumRounds(q.app.RoundStore.Count(), nTieBreakers, nDeferred)

	return true
}

// Get data to edit rounds

func (q *QuizState) forEditRounds(token string) *roundsFormData {

	// serialisation
	defer q.updatesNone()()

	// rounds
	rounds := q.app.RoundStore.All()
	qc := q.quizCached

	// form, with sub-forms
	f := forms.NewRounds(make(url.Values), token)

	f.Set("nTieBreakers", strconv.Itoa(qc.NTieBreakers))
	f.AddTemplate(len(rounds))
	for i, r := range rounds {
		f.Add(i, r)
	}

	// template fields
	d := roundsFormData{
		Rounds: f,
	}

	return &d
}

// Processing when rounds are modified.
//
// Returns true if no client errors.

func (q *QuizState) onEditRounds(rsSrc []*forms.Round) (int, etx.TxId) {

	// serialisation
	defer q.updatesQuiz()()

	// start extended transaction
	tx := q.app.tm.Begin()

	// skip template
	iSrc := 1
	iDest := 0
	count := 0

	// compare modified rounds against current rounds, and update
	rsDest := q.app.RoundStore.All()
	nSrc := len(rsSrc)
	nDest := len(rsDest)

	var restart bool

	for iSrc < nSrc || iDest < nDest {

		if iSrc == nSrc {
			// no more source rounds - delete from destination
			q.onRemoveRound(tx, rsDest[iDest].Id)
			restart = true
			iDest++

		} else if iDest == nDest {
			// no more destination rounds - add new one
			r := rsSrc[iSrc].Round

			q.app.RoundStore.Update(&r)
			iSrc++
			count++

		} else {
			ix := rsSrc[iSrc].ChildIndex
			if ix > iDest {
				// source round removed - delete from destination
				q.onRemoveRound(tx, rsDest[iDest].Id)
				restart = true
				iDest++

			} else if ix == iDest {
				// check if details changed
				rDest := rsDest[iDest]
				if rsSrc[iSrc].QuizOrder != rDest.QuizOrder ||
					rsSrc[iSrc].Title != rDest.Title ||
					rsSrc[iSrc].Format != rDest.Format {
					rDest.QuizOrder = rsSrc[iSrc].QuizOrder
					rDest.Title = rsSrc[iSrc].Title
					rDest.Format = rsSrc[iSrc].Format

					q.app.RoundStore.Update(rDest)
				}
				iSrc++
				iDest++
				count++

			} else {
				// out of sequence round index
				return q.rollback(http.StatusBadRequest, nil), 0
			}
		}
	}

	// if round removed, potentially invalidating scores, restart quiz
	if restart {
		q.restartQuiz()
	}

	// cached numbers of rounds
	q.setNumRounds(count, q.quizCached.NTieBreakers, q.quizCached.NDeferred)

	return 0, tx
}

// Get data to edit teams

func (q *QuizState) forEditTeams(token string) *teamsFormData {

	// serialisation
	defer q.updatesNone()()

	// teams
	teams := q.app.TeamStore.ByName()

	// form, with sub-forms
	f := forms.NewTeams(make(url.Values), token)
	f.AddTemplate()
	for i, t := range teams {
		f.Add(i, t)
	}

	// template fields
	td := &teamsFormData{
		Teams: f,
	}

	return td
}

func (q *QuizState) onEditTeams(tsSrc []*forms.Team) int {

	// serialisation
	defer q.updatesQuiz()()

	// skip template
	iSrc := 1
	iDest := 0

	// compare modified teams against current teams, and update teams
	tsDest := q.app.TeamStore.ByName()
	nSrc := len(tsSrc)
	nDest := len(tsDest)

	for iSrc < nSrc || iDest < nDest {

		if iSrc == nSrc {
			// no more source teams - delete from destination
			q.onRemoveTeam(tsDest[iDest])
			iDest++

		} else if iDest == nDest {
			// no more destination teams - add new team
			t := models.Team{Name: tsSrc[iSrc].Name}
			q.app.TeamStore.Update(&t)
			iSrc++

		} else {
			ix := tsSrc[iSrc].ChildIndex
			if ix > iDest {
				// source team removed - delete from destination
				q.onRemoveTeam(tsDest[iDest])
				iDest++

			} else if ix == iDest {
				// check if team name changed
				tDest := tsDest[iDest]
				if tsSrc[iSrc].Name != tDest.Name {
					tDest.Name = tsSrc[iSrc].Name
					q.app.TeamStore.Update(tDest)
				}
				iSrc++
				iDest++

			} else {
				// out of sequence team index
				return q.rollback(http.StatusBadRequest, nil)
			}
		}
	}

	return 0
}

// Get rounds in sequence order

func (q *QuizState) rounds() []*models.Round {

	// Serialisation
	defer q.updatesNone()()

	return q.app.RoundStore.All()
}

// onRemoveProcessing when a round is removed

func (q *QuizState) onRemoveRound(tx etx.TxId, roundId int64) error {

	// questions will be removed by cascade delete
	q.app.RoundStore.DeleteId(roundId)

	// request worker to remove media files
	return q.txBeginRound(tx, &OpUpdateRound{
		RoundId: roundId,
		tx:      0},
	)
}

// Processing when a team is removed
//
// Does not restart the quiz - the team may have withdrawn after the start.

func (q *QuizState) onRemoveTeam(team *models.Team) {

	// remove relationship between team and quiz
	// scores will be removed by onDelete="CASCADE"
	q.app.TeamStore.DeleteId(team.Id)

	// adjust cached number of teams
	q.app.NTeams--

	// update all displays
	q.changedAll()
}

// sanitize returns sanitized HTML, assuming the current string is safe.
func (q *QuizState) sanitize(new string, current string) string {
	if new == current {
		return current
	}

	return q.app.sanitizer.Sanitize(new)
}
