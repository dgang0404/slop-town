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
	if strings.Count(first, "<img ") != 2 || strings.Contains(first, "wooden spoon") ||
		!strings.Contains(first, "Ink sketch of the crate.") || !strings.Contains(first, "Ink sketch of the bakery door.") {
		t.Fatal("expected only the two objects seen before the crate opened")
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
	if !seen || !pickedUp || !strings.Contains(illustration, "Ink sketch of the wooden spoon.") ||
		strings.Count(book, "<img ") != 3 || strings.Count(book, "data:image/webp;base64,") != 3 {
		t.Fatal("expected a single embedded spoon sketch in its discovery passage")
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
	if !strings.Contains(html.UnescapeString(empty), emptyJournal) || strings.Contains(empty, "<img ") {
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
		strings.Contains(book, "<img ") {
		t.Fatal("a reported or similarly named object should not be pictured, and names must remain text")
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
		Action: simulation.ActionWait,
		FirstSeen: []simulation.Discovery{{
			Target: simulation.ObservedTarget{Ref: "bakery/spoon", Name: "wooden spoon"},
			Label:  "wooden spoon", Location: simulation.LocationContainer, Container: "blue crate",
		}},
	}}
	var book bytes.Buffer
	if err := writeIllustratedJournal(&book, entries[:1]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(book.String(), "I heard about a wooden spoon.") || strings.Contains(book.String(), "<img ") || strings.Contains(book.String(), "I saw the wooden spoon") {
		t.Fatal("the account should contain only the words heard")
	}
	book.Reset()
	if err := writeIllustratedJournal(&book, entries); err != nil {
		t.Fatal(err)
	}
	_, after, ok := strings.Cut(book.String(), "<p>I saw the wooden spoon in the blue crate.</p>")
	if !ok || strings.Count(book.String(), "<img ") != 1 || !strings.Contains(after, "Ink sketch of the wooden spoon.") {
		t.Fatal("expected the spoon sketch at the firsthand discovery during the wait")
	}
}
