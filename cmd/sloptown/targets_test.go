// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"database/sql"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

// MultiReader reaches this boundary after the terminal has consumed the earlier lines.
type betweenCommands func()

func (f betweenCommands) Read([]byte) (int, error) { f(); return 0, io.EOF }

func TestBakeryClarification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "choices.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player = "bakery/player"
	if _, err := storage.OpenBakery(path, scope, 42, player); err != nil {
		t.Fatal(err)
	}
	// Stock a test scene at tick zero. The original spoon stays behind the closed door.
	b, err := simulation.GenerateBakery(scope.WorldID, simulation.NewTimeline(42))
	if err != nil {
		t.Fatal(err)
	}
	here := b.Entities[player].Placement
	north := here
	north.Cell.Y--
	for ref, at := range map[string]simulation.Placement{"bakery/first": north, "bakery/second": here} {
		b.Entities[ref] = simulation.SpatialEntity{Name: "wooden spoon", Kind: simulation.Item, Portable: true, Placement: at}
	}
	for _, name := range []string{"flour box", "tea box"} {
		b.Entities["bakery/"+name] = simulation.SpatialEntity{Name: name, Kind: simulation.Container, Open: true, Placement: here}
	}
	b.Entities["bakery/baker"] = simulation.SpatialEntity{Name: "baker", Kind: simulation.Actor, Placement: here}
	data, err := b.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec("UPDATE timeline_state SET bakery=? WHERE world=? AND timeline=?", data, scope.WorldID, scope.TimelineID); err != nil {
		t.Fatal(err)
	}
	initial, err := storage.SaveCheckpoint(path, scope)
	if err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	play := func(input io.Reader) string {
		t.Helper()
		out.Reset()
		diag.Reset()
		if code := runBakery([]string{"-play", "-db", path}, input, &out, &diag); code != 0 {
			t.Fatalf("session exited %d: %s", code, &diag)
		}
		return out.String()
	}
	replay := func() storage.ReplayReport {
		t.Helper()
		head, err := storage.Replay(path, scope, 0)
		if err != nil {
			t.Fatal(err)
		}
		return head
	}
	const choices = "1. wooden spoon to the north\n  2. wooden spoon here"
	if out := play(strings.NewReader("walk to spoon\n1\n")); !strings.Contains(out, choices) || !strings.Contains(out, "already within reach") || strings.Contains(out, "3. wooden spoon") || replay() != initial {
		t.Fatalf("walk selected an unknown spoon or spent time: %s", out)
	}
	s := play(strings.NewReader("actions\ntake it\ntake spoon\n0\nnarration story\nlook\ncancel\ntake spoon\nquit\n"))
	if !strings.Contains(s, "Name the object") || strings.Count(s, choices) != 2 ||
		strings.Count(s, "  take the wooden spoon\n") != 1 || strings.Contains(s, "3. wooden spoon") ||
		strings.Contains(s, "bakery/") || !strings.Contains(s, "Narration is story.") || !strings.Contains(s, "Cancelled.") ||
		replay() != initial {
		t.Fatalf("unexpected clarification or changed save: %s", s)
	}
	s = play(strings.NewReader("take spoon\n2\nput spoon in box\n2\n2\ntake it\ndrop it\n"))
	if !strings.Contains(s, "Which box") || !strings.Contains(s, "You put the wooden spoon in the tea box.") ||
		strings.Count(s, "You take the wooden spoon.") != 2 || replay().Tick != 4 {
		t.Fatalf("two-noun clarification or pronoun failed: %s", s)
	}
	bakerActs := func(request string, action simulation.ActionKind, ref string) {
		t.Helper()
		r, _, err := storage.SubmitBakery(path, scope, "bakery/baker", simulation.Proposal{
			Scope: scope, Version: 1, RequestID: request, Action: action, Args: simulation.ActionArgs{Target: ref},
		})
		if err != nil || r.Status != "accepted" {
			t.Fatalf("baker could not %s spoon: %+v, %v", action, r, err)
		}
	}
	// While the numbered choices are on screen, the baker takes the first
	// spoon. Number 1 must never be rebound to the remaining visible spoon.
	input := io.MultiReader(strings.NewReader("take spoon\n"), betweenCommands(func() {
		if !strings.Contains(out.String(), choices) {
			t.Fatalf("expected choices before the scene changed: %s", &out)
		}
		bakerActs("take-first", simulation.ActionTake, "bakery/first")
	}), strings.NewReader("1\ndrop it\n"))
	s = play(input)
	if !strings.Contains(s, "You can't reach that target.") || !strings.Contains(s, "Name the object") ||
		replay().Tick != 5 {
		t.Fatalf("stale selection changed state or focus: %s", s)
	}
	s = play(io.MultiReader(strings.NewReader("take spoon\ndrop it\n"), betweenCommands(func() {
		bakerActs("take-second", simulation.ActionTake, "bakery/second")
		bakerActs("drop-first", simulation.ActionDrop, "bakery/first")
	}), strings.NewReader("take it\n")))
	if !strings.Contains(s, "You can't reach that target.") || replay().Tick != 9 {
		t.Fatalf("pronoun followed a different spoon: %s", s)
	}
}
