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

// Recreate the tick-zero actor layouts from checkpoints 1 through 3. Their
// published replay hashes below pin both the saved content and its behavior.
func legacyResidentSave(t *testing.T, path string, s simulation.Scope, version int) simulation.State {
	t.Helper()
	clock := simulation.NewTimeline(42)
	b, err := simulation.GenerateBakery(s.WorldID, clock)
	if err != nil {
		t.Fatal(err)
	}
	station := b.Entities["bakery/door"].Placement
	station.Cell.X++
	b.Entities["bakery/baker"] = simulation.SpatialEntity{Name: "baker", Kind: simulation.Actor, Placement: station}
	if version == 4 {
		street := b.Entities["bakery/player"].Placement
		street.Cell.Y = len(b.Cells) - 1
		b.Entities["bakery/porter"] = simulation.SpatialEntity{Name: "porter", Kind: simulation.Actor, Placement: street}
	}
	b.ContentVersion = version
	data, err := b.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.OpenBakery(path, s, 42, "bakery/player"); err != nil {
		t.Fatal(err)
	}
	execSQL(t, path, "UPDATE timeline_state SET bakery=? WHERE world=? AND timeline=?", data, s.WorldID, s.TimelineID)
	return simulation.State{Clock: clock, Bakery: b}
}

func TestBakerRollbackAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resident.sqlite")
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player = "bakery/player"
	legacyResidentSave(t, path, s, 2)
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

func TestBakerRestAcrossRestart(t *testing.T) {
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player = "bakery/player"
	path := filepath.Join(t.TempDir(), "rest.sqlite")
	live := legacyResidentSave(t, path, s, 3)
	var history []simulation.Event
	if _, err := storage.SaveCheckpoint(path, s); err != nil {
		t.Fatal(err)
	}
	var mid storage.ReplayReport
	for tick := 1; tick <= 13; tick++ {
		p := simulation.Proposal{Scope: s, Version: 1, RequestID: fmt.Sprint(tick), Action: simulation.ActionWait}
		if tick == 12 {
			before, entities := inspectScope(t, path, s), bakeryBytes(t, path, s)
			execSQL(t, path, `CREATE TRIGGER fail_rest BEFORE INSERT ON timeline_event
				WHEN json_extract(CAST(NEW.event AS TEXT), '$.Kind')='actor_rested'
				BEGIN SELECT RAISE(FAIL, 'injected during rest'); END`)
			failed, _, err := storage.SubmitBakery(path, s, player, p)
			if err == nil || !strings.Contains(err.Error(), "injected during rest") || failed.Status == "accepted" {
				t.Fatalf("expected interrupted rest: %+v, %v", failed, err)
			}
			if !reflect.DeepEqual(before, inspectScope(t, path, s)) || !bytes.Equal(entities, bakeryBytes(t, path, s)) {
				t.Fatal("failed rest changed the saved actors, clock, receipt or events")
			}
			execSQL(t, path, "DROP TRIGGER fail_rest")
		}
		want, step := live.Resolve(s, uint64(len(history)), player, p, nil)
		if step == nil {
			t.Fatal(want)
		}
		live = step.State
		history = append(history, step.Events...)
		got, repeated, err := storage.SubmitBakery(path, s, player, p)
		if err != nil || repeated || want.Status != "accepted" || !reflect.DeepEqual(got, want) {
			t.Fatalf("tick %d: reopened result %+v, uninterrupted %+v, %v", tick, got, want, err)
		}
		if tick == 10 {
			mid, err = storage.SaveCheckpoint(path, s)
			if err != nil || mid.Tick != 10 || mid.Sequence != 27 {
				t.Fatalf("checkpoint on the way to rest: %+v, %v", mid, err)
			}
		}
		if tick == 12 {
			view, err := storage.OpenBakery(path, s, 99, player)
			if err != nil || !reflect.DeepEqual(view, want.Observation) || !strings.Contains(view.Text, "the baker sitting") {
				t.Fatalf("reopened during rest: %+v, %v", view, err)
			}
			if got, repeated, err := storage.SubmitBakery(path, s, player, p); err != nil || !repeated || !reflect.DeepEqual(got, want) {
				t.Fatalf("retry after rest: %+v, %v", got, err)
			}
		}
	}
	if !reflect.DeepEqual(inspectScope(t, path, s).events, history) {
		t.Fatal("reopened history differs from uninterrupted work, break and return")
	}
	const want = "8e2b1876a8e108382a11b3319e77e24f01d54220ba15932e9ab74f3f497ce535"
	for _, from := range []uint64{0, mid.Sequence} {
		head, err := storage.Replay(path, s, from)
		if err != nil || head.Tick != 13 || head.Sequence != 36 || head.Hash != want {
			t.Fatalf("replay from %d: %+v, %v", from, head, err)
		}
	}
}
