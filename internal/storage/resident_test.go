// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestBakerRollbackAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resident.sqlite")
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player = "bakery/player"
	_, err := storage.OpenBakerySceneWithSetup(path, s, 42, player, simulation.BakerySetup{Resident: true})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := storage.SaveCheckpoint(path, s)
	if err != nil {
		t.Fatal(err)
	}
	before, entities := inspectScope(t, path, s), bakeryBytes(t, path, s)
	wait := simulation.Proposal{Version: 1, Scope: s, RequestID: "1", Action: simulation.ActionWait}
	execSQL(t, path, `CREATE TRIGGER fail_baker BEFORE INSERT ON timeline_event WHEN NEW.seq=3 BEGIN
		SELECT RAISE(FAIL, 'injected at baker action'); END`)
	r, _, err := storage.SubmitBakery(path, s, player, wait)
	if err == nil || !strings.Contains(err.Error(), "injected at baker action") || r.Status == "accepted" {
		t.Fatalf("expected unpublished baker action, got %+v, %v", r, err)
	}
	if !reflect.DeepEqual(before, inspectScope(t, path, s)) || !bytes.Equal(entities, bakeryBytes(t, path, s)) {
		t.Fatal("part of the player or baker turn survived rollback")
	}
	if head, err := storage.Replay(path, s, 0); err != nil || head != initial {
		t.Fatalf("rollback changed replay: %+v, %v", head, err)
	}
	execSQL(t, path, "DROP TRIGGER fail_baker")
	var first simulation.Result
	for tick := 1; tick <= 10; tick++ {
		wait.RequestID = fmt.Sprint(tick)
		r, repeated, err := storage.SubmitBakery(path, s, player, wait)
		if err != nil || repeated || r.Status != "accepted" {
			t.Fatalf("wait %d: %+v, repeated %v, %v", tick, r, repeated, err)
		}
		if tick == 1 {
			first = r
			mid, err := storage.SaveCheckpoint(path, s)
			if err != nil || mid.Sequence != 3 || mid.Tick != 1 {
				t.Fatalf("checkpoint after opening: %+v, %v", mid, err)
			}
		}
	}
	for _, from := range []uint64{0, 3} {
		head, err := storage.Replay(path, s, from)
		const want = "07d0038bf8e137f1431b101ed11ff48519186a2b315b23aabd8de22ad64d591c"
		if err != nil || head.Tick != 10 || head.Sequence != 27 || head.Hash != want {
			t.Fatalf("replay from %d: %+v, %v", from, head, err)
		}
		t.Logf("replay from %d: tick %d, sequence %d, sha256 %s", from, head.Tick, head.Sequence, head.Hash)
	}
	final := inspectScope(t, path, s)
	wait.RequestID = "1"
	r, repeated, err := storage.SubmitBakery(path, s, player, wait)
	if err != nil || !repeated || !reflect.DeepEqual(first, r) || !reflect.DeepEqual(final, inspectScope(t, path, s)) {
		t.Fatalf("late retry lost its receipt or advanced the baker: %+v, %v", r, err)
	}
}
