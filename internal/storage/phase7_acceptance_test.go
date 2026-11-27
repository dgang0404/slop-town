// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestPhase7Acceptance(t *testing.T) {
	play := newAcceptancePlayer(t, "visit")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const opening = "rest\neast\neast\nsouth\n"
	const story = "north\ntalk to baker\n5\n4\ncancel\ntalk to porter\n4\ncancel\nopen crate\n"
	const read = "journal accounts\njournal\nexport journal\nexport illustrated journal\n"
	for _, visit := range []struct {
		name, ending string
		followed     bool
	}{
		{"asked", story, true},
		{"missed", strings.Repeat("wait\n", 5), false},
	} {
		t.Run(visit.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			db := filepath.Join(dir, "visit.sqlite")
			if _, err := storage.OpenBakerySceneWithSetup(db, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
				t.Fatal(err)
			}
			start := checkpointAtHead(t, db, scope)
			play(db, opening, "")
			corner := checkpointAtHead(t, db, scope)
			if corner.Tick != 4 || corner.Sequence != 15 {
				t.Fatal(corner)
			}
			out := play(db, opening+visit.ending, "")
			if strings.Count(out, "already completed") != 4 {
				t.Fatal(out)
			}
			head := checkpointAtHead(t, db, scope)
			if head.Tick != 9 {
				t.Fatal(head)
			}
			out = play(db, read, "")
			_, notes, ok := strings.Cut(out, "Accounts I've heard\n")
			notes, onScreen, journalFound := strings.Cut(notes, "My journal\n")
			onScreen, _, exported := strings.Cut(onScreen, "Saved your journal to ")
			book, err := os.ReadFile("slop-town-journal.txt")
			if err != nil || !ok || !journalFound || !exported || string(book) != "My journal\n"+onScreen {
				t.Fatalf("on-screen journal and export differ: %v", err)
			}
			illustrated, err := os.ReadFile("slop-town-journal.html")
			if err != nil {
				t.Fatal(err)
			}
			page := html.UnescapeString(string(illustrated))
			for _, line := range strings.Split(string(book), "\n") {
				if line != "" && !strings.HasPrefix(line, "Chapter ") && line != "My journal" && !strings.Contains(page, "<p>"+line+"</p>") {
					t.Fatalf("missing illustrated passage %q", line)
				}
			}
			for _, edition := range []string{string(book), page} {
				if !visit.followed && (strings.Contains(edition, "wooden spoon") || strings.Contains(edition, "told me") || strings.Contains(edition, "I saw you sit down")) {
					t.Fatal("an undiscovered object or unheard claim reached the journal")
				}
				if strings.Contains(edition, "I moved east.") || strings.Contains(edition, "I was in the bakery.") {
					t.Fatal("the journal repeated the steps into the bakery")
				}
				for _, line := range []string{
					"The baker replied, “The porter told me they saw you sit down.”",
					"The porter replied, “I saw you sit down.”",
					"I opened the crate. I saw the wooden spoon in the crate.",
				} {
					if strings.Contains(edition, line) != visit.followed {
						t.Fatalf("wrong evidence for %q", line)
					}
				}
			}
			if strings.Contains(page, "Ink sketch of the wooden spoon.") != visit.followed {
				t.Fatal("spoon illustration did not match discovery")
			}
			wantNotes := "I haven't heard any accounts yet.\n"
			if visit.followed {
				wantNotes = "\nDay 1 of spring, year 1 — The baker said the porter saw me sit down.\n" +
					"Day 1 of spring, year 1 — The porter told me they saw me sit down.\n"
			}
			if notes != wantNotes {
				t.Fatalf("accounts: %q", notes)
			}
			for _, from := range []uint64{start.Sequence, corner.Sequence} {
				if replay, err := storage.Replay(db, scope, from); err != nil || replay != head {
					t.Fatal(replay, err)
				}
			}
			t.Logf("reopen and replay reached tick %d, sequence %d, sha256 %s", head.Tick, head.Sequence, head.Hash)
		})
	}
}
