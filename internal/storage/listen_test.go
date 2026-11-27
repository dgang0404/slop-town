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

func TestListeningAcrossRollbackAndReplay(t *testing.T) {
	const player = "bakery/player"
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	dir := t.TempDir()
	path, clean := filepath.Join(dir, "retry.sqlite"), filepath.Join(dir, "clean.sqlite")
	for _, db := range []string{path, clean} {
		if _, err := storage.OpenBakerySceneWithSetup(db, s, 4, player, simulation.BakerySetup{Practice: true}); err != nil {
			t.Fatal(err)
		}
	}
	play := newAcceptancePlayer(t, "listen")
	start := checkpointAtHead(t, path, s)
	const walk = "south\nsouth\nwait\nwait\nwait\n"
	const listen = walk + "listen\n"
	const visit = listen + "listen\njournal\n"
	play(clean, visit, "")
	play(path, walk, "")
	before := checkpointAtHead(t, path, s)
	town, history := bakeryBytes(t, path, s), inspectScope(t, path, s)
	if out := play(path, "look\nactions\nlisten baker\njournal\n", ""); !strings.Contains(out, "  listen\n") || !strings.Contains(out, "Type listen to spend one turn here.") {
		t.Fatal(out)
	}
	execSQL(t, path, "CREATE TRIGGER fail_listen BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted listen'); END")
	play(path, listen, "interrupted listen")
	if !reflect.DeepEqual(history, inspectScope(t, path, s)) || !bytes.Equal(town, bakeryBytes(t, path, s)) {
		t.Fatal("browsing or the interrupted write changed the history, clock or town")
	}
	execSQL(t, path, "DROP TRIGGER fail_listen")
	out := play(path, visit, "")
	live, journal, ok := strings.Cut(out, "My journal\n")
	if !ok || strings.Count(live, "You hear footsteps.") != 1 || !strings.Contains(live, "You listen for a moment.") ||
		strings.Count(journal, "I heard footsteps.") != 1 || strings.Contains(journal, "listened for a moment") {
		t.Fatal(out)
	}
	r, err := storage.BakeryReceipt(path, s, player, "listen/6")
	if err != nil || r == nil || !slices.Equal(r.Result.Observation.Heard, []simulation.PerceptionKind{simulation.Footsteps}) {
		t.Fatal(r, err)
	}
	for _, target := range r.Result.Observation.Targets {
		if target.Ref == "bakery/baker" || target.Ref == "bakery/porter" {
			t.Fatal("hearing an unseen walker revealed them", target)
		}
	}
	head := checkpointAtHead(t, path, s)
	again := play(path, visit, "")
	_, reopened, _ := strings.Cut(again, "My journal\n")
	if reopened != journal || strings.Count(again, "already completed") != 7 ||
		!reflect.DeepEqual(inspectScope(t, clean, s), inspectScope(t, path, s)) || !bytes.Equal(bakeryBytes(t, clean, s), bakeryBytes(t, path, s)) {
		t.Fatal("uninterrupted and retried visits differ")
	}
	for _, from := range []uint64{start.Sequence, before.Sequence} {
		if replay, err := storage.Replay(path, s, from); err != nil || replay != head || replay.Tick != 7 {
			t.Fatal(replay, err)
		}
	}
	t.Logf("replay from %d and %d reached tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, head.Tick, head.Sequence, head.Hash)
}
