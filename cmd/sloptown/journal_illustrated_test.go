// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"errors"
	"html"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestIllustratedJournalFollowsTheSpoon(t *testing.T) {
	t.Chdir(t.TempDir())
	// Reserving the default book name for a save exercises collision handling.
	const db = "slop-town-journal.html"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	export := func(timeline string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := run([]string{"journal", "-illustrated", "-db", db, "-timeline", timeline}, &out, &diag); code != 0 {
			t.Fatalf("export exit %d: %s", code, &diag)
		}
		return out.String()
	}
	const arrive = "wait\nopen door\ne\ne\n"
	const discover = "open crate\ntake spoon\nput it in crate\nclose crate\n"
	args := []string{"-db", db}
	journalVisit(t, "journal\n", args...)
	if _, err := storage.SaveCheckpoint(db, scope); err != nil {
		t.Fatal(err)
	}
	journalVisit(t, arrive+"journal\n", args...)
	first := export("main")
	if strings.Count(first, "<figure>") != 2 || strings.Contains(first, "wooden spoon") ||
		strings.Contains(first, "c*") || !strings.Contains(first, "?") {
		t.Fatal("expected the views before the crate opened")
	}
	batch := []string{"-db", db, "-batch", "discover"}
	journalVisit(t, discover+"journal\n", batch...)
	head, err := storage.Replay(db, scope, 0)
	if err != nil || head.Tick != 8 {
		t.Fatal(head, err)
	}
	book := export("main")
	_, later, seen := strings.Cut(book, "I saw the wooden spoon in the crate.")
	illustration, _, pickedUp := strings.Cut(later, "I took the wooden spoon.")
	if !seen || !pickedUp || !strings.Contains(illustration, "c*") ||
		strings.Count(book, "<figure>") != 3 || strings.Contains(book, "data:image/") {
		t.Fatal("expected the open crate and its visible contents on the discovery map")
	}
	for _, tc := range []struct {
		args  []string
		route string
		name  string
	}{
		{batch, discover, "slop-town-journal-2.html"}, // A retry from SQLite.
		{nil, arrive + discover, "slop-town-journal-3.html"},
	} {
		var out, diag bytes.Buffer
		if code := runBakery(append([]string{"-play"}, tc.args...), strings.NewReader(tc.route+"export illustrated journal\n"), &out, &diag); code != 0 ||
			!strings.Contains(out.String(), "Saved your journal to "+tc.name+".") {
			t.Fatalf("play exit %d: %s, %s", code, &out, &diag)
		}
		if b, err := os.ReadFile(tc.name); err != nil || string(b) != book {
			t.Fatalf("export %s differs: %v", tc.name, err)
		}
	}
	journalVisit(t, "journal\n", "-db", db, "-timeline", "unvisited")
	empty := export("unvisited")
	if !strings.Contains(html.UnescapeString(empty), emptyJournal) || strings.Contains(empty, "<figure>") {
		t.Fatal("expected an empty book on another timeline")
	}
	if after, err := storage.Replay(db, scope, 0); err != nil || after != head {
		t.Fatal(after, err)
	}
}

func TestIllustratedJournalKeepsQuotedWordsAsText(t *testing.T) {
	entries := []simulation.JournalEntry{{
		Conversation: simulation.JournalSpeech{With: "baker", Words: "I heard about a wooden spoon.", Reply: "Ask Kit & <Jo>."},
		FirstSeen: []simulation.Discovery{
			{Target: simulation.ObservedTarget{Ref: "bakery/other", Name: "wooden spoon"}, Label: "wooden spoon", Location: simulation.LocationHere},
			{Target: simulation.ObservedTarget{Ref: "bakery/spoon", Name: "wooden spoon rack"}, Label: "wooden spoon rack", Location: simulation.LocationHere},
		},
	}}
	var out bytes.Buffer
	if err := writeIllustratedJournal(&out, entries); err != nil {
		t.Fatal(err)
	}
	book := out.String()
	if !strings.Contains(book, "I heard about a wooden spoon.") || !strings.Contains(book, "Kit &amp; &lt;Jo&gt;") ||
		strings.Contains(book, "<figure>") {
		t.Fatal("names alone cannot reconstruct a missing observation grid")
	}
	r, w := io.Pipe()
	r.Close()
	defer w.Close()
	if err := writeIllustratedJournal(w, entries); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

func TestJournalHearsAboutSpoonBeforeSeeingIt(t *testing.T) {
	entries := []simulation.JournalEntry{{
		Conversation: simulation.JournalSpeech{With: "baker", Words: "I heard about a wooden spoon."},
	}, {
		Action:        simulation.ActionWait,
		DiscoveryGrid: "@ c*",
		FirstSeen: []simulation.Discovery{{
			Target: simulation.ObservedTarget{Ref: "bakery/spoon", Name: "wooden spoon"},
			Label:  "wooden spoon", Location: simulation.LocationContainer, Container: "blue crate",
		}},
	}}
	var book bytes.Buffer
	if err := writeIllustratedJournal(&book, entries[:1]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(book.String(), "I heard about a wooden spoon.") || strings.Contains(book.String(), "<figure>") || strings.Contains(book.String(), "I saw the wooden spoon") {
		t.Fatal("the account should contain only the words heard")
	}
	book.Reset()
	if err := writeIllustratedJournal(&book, entries); err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(book.String(), "<p>I saw the wooden spoon in the blue crate.</p>")
	if !ok || strings.Count(book.String(), "<figure>") != 1 || !strings.Contains(after, "<pre>@ c*</pre>") {
		t.Fatal("expected the recorded view at the firsthand discovery during the wait")
	}
}

func TestGeneratedFurnishingsAcrossRestart(t *testing.T) {
	t.Chdir(t.TempDir())
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const db = "generated.sqlite"
	visit := func(input string, flags ...string) string {
		t.Helper()
		return journalVisit(t, input+"journal\n", append([]string{"-db", db}, flags...)...)
	}
	visit("", "-generated", "-seed", "42")
	if _, err := storage.SaveCheckpoint(db, scope); err != nil {
		t.Fatal(err)
	}
	visit("open door\nwalk to crate\n")
	corner, err := storage.SaveCheckpoint(db, scope)
	if err != nil {
		t.Fatal(err)
	}
	const discover = "open crate\nexamine scraper\ntake scraper\n"
	book := visit(discover, "-practice", "-seed", "43", "-batch", "supplies")
	if !strings.Contains(book, "I found the dough scraper in the crate, a brass rivet in its handle.") ||
		!strings.Contains(book, "I took the dough scraper.") {
		t.Fatal(book)
	}
	if repeated := visit(discover, "-batch", "supplies"); repeated != book {
		t.Fatal("retry changed the journal")
	}
	head, err := storage.Replay(db, scope, 0)
	if err != nil || head.Tick != 7 {
		t.Fatal(head, err)
	}
	if again, err := storage.Replay(db, scope, corner.Sequence); err != nil || again != head {
		t.Fatal(again, err)
	}
	entries, err := storage.BakeryJournal(db, scope, "bakery/player")
	if err != nil {
		t.Fatal(err)
	}
	var htmlBook bytes.Buffer
	if err := writeIllustratedJournal(&htmlBook, entries); err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(htmlBook.String(), "I saw the dough scraper in the crate. I saw the wooden spoon in the crate.")
	grid, _, next := strings.Cut(after, "I found the dough scraper")
	if !ok || !next || !strings.Contains(grid, "c**") || strings.Count(grid, "<figure>") != 1 {
		t.Fatal("both discovered tools should share the grid recorded before pickup")
	}
	t.Logf("tick %d, sequence %d, sha256 %s", head.Tick, head.Sequence, head.Hash)
}
