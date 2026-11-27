// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestNotebookWitnessesAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	runner, save := filepath.Join(dir, "sloptown.exe"), filepath.Join(dir, "bakery.sqlite")
	if out, err := exec.Command("go", "build", "-mod=readonly", "-o", runner, "../../cmd/sloptown").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	run := func(input, command string, args ...string) string {
		t.Helper()
		cmd := exec.Command(runner, append([]string{command, "-db", save}, args...)...)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v: %s", command, args, err, out)
		}
		return string(out)
	}
	legacyResidentSave(t, save, s, 4)
	run("", "bakery", "-resident", "-seed", "42")
	run("", "checkpoint", "-world", "bakery")
	run("", "bakery", "-batch", "outside", "rest")
	run("", "checkpoint", "-world", "bakery")
	read := func(owner string) string { return run("", "notebook", "-actor", "bakery/"+owner) }
	street := "Notebook of bakery/porter (timeline main)\nTick 1, event 2 — saw bakery/player sit down.\n"
	if got := read("porter"); got != street {
		t.Fatal(got)
	}
	if got := read("baker"); !strings.HasSuffix(got, "No notebook entries.\n") {
		t.Fatal(got)
	}
	out := run("notebook\nlook\nactions\nquit\n", "bakery", "-play")
	if !strings.Contains(out, "I don't understand") || strings.Contains(out, "saw bakery/") {
		t.Fatalf("unexpected player view: %s", out)
	}
	run("", "bakery", "-batch", "inside", "move", "east", "move", "east", "rest", "open", "crate")
	before, entities := inspectScope(t, save, s), bakeryBytes(t, save, s)
	// Fail after the new notebook page and its source event have been staged.
	execSQL(t, save, `CREATE TRIGGER fail_notebook BEFORE INSERT ON request_receipt
		BEGIN SELECT RAISE(FAIL, 'interrupted turn'); END`)
	p := simulation.Proposal{Scope: s, Version: 1, RequestID: "close/1", ByName: true, Action: simulation.ActionClose, Args: simulation.ActionArgs{Target: "crate"}}
	if r, _, err := storage.SubmitBakery(save, s, "bakery/player", p); err == nil || !strings.Contains(err.Error(), "interrupted turn") || r.Status == "accepted" {
		t.Fatalf("expected interrupted turn: %+v, %v", r, err)
	}
	if !bytes.Equal(bakeryBytes(t, save, s), entities) || !reflect.DeepEqual(inspectScope(t, save, s), before) {
		t.Fatal("failed turn published part of a sighting or its cause")
	}
	execSQL(t, save, "DROP TRIGGER fail_notebook")
	run("", "bakery", "-batch", "close", "close", "crate")
	if out := run("", "bakery", "-batch", "close", "close", "crate"); !strings.Contains(out, "already completed") {
		t.Fatal(out)
	}
	inside := "Notebook of bakery/baker (timeline main)\nTick 4, event 11 — saw bakery/player sit down.\nTick 6, event 16 — saw bakery/player close bakery/crate.\n"
	if got := read("baker"); got != inside {
		t.Fatal(got)
	}
	if got := read("porter"); got != street {
		t.Fatal(got)
	}
	run("", "bakery", "-resident", "-timeline", "other")
	if other := run("", "notebook", "-timeline", "other", "-actor", "bakery/porter"); !strings.HasSuffix(other, "No notebook entries.\n") {
		t.Fatal(other)
	}
	for _, from := range []string{"0", "3"} {
		var head storage.ReplayReport
		out := run("", "replay", "-world", "bakery", "-from", from)
		if err := json.Unmarshal([]byte(out), &head); err != nil || head.Tick != 6 || head.Sequence != 17 ||
			head.Hash != "b6ba766b3ae5505f6bc231b035a5a53873aefd2843c05791095b9b446d495a56" {
			t.Fatalf("replay from %s: %s", from, out)
		}
	}
}
