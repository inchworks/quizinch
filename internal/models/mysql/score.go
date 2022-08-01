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

// SQL operations on scores.
//
// Includes joins that start with score, even where more information is returned about the team.

package mysql

import (
	"log"

	"github.com/jmoiron/sqlx"
	"inchworks.com/quiz/internal/models"
)

// ## Define an interface if we want to have a test DB implementation

const (
	scoreDelete    = `DELETE FROM score WHERE id = ?`
	scoreDeleteAll = `DELETE FROM score WHERE team = ANY(SELECT id FROM team WHERE quiz = ?)`

	scoreInsert = `
		INSERT INTO score (team, round, responses, score, confirm) VALUES (:team, :round, :responses, :score, :confirm)`

	scoreUpdate = `
		UPDATE score
		SET team=:team, round=:round, responses=:responses, score=:score, confirm:=confirm
		WHERE id=:id
	`
)

// ## scoresWithTeam don't specify a quiz!

const (
	scoreSelect    = `SELECT * FROM score`
	scoresWithTeam = `
		SELECT score.score AS value, team.* FROM score
		INNER JOIN team ON team.id = score.team
	`

	scoreOrderAscending = ` ORDER BY score ASC, team.name`
	scoreOrderName      = ` ORDER BY team.name`
	scoreOrderRankAsc   = ` ORDER BY team.rank ASC, team.name`
	scoreOrderRankDesc  = ` ORDER BY team.rank DESC, team.name`

	scoreWhereId           = scoreSelect + ` WHERE id = ?`
	scoreWhereTeamAndRound = scoreSelect + ` WHERE team = ? and round = ?`
	scoresWhereCompleted   = scoreSelect + ` WHERE team = ? and round <= ?`

	scoresWhereRound = ` WHERE round = ? AND team.quiz = ?
	`
	scoresWhereRoundAndRank = ` WHERE round = ? AND rank <= ? AND team.quiz = ?`

	scoresByAscendingRank  = scoresWithTeam + scoresWhereRound + scoreOrderRankAsc
	scoresByDescendingRank = scoresWithTeam + scoresWhereRoundAndRank + scoreOrderRankDesc
	scoresByOrder          = scoresWithTeam + scoresWhereRound + scoreOrderAscending
	scoresByName           = scoresWithTeam + scoresWhereRound + scoreOrderName
)

type ScoreStore struct {
	QuizId int64
	store
}

func NewScoreStore(db *sqlx.DB, tx **sqlx.Tx, log *log.Logger) *ScoreStore {

	return &ScoreStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  log,
			sqlDelete: scoreDelete,
			sqlInsert: scoreInsert,
			sqlUpdate: scoreUpdate,
		},
	}
}

// Completed round scores for team

func (st *ScoreStore) CompletedForTeam(teamId int64, nRound int) []*models.Score {

	var scores []*models.Score

	if err := st.DBX.Select(&scores, scoresWhereCompleted, teamId, nRound); err != nil {
		st.logError(err)
		return nil
	}
	return scores
}

// Delete all scores for quiz
func (st *ScoreStore) DeleteAll(quizId int64) error {

	tx := *st.ptx
	if _, err := tx.Exec(scoreDeleteAll, quizId); err != nil {
		return st.logError(err)
	}

	return nil
}

// Scores for round, in team name order

func (st *ScoreStore) ForRoundByTeam(nRound int) []*models.TeamScore {

	var scores []*models.TeamScore

	if err := st.DBX.Select(&scores, scoresByName, nRound, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return scores

}

// Scores for round, in rank order, with teams

func (st *ScoreStore) ForRoundByRank(nRound int) []*models.TeamScore {

	var scores []*models.TeamScore

	if err := st.DBX.Select(&scores, scoresByAscendingRank, nRound, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return scores

}

// Top ranked, in descending rank (reverse order), with teams
// Low rank and unscored teams are omitted.

func (st *ScoreStore) ForRoundByReverseRank(nRound int, nTop int) []*models.TeamScore {

	var scores []*models.TeamScore

	if err := st.DBX.Select(&scores, scoresByDescendingRank, nRound, nTop, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return scores
}

// All scores for round, in ascending score order, with teams

func (st *ScoreStore) ForRoundByScore(nRound int) []*models.TeamScore {

	var scores []*models.TeamScore

	if err := st.DBX.Select(&scores, scoresByOrder, nRound, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return scores

}

// Single score, may not exist

func (st *ScoreStore) ForTeamAndRound(teamId int64, nRound int) (*models.Score, error) {

	var s models.Score

	if err := st.DBX.Get(&s, scoreWhereTeamAndRound, teamId, nRound); err != nil {
		return nil, err
	}
	return &s, nil
}

// Insert or update score. Team ID must be set in struct.

func (st *ScoreStore) Update(s *models.Score) error {

	return st.updateData(&s.Id, s)
}
