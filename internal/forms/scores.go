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

	"github.com/inchworks/webparts/multiforms"
)

type Scores struct {
	*multiforms.Form
	Children []*Score
}

type Score struct {
	multiforms.Child
	Name  string
	Score string
}

// NewScores returns a form to edit scores

func NewScores(data url.Values, token string) *Scores {
	return &Scores{
		Form:     multiforms.New(data, token),
		Children: make([]*Score, 0, 16),
	}
}

// Add appends a score sub-form

func (f *Scores) Add(index int, name string, score string) {

	f.Children = append(f.Children, &Score{
		Child: multiforms.Child{Parent: f.Form, ChildIndex: index},
		Name:  name,
		Score: score,
	})
}

// GetScores returns the user data as an array of structs.
// They are sent in the HTML form as arrays of values for each field name.

func (f *Scores) GetScores() (items []*Score, err error) {

	nItems := f.NChildItems()

	for i := 0; i < nItems; i++ {

		ix, err := f.ChildIndex("index", i)
		if err != nil {
			return nil, err
		}

		items = append(items, &Score{
			Child: multiforms.Child{Parent: f.Form, ChildIndex: ix},
			Score: f.ChildGet("score", i),
		})
	}

	// Add the child items back into the form, in case we need to redisplay it
	f.Children = items

	return items, nil
}
