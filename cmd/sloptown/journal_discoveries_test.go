// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestDiscoveryLookupAcrossVisits(t *testing.T) {
	db := filepath.Join(t.TempDir(), "bakery.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	play := func(commands string, args ...string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := runBakery(append([]string{"-play"}, args...), strings.NewReader(commands), &out, &diag); code != 0 {
			t.Fatalf("exit %d: %s", code, &diag)
		}
		_, sightings, ok := strings.Cut(out.String(), "My first sightings\n")
		if !ok {
			_, sightings, ok = strings.Cut(out.String(), "My last recorded sightings\n")
		}
		if !ok {
			t.Fatal(out.String())
		}
		return sightings
	}
	const empty = "I haven’t recorded any first sightings yet.\n"
	const missing = "I haven’t recorded a first sighting by that name.\n"
	const lastSearch = "JOURNAL  LAST  SEEN  the SPOON.\n"
	const lastFound = "\nDay 2 of spring, year 1\nI was on the street. I last saw the wooden spoon right beside me.\n"
	const found = "\nDay 2 of spring, year 1\nI was in the bakery. I saw the wooden spoon in the crate.\n"
	const search = "JOURNAL  DISCOVERIES  the SPOON.\n"
	enter := "open door\ne\ne\n" + strings.Repeat("wait\n", 20)
	const leave = "open crate\ntake spoon\nw\nw\ndrop spoon\n"
	if got := play(enter+"journal discoveries spoon\n", "-db", db); got != missing {
		t.Fatal(got)
	}
	if got := play(lastSearch, "-db", db); got != "I haven’t recorded a sighting by that name.\n" {
		t.Fatal(got)
	}
	cp, err := storage.SaveCheckpoint(db, scope)
	if err != nil || cp.Tick != 23 {
		t.Fatal(cp, err)
	}
	if got := play(leave+search, "-db", db); got != found {
		t.Fatal(got)
	}
	head, err := storage.Replay(db, scope, cp.Sequence)
	if err != nil || head.Tick != 28 {
		t.Fatal(head, err)
	}
	if got := play(search, "-db", db); got != found {
		t.Fatal(got)
	}
	if got := play(lastSearch, "-db", db); got != lastFound {
		t.Fatal(got)
	}
	const firstDay = "\nDay 1 of spring, year 1\n"
	const door = "I was on the street. I saw the open bakery door to the east.\n"
	const crate = "I was in the bakery doorway. I saw the closed crate to the northeast.\n"
	if got := play("journal discoveries\n", "-db", db); got != firstDay+door+crate+found {
		t.Fatal(got)
	}
	if got := play("journal discoveries crate\n", "-db", db); got != firstDay+crate {
		t.Fatal(got)
	}
	if got := play(enter + leave + search); got != found {
		t.Fatal(got)
	}
	if got := play(enter + leave + lastSearch); got != lastFound {
		t.Fatal(got)
	}
	hide := "open door\ne\ne\nopen crate\nclose crate\n" + strings.Repeat("wait\n", 24)
	if got := play(hide + lastSearch); got != "\nDay 1 of spring, year 1\nI was in the bakery. I last saw the wooden spoon in the crate.\n" {
		t.Fatal(got)
	}
	if got := play("journal last seen\n", "-db", db, "-timeline", "unvisited"); got != "I haven’t recorded any sightings yet.\n" {
		t.Fatal(got)
	}
	if got := play("journal discoveries spo\n", "-db", db); got != missing {
		t.Fatal(got)
	}
	if got := play("journal discoveries\n", "-db", db, "-timeline", "unvisited"); got != empty {
		t.Fatal(got)
	}
	if after, err := storage.Replay(db, scope, cp.Sequence); err != nil || after != head {
		t.Fatal(after, err)
	}
}

func TestDiscoveryLookupDoesNotSearchDialogue(t *testing.T) {
	entries := []simulation.JournalEntry{{Conversation: simulation.JournalSpeech{With: "baker", Words: "I heard about a wooden spoon."}}}
	var out bytes.Buffer
	if err := writeSightingIndex(&out, entries, "spoon", false); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "My first sightings\nI haven’t recorded a first sighting by that name.\n" {
		t.Fatal(got)
	}
	out.Reset()
	if err := writeSightingIndex(&out, entries, "spoon", true); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "My last recorded sightings\nI haven’t recorded a sighting by that name.\n" {
		t.Fatal(got)
	}
}
