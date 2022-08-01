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
	quizInsert = `
		INSERT INTO quiz
		(title, organiser, n_tie_breakers, n_deferred, refresh, access, n_final_scores, response_round, scoring_round)
		VALUES (:title, :organiser, :n_tie_breakers, :n_deferred, :refresh, :access, :n_final_scores, :response_round, :scoring_round)
	`
	quizUpdate = `
		UPDATE quiz
		SET title=:title, organiser=:organiser, n_tie_breakers=:n_tie_breakers,
			n_deferred=:n_deferred, refresh=:refresh, access=:access,
			n_final_scores=:n_final_scores, response_round=:response_round, scoring_round=:scoring_round
		WHERE id=:id
	`
)

type QuizStore struct {
	store
}

func NewQuizStore(db *sqlx.DB, tx **sqlx.Tx, log *log.Logger) *QuizStore {

	return &QuizStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  log,
			sqlInsert: quizInsert,
			sqlUpdate: quizUpdate,
		},
	}
}

// Get returns the quiz for specified ID.
// Unlike most store functions, it does not log an error.
func (st *QuizStore) Get(id int64) (*models.Quiz, error) {

	q := &models.Quiz{}

	if err := st.DBX.Get(q, "SELECT * FROM quiz WHERE id = ?", id); err != nil {
		return nil, err
	}
	return q, nil
}

// Insert or update quiz

func (st *QuizStore) Update(q *models.Quiz) error {

	return st.updateData(&q.Id, q)
}
