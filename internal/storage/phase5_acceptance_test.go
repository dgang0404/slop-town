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

func TestPhase5Acceptance(t *testing.T) {
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	dir := t.TempDir()
	clean, interrupted := filepath.Join(dir, "clean.sqlite"), filepath.Join(dir, "interrupted.sqlite")
	play := newAcceptancePlayer(t, "visit")
	for _, path := range []string{clean, interrupted} {
		if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, player, simulation.BakerySetup{Practice: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := storage.SaveCheckpoint(path, scope); err != nil {
			t.Fatal(err)
		}
	}

	const street = "walk to the porter\nintroduce yourself to the porter\nwest\nnorth\neast\nsouth\nnorth\nrest\n"
	const atCrate = street + "east\neast\neast\nsouth\nsouth\nwalk to the crate\n"
	const recovered = atCrate + "rest\n"
	const visit = recovered + "south\nsouth\nwest\nnorth\nnorth\neast\n"
	play(clean, visit, "")
	if out := play(interrupted, atCrate, ""); strings.Count(out, "You walk toward the crate.") != 1 || !strings.Contains(out, "You are within reach of the crate.") {
		t.Fatal(out)
	}

	mid, err := storage.SaveCheckpoint(interrupted, scope)
	if err != nil || mid.Tick != 16 || mid.Sequence != 54 {
		t.Fatalf("before the second rest: %+v, %v", mid, err)
	}
	before, actors := inspectScope(t, interrupted, scope), bakeryBytes(t, interrupted, scope)
	execSQL(t, interrupted, `CREATE TRIGGER fail_recovery BEFORE INSERT ON request_receipt
		BEGIN SELECT RAISE(FAIL, 'interrupted second rest'); END`)
	play(interrupted, recovered, "interrupted second rest")
	if !reflect.DeepEqual(before, inspectScope(t, interrupted, scope)) || !bytes.Equal(actors, bakeryBytes(t, interrupted, scope)) {
		t.Fatal("failed recovery changed the saved actors, notebooks or ledger")
	}
	if head, err := storage.Replay(interrupted, scope, 0); err != nil || head != mid {
		t.Fatalf("after failed recovery: %+v, %v; want %+v", head, err, mid)
	}
	execSQL(t, interrupted, "DROP TRIGGER fail_recovery")

	if out := play(interrupted, recovered, ""); strings.Count(out, "already completed") != 14 || !strings.Contains(out, "You feel relieved.") {
		t.Fatal(out)
	}
	afterRest, err := storage.SaveCheckpoint(interrupted, scope)
	if err != nil || afterRest.Tick != 17 || afterRest.Sequence != 58 {
		t.Fatalf("after successful recovery: %+v, %v", afterRest, err)
	}
	play(interrupted, visit, "")
	if out := play(interrupted, visit, ""); strings.Count(out, "already completed") != 21 {
		t.Fatal(out)
	}

	history := inspectScope(t, interrupted, scope)
	if len(history.events) != 80 || len(history.receipts) != 23 || !reflect.DeepEqual(history, inspectScope(t, clean, scope)) || !bytes.Equal(bakeryBytes(t, interrupted, scope), bakeryBytes(t, clean, scope)) {
		t.Fatal("uninterrupted and restarted saves diverged")
	}
	var recoveries []uint64
	for _, event := range history.events {
		if event.ActorID == player && event.Kind == "actor_rested" && event.Payload.EndurancePractice && event.Payload.ReliefStarted {
			recoveries = append(recoveries, event.Sequence)
		}
	}
	if !slices.Equal(recoveries, []uint64{26, 56}) {
		t.Fatalf("player recovery events: %v", recoveries)
	}

	// Compare what the player actually saw before and after training.
	views := make(map[string]string)
	for _, receipt := range history.receipts {
		views[receipt.Result.RequestID] = receipt.Result.Observation.Text
	}
	for _, tc := range []struct {
		turn, phrase string
		want         bool
	}{
		{"visit/5", "Your legs feel tired.", false}, {"visit/6", "Your legs feel tired.", true},
		{"visit/17", "You feel relieved.", true}, {"visit/18", "A little of that relief lingers.", true},
		{"visit/19", "A little of that relief lingers.", true},
		{"visit/20", "A little of that relief lingers.", false}, {"visit/20", "You feel relieved.", false},
		{"visit/22", "Your legs feel tired.", false}, {"visit/23", "Your legs feel tired.", true},
	} {
		view, ok := views[tc.turn]
		if !ok || strings.Contains(view, tc.phrase) != tc.want {
			t.Fatalf("%s: expected %q present=%t in %q", tc.turn, tc.phrase, tc.want, view)
		}
	}

	// All three are in sight at the end. Each recognition must trace back
	// to an introduction received by this actor alone.
	for _, tc := range []struct {
		actor, speaker string
		source         uint64
	}{
		{player, "", 0}, {baker, porter, 12}, {porter, player, 6},
	} {
		view, err := storage.OpenBakery(interrupted, scope, 42, tc.actor)
		if err != nil {
			t.Fatal(err)
		}
		control, err := storage.OpenBakery(clean, scope, 42, tc.actor)
		if err != nil || !reflect.DeepEqual(view, control) {
			t.Fatalf("%s sees a different scene: %v", tc.actor, err)
		}
		notes, err := storage.BakeryNotebook(interrupted, scope, tc.actor)
		if err != nil {
			t.Fatal(err)
		}
		var intros []simulation.NotebookEntry
		restSightings := 0
		for _, note := range notes {
			if note.Action == simulation.ActionIntroduce {
				intros = append(intros, note)
			}
			if note.Action == simulation.ActionRest && note.Subject == player && note.Tick == 17 && note.SourceSequence == 56 {
				restSightings++
			}
		}
		wantSightings := 1
		if tc.actor == player {
			wantSightings = 0
		}
		if restSightings != wantSightings {
			t.Fatalf("%s recorded %d sightings of the second rest, want %d", tc.actor, restSightings, wantSightings)
		}
		if tc.speaker == "" {
			if len(intros) != 0 || strings.Contains(view.Text, "You recognize") {
				t.Fatal("player learned a bystander's introduction", notes, view.Text)
			}
			continue
		}
		event := history.events[tc.source-1]
		if event.Sequence != tc.source || event.Kind != "actor_introduced" || event.ActorID != tc.speaker || event.Payload.Target != tc.actor {
			t.Fatalf("wrong introduction source for %s: %+v", tc.actor, event)
		}
		wantNote := simulation.NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence, Action: simulation.ActionIntroduce, Subject: tc.speaker, Target: tc.actor}
		if !slices.Equal(intros, []simulation.NotebookEntry{wantNote}) {
			t.Fatalf("%s has introductions %+v, want %+v", tc.actor, intros, wantNote)
		}
		recognition := "You recognize the " + strings.TrimPrefix(tc.speaker, "bakery/") + " from their introduction."
		if strings.Count(view.Text, "You recognize") != 1 || !strings.Contains(view.Text, recognition) {
			t.Fatalf("%s: expected %q in %q", tc.actor, recognition, view.Text)
		}
	}
	for _, from := range []uint64{0, mid.Sequence, afterRest.Sequence} {
		head, err := storage.Replay(interrupted, scope, from)
		if err != nil || head.Tick != 23 || head.Sequence != 80 || head.Hash != "8f7f7fbee48b621d5d33d54cf4cbb4353a8ee88c89f2ef1c131e89c62302f1a4" {
			t.Fatalf("from %d: %+v, %v", from, head, err)
		}
		t.Logf("from %d: tick %d, sequence %d, sha256 %s", from, head.Tick, head.Sequence, head.Hash)
	}
}
