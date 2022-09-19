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

// Requests for scorers

package quiz

import (
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/nosurf"

	"inchworks.com/quiz/internal/forms"
	"inchworks.com/quiz/internal/models"
)

// Confirm team scores for a question.

func (app *Application) confirmQuestion(w http.ResponseWriter, r *http.Request) {

	// question number
	nQuestion, err := strconv.Atoi(r.FormValue("nQuestion"))
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// from team responses?
	from := r.FormValue("from")
	fromResp := from == "R"

	// confirm scores for question
	next := app.quizState.onConfirmQuestion(nQuestion, fromResp)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// Enter or edit scores for a round

func (app *Application) getFormEditScores(w http.ResponseWriter, r *http.Request) {

	app.getFormScores(w, r, "edit-scores")
}

func (app *Application) postFormEditScores(w http.ResponseWriter, r *http.Request) {

	app.postFormScores(w, r, true, "edit-scores", "/scorer-summary")
}

func (app *Application) getFormEnterScores(w http.ResponseWriter, r *http.Request) {

	app.getFormScores(w, r, "score-round")
}

func (app *Application) postFormEnterScores(w http.ResponseWriter, r *http.Request) {

	app.postFormScores(w, r, false, "score-round", "/scorers")
}

// getFormScoreQuestion displays the form to score all teams on a single question
func (app *Application) getFormScoreQuestion(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nQuestion, _ := strconv.Atoi(ps.ByName("nQuestion"))

	// get template data
	td, err := app.quizState.forScoreQuestion(nQuestion, nosurf.Token(r))
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// display form
	app.render(w, r, "score-question.page.tmpl", td)
}

// postFormScoreQuestion preocesses the form to score all teams on a single question
func (app *Application) postFormScoreQuestion(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// process form data
	f := forms.NewScoreQuestion(r.PostForm, nosurf.Token(r))
	nQuestion, _ := strconv.Atoi(f.Get("nQuestion"))
	ss, err := f.GetScores()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// redisplay form if data invalid
	if !f.Valid() {

		// #### lots more to fill in, needs a rethink - get data and fill in? Not good when items can be added/deleted.
		app.render(w, r, "score-question.page.tmpl", &scoreQuestionFormData{
			ScoreQuestion: f,
		})
		return
	}

	// save changes
	next := app.quizState.onScoreQuestion(nQuestion, ss)

	http.Redirect(w, r, next, http.StatusSeeOther)
}

// Common processing for new scores and editing

func (app *Application) getFormScores(w http.ResponseWriter, r *http.Request, action string) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))

	// get template data
	td := app.quizState.forEditScores(action, nRound, nosurf.Token(r))

	// display form
	app.render(w, r, "score-round.page.tmpl", td)
}

func (app *Application) postFormScores(w http.ResponseWriter, r *http.Request, edited bool, action string, next string) {

	err := r.ParseForm()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// process form data
	f := forms.NewScores(r.PostForm, nosurf.Token(r))
	nRound, _ := strconv.Atoi(f.Get("nRound"))
	ss, err := f.GetScores()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// redisplay form if data invalid
	if !f.Valid() {

		// round title
		var title string
		if round, _ := app.RoundStore.GetByN(nRound); round != nil {
			title = round.Title
		}

		app.render(w, r, "score-round.page.tmpl", &scoresFormData{
			Scores: f,
			Action: action,
			Round:  nRound,
			Title:  title,
		})
		return
	}

	// save changes
	app.quizState.onEditScores(nRound, ss, edited)

	http.Redirect(w, r, next, http.StatusSeeOther)
}

// Publish scores for round.

func (app *Application) publishRound(w http.ResponseWriter, r *http.Request) {

	// round number
	nRound, err := strconv.Atoi(r.FormValue("nRound"))
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// from team responses?
	from := r.FormValue("from")
	fromResp := from == "R"

	// publish round
	message := app.quizState.PublishRound(nRound, fromResp)
	app.session.Put(r, "flash", message)

	var path string
	if fromResp {
		path = "/scorer-rounds"
	} else {
		path = "/scorers"
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}

// scorerQuestions renders a page of question statuses for a round.
func (app *Application) scorerQuestions(w http.ResponseWriter, r *http.Request) {

	// round number
	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))

	// data for page
	data, err := app.quizState.GetScorerQuestions(nRound)
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// common data
	data.dataDisplay.set(app, DisplayScorer, models.PageScorerQuestions, nRound, nosurf.Token(r))

	app.render(w, r, `scorer-questions.page.tmpl`, data)
}

// scorerRounds renders a page of round statuses.
func (app *Application) scorerRounds(w http.ResponseWriter, r *http.Request) {

	// get rounds data
	data := app.quizState.GetScorerRounds()

	// common data
	data.dataDisplay.set(app, DisplayScorer, models.PageScorerRounds, 0, nosurf.Token(r))

	app.render(w, r, `scorer-rounds.page.tmpl`, data)
}

// scorersSummary renders a table of scores for all rounds.
func (app *Application) scorerSummary(w http.ResponseWriter, r *http.Request) {

	// get summary data
	data := app.quizState.GetScorerSummary()

	app.render(w, r, "scorer-summary.page.tmpl", data)
}
