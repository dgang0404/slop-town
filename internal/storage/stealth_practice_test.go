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

func TestStealthPracticeAcrossRestart(t *testing.T) {
	const player = "bakery/player"
	const opening = "wait\ne\ne\nopen crate\nwait\ndiscreetly take spoon\n"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	play := newAcceptancePlayer(t, "practice")
	for _, tc := range []struct {
		placement string
		credits   []string
		account   string
	}{
		{"drop spoon", []string{"container", "floor"}, "The porter told me they saw a hand move near the crate."},
		{"put spoon in crate", []string{"container"}, "The porter told me they saw me take the wooden spoon."},
	} {
		t.Run(tc.placement, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "spoon.sqlite")
			if _, err := storage.OpenBakerySceneWithSetup(path, scope, 51, player, simulation.BakerySetup{Practice: true}); err != nil {
				t.Fatal(err)
			}
			start := checkpointAtHead(t, path, scope)
			beforePickup := opening + tc.placement + "\n"
			play(path, beforePickup, "")
			before := checkpointAtHead(t, path, scope)
			pickup := beforePickup + "discreetly take spoon\n"
			if tc.placement == "drop spoon" {
				state, history := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
				play(path, "look\nactions\njournal\n", "")
				execSQL(t, path, "CREATE TRIGGER fail_practice BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted practice'); END")
				play(path, pickup, "interrupted practice")
				if !bytes.Equal(state, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(history, inspectScope(t, path, scope)) {
					t.Fatal("interrupted pickup changed the item, practice, draws, memories or history")
				}
				execSQL(t, path, "DROP TRIGGER fail_practice")
			}
			play(path, pickup, "")
			after := checkpointAtHead(t, path, scope)
			visit := pickup + "put spoon in crate\ndiscreetly take spoon\ntalk to porter\n3\njournal accounts\n"
			out := play(path, visit, "")
			_, account, ok := strings.Cut(out, "Accounts I've heard\n")
			if !ok || !strings.HasSuffix(account, tc.account+"\n") {
				t.Fatal(out)
			}
			head := checkpointAtHead(t, path, scope)
			retry := play(path, visit, "")
			_, reopened, _ := strings.Cut(retry, "Accounts I've heard\n")
			if reopened != account || strings.Count(retry, "already completed") != 11 {
				t.Fatal(retry)
			}
			var credits []string
			for _, event := range inspectScope(t, path, scope).events {
				if practice := event.Payload.StealthPractice; practice != "" {
					if event.Kind != "item_taken_discreetly" || event.ActorID != player || event.Payload.Target != "bakery/spoon" {
						t.Fatal(event)
					}
					credits = append(credits, practice)
				}
			}
			if !slices.Equal(tc.credits, credits) {
				t.Fatalf("got practice %v, want %v", credits, tc.credits)
			}
			for _, from := range []uint64{start.Sequence, before.Sequence, after.Sequence} {
				if replay, err := storage.Replay(path, scope, from); err != nil || replay != head || replay.Tick != 11 {
					t.Fatal(replay, err)
				}
			}
			t.Logf("replay from %d, %d and %d reached tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, after.Sequence, head.Tick, head.Sequence, head.Hash)
		})
	}
}
