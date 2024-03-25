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

// Sequencing of quiz slides.

import (
	"strconv"

	"inchworks.com/quiz/internal/models"
)

// request from controller display to update puppets
type ReqControlIndex struct {
	Index    int
	Sync     int
	TouchNav int
}

// request from controller display to step forward/back
type ReqControlStep struct {
	Next int
	Sync int
}

// request from controller display for update
type ReqControlUpdate struct {
	Page   int
	Param  int
	Index  int
	Update int
	Second int
}

// request from puppet display for update
type ReqPuppet struct {
	Puppet string
	Access string
	Page   int
	Param  int
	Index  int
	Update int
}

// reply to display update requests (public for tester)
type RepDisplay struct {
	HRef  string `json:"newHRef"`
	Index int    `json:"newIndex"`
	Tick  string `json:"newTick"`
}

// getPuppetResponse returns the updated state for puppet display.
func (d *DisplayState) getPuppetResponse(r *ReqPuppet) RepDisplay {

	// serialisation
	defer d.updatesNone()()

	s := d.contest

	switch r.Puppet {
	default:
		// must be an access token
		if len(r.Puppet) <= 1 {
			return RepDisplay{
				HRef:  "/error",
				Index: 0,
				Tick:  "",
			}
		} else if r.Page == models.PageRespondWait {
			return d.getPuppetRespond(r.Access, r.Param, r.Index)
		}
		fallthrough

	case DisplayReplica:
		if !s.Live {
			return d.getPuppetPractice()
		} else if s.CurrentPage == models.PageStatic {
			return d.getPuppetMainStatic(r.Page, r.Param, r.Puppet)
		} else {
			return d.getPuppetMainRound(r.Page, r.Param, r.Update, r.Puppet)
		}

	case DisplayQuizmaster:
		return d.getPuppetQuizmaster(r.Page, r.Param, r.Index, r.Update)

	case DisplayScoreboard:
		if s.Live {
			return d.getPuppetScoreboard(r.Page, r.Param, r.Update)
		} else {
			return d.getPuppetPractice()
		}

	case DisplayScorer:
		return d.getPuppetScorer(r.Page, r.Param, r.Update)
	}
}

// Get update for controller display

// ## Don't seem to use all the request parameters!

func (d *DisplayState) getUpdateResponse(r *ReqControlUpdate) RepDisplay {

	// serialisation
	defer d.updatesDisplays()()
	var contestUpdated bool

	s := d.contest

	var newUrl string

	// if we are waiting for scores updates
	switch s.CurrentPage {

	// showing or waiting for scores
	case models.PageScoresWait, models.PageScores, models.PageFinal:

		// We don't trust the request to supply the page or round,
		// as there is a small chance this request was sent just before the controller changed the page.
		nRound := s.CurrentRound

		// check if scores have changed (either new scores available, or a revision)
		if r.Update != d.app.quizState.publishedUpdate {
			oldPage := s.CurrentPage

			// reload page with new scores (may advance LeaderboardIndex)
			newUrl = pathToPage(d.scoresOrWait(nRound), DisplayController, nRound, 0)
			contestUpdated = true

			// Ensure index is 0 for a new page. This is important because the leaderboard may refresh
			// (and look at CurrentIndex) before the controller display reloads.
			if s.CurrentPage != oldPage {
				s.CurrentIndex = 0
			}
		}

	default:
		// nothing to do
	}

	// update the text shown on puppets to show they are live
	newTick := ""
	if d.updateTick(r.Second) {
		newTick = s.Tick
		contestUpdated = true
	}

	// cancel database update if nothing changed
	if !contestUpdated {
		d.cancelUpdate()
	}

	return RepDisplay{
		HRef:  newUrl,
		Index: s.CurrentIndex,
		Tick:  newTick,
	}
}

// pageBack selects the previous quiz page. It returns a reply containing the route.
func (d *DisplayState) pageBack(sync int) RepDisplay {

	// serialisation
	defer d.updatesAll()() // only for prepareForScores :-(

	// controller must be synchronised with server
	if r := d.checkSync(sync); r != nil {
		return *r
	}

	qs := &d.app.quizState
	s := d.contest

	page := s.CurrentPage
	nRound := s.CurrentRound
	nRounds := qs.nFullRounds

	var route string
	var param int

	switch page {

	case models.PageStatic:

		if s.CurrentStatic == models.StaticEnd {

			if d.roundFormat.combined {
				// back to sudden death round
				route = `quiz-questions`
				s.CurrentPage = models.PageQuestions
				d.allowResponses()

			} else if nRound != 1 {

				// assume end of quiz
				nRound = nRounds + qs.nTieRounds
				route = d.scoresOrWait(nRound)
				d.setRound(nRound)
			}

		} else if s.CurrentStatic == models.StaticInterval {
			// scores before interval
			route = d.scoresOrWait(nRound)

		} else {
			// still the start
			route = `quiz-start`
		}

	case models.PageQuestions:

		nDefer := d.deferAnswers(nRound, nRounds)

		if nRound == 1 {

			// start of quiz
			route = `quiz-start`
			s.CurrentPage = models.PageStatic
			s.CurrentStatic = models.StaticStart

		} else if nRound <= nDefer+1 {
			 // answers were deferred for previous round
			route = `quiz-questions`
			d.setRound(nRound - 1) // answers were deferred
			d.allowResponses()

		} else {
			// usually scores ..
			nRound = nRound - nDefer - 1
			route = d.scoresOrWait(nRound)
			d.setRound(nRound)

			if d.roundFormat.interval {
				// .. but scores were followed by an interval
				route = `quiz-static`
				s.CurrentPage = models.PageStatic
				s.CurrentStatic = models.StaticInterval
			}
		}

		// scorers to see that questions are closed
		qs.changedScorer()

	case models.PageAnswers:

		// roll back scores
		// Note that this step backwards will be seen by quizmaster.
		d.prepareForScores(nRound - 1)

		nDefer := d.deferAnswers(nRound, nRounds)

		if (nRound > nRounds) || (nRound <= nRounds-nDefer) {

			// previous question round
			route = `quiz-questions`
			s.CurrentPage = models.PageQuestions
			d.setRound(nRound + nDefer)
			d.allowResponses()

		} else {

			// answers preceeding deferred answers
			nRound = nRound - 1
			route = d.scoresOrWait(nRound)
			d.setRound(nRound)
		}

	case models.PageFinal, models.PageScores, models.PageScoresWait:

		route = `quiz-answers`
		s.CurrentPage = models.PageAnswers

		// ready to show the scores, if available
		// (we should have already done this, but redo it - in case we are stepping back because something went wrong)
		d.prepareForScores(nRound)

	default:
		route = `quiz-start`
		s.CurrentPage = models.PageStatic
		s.CurrentStatic = models.StaticStart
	}

	// parameter for page
	if s.CurrentPage == models.PageStatic {
		param = s.CurrentStatic
	} else {
		param = s.CurrentRound
	}

	// start of page (we always skip back to the start of the previous page)
	s.CurrentIndex = 0

	d.setPrompts()

	// controller must resynchronise
	d.changedSync()

	// route for controller redirection
	return RepDisplay{HRef: pathToPage(route, DisplayController, param, 0)}
}

// pageNext selects the next quiz page. It returns a reply containing the route.
func (d *DisplayState) pageNext(sync int) RepDisplay {

	// serialisation
	defer d.updatesAll()() // needed for allowResponses, prepareForScores :-(

	// controller must be synchronised with server
	if r := d.checkSync(sync); r != nil {
		return *r
	}

	qs := &d.app.quizState
	s := d.contest

	nRound := s.CurrentRound

	var route string
	var param int

	switch s.CurrentPage {

	case models.PageStatic:

		if s.CurrentStatic == models.StaticStart {
			// start quiz
			route = `quiz-questions`
			s.CurrentPage = models.PageQuestions
			d.setRound(1)
			d.allowResponses()

		} else if s.CurrentStatic == models.StaticInterval {
			// interval followed scores
			route = d.afterScores(nRound)

		} else {
			// still the end
			route = `quiz-static`
		}

	case models.PageQuestions:

		nDefer := d.deferAnswers(nRound, qs.nFullRounds)

		if nRound <= nDefer {
			route = `quiz-questions`
			d.setRound(nRound + 1) // deferring answers
			d.allowResponses()

		} else if d.roundFormat.combined {

			// ## this is getting too complicated for an unikely case :-(
			// ## only one sudden death, and could just count number of tie breakers?
			if nRound < qs.nFullRounds+qs.nTieRounds {

				// start next tie breaker
				route = `quiz-questions`
				s.CurrentPage = models.PageQuestions
				d.setRound(nRound + 1)
				d.allowResponses()

			} else {

				// quiz is over - there is nothing left
				route = `quiz-static`
				s.CurrentPage = models.PageStatic
				s.CurrentStatic = models.StaticEnd
			}

		} else {
			route = `quiz-answers`
			s.CurrentPage = models.PageAnswers
			nRound -= nDefer
			d.setRound(nRound)

			// ready to show the scores, if available
			d.prepareForScores(nRound)
		}

		// scorers to see that questions are closed
		qs.changedScorer()

	case models.PageAnswers:

		// check if scores are published yet
		route = d.scoresOrWait(nRound)

	case models.PageFinal, models.PageScores, models.PageScoresWait:

		if d.roundFormat.interval {
			// interval following scores
			route = `quiz-static`
			s.CurrentPage = models.PageStatic
			s.CurrentStatic = models.StaticInterval
		} else {
			// whatever's next
			route = d.afterScores(nRound)
		}

	default:

		route = `quiz-static`
		s.CurrentPage = models.PageStatic
		s.CurrentStatic = models.StaticEnd
	}

	// parameter for page
	if s.CurrentPage == models.PageStatic {
		param = s.CurrentStatic
	} else {
		param = s.CurrentRound
	}

	// start of page
	s.CurrentIndex = 0

	d.setPrompts()

	// controller must resynchronise
	d.changedSync()

	// route for redirection
	return RepDisplay{HRef: pathToPage(route, DisplayController, param, 0)}
}

// Resume quiz at current page
//
// Returns route for redirection.

func (d *DisplayState) resumeQuiz() string {

	// serialisation
	defer d.updatesDisplays()()

	s := d.contest
	page := s.CurrentPage

	// set round format
	d.setRound(s.CurrentRound)

	// re-enable touchscreen, because we may be resuming on a different device
	s.TouchController = true

	var route string
	var param int

	switch page {

	case models.PageStatic:
		switch s.CurrentStatic {

		case models.StaticStart:
			route = `quiz-start`

		case models.StaticInterval, models.StaticEnd:
			route = `quiz-static`

		default:
			route = `puppet-error`
		}

	case models.PageQuestions:
		route = `quiz-questions`

	case models.PageAnswers:
		route = `quiz-answers`

	case models.PageFinal:
		route = `quiz-final`

	case models.PageScores:
		route = `quiz-scores`

	case models.PageScoresWait:
		route = `quiz-wait`

	default:
		route = `puppet-error`
	}

	d.setPrompts()

	// parameter for page
	if s.CurrentPage == models.PageStatic {
		param = s.CurrentStatic
	} else {
		param = s.CurrentRound
	}

	return pathToPage(route, DisplayController, param, 0)
}

// setPrompts makes prompter text for the quizmaster.
func (d *DisplayState) setPrompts() {

	qs := &d.app.quizState
	s := d.contest
	nRound := s.CurrentRound
	nStatic := s.CurrentStatic

	switch s.CurrentPage {

	case models.PageStatic:

		// start quiz
		if nStatic == models.StaticStart {
			d.doNow = "Welcome"
			d.doNext = prompt('Q', nRound)
		} else if nStatic == models.StaticInterval {
			nDefer := d.deferAnswers(nRound, qs.nFullRounds)
			d.doNow = "Interval"
			d.doNext = prompt('Q', nRound + nDefer + 1)
		} else {
			d.doNow = "End"
			d.doNext = ""
		}

	case models.PageQuestions:

		nDefer := d.deferAnswers(nRound, qs.nFullRounds)

		if nRound <= nDefer {
			// answers deferred
			d.doNow = prompt('Q', nRound)
			d.doNext = prompt('Q', nRound+1)

		} else if d.roundFormat.combined {
			// sudden death tie break
			// ## assumes sudden-death only used for tie breaks
			d.doNow = prompt('D', nRound-qs.nFullRounds)

			if nRound < qs.nFullRounds+qs.nTieRounds {
				// followed by another one
				d.doNext  = prompt('D', nRound-qs.nFullRounds+1)
			} else {
				// quiz is over - there is nothing left
				d.doNext = "Goodbye"
			}

		} else if nRound <= qs.nFullRounds {
			// questions followed by answers for earlier round
			d.doNow = prompt('Q', nRound)
			d.doNext  = prompt('A', nRound-nDefer)

		} else {
			// tie-break questions followed by answers
			d.doNow = prompt('T', nRound-qs.nFullRounds)
			d.doNext  = prompt('U', nRound-qs.nFullRounds)
		}

	case models.PageAnswers:

		// scores always follow answers
		if nRound <= qs.nFullRounds {
			// normal round
			d.doNow = "Answers"
			d.doNext  = "Scores"

		} else {
			// tie-break
			d.doNow = prompt('U', nRound-qs.nFullRounds)
			d.doNext  = prompt('V', nRound-qs.nFullRounds)
		}
	case models.PageScoresWait:

		// scores followed by ..
		d.doNow = "Waiting"
		nDefer := d.deferAnswers(nRound, qs.nFullRounds)

		if d.roundFormat.interval {
			// .. interval
			d.doNext = "Interval"

		} else if nRound < qs.nFullRounds-nDefer {
			// .. next round questions
			d.doNext  = prompt('Q', nRound+nDefer+1)

		} else if nRound < qs.nFullRounds {
			// .. catchup round answers
			d.doNext  = prompt('A', nRound+1)
		} else {
			d.doNext = "Goodbye"
		}

	case models.PageScores:

		// scores followed by ..
		d.doNow = "Scores"
		nDefer := d.deferAnswers(nRound, qs.nFullRounds)

		if d.roundFormat.interval {
			// .. interval
			d.doNext = "Interval"

		} else if nRound < qs.nFullRounds-nDefer {
			// .. next round questions
			d.doNext  = prompt('Q', nRound+nDefer+1)

		} else {
			// .. catchup round answers
			d.doNext  = prompt('A', nRound+1)
		}

	case models.PageFinal:

		if nRound == qs.nFullRounds {
			// final scores followed by tie-break or end
			d.doNow = "Scores"
			if d.isTied && qs.nTieRounds > 0 {
				d.doNext = prompt('T', 1) // need a tie-break, if we have one
			} else {
				d.doNext = "Goodbye"
			}

		} else if nRound < qs.nFullRounds+qs.nTieRounds {
			// tie-break scores followed by another tie break, or end
			nTie := nRound-qs.nFullRounds
			d.doNow = prompt('V', nTie)
			if d.isTied && nTie < qs.nTieRounds {
				d.doNext = prompt('T', nTie+1) // need another tie-break, if we have one
			} else {
				d.doNext  = "Goodbye"
			}
		}

	default:
		d.doNow = "End"
		d.doNext = ""
	}

	d.changedQuizmaster() // update quizmaster's display
}

// Set current position for puppet displays

func (d *DisplayState) setPuppet(r *ReqControlIndex) RepDisplay {

	// serialisation
	defer d.updatesDisplays()()

	// controller must be synchronised with server
	if r := d.checkSync(r.Sync); r != nil {
		return *r
	}

	s := d.contest

	// note new position
	s.CurrentIndex = r.Index

	// turn off controller touch buttons if not needed
	// Note that the only way to turn it back on is to restart or resume the quiz.
	if s.TouchController && r.TouchNav == 0 {
		s.TouchController = false
	}

	return RepDisplay{} // no controller action needed
}

// StartQuiz starts a quiz contest. It returns the initial route for the contoller.
func (d *DisplayState) startQuiz(live bool) string {

	// serialisation
	qs := &d.app.quizState
	defer qs.updatesQuiz()()

	// reset scores and restart displays
	qs.restartQuiz()
	d.setPrompts()

	// no tick available until controller starts to send it
	d.contest.Tick = "*"

	// select live or practice mode
	d.contest.Live = live

	// reload the current team access tokens, in case they have been updated
	// Team access is not allowed until the quiz has started, so it doesn't matter if they were out of date earlier.
	d.app.cacheTeams()

	// first page for controller
	return pathToPage(`quiz-start`, DisplayController, 0, 0)
}

// ==== Support Functions ====

// afterScores steps to the next page following scores or the interval. It returns the route.
func (d *DisplayState) afterScores(nRound int) (route string) {

	qs := &d.app.quizState
	s := d.contest

	nDefer := d.deferAnswers(nRound, qs.nFullRounds)

	if nRound < qs.nFullRounds-nDefer {

		// start next round
		route = `quiz-questions`
		s.CurrentPage = models.PageQuestions
		d.setRound(nRound + nDefer + 1)
		d.allowResponses()

	} else if nRound < qs.nFullRounds {

		// deferred round answers
		route = `quiz-answers`
		s.CurrentPage = models.PageAnswers
		nRound++
		d.setRound(nRound)

		// ready to show the scores, if available
		d.prepareForScores(nRound)

	} else if nRound < qs.nFullRounds+qs.nTieRounds {

		// start next tie breaker
		route = `quiz-questions`
		s.CurrentPage = models.PageQuestions
		d.setRound(nRound + 1)
		d.allowResponses()

	} else {

		// quiz is over - there is nothing left
		route = `quiz-static`
		s.CurrentPage = models.PageStatic
		s.CurrentStatic = models.StaticEnd
	}
	return
}

// allowResponses sets the quiz state to accept team responses
func (d *DisplayState) allowResponses() {

	qs := &d.app.quizState

	qs.quizCached.ResponseRound = d.contest.CurrentRound
	qs.dirtyQuiz = true
}


// checkSync verifies that the controller is synchronised with the server.
// If not, it returns a resynchronisation response for the controller.
func (d *DisplayState) checkSync(sync int) *RepDisplay {
	if sync == d.syncUpdate {
		return nil // ok
	}

	// treat the controller like a puppet
	s := d.contest
	var rd RepDisplay
	if s.CurrentPage == models.PageStatic {
		rd = d.getPuppetMainStatic(-1, -1, DisplayController)
	} else {
		rd = d.getPuppetMainRound(-1, -1, -1, DisplayController)
	}
	return &rd
}


// Return the number of rounds for which answers are deferred

func (d *DisplayState) deferAnswers(nRound int, nRounds int) int {

	if nRound <= nRounds {
		return d.app.quizState.nDeferAnswers
	} else {
		return 0 // tie-breaker round
	}
}

// Get response for puppet main display : round-specific page
// ## Could take unmarshalled JSON as a single struct param?
//  ## -> sessionManager??

func (d *DisplayState) getPuppetMainRound(page int, param int, update int, puppet string) RepDisplay {

	s := d.contest
	currentPage := s.CurrentPage
	currentRound := s.CurrentRound
	var newUrl string
	var newIndex int

	// check that if we are on the current page, the current round, and the latest scores
	if (page != currentPage) ||
		(param != currentRound) ||
		(d.isScoresPage(page) && (update != d.app.quizState.publishedUpdate)) {

		var route string

		switch currentPage {

		case models.PageQuestions:
			route = `quiz-questions`

		case models.PageAnswers:
			route = `quiz-answers`

		case models.PageFinal:
			route = `quiz-final`

		case models.PageScores:
			route = `quiz-scores`

		case models.PageScoresWait:
			route = `quiz-wait`

		default:
			// #### need an error page
			route = `quiz-error`
		}

		// make up the URL
		newUrl = pathToPage(route, puppet, currentRound, s.CurrentIndex)

		// cannot give the current index yet, we must wait for the page to reload
		newIndex = 0

	} else {
		// stay on current page
		newUrl = ""
		newIndex = s.CurrentIndex
	}

	newTick := ""
	if currentPage == models.PageScoresWait {
		newTick = s.Tick
	}

	return RepDisplay{
		HRef:  newUrl,
		Index: newIndex,
		Tick:  newTick,
	}
}

// Get response for puppet main display : static page

func (d *DisplayState) getPuppetMainStatic(page int, param int, puppet string) RepDisplay {

	s := d.contest
	currentStatic := s.CurrentStatic
	var newUrl string
	var newIndex int

	if (page != models.PageStatic) || (param != currentStatic) {

		var route string

		switch currentStatic {
		case models.StaticStart:
			route = `quiz-start`

		case models.StaticInterval, models.StaticEnd:
			route = `quiz-static`

		default:
			// ## need an error page
			route = `quiz-error`
		}

		// make up the URL
		newUrl = pathToPage(route, puppet, currentStatic, s.CurrentIndex)

		// cannot give the current index yet, we must wait for the page to reload
		newIndex = 0

	} else {
		// stay on current page
		newUrl = ""
		newIndex = s.CurrentIndex
	}

	return RepDisplay{
		HRef:  newUrl,
		Index: newIndex,
		Tick:  s.Tick,
	}
}

// Get response for puppet display in practice mode

func (d *DisplayState) getPuppetPractice() RepDisplay {

	// stay on current page
	return RepDisplay{
		HRef:  "",
		Index: 0,
		Tick:  d.contest.Tick,
	}
}

// Get response for puppet quizmaster's scores
//
// This doesn't follow the main display - it changes on the answer round.
// The quizmaster can step between teams ordered by round and ordered by rank.

func (d *DisplayState) getPuppetQuizmaster(page int, round int, index int, update int) RepDisplay {

	// change the round shown to quizmaster on the answer page
	qs := &d.app.quizState
	qc := qs.quizCached
	s := d.contest
	quizmasterRound := s.QuizmasterRound

	var newRoute string
	var newPage int
	var newIndex int
	var newURL string
	newUpdate := d.quizmasterUpdate

	if d.app.isOnline && s.CurrentPage == models.PageQuestions {

		// displaying questions - show quizmaster the team responses
		newRoute = `quizmaster-responses`
		newPage = models.PageQuizResponses
		quizmasterRound = s.CurrentRound
		newUpdate = d.app.quizState.responseUpdate

	} else if quizmasterRound == 0 {

		// starting: show just the team names
		newRoute = `scoreboard-start`
		newPage = models.PageStart

	} else if quizmasterRound <= qc.ScoringRound-1 {

		// show scores for this round
		if quizmasterRound < qs.nFullRounds {
			newRoute = `quiz-scores`
			newPage = models.PageScores
		} else {
			newRoute = `quiz-final`
			newPage = models.PageFinal
		}

	} else {

		// wait for scores for this round
		newRoute = `quizmaster-wait`
		newPage = models.PageScoresWait
	}

	if (page != newPage) ||
		(round != quizmasterRound) ||
		(update != newUpdate) {

		// reload page with a new round or updated scores
		newURL = pathToPage(newRoute, DisplayQuizmaster, quizmasterRound, 0)
		newIndex = 0
	} else {
		// no change
		newURL = ""
		newIndex = index
	}

	return RepDisplay{
		HRef:  newURL,
		Index: newIndex,
		Tick:  s.Tick,
	}
}

// getPuppetRespond returns an update response for a team waiting to answer questions
func (ds *DisplayState) getPuppetRespond(access string, nTeam int, index int) RepDisplay {

	qs := &ds.app.quizState
	round := qs.quizCached.ResponseRound

	var newURL string

	// are we showing questions, and has team not answered?
	if ds.contest.CurrentPage == models.PageQuestions {
		// ## better to cache team response states?
		sc, _ := qs.app.ScoreStore.ForTeamAndRound(int64(nTeam), round)
		if sc == nil {
			newURL = pathToRespond(access, nTeam, round) // enter answers
		}

	} else if index != round {
		newURL = pathToRespondWait(access, nTeam) // refresh wait page
	}

	return RepDisplay{
		HRef: newURL,
	}
}

// Get response for puppet scoreboard

func (d *DisplayState) getPuppetScoreboard(page int, round int, update int) RepDisplay {

	qs := &d.app.quizState
	s := d.contest
	scoreboardRound := s.ScoreboardRound

	var newRoute string
	var newPage int
	var newIndex int
	var newURL string
	newUpdate := update

	if scoreboardRound == 0 {

		// no scores to show yet
		newRoute = `scoreboard-start`
		newPage = models.PageStart
		newIndex = 0

	} else if s.QuizmasterRound > scoreboardRound {

		// updated scores previewed to quizmaster - mustn't leak the changes to the audience
		newRoute = `scoreboard-wait`
		newPage = models.PagePublicWait
		newIndex = 0

	} else {
		// showing scores
		if scoreboardRound < qs.nFullRounds {
			newRoute = `quiz-scores`
			newPage = models.PageScores
		} else {
			newRoute = `quiz-final`
			newPage = models.PageFinal
		}

		// follow the main scoreboard, if it is showing
		// This is the only case where score update changes the scoreboard,
		// because otherwise a refresh and change of index causes the screen to glitch.
		if d.isScoresPage(s.CurrentPage) {
			newIndex = s.CurrentIndex
			newUpdate = d.app.quizState.publishedUpdate

		} else {
			// otherwise show the leaderboard
			newIndex = s.LeaderboardIndex
		}
	}

	if (page != newPage) ||
		(round != scoreboardRound) ||
		(update != newUpdate) {

		// reload page with a new round or updated scores
		// ## Would like to do direct to target index, but heading (parent slide) doesn't appear.
		// ## I guess client must step through it.
		newURL = pathToPage(newRoute, DisplayScoreboard, scoreboardRound, 0)
	} else {
		// stay on current page
		newURL = ""
	}

	return RepDisplay{
		HRef:  newURL,
		Index: newIndex,
		Tick:  s.Tick,
	}
}

// getPuppetScorer returns an update response for scorer's displays.
func (ds *DisplayState) getPuppetScorer(page int, round int, update int) RepDisplay {

	qs := &ds.app.quizState

	var newURL string

	switch page {
	case models.PageScorerRounds:
		if update != qs.scorerUpdate {

			// reload page with updated status
			newURL = "/scorer-rounds"
		}

	case models.PageScorerQuestions:
		if update != qs.scorerUpdate {
			if round != qs.quizCached.ScoringRound {
				// return to rounds, if this isn't the scoring round
				// E.g. another scorer just published this round.
				newURL = "/scorer-rounds"
			} else {
				newURL = pathToScore(round)
			}
		}
	}

	return RepDisplay{
		HRef: newURL,
	}
}

// Is page a scores page?

func (d *DisplayState) isScoresPage(page int) bool {

	return page == models.PageScores || page == models.PageFinal
}

// setRound changes the current round. The round format is decoded and cached.
func (d *DisplayState) setRound(nRound int) {

	d.contest.CurrentRound = nRound

	// get round
	round, err := d.app.RoundStore.GetByN(nRound)
	if err != nil {
		d.app.errorLog.Print(err)
		return
	}
	d.roundFormat = decodeFormat(round.Format, d.app.cfg.SlideItems)
}

// Prepare to show scores, when answers are shown
//
// We do it now, not just before showing the scores, so that the quizmaster can see the scores while
// giving out the answers. We can't do it any earlier, or we might remove the previous round's scores
// from the quizmasters display too soon.

func (d *DisplayState) prepareForScores(nRound int) {

	q := &d.app.quizState

	// recalculate up to the current round
	if nRound < q.quizCached.ScoringRound {
		q.calculateTotalsAndRank(nRound, d.contest)
	} else {
		d.contest.QuizmasterRound = nRound // show the quizmaster that the scores are delayed
	}
}

// prompt returns the prompter text for a round
func prompt(page rune, round int) string {
	switch page {
	case 'A':
		return "R" + strconv.Itoa(round) + " Answers"
	case 'D':
		return "Tie " + strconv.Itoa(round) + " Sudden Death"
	case 'Q':
		return "R" + strconv.Itoa(round) + " Questions"
	case 'S':
		return "R" + strconv.Itoa(round) + " Scores"
	case 'T':
		return "Tie " + strconv.Itoa(round) + " Questions"
	case 'U':
		return "Tie " + strconv.Itoa(round) + " Answers"
	case 'V':
		return "Tie " + strconv.Itoa(round) + " Scores"
	default:
		return "??"
	}
}

// Quiz is retarted. Called by quizState while serialised.

func (d *DisplayState) restartQuiz(s *models.Contest) {

	d.isTied = false
	s.CurrentPage = models.PageStatic
	s.CurrentStatic = models.StaticStart
	s.QuizmasterRound = 0
	s.ScoreboardRound = 0

	// assume touchscreen required
	s.TouchController = true
}

// Choose scores or waiting page.
//
// Returns route.

func (d *DisplayState) scoresOrWait(nRound int) string {

	qs := &d.app.quizState
	qc := qs.quizCached
	s := d.contest
	scoredRound := qc.ScoringRound - 1

	if nRound <= scoredRound {

		// all displays can show scores to audience, when the controller is changing the round
		s.ScoreboardRound = nRound

		var route string
		if nRound < qs.nFullRounds {

			s.CurrentPage = models.PageScores
			route = `quiz-scores`
		} else {
			s.CurrentPage = models.PageFinal
			route = `quiz-final`
		}
		return route

	} else {
		s.CurrentPage = models.PageScoresWait
		return `quiz-wait`
	}
}

// Update tick text, used to indicate live displays

func (d *DisplayState) updateTick(nSecond int) bool {

	s := d.contest

	// every 5 seconds, to reduce overhead
	if nSecond%10 == 0 {
		s.Tick = "/"
		return true
	} else if nSecond%5 == 0 {
		s.Tick = "\\"
		return true
	} else {
		return false
	}
}
