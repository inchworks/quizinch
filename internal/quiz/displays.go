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

// Processing related to quiz displays
//
// These functions should not modify application state.

package quiz

import (
	"html/template"
	"strconv"
	"strings"

	"github.com/inchworks/webparts/uploader"
	"inchworks.com/quiz/internal/models"
)

// Data returned to web request handlers
type Slide struct {
	NSlide    int
	Questions []*Question
}

type Question struct {
	Question template.HTML
	Answer   template.HTML
	File     string
	Type     string
}

// roundFormat is the decoded format for a round.
type roundFormat struct {
	slideQuestions int    // max per slide
	slideAnswers   int    // max per slide
	combined       bool   // combined Q&A
	interval       bool   // follow scores by interval
	additional     string // slide for interval, end of quiz, etc.
}

// Leaderboard for current round. Returns scores in rank order, with teams.
//
// Only shown while waiting for scores, to quizmaster.

func (d *DisplayState) displayLeaderboard(nRound int, puppet string) *dataScores {

	// serialisation
	defer d.updatesNone()()

	// round
	round, err := d.app.RoundStore.GetByN(nRound)
	if err != nil {
		return nil
	}

	// teams sorted by overall rank, up to previous round
	scoreStore := d.app.ScoreStore
	scoresRanked := scoreStore.ForRoundByRank(nRound-1, d.app.NTeams)

	// update state
	var update int
	if puppet == `Q` {
		update = d.quizmasterUpdate
	} else {
		update = d.app.quizState.publishedUpdate
	}
	
	return &dataScores{
		Title:        round.Title,
		ScoredTo:     d.scoredTo(puppet),
		ReadyTo:      d.readyTo(),
		ScoresByRank: scoresRanked,
		dataDisplay: dataDisplay{

			Index:      d.slideIndex(puppet),
			Update:     update,
			Tick:       d.contest.Tick,
			BreakSlide: d.app.displayState.roundFormat.additional,
			Organiser:  d.app.quizState.quizCached.Organiser,
			TouchNav:   d.navigation(puppet),
			DoNow:      d.doNow,
			DoNext:     d.doNext,
		},
	}
}

// displayResponses returns team response statuses for the specified round.
//
// Only shown while waiting for scores, to quizmaster.
func (d *DisplayState) displayResponses(nRound int, puppet string) *dataResponded {

	// serialisation
	defer d.updatesNone()()

	// round
	r, err := d.app.RoundStore.GetByN(nRound)
	if err != nil {
		return nil
	}

	// teams with response status
	ts := d.app.TeamStore.AllWithStatus(r.Id)

	return &dataResponded{
		Title:   r.Title,
		NRound:  nRound,
		ReadyTo: d.readyTo(),
		Teams:   ts,
		dataDisplay: dataDisplay{

			Index:      d.slideIndex(puppet),
			Update:     d.quizmasterUpdate,
			Tick:       d.contest.Tick,
			BreakSlide: d.app.displayState.roundFormat.additional,
			Organiser:  d.app.quizState.quizCached.Organiser,
			TouchNav:   d.navigation(puppet),
			DoNow:      d.doNow,
			DoNext:     d.doNext,
		},
	}
}

// Questions or answers for current round
//
// Returns slides for round

func (d *DisplayState) displayRound(puppet string, nRound int, forQuestions bool) (string, *dataRound) {

	defer d.updatesNone()()

	// get questions
	round, err := d.app.RoundStore.GetByN(nRound)
	if err != nil {
		return "quiz-error.page.tmpl", &dataRound{
			Error: "This round is undefined (" + err.Error() + ").",
		}
	}

	var slides []*Slide
	var split bool
	var roundTemplate string
	rf := d.app.displayState.roundFormat

	if forQuestions {
		// split questions into slides
		slides, split = d.splitRound(round.Id, rf.slideQuestions)
		if rf.combined {
			if split {
				roundTemplate = "quiz-q-a-multi.page.tmpl"
			} else {
				roundTemplate = "quiz-q-a.page.tmpl"
			}
		} else {
			if split {
				roundTemplate = "quiz-questions-multi.page.tmpl"
			} else {
				roundTemplate = "quiz-questions.page.tmpl"
			}
		}
	} else {
		// split answers into slides
		slides, split = d.splitRound(round.Id, rf.slideAnswers)
		if split {
			roundTemplate = "quiz-answers-multi.page.tmpl"
		} else {
			roundTemplate = "quiz-answers.page.tmpl"
		}
	}

	// template and its data
	return roundTemplate, &dataRound{
		Title:  round.Title,
		Slides: slides,
		dataDisplay: dataDisplay{
			Index:      d.slideIndex(puppet),
			Sync:       d.syncUpdate,
			BreakSlide: "P", // always pause after questions or answers
			Organiser:  d.app.quizState.quizCached.Organiser,
			TouchNav:   d.navigation(puppet),
		},
	}
}

// Scores for current round
//
// Omits unscored teams on a tie-break round.
//
// Data returned includes
//  - scores for last completed round, ordered by score for round (lowest first)
//  - scores for last completed round, ordered by team rank
//  - scores for top teams in reverse order (on final rounds)

func (d *DisplayState) displayScores(nRound int, final bool, puppet string) *dataScores {

	// serialisation
	if puppet == DisplayController {
		defer d.updatesDisplays()()
	} else {
		defer d.updatesNone()()
	}

	// round
	round, err := d.app.RoundStore.GetByN(nRound)
	if err != nil {
		return nil
	}
	tieBreak := nRound > d.app.quizState.nFullRounds

	// teams sorted by round score
	scoreStore := d.app.ScoreStore
	scoresRound := scoreStore.ForRoundByScore(nRound)

	// teams sorted by ranked scores
	nTopTeams := d.app.cfg.TopTeams
	if nTopTeams <= 0 {
		nTopTeams = d.app.NTeams
	}
	scoresRanked := scoreStore.ForRoundByRank(nRound, nTopTeams)

	// top teams in reverse order
	var scoresTop []*models.TeamScore
	if final {
		// limit round scores to top teams on final full round
		var nFinal int
		if tieBreak {
			nFinal = d.app.NTeams // tie break need't be for a top team
		} else {
			nFinal = d.app.quizState.quizCached.NFinalScores
		}

		scoresTop = scoreStore.ForRoundByReverseRank(nRound, nFinal)
	}

	// leaderboard slide index (for puppet scoreboard).
	if puppet == DisplayController {
		nScores := len(scoresRanked)

		if d.contest.CurrentRound < d.app.quizState.nFullRounds {
			d.contest.LeaderboardIndex = nScores + 2 // heading + scores + heading + leaderboard - 1
		} else {
			d.contest.LeaderboardIndex = nScores // heading + scores - 1 (to stay on final score, no leaderboard)
		}
	}

	// update state
	var update int
	if puppet == `Q` {
		update = d.quizmasterUpdate
	} else {
		update = d.app.quizState.publishedUpdate
	}

	return &dataScores{
		Title:         round.Title,
		ScoredTo:      d.scoredTo(puppet),
		ReadyTo:       d.readyTo(),
		ScoresTop:     scoresTop,
		ScoresByRound: scoresRound,
		ScoresByRank:  scoresRanked,
		TieBreak:      tieBreak,
		dataDisplay: dataDisplay{

			Index:      d.slideIndex(puppet),
			Update:     update,
			Sync:       d.syncUpdate,
			Tick:       d.contest.Tick,
			BreakSlide: d.app.displayState.roundFormat.additional,
			Organiser:  d.app.quizState.quizCached.Organiser,
			TouchNav:   d.navigation(puppet),
			DoNow:      d.doNow,
			DoNext:     d.doNext,
		},
	}
}

// Static page

func (d *DisplayState) displayStatic(puppet string) *dataStatic {

	// serialisation
	defer d.updatesNone()()

	return &dataStatic{
		Title: d.app.quizState.quizCached.Title,
		dataDisplay: dataDisplay{
			BreakSlide: "P",
			Organiser:  d.app.quizState.quizCached.Organiser,
			Sync:       d.syncUpdate,
			Tick:       d.contest.Tick,
			TouchNav:   d.navigation(puppet),
		},
	}
}

// Get teams ordered by name, for start of quiz

func (d *DisplayState) displayTeams(puppet string) *dataTeams {

	// serialisation
	defer d.updatesNone()()

	teams := d.app.TeamStore.ByName()

	// update state
	var update int
	if puppet == `Q` {
		update = d.quizmasterUpdate
	} else {
		update = d.app.quizState.publishedUpdate
	}
	
	return &dataTeams{
		Teams:   teams,
		ReadyTo: d.readyTo(),
		dataDisplay: dataDisplay{
			Update:    update,
			Sync:      d.syncUpdate,
			Tick:      d.contest.Tick,
			Organiser: d.app.quizState.quizCached.Organiser,
			TouchNav:  d.navigation(puppet),
			DoNow:     d.doNow,
			DoNext:    d.doNext,
		},
	}
}

// Static page, waiting for scores

func (d *DisplayState) displayWait(nRound int, puppet string) *dataWait {

	// serialisation
	defer d.updatesNone()()

	round, err := d.app.RoundStore.GetByN(nRound)
	if err != nil {
		return &dataWait{
			Error: "This round is undefined (" + err.Error() + ").",
		}
	}

	return &dataWait{
		Title: round.Title,
		dataDisplay: dataDisplay{
			Update:     d.app.quizState.publishedUpdate,
			Sync:       d.syncUpdate,
			Tick:       d.contest.Tick,
			BreakSlide: d.app.displayState.roundFormat.additional,
			Organiser:  d.app.quizState.quizCached.Organiser,
			TouchNav:   d.navigation(puppet),
		},
	}
}

// roundTitle returns the title for a round.
func (d *DisplayState) roundTitle(nRound int) string {

	// serialisation
	defer d.updatesNone()()

	r, _ := d.app.RoundStore.GetByN(nRound)
	if r == nil {
		return ""
	}

	return r.Title
}

// decodeFormat returns round format, decoded.
func decodeFormat(format string, slideItems int) roundFormat {

	// default format
	rf := roundFormat{
		slideQuestions: slideItems,
		slideAnswers:   slideItems,
	}

	// round format
	flags := strings.Split(format, "|")

	for _, f := range flags {
		if len(f) > 0 {
			c := f[0]
			switch c {

			case 'E':
				rf.additional = f // additional slide

			case 'I':
				rf.interval = true

			case 'Q':
				rf.slideQuestions = decodeMax(f, slideItems)

			case 'A':
				rf.slideAnswers = decodeMax(f, slideItems)

			case 'C':
				rf.slideQuestions = decodeMax(f, slideItems/2)
				rf.combined = true
			}
		}
	}
	return rf
}

func decodeMax(flag string, slideItems int) int {

	max, err := strconv.Atoi(flag[1:])
	if err != nil || max <= 1 {
		max = slideItems // default
	}
	return max
}

// mediaType returns the type as a string (for template use) and true if a media needs its own slide.
func (d *DisplayState) mediaType(file string) (mediaType string, singleton bool) {

	if file == "" {
		return "T", false // text
	}
	switch d.app.uploader.MediaType(file) {
	case uploader.MediaAudio:
		return "A", false

	case uploader.MediaImage:
		return "P", true

	case uploader.MediaVideo:
		return "V", true

	default:
		return "T", false // unknown - ignore
	}
}

// navigation returns a CSS class name if buttons for a touch screen should be shown.
func (d *DisplayState) navigation(puppet string) string {

	switch puppet {
	case DisplayController:
		if d.contest.TouchController {
			return "controller"
		} else {
			return "" // turn off if not needed
		}

	case DisplayQuizmaster:
		return "quizmaster" // always shown

	default:
		return ""
	}
}

// Quizmaster's indicator of scored round

func (d *DisplayState) readyTo() string {

	readyTo := d.app.quizState.quizCached.ScoringRound - 1

	if readyTo > d.contest.QuizmasterRound {
		return "[R" + strconv.Itoa(readyTo) + " ready]"
	} else {
		return ""
	}
}

// Latest scored round, for leaderboard

func (d *DisplayState) scoredTo(display string) int {

	if display == DisplayQuizmaster {
		return d.contest.QuizmasterRound
	} else {
		return d.contest.ScoreboardRound
	}
}

// Index for current slide

func (d *DisplayState) slideIndex(puppet string) int {

	s := d.contest

	switch puppet {

	case DisplayController, DisplayPractice, DisplayReplica:
		return s.CurrentIndex

	case DisplayQuizmaster:
		// always reload on first slide
		return 0

	case DisplayScoreboard:
		// synchronised to controller scores page, if showing, otherwise restart
		switch s.CurrentPage {
		case models.PageFinal, models.PageScores:
			return s.CurrentIndex
		default:
			return s.LeaderboardIndex
		}

	case DisplayScorer:
		// index not used
		return 0

	default:
		// team response: access token
		return s.CurrentIndex
	}
}

// splitRound returns questions (or answers) for a round split into slides.
// Returns true if round split
func (d *DisplayState) splitRound(roundId int64, nPerSlide int) ([]*Slide, bool) {

	// ### And why not use the current array as source, if we have it?
	// #### Fetch models.Round or keep it somewhere else? #####

	var slides []*Slide

	nSlides := 0

	// add as many items as will fit to each slide
	s := &Slide{NSlide: nSlides + 1}
	nItem := 0

	for _, q := range d.app.QuestionStore.ForRound(roundId) {

		mt, singleton := d.mediaType(q.File)

		// question media needs its own slide?
		if singleton {
			if nItem > 0 {
				slides = append(slides, s)
				nSlides++

				// start new slide
				s = &Slide{NSlide: nSlides + 1}
			}
			nItem = nPerSlide // force new slide afterwards
		}

		// add question to slide
		qd := &Question{
			Question: q.QuestionBr(),
			Answer:   q.AnswerBr(),
			File:     q.File,
			Type:     mt,
		}
		s.Questions = append(s.Questions, qd)
		nItem++

		if nItem >= nPerSlide {
			slides = append(slides, s)
			nSlides++

			// start new slide
			s = &Slide{NSlide: nSlides + 1}
			nItem = 0
		}
	}

	// append final slide
	if nItem > 0 {
		slides = append(slides, s)
		nSlides++
	}

	return slides, nSlides > 1
}
