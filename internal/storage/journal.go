// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import (
	"encoding/json"

	"github.com/dgang0404/slop-town/internal/simulation"
)

// BakeryJournal reads the receipts, witnessed events and notebook at one
// committed head, including turns taken by another session on the same save.
func BakeryJournal(path string, scope simulation.Scope, actor string) ([]simulation.JournalEntry, error) {
	db, err := openTimelineDB(path, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	state, err := committedState(tx, scope, 4)
	if err != nil {
		return nil, err
	}
	visit := func(record func(simulation.Event) error) error {
		rows, err := tx.Query("SELECT event FROM timeline_event WHERE world=? AND timeline=? ORDER BY seq", scope.WorldID, scope.TimelineID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var data []byte
			var event simulation.Event
			if err := rows.Scan(&data); err != nil {
				return err
			}
			if err := json.Unmarshal(data, &event); err != nil {
				return err
			}
			if err := record(event); err != nil {
				return err
			}
		}
		return rows.Err()
	}
	return state.Bakery.Journal(scope, actor, visit, func(request string) (*simulation.Receipt, error) {
		return loadReceipt(tx, scope, actor, request)
	})
}
