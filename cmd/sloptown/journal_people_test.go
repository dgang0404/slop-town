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

func TestPeopleBookAcrossIntroductionsAndRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "people.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	play := func(batch, input string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := runBakery([]string{"-practice", "-play", "-db", path, "-seed", "42", "-batch", batch}, strings.NewReader(input), &out, &diag); code != 0 {
			t.Fatal(diag.String())
		}
		return out.String()
	}
	book := func(batch, input string) string {
		t.Helper()
		out := play(batch, input)
		_, people, ok := strings.Cut(out, "People I’ve met\n")
		if !ok {
			t.Fatal(out)
		}
		return people
	}
	if got := book("look", "examine porter\nexamine porter\ncontacts\n"); got != "I haven’t exchanged introductions with anyone yet.\n" {
		t.Fatal("sighting counted as a meeting", got)
	}
	got := book("meet", "introduce yourself to porter\njournal people porter\n")
	if !strings.Contains(got, "The porter\nI introduced myself on day 1") || !strings.Contains(got, "I noticed one sleeve frayed at the seam.") {
		t.Fatal(got)
	}
	all := book("return", "wait\ncontacts\n")
	if !strings.Contains(all, "The baker\nThey introduced themselves to me") || strings.Count(all, "The porter\n") != 1 || strings.Contains(all, "flour caught") {
		t.Fatal("wrong meetings or unseen detail", all)
	}
	head, err := storage.SaveCheckpoint(path, scope)
	if err != nil {
		t.Fatal(err)
	}
	if got := book("read", "journal people\n"); got != all {
		t.Fatal("reopening changed the book", got)
	}
	if got := book("read", "journal people stranger\n"); got != "I haven’t met anyone by that name.\n" {
		t.Fatal(got)
	}
	if after, err := storage.Replay(path, scope, head.Sequence); err != nil || after != head {
		t.Fatal("reading advanced the town", after, err)
	}
	away := book("down-street", "south\nsouth\njournal people porter\n")
	if !strings.Contains(away, "I last saw the porter in the bakery doorway.") ||
		book("later", "wait\nwait\nwait\njournal people porter\n") != away {
		t.Fatal("the book followed the porter out of sight", away)
	}
}

func TestPeopleBookKeepsNamesakesAndQuotesSeparate(t *testing.T) {
	a := simulation.ObservedTarget{Ref: "town/1", Name: "porter"}
	b := simulation.ObservedTarget{Ref: "town/2", Name: "porter"}
	entries := []simulation.JournalEntry{
		{IntroducedTo: a, ExaminedRef: b.Ref, ExaminedDetail: "a patched coat"},
		{Witnessed: []simulation.WitnessedAction{{Action: simulation.ActionIntroduce, Actor: b}}},
		{Conversation: simulation.JournalSpeech{WithRef: a.Ref, IStarted: true, Words: "What happened?", Reply: "I heard footsteps."}},
		{Conversation: simulation.JournalSpeech{WithRef: b.Ref, Words: "I saw a crate."}},
		{IntroducedTo: a},
	}
	var buf bytes.Buffer
	if err := writePeople(&buf, entries, "porter"); err != nil {
		t.Fatal(err)
	}
	rows := strings.Split(buf.String(), "The porter\n")
	if len(rows) != 3 || !strings.Contains(rows[1], "I heard footsteps.") || strings.Contains(rows[1], "patched coat") ||
		!strings.Contains(rows[2], "I saw a crate.") || !strings.Contains(rows[2], "patched coat") || strings.Contains(buf.String(), "What happened?") {
		t.Fatal(buf.String())
	}
}
