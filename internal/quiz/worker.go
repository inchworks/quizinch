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

// Worker goroutine for all background processing

import (
	"runtime"

	"github.com/inchworks/webparts/etx"
	"github.com/inchworks/webparts/uploader"
)

// Implement RM interface for webparts.etx.

// Operation types
const (
	OpRound = iota
)

// We need an arbitary status code for rollback(). This one is ideal!
const statusTeapot = 418

func (s *QuizState) Name() string {
	return "quiz"
}

func (s *QuizState) ForOperation(opType int) etx.Op {
	switch opType {
	case OpRound:
		return &OpUpdateRound{}
	default:
		var unknown struct{}
		return &unknown
	}
}

// Do operation requested via TM.
func (s *QuizState) Operation(id etx.TxId, opType int, op etx.Op) {

	// send the request to the worker
	switch req := op.(type) {
	case *OpUpdateRound:
		req.tx = id
		s.app.chRound <- *req

	default:
		s.app.errorLog.Print("Unknown TX operation")
	}
}

// worker does all background processing for QuizInch.
func (s *QuizState) worker(
	chRound <-chan OpUpdateRound,
	done <-chan bool) {

	for {
		// returns to client sooner?
		runtime.Gosched()

		select {
		case req := <-chRound:

			// a round has been updated or removed
			s.onUpdateRound(req.RoundId, req.tx, req.Revised)

		case <-done:
			// ## do something to finish other pending requests
			return
		}
	}
}

// onUpdateRound processes an updated or deleted round.
func (s *QuizState) onUpdateRound(roundId int64, tx etx.TxId, revised bool) int {

	// setup
	bind := s.app.uploader.StartBind(roundId, tx)

	// set versioned media, and update round
	if st := s.updateMedia(roundId, revised, bind); st != 0 {
		return st
	}

	// remove unused versions
	if err := bind.End(); err != nil {
		s.app.log(err)
		return statusTeapot
	}

	// terminate the extended transaction
	defer s.updatesQuiz()()
	if err := s.app.tm.End(tx); err != nil {
		return s.rollback(statusTeapot, err)
	} else {
		return 0
	}
}

// updateMedia changes media versions for a round. It also sets the slideshow revision time.
func (s *QuizState) updateMedia(roundId int64, revised bool, bind *uploader.Bind) int {

	// serialise display state while slides are changing
	defer s.updatesQuiz()()

	// check if this is an update or deletion
	round := s.app.RoundStore.GetIf(roundId)
	if round == nil {
		// No questions to be updated. A following call to imager.RemoveVersions will delete all media.
		return 0
	}

	// check each question for an updated media file
	qs := s.app.QuestionStore.ForRound(roundId)
	for _, q := range qs {

		if q.File != "" {

			var newFile string
			var err error
			if newFile, err = bind.File(q.File); err != nil {
				// ## We have lost the file, but have no way to warn the user :-(
				// We must remove the reference so that all viewers don't get a missing file error.
				// log the error, but process the remaining slides
				q.File = ""
				s.app.QuestionStore.Update(q)
				s.app.errorLog.Print(err.Error())

			} else if newFile != "" {
				// updated media
				q.File = newFile
				s.app.QuestionStore.Update(q)
			}
		}
	}

	s.app.RoundStore.Update(round)

	return 0
}
