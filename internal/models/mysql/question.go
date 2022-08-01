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

// SQL operations of questions.
// Includes joins that start with question.

import (
	"log"

	"github.com/jmoiron/sqlx"

	"inchworks.com/quiz/internal/models"
)

const (
	questionDelete = `DELETE FROM question WHERE id = ?`

	questionInsert = `
		INSERT INTO question (round, quiz_order, question, answer, file)
		VALUES (:round, :quiz_order, :question, :answer, :file)`

	questionUpdate = `
		UPDATE question
		SET quiz_order=:quiz_order, question=:question, answer=:answer, file=:file
		WHERE id = :id
	`
)

const (
	questionSelect = `SELECT * FROM question`

	questionOrder = ` ORDER BY quiz_order`

	questionWhereId     = questionSelect + ` WHERE id = ?`
	questionsWhereRound = questionSelect + ` WHERE round = ?` + questionOrder

	questionsWithResponse = `
		SELECT question.id AS questionid, question.question, response.id AS responseid, response.value FROM question
		LEFT JOIN response ON response.question = question.id AND response.team = ?
		WHERE question.round = ?
		ORDER BY question.quiz_order ASC
	`
)

type QuestionStore struct {
	store
}

func NewQuestionStore(db *sqlx.DB, tx **sqlx.Tx, log *log.Logger) *QuestionStore {

	return &QuestionStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  log,
			sqlDelete: questionDelete,
			sqlInsert: questionInsert,
			sqlUpdate: questionUpdate,
		},
	}
}

// ForRound returns all questions for a round, in order.
func (st *QuestionStore) ForRound(roundId int64) []*models.Question {

	var questions []*models.Question

	if err := st.DBX.Select(&questions, questionsWhereRound, roundId); err != nil {
		st.logError(err)
		return nil
	}
	return questions
}

// ResponsesForRound returns the team responses for a round
func (st *QuestionStore) ForTeamRound(teamId int64, roundId int64) []*models.QuestionResponse {

	var responses []*models.QuestionResponse

	if err := st.DBX.Select(&responses, questionsWithResponse, teamId, roundId); err != nil {
		st.logError(err)
		return nil
	}
	return responses
}

// Get returns a single question
func (st *QuestionStore) Get(id int64) (*models.Question, error) {

	var q models.Question

	if err := st.DBX.Get(&q, questionWhereId, id); err != nil {
		return nil, st.logError(err)
	}

	return &q, nil
}

// Update inserts or or updates a question.
func (st *QuestionStore) Update(q *models.Question) error {

	return st.updateData(&q.Id, q)
}
