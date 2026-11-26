// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestChapteredJournalExport(t *testing.T) {
	t.Chdir(t.TempDir())
	// Deliberately give the save the first export's filename.
	const db = "slop-town-journal.txt"
	const route = "wait\nwait\nwait\nwait\n"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	args := []string{"-practice", "-db", db, "-batch", "visit"}
	journalVisit(t, "journal\n", args...)
	if _, err := storage.SaveCheckpoint(db, scope); err != nil {
		t.Fatal(err)
	}
	book := "My journal\n" + journalVisit(t, route+"journal\n", args...)
	before, err := storage.Replay(db, scope, 0)
	if err != nil || before.Tick != 4 {
		t.Fatalf("before export: %+v, %v", before, err)
	}
	const meeting = "I waited a while.\n\nChapter 2 — Meeting the baker and the porter\n\nI waited. The baker introduced themselves to me. The porter introduced themselves to me."
	if !strings.Contains(book, meeting) || strings.Count(book, "Chapter ") != 2 {
		t.Fatal(book)
	}
	// Reopen and retry the visit, then export a separate in-memory run in story mode.
	for _, tc := range []struct {
		args []string
		file string
	}{
		{args, "slop-town-journal-2.txt"},
		{[]string{"-practice", "-narration", "story"}, "slop-town-journal-3.txt"},
	} {
		var out, diag bytes.Buffer
		if code := runBakery(append([]string{"-play"}, tc.args...), strings.NewReader(route+"export journal\n"), &out, &diag); code != 0 ||
			!strings.Contains(out.String(), "Saved your journal to "+tc.file+".") {
			t.Fatalf("exit %d: %s, %s", code, &out, &diag)
		}
		data, err := os.ReadFile(tc.file)
		if err != nil || string(data) != book {
			t.Fatalf("export %s: %s, %v", tc.file, data, err)
		}
	}
	var out, diag bytes.Buffer
	if code := run([]string{"journal", "-db", db}, &out, &diag); code != 0 || out.String() != book || diag.Len() != 0 {
		t.Fatalf("shell export: exit %d, %s, %s", code, &out, &diag)
	}
	if after, err := storage.Replay(db, scope, 0); err != nil || after != before {
		t.Fatalf("after export: %+v, %v", after, err)
	}

	away := journalVisit(t, "south\nsouth\nwait\nwait\njournal\n", append(args, "-timeline", "away")...)
	out.Reset()
	if code := run([]string{"journal", "-db", db, "-timeline", "away"}, &out, &diag); code != 0 ||
		out.String() != "My journal\n"+away || strings.Contains(away, "Meeting ") || strings.Count(away, "Chapter ") != 1 {
		t.Fatalf("exported a missed meeting: %s, %s", &out, &diag)
	}
}

func TestChaptersUseFirsthandMilestones(t *testing.T) {
	book := journalVisit(t, "open door\ne\ne\nw\nw\ne\ne\njournal\n")
	if strings.Count(book, "Chapter ") != 2 ||
		!strings.Contains(book, "Chapter 2 — Inside the bakery\n\nI moved east. I was in the bakery.") {
		t.Fatal(book)
	}
	outgoing := journalVisit(t, "south\nintroduce yourself to the porter\nintroduce yourself to the porter\njournal\n", "-practice")
	if !strings.Contains(outgoing, "Chapter 2 — Meeting the porter\n\nI introduced myself to the porter.") ||
		strings.Count(outgoing, "Meeting the porter") != 1 || strings.Count(outgoing, "I introduced myself to the porter.") != 2 {
		t.Fatal(outgoing)
	}
	// Personal introductions count in both directions and keep distinct people
	// separate even when their visible names match.
	one := simulation.ObservedTarget{Ref: "first", Name: "porter"}
	two := simulation.ObservedTarget{Ref: "second", Name: "porter"}
	c := journalChapters{met: make(map[string]bool)}
	heard := func(person simulation.ObservedTarget) simulation.WitnessedAction {
		return simulation.WitnessedAction{Actor: person, Action: simulation.ActionIntroduce}
	}
	if title := c.begin(simulation.JournalEntry{IntroducedTo: one}); title != "Meeting the porter" || c.number != 1 {
		t.Fatal(title, c.number)
	}
	if title := c.begin(simulation.JournalEntry{Witnessed: []simulation.WitnessedAction{heard(one)}}); title != "" || c.number != 1 {
		t.Fatal(title, c.number)
	}
	if title := c.begin(simulation.JournalEntry{Witnessed: []simulation.WitnessedAction{heard(two)}}); title != "Meeting the porter" || c.number != 2 {
		t.Fatal(title, c.number)
	}
	// When the first recorded turn contains several milestones, it has one opening.
	c = journalChapters{met: make(map[string]bool)}
	first := simulation.JournalEntry{Observations: []string{"You are in the bakery."}, Witnessed: []simulation.WitnessedAction{heard(one), heard(two)}}
	if title := c.begin(first); title != "Introductions" || c.number != 1 {
		t.Fatal(title, c.number)
	}
	if title := c.begin(first); title != "" || c.number != 1 {
		t.Fatal(title, c.number)
	}
}

func TestJournalExportErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.sqlite")
	var out, diag bytes.Buffer
	if code := run([]string{"journal", "-db", path}, &out, &diag); code != 1 || out.Len() != 0 || diag.Len() == 0 {
		t.Fatalf("missing save: exit %d, %s, %s", code, &out, &diag)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing save was created: %v", err)
	}
	journalVisit(t, "journal\n", "-db", path)
	r, w := io.Pipe()
	r.Close()
	defer w.Close()
	diag.Reset()
	if code := run([]string{"journal", "-db", path}, w, &diag); code != 1 || diag.Len() == 0 {
		t.Fatalf("write error: exit %d, %s", code, &diag)
	}
}
