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

import (
	"io/fs"
	"net/http"
	"path"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

// puppet display types, used in URLs
const (
	// used in URLs
	DisplayController = "C"
	DisplayPractice   = "P"
	DisplayQuizmaster = "Q"
	DisplayScoreboard = "S"
	DisplayReplica    = "R"

	// fixed in page
	DisplayScorer = "X"
)

// Routes registers all web handlers.
func (app *Application) Routes() http.Handler {

	var adminHs, dynHs, organiserHs, publicHs, puppetHs, staticHs, teamHs alice.Chain

	commonHandlers := alice.New(secureHeaders, app.noQuery, wwwRedirect)

	// dynamic and static page handlers
	if app.isOnline {
		dynHs = alice.New(app.limitPage, app.session.Enable, app.noSurf, app.authenticate, app.logRequest)
		staticHs = alice.New(app.limitFile)
	} else {
		dynHs = alice.New(app.session.Enable, app.noSurf)
		staticHs = alice.New()
	}

	// access to pages
	if app.isOnline {
		adminHs = dynHs.Append(app.requireAdmin)
		organiserHs = dynHs.Append(app.requireOrganiser)
		publicHs = dynHs.Append(app.publicHeaders)
		puppetHs = dynHs.Append(app.requirePuppet)
		teamHs = dynHs.Append(app.requireTeam)
	} else {
		adminHs = dynHs.Append(app.offlineHeaders)
		organiserHs = dynHs.Append(app.offlineHeaders)
		publicHs = dynHs.Append(app.offlineHeaders)
		puppetHs = dynHs.Append(app.offlineHeaders)
		teamHs = dynHs.Append(app.offlineHeaders)
	}

	// HttpRouter wrapped to allow middleware handlers
	router := httprouter.New()

	// panic handler
	router.PanicHandler = app.recoverPanic()

	// log rejected routes
	if app.isOnline {
		router.NotFound = app.routeNotFound()
	}

	// public pages
	router.Handler("GET", "/", publicHs.ThenFunc(app.home))
	router.Handler("GET", "/no-access", publicHs.ThenFunc(app.noAccess))
	if app.hasRemote {
		router.Handler("GET", "/join/:puppet", puppetHs.ThenFunc(app.quizJoin))
		router.Handler("GET", "/join-team/:access/:nTeam", teamHs.ThenFunc(app.respondWait))
	}

	// menus
	router.Handler("GET", "/controller", organiserHs.ThenFunc(app.controller))
	router.Handler("GET", "/displays", organiserHs.ThenFunc(app.displays))
	router.Handler("GET", "/scorers", organiserHs.ThenFunc(app.scorers))

	// setup
	router.Handler("GET", "/setup-questions/:nRound", adminHs.ThenFunc(app.getFormQuestions))
	router.Handler("POST", "/setup-questions", adminHs.ThenFunc(app.postFormQuestions))
	router.Handler("GET", "/setup-quiz", adminHs.ThenFunc(app.getFormQuiz))
	router.Handler("POST", "/setup-quiz", adminHs.ThenFunc(app.postFormQuiz))
	router.Handler("GET", "/setup-rounds", adminHs.ThenFunc(app.getFormRounds))
	router.Handler("POST", "/setup-rounds", adminHs.ThenFunc(app.postFormRounds))
	router.Handler("GET", "/setup-teams", adminHs.ThenFunc(app.getFormTeams))
	router.Handler("POST", "/setup-teams", adminHs.ThenFunc(app.postFormTeams))

	// upload media files
	router.Handler("POST", "/upload", adminHs.ThenFunc(app.postFormMedia))

	// controller
	router.Handler("POST", "/control-change", organiserHs.ThenFunc(app.controlChange))
	router.Handler("POST", "/control-puppet", puppetHs.ThenFunc(app.controlPuppet))
	router.Handler("POST", "/control-update", organiserHs.ThenFunc(app.controlUpdate))
	router.Handler("POST", "/control-resume", organiserHs.ThenFunc(app.controlResume))
	router.Handler("POST", "/control-start", organiserHs.ThenFunc(app.controlStart))
	router.Handler("POST", "/control-step", organiserHs.ThenFunc(app.controlStep))

	// monitor
	router.Handler("GET", "/monitor-displays", adminHs.ThenFunc(app.monitorDisplays))
	router.Handler("POST", "/monitor-update", adminHs.ThenFunc(app.monitorUpdate))

	// team responses
	if app.hasRemote {
		router.Handler("GET", "/respond-round/:access/:nTeam/:nRound", teamHs.ThenFunc(app.getFormResponses))
		router.Handler("POST", "/respond-round", teamHs.ThenFunc(app.postFormResponses))
		router.Handler("GET", "/respond-wait/:access/:nTeam", teamHs.ThenFunc(app.respondWait))
	}

	// scorer
	if app.hasRemote {
		router.Handler("POST", "/confirm-question", organiserHs.ThenFunc(app.confirmQuestion))
		router.Handler("GET", "/score-question/:nQuestion", organiserHs.ThenFunc(app.getFormScoreQuestion))
		router.Handler("POST", "/score-question", organiserHs.ThenFunc(app.postFormScoreQuestion))
		router.Handler("GET", "/scorer-questions/:nRound", organiserHs.ThenFunc(app.scorerQuestions))
		router.Handler("GET", "/scorer-rounds", organiserHs.ThenFunc(app.scorerRounds))
	}
	router.Handler("GET", "/edit-scores/:nRound", organiserHs.ThenFunc(app.getFormEditScores))
	router.Handler("POST", "/edit-scores", organiserHs.ThenFunc(app.postFormEditScores))
	router.Handler("POST", "/publish-round", organiserHs.ThenFunc(app.publishRound))
	router.Handler("GET", "/score-round/:nRound", organiserHs.ThenFunc(app.getFormEnterScores))
	router.Handler("POST", "/score-round", organiserHs.ThenFunc(app.postFormEnterScores))
	router.Handler("GET", "/scorer-summary", organiserHs.ThenFunc(app.scorerSummary))

	// displays
	router.Handler("GET", "/quiz-answers/:puppet/:nRound", puppetHs.ThenFunc(app.quizAnswers))
	router.Handler("GET", "/quiz-final/:puppet/:nRound", puppetHs.ThenFunc(app.quizFinal))
	router.Handler("GET", "/quiz-questions/:puppet/:nRound", puppetHs.ThenFunc(app.quizQuestions))
	router.Handler("GET", "/quiz-scores/:puppet/:nRound", puppetHs.ThenFunc(app.quizScores))
	router.Handler("GET", "/quiz-start/:puppet/:nPage", puppetHs.ThenFunc(app.quizStart))
	router.Handler("GET", "/quiz-static/:puppet/:nPage", puppetHs.ThenFunc(app.quizStatic))
	router.Handler("GET", "/quiz-wait/:puppet/:nRound", puppetHs.ThenFunc(app.quizWait))

	router.Handler("GET", "/quizmaster-responses/:puppet/:nRound", organiserHs.ThenFunc(app.quizmasterResponses))
	router.Handler("GET", "/quizmaster-wait/:puppet/:nRound", organiserHs.ThenFunc(app.quizmasterWait))

	router.Handler("GET", "/scoreboard-start/:puppet/:nRound", organiserHs.ThenFunc(app.scoreboardStart)) // also for quizmaster
	router.Handler("GET", "/scoreboard-wait/:puppet/:nRound", organiserHs.ThenFunc(app.scoreboardWait))

	if app.isOnline {
		router.Handler("GET", "/usage-days", adminHs.ThenFunc(app.usageDays))
		router.Handler("GET", "/usage-months", adminHs.ThenFunc(app.usageMonths))
	}

	if app.isOnline {
		// user management
		router.Handler("GET", "/edit-users", adminHs.ThenFunc(app.users.GetFormEdit))
		router.Handler("POST", "/edit-users", adminHs.ThenFunc(app.users.PostFormEdit))

		// user authentication
		router.Handler("GET", "/user/login", dynHs.ThenFunc(app.users.GetFormLogin))
		router.Handler("POST", "/user/login", dynHs.Append(app.limitLogin).ThenFunc(app.login))
		router.Handler("POST", "/user/logout", dynHs.Append(app.requireAuthentication).ThenFunc(app.logout))
		router.Handler("GET", "/user/signup", dynHs.ThenFunc(app.users.GetFormSignup))
		router.Handler("POST", "/user/signup", dynHs.Append(app.limitLogin).ThenFunc(app.users.PostFormSignup))
	}

	// these are just a courtesy, say no immediately instead of redirecting to "/path/" first
	router.Handler("GET", "/media", http.NotFoundHandler())
	router.Handler("GET", "/static", http.NotFoundHandler())

	// file systems that block directory listing
	fsMedia := noDirFileSystem{http.Dir(MediaPath)}
	fsStatic := noDirFileSystem{http.FS(app.StaticFS)}

	// serve static files and pictures
	router.Handler("GET", "/media/*filepath", staticHs.Then(http.StripPrefix("/media", http.FileServer(fsMedia))))
	router.Handler("GET", "/static/*filepath", staticHs.Then(http.StripPrefix("/static", http.FileServer(fsStatic))))

	// files that must be in root
	fsImages, _ := fs.Sub(app.StaticFS, "images")
	fsRoot := http.FS(fsImages)
	router.Handler("GET", "/robots.txt", staticHs.Then(http.FileServer(fsStatic)))
	router.Handler("GET", "/apple-touch-icon.png", staticHs.Then(http.FileServer(fsRoot)))
	router.Handler("GET", "/favicon.ico", staticHs.Then(http.FileServer(fsRoot)))

	// return 'standard' middleware chain followed by router
	return commonHandlers.Then(router)
}

// getTeamURL returns the web address for a team to join the quiz.
func (app *Application) getTeamURL(access string, teamId int64) string {

	// use first domain, if we have one
	var d string
	if len(app.cfg.Domains) > 0 {
		d = app.cfg.Domains[0]
	}

	return "https://" + path.Join(d, "join-team", access, strconv.FormatInt(teamId, 10))
}

// pathToPage returns a display path
func pathToPage(route string, display string, nPage int, index int) string {

	// URL
	path := path.Join("/", route, display, strconv.Itoa(nPage))

	// add slide index
	if index > 0 {
		path = path + "#slide-" + strconv.Itoa(index)
	}

	return path
}

// pathToRespond returns the path to a page
func pathToRespond(access string, nTeam int, nRound int) string {
	return path.Join("/respond-round", access, strconv.Itoa(nTeam), strconv.Itoa(nRound))
}

// pathToRespondWait returns the path to a page
func pathToRespondWait(access string, nTeam int) string {
	return path.Join("/respond-wait", access, strconv.Itoa(nTeam))
}

// pathToScore returns a response path for the scorers
func pathToScore(round int) string {

	return path.Join("/scorer-questions", strconv.Itoa(round))
}
