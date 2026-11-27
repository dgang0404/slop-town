// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestDropHeardFromTheStreetAcrossRestart(t *testing.T) {
	const player, porter = "bakery/player", "bakery/porter"
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "corner.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, s, 42, player, simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "corner")
	start := checkpointAtHead(t, path, s)
	const carry = "wait\ne\ne\nopen crate\ntake spoon\nw\nw\ns\n"
	const drop = carry + "drop it\n"
	play(path, carry, "")
	before := checkpointAtHead(t, path, s)
	town, history := bakeryBytes(t, path, s), inspectScope(t, path, s)
	execSQL(t, path, "CREATE TRIGGER fail_impact BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted floor impact'); END")
	play(path, drop, "interrupted floor impact")
	if !bytes.Equal(town, bakeryBytes(t, path, s)) || !reflect.DeepEqual(history, inspectScope(t, path, s)) {
		t.Fatal("interrupted drop changed the item, memories, clock or history")
	}
	execSQL(t, path, "DROP TRIGGER fail_impact")
	if out := play(path, drop, ""); !strings.Contains(out, "You drop the wooden spoon. You hear a clatter.") {
		t.Fatal(out)
	}
	const visit = drop + "walk to porter\ntalk to porter\n8\ncancel\njournal accounts\n"
	out := play(path, visit, "")
	_, accounts, ok := strings.Cut(out, "Accounts I've heard\n")
	if !ok || !strings.Contains(out, "The porter replies, “I heard a clatter.”") ||
		strings.Count(accounts, "The porter told me they heard a clatter.") != 1 || strings.Contains(accounts, "spoon") {
		t.Fatal(out)
	}
	notes, err := storage.BakeryNotebook(path, s, porter)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, note := range notes {
		if note.Action == simulation.ActionDrop {
			t.Fatal("the porter saw the hidden drop", note)
		}
		if note.Sound == simulation.Clatter {
			count++
			if note.Tick != 9 || note.Subject != "" || note.Target != "" {
				t.Fatal(note)
			}
		}
	}
	if count != 1 {
		t.Fatalf("porter recorded %d impacts", count)
	}
	head := checkpointAtHead(t, path, s)
	retry := play(path, visit, "")
	_, reopened, _ := strings.Cut(retry, "Accounts I've heard\n")
	if reopened != accounts || strings.Count(retry, "already completed") != 11 {
		t.Fatal(retry)
	}
	for _, from := range []uint64{start.Sequence, before.Sequence} {
		if replay, err := storage.Replay(path, s, from); err != nil || replay != head || replay.Tick != 11 {
			t.Fatal(replay, err)
		}
	}
	t.Logf("replay from %d and %d reached tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, head.Tick, head.Sequence, head.Hash)
}
