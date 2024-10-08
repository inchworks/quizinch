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
	"github.com/inchworks/webparts/v2/uploader"

	"inchworks.com/quiz/internal/models"
)

type Questions struct {
	*multiforms.Form
	Children []*Question
}

type Question struct {
	multiforms.Child
	QuizOrder int
	Question  string
	Answer    string
	MediaName string
	Version   int
}

type ValidTypeFunc func(string) bool

// NewQuestions returns a form to edit questions

func NewQuestions(data url.Values, token string) *Questions {
	return &Questions{
		Form:     multiforms.New(data, token),
		Children: make([]*Question, 0, 16),
	}
}

// Add appends a question sub-form.
func (f *Questions) Add(index int, q *models.Question) {

	media := uploader.NameFromFile(q.File)

	f.Children = append(f.Children, &Question{
		Child:     multiforms.Child{Parent: f.Form, ChildIndex: index},
		QuizOrder: q.QuizOrder,
		Question:  q.Question,
		Answer:    q.Answer,
		MediaName: media,
	})
}

// AddTemplate appends the sub-form template to add a question.
func (f *Questions) AddTemplate(nQuestions int) {

	f.Children = append(f.Children, &Question{
		Child:     multiforms.Child{Parent: f.Form, ChildIndex: -1},
		QuizOrder: nQuestions + 1,
	})
}

// GetQuestions returns the user data as an array of structs.
// They are sent in the HTML form as arrays of values for each field name.
func (f *Questions) GetQuestions(vt ValidTypeFunc) (items []*Question, err error) {

	nItems := f.NChildItems()

	for i := 0; i < nItems; i++ {

		ix, err := f.ChildIndex("index", i)
		if err != nil {
			return nil, err
		}

		items = append(items, &Question{
			Child:     multiforms.Child{Parent: f.Form, ChildIndex: ix},
			QuizOrder: f.ChildMin("quizOrder", i, ix, 1),
			Question:  f.ChildText("question", i, ix, 2, 512),
			Answer:    f.ChildText("answer", i, ix, 1, 512),
			MediaName: f.ChildFile("mediaName", i, ix, vt),
			Version:   f.ChildPositive("mediaVersion", i, ix),
		})
	}

	// Add the child items back into the form, in case we need to redisplay it
	f.Children = items

	return items, nil
}
