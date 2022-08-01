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
	contestInsert = `
		INSERT INTO contest
		(quiz, current_index, current_page, current_round, current_static,
			leaderboard_index, quizmaster_round, scoreboard_round, tick, live, touch_controller)
		VALUES (:quiz, :current_index, :current_page, :current_round, :current_static,
			:leaderboard_index, :quizmaster_round, :scoreboard_round, :tick, :live, :touch_controller)
	`
	contestUpdate = `
		UPDATE contest
		SET current_index=:current_index, current_page=:current_page, current_round=:current_round,
			current_static=:current_static, leaderboard_index=:leaderboard_index,
			quizmaster_round=:quizmaster_round, scoreboard_round=:scoreboard_round,
			tick=:tick, live=:live, touch_controller=:touch_controller
		WHERE id=:id
	`
)

const (
	contestSelect = `SELECT * FROM contest`

	contestWhereId = contestSelect + ` WHERE quiz = ?`
)

type ContestStore struct {
	QuizId int64
	store
}

func NewContestStore(db *sqlx.DB, tx **sqlx.Tx, log *log.Logger) *ContestStore {

	return &ContestStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  log,
			sqlInsert: contestInsert,
			sqlUpdate: contestUpdate,
		},
	}
}

// Get contest

func (st *ContestStore) Get() (*models.Contest, error) {

	var s models.Contest

	if err := st.DBX.Get(&s, contestWhereId, st.QuizId); err != nil {
		return nil, st.logError(err)
	}

	return &s, nil
}

// Insert or update contest

func (st *ContestStore) Update(s *models.Contest) error {

	s.Quiz = st.QuizId
	return st.updateData(&s.Id, s)
}
