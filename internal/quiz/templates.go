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
	"html/template"
	"net/http"
	"time"

	"github.com/inchworks/usage"
	"github.com/inchworks/webparts/multiforms"
	"github.com/inchworks/webparts/users"
	"inchworks.com/quiz/internal/forms"
	"inchworks.com/quiz/internal/models"
)

// Template data for all pages
// - implements TemplateData interface so that render() can add data without knowing which template it has.

type templateData interface {
	addDefaultData(app *Application, r *http.Request)
}

type dataCommon struct {
	Canonical  string // canonical domain
	Flash      string // flash message
	ParentHRef string

	// To configure menus and displays, not to check authorisation
	HasRemote   bool // has remote teams
	IsOnline    bool // accessed via public internet, not stand-alone server at a venue
	IsAdmin     bool // user is an administrator
	IsOrganiser bool // user is a quiz organiser
	IsTeam      bool // user is a team member
}

func (d *dataCommon) addDefaultData(app *Application, r *http.Request) {

	d.Flash = app.session.PopString(r, "flash")

	d.HasRemote = app.hasRemote

	if app.isOnline {
		d.IsOnline = true

		role := app.authenticatedRole(r)
		d.IsAdmin = role >= models.UserAdmin
		d.IsOrganiser = role >= models.UserOrganiser
		d.IsTeam = role >= models.UserTeam
	}
}

// Common template data for display pages

type dataDisplay struct {
	Puppet     string // controller or puppet display
	Access     string // access token
	Page       int    // page type
	Param      int    // parameter, page specific
	Index      int    // slide index, or 2nd parameter
	Update     int    // data update timestamp
	Monitor    int    // client monitor index
	Interval   int    // refresh interval (mS)
	Sync       int    // controller synchronisation
	Tick       string // timer tick, to stop browsers sleeping
	CSRFToken  string // CSRF token
	BreakSlide string // optional final slide
	Organiser  string // needed just for break slide
	TouchNav   string // touch control class
	DoNow      string // quizmaster prompt
	DoNext     string // quizmaster prompt
	IsLive     bool   // quiz started and not in practice mode
	dataCommon
}

type dataResponded struct {
	NRound  int
	Title   string
	ReadyTo string
	Teams   []*models.TeamResponded
	dataDisplay
}

type dataRound struct {
	Title  string
	Slides []*Slide
	Error  string
	dataDisplay
}

type dataScores struct {
	Title         string
	ScoredTo      int
	ReadyTo       string
	ScoresTop     []*models.TeamScore
	ScoresByRound []*models.TeamScore
	ScoresByRank  []*models.TeamScore
	TieBreak      bool
	dataDisplay
}

type dataStatic struct {
	Title string
	dataDisplay
}

type dataTeams struct {
	Teams   []*models.Team
	ReadyTo string
	dataDisplay
}

type dataWait struct {
	Title string
	Error string
	dataDisplay
}

type dataRespondWait struct {
	NTeam  int
	Team   string
	Rounds []*models.Round
	dataDisplay
}

type dataScorerQuestions struct {
	Title     string
	NRound    int
	Questions []*scoreQuestionData
	dataDisplay
}

type dataScorerRounds struct {
	Rounds []*scoreRoundData
	dataDisplay
}

// template data for pages that have action buttons

type dataSimple struct {
	Organiser string
	Val       string
	CSRFToken string
	dataCommon
}

// Template data for forms pages

type questionsFormData struct {
	*forms.Questions
	Title     string
	NRound    int
	MaxUpload int // in MB
	dataCommon
}

type quizFormData struct {
	*multiforms.Form
	Rounds []*models.Round
	Teams  []teamData
	dataCommon
}

type scoreQuestionFormData struct {
	*forms.ScoreQuestion
	NRound    int
	Title     string
	NQuestion int
	Order     int
	Question  string
	Answer    string
	dataCommon
}

type teamData struct {
	*models.Team
	URL string
}

type responsesFormData struct {
	*forms.Responses
	NRound int
	NTeam  int
	Access string
	Round  string
	Team   string
	dataCommon
}

type roundsFormData struct {
	*forms.Rounds
	dataCommon
}

type scoresFormData struct {
	*forms.Scores
	Action string
	Round  int
	Title  string
	dataCommon
}

type teamsFormData struct {
	*forms.Teams
	dataCommon
}

type usersFormData struct {
	Users interface{}
	dataCommon
}

// For scorers rounds

type scoreQuestionData struct {
	*models.Question
	Alert  string
	Status string
	Btn    string
}

type scoreRoundData struct {
	*models.Round
	Alert  string
	Status string
	Btn    string
}

// For scorers summary

type scoreSummaryData struct {
	Rounds []*heading
	Scores []*teamScores
	dataCommon
}

type heading struct {
	N   int
	Txt string
}

type teamScores struct {
	Name   string
	Rank   int
	Total  float64
	Rounds []float64
	dataCommon
}

// Template data for usage reports

type dataUsagePeriods struct {
	Title string
	Usage []*dataUsage
	dataCommon
}

type dataUsage struct {
	Date  string
	Stats []*usage.Statistic
}

// Define functions callable from a template

var templateFuncs = template.FuncMap{
	"humanDate":  humanDate,
	"itemNumber": itemNumber,
	"teamStatus": teamStatus,
	"userRole":   userRole,
	"userStatus": userStatus,
}

// humanDate returns a date/time in a user-friendly format
func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format("02 Jan 2006 at 15:04")
}

// itemNumber converts a sub-form index (0 based) to an item number (1 based)
func itemNumber(ix int) int {
	return ix + 1
}

// teamStatus converts a teams response status to a string
func teamStatus(r bool) (s string) {

	if r {
		s = "yes"
	} else {
		s = "waiting"
	}
	return
}

// convert user's role to string

func userRole(n int) (s string) {

	switch n {
	case models.UserAudience:
		s = "audience"

	case models.UserTeam:
		s = "team"

	case models.UserOrganiser:
		s = "organiser"

	case models.UserAdmin:
		s = "admin"

	default:
		s = "??"
	}

	return
}

// convert user's status to string

func userStatus(n int) (s string) {

	switch n {
	case users.UserSuspended:
		s = "suspended"

	case users.UserKnown:
		s = "-"

	case users.UserActive:
		s = "signed-up"

	default:
		s = "??"
	}

	return
}
