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

func TestPersonalityChoicesAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "break.sqlite")
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player = "bakery/player"
	if _, err := storage.OpenBakerySceneWithSetup(path, s, 42, player, simulation.BakerySetup{Resident: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.SaveCheckpoint(path, s); err != nil {
		t.Fatal(err)
	}
	for tick := 1; tick <= 13; tick++ {
		p := simulation.Proposal{Scope: s, Version: 1, RequestID: fmt.Sprint(tick), Action: simulation.ActionWait}
		if tick == 10 {
			before, bakery := inspectScope(t, path, s), bakeryBytes(t, path, s)
			execSQL(t, path, `CREATE TRIGGER interrupt_choice BEFORE INSERT ON timeline_event WHEN NEW.seq=34
				BEGIN SELECT RAISE(FAIL, 'interrupted second decision'); END`)
			r, _, err := storage.SubmitBakery(path, s, player, p)
			if err == nil || !strings.Contains(err.Error(), "interrupted second decision") || r.Status == "accepted" {
				t.Fatal("expected the second decision to fail", r, err)
			}
			if !reflect.DeepEqual(before, inspectScope(t, path, s)) || !bytes.Equal(bakery, bakeryBytes(t, path, s)) {
				t.Fatal("failed turn changed the workers, clock, random streams or history")
			}
			execSQL(t, path, "DROP TRIGGER interrupt_choice")
		}
		r, repeated, err := storage.SubmitBakery(path, s, player, p)
		if err != nil || repeated || r.Status != "accepted" {
			t.Fatal(r, err)
		}
		if tick == 10 {
			if !strings.Contains(r.Observation.Text, "You see the porter sitting to the east.") || strings.Contains(r.Observation.Text, "%") {
				t.Fatal(r.Observation.Text)
			}
			if retry, repeated, err := storage.SubmitBakery(path, s, player, p); err != nil || !repeated || !reflect.DeepEqual(r, retry) {
				t.Fatal("retry lost original turn", retry, err)
			}
		}
		if tick == 9 || tick == 10 {
			if _, err := storage.SaveCheckpoint(path, s); err != nil {
				t.Fatal(err)
			}
		}
	}
	for role, want := range map[string]int{"baker": 40, "porter": 87} {
		if got, err := storage.BakeryRestPreference(path, s, "bakery/"+role); err != nil || got != want {
			t.Fatalf("%s after reopen: got %d, %v", role, got, err)
		}
	}
	for _, from := range []uint64{0, 29, 35} {
		head, err := storage.Replay(path, s, from)
		if err != nil || head.Tick != 13 || head.Sequence != 46 || head.Hash != "2b082bf8382a5596334b829fb3f7eded21338139322d520041cf80e0a9ec7bdf" {
			t.Fatalf("replay from %d: %+v, %v", from, head, err)
		}
	}
}
