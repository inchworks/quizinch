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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"inchworks.com/quiz/internal/forms"
	"inchworks.com/quiz/internal/models"
)

// authenticatedRole returns the role for an authenticated user, or -1
func (app *Application) authenticatedRole(r *http.Request) int {

	auth, ok := r.Context().Value(contextKeyUser).(AuthenticatedUser)
	if !ok {
		return -1
	}
	return auth.role
}

// authenticatedUser returns the ID for and authenticated
func (app *Application) authenticatedUser(r *http.Request) int64 {

	auth, ok := r.Context().Value(contextKeyUser).(AuthenticatedUser)
	if !ok {
		return 0
	}

	// active user?
	if auth.role >= models.UserTeam {
		return auth.id
	} else {
		return 0
	}
}

// cacheTeams sets the access codes for teams to join the quiz
func (app *Application) cacheTeams() {

	// team access codes and number of teams
	teams := app.TeamStore.All()
	app.NTeams = len(teams)
	app.teamAccess = make(map[string]string, app.NTeams)
	for _, t := range teams {
		nTeam := strconv.FormatInt(t.Id, 10)
		app.teamAccess[nTeam] = t.Access
	}
}

// checkAccessPuppet validates a visitor access token.
func (app *Application) checkAccessPuppet(t string) bool {

	// access to main display?
	if t == app.quizState.quizCached.Access {

		// we don't want people polling the server until the quiz has started
		return app.displayState.contest.Live

	} else {
		return false
	}
}

// checkAccessPuppet validates a visitor access token.
func (app *Application) checkAccessTeam(t string, team string) bool {

	if app.teamAccess[team] == t {

		// we don't want people polling the server until the quiz has started
		return app.displayState.contest.Live

	} else {
		return false
	}
}

// The following functions return status code and corresponding description HTTP client.
// They just make the code a bit easier to read.
// BadRequest and ServerError indicate faults with the Quiz software,
// on the client and server sides respectively, and so should be logged.
// The other errors should be detected and reported nicely when they are genuine user errors,
// but can occur from e.g. old URLs being re-requested and then a direct HTTP error is good enough.

func (app *Application) httpBadRequest(w http.ResponseWriter, err error) {

	app.log(err)
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

func httpServerError(w http.ResponseWriter) {

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func httpTooLarge(w http.ResponseWriter) {

	http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
}

// Check if request is by an authenticated active user (saved in context from contest)

func (app *Application) isAuthenticated(r *http.Request, minRole int) bool {

	auth, ok := r.Context().Value(contextKeyUser).(AuthenticatedUser)
	if !ok {
		return false
	}
	return auth.role >= minRole
}

// Get integer value of parameter

func (app *Application) intParam(r *http.Request, s string) int {

	i, err := strconv.Atoi(r.FormValue(s))
	if err != nil {
		app.log(fmt.Errorf("bad param %s : %v", s, err))
	}

	return i
}

// log records an error for debugging
func (app *Application) log(err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)
}

// option returns true if the specified option is selected.
// Comparison is case-blind.
func (cfg *Configuration) Option(opt string) bool {
	for _, o := range cfg.Options {
		if strings.EqualFold(o, opt) {
			return true
		}
	}
	return false
}

// render fetches a template from the cache and writes the result as an HTTP response.
func (app *Application) render(w http.ResponseWriter, r *http.Request, name string, td templateData) {

	// page may have no data
	if td == nil {
		td = &dataCommon{}
	}

	// common data for all pages
	td.addDefaultData(app, r, strings.HasPrefix(name, "home-"))

	// Retrieve the appropriate template set from the cache based on the page name
	// (like `home.page.tmpl`).
	ts, ok := app.templates[name]
	if !ok {
		app.log(fmt.Errorf("template %s does not exist", name))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// write template via buffer, to catch any error instead of sending a part executed page
	buf := new(bytes.Buffer)

	// Execute the template set, passing in any dynamic data.
	err := ts.Execute(buf, td)
	if err != nil {
		app.log(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// write the buffer (pass http.ResponseWriter to a func that takes an io.Writer)
	buf.WriteTo(w)
}

// reply sends a JSON reply to an AJAX request
func (app *Application) reply(w http.ResponseWriter, v interface{}) {

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		// ## Need to send JSON response with error, not a normal HTTP error, instead of panic
		panic(err)
	}
}

// validTypeCheck returns a function to check for acceptable file types
func (app *Application) validTypeCheck() forms.ValidTypeFunc {

	return func(name string) bool {
		return app.uploader.MediaType(name) != 0
	}
}