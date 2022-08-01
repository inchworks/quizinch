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

// SQL operations on statistic table.

import (
	"errors"
	"log"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/inchworks/usage"
)

const (
	statsDelete = `DELETE FROM statistic WHERE id = ?`

	statsInsert = `
		INSERT INTO statistic (event, category, count, start, detail) VALUES (:event, :category, :count, :start, :detail)`

	statsUpdate = `
		UPDATE statistic
		SET event=:event, category=:category, count=:count, start=:start, detail=:detail
		WHERE id=:id
	`
)

const (
	statsSelect        = `SELECT * FROM statistic`
	statsOrderCategory = ` ORDER BY category, start`
	statsOrderEvent    = ` ORDER BY event, start`
	statsOrderTime     = ` ORDER BY start DESC, category ASC, count DESC, event ASC`

	statsWhereBefore = statsSelect + ` WHERE start < ? AND detail = ?`
	statsWhereStart  = statsSelect + ` WHERE event = ? AND start = ? AND detail = ?`
	statsWhereEvent  = statsSelect + ` WHERE event = ? AND detail = ?`

	statsBeforeByCategory = statsWhereBefore + statsOrderCategory
	statsBeforeByEvent    = statsWhereBefore + statsOrderEvent
	statsBeforeByTime     = statsWhereBefore + statsOrderTime

	statsDeleteOld = `DELETE FROM statistic WHERE start < ? AND detail = ?`
)

type StatisticStore struct {
	GalleryId int64
	tx        *sqlx.Tx
	store
}

// NewStatisticStore returns a database store, used for the lifetime of the server.
func NewStatisticStore(db *sqlx.DB, errorLog *log.Logger) *StatisticStore {

	st := StatisticStore{
		store: store{
			DBX:       db,
			errorLog:  errorLog,
			sqlDelete: statsDelete,
			sqlInsert: statsInsert,
			sqlUpdate: statsUpdate,
		},
	}
	st.ptx = &st.tx
	return &st
}

// BeforeByTime returns statistics for a specified period, ordered.
func (st *StatisticStore) BeforeByTime(before time.Time, detail usage.Detail) []*usage.Statistic {

	var stats []*usage.Statistic

	if err := st.DBX.Select(&stats, statsBeforeByTime, before, detail); err != nil {
		st.logError(err)
		return nil
	}
	return stats
}

// BeforeByCategory returns statistics before the specified start time, ordered by category and time.
func (st *StatisticStore) BeforeByCategory(before time.Time, detail usage.Detail) []*usage.Statistic {

	var stats []*usage.Statistic

	if err := st.DBX.Select(&stats, statsBeforeByCategory, before, detail); err != nil {
		st.logError(err)
		return nil
	}
	return stats
}

// BeforeByEvent returns statistics before the specified start time, ordered by event and time.
func (st *StatisticStore) BeforeByEvent(before time.Time, detail usage.Detail) []*usage.Statistic {

	var stats []*usage.Statistic

	if err := st.DBX.Select(&stats, statsBeforeByEvent, before, detail); err != nil {
		st.logError(err)
		return nil
	}
	return stats
}

// Delete old statistics
//
// Note that this is atypical as no other tables have specific functions for updates.

func (st *StatisticStore) DeleteOld(before time.Time, detail usage.Detail) error {

	tx := *st.ptx
	if tx == nil {
		return errors.New("statics: cannot update store")
	}

	if _, err := tx.Exec(statsDeleteOld, before, detail); err != nil {
		return st.logError(err)
	}

	return nil
}

// Get single statistic, need not exist

func (st *StatisticStore) GetEvent(event string, start time.Time, detail usage.Detail) *usage.Statistic {

	var s usage.Statistic

	if err := st.DBX.Get(&s, statsWhereStart, event, start, detail); err != nil {
		return nil
	}

	return &s
}

// Get lifetime statistic, need not exist

func (st *StatisticStore) GetMark(event string) *usage.Statistic {

	var s usage.Statistic

	if err := st.DBX.Get(&s, statsWhereEvent, event, usage.Mark); err != nil {
		return nil
	}

	return &s
}

// Transaction starts a transaction for updates.
// It returns a function to be called to end the transaction.
// Note that this function is atypical as other tables share an application-level transaction.
func (st *StatisticStore) Transaction() func() {

	// start transaction
	// ## sometimes fails during debugging, and I don't understand why
	var err error
	st.tx, err = st.DBX.Beginx()
	if err != nil {
		st.logError(err)
	}

	return func() {
		// end transaction, if there is one
		tx := st.tx
		if tx != nil {
			tx.Commit()
		}

		st.tx = nil
	}
}

// Insert or update statistic

func (st *StatisticStore) Update(s *usage.Statistic) error {

	if *st.ptx == nil {
		return errors.New("statics: cannot update store")
	}
	return st.updateData(&s.Id, s)
}
