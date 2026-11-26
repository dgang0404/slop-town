// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func bakeryBytes(t *testing.T, path string, scope simulation.Scope) []byte {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var data []byte
	if err := db.QueryRow("SELECT bakery FROM timeline_state WHERE world=? AND timeline=?", scope.WorldID, scope.TimelineID).Scan(&data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPhase2Acceptance(t *testing.T) {
	dir := t.TempDir()
	path, runner := filepath.Join(dir, "town.sqlite"), filepath.Join(dir, "sloptown.exe")
	if out, err := exec.Command("go", "build", "-mod=readonly", "-o", runner, "../../cmd/sloptown").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	run := func(want int, args ...string) string {
		t.Helper()
		cmd := exec.Command(runner, append([]string{args[0], "-db", path}, args[1:]...)...)
		out, err := cmd.CombinedOutput()
		if cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != want {
			t.Fatalf("%v: %v: %s", args, err, out)
		}
		return string(out)
	}
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const actor = "bakery/player"
	live := simulation.State{Clock: simulation.NewTimeline(42)}
	var err error
	live.Bakery, err = simulation.GenerateBakery(s.WorldID, live.Clock)
	if err != nil {
		t.Fatal(err)
	}
	run(0, "bakery", "-seed", "42")
	run(0, "checkpoint", "-world", "bakery")
	var events []simulation.Event
	var receipts []simulation.Receipt
	check := func() {
		t.Helper()
		disk := inspectScope(t, path, s)
		wantClock, err := live.Clock.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		wantBakery, err := live.Bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bakeryBytes(t, path, s), wantBakery) || disk.clock != string(wantClock) || disk.seq != int64(len(events)) ||
			!slices.Equal(disk.events, events) || !reflect.DeepEqual(disk.receipts, receipts) {
			t.Fatalf("reopened state at tick %d differs from uninterrupted execution", live.Clock.Tick())
		}
	}
	const mid = "c35a6eb2830ac7c08e171b564543528af3952f11cef5102eab379aca54285812"
	const final = "9f421909302942c47ba2ac7c46c627591ef95360749cff544ce23d206bafe58b"
	report := func(tick int64, hash string, args ...string) {
		t.Helper()
		out := run(0, args...)
		var got storage.ReplayReport
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		if got.Scope != s || got.Tick != tick || got.Sequence != uint64(tick*2) || got.Hash != hash {
			t.Fatalf("unexpected %v: %s", args, out)
		}
		t.Logf("%s: tick %d, sequence %d, sha256 %s", args[0], tick, got.Sequence, got.Hash)
	}
	for i, step := range []struct {
		action simulation.ActionKind
		arg    string
	}{
		{simulation.ActionOpen, "bakery door"}, {simulation.ActionMove, "east"}, {simulation.ActionMove, "east"}, {simulation.ActionOpen, "crate"}, {simulation.ActionTake, "wooden spoon"},
		{simulation.ActionClose, "crate"}, {simulation.ActionMove, "west"}, {simulation.ActionMove, "west"}, {simulation.ActionClose, "bakery door"}, {simulation.ActionDrop, "wooden spoon"},
	} {
		tick := i + 1
		batch := fmt.Sprintf("step-%02d", tick)
		// Supplying a different seed on every reopen must keep the original layout.
		args := []string{"bakery", "-seed", "43", "-batch", batch, string(step.action), step.arg}
		if tick == 5 {
			execSQL(t, path, `CREATE TRIGGER fail_pickup BEFORE INSERT ON request_receipt WHEN NEW.request='step-05/1'
				BEGIN SELECT RAISE(FAIL, 'injected pickup failure'); END`)
			if out := run(1, args...); !strings.Contains(out, "injected pickup failure") || strings.Contains(out, "You take the") {
				t.Fatal(out)
			}
			check()
			if view := run(0, "bakery"); !strings.Contains(view, "wooden spoon in the crate") {
				t.Fatal(view)
			}
			execSQL(t, path, "DROP TRIGGER fail_pickup")
			t.Log("pickup failed after the staged placement and events were written; reopen preserved the tick-4 crate and history")
		}
		p := simulation.Proposal{Scope: s, Version: 1, RequestID: batch + "/1", Action: step.action}
		if step.action == simulation.ActionMove {
			p.Args.Direction = simulation.Direction(step.arg)
		} else {
			p.ByName = true
			p.Args.Target = step.arg
		}
		want, next := live.Resolve(s, uint64(len(events)), actor, p, nil)
		if next == nil || want.Status != "accepted" {
			t.Fatal(want)
		}
		live = next.State
		events = append(events, next.Events...)
		receipts = append(receipts, simulation.Receipt{Proposal: p, Result: want})
		scene := want.Observation.Text + "\n" + want.Observation.Grid + "\n"
		if out := run(0, args...); !strings.HasSuffix(out, scene) {
			t.Fatalf("wrong action view: %s", out)
		}
		if out := run(0, "bakery"); out != scene {
			t.Fatalf("wrong reopened view: %s", out)
		}
		check()
		if tick == 5 {
			run(0, args...)
			check()
			report(5, mid, "checkpoint", "-world", "bakery")
		}
	}
	report(10, final, "replay", "-world", "bakery", "-from", "10")
	report(10, final, "replay", "-world", "bakery", "-from", "0")
	wantStreet := simulation.Placement{Cell: simulation.CellPosition{FloorID: "bakery/ground", X: 1, Y: 1}}
	if live.Bakery.Entities["bakery/spoon"].Placement != wantStreet || live.Bakery.Entities["bakery/crate"].Open || live.Bakery.Entities["bakery/door"].Open {
		t.Fatal("unexpected final placement or open state")
	}
	// A late retry while the crate is hidden returns the original receipt to
	// the caller. The terminal must continue to show the current street view.
	old := receipts[3]
	got, repeated, err := storage.SubmitBakery(path, s, actor, old.Proposal)
	if err != nil || !repeated || !reflect.DeepEqual(got, old.Result) {
		t.Fatalf("late retry: %+v, %v", got, err)
	}
	if out := run(0, "bakery", "-batch", "step-04", "open", "crate"); strings.Contains(out, "crate") || !strings.Contains(out, "already completed") {
		t.Fatalf("late retry exposed a stale view: %s", out)
	}
	run(1, "bakery", "-batch", "step-04", "move", "west")
	check()

	// The two histories share the initial seed and stable IDs, then diverge.
	run(0, "bakery", "-timeline", "side", "-seed", "42")
	other := simulation.Scope{WorldID: s.WorldID, TimelineID: "side"}
	clock := simulation.NewTimeline(42)
	initial, err := simulation.GenerateBakery(s.WorldID, clock)
	if err != nil {
		t.Fatal(err)
	}
	data, err := initial.Snapshot()
	if err != nil || !bytes.Equal(bakeryBytes(t, path, other), data) {
		t.Fatal("side timeline did not start with the original layout")
	}
	run(0, "bakery", "-timeline", "side", "-batch", "step-01", "move", "west")
	if inspectScope(t, path, other).seq != 2 || bytes.Equal(bakeryBytes(t, path, other), data) {
		t.Fatal("side timeline did not diverge")
	}
	check()
}
