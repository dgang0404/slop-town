// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"testing"
)

func TestSecondMintHarvest(t *testing.T) {
	const planter, player = "bakery/mint", "bakery/player"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.ContentVersion < bakingContentVersion {
		t.Fatalf("content version %d", c.bakery.ContentVersion)
	}
	water := spatialProposal(c, "water", ActionWater, ActionArgs{Target: planter})
	harvest := spatialProposal(c, "pick", ActionHarvest, ActionArgs{Target: planter})
	if r := c.Submit(player, water); r.Status != "accepted" {
		t.Fatal(r)
	}
	delay := int(c.bakery.leafDelay(planter))
	for i := 0; i < delay; i++ {
		if r := c.Submit(player, spatialProposal(c, fmt.Sprintf("wait-%d", i), ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
		if i == delay-1 {
			scene, err := c.Scene(player)
			if err != nil || scene.Plants[planter] != newLeaves {
				t.Fatalf("after %d waits plant=%v err=%v", i+1, scene.Plants[planter], err)
			}
		}
	}
	first := sprigID(planter, 1)
	if r := c.Submit(player, harvest); r.Status != "accepted" || r.SelectedTarget != first {
		t.Fatal(r)
	}
	firstItem := c.bakery.Entities[first]
	regrowthDelay := int(c.bakery.regrowthDelay(planter))
	for i := 0; i < regrowthDelay-1; i++ {
		if r := c.Submit(player, spatialProposal(c, fmt.Sprintf("regrow-%d", i), ActionWait, ActionArgs{})); r.Status != "accepted" || r.Observation.Plants[planter] != clippedStems {
			t.Fatal("regrew early", i, r)
		}
	}
	if r := c.Submit(player, spatialProposal(c, "regrown", ActionWait, ActionArgs{})); r.Status != "accepted" || r.Observation.Plants[planter] != newLeaves {
		t.Fatal("missing second leaves", r)
	}
	second := sprigID(planter, 2)
	secondPick := spatialProposal(c, "pick-2", ActionHarvest, ActionArgs{Target: planter})
	if r := c.Submit(player, secondPick); r.Status != "accepted" || r.SelectedTarget != second || firstItem.Placement.HolderID != player {
		t.Fatal("second harvest changed the first sprig", r, firstItem.Placement)
	}
	if first == second || c.bakery.Entities[first].Placement.HolderID != player || c.bakery.Entities[second].Placement.HolderID != player {
		t.Fatal("expected two sprigs with the player")
	}
	if r := submitUnchanged(t, c, player, spatialProposal(c, "pick-3", ActionHarvest, ActionArgs{Target: planter})); r.Code != "already_in_state" {
		t.Fatal("third harvest", r)
	}
	var unfolds int
	for _, e := range c.History() {
		if e.Kind == "leaves_unfolded" {
			unfolds++
		}
	}
	if unfolds != 2 {
		t.Fatal("unexpected growth events", unfolds)
	}
}

func TestOlderPracticeAllowsOneHarvest(t *testing.T) {
	const planter, player = "bakery/mint", "bakery/player"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	usePracticeVersion(c.bakery, harvestContentVersion)
	harvest := spatialProposal(c, "pick", ActionHarvest, ActionArgs{Target: planter})
	if r := c.Submit(player, spatialProposal(c, "water", ActionWater, ActionArgs{Target: planter})); r.Status != "accepted" {
		t.Fatal(r)
	}
	for i := int64(0); i < c.bakery.leafDelay(planter); i++ {
		if r := c.Submit(player, spatialProposal(c, fmt.Sprint(i), ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	if r := c.Submit(player, harvest); r.Status != "accepted" {
		t.Fatal(r)
	}
	for range 12 {
		if r := c.Submit(player, spatialProposal(c, "wait", ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	if r := submitUnchanged(t, c, player, spatialProposal(c, "pick-2", ActionHarvest, ActionArgs{Target: planter})); r.Code != "already_in_state" {
		t.Fatal("old town allowed a second harvest", r)
	}
}