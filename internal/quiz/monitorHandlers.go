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

// Requests for display monitoring

import (
	"net/http"

	"github.com/inchworks/webparts/monitor"
)

type RepMonitor struct {
	Displays []monitor.Monitored
}

// Quiz monitor

func (app *Application) monitorDisplays(w http.ResponseWriter, r *http.Request) {

	app.render(w, r, "monitor.page.tmpl", &dataDisplay{
		Interval:  app.cfg.MonitorInterval,
		CSRFToken: app.Token(r),
	})
}

// Get updated status

func (app *Application) monitorUpdate(w http.ResponseWriter, r *http.Request) {

	// monitor
	disps := app.Monitor.Status()

	// JSON response
	app.reply(w, RepMonitor{Displays: disps})
}
