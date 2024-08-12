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

// SQL operations on user table.

import (
	"log"

	"github.com/inchworks/webparts/v2/users"
	"github.com/jmoiron/sqlx"

	"inchworks.com/quiz/internal/models"
)

const (
	userDelete = `DELETE FROM user WHERE id = ?`

	userInsert = `
		INSERT INTO user (parent, username, name, role, status, password, created) VALUES (:parent, :username, :name, :role, :status, :password, :created)`

	userUpdate = `
		UPDATE user
		SET username=:username, name=:name, role=:role, status=:status, password=:password, created=:created
		WHERE id=:id
	`
)

const (
	userSelect    = `SELECT * FROM user`
	userOrderName = ` ORDER BY name`

	userWhereId      = userSelect + ` WHERE id = ?`
	userWhereName    = userSelect + ` WHERE parent = ? AND username = ?`
	usersWhereparent = userSelect + ` WHERE parent = ?`

	usersByName = usersWhereparent + userOrderName

	userCount = `SELECT COUNT(*) FROM user WHERE parent = ?`
)

type UserStore struct {
	ParentId int64
	store
}

func NewUserStore(db *sqlx.DB, tx **sqlx.Tx, errorLog *log.Logger) *UserStore {

	return &UserStore{
		store: store{
			DBX:       db,
			ptx:       tx,
			errorLog:  errorLog,
			sqlDelete: userDelete,
			sqlInsert: userInsert,
			sqlUpdate: userUpdate,
		},
	}
}

// All users, unordered

func (st *UserStore) All() []*users.User {

	var users []*users.User

	if err := st.DBX.Select(&users, usersWhereparent, st.ParentId); err != nil {
		st.logError(err)
		return nil
	}
	return users
}

// All users, in name order

func (st *UserStore) ByName() []*users.User {

	var users []*users.User

	if err := st.DBX.Select(&users, usersByName, st.ParentId); err != nil {
		st.logError(err)
		return nil
	}
	return users
}

// Count of users

func (st *UserStore) Count() int {

	var n int

	if err := st.DBX.Get(&n, userCount, st.ParentId); err != nil {
		st.logError(err)
		return 0
	}

	return n
}

// Get user

func (st *UserStore) Get(id int64) (*users.User, error) {

	var t users.User

	if err := st.DBX.Get(&t, userWhereId, id); err != nil {
		// unknown user ID is possible, not logged as an error
		return nil, st.convertError(err)
	}

	return &t, nil
}

// Get user ID for username

func (st *UserStore) GetNamed(username string) (*users.User, error) {

	var t users.User

	if err := st.DBX.Get(&t, userWhereName, st.ParentId, username); err != nil {
		// unknown users are expected, not logged as an error
		return nil, st.convertError(err)
	}

	return &t, nil
}

// IsNoRecord returns true if error is "record not found"

func (st *UserStore) IsNoRecord(err error) bool {
	return err == models.ErrNoRecord
}

// Convenience function for user's name

func (st *UserStore) Name(id int64) string {

	u, err := st.Get(id)

	if err != nil {
		return ""
	} else {
		return u.Name
	}
}

// Redundant function, never used.
func (st *UserStore) Rollback() {
}

// Insert or update user

func (st *UserStore) Update(u *users.User) error {

	u.Parent = st.ParentId

	return st.updateData(&u.Id, u)
}
