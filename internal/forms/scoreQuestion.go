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

package forms

import (
	"net/url"

	"github.com/inchworks/webparts/v2/multiforms"
	"inchworks.com/quiz/internal/models"
)

type ScoreQuestion struct {
	*multiforms.Form
	Children []*ScoreResponse
}

type ScoreResponse struct {
	multiforms.Child
	*models.ResponseTeam
	Score string
}

// NewScoreQuestion returns a form to edit scores for a single question
func NewScoreQuestion(data url.Values, token string) *ScoreQuestion {
	return &ScoreQuestion{
		Form:     multiforms.New(data, token),
		Children: make([]*ScoreResponse, 0, 16),
	}
}

// Add appends a response sub-form
func (f *ScoreQuestion) Add(index int, r *models.ResponseTeam, score string) {

	f.Children = append(f.Children, &ScoreResponse{
		Child:        multiforms.Child{Parent: f.Form, ChildIndex: index},
		ResponseTeam: r,
		Score:        score,
	})
}

// GetScores returns the scores as an array of structs.
// They are sent in the HTML form as arrays of values for each field name.
func (f *ScoreQuestion) GetScores() (items []*ScoreResponse, err error) {

	nItems := f.NChildItems()

	for i := 0; i < nItems; i++ {

		ix, err := f.ChildIndex("index", i)
		if err != nil {
			return nil, err
		}

		_, s := f.ChildFloat("score", i, ix, 0, 99, 1)
		items = append(items, &ScoreResponse{
			Child: multiforms.Child{Parent: f.Form, ChildIndex: ix},
			Score: s,
		})
	}

	// Add the child items back into the form, in case we need to redisplay it
	f.Children = items

	return items, nil
}
