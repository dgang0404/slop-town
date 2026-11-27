// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"testing"
)

func TestMintGrowsOnTownTime(t *testing.T) {
	const mint, player, baker = "bakery/mint", "bakery/player", "bakery/baker"
	scope := Scope{"bakery", "main"}
	for _, seed := range []int64{42, 43, 44} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, seed)
			if err != nil {
				t.Fatal(err)
			}
			water := spatialProposal(c, "watering", ActionWater, ActionArgs{Target: mint})
			if r := submitUnchanged(t, c, baker, water); r.Code != "target_unavailable" {
				t.Fatal("worker watered the plant from behind the closed door", r)
			}
			worker := c.bakery.Entities[baker]
			worker.Placement = c.bakery.Entities[player].Placement
			c.bakery.Entities[baker] = worker
			// A whole dry day passes even though the plant is in sight.
			wait := func() Result {
				t.Helper()
				r := c.Submit(player, spatialProposal(c, fmt.Sprintf("wait-%d", c.clock.Tick()), ActionWait, ActionArgs{}))
				if r.Status != "accepted" {
					t.Fatal(r)
				}
				return r
			}
			for range 24 {
				if got := wait().Observation.Plants[mint]; got != soilDry {
					t.Fatal("dry mint grew", got)
				}
			}
			// Put the resident back at the planter after their day's work.
			worker = c.bakery.Entities[baker]
			worker.Placement = c.bakery.Entities[player].Placement
			c.bakery.Entities[baker] = worker
			r := c.Submit(baker, water)
			if r.Status != "accepted" || r.Observation.Plants[mint] != soilDamp {
				t.Fatal(r)
			}
			g := c.bakery.Entities[mint].Growth
			cause := c.History()[g.WateredSequence-1]
			if cause.Kind != "plant_watered" || cause.ActorID != baker || cause.Tick != g.WateredTick {
				t.Fatal(cause, g)
			}
			// A caller can edit its view without changing the receipt or the clock.
			r.Observation.Plants[mint] = newLeaves
			if retry := submitUnchanged(t, c, baker, water); retry.Observation.Plants[mint] != soilDamp {
				t.Fatal(retry)
			}
			if again := submitUnchanged(t, c, player, spatialProposal(c, "again", ActionWater, water.Args)); again.Code != "already_in_state" {
				t.Fatal(again)
			}
			delay := c.bakery.leafDelay(mint)
			for i := int64(1); i < delay; i++ {
				if got := wait().Observation.Plants[mint]; got != soilDamp {
					t.Fatal("mint grew early", i, got)
				}
			}
			for range 3 {
				if got := wait().Observation.Plants[mint]; got != newLeaves {
					t.Fatal("missing leaves", got)
				}
			}
			var growth []Event
			for _, e := range c.History() {
				if e.Kind == "leaves_unfolded" {
					growth = append(growth, e)
				}
			}
			if len(growth) != 1 || growth[0].Tick != cause.Tick+delay || growth[0].ParentID != cause.ID ||
				growth[0].ActorID != mint || growth[0].Payload.Target != mint {
				t.Fatal(growth)
			}
			menu, err := c.AvailableActions(player)
			if err != nil || slices.ContainsFunc(menu, func(o ActionOption) bool { return o.Action == ActionWater }) {
				t.Fatal(menu, err)
			}
		})
	}
}

func TestRestorePlantGrowth(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const mint = "bakery/mint"
	p := spatialProposal(c, "water", ActionWater, ActionArgs{Target: mint})
	if r := c.Submit("bakery/player", p); r.Status != "accepted" {
		t.Fatal(r)
	}
	plant := c.bakery.Entities[mint]
	for _, bad := range []plantGrowth{
		{NewLeaves: true}, {WateredTick: 2, WateredSequence: 2}, {WateredTick: 1},
		{WateredTick: 1, WateredSequence: 2, NewLeaves: true},
	} {
		plant.Growth = bad
		c.bakery.Entities[mint] = plant
		data, err := c.bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = RestoreBakery(data, "bakery", c.clock); err == nil {
			t.Fatal("restored impossible plant history", bad)
		}
	}
}
