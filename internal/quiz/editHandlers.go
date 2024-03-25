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

// Form handling for quiz setup

package quiz

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/inchworks/webparts/etx"
	"github.com/inchworks/webparts/multiforms"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/nosurf"

	"inchworks.com/quiz/internal/forms"
)

type RepUpload struct {
	Error string `json:"error"`
}

// postFormImage handles an uploaded media file.
func (app *Application) postFormMedia(w http.ResponseWriter, r *http.Request) {

	timestamp := r.FormValue("timestamp")

	// multipart form
	// (The limit, 10 MB, is just for memory use, not the size of the upload)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// get file returned with form
	f := r.MultipartForm.File["media"]
	if len(f) == 0 {
		// ## don't know how we can get a form without a file, but we do
		app.httpBadRequest(w, errors.New("upload received without media"))
		return
	}

	// check file size, rounded to nearest MB
	// (Our client script checks file sizes, so we needn't send a nice error.)
	fh := f[0]
	sz := (fh.Size + (1 << 19)) >> 20
	if sz > int64(app.cfg.MaxUpload) {
		httpTooLarge(w)
		return
	}

	// schedule upload to be saved as a file
	id, err := etx.Id(timestamp)
	if err != nil {
		app.log(err)
		httpServerError(w)
	}

	var s string
	err, byUser := app.uploader.Save(fh, id)
	if err != nil {
		if byUser {
			s = err.Error()

		} else {
			// server error
			app.log(err)
			httpServerError(w)
			return
		}
	}

	// return response
	app.reply(w, RepUpload{Error: s})
}

// Form to setup questions

func (app *Application) getFormQuestions(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))

	status, qd := app.quizState.forEditQuestions(nRound, nosurf.Token(r))
	if status != 0 {
		http.Error(w, http.StatusText(status), status)
		return
	}

	// display form
	app.render(w, r, "setup-questions.page.tmpl", qd)
}

func (app *Application) postFormQuestions(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// process form data
	f := forms.NewQuestions(r.PostForm, nosurf.Token(r))
	nRound, err := strconv.Atoi(f.Get("nRound"))
	if err != nil {
		app.httpBadRequest(w, err)
	}

	// questions
	items, err := f.GetQuestions(app.validTypeCheck())
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	tx, err := etx.Id(f.Get("timestamp"))
	if err != nil {
		app.httpBadRequest(w, err)
	}

	// redisplay form if data invalid
	if !f.Valid() {
		t := app.displayState.roundTitle(nRound)
		app.render(w, r, "setup-questions.page.tmpl", &questionsFormData{
			Questions: f,
			Title:     t,
			NRound:    nRound,
			MaxUpload: app.cfg.MaxUpload,
		})
		return
	}

	// save changes
	status := app.quizState.onEditQuestions(nRound, tx, items)
	if status == 0 {

		// bind updated media, now that update is committed
		app.uploader.DoNext(tx)

		http.Redirect(w, r, "/setup-quiz", http.StatusSeeOther)

	} else {
		http.Error(w, http.StatusText(status), status)
	}

}

// Main form to setup quiz

func (app *Application) getFormQuiz(w http.ResponseWriter, r *http.Request) {

	td := app.quizState.forEditQuiz(nosurf.Token(r))

	// display form
	app.render(w, r, "setup-quiz.page.tmpl", td)
}

func (app *Application) postFormQuiz(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// process form data
	f := multiforms.New(r.PostForm, nosurf.Token(r))
	f.Required("title", "organiser", "nTieBreakers", "nFinalScores", "nWinners", "nDeferred", "refresh")
	f.MaxLength("title", 60)
	f.MaxLength("organiser", 60)
	f.MaxLength("access", 60)
	nTieBreakers := f.Positive("nTieBreakers")
	nFinalScores := f.Positive("nFinalScores")
	nWinners := f.Positive("nWinners")
	nDeferred := f.Positive("nDeferred")
	refresh := f.Positive("refresh")

	// redisplay form if data invalid
	if !f.Valid() {

		app.render(w, r, "setup-quiz.page.tmpl", &quizFormData{
			Form:   f,
			Rounds: app.quizState.rounds(),
		})
		return
	}

	// save changes
	if app.quizState.onEditQuiz(f.Get("title"), f.Get("organiser"), nTieBreakers, nFinalScores, nWinners, nDeferred, f.Get("access"), refresh) {
		http.Redirect(w, r, "/", http.StatusSeeOther)

	} else {
		app.httpBadRequest(w, err)
	}
}

// Form to setup rounds

func (app *Application) getFormRounds(w http.ResponseWriter, r *http.Request) {

	f := app.quizState.forEditRounds(nosurf.Token(r))

	// display form
	app.render(w, r, "setup-rounds.page.tmpl", f)
}

func (app *Application) postFormRounds(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}
	// process form data
	f := forms.NewRounds(r.PostForm, nosurf.Token(r))

	// rounds
	rs, err := f.GetRounds()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// redisplay form if data invalid
	if !f.Valid() {
		app.render(w, r, "setup-rounds.page.tmpl", &roundsFormData{
			Rounds: f,
		})
		return
	}

	// save changes
	status, tx := app.quizState.onEditRounds(rs)
	if status == 0 {
		// bind updated media, now that update is committed
		app.tm.DoNext(tx)

		http.Redirect(w, r, "/setup-quiz", http.StatusSeeOther)

	} else {
		http.Error(w, http.StatusText(status), status)
	}
}

// Form to setup teams

func (app *Application) getFormTeams(w http.ResponseWriter, r *http.Request) {

	f := app.quizState.forEditTeams(nosurf.Token(r))

	// display form
	app.render(w, r, "setup-teams.page.tmpl", f)
}

func (app *Application) postFormTeams(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// process form data
	f := forms.NewTeams(r.PostForm, nosurf.Token(r))

	// teams
	ts, err := f.GetTeams()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// redisplay form if data invalid
	if !f.Valid() {
		app.render(w, r, "setup-teams.page.tmpl", &teamsFormData{
			Teams: f,
		})
		return
	}

	// save changes
	status := app.quizState.onEditTeams(ts)
	if status == 0 {
		http.Redirect(w, r, "/setup-quiz", http.StatusSeeOther)

	} else {
		http.Error(w, http.StatusText(status), status)
	}
}
