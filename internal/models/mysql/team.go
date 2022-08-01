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

// SQL operations on teams.
//
// Includes joins that start with team, even where more information is returned about the score.

package mysql

import (
	"log"

	"github.com/jmoiron/sqlx"

	"inchworks.com/quiz/internal/models"
)

const (
	teamDelete = `DELETE FROM team WHERE id = ?`

	teamInsert = `
		INSERT INTO team (quiz, name, access, rank, total) VALUES (:quiz, :name, :access, :rank, :total)`

	teamUpdate = `
		UPDATE team
		SET name=:name, access=:access, rank=:rank, total=:total
		WHERE id=:id
	`
)

const (
	teamSelect    = `SELECT * FROM team`
	teamOrderName = ` ORDER BY name`

	teamWhereId    = teamSelect + ` WHERE id = ?`
	teamsWhereQuiz = teamSelect + ` WHERE quiz = ?`

	teamsByName = teamsWhereQuiz + teamOrderName

	teamCount = `SELECT COUNT(*) FROM team WHERE quiz = ?`

	teamCountResponded = `
		SELECT COUNT(*) FROM team WHERE EXISTS(
			SELECT response.id FROM response
				INNER JOIN question ON question.id = response.question
				WHERE question.round = ? AND response.team = team.id
		)
	`

	teamsWithResponded = `
		SELECT
			team.*,
			CASE WHEN EXISTS(
				SELECT response.id FROM response
					INNER JOIN question ON question.id = response.question
					WHERE question.round = ? AND response.team = team.id
				) THEN 1 ELSE 0 END AS responded
			FROM team
			WHERE team.quiz = ?
			ORDER BY team.name ASC
	`
	teamsWithResponses = `
		SELECT team.*, SUM(response.score) as value FROM team
			INNER JOIN response ON response.team = team.id
			INNER JOIN question ON question.id = response.question
			WHERE question.round = ? AND team.quiz = ?
			GROUP BY team.id
	`
	teamsWithScores = `
		SELECT team.id AS teamid, team.name, score.id AS scoreid, score.score as value FROM team
			LEFT JOIN score ON score.team = team.id AND score.round = ?
			WHERE team.quiz = ?
			ORDER BY team.name ASC
	`
	teamsWithTotals = `
		SELECT team.*, SUM(score.score) as value FROM team
			INNER JOIN score ON score.team = team.id
			WHERE score.round <= ? AND team.quiz = ?
			GROUP BY team.id
			ORDER BY value DESC
	`
)

type TeamStore struct {
	QuizId int64
	store
}

func NewTeamStore(db *sqlx.DB, tx **sqlx.Tx, log *log.Logger) *TeamStore {

	return &TeamStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  log,
			sqlDelete: teamDelete,
			sqlInsert: teamInsert,
			sqlUpdate: teamUpdate,
		},
	}
}

// All teams, unordered

func (st *TeamStore) All() []*models.Team {

	var teams []*models.Team

	if err := st.DBX.Select(&teams, teamsWhereQuiz, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return teams
}

// AllWithResponses returns all teams, with total response scores for a round.
func (st *TeamStore) AllWithResponses(nRound int) []*models.TeamScore {

	var ts []*models.TeamScore
	var err error

	// may be called after updating responses in a transaction
	if *st.ptx != nil {
		err = (*st.ptx).Select(&ts, teamsWithResponses, nRound, st.QuizId)
	} else {
		err = st.DBX.Select(&ts, teamsWithResponses, nRound, st.QuizId)
	}

	if err != nil {
		st.logError(err)
		return nil
	}
	return ts
}

// AllWithResponded returns all teams, with a response status for a round.
func (st *TeamStore) AllWithStatus(roundId int64) []*models.TeamResponded {

	var ts []*models.TeamResponded

	// may be called after updating responses in a transaction
	err := st.DBX.Select(&ts, teamsWithResponded, roundId, st.QuizId)

	if err != nil {
		st.logError(err)
		return nil
	}
	return ts
}

// All team names, with scores, including missing ones, ordered by name
// For scorers. Need score ID, for update.
// Unlike most queries, this returns some nullable values

func (st *TeamStore) AllWithScores(nRound int) []*models.TeamNameScore {

	var scoresTeam []*models.TeamNameScore

	if err := st.DBX.Select(&scoresTeam, teamsWithScores, nRound, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return scoresTeam
}

// All teams, with unpublished total score, in descending score order
// (For publishing. Need whole team record, for update.)

func (st *TeamStore) AllWithTotals(nRound int) []*models.TeamScore {

	var teamTotals []*models.TeamScore
	var err error

	// may be called after updating scores in a transaction
	if *st.ptx != nil {
		err = (*st.ptx).Select(&teamTotals, teamsWithTotals, nRound, st.QuizId)
	} else {
		err = st.DBX.Select(&teamTotals, teamsWithTotals, nRound, st.QuizId)
	}

	if err != nil {
		st.logError(err)
		return nil
	}
	return teamTotals
}

// All teams, in name order

func (st *TeamStore) ByName() []*models.Team {

	var teams []*models.Team

	if err := st.DBX.Select(&teams, teamsByName, st.QuizId); err != nil {
		st.logError(err)
		return nil
	}
	return teams
}

// Count returns the number of teams
func (st *TeamStore) Count() int {

	var n int

	if err := st.DBX.Get(&n, teamCount, st.QuizId); err != nil {
		st.logError(err)
		return 0
	}

	return n
}

// CountResponded returns that have responded for a round.
func (st *TeamStore) CountResponded(roundId int64) int {

	var n int

	if err := st.DBX.Get(&n, teamCountResponded, roundId); err != nil {
		st.logError(err)
		return 0
	}

	return n
}

// Get team

func (st *TeamStore) Get(id int64) (*models.Team, error) {

	var t models.Team

	if err := st.DBX.Get(&t, teamWhereId, id); err != nil {
		return nil, st.logError(err)
	}

	return &t, nil
}

// Insert or update team

func (st *TeamStore) Update(t *models.Team) error {

	t.Quiz = st.QuizId

	return st.updateData(&t.Id, t)
}
