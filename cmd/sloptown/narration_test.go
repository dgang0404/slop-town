// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
)

func TestReadableBakeryTranscript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bakery.sqlite")
	play := func(input string, args ...string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := runBakery(append([]string{"-play"}, args...), strings.NewReader(input), &out, &diag); code != 0 {
			t.Fatalf("exit %d: %s", code, &diag)
		}
		if strings.ContainsRune(out.String(), '\x1b') {
			t.Fatal("redirected output contains terminal formatting")
		}
		return out.String()
	}
	const route = "open door\ne\ne\nopen crate\nwait\nlook\nwait\n"
	saved := play(route, "-db", path, "-batch", "visit")
	if saved != play(route) || strings.Count(saved, "@") != 2 ||
		!strings.Contains(saved, "\nYou open the crate. Inside lies a wooden spoon.\n\nYou wait.\nYou are in the bakery.\n") ||
		!strings.HasSuffix(saved, "\n\nYou wait.\n") {
		t.Fatal(saved)
	}
	reopened := play("wait\n", "-db", path, "-batch", "later")
	if strings.Count(reopened, "@") != 1 || strings.Count(reopened, "wooden spoon") != 1 || !strings.HasSuffix(reopened, "\n\nYou wait.\n") {
		t.Fatal(reopened)
	}
	retry := play(route, "-db", path, "-batch", "visit")
	if strings.Count(retry, "already completed") != 6 || strings.Contains(retry, "Inside lies") {
		t.Fatal(retry)
	}
	lost := play("wait\nwait\nwait\nwest\nnorth\nwalk to baker\nlook\n", "-practice")
	_, last, ok := strings.Cut(lost, "You walk toward the baker.")
	if !ok || strings.Count(last, "baker.") != 1 || !strings.Contains(last, "You lose sight of the baker.") ||
		strings.Contains(last, "destination") || strings.Count(last, "@") != 1 {
		t.Fatal(lost)
	}
}

func TestNarrationNamesOnlyWitnessedActions(t *testing.T) {
	play := func(input string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := runBakery([]string{"-practice", "-play"}, strings.NewReader(input), &out, &diag); code != 0 {
			t.Fatal(diag.String())
		}
		return out.String()
	}
	shopfront := play(strings.Repeat("wait\n", 12))
	downStreet := play("south\nsouth\n" + strings.Repeat("wait\n", 10))
	if !strings.Contains(shopfront, "The bakery door is now open.") || strings.Contains(shopfront, "opens the bakery door") {
		t.Fatal("the closed door hid who opened it", shopfront)
	}
	if strings.Count(shopfront, "The baker sits down.") != 1 || strings.Count(shopfront, "The baker's shoulders look relaxed.") != 1 ||
		strings.Contains(downStreet, "The baker sits down.") || strings.Contains(downStreet, "The baker's shoulders look relaxed.") {
		t.Fatal("the two observers had different views of the break", shopfront, downStreet)
	}
}

func TestWalkNarrationKeepsWitnessedInterruption(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player, crate = "bakery/player", "bakery/crate"
	state := simulation.State{Clock: simulation.NewTimeline(42)}
	bakery, err := (simulation.BakerySetup{Practice: true}).Generate(scope.WorldID, state.Clock)
	if err != nil {
		t.Fatal(err)
	}
	state.Bakery = bakery
	// The residents meet along the visible route to this street corner.
	target := bakery.Entities[crate]
	target.Placement.Cell.X, target.Placement.Cell.Y = 0, 0
	bakery.Entities[crate] = target
	initial, err := bakery.Scene(scope, player)
	if err != nil {
		t.Fatal(err)
	}
	plain, bold := turnNarrator{previous: initial}, turnNarrator{previous: initial, bold: true}
	var transcript, styled bytes.Buffer
	var sequence uint64
	move := simulation.Proposal{Action: simulation.ActionMove, Args: simulation.ActionArgs{Direction: simulation.South}}
	walk := simulation.Proposal{Action: simulation.ActionApproach, Args: simulation.ActionArgs{Target: crate}}
	for i, p := range []simulation.Proposal{move, move, walk, walk} {
		p.Scope, p.Version, p.RequestID = scope, 1, fmt.Sprint(i)
		r, step := state.Resolve(scope, sequence, player, p, nil)
		if step == nil || r.ContinueWalk != (i == 2) {
			t.Fatalf("step %d: %+v", i, r)
		}
		cmd := bakeryCommand{Proposal: p, continuation: i == 3}
		turn := simulation.Turn{Result: r, Scene: step.Scene}
		plain.write(&transcript, cmd, turn)
		bold.write(&styled, cmd, turn)
		state, sequence = step.State, sequence+uint64(len(step.Events))
	}
	out := transcript.String()
	if strings.Count(out, "You walk toward the crate.") != 1 || !strings.Contains(out, "introduces themselves to you. You pause your walk.") ||
		strings.Contains(out, "within reach") || state.Clock.Tick() != 4 {
		t.Fatal(out)
	}
	unstyle := strings.NewReplacer("\x1b[1m", "", "\x1b[0m", "")
	if !strings.Contains(styled.String(), "\x1b[1mYou pause your walk.\x1b[0m") || unstyle.Replace(styled.String()) != out {
		t.Fatal(styled.String())
	}
}

func TestNarrateReplacedSpoonByReference(t *testing.T) {
	spoon := func(ref, location string) simulation.Scene {
		return simulation.Scene{
			Observation: simulation.Observation{ObserverID: "player", Targets: []simulation.ObservedTarget{{Ref: ref, Name: "wooden spoon"}}},
			Details:     map[string]simulation.VisibleDetail{ref: {Label: "wooden spoon", Location: location}},
		}
	}
	n := turnNarrator{previous: spoon("first", "here"), bold: true}
	turn := simulation.Turn{Result: simulation.Result{Message: "You wait."}, Scene: spoon("second", "to the north")}
	var out bytes.Buffer
	n.write(&out, bakeryCommand{Proposal: simulation.Proposal{Action: simulation.ActionWait}}, turn)
	const want = "\nYou wait. You see \x1b[1mthe wooden spoon\x1b[0m to the north. You lose sight of \x1b[1mthe wooden spoon here\x1b[0m.\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}
