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

package quiz

// Requests for quiz display pages

import (
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/nosurf"

	"github.com/inchworks/usage"
	"inchworks.com/quiz/internal/models"
)

// #### move all serialisation to DisplayManager

// Answers : controller or puppet

func (app *Application) quizAnswers(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	puppet := ps.ByName("puppet")

	// round template and data for slides
	template, data := app.displayState.displayRound(puppet, nRound, false)

	// common data
	data.dataDisplay.set(app, puppet, models.PageAnswers, nRound, nosurf.Token(r))

	// display page
	app.render(w, r, template, data)
}

// End of quiz
func (app *Application) quizEnd(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nPage, _ := strconv.Atoi(ps.ByName("nPage"))
	puppet := ps.ByName("puppet")

	// Page parameter is a dummy, so that all pages have the same number of parameters.
	// Needed to make window.location.href = "../quizNext work.

	// data for page
	data := app.displayState.displayStatic(puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PageStatic, nPage, nosurf.Token(r))

	app.render(w, r, `quiz-end.page.tmpl`, data)
}

// Final scores: controller, puppet, scoreboard or quizmaster
func (app *Application) quizFinal(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	puppet := ps.ByName("puppet")

	data := app.displayState.displayScores(nRound, true, puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PageFinal, nRound, nosurf.Token(r))

	// ## could cache the teams with totals and rank? Perhaps little saving compared to scores

	var template string
	if puppet == DisplayQuizmaster {
		if data.TieBreak {
			template = `quizmaster-tie.page.tmpl`
		} else {
			template = `quizmaster-final.page.tmpl`

		}
	} else {
		if data.TieBreak {
			template = `quiz-tie.page.tmpl`
		} else {
			template = `quiz-final.page.tmpl`
		}
	}

	// display page
	app.render(w, r, template, data)
}

// quizJoin displays the first page of the quiz.
// It assumes the visitor has supplied a valid access code as the puppet parameter.
func (app *Application) quizJoin(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	puppet := ps.ByName("puppet")

	// data for page
	data := app.displayState.displayStatic(puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PageStatic, 0, nosurf.Token(r))

	app.render(w, r, `quiz-start.page.tmpl`, data)
}

// Questions: controller or puppet

func (app *Application) quizQuestions(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	puppet := ps.ByName("puppet")

	// round template and data for slides
	template, data := app.displayState.displayRound(puppet, nRound, true)

	// common data
	data.dataDisplay.set(app, puppet, models.PageQuestions, nRound, nosurf.Token(r))

	// display page
	app.render(w, r, template, data)
}

// Scores: controller, puppet, scoreboard or quizmaster

func (app *Application) quizScores(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	puppet := ps.ByName("puppet")

	// data for slide
	// ## doesn't need scoresTop
	data := app.displayState.displayScores(nRound, false, puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PageScores, nRound, nosurf.Token(r))

	var template string
	switch puppet {
	case DisplayQuizmaster:
		template = `quizmaster-scores.page.tmpl`

	case DisplayScoreboard:
		template = `scoreboard-scores.page.tmpl`

	default:
		template = `quiz-scores.page.tmpl`
	}

	// display page
	app.render(w, r, template, data)
}

// First page of quiz: controller or puppet

func (app *Application) quizStart(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nPage, _ := strconv.Atoi(ps.ByName("nPage"))
	puppet := ps.ByName("puppet")

	// Page parameter is a dummy, so that all pages have the same number of parameters.
	// Needed to make window.location.href = "../quizNext work.

	// practice mode is same as controller
	if puppet == DisplayPractice {
		puppet = DisplayController
	}

	// data for page
	data := app.displayState.displayStatic(puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PageStatic, nPage, nosurf.Token(r))

	app.render(w, r, `quiz-start.page.tmpl`, data)
}

// Wait for scores to be published: controller or puppet

func (app *Application) quizWait(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	puppet := ps.ByName("puppet")

	// data for page
	data := app.displayState.displayWait(nRound, puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PageScoresWait, nRound, nosurf.Token(r))

	// display page
	app.render(w, r, `quiz-wait.page.tmpl`, data)
}

// quizmasterResponses shows response statuses for the round, with the leaderboard
func (app *Application) quizmasterResponses(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))

	data := app.displayState.displayResponses(nRound, "Q")

	// common data
	data.dataDisplay.set(app, DisplayQuizmaster, models.PageQuizResponses, nRound, nosurf.Token(r))

	// display page
	app.render(w, r, `quizmaster-responses.page.tmpl`, data)
}

// quizmasterWait shows waiting for scores, with leaderboard
func (app *Application) quizmasterWait(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))

	data := app.displayState.displayLeaderboard(nRound, "Q")

	// common data
	data.dataDisplay.set(app, DisplayQuizmaster, models.PageScoresWait, nRound, nosurf.Token(r))

	// display page
	app.render(w, r, `quizmaster-wait.page.tmpl`, data)
}

// Scoreboard at start of quiz, also shown to quizmaster

func (app *Application) scoreboardStart(w http.ResponseWriter, r *http.Request) {

	// (Round param to make all slides the same)
	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	puppet := ps.ByName("puppet")

	// data for template
	data := app.displayState.displayTeams(puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PageStart, nRound, nosurf.Token(r))

	var template string
	if puppet == DisplayQuizmaster {
		template = `quizmaster-start.page.tmpl`
	} else {
		template = `scoreboard-start.page.tmpl`
	}

	// display page
	app.render(w, r, template, data)
}

// Scoreboard waiting for quizmaster to announce scores

func (app *Application) scoreboardWait(w http.ResponseWriter, r *http.Request) {

	ps := httprouter.ParamsFromContext(r.Context())
	nRound, _ := strconv.Atoi(ps.ByName("nRound"))
	puppet := ps.ByName("puppet") // ## should preset it here

	// application processor
	data := app.displayState.displayWait(nRound, puppet)

	// common data
	data.dataDisplay.set(app, puppet, models.PagePublicWait, nRound, nosurf.Token(r))

	// display page
	app.render(w, r, `scoreboard-wait.page.tmpl`, data)
}

// Usage statistics

func (app *Application) usageDays(w http.ResponseWriter, r *http.Request) {

	data := app.quizState.ForUsage(usage.Day)

	app.render(w, r, "usage.page.tmpl", data)
}

func (app *Application) usageMonths(w http.ResponseWriter, r *http.Request) {

	data := app.quizState.ForUsage(usage.Month)

	app.render(w, r, "usage.page.tmpl", data)
}

// set adds data common to all display pages, and registers for monitoring.
// The data items are the static ones that can be read without serialisation.
func (d *dataDisplay) set(app *Application, puppet string, page int, param int, token string) {

	mSrefresh := 500 * (1 << app.quizState.quizCached.Refresh) // mS from refresh level

	d.Puppet = puppet
	d.Page = page
	d.Param = param
	d.Interval = mSrefresh
	d.Monitor = app.Monitor.Register(puppet, time.Duration(int64(mSrefresh)*int64(time.Millisecond)))
	d.CSRFToken = token
}
