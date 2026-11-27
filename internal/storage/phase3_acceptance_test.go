// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

// Each call starts a new executable. The two saves have identical actor and
// request IDs so differences in events and player observations are meaningful.
func TestPhase3Acceptance(t *testing.T) {
	dir := t.TempDir()
	runner := filepath.Join(dir, "sloptown.exe")
	if out, err := exec.Command("go", "build", "-mod=readonly", "-o", runner, "../../cmd/sloptown").CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	run := func(path, input, command string, args ...string) string {
		t.Helper()
		cmd := exec.Command(runner, append([]string{command, "-db", path}, args...)...)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v: %s", command, args, err, out)
		}
		return string(out)
	}
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	parsed, explicit := filepath.Join(dir, "parsed.sqlite"), filepath.Join(dir, "explicit.sqlite")
	report := func(path, command string, args ...string) storage.ReplayReport {
		t.Helper()
		out := run(path, "", command, append([]string{"-world", s.WorldID}, args...)...)
		var r storage.ReplayReport
		if err := json.Unmarshal([]byte(out), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	// Give the two spoons and the two crates distinct full names so the
	// explicit runner can choose the same objects without a menu.
	bakery, err := simulation.GenerateBakery(s.WorldID, simulation.NewTimeline(42))
	if err != nil {
		t.Fatal(err)
	}
	brass := bakery.Entities["bakery/spoon"]
	brass.Name = "brass spoon"
	bakery.Entities["bakery/brass"] = brass
	crate := bakery.Entities["bakery/crate"]
	crate.Name = "wooden crate"
	bakery.Entities["bakery/crate"] = crate
	crate.Name = "flour crate"
	crate.Open = true
	crate.Placement.Cell.X++
	crate.Placement.Cell.Y++
	bakery.Entities["bakery/flour"] = crate
	stocked, err := bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{parsed, explicit} {
		run(path, "", "bakery", "-seed", "42")
		execSQL(t, path, "UPDATE timeline_state SET bakery=? WHERE world=? AND timeline=?", stocked, s.WorldID, s.TimelineID)
		report(path, "checkpoint")
	}
	compare := func(tick int64) {
		t.Helper()
		a, b := inspectScope(t, parsed, s), inspectScope(t, explicit, s)
		if a.seq != tick*2 || a.seq != b.seq || a.clock != b.clock || !slices.Equal(a.events, b.events) ||
			int64(len(a.receipts)) != tick || len(a.receipts) != len(b.receipts) ||
			!bytes.Equal(bakeryBytes(t, parsed, s), bakeryBytes(t, explicit, s)) {
			t.Fatalf("saves diverged at tick %d", tick)
		}
		for i, receipt := range a.receipts {
			x, y := receipt.Result, b.receipts[i].Result
			if x.Status != "accepted" || x.Status != y.Status || x.RequestID != y.RequestID || x.Message != y.Message ||
				!reflect.DeepEqual(x.Observation, y.Observation) {
				t.Fatalf("different outcome for %s at tick %d", x.RequestID, tick)
			}
		}
		if scene, other := run(parsed, "", "bakery"), run(explicit, "", "bakery"); scene != other {
			t.Fatalf("reopened scenes differ at tick %d", tick)
		}
	}
	play := func(batch, input string) string {
		return run(parsed, input, "bakery", "-play", "-batch", batch)
	}
	play("enter", "open door\ne\nwalk east\nopen wooden crate\n")
	run(explicit, "", "bakery", "-batch", "enter", "open", "bakery door", "move", "east", "move", "east", "open", "wooden crate")
	compare(4)
	mid := report(parsed, "checkpoint")
	if other := report(explicit, "checkpoint"); mid != other || mid.Sequence != 8 || mid.Tick != 4 {
		t.Fatalf("different midpoints: %+v, %+v", mid, other)
	}

	const pickup = "pick up the spoon\n2\nput spoon in crate\n2\n1\ntake it\nput it in crate\n1\nclose wooden crate\n"
	out := play("handling", pickup)
	if !strings.Contains(out, "2. wooden spoon in the wooden crate") || strings.Count(out, "You put the wooden spoon in the wooden crate.") != 2 {
		t.Fatalf("clarification or pronoun failed: %s", out)
	}
	explicitHandling := []string{"-batch", "handling", "take", "wooden spoon", "put", "wooden spoon", "wooden crate",
		"take", "wooden spoon", "put", "wooden spoon", "wooden crate", "close", "wooden crate"}
	run(explicit, "", "bakery", explicitHandling...)
	compare(9)
	before := inspectScope(t, parsed, s)
	// At restart both spoons are hidden. Repeating the original transcript
	// must recognize its numbers without exposing the old scene.
	out = play("handling", pickup)
	if strings.Count(out, "already completed") != 5 || strings.Contains(out, "wooden spoon") ||
		strings.Contains(out, "brass spoon") || !reflect.DeepEqual(before, inspectScope(t, parsed, s)) {
		t.Fatalf("late retry failed: %s", out)
	}
	out = play("handling", "get spoon\n1\n")
	if !strings.Contains(out, "different action") || strings.Contains(out, "already completed") ||
		!reflect.DeepEqual(before, inspectScope(t, parsed, s)) {
		t.Fatalf("different numbered answer reused the receipt: %s", out)
	}
	out = play("handling", "get spoon\n2\nput spoon in crate\n2\n2\n")
	if !strings.Contains(out, "different action") || strings.Count(out, "already completed") != 1 ||
		!reflect.DeepEqual(before, inspectScope(t, parsed, s)) {
		t.Fatalf("different destination reused the receipt: %s", out)
	}
	compare(9)
	for _, from := range []string{"0", "8"} {
		a, b := report(parsed, "replay", "-from", from), report(explicit, "replay", "-from", from)
		const closedHash = "0a8676c594c1182642dc6dd91a731f48e57aab67b37a7913826f5bd009a6e645"
		if a != b || a.Scope != s || a.Tick != 9 || a.Sequence != 18 || a.Hash != closedHash {
			t.Fatalf("replay from %s diverged: %+v, %+v", from, a, b)
		}
		t.Logf("parsed and explicit replay from %s: tick %d, sequence %d, sha256 %s", from, a.Tick, a.Sequence, a.Hash)
	}
	// Resume beyond the acknowledged prefix. The last receipt supplies the
	// subject of "open it", and only this new action advances the timeline.
	out = play("handling", pickup+"open it\n")
	if strings.Count(out, "already completed") != 5 || !strings.Contains(out, "You open the wooden crate.") {
		t.Fatalf("could not continue the batch: %s", out)
	}
	run(explicit, "", "bakery", append(explicitHandling, "open", "wooden crate")...)
	compare(10)
	a, b := report(parsed, "replay", "-from", "0"), report(explicit, "replay", "-from", "0")
	const reopenedHash = "ff2bd8ca717ff3579b5dd0c0ae6fbc8a555d24cae30fd6bcb95d8d30733573d8"
	if a != b || a.Tick != 10 || a.Sequence != 20 || a.Hash != reopenedHash {
		t.Fatalf("resumed replays diverged: %+v, %+v", a, b)
	}
	t.Logf("resumed with one new action: tick %d, sequence %d, sha256 %s", a.Tick, a.Sequence, a.Hash)
}
