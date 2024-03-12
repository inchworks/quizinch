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

// Requests for quiz display control

import (
	"net/http"
)

// controlChange sets the current slide position, changed by the controller
func (app *Application) controlChange(w http.ResponseWriter, r *http.Request) {

	// AJAX data request
	req := ReqControlIndex{
		Index:    app.intParam(r, "index"),
		Sync:     app.intParam(r, "sync"),
		TouchNav: app.intParam(r, "touchNav"),
	}

	// save current position, for use by puppet display ..
	// .. and note if touchscreen navigation required
	rep := app.displayState.setPuppet(&req)

	// JSON response
	app.reply(w, rep)
}

// controlPuppet gets the current page or slide position, polled by puppet displays.
func (app *Application) controlPuppet(w http.ResponseWriter, r *http.Request) {

	// AJAX data request
	// Note that a jQuery AJAX request always sends data as form values, not as JSON!
	req := ReqPuppet{
		Puppet: r.FormValue("puppet"),
		Access: r.FormValue("access"),
		Page:   app.intParam(r, "page"),
		Param:  app.intParam(r, "param"),
		Index:  app.intParam(r, "index"),
		Update: app.intParam(r, "update"),
	}

	// monitor
	m := app.intParam(r, "monitor")
	app.Monitor.Alive(m)

	// get current position, for puppet display
	rep := app.displayState.getPuppetResponse(&req)

	// JSON response
	app.reply(w, rep)
}

// Check for update, polled by controller display

func (app *Application) controlUpdate(w http.ResponseWriter, r *http.Request) {

	// AJAX data request
	req := ReqControlUpdate{
		Page:   app.intParam(r, "page"),
		Param:  app.intParam(r, "param"),
		Index:  app.intParam(r, "index"),
		Update: app.intParam(r, "update"),
		Second: app.intParam(r, "second"),
	}

	// monitor
	m := app.intParam(r, "monitor")
	app.Monitor.Alive(m)

	// check for update (typically while waiting for scores)
	rep := app.displayState.getUpdateResponse(&req)

	// JSON response
	app.reply(w, rep)
}

// controlStep handles a forward/back arrow or button pressed in the browser.
func (app *Application) controlStep(w http.ResponseWriter, r *http.Request) {

	// rantMode=on
	// It's when we get here that we realise that the web is just one cludge piled on another.
	// An "AJAX" request sends a form, not JSON. But although it looks like a normal form submission
	// to the server, we can't reply with a usual redirect because the browser will respond to
	// the redirect and then return the contents of the page to the client AJAX hander. Instead
	// we must return some JSON containing the new path, and have the AJAX hander go the the next page.
	// rantMode=off

	req := ReqControlStep{
		Next: app.intParam(r, "next"),
		Sync: app.intParam(r, "sync"),
	}

	var rep RepDisplay
	if req.Next > 0 {
		rep = app.displayState.pageNext(req.Sync)
	} else {
		rep = app.displayState.pageBack(req.Sync)
	}

	// JSON response
	app.reply(w, rep)
}
