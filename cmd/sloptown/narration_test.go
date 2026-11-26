// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
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
	const loseResidents = "wait\nwait\nwait\nwest\nnorth\nwalk to baker\nlook\n"
	lost := play(loseResidents, "-practice")
	_, last, ok := strings.Cut(lost, "You walk toward the baker.")
	if !ok || strings.Count(last, "baker.") != 1 || !strings.Contains(last, "You lose sight of the baker.") ||
		strings.Contains(last, "destination") || strings.Count(last, "@") != 1 {
		t.Fatal(lost)
	}
	story := play(loseResidents, "-practice", "-narration", "story")
	_, storyEnd, _ := strings.Cut(story, "You walk toward the baker.")
	notice, fullScene, _ := strings.Cut(storyEnd, "\n")
	if notice != " You lose sight of the baker and the porter, last seen to the southeast before your last step." ||
		strings.Contains(fullScene, "the baker ") || strings.Contains(fullScene, "the porter") {
		t.Fatal(storyEnd)
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
	story := turnNarrator{previous: initial, style: "story"}
	var transcript, styled, passage bytes.Buffer
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
		story.write(&passage, cmd, turn)
		state, sequence = step.State, sequence+uint64(len(step.Events))
	}
	for _, out := range []string{transcript.String(), passage.String()} {
		if strings.Count(out, "You walk toward the crate.") != 1 || !strings.Contains(out, "introduces themselves to you. You pause your walk.") ||
			strings.Contains(out, "within reach") || state.Clock.Tick() != 4 {
			t.Fatal(out)
		}
	}
	unstyle := strings.NewReplacer("\x1b[1m", "", "\x1b[0m", "")
	if !strings.Contains(styled.String(), "\x1b[1mYou pause your walk.\x1b[0m") || unstyle.Replace(styled.String()) != transcript.String() {
		t.Fatal(styled.String())
	}
}

func TestNarrationChoiceAcrossReplay(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const route = "open door\ne\ne\nopen crate\ntake spoon\nput it in crate\nwait\nlook\n"
	play := func(path, input string, flags ...string) string {
		t.Helper()
		var out, diag bytes.Buffer
		args := append([]string{"-play", "-batch", "visit", "-db", path}, flags...)
		if code := runBakery(args, strings.NewReader(input), &out, &diag); code != 0 {
			t.Fatalf("exit %d: %s", code, &diag)
		}
		return out.String()
	}
	var baseline storage.ReplayReport
	var receipts []*simulation.Receipt
	for _, mode := range []string{"brief", "story", "switch"} {
		path := filepath.Join(t.TempDir(), "town.sqlite")
		play(path, "quit\n")
		if _, err := storage.SaveCheckpoint(path, scope); err != nil {
			t.Fatal(err)
		}
		input := route
		var flags []string
		switch mode {
		case "story":
			flags = []string{"-narration", "story"}
		case "switch":
			input = strings.Replace(input, "open crate", "narration story\nnarration typo\nnarration\nopen crate", 1)
			input = strings.Replace(input, "put it", "narration brief\nput it", 1)
		}
		out := play(path, input, flags...)
		if strings.Count(out, "@") != 2 || strings.ContainsRune(out, '\x1b') || !strings.Contains(out, "\nYou wait.\n") {
			t.Fatal(out)
		}
		if mode == "brief" && !strings.Contains(out, "Inside lies a wooden spoon.") ||
			mode != "brief" && !strings.Contains(out, "You open the crate and find a wooden spoon inside.\n\nYou pick up the wooden spoon you saw in the crate.") ||
			mode == "story" && !strings.Contains(out, "You step east into the bakery. The closed crate is beside you now.") ||
			mode == "switch" && !strings.Contains(out, "Narration is story. Choose narration brief or narration story.") {
			t.Fatal(out)
		}
		if retry := play(path, "narration\n"+route); strings.Count(retry, "already completed") != 7 ||
			!strings.Contains(retry, "Narration is brief.") || strings.Contains(retry, "Inside lies") {
			t.Fatal(retry)
		}
		head, err := storage.Replay(path, scope, 0)
		if err != nil || head.Tick != 7 || head.Sequence != 14 {
			t.Fatalf("%s: %+v, %v", mode, head, err)
		}
		var saved []*simulation.Receipt
		for i := 1; i <= 7; i++ {
			r, err := storage.BakeryReceipt(path, scope, "bakery/player", fmt.Sprintf("visit/%d", i))
			if err != nil || r == nil {
				t.Fatalf("receipt %d: %v", i, err)
			}
			saved = append(saved, r)
		}
		if mode == "brief" {
			baseline, receipts = head, saved
		} else if head != baseline || !reflect.DeepEqual(saved, receipts) {
			t.Fatalf("%s changed the saved visit: %+v, want %+v", mode, head, baseline)
		}
	}
}

func TestInvalidNarrationFlag(t *testing.T) {
	for _, args := range [][]string{{"-play", "-narration", "typo"}, {"-narration", "story", "wait"}} {
		path := filepath.Join(t.TempDir(), "town.sqlite")
		var out, diag bytes.Buffer
		if code := runBakery(append([]string{"-db", path}, args...), strings.NewReader(""), &out, &diag); code != 2 || out.Len() != 0 || diag.Len() == 0 {
			t.Fatalf("%v: exit %d, %s, %s", args, code, &out, &diag)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("invalid flags touched the save: %v", err)
		}
	}
}

func TestStoryGroupsVisibleContentsOfSelectedCrate(t *testing.T) {
	// A second crate with the same name can become visible on this turn too.
	// Only the selected crate's newly visible contents belong in the opening sentence.
	objects := []simulation.ObservedTarget{
		{Ref: "first", Name: "crate"}, {Ref: "second", Name: "crate"},
		{Ref: "apple", Name: "apple"}, {Ref: "spoon", Name: "wooden spoon"}, {Ref: "cup", Name: "cup"},
	}
	prior := simulation.Scene{
		Observation: simulation.Observation{ObserverID: "player", Targets: objects[:1]},
		Details:     map[string]simulation.VisibleDetail{"first": {Label: "closed crate", State: "closed"}},
	}
	current := simulation.Scene{
		Observation: simulation.Observation{ObserverID: "player", Targets: objects},
		Details: map[string]simulation.VisibleDetail{
			"first": {Label: "open crate", State: "open"}, "second": {Label: "open crate", Location: "here"},
			"apple": {Label: "apple", Holder: "first"}, "spoon": {Label: "wooden spoon", Holder: "first"},
			"cup": {Label: "cup", Holder: "second", Location: "in the crate"},
		},
	}
	var out bytes.Buffer
	n := turnNarrator{style: "story", previous: prior, bold: true}
	turn := simulation.Turn{Result: simulation.Result{Message: "You open the crate.", SelectedTarget: "first"}, Scene: current}
	n.write(&out, bakeryCommand{Proposal: simulation.Proposal{Action: simulation.ActionOpen}}, turn)
	if !strings.HasPrefix(out.String(), "\nYou open the crate and find \x1b[1man apple\x1b[0m and \x1b[1ma wooden spoon\x1b[0m inside.") ||
		!strings.Contains(out.String(), "You see \x1b[1mthe cup\x1b[0m in the crate.") {
		t.Fatal(&out)
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
