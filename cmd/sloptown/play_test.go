// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestParseBakeryAction(t *testing.T) {
	for _, tc := range []struct {
		line string
		want simulation.ActionKind
		args simulation.ActionArgs
	}{
		{"wait", simulation.ActionWait, simulation.ActionArgs{}},
		{"rest", simulation.ActionRest, simulation.ActionArgs{}},
		{"open the bakery door", simulation.ActionOpen, simulation.ActionArgs{Target: "bakery door"}},
		{"close a crate", simulation.ActionClose, simulation.ActionArgs{Target: "crate"}},
		{"n", simulation.ActionMove, simulation.ActionArgs{Direction: simulation.North}},
		{"go south", simulation.ActionMove, simulation.ActionArgs{Direction: simulation.South}},
		{"move e", simulation.ActionMove, simulation.ActionArgs{Direction: simulation.East}},
		{"walk west", simulation.ActionMove, simulation.ActionArgs{Direction: simulation.West}},
		{"pick up the wooden spoon", simulation.ActionTake, simulation.ActionArgs{Target: "wooden spoon"}},
		{"get wooden spoon", simulation.ActionTake, simulation.ActionArgs{Target: "wooden spoon"}},
		{"put a wooden spoon into the crate", simulation.ActionPut, simulation.ActionArgs{Target: "wooden spoon", Container: "crate"}},
		{"drop the wooden spoon", simulation.ActionDrop, simulation.ActionArgs{Target: "wooden spoon"}},
	} {
		action, args, err := parseBakeryAction(strings.Fields(tc.line))
		if err != nil || action != tc.want || args != tc.args {
			t.Errorf("%q produced %q %+v, %v", tc.line, action, args, err)
		}
	}
	for _, line := range []string{"rest baker", "wait twice", "open the", "pick", "pick up", "go northeast", "east twice", "put spoon", "put the in crate", "put spoon in", "dance"} {
		if _, _, err := parseBakeryAction(strings.Fields(line)); err == nil {
			t.Errorf("accepted malformed command %q", line)
		}
	}
}

func TestBakeryPlaySession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "town.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	play := func(batch, input string) string {
		t.Helper()
		var out, diag bytes.Buffer
		code := runBakery([]string{"-play", "-seed", "42", "-db", path, "-batch", batch}, strings.NewReader(input), &out, &diag)
		if code != 0 || diag.Len() != 0 {
			t.Fatalf("session exited %d: %s", code, diag.String())
		}
		return out.String()
	}
	// A fresh session can recover from syntax errors and rejected proposals.
	street := play("errors", "help\nlook\nWhat can I do here?\nactions\nflap\neast\nopen the crate\nopen the cupboard\nquit\nopen the bakery door\n")
	if !strings.Contains(street, "I don't understand") || !strings.Contains(street, "You can't move there") ||
		strings.Count(street, "You can't reach that target.") != 2 || strings.Contains(street, "crate") || strings.Contains(street, "spoon") {
		t.Fatalf("unexpected response from the street: %s", street)
	}
	cp, err := storage.SaveCheckpoint(path, scope)
	if err != nil || cp.Tick != 0 || cp.Sequence != 0 {
		t.Fatalf("queries or rejected input advanced the save: %+v, %v", cp, err)
	}
	play("pickup", "east\n OPEN   the bakery door.\r\nGO East\ne\nopen the crate\nactions\npick up the wooden spoon\n")
	// Canonically equivalent phrases use the existing receipts across sessions.
	retry := play("pickup", "east\nopen bakery door\nwalk east\nmove east\nopen crate\nget wooden spoon\n")
	if strings.Count(retry, "already completed") != 5 {
		t.Fatalf("retry did not reuse all five receipts: %s", retry)
	}
	out := play("return", "close the crate\nwest\nw\nclose door\ndrop spoon\nlook\n")
	if !strings.Contains(out, "You see the wooden spoon here.") {
		t.Fatalf("unexpected final scene: %s", out)
	}
	head, err := storage.Replay(path, scope, cp.Sequence)
	const want = "9f421909302942c47ba2ac7c46c627591ef95360749cff544ce23d206bafe58b"
	if err != nil || head.Hash != want || head.Tick != 10 || head.Sequence != 20 {
		t.Fatalf("parsed walkthrough diverged from Phase 2: %+v, %v", head, err)
	}
}

func TestBakerTerminalSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "town.sqlite")
	run := func(input string, args ...string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := runBakery(append([]string{"-db", path}, args...), strings.NewReader(input), &out, &diag); code != 0 {
			t.Fatalf("bakery %v exited %d: %s", args, code, diag.String())
		}
		return out.String()
	}
	start := run("look\nactions\neast\n", "-resident", "-play")
	s := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	cp, err := storage.SaveCheckpoint(path, s)
	if err != nil || cp.Tick != 0 || strings.Contains(start, "the baker ") || !strings.Contains(start, "  wait\n") {
		t.Fatalf("starting street: %s, checkpoint %+v, %v", start, cp, err)
	}
	first := run("", "-batch", "visit", "wait")
	if !strings.Contains(first, "You see the open bakery door") || !strings.Contains(first, "You see the baker to the east.") {
		t.Fatal(first)
	}
	visit := run("wait\nwait\nwait\nlook\n", "-play", "-batch", "visit")
	if strings.Count(visit, "already completed") != 1 || strings.Count(visit, "You wait.") != 2 || !strings.Contains(visit, "You see the baker here.") {
		t.Fatal(visit)
	}
	head, err := storage.Replay(path, s, 0)
	if err != nil || head.Tick != 3 || head.Sequence != 9 {
		t.Fatalf("resumed visit: %+v, %v", head, err)
	}
	if !strings.Contains(start, "  rest\n") || !strings.Contains(run("", "-batch", "break", "rest"), "You are sitting down.") {
		t.Fatal("the terminal should offer rest and commit the player's break")
	}
	if out := run("rest\nlook\n", "-batch", "break", "-play"); !strings.Contains(out, "already completed") || !strings.Contains(out, "You are sitting down.") {
		t.Fatal(out)
	}
	run("", "-timeline", "classic")
	classic := run("", "-timeline", "classic", "-resident", "wait")
	if strings.Contains(classic, "the baker ") || strings.Contains(classic, "the open bakery door") {
		t.Fatal("reopening a classic timeline changed its content:", classic)
	}
}

func TestBakeryActionDiscovery(t *testing.T) {
	const allMoves = "go north|go east|go south|go west"
	steps := []struct {
		input string
		menu  string
	}{
		{"", "go north|go south|go west|open the bakery door"},
		{"open the bakery door", allMoves + "|close the bakery door"},
		{"go east", "go east|go west"}, // The doorway is occupied. The visible crate is out of reach.
		{"go east", allMoves + "|open the crate|close the bakery door"},
		{"open the crate", allMoves + "|close the crate|close the bakery door|take the wooden spoon"},
		{"take the wooden spoon", allMoves + "|close the crate|close the bakery door|drop the wooden spoon|put the wooden spoon in the crate"},
		{"put the wooden spoon in the crate", allMoves + "|close the crate|close the bakery door|take the wooden spoon"},
		{"go west\ngo west", allMoves + "|close the bakery door"},
	}
	for _, mode := range []string{"memory", "sqlite"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"-play", "-seed", "42"}
			path := filepath.Join(t.TempDir(), "bakery.sqlite")
			if mode == "sqlite" {
				args = append(args, "-db", path)
			}
			var input strings.Builder
			for _, s := range steps {
				input.WriteString(s.input + "\nWhat  CAN I do here?\n")
			}
			run := func(lines string) string {
				t.Helper()
				var out, diag bytes.Buffer
				if code := runBakery(args, strings.NewReader(lines), &out, &diag); code != 0 {
					t.Fatalf("session exited %d: %s", code, diag.String())
				}
				return out.String()
			}
			menus := strings.Split(run(input.String()), "Available actions\n")[1:]
			if len(menus) != len(steps) {
				t.Fatalf("got %d menus, want %d", len(menus), len(steps))
			}
			for i, menu := range menus {
				got, _, _ := strings.Cut(strings.TrimSuffix(menu, "\n"), "\n\n")
				want := "  " + strings.ReplaceAll(steps[i].menu+"|wait", "|", "\n  ")
				if got != want {
					t.Errorf("step %d after %q:\ngot\n%s\nwant\n%s", i, steps[i].input, got, want)
				}
			}
			if mode == "sqlite" {
				reopened := run("actions\n")
				if !strings.HasSuffix(reopened, "Available actions\n"+menus[len(menus)-1]) {
					t.Fatalf("reopened street offered different actions: %s", reopened)
				}
			}
		})
	}
}
