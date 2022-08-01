// Copyright © Rob Burke inchworks.com, 2019.

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

// Processing to show server usage.

package quiz

import (
	"github.com/inchworks/usage"
)

// Daily and monthly usage statistics

func (s *QuizState) ForUsage(detail usage.Detail) *dataUsagePeriods {

	var title string
	var fmt string
	switch detail {
	case usage.Day:
		title = "Daily Usage"
		fmt = "Mon 2 Jan"

	case usage.Month:
		title = "Monthly Usage"
		fmt = "January 2006"

	default:
		return nil
	}

	// serialisation
	defer s.updatesNone()()

	// get stats
	stats := usage.Get(s.app.statisticStore, detail)
	var us []*dataUsage

	for _, s := range stats {

		us = append(us, &dataUsage{
			Date:  s[0].Start.Format(fmt),
			Stats: s,
		})
	}

	return &dataUsagePeriods{
		Title: title,
		Usage: us,
	}
}
