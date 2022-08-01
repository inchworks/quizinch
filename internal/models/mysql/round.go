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

package mysql

import (
	"log"

	"github.com/jmoiron/sqlx"

	"inchworks.com/quiz/internal/models"
)

const (
	roundDelete = `DELETE FROM round WHERE id = ?`

	roundInsert = `
		INSERT INTO round (quiz, quiz_order, title, format) VALUES (:quiz, :quiz_order, :title, :format)`

	roundUpdate = `
		UPDATE round
		SET quiz_order=:quiz_order, title=:title, format=:format 
		WHERE id = :id
	`
)

const (
	roundSelect = `SELECT * FROM round`
	roundOrder  = ` ORDER BY quiz_order`

	roundWhereId     = roundSelect + ` WHERE id = ?`
	roundWhereNumber = roundSelect + ` WHERE quiz = ? AND quiz_order = ?`
	roundsWhereQuiz  = roundSelect + ` WHERE quiz = ?`

	roundsByNumber = roundsWhereQuiz + roundOrder

	roundCount = `SELECT COUNT(*) FROM round WHERE quiz = ?`

	roundsCurrent = roundSelect + ` WHERE quiz = ? AND quiz_order >= ?` + roundOrder + ` LIMIT ?`
)

type RoundStore struct {
	QuizId int64
	store
}

func NewRoundStore(db *sqlx.DB, tx **sqlx.Tx, log *log.Logger) *RoundStore {

	return &RoundStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  log,
			sqlDelete: roundDelete,
			sqlInsert: roundInsert,
			sqlUpdate: roundUpdate,
		},
	}
}

// All returns all rounds, in sequence order.
func (st *RoundStore) All() []*models.Round {

	var rounds []*models.Round

	if err := st.DBX.Select(&rounds, roundsByNumber, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return rounds
}

// Count returns the number of rounds.
func (st *RoundStore) Count() int {

	var n int

	if err := st.DBX.Get(&n, roundCount, st.QuizId); err != nil {
		st.logError(err)
		return 0
	}

	return n
}

// Current returns a limited number of rounds.
func (st *RoundStore) Current(nRound int, limit int) []*models.Round {

	var rounds []*models.Round

	if err := st.DBX.Select(&rounds, roundsCurrent, st.QuizId, nRound, limit); err != nil {
		st.logError(err)
		return nil
	}
	return rounds
}

// Get returns a round by ID.
func (st *RoundStore) Get(roundId int64) (*models.Round, error) {

	var r models.Round

	if err := st.DBX.Get(&r, roundWhereId, roundId); err != nil {
		return nil, st.logError(err)
	}

	return &r, nil
}

// GetIf returns a round by ID, if it exists
func (st *RoundStore) GetIf(id int64) *models.Round {

	var r models.Round

	if err := st.DBX.Get(&r, roundWhereId, id); err != nil {
		if st.convertError(err) != models.ErrNoRecord {
			st.logError(err)
		}
		return nil
	}

	return &r
}

// GetByN returns a round by number.
func (st *RoundStore) GetByN(nRound int) (*models.Round, error) {

	var r models.Round

	if err := st.DBX.Get(&r, roundWhereNumber, st.QuizId, nRound); err != nil {
		return nil, st.logError(err)
	}

	return &r, nil
}

// Insert or update round

func (st *RoundStore) Update(r *models.Round) error {
	r.Quiz = st.QuizId

	return st.updateData(&r.Id, r)
}
