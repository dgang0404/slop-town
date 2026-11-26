// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"strings"
	"testing"
)

func TestWalkingPractice(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player = "bakery/player"
	submit := func(action ActionKind, dir Direction) Result {
		t.Helper()
		p := spatialProposal(c, fmt.Sprint(c.clock.Tick()+1), action, ActionArgs{Direction: dir})
		r := c.Submit(player, p)
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		return r
	}
	rest := func(want int) {
		t.Helper()
		r := submit(ActionRest, "")
		if actor := c.bakery.Entities[player]; actor.EnduranceBouts != want || actor.WalkedCells != 0 || actor.Fatigue != 0 || r.Message != "You rest for a turn." {
			t.Fatalf("after rest: %+v, %q", actor, r.Message)
		}
	}
	for range 3 {
		submit(ActionMove, West)
		submit(ActionMove, East)
	}
	rest(0) // Six steps between two cells.
	for walk := 1; walk <= 3; walk++ {
		tiredAt := 5
		if walk == 3 {
			tiredAt = 6
		}
		for i, dir := range []Direction{West, South, East, North, South, North} {
			r := submit(ActionMove, dir)
			if strings.Contains(r.Observation.Text, "Your legs feel tired.") != (i+1 >= tiredAt) {
				t.Fatalf("walk %d step %d: %s", walk, i+1, r.Observation.Text)
			}
		}
		rest(min(walk, trainedBouts))
		rest(min(walk, trainedBouts)) // Sitting again earns no additional practice.
	}
	for _, id := range []string{"bakery/porter", "bakery/baker"} {
		credits := 0
		for _, e := range c.History() {
			if e.ActorID == id && e.Payload.EndurancePractice {
				if e.Kind != "actor_rested" {
					t.Fatal(e)
				}
				credits++
			}
		}
		if credits == 0 || c.bakery.Entities[id].EnduranceBouts != credits {
			t.Fatalf("%s: got %d earned bouts", id, credits)
		}
	}
}

func TestInvalidWalkingPractice(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const id = "bakery/player"
	for _, tc := range []struct {
		bouts, fatigue int
		cells          uint32
	}{
		{3, 0, 0}, {0, 1, 0}, {0, 0, 2}, {2, 1, 2}, {1, 5, 1 << 30}, {1, 5, 1<<2 | walkingCell(c.bakery.Entities[id].Placement.Cell)},
	} {
		actor := c.bakery.Entities[id]
		actor.EnduranceBouts, actor.Fatigue, actor.WalkedCells = tc.bouts, tc.fatigue, tc.cells
		c.bakery.Entities[id] = actor
		data, err := c.bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(data, "bakery", c.clock); err == nil {
			t.Fatalf("accepted invalid practice %+v", tc)
		}
	}
}
