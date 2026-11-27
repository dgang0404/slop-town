// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"strings"
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

func TestWorkroomSupplyOrder(t *testing.T) {
	t.Chdir(t.TempDir())
	const db = "order.sqlite"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	visit := func(commands string, flags ...string) string {
		t.Helper()
		return journalVisit(t, commands+"journal\n", append([]string{"-db", db}, flags...)...)
	}
	visit("", "-workroom")
	if _, err := storage.SaveCheckpoint(db, scope); err != nil {
		t.Fatal(err)
	}
	const start = "open bakery door\nwalk to workroom door\nopen workroom door\nwalk to drawer\nexamine order\n"
	book := visit(start)
	if !strings.Contains(book, "leave the dough scraper in the workbench drawer") || strings.Contains(book, "requested tool is") {
		t.Fatal(book)
	}
	book = visit("open drawer\nexamine order\nnorth\nwalk to crate\nopen crate\ntake scraper\n")
	if !strings.Contains(book, "missing from the open drawer") {
		t.Fatal(book)
	}
	const finish = "put scraper in drawer\nexamine order\n"
	book = visit(finish, "-batch", "deliver")
	if !strings.Contains(book, "The requested tool is in the drawer.") || visit(finish, "-batch", "deliver") != book {
		t.Fatal(book)
	}
	if _, err := storage.Replay(db, scope, 0); err != nil {
		t.Fatal(err)
	}
}

func TestBakerCollectsOrderAcrossRestart(t *testing.T) {
	t.Chdir(t.TempDir())
	const db = "rounds.sqlite"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	visit := func(commands string, flags ...string) string {
		t.Helper()
		return journalVisit(t, commands+"journal\n", append([]string{"-db", db}, flags...)...)
	}
	visit("", "-workroom")
	if _, err := storage.SaveCheckpoint(db, scope); err != nil {
		t.Fatal(err)
	}
	const delivery = "open bakery door\nwalk to workroom door\nopen workroom door\nwalk to drawer\nexamine order\nopen drawer\nnorth\nwalk to crate\nopen crate\ntake scraper\nput scraper in drawer\nclose drawer\n"
	visit(delivery + strings.Repeat("wait\n", 12))
	mid, err := storage.SaveCheckpoint(db, scope)
	if err != nil || mid.Tick != 26 {
		t.Fatal(mid, err)
	}
	read := func() simulation.Scene {
		t.Helper()
		scene, err := storage.OpenBakeryScene(db, scope, 42, "bakery/baker")
		if err != nil {
			t.Fatal(err)
		}
		return scene
	}
	if holder := read().Details["bakery/scraper"].Holder; holder != "bakery/workbench" {
		t.Fatal(holder)
	}
	visit("wait\n", "-batch", "pickup")
	if holder := read().Details["bakery/scraper"].Holder; holder != "bakery/baker" {
		t.Fatal(holder)
	}
	visit("wait\n", "-batch", "pickup")
	for _, seq := range []uint64{0, mid.Sequence} {
		r, err := storage.Replay(db, scope, seq)
		if err != nil || r.Tick != 27 {
			t.Fatal(r, err)
		}
	}
}
