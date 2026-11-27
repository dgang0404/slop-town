// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestSavedWorkroom(t *testing.T) {
	t.Chdir(t.TempDir())
	const db, player = "visit.sqlite", "bakery/player"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	visit := func(input string, flags ...string) {
		t.Helper()
		journalVisit(t, input+"journal\n", append([]string{"-db", db}, flags...)...)
	}
	visit("", "-workroom", "-seed", "42")
	if _, err := storage.SaveCheckpoint(db, scope); err != nil {
		t.Fatal(err)
	}
	visit("open bakery door\neast\neast\neast\nopen workroom door\nsouth\nsouth\nclose workroom door\nopen drawer\nrest\n")
	mid, err := storage.SaveCheckpoint(db, scope)
	if err != nil || mid.Tick != 10 {
		t.Fatal(mid, err)
	}
	scene, err := storage.OpenBakeryScene(db, scope, 43, player)
	if err != nil || scene.Self.Place != simulation.InWorkroom || scene.Details["bakery/workroom-door"].State != "closed" || scene.Details["bakery/workbench"].State != "open" {
		t.Fatal(scene, err)
	}
	visit("open workroom door\nnorth\nnorth\nwest\nwest\nwest\n")
	for _, from := range []uint64{0, mid.Sequence} {
		report, err := storage.Replay(db, scope, from)
		if err != nil || report.Tick != 16 {
			t.Fatal(report, err)
		}
	}
}
