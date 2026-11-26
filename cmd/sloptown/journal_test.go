// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func journalVisit(t *testing.T, input string, args ...string) string {
	t.Helper()
	var out, diag bytes.Buffer
	if code := runBakery(append([]string{"-play"}, args...), strings.NewReader(input), &out, &diag); code != 0 {
		t.Fatalf("bakery exited %d: %s", code, &diag)
	}
	_, book, found := strings.Cut(out.String(), "My journal\n")
	if !found {
		t.Fatal(out.String())
	}
	return book
}

func TestJournalAcrossVisits(t *testing.T) {
	const empty = "I haven't recorded anything yet.\n"
	const route = "open door\ne\ne\nopen crate\ntake spoon\nput it in crate\nwait\n"
	path := filepath.Join(t.TempDir(), "bakery.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	args := []string{"-db", path, "-batch", "visit"}
	if got := journalVisit(t, "look\nactions\nopen cupboard\njournal\nquit\n", args...); got != empty {
		t.Fatal(got)
	}
	if _, err := storage.SaveCheckpoint(path, scope); err != nil {
		t.Fatal(err)
	}
	want := journalVisit(t, route+"journal\n")
	if !strings.Contains(want, "I opened the crate. I saw the wooden spoon in the crate.\nI took the wooden spoon.\nI put the wooden spoon in the crate.") ||
		strings.Contains(want, "bakery/") || strings.Contains(want, "@") {
		t.Fatal(want)
	}
	if got := journalVisit(t, route+"journal\n", append(args, "-narration", "story")...); got != want {
		t.Fatal(got)
	}
	head := func() storage.ReplayReport {
		t.Helper()
		r, err := storage.Replay(path, scope, 0)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	before := head()
	if before.Tick != 7 || before.Sequence != 14 {
		t.Fatal(before)
	}
	for _, input := range []string{"journal\n", route + "journal\n", "journal\njournal\n"} {
		got := journalVisit(t, input, args...)
		for _, reading := range strings.Split(got, "My journal\n") {
			if reading != want {
				t.Fatal(reading)
			}
		}
	}
	if got := journalVisit(t, "journal\n", append(args, "-timeline", "another")...); got != empty || head() != before {
		t.Fatal("journal crossed timeline or advanced the save", got)
	}

	// The receipt write fails after the rest of a turn has been staged.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TRIGGER fail_page BEFORE INSERT ON request_receipt
		BEGIN SELECT RAISE(FAIL,'interrupted save'); END`); err != nil {
		t.Fatal(err)
	}
	closeCrate := simulation.Proposal{Scope: scope, Version: 1, RequestID: "close", ByName: true, Action: simulation.ActionClose, Args: simulation.ActionArgs{Target: "crate"}}
	if _, _, err := storage.SubmitBakery(path, scope, "bakery/player", closeCrate); err == nil || !strings.Contains(err.Error(), "interrupted save") {
		t.Fatal(err)
	}
	if got := journalVisit(t, "journal\n", args...); got != want || head() != before {
		t.Fatal("interrupted action left a journal entry", got)
	}
}

func TestJournalRemembersOnlyFirsthandEncounters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "town.sqlite")
	args := []string{"-practice", "-db", path}
	upClose := journalVisit(t, strings.Repeat("wait\n", 12)+"journal\n", args...)
	for _, phrase := range []string{"The porter introduced themselves to me.", "The baker introduced themselves to me.", "I saw the baker sit down.", "I saw the porter sit down."} {
		if strings.Count(upClose, phrase) != 1 {
			t.Fatalf("wanted one %q in %s", phrase, upClose)
		}
	}
	away := journalVisit(t, "south\nsouth\n"+strings.Repeat("wait\n", 10)+"journal\n", append(args, "-timeline", "away")...)
	if strings.Contains(away, "introduced") || strings.Contains(away, "the baker sit down") || strings.Contains(upClose, "open the bakery door") {
		t.Fatalf("unwitnessed interaction in journal:\n%s\n%s", upClose, away)
	}
}

func TestJournalHearsAnotherActorsTurn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "town.sqlite")
	args := []string{"-practice", "-db", path}
	journalVisit(t, "south\njournal\n", args...)
	p := simulation.Proposal{Scope: simulation.Scope{WorldID: "bakery", TimelineID: "main"}, Version: 1,
		Action: simulation.ActionIntroduce, Args: simulation.ActionArgs{Target: "bakery/player"}}
	// Two addressed introductions are audible even though familiarity is only learned once.
	for _, id := range []string{"first", "again"} {
		p.RequestID = id
		r, _, err := storage.SubmitBakery(path, p.Scope, "bakery/porter", p)
		if err != nil || r.Status != "accepted" {
			t.Fatal(r, err)
		}
	}
	book := journalVisit(t, "journal\n", args...)
	if strings.Count(book, "The porter introduced themselves to me.") != 2 || strings.Contains(book, "I introduced myself") {
		t.Fatal(book)
	}
}

func TestJournalRecallsAWalk(t *testing.T) {
	book := journalVisit(t, "wait\neast\neast\neast\nsouth\nsouth\nwalk to crate\nopen it\njournal\n", "-practice")
	if strings.Count(book, "I walked toward the crate.") != 1 || !strings.Contains(book, "I was within reach of the crate.") ||
		!strings.Contains(book, "My legs felt tired.") || !strings.Contains(book, "I opened the crate.") {
		t.Fatal(book)
	}
	lost := journalVisit(t, "wait\nwait\nwait\nwest\nnorth\nwalk to baker\njournal\n", "-practice")
	if !strings.Contains(lost, "I walked toward the baker. I lost sight of my destination.") {
		t.Fatal(lost)
	}
}
