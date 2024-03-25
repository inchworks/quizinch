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

package models

// Database models for QuizInch.

import (
	"database/sql"
	"errors"
	"html/template"
	"strings"
)

// Database field names are the same as structure names, with lower case first letter.

const (
	// page codes - saved in contest
	PageStatic = iota
	PageQuestions
	PageAnswers
	PageScoresWait // waiting on scorers
	PageScores
	PageFinal // final scores

	// page codes - for specific puppets
	PageStart           = 10 // quizmaster or scoreboard waiting for first scores
	PagePublicWait      = 11 // scoreboard waiting on public scores
	PageRespondWait     = 12 // team waiting to respond
	PageQuizResponses   = 13 // quizmaster's responses from teams
	PageScorerQuestions = 14 // scorer's question status for a round
	PageScorerRounds    = 15 // scorer's rounds status

	// user roles
	// These must match the database, so prefer specified values to iota.
	UserUnknown   = 0
	UserAudience  = 1 // not used
	UserTeam      = 2
	UserOrganiser = 3
	UserAdmin     = 4
)

const (
	// static page codes
	StaticStart    = 0
	StaticInterval = 1
	StaticEnd      = 999
)

var ErrNoRecord = errors.New("models: no matching record found")

type Question struct {
	Id        int64
	Round     int64
	QuizOrder int `db:"quiz_order"`
	Question  string
	Answer    string
	File      string
}

type Quiz struct {
	Id int64

	// quiz parameters
	Title        string
	Organiser    string
	NTieBreakers int `db:"n_tie_breakers"`
	NDeferred    int `db:"n_deferred"`
	NFinalScores int `db:"n_final_scores"`
	NWinners     int `db:"n_winners"`
	Refresh      int
	Access       string

	// state of scoring
	ResponseRound int `db:"response_round"`
	ScoringRound  int `db:"scoring_round"`
}

type Response struct {
	Id       int64
	Question int64
	Team     int64
	Value    string
	Score    float64
	Confirm  float64
}

type Round struct {
	Id        int64
	Quiz      int64
	QuizOrder int `db:"quiz_order"`
	Format    string
	Title     string
}

type Score struct {
	Id        int64
	Team      int64
	NRound    int `db:"round"`
	Responses int
	Value     float64 `db:"score"`
	Confirm   float64
}

type Contest struct {
	Id   int64
	Quiz int64

	// state of displays
	CurrentIndex     int `db:"current_index"`
	CurrentPage      int `db:"current_page"`
	CurrentRound     int `db:"current_round"`
	CurrentStatic    int `db:"current_static"`
	LeaderboardIndex int `db:"leaderboard_index"`
	QuizmasterRound  int `db:"quizmaster_round"`
	ScoreboardRound  int `db:"scoreboard_round"`
	Tick             string
	Live             bool
	TouchController  bool `db:"touch_controller"`
}

type Team struct {
	Id     int64
	Quiz   int64
	Name   string
	Access string
	Rank   int
	Total  float64
}

// Composite query results

type QuestionResponse struct {
	QuestionId int64
	Question   string
	ResponseId sql.NullInt64
	Value      sql.NullString
}

type ResponseTeam struct {
	Response
	Name string
}

type TeamNameScore struct {
	TeamId  int64
	Name    string
	ScoreId sql.NullInt64
	Value   sql.NullFloat64
}

type TeamResponded struct {
	Team
	Responded bool
}

type TeamScore struct {
	Team
	Value float64
}

// Fields with newlines replaced by breaks, and HTML formatting allowed.

func (q *Question) QuestionBr() template.HTML {
	return template.HTML(nl2br(q.Question))
}

func (q *Question) AnswerBr() template.HTML {
	return template.HTML(nl2br(q.Answer))
}

func nl2br(str string) string {
	return strings.Replace(str, "\n", "<br>", -1)
}
