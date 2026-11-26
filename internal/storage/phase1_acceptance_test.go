// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"encoding/json"
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

// Each CLI call starts a fresh process. The reference execution keeps its
// clock and history in memory for all three requests.
func TestPhase1Acceptance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "town.sqlite")
	binary := filepath.Join(dir, "sloptown.exe")
	if out, err := exec.Command("go", "build", "-mod=readonly", "-o", binary, "../../cmd/sloptown").CombinedOutput(); err != nil {
		t.Fatalf("build runner: %v\n%s", err, out)
	}

	command := func(args ...string) *exec.Cmd {
		return exec.Command(binary, append(args, "-db", path)...)
	}
	runJSON := func(dst any, args ...string) {
		t.Helper()
		out, err := command(args...).CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		if err := json.Unmarshal(out, dst); err != nil {
			t.Fatalf("%v: invalid JSON: %v\n%s", args, err, out)
		}
	}

	s := simulation.Scope{WorldID: "smoke", TimelineID: "main"}
	live := simulation.NewTimeline(42)
	var history []simulation.Event
	var receipts []simulation.Receipt
	step := func(id string) simulation.Result {
		t.Helper()
		var got simulation.Result
		runJSON(&got, "step", "-seed", "42", "-request", id)
		p := proposal(id)
		p.Scope = s
		want, transition := simulation.Resolve(s, live, uint64(len(history)), "player", p, nil)
		if got != want || got.Status != "accepted" || transition == nil {
			t.Fatalf("step %s: got %+v, want %+v", id, got, want)
		}
		live = transition.Clock
		history = append(history, transition.Events...)
		receipts = append(receipts, simulation.Receipt{Proposal: p, Result: want})
		state := inspectScope(t, path, s)
		clock, err := live.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if state.clock != string(clock) || state.seq != int64(len(history)) {
			t.Fatalf("step %s: reopened clock or sequence diverged: %+v", id, state)
		}
		if !slices.Equal(state.events, history) || !slices.Equal(state.receipts, receipts) {
			t.Fatalf("step %s: reopened events or receipts diverged from uninterrupted execution: %+v", id, state)
		}
		return got
	}

	report := func(tick int64, digest string, args ...string) {
		t.Helper()
		var got storage.ReplayReport
		runJSON(&got, args...)
		want := storage.ReplayReport{Scope: s, Tick: tick, Sequence: uint64(2 * tick), Hash: digest}
		if got != want {
			t.Fatalf("%v: got %+v, want %+v", args, got, want)
		}
		t.Logf("%s: tick %d, sequence %d, sha256 %s", args[0], got.Tick, got.Sequence, got.Hash)
	}

	step("r1")
	report(1, "d899d2cf3ae733c0ba0fce7d0f64a6ad5236654ae33a5284966006d373abe034", "checkpoint")
	second := step("r2")
	const atTwo = "d508c6d2e9416be9d60c32bbc51cb36d2c2be7e13c146057d74dad10c525554d"
	report(2, atTwo, "replay", "-from", "2")
	before := inspectScope(t, path, s)

	var retry simulation.Result
	runJSON(&retry, "step", "-request", "r2")
	if retry != second || !reflect.DeepEqual(before, inspectScope(t, path, s)) {
		t.Fatal("duplicate request changed the receipt or saved state")
	}
	t.Log("retry r2 returned the original receipt; clock, streams and events matched the pre-retry save")

	// The receipt is the last write in a step, after the clock and both events.
	execSQL(t, path, `CREATE TRIGGER fail_r3 BEFORE INSERT ON request_receipt WHEN NEW.request='r3'
		BEGIN SELECT RAISE(FAIL, 'injected r3 receipt failure'); END`)
	failed := command("step", "-request", "r3")
	var diagnostic bytes.Buffer
	failed.Stderr = &diagnostic
	out, err := failed.Output()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(out) != 0 || !strings.Contains(diagnostic.String(), "injected r3 receipt failure") {
		t.Fatalf("expected injected failure with exit 1, got %v, stdout %q, stderr %q", err, out, diagnostic.String())
	}
	if !reflect.DeepEqual(before, inspectScope(t, path, s)) {
		t.Fatal("failed request changed the committed clock, stream counters, event history or receipts")
	}
	t.Log("r3 receipt write failed with exit 1; reopened save matched the pre-failure state")
	report(2, atTwo, "replay", "-from", "2")
	execSQL(t, path, "DROP TRIGGER fail_r3")
	step("r3")
	report(3, "39dbddba99553b8b9754617c159317ff7627457297497b73dd3886997e9c4d4c", "replay", "-from", "2")
}
