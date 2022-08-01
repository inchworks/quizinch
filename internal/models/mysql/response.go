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

// SQL operations on responses by teams.

package mysql

import (
	"log"

	"github.com/jmoiron/sqlx"
	"inchworks.com/quiz/internal/models"
)

// ## Define an interface if we want to have a test DB implementation

const (
	responseDelete    = `DELETE FROM response WHERE id = ?`
	responseDeleteAll = `DELETE FROM response WHERE team = ANY(SELECT id FROM team WHERE quiz = ?)`

	responseInsert = `
		INSERT INTO response (question, team, value, score, confirm) VALUES (:question, :team, :value, :score, :confirm)`

	responseUpdate = `
		UPDATE response
		SET question=:question, team=:team, value=:value, score=:score, confirm=:confirm
		WHERE id=:id
	`
)

const (
	responseSelect    = `SELECT * FROM response`
	responsesWithTeam = `
		SELECT response.*, team.name FROM response
		INNER JOIN team ON team.id = response.team
	`

	responseWhereId        = responseSelect + ` WHERE id = ?`
	responsesWhereQuestion = responsesWithTeam + ` WHERE question = ? ORDER BY team.name`
)

type ResponseStore struct {
	QuizId int64
	store
}

func NewResponseStore(db *sqlx.DB, tx **sqlx.Tx, log *log.Logger) *ResponseStore {

	return &ResponseStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  log,
			sqlDelete: responseDelete,
			sqlInsert: responseInsert,
			sqlUpdate: responseUpdate,
		},
	}
}

// DeleteAll removed all responses for quiz
func (st *ResponseStore) DeleteAll(quizId int64) error {

	tx := *st.ptx
	if _, err := tx.Exec(responseDeleteAll, quizId); err != nil {
		return st.logError(err)
	}

	return nil
}

// ResponsesForQuestion returns all team responses for a single question
func (st *ResponseStore) ResponsesForQuestion(questionId int64) []*models.ResponseTeam {

	var responses []*models.ResponseTeam

	if err := st.DBX.Select(&responses, responsesWhereQuestion, questionId); err != nil {
		st.logError(err)
		return nil
	}
	return responses
}

// Update inserts or updates a response. The Question and Team IDs must be set in struct.
func (st *ResponseStore) Update(s *models.Response) error {

	return st.updateData(&s.Id, s)
}
