// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

// BakeryDate is the date shared by characters on the same bakery timeline.
// Day counts from one within each season. Year counts from one at world creation.
type BakeryDate struct {
	Day    int64
	Season string
	Year   int64
}

// bakeryDateAt uses the committed clock or an event's tick. The starting
// instant is day one and the action that reaches tick 24 lands on day two.
func bakeryDateAt(tick int64) BakeryDate {
	const ticksPerDay, daysPerSeason = 24, 7
	seasons := [...]string{"spring", "summer", "autumn", "winter"}
	day := tick / ticksPerDay
	season := day / daysPerSeason
	const seasonsPerYear = int64(len(seasons))
	return BakeryDate{
		Day:    day%daysPerSeason + 1,
		Season: seasons[season%seasonsPerYear],
		Year:   season/seasonsPerYear + 1,
	}
}
