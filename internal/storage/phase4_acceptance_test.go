// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func newAcceptancePlayer(t *testing.T, batch string) func(path, transcript, wantFailure string) string {
	t.Helper()
	runner := filepath.Join(t.TempDir(), "sloptown.exe")
	if out, err := exec.Command("go", "build", "-mod=readonly", "-o", runner, "../../cmd/sloptown").CombinedOutput(); err != nil {
		t.Fatalf("build runner: %v: %s", err, out)
	}
	return func(path, transcript, wantFailure string) string {
		t.Helper()
		cmd := exec.Command(runner, "bakery", "-db", path, "-play", "-batch", batch)
		cmd.Stdin = strings.NewReader(transcript)
		out, err := cmd.CombinedOutput()
		if wantFailure == "" {
			if err != nil {
				t.Fatalf("walk: %v: %s", err, out)
			}
		} else {
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(string(out), wantFailure) {
				t.Fatalf("expected save failure %q, got %v: %s", wantFailure, err, out)
			}
		}
		return string(out)
	}
}

func TestPhase4Acceptance(t *testing.T) {
	dir := t.TempDir()
	play := newAcceptancePlayer(t, "walk")
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	clean, restarted := filepath.Join(dir, "clean.sqlite"), filepath.Join(dir, "restarted.sqlite")
	for _, path := range []string{clean, restarted} {
		if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, player, simulation.BakerySetup{Resident: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := storage.SaveCheckpoint(path, scope); err != nil {
			t.Fatal(err)
		}
	}
	beforeBreak := "rest\nsouth\nsouth\n" + strings.Repeat("wait\n", 8)
	atBreak := beforeBreak + "wait\n"
	walk := atBreak + "north\nnorth\n"
	play(clean, walk, "")
	play(restarted, beforeBreak, "")

	mid, err := storage.SaveCheckpoint(restarted, scope)
	if err != nil || mid.Tick != 11 || mid.Sequence != 39 {
		t.Fatalf("before the baker rests: %+v, %v", mid, err)
	}
	before, bakeryBefore := inspectScope(t, restarted, scope), bakeryBytes(t, restarted, scope)
	execSQL(t, restarted, `CREATE TRIGGER fail_break BEFORE INSERT ON request_receipt
		BEGIN SELECT RAISE(FAIL, 'interrupted break receipt'); END`)
	play(restarted, atBreak, "interrupted break receipt")
	if !reflect.DeepEqual(before, inspectScope(t, restarted, scope)) || !bytes.Equal(bakeryBefore, bakeryBytes(t, restarted, scope)) {
		t.Fatal("failed turn changed the clock, bakery, events or receipts")
	}
	if head, err := storage.Replay(restarted, scope, 0); err != nil || head != mid {
		t.Fatalf("after failed save: %+v, %v; want %+v", head, err, mid)
	}
	execSQL(t, restarted, "DROP TRIGGER fail_break")
	if out := play(restarted, atBreak, ""); strings.Count(out, "already completed") != 11 {
		t.Fatalf("expected eleven acknowledgements and one new turn: %s", out)
	}

	view := func(path, actor string) simulation.Observation {
		t.Helper()
		obs, err := storage.OpenBakery(path, scope, 42, actor)
		if err != nil {
			t.Fatal(err)
		}
		return obs
	}
	playerView := view(restarted, player)
	bakerVisible := slices.ContainsFunc(playerView.Targets, func(target simulation.ObservedTarget) bool {
		return target.Ref == baker
	})
	if bakerVisible || strings.Contains(playerView.Text, "You see the baker ") {
		t.Fatalf("player down the street could see the baker: %+v", playerView)
	}
	if obs := view(restarted, porter); !strings.Contains(obs.Text, "You see the baker sitting to the east.") {
		t.Fatalf("porter missed the baker's break: %s", obs.Text)
	}
	if obs := view(restarted, baker); !strings.Contains(obs.Text, "You are sitting down.") {
		t.Fatalf("baker's view after retry: %s", obs.Text)
	}
	play(restarted, walk, "")
	if obs := view(restarted, player); !strings.Contains(obs.Text, "You see the baker to the east.") {
		t.Fatalf("player did not see the baker on returning: %s", obs.Text)
	}
	if out := play(restarted, walk, ""); strings.Count(out, "already completed") != 14 {
		t.Fatalf("late retry did not acknowledge the fourteen actions: %s", out)
	}
	history := inspectScope(t, restarted, scope)
	if history.seq != 49 || len(history.events) != 49 || len(history.receipts) != 14 || !reflect.DeepEqual(history, inspectScope(t, clean, scope)) ||
		!bytes.Equal(bakeryBytes(t, restarted, scope), bakeryBytes(t, clean, scope)) {
		t.Fatal("uninterrupted and restarted histories differ after the return")
	}
	// Returning to the shopfront lets the player see the baker again. Only
	// the porter was there to record the baker sitting down on tick 12.
	wantNotes := map[string][]simulation.NotebookEntry{
		player: nil,
		baker:  {{Tick: 10, SourceSequence: 35, Subject: porter, Action: simulation.ActionRest}},
		porter: {
			{Tick: 1, SourceSequence: 2, Subject: player, Action: simulation.ActionRest},
			{Tick: 12, SourceSequence: 42, Subject: baker, Action: simulation.ActionRest},
		},
	}
	for actor, want := range wantNotes {
		for _, path := range []string{clean, restarted} {
			if got, err := storage.BakeryNotebook(path, scope, actor); err != nil || !slices.Equal(got, want) {
				t.Fatalf("%s notebook in %s: %v, %v; want %v", actor, path, got, err, want)
			}
		}
		if !reflect.DeepEqual(view(clean, actor), view(restarted, actor)) {
			t.Fatalf("%s sees a different scene after restart", actor)
		}
		for _, note := range want {
			event := history.events[note.SourceSequence-1]
			if event.Sequence != note.SourceSequence || event.Tick != note.Tick || event.ActorID != note.Subject || event.Kind != "actor_rested" {
				t.Fatalf("%s sighting %+v points to the wrong event: %+v", actor, note, event)
			}
		}
	}
	const digest = "17bce67289ee67188f7dde0f35e9a5b3151ce228e917b74666ebc39d219ee831"
	for _, from := range []uint64{0, mid.Sequence} {
		head, err := storage.Replay(restarted, scope, from)
		if err != nil || head.Tick != 14 || head.Sequence != 49 || head.Hash != digest {
			t.Fatalf("replay from %d: %+v, %v", from, head, err)
		}
		t.Logf("replay from %d: tick %d, sequence %d, sha256 %s", from, head.Tick, head.Sequence, head.Hash)
	}
}
