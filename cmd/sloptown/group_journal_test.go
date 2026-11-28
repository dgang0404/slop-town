// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestOverheardCheckInsAcrossSQLiteRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	for _, tc := range []struct {
		name, start string
		hearings    int
	}{
		{"shopfront", "wait\nwait\n", 2},
		{"down the street", "south\nsouth\n", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := filepath.Join(t.TempDir(), "party.sqlite")
			journalVisit(t, "journal\n", "-practice", "-seed", "42", "-db", db)
			if _, err := storage.SaveCheckpoint(db, scope); err != nil {
				t.Fatal(err)
			}
			journalVisit(t, tc.start+strings.Repeat("wait\n", 6)+"journal\n", "-db", db)
			mid, err := storage.SaveCheckpoint(db, scope)
			if err != nil || mid.Tick != 8 {
				t.Fatal(mid, err)
			}
			finish := strings.Repeat("wait\n", 8) + "journal\n"
			book := journalVisit(t, finish, "-db", db, "-batch", "finish")
			if n := strings.Count(book, "I heard the baker check in on the porter. They said they felt fine."); n != tc.hearings {
				t.Fatalf("heard %d exchanges: %s", n, book)
			}
			if journalVisit(t, "journal\n", "-db", db) != book || journalVisit(t, finish, "-db", db, "-batch", "finish") != book {
				t.Fatal("reopen or retry changed the journal")
			}
			head, err := storage.Replay(db, scope, 0)
			if err != nil || head.Tick != 16 {
				t.Fatal(head, err)
			}
			if again, err := storage.Replay(db, scope, mid.Sequence); err != nil || again != head {
				t.Fatal(again, err)
			}
		})
	}
}
