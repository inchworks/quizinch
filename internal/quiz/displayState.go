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

// Data and common functions related to quiz displays

import (
	"sync"

	"inchworks.com/quiz/internal/models"
)

type DisplayState struct {
	app        *Application
	muDisplays sync.RWMutex

	// cached durable state
	contest      *models.Contest
	dirtyContest bool

	// volatile
	roundFormat      roundFormat // format of current round
	quizmasterUpdate int // update to quizmaster
	syncUpdate       int // update to controller synchronisation

	// prompts for quizmaster
	doNow  string // current slide
	doNext string // next slide
	isTied bool   // tie on final round or tie-break
}

// Initialisation

func (d *DisplayState) Init(s *models.Contest) {
	d.contest = s
	d.setRound(s.CurrentRound)
	d.setPrompts()
}

// Cancel unneeded update

func (d *DisplayState) cancelUpdate() {

	d.dirtyContest = false
}

// changedQuizmaster notes that the quizmaster must re-synchronise.
func (d *DisplayState) changedQuizmaster() {

	d.quizmasterUpdate = timestamp()
}

// changedSync notes that the controller must re-synchronise.
func (d *DisplayState) changedSync() {

	d.syncUpdate = timestamp()
}

// Get serialised contest, for update by quizState
//
// This is used in the uncommon case that the quiz needs to update the contest,
// typically to rank totals and update the quizmaster, when scores are updated.

func (d *DisplayState) forUpdate() (*models.Contest, func()) {

	// quiz already serialised, and transaction started

	// lock contest
	d.muDisplays.Lock()
	d.dirtyContest = true

	return d.contest, func() {
		// save contest changes
		if d.dirtyContest {
			if err := d.app.ContestStore.Update(d.contest); err != nil {
				d.app.log(err)
			}
			d.dirtyContest = false
		}

		// release lock (transaction ended by quizState)
		d.muDisplays.Unlock()
	}
}

// Take mutexes and transaction for update to quiz and displays

func (d *DisplayState) updatesAll() func() {

	// acquire locks (transaction started by quizState)
	qUnlock := d.app.quizState.updatesQuiz()
	d.muDisplays.Lock()

	d.dirtyContest = true

	return func() {

		// save contest changes
		if d.dirtyContest {
			if err := d.app.ContestStore.Update(d.contest); err != nil {
				d.app.log(err)
			}
			d.dirtyContest = false
		}

		// release locks (transaction ended by quiz state)
		d.muDisplays.Unlock()
		qUnlock()
	}
}

// Take mutexes and transaction for update to displays by controller (controller display)

func (d *DisplayState) updatesDisplays() func() {

	// acquire locks
	qUnlock := d.app.quizState.updatesNone()
	d.muDisplays.Lock()

	d.dirtyContest = true

	// start transaction
	d.app.tx = d.app.db.MustBegin()

	return func() {

		// save contest changes
		var err error
		if d.dirtyContest {
			if err = d.app.ContestStore.Update(d.contest); err != nil {
				d.app.log(err)
			}
			d.dirtyContest = false
		}

		// end transaction
		if err != nil {
			d.app.tx.Rollback()
		} else {
			d.app.tx.Commit()
		}

		d.app.tx = nil

		// release locks
		d.muDisplays.Unlock()
		qUnlock()
	}
}

// Take mutexes for non-updating request

func (d *DisplayState) updatesNone() func() {

	// aquire shared locks
	qUnlock := d.app.quizState.updatesNone()
	d.muDisplays.RLock()

	return func() {

		// release locks
		d.muDisplays.RUnlock()
		qUnlock()
	}
}
