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

func TestReliefAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "breaks.sqlite")
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	if _, err := storage.OpenBakerySceneWithSetup(path, s, 42, player, simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	checkpoint := func() storage.ReplayReport {
		t.Helper()
		r, err := storage.SaveCheckpoint(path, s)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	from := []uint64{checkpoint().Sequence}
	for tick := 1; tick <= 15; tick++ {
		p := simulation.Proposal{Version: 1, Scope: s, RequestID: fmt.Sprint(tick), Action: simulation.ActionWait}
		if tick == 12 {
			before, actors := inspectScope(t, path, s), bakeryBytes(t, path, s)
			execSQL(t, path, `CREATE TRIGGER interrupt_break BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL, 'interrupted break'); END`)
			if _, _, err := storage.SubmitBakery(path, s, player, p); err == nil || !strings.Contains(err.Error(), "interrupted break") {
				t.Fatalf("expected interrupted break, got %v", err)
			}
			if !bytes.Equal(actors, bakeryBytes(t, path, s)) || !reflect.DeepEqual(before, inspectScope(t, path, s)) {
				t.Fatal("interrupted save changed state or history")
			}
			execSQL(t, path, "DROP TRIGGER interrupt_break")
		}
		r, repeated, err := storage.SubmitBakery(path, s, player, p)
		if err != nil || repeated || r.Status != "accepted" {
			t.Fatal(r, repeated, err)
		}
		if tick == 10 || tick == 12 {
			from = append(from, checkpoint().Sequence)
			actor := porter
			if tick == 12 {
				actor = baker
			}
			view, err := storage.OpenBakery(path, s, 42, actor)
			if err != nil || !strings.Contains(view.Text, "You feel relieved.") {
				t.Fatal(view, err)
			}
			if r2, repeated, err := storage.SubmitBakery(path, s, player, p); err != nil || !repeated || !reflect.DeepEqual(r2, r) {
				t.Fatal("lost original receipt", r2, err)
			}
		}
	}
	for _, actor := range []string{porter, baker} {
		view, err := storage.OpenBakery(path, s, 42, actor)
		if err != nil || strings.Contains(view.Text, "relief") || strings.Contains(view.Text, "relieved") {
			t.Fatal("relief persisted after expiry", view, err)
		}
	}
	var triggers []string
	for _, e := range inspectScope(t, path, s).events {
		if e.Payload.ReliefStarted {
			triggers = append(triggers, fmt.Sprintf("%s %s %d", e.ActorID, e.Kind, e.Tick))
		}
	}
	if !reflect.DeepEqual(triggers, []string{porter + " actor_rested 10", baker + " actor_rested 12"}) {
		t.Fatal("wrong relief sources", triggers)
	}
	head := checkpoint()
	for _, seq := range from {
		got, err := storage.Replay(path, s, seq)
		if err != nil || got != head || got.Tick != 15 {
			t.Fatalf("from %d: %+v, %v; want %+v", seq, got, err, head)
		}
	}
	t.Logf("replay from %v: tick %d, sequence %d, sha256 %s", from, head.Tick, head.Sequence, head.Hash)
}
