// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

// Replay historical practice histories from the original tick-zero rules.
func usePracticeVersion(t *testing.T, path string, scope simulation.Scope, version int) {
	t.Helper()
	execSQL(t, path, `UPDATE timeline_state SET bakery=CAST(json_set(bakery, '$.bakery.ContentVersion', ?) AS BLOB)
		WHERE world=? AND timeline=? AND last_sequence=0`, version, scope.WorldID, scope.TimelineID)
}

func TestIntroductionAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	runner, path := filepath.Join(dir, "sloptown.exe"), filepath.Join(dir, "practice.sqlite")
	if out, err := exec.Command("go", "build", "-mod=readonly", "-o", runner, "../../cmd/sloptown").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	run := func(input, command string, args ...string) string {
		t.Helper()
		cmd := exec.Command(runner, append([]string{command, "-db", path}, args...)...)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v: %s", command, err, out)
		}
		return string(out)
	}
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	run("", "bakery", "-practice", "-seed", "42")
	usePracticeVersion(t, path, scope, 8)
	run("", "checkpoint", "-world", "bakery")
	play := func(script string) string { return run(script, "bakery", "-play", "-batch", "visit") }
	if out := play("south\nactions\n"); !strings.Contains(out, "introduce yourself to the porter") {
		t.Fatal(out)
	}
	const intro = "south\nintroduce yourself to the porter\n"
	if out := play(intro); !strings.Contains(out, "You introduce yourself to the porter.") {
		t.Fatal(out)
	}
	mid, err := storage.SaveCheckpoint(path, scope)
	if err != nil {
		t.Fatal(err)
	}
	before, actors := inspectScope(t, path, scope), bakeryBytes(t, path, scope)
	// The next wait lets the porter meet the baker independently.
	execSQL(t, path, `CREATE TRIGGER fail_intro BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL, 'interrupted introduction'); END`)
	wait := simulation.Proposal{Version: 1, Scope: scope, RequestID: "visit/3", Action: simulation.ActionWait}
	if _, _, err := storage.SubmitBakery(path, scope, "bakery/player", wait); err == nil || !strings.Contains(err.Error(), "interrupted introduction") {
		t.Fatal("expected interrupted receipt", err)
	}
	if !reflect.DeepEqual(before, inspectScope(t, path, scope)) || !bytes.Equal(actors, bakeryBytes(t, path, scope)) {
		t.Fatal("failed introduction changed the save")
	}
	execSQL(t, path, "DROP TRIGGER fail_intro")
	walk := intro + strings.Repeat("wait\n", 13)
	play(walk)
	if out := play(walk); strings.Count(out, "already completed") != 15 {
		t.Fatal(out)
	}
	for owner, line := range map[string]string{"porter": "Tick 2, event 6 — bakery/player introduced themselves to you.", "baker": "Tick 3, event 12 — bakery/porter introduced themselves to you.", "player": ""} {
		out := run("", "notebook", "-actor", "bakery/"+owner)
		want := 0
		if line != "" {
			want = 1
		}
		if strings.Count(out, "introduced themselves to you.") != want || !strings.Contains(out, line) {
			t.Fatal(out)
		}
	}
	for _, from := range []uint64{0, mid.Sequence} {
		head, err := storage.Replay(path, scope, from)
		if err != nil || head.Tick != 15 || head.Sequence != 52 || head.Hash != "3af00083bdcb72e16a28e3dc3273bd488f203e39188b39e8066c6031875b546a" {
			t.Fatalf("from %d: %+v, %v", from, head, err)
		}
		t.Logf("from %d: tick %d, sequence %d, sha256 %s", from, head.Tick, head.Sequence, head.Hash)
	}
}
