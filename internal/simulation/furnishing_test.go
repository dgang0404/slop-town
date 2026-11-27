// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"reflect"
	"testing"
)

func TestGeneratedBakeryAccessAndVariety(t *testing.T) {
	setup := BakerySetup{Generated: true}
	locations := make(map[[4]CellPosition]bool)
	supplies := make(map[string]bool)
	const prefix = "test%2Ftown/"
	for seed := int64(0); seed < 100; seed++ {
		clock := NewTimeline(seed)
		b, err := setup.Generate("test/town", clock)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		other := NewTimeline(seed)
		draw(t, other, "weather")
		repeat, err := setup.Generate("test/town", other)
		if err != nil || !reflect.DeepEqual(b, repeat) || clock.Tick() != 0 || clock.draws["furnishing"] != 5 {
			t.Fatalf("seed %d changed on repeat or consumed the wrong draws: %v", seed, err)
		}
		var layout [4]CellPosition
		for i, id := range []string{"bakery-phone", "crate", "street-phone", "mint"} {
			layout[i] = b.Entities[prefix+id].Placement.Cell
		}
		locations[layout] = true
		prep := b.Entities[prefix+"baker"].Placement.Cell
		contents := 0
		for id, e := range b.Entities {
			if e.Placement.HolderID == prefix+"crate" {
				contents++
				if e.Kind != Item || !e.Portable || e.Name == "" || e.CloseDetail == "" {
					t.Fatalf("seed %d has unusable supply %s", seed, id)
				}
				if id != prefix+"spoon" {
					supplies[e.Name] = true
				}
			}
		}
		if contents != 2 || !inReach(prep, layout[0]) || layout[0] == layout[1] || layout[2] == layout[3] {
			t.Fatalf("bad workstations or crate for seed %d: %v", seed, layout)
		}
		// Play both opening actions through ordinary movement and reach rules.
		scope := Scope{"test/town", "main"}
		player := prefix + "player"
		act := func(kind ActionKind, args ActionArgs) {
			t.Helper()
			effect, fail := b.resolveSpatial(player, Proposal{Scope: scope, Action: kind, Args: args})
			if fail != nil {
				t.Fatalf("seed %d, %s: %v", seed, kind, fail)
			}
			b.Entities[effect.entityID] = effect.entity
		}
		act(ActionOpen, ActionArgs{Target: prefix + "door"})
		for x := b.Entities[player].Placement.Cell.X; x < layout[1].X; x++ {
			act(ActionMove, ActionArgs{Direction: East})
		}
		act(ActionOpen, ActionArgs{Target: prefix + "crate"})
		act(ActionTake, ActionArgs{Target: prefix + "spoon"})
	}
	if len(locations) < 2 || len(supplies) != 3 {
		t.Fatalf("got %d layouts and supplies %v", len(locations), supplies)
	}
	t.Logf("100 seeds produced %d distinct furniture layouts and %d supplies", len(locations), len(supplies))
}

func TestFurnishingRejectsBlockedAccessAndPreservesClockOnFailure(t *testing.T) {
	setup := BakerySetup{Generated: true}
	clock := NewTimeline(42)
	draw(t, clock, "furnishing")
	before := snapshot(t, clock)
	if _, err := setup.Generate("town", clock); err == nil || !bytes.Equal(before, snapshot(t, clock)) {
		t.Fatal("failed generation changed the clock")
	}
	b, err := setup.Generate("town", NewTimeline(42))
	if err != nil {
		t.Fatal(err)
	}
	// Isolate the interior despite a reachable front door.
	prep := b.Entities["town/baker"].Placement.Cell
	for y := range b.Cells {
		b.Cells[y][prep.X].Wall = true
	}
	if err := b.checkFurnishingAccess("town/"); err == nil {
		t.Fatal("accepted a disconnected shop")
	}
}
