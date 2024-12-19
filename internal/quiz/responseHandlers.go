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

// Form handling for reponses from teams
// URL validation is important, because we give response URLs to the teams.

package quiz

import (
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/nosurf"

	"inchworks.com/quiz/internal/forms"
	"inchworks.com/quiz/internal/models"
)

// getFormResponses returns a form to submit a team's answers.
func (app *Application) getFormResponses(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	nTeam, _ := strconv.Atoi(ps.ByName("nTeam"))

	f, err := app.quizState.forEditResponses(nRound, nTeam, nosurf.Token(r))
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// display form
	f.Access = ps.ByName("access")
	app.render(w, r, "respond-round.page.tmpl", f)
}

// postFormResponses handles the form with a team's answers.
func (app *Application) postFormResponses(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// process form data
	f := forms.NewResponses(r.PostForm, nosurf.Token(r))

	access := f.Get("access")
	fRound := f.Get("nRound")
	nRound, err := strconv.Atoi(fRound)
	if err != nil {
		app.httpBadRequest(w, err)
	}
	fTeam := f.Get("nTeam")
	nTeam, err := strconv.Atoi(fTeam)
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}
	teamId := int64(nTeam)

	// questions
	items, err := f.GetResponses()
	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// redisplay form if data invalid
	if !f.Valid() {
		round, _ := app.RoundStore.GetByN(nRound)
		team, _ := app.TeamStore.Get(teamId)

		app.render(w, r, "respond-round.page.tmpl", &responsesFormData{
			Responses: f,
			NRound:    nRound,
			NTeam:     nTeam,
			Access:    access,
			Round:     round.Title,
			Team:      team.Name,
		})
		return
	}

	// save changes
	title, err := app.quizState.onEditResponses(nRound, nTeam, items)
	if err == nil {
		app.session.Put(r.Context(), "flash", title+" answers saved.")
		http.Redirect(w, r, pathToRespondWait(access, nTeam), http.StatusSeeOther)
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// respondWait displays a "wait" page, with rounds that might be accepted.
func (app *Application) respondWait(w http.ResponseWriter, r *http.Request) {

	// parameters
	ps := httprouter.ParamsFromContext(r.Context())
	nTeam, err := strconv.Atoi(ps.ByName("nTeam"))

	if err != nil {
		app.httpBadRequest(w, err)
		return
	}

	// data for page
	data := app.quizState.forRespondWait(nTeam)
	data.Access = ps.ByName("access")

	// common data
	data.dataDisplay.set(app, app.quizState.quizCached.Access, models.PageRespondWait, nTeam, nosurf.Token(r))

	app.render(w, r, `respond-wait.page.tmpl`, data)
}
