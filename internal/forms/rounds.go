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

type Rounds struct {
	*multiforms.Form
	Children []*Round
}

type Round struct {
	multiforms.Child
	models.Round
}

// NewRounds returns a form to edit rounds

func NewRounds(data url.Values, token string) *Rounds {
	return &Rounds{
		Form:     multiforms.New(data, token),
		Children: make([]*Round, 0, 16),
	}
}

// Add appends a round sub-form to the form

func (f *Rounds) Add(index int, r *models.Round) {

	f.Children = append(f.Children, &Round{
		Child: multiforms.Child{Parent: f.Form, ChildIndex: index},
		Round: *r,
	})
}

// AddTemplate appends the sub-form template to add a round.
func (f *Rounds) AddTemplate(nRounds int) {

	f.Children = append(f.Children, &Round{
		Child: multiforms.Child{Parent: f.Form, ChildIndex: -1},
		Round: models.Round{QuizOrder: nRounds + 1},
	})
}

// GetRounds returns the user data as an array of structs.
// They are sent in the HTML form as arrays of values for each field name.

func (f *Rounds) GetRounds() (items []*Round, err error) {

	nItems := f.NChildItems()

	for i := 0; i < nItems; i++ {

		ix, err := f.ChildIndex("index", i)
		if err != nil {
			return nil, err
		}

		items = append(items, &Round{
			Child: multiforms.Child{Parent: f.Form, ChildIndex: ix},
			Round: models.Round{
				QuizOrder: f.ChildMin("quizOrder", i, ix, 1),
				Format:    f.ChildText("format", i, ix, 0, 20),
				Title:     f.ChildText("title", i, ix, 1, 128),
			},
		})
	}

	// Add the child items back into the form, in case we need to redisplay it
	f.Children = items

	return items, nil
}
