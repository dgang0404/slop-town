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

func TestCalendarAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bakery.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	play := func(batch, input string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := runBakery([]string{"-play", "-db", path, "-batch", batch}, strings.NewReader(input), &out, &diag); code != 0 {
			t.Fatalf("exit %d: %s", code, &diag)
		}
		return out.String()
	}
	play("spring", strings.Repeat("wait\n", 23))
	cp, err := storage.SaveCheckpoint(path, scope)
	if err != nil || cp.Tick != 23 {
		t.Fatal(cp, err)
	}
	const input = "calendar\nwalk to door\nwait\ncalendar\njournal\n"
	out := play("next", input)
	_, book, ok := strings.Cut(out, "My journal\n")
	if !ok || !strings.Contains(out, "Day 1 of spring, year 1.\n") || !strings.Contains(out, "Day 2 of spring, year 1.\n") ||
		!strings.Contains(book, "I waited a while.\n\nChapter 2 — Day 2 of spring, year 1\n\nI waited.\n") {
		t.Fatal(out)
	}
	before, err := storage.Replay(path, scope, cp.Sequence)
	if err != nil || before.Tick != 24 {
		t.Fatal(before, err)
	}
	retry := play("next", input)
	if strings.Count(retry, "Day 2 of spring, year 1.\n") != 2 || !strings.HasSuffix(retry, book) {
		t.Fatal(retry)
	}
	var exported, diag bytes.Buffer
	if code := run([]string{"journal", "-db", path}, &exported, &diag); code != 0 || exported.String() != "My journal\n"+book {
		t.Fatalf("export exit %d: %s, %s", code, &exported, &diag)
	}
	if after, err := storage.Replay(path, scope, cp.Sequence); err != nil || after != before {
		t.Fatal(after, err)
	}
}

func TestSeasonChapterSharesAnArrival(t *testing.T) {
	// Cross the threshold into the bakery on the first turn of summer.
	book := journalVisit(t, strings.Repeat("wait\n", 165)+"open door\ne\ne\nopen crate\njournal\n")
	want := "Chapter 8 — Day 1 of summer, year 1 · Inside the bakery\n\nI opened the crate. I saw the wooden spoon in the crate."
	if !strings.Contains(book, want) || strings.Count(book, "Chapter ") != 8 {
		t.Fatal(book)
	}
}

func TestJournalWalkAcrossDays(t *testing.T) {
	for _, tc := range []struct {
		waits  int
		dayTwo string
	}{
		{15, "I reached the crate.\n"},
		{16, "I continued toward the crate.\nI reached the crate.\n"},
	} {
		route := "open door\ne\ne\ne\ns\ns\n" + strings.Repeat("wait\n", tc.waits) + "walk to crate\njournal\n"
		book := journalVisit(t, route)
		oldDay, newDay, ok := strings.Cut(book, "Day 2 of spring, year 1\n\n")
		if !ok || strings.Count(oldDay, "I walked toward the crate.") != 1 || newDay != tc.dayTwo {
			t.Fatal(book)
		}
	}
}
