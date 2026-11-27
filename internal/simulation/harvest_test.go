// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"testing"
)

func TestHarvestAtThePlanter(t *testing.T) {
	const planter, player, baker, porter = "bakery/mint", "bakery/player", "bakery/baker", "bakery/porter"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	harvest := spatialProposal(c, "pick", ActionHarvest, ActionArgs{Target: planter})
	checkMenu := func(actor string, want bool) {
		t.Helper()
		menu, err := c.AvailableActions(actor)
		if err != nil || slices.Contains(menu, ActionOption{ActionHarvest, ActionArgs{Target: "mint planter"}}) != want {
			t.Fatalf("harvest option for %s, want %t: %v, %v", actor, want, menu, err)
		}
	}
	checkUnripe := func() {
		t.Helper()
		checkMenu(player, false)
		if r := submitUnchanged(t, c, player, harvest); r.Code != "action_unavailable" {
			t.Fatal(r)
		}
	}
	checkUnripe()
	if r := c.Submit(player, spatialProposal(c, "water", ActionWater, harvest.Args)); r.Status != "accepted" {
		t.Fatal(r)
	}
	for i := int64(0); i < c.bakery.leafDelay(planter); i++ {
		checkUnripe()
		if r := c.Submit(player, spatialProposal(c, fmt.Sprint(i), ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	usePracticeVersion(c.bakery, plantContentVersion)
	checkMenu(player, false)
	if r := submitUnchanged(t, c, player, harvest); r.Status != "rejected" {
		t.Fatal("old town allowed a harvest", r)
	}
	c.bakery.ContentVersion = harvestContentVersion
	checkMenu(player, true)
	// The player can see the baker across the open doorway. Only someone
	// beside the planter can take the cutting. The porter is around the corner.
	worker := c.bakery.Entities[baker]
	worker.Placement = c.bakery.Entities["bakery/door"].Placement
	c.bakery.Entities[baker] = worker
	scene, err := c.Scene(baker)
	if err != nil || !scene.hasTarget(planter) {
		t.Fatal("expected a visible planter from the doorway", scene, err)
	}
	checkMenu(baker, false)
	if r := submitUnchanged(t, c, baker, harvest); r.Code != "target_unavailable" {
		t.Fatal(r)
	}
	worker.Placement = c.bakery.Entities[player].Placement
	c.bakery.Entities[baker] = worker
	other := c.bakery.Entities[porter]
	other.Placement.Cell.X, other.Placement.Cell.Y = 4, 3
	c.bakery.Entities[porter] = other
	scene, err = c.Scene(porter)
	if err != nil || scene.hasTarget(planter) {
		t.Fatal("expected planter to be out of sight", scene, err)
	}
	checkMenu(baker, true)
	start := len(c.History())
	r := c.Submit(baker, harvest)
	yield := sprigID(planter)
	if r.Status != "accepted" || r.SelectedTarget != yield || !r.Observation.hasTarget(yield) || r.Observation.Plants[planter] != clippedStems {
		t.Fatal(r)
	}
	event := c.History()[start+1]
	if event.Kind != "plant_harvested" || event.Tick != 8 || event.Payload != (EventPayload{Target: planter, ItemID: yield, HolderID: baker}) {
		t.Fatal(event)
	}
	item := c.bakery.Entities[yield]
	if item.Kind != Item || !item.Portable || item.Placement != (Placement{HolderID: baker}) {
		t.Fatal(item)
	}
	if e, ok := c.bakery.entryAt(player, event.Sequence); !ok || e.Action != ActionHarvest || e.Subject != baker || e.Target != planter {
		t.Fatal("missing visible harvest", e)
	}
	if _, ok := c.bakery.entryAt(porter, event.Sequence); ok {
		t.Fatal("harvest reported through a wall")
	}
	scene, err = c.Scene(player)
	if err != nil || scene.hasTarget(yield) || scene.Plants[planter] != clippedStems {
		t.Fatal("observer saw inside the baker's inventory or missed the clipped stems", scene, err)
	}
	checkMenu(player, false)
	if r := submitUnchanged(t, c, player, harvest); r.Code != "already_in_state" {
		t.Fatal("second actor harvested the same cutting", r)
	}
	if r := submitUnchanged(t, c, baker, harvest); r.Status != "accepted" || r.SelectedTarget != yield {
		t.Fatal("retry lost its original cutting", r)
	}
	// A save cannot have the stems without their sprig, or a sprig that the plant never yielded.
	delete(c.bakery.Entities, yield)
	if err := c.bakery.Validate(); err == nil {
		t.Fatal("accepted a missing sprig")
	}
	c.bakery.Entities[yield] = item
	plant := c.bakery.Entities[planter]
	plant.Growth.Harvested = false
	c.bakery.Entities[planter] = plant
	if err := c.bakery.Validate(); err == nil {
		t.Fatal("accepted a sprig from an unharvested planter")
	}
}
