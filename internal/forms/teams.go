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

	"inchworks.com/quiz/internal/models"
)

type Teams struct {
	*multiforms.Form
	Children []*Team
}

type Team struct {
	multiforms.Child
	models.Team
}

// NewTeams returns a form to edit teams
func NewTeams(data url.Values, token string) *Teams {
	return &Teams{
		Form:     multiforms.New(data, token),
		Children: make([]*Team, 0, 16),
	}
}

// Add appends a user sub-form to the form
func (f *Teams) Add(index int, t *models.Team) {

	f.Children = append(f.Children, &Team{
		Child: multiforms.Child{Parent: f.Form, ChildIndex: index},
		Team:  *t,
	})
}

// AddTemplate appends the sub-form template to add a team.
func (f *Teams) AddTemplate() {

	f.Children = append(f.Children, &Team{
		Child: multiforms.Child{Parent: f.Form, ChildIndex: -1},
	})
}

// GetTeams returns the user data as an array of structs.
// They are sent in the HTML form as arrays of values for each field name.
func (f *Teams) GetTeams() (items []*Team, err error) {

	nItems := f.NChildItems()

	for i := 0; i < nItems; i++ {

		ix, err := f.ChildIndex("index", i)
		if err != nil {
			return nil, err
		}

		items = append(items, &Team{
			Child: multiforms.Child{Parent: f.Form, ChildIndex: ix},
			Team: models.Team{
				Name:   f.ChildRequired("name", i, ix),
				Access: f.ChildGet("access", i),
			},
		})
	}

	// Add the child items back into the form, in case we need to redisplay it
	f.Children = items

	return items, nil
}
