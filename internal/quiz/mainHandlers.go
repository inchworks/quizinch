// Requests for main functions and quiz setup

package quiz

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

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/justinas/nosurf"
)

// home renders the application home page.
func (app *Application) home(w http.ResponseWriter, r *http.Request) {

	var pg string
	if app.isOnline {
		pg = "home-online.page.tmpl"
	} else {
		pg = "home-local.page.tmpl"
	}
	app.render(w, r, pg, &dataSimple{
		Organiser: app.quizState.quizCached.Organiser,
		Val:       app.Version,
		CSRFToken: nosurf.Token(r),
	})
}

// Quiz controller

func (app *Application) controller(w http.ResponseWriter, r *http.Request) {

	// has the quiz started?
	// ## set scoring round to 0 before starting
	started := ""
	if app.quizState.quizCached.ScoringRound > 0 {
		started = "Y"
	}

	app.render(w, r, "controller.page.tmpl", &dataSimple{
		Val:       started,
		CSRFToken: nosurf.Token(r),
	})
}

// Menu of puppet displays

func (app *Application) displays(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "displays.page.tmpl", nil)
}

// Login

func (app *Application) login(w http.ResponseWriter, r *http.Request) {

	app.users.PostFormLogin(w, r)
}

// Logout user

func (app *Application) logout(w http.ResponseWriter, r *http.Request) {

	// remove user ID from the contest data
	app.session.Remove(r, "authenticatedUserID")

	// flash message to confirm logged out
	app.session.Put(r, "flash", "You are logged out")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Resume quiz

func (app *Application) controlResume(w http.ResponseWriter, r *http.Request) {

	http.Redirect(w, r, app.displayState.resumeQuiz(), http.StatusSeeOther)
}

// Start quiz

func (app *Application) controlStart(w http.ResponseWriter, r *http.Request) {

	// controller or practice display?
	var live bool
	switch r.FormValue("display") {
	case DisplayController:
		live = true
	case DisplayPractice:
		live = false
	default:
		app.httpBadRequest(w, errors.New("unknown display name"))
		return
	}

	http.Redirect(w, r, app.displayState.startQuiz(live), http.StatusSeeOther)
}

// noAccess shows a page when an access token has been rejected.
func (app *Application) noAccess(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "no-access.page.tmpl", &dataDisplay{
		IsLive:    app.displayState.contest.Live,
		Organiser: app.quizState.quizCached.Organiser,
	})
}

// scorers renders the home page for scorers.
func (app *Application) scorers(w http.ResponseWriter, r *http.Request) {

	// Note that it is necessary to read the scoring round via quizManager,
	// (a) for serialisation
	// (b) to get the quiz data cached, in case this is the first page opened

	// display page
	var pg string
	if app.hasRemote {
		pg = "scorers-remote.page.tmpl"
	} else {
		pg = "scorers-local.page.tmpl"
	}
	app.render(w, r, pg, &dataSimple{
		Val:       strconv.Itoa(app.quizState.GetScoringRound()),
		CSRFToken: nosurf.Token(r),
	})
}
