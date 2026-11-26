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

func TestEnduranceAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "practice.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player = "bakery/player"
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, player, simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.SaveCheckpoint(path, scope); err != nil {
		t.Fatal(err)
	}
	var mid storage.ReplayReport
	const failed = "interrupted endurance recovery"
	turn := 0
	for bout := 1; bout <= 2; bout++ {
		for _, dir := range []simulation.Direction{simulation.West, simulation.South, simulation.East, simulation.North, simulation.South, simulation.North, ""} {
			turn++
			p := simulation.Proposal{Version: 1, Scope: scope, RequestID: fmt.Sprintf("walk/%d", turn), Action: simulation.ActionMove, Args: simulation.ActionArgs{Direction: dir}}
			if dir == "" {
				p.Action = simulation.ActionRest
			}
			if turn == 14 {
				var err error
				mid, err = storage.SaveCheckpoint(path, scope)
				if err != nil {
					t.Fatal(err)
				}
				before, actors := inspectScope(t, path, scope), bakeryBytes(t, path, scope)
				execSQL(t, path, `CREATE TRIGGER fail_recovery BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL, 'interrupted endurance recovery'); END`)
				if _, _, err := storage.SubmitBakery(path, scope, player, p); err == nil || !strings.Contains(err.Error(), failed) {
					t.Fatal("expected interrupted receipt", err)
				}
				if !bytes.Equal(actors, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(before, inspectScope(t, path, scope)) {
					t.Fatal("failed recovery changed the save")
				}
				execSQL(t, path, "DROP TRIGGER fail_recovery")
			}
			r, repeated, err := storage.SubmitBakery(path, scope, player, p)
			if err != nil || repeated || r.Status != "accepted" {
				t.Fatal(r, repeated, err)
			}
			if turn == 14 {
				retry, repeated, err := storage.SubmitBakery(path, scope, player, p)
				if err != nil || !repeated || !reflect.DeepEqual(r, retry) {
					t.Fatal("lost recovery receipt", retry, err)
				}
			}
		}
	}
	head, err := storage.Replay(path, scope, 0)
	if err != nil || head.Tick != 14 {
		t.Fatal(head, err)
	}
	again, err := storage.Replay(path, scope, mid.Sequence)
	if err != nil || again != head {
		t.Fatal(again, err)
	}
	credits := 0
	for _, e := range inspectScope(t, path, scope).events {
		if e.ActorID == player && e.Payload.EndurancePractice {
			credits++
		}
	}
	if credits != 2 {
		t.Fatalf("expected two recoveries, got %d", credits)
	}
	t.Logf("replay from 0 and %d: tick %d, sequence %d, sha256 %s", mid.Sequence, head.Tick, head.Sequence, head.Hash)
}
