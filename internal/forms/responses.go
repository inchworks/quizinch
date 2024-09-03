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
	"database/sql"
	"net/url"

	"github.com/inchworks/webparts/v2/multiforms"

	"inchworks.com/quiz/internal/models"
)

type Responses struct {
	*multiforms.Form
	Children []*Response
}

type Response struct {
	multiforms.Child
	models.QuestionResponse
}

// NewQuestions returns a form to edit responses
func NewResponses(data url.Values, token string) *Responses {
	return &Responses{
		Form:     multiforms.New(data, token),
		Children: make([]*Response, 0, 8),
	}
}

// Add appends a response sub-form
func (f *Responses) Add(index int, qr *models.QuestionResponse) {

	f.Children = append(f.Children, &Response{
		Child:            multiforms.Child{Parent: f.Form, ChildIndex: index},
		QuestionResponse: *qr,
	})
}

// GetResponses returns the user data as an array of structs.
// They are received in the HTML form as arrays of values for each field name.
func (f *Responses) GetResponses() (items []*Response, err error) {

	nItems := f.NChildItems()

	for i := 0; i < nItems; i++ {

		ix, err := f.ChildIndex("index", i)
		if err != nil {
			return nil, err
		}

		items = append(items, &Response{
			Child: multiforms.Child{Parent: f.Form, ChildIndex: ix},
			QuestionResponse: models.QuestionResponse{
				Value: sql.NullString{
					Valid:  true,
					String: f.ChildText("value", i, ix, 0, 128),
				},
			},
		})
	}

	// Add the child items back into the form, in case we need to redisplay it
	f.Children = items

	return items, nil
}
