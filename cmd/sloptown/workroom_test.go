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
	visit("introduce yourself to baker\n")
	const hello = "talk to baker\n1\ncancel\n"
	book := visit(hello, "-batch", "helper-greeting")
	if !strings.Contains(book, "Thanks for bringing the dough scraper. You can reach me on 201") || visit(hello, "-batch", "helper-greeting") != book {
		t.Fatal(book)
	}
	for _, actor := range []string{"player", "baker", "porter"} {
		scene, err := storage.OpenBakeryScene(db, scope, 42, "bakery/"+actor)
		if err != nil {
			t.Fatal(err)
		}
		if actor == "player" {
			if len(scene.PhoneBook.People) != 1 || scene.PhoneBook.Details["bakery/baker"] != "201" {
				t.Fatal(scene.PhoneBook)
			}
		} else if len(scene.PhoneBook.People) != 0 {
			t.Fatal(scene.PhoneBook)
		}
	}
	if r, err := storage.Replay(db, scope, mid.Sequence); err != nil || r.Tick != 29 {
		t.Fatal(r, err)
	}

}

func TestAskBakerAboutSupplies(t *testing.T) {
	t.Chdir(t.TempDir())
	const db = "request.sqlite"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	journalVisit(t, "journal\n", "-db", db, "-workroom")
	if _, err := storage.SaveCheckpoint(db, scope); err != nil {
		t.Fatal(err)
	}
	book := journalVisit(t, "open bakery door\ntalk to baker\n4\ncancel\njournal\n", "-db", db)
	if !strings.Contains(book, "The baker replied, “I could use the dough scraper. Please leave it in the workbench drawer.”") || strings.Contains(book, "I saw the dough scraper") {
		t.Fatal(book)
	}
	if reopened := journalVisit(t, "journal\n", "-db", db); reopened != book {
		t.Fatal(reopened)
	}
	if report, err := storage.Replay(db, scope, 0); err != nil || report.Tick != 2 {
		t.Fatal(report, err)
	}
}
