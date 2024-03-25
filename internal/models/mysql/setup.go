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

// Setup application database

import (
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/inchworks/webparts/users"
	"inchworks.com/quiz/internal/models"
)

var cmds = [...]string{

	"SET NAMES utf8;",

	"SET time_zone = '+00:00';",

	"SET foreign_key_checks = 0;",

	"SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';",

	`CREATE TABLE question (
		id int(11) NOT NULL AUTO_INCREMENT,
		round int(11) NOT NULL,
		quiz_order int(11) NOT NULL,
		question varchar(512) COLLATE utf8_unicode_ci NOT NULL,
		answer varchar(512) COLLATE utf8_unicode_ci NOT NULL,
		file varchar(256) COLLATE utf8_unicode_ci NOT NULL,
		PRIMARY KEY (id),
		KEY IDX_ROUND (round),
		CONSTRAINT FK_ROUND FOREIGN KEY (round) REFERENCES round (id) ON DELETE CASCADE
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE quiz (
		id int(11) NOT NULL AUTO_INCREMENT,
		title varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		organiser varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		n_tie_breakers int(11) NOT NULL,
		n_deferred int(11) NOT NULL,
		refresh int(11) NOT NULL,
		access varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		n_final_scores int(11) NOT NULL,
		n_winners int(11) NOT NULL,
		response_round int(11) NOT NULL,
		scoring_round int(11) NOT NULL,
		PRIMARY KEY (id)
	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE redo (
		id BIGINT NOT NULL,
		manager varchar(32) COLLATE utf8_unicode_ci NOT NULL,
		optype int(11) NOT NULL,
		operation JSON NOT NULL,
		PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE response (
		id int(11) NOT NULL AUTO_INCREMENT,
		question int(11) NOT NULL,
		team int(11) NOT NULL,
		value varchar(128) COLLATE utf8_unicode_ci NOT NULL,
		score double DEFAULT NULL,
		confirm double DEFAULT NULL,
		PRIMARY KEY (id),
		KEY IDX_ROUND (question),
		KEY IDX_TEAM (team),
		CONSTRAINT RESPONSE_QUESTION FOREIGN KEY (question) REFERENCES question (id) ON DELETE CASCADE,
		CONSTRAINT RESPONSE_TEAM FOREIGN KEY (team) REFERENCES team (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE round (
		id int(11) NOT NULL AUTO_INCREMENT,
		quiz int(11) NOT NULL,
		quiz_order int(11) NOT NULL,
		title varchar(128) COLLATE utf8_unicode_ci NOT NULL,
		format varchar(16) COLLATE utf8_unicode_ci NOT NULL,
		PRIMARY KEY (id),
		KEY IDX_ROUND_QUIZ (quiz),
		CONSTRAINT FK_ROUND_QUIZ FOREIGN KEY (quiz) REFERENCES quiz (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE score (
		id int(11) NOT NULL AUTO_INCREMENT,
		team int(11) NOT NULL,
		round int(11) NOT NULL,
		responses int(11) NOT NULL,
		score double DEFAULT NULL,
		confirm double DEFAULT NULL,
		PRIMARY KEY (id),
		KEY IDX_TEAM (team),
		KEY IDX_ROUND (round),
		CONSTRAINT SCORE_TEAM FOREIGN KEY (team) REFERENCES team (id) ON DELETE CASCADE,
		CONSTRAINT SCORE_ROUND FOREIGN KEY (round) REFERENCES round (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE contest (
		id int(11) NOT NULL AUTO_INCREMENT,
		quiz int(11) DEFAULT NULL,
		current_index int(11) NOT NULL,
		current_page int(11) NOT NULL,
		current_round int(11) NOT NULL,
		current_static int(11) NOT NULL,
		quizmaster_round int(11) NOT NULL,
		scoreboard_round int(11) NOT NULL,
		leaderboard_index int(11) NOT NULL,
		touch_controller tinyint(1) NOT NULL,
		tick varchar(10) COLLATE utf8_unicode_ci NOT NULL,
		live tinyint(1) NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY IDX_SESSION_QUIZ (quiz),
		CONSTRAINT FK_SESSION_QUIZ FOREIGN KEY (quiz) REFERENCES quiz (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE statistic (
		id int(11) NOT NULL AUTO_INCREMENT,
		event varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		category varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		count int(11) NOT NULL,
		start datetime NOT NULL,
		detail smallint(6) NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY IDX_STATISTIC (event, start, detail),
		KEY IDX_START_PERIOD (start, detail)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE team (
		id int(11) NOT NULL AUTO_INCREMENT,
		quiz int(11) NOT NULL,
		name varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		access varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		rank int(11) NOT NULL,
		total double NOT NULL,
		PRIMARY KEY (id),
		KEY IDX_TEAM_QUIZ (quiz),
		CONSTRAINT FK_TEAM_QUIZ FOREIGN KEY (quiz) REFERENCES quiz (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,

	`CREATE TABLE user (
		id int(11) NOT NULL AUTO_INCREMENT,
		parent int(11) NOT NULL,
		username varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		name varchar(60) COLLATE utf8_unicode_ci NOT NULL,
		role smallint(6) NOT NULL,
		status smallint(6) NOT NULL,
		password char(60) COLLATE utf8_unicode_ci NOT NULL,
		created datetime NOT NULL,
		PRIMARY KEY (id),
		UNIQUE KEY IDX_USERNAME (username),
		KEY IDX_USER_PARENT (parent)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;`,
}

// Setup new database, if it has no tables.
// Add quiz record and specified administrator if needed.
//
// Returns quiz record.

func Setup(stQuiz *QuizStore, stSess *ContestStore, stUser *UserStore, quizId int64, adminName string, adminPW string, refresh int) (*models.Quiz, *models.Contest, error) {

	// look for quiz record
	quiz, err := stQuiz.Get(quizId)
	if err != nil {
		if driverErr, ok := err.(*mysql.MySQLError); ok {
			if driverErr.Number == 1146 {
				// no quiz table - make the database
				err = setupTables(stQuiz.DBX, *stQuiz.ptx)
			}
		} else if stQuiz.convertError(err) == models.ErrNoRecord {
			// ok if no gallery record yet
			err = nil
		}
	}
	if err != nil {
		return nil, nil, stQuiz.logError(err)
	}

	var sess *models.Contest
	if quiz == nil {
		// create first quiz ..
		quiz = &models.Quiz{
			Title:         "The Quiz",
			Organiser:     "Inchworks",
			NTieBreakers:  0,
			NDeferred:     1,
			Refresh:       refresh,
			Access:        "",
			NFinalScores:  4,
			NWinners:      1,
			ResponseRound: 0,
			ScoringRound:  0,
		}
		if err = stQuiz.Update(quiz); err != nil {
			return nil, nil, err
		}

		// .. and contest
		stSess.QuizId = quiz.Id
		sess = &models.Contest{
			Quiz: quiz.Id,
		}
		if err = stSess.Update(sess); err != nil {
			return quiz, nil, err
		}
	} else {
		// contest for quiz
		stSess.QuizId = quiz.Id
		if sess, err = stSess.Get(); err != nil {
			return quiz, nil, err
		}
	}

	// look for admin user
	stUser.ParentId = quiz.Id
	admin, err := stUser.GetNamed(adminName)
	if err != nil && err != models.ErrNoRecord {
		return quiz, sess, err
	}

	if admin == nil && len(adminName) > 0 {

		// configured admin user doesn't exist - add one
		if err := setupAdmin(stUser, adminName, adminPW); err != nil {
			return quiz, sess, err
		}
	}
	return quiz, sess, nil
}

// MigrateQuiz1 upgrades the database for version 1.0.4.
func MigrateQuiz1(st *QuizStore, tx *sqlx.Tx) error {

	var cmdQuiz = `ALTER TABLE quiz ADD COLUMN n_winners int(11) NOT NULL;`

	// has winners column been added yet?
	if _, err := tx.Exec(cmdQuiz); err != nil {
		return nil
	}

	// set default winners
	q, err := st.Get(1)
	if err == nil {
		q.NWinners = 1
		err = st.Update(q)
	}
	return err
}


// create admin user

func setupAdmin(st *UserStore, adminName string, adminPW string) error {

	admin := &users.User{
		Username: adminName,
		Name:     "Administrator",
		Role:     models.UserAdmin,
		Status:   users.UserActive,
		Created:  time.Now(),
	}
	if err := admin.SetPassword(adminPW); err != nil {
		return err
	}

	if err := st.Update(admin); err != nil {
		return err
	}

	return nil
}

// create database tables

func setupTables(db *sqlx.DB, tx *sqlx.Tx) error {

	for _, cmd := range cmds {
		if _, err := tx.Exec(cmd); err != nil {
			return err
		}
	}
	return nil
}
