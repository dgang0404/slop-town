// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestDiscreetTakeSurvivesRollbackAndRetelling(t *testing.T) {
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	dir := t.TempDir()
	path, clean := filepath.Join(dir, "retry.sqlite"), filepath.Join(dir, "clean.sqlite")
	play := newAcceptancePlayer(t, "discreet")
	const prelude = "wait\ne\ne\nopen crate\nwait\n"
	const pickup = prelude + "discreetly take spoon\n"
	for _, db := range []string{path, clean} {
		if _, err := storage.OpenBakerySceneWithSetup(db, scope, 4, player, simulation.BakerySetup{Practice: true}); err != nil {
			t.Fatal(err)
		}
	}
	start := checkpointAtHead(t, path, scope)
	play(path, prelude, "")
	before := checkpointAtHead(t, path, scope)
	town, history := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
	if out := play(path, "look\nactions\njournal\n", ""); !strings.Contains(out, "discreetly take the wooden spoon") {
		t.Fatal(out)
	}
	execSQL(t, path, "CREATE TRIGGER fail_pickup BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted discreet pickup'); END")
	play(path, pickup, "interrupted discreet pickup")
	if !bytes.Equal(town, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(history, inspectScope(t, path, scope)) {
		t.Fatal("read or failed pickup changed the clock, item, notebooks or receipts")
	}
	execSQL(t, path, "DROP TRIGGER fail_pickup")
	play(path, pickup, "")
	play(clean, pickup, "")
	if retry := play(path, prelude+"quietly take the spoon\n", ""); strings.Count(retry, "already completed") != 6 {
		t.Fatal(retry)
	}
	if !bytes.Equal(bakeryBytes(t, clean, scope), bakeryBytes(t, path, scope)) || !reflect.DeepEqual(inspectScope(t, clean, scope), inspectScope(t, path, scope)) {
		t.Fatal("retry differs from the uninterrupted visit")
	}
	takeSequence := before.Sequence + 2
	for owner, want := range map[string]simulation.NotebookEntry{
		baker:  {Tick: 6, SourceSequence: takeSequence, Action: simulation.SightHandMovement, Container: "bakery/crate"},
		porter: {Tick: 6, SourceSequence: takeSequence, Action: simulation.ActionTake, Subject: player, Target: "bakery/spoon"},
	} {
		entries, err := storage.BakeryNotebook(path, scope, owner)
		if err != nil || !slices.Contains(entries, want) {
			t.Fatal(owner, entries, err)
		}
	}
	// Ask immediately on the clean timeline. On the other, let the baker
	// tell the porter first, then follow that specific account to its source.
	direct := play(clean, pickup+"talk to baker\n3\ncancel\ntalk to porter\n3\ncancel\njournal accounts\n", "")
	_, accounts, _ := strings.Cut(direct, "Accounts I've heard\n")
	if !strings.Contains(accounts, "The baker told me they saw a hand move near the crate.") ||
		!strings.Contains(accounts, "The porter told me they saw me take the wooden spoon.") {
		t.Fatal(accounts)
	}
	visit := pickup + strings.Repeat("wait\n", 9) + "talk to porter\n7\ncancel\njournal leads\ntalk to baker\n4\ncancel\njournal accounts\njournal leads\n"
	out := play(path, visit, "")
	_, accounts, _ = strings.Cut(out, "Accounts I've heard\n")
	if !strings.Contains(out, "Did the baker see a hand move near the crate, as the porter said?") ||
		!strings.Contains(accounts, "The porter said the baker saw a hand move near the crate.") ||
		!strings.Contains(accounts, "The baker told me they saw a hand move near the crate.") ||
		strings.Contains(accounts, "spoon") || !strings.HasSuffix(accounts, "I have no accounts to follow up.\n") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	for _, from := range []uint64{start.Sequence, before.Sequence} {
		if r, err := storage.Replay(path, scope, from); err != nil || r != head || r.Tick != 17 {
			t.Fatal(r, err)
		}
	}
	t.Logf("replayed from %d and %d to tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, head.Tick, head.Sequence, head.Hash)
}
