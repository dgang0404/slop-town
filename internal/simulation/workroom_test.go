// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"strconv"
	"testing"
)

func TestWorkroomRoundTripAcrossSeeds(t *testing.T) {
	const player = "bakery/player"
	for seed := int64(0); seed < 100; seed++ {
		c, err := (BakerySetup{Workroom: true}).NewCoordinator(Scope{"bakery", "main"}, seed)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		i := 0
		act := func(kind ActionKind, args ActionArgs) {
			t.Helper()
			i++
			r := c.Submit(player, spatialProposal(c, strconv.Itoa(i), kind, args))
			if r.Status != "accepted" {
				t.Fatalf("seed %d, %s: %v", seed, kind, r)
			}
		}
		walk := func(dir Direction) { act(ActionMove, ActionArgs{Direction: dir}) }
		toward, back := South, North
		if c.bakery.Entities["bakery/door"].Placement.Cell.Y == 2 {
			toward, back = North, South
		}
		act(ActionOpen, ActionArgs{Target: "bakery/door"})
		for range 3 {
			walk(East)
		}
		view, err := c.Scene(player)
		if err != nil || view.hasTarget("bakery/workbench") {
			t.Fatalf("seed %d: closed room leaked", seed)
		}
		act(ActionOpen, ActionArgs{Target: "bakery/workroom-door"})
		walk(toward)
		walk(toward)
		view, err = c.Scene(player)
		if err != nil || view.Self.Place != InWorkroom || !view.hasTarget("bakery/workbench") {
			t.Fatal(seed, view, err)
		}
		act(ActionClose, ActionArgs{Target: "bakery/workroom-door"})
		act(ActionOpen, ActionArgs{Target: "bakery/workbench"})
		act(ActionRest, ActionArgs{})
		act(ActionOpen, ActionArgs{Target: "bakery/workroom-door"})
		walk(back)
		walk(back)
		for range 3 {
			walk(West)
		}
		view, err = c.Scene(player)
		if err != nil || view.Self.Place != OnStreet {
			t.Fatal(seed, err)
		}
	}
}
