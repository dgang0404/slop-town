// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"testing"
)

func TestWitnessRemembersWhereAnItemWasPut(t *testing.T) {
	const player, baker, outside = "bakery/player", "bakery/baker", "bakery/miller"
	const crate, spoon, other = "bakery/crate", "bakery/spoon", "bakery/other-spoon"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	at := b.Entities[crate].Placement
	for _, id := range []string{player, baker} {
		actor := b.Entities[id]
		actor.Placement = at
		b.Entities[id] = actor
	}
	b.Entities[outside] = SpatialEntity{Name: "miller", Kind: Actor, Placement: Placement{Cell: CellPosition{FloorID: b.FloorID}}}
	b.Entities[other] = SpatialEntity{Name: b.Entities[spoon].Name, Kind: Item, Portable: true, Placement: Placement{HolderID: baker}}
	for i, action := range []ActionKind{ActionOpen, ActionTake, ActionPut, ActionTake, ActionDrop} {
		args := ActionArgs{Target: spoon}
		if action == ActionOpen {
			args.Target = crate
		}
		if action == ActionPut {
			args.Container = crate
		}
		r := c.Submit(baker, spatialProposal(c, fmt.Sprint(i), action, args))
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		view, err := c.Observe(player)
		if err != nil || view.hasTarget(other) || view.hasTarget(spoon) == (action == ActionTake) {
			t.Fatal("unexpected view of the baker's inventory", view, err)
		}
	}
	pages, err := c.Journal(player)
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	for _, page := range pages {
		for _, act := range page.Witnessed {
			if act.Actor.Ref == baker && act.Target.Ref == spoon {
				seen = append(seen, string(act.Action)+" "+act.ObjectPhrase())
				if act.Action == ActionPut && act.Container.Ref != crate {
					t.Fatal(act)
				}
			}
		}
	}
	want := []string{"take the wooden spoon", "put the wooden spoon in the crate", "take the wooden spoon", "drop the wooden spoon"}
	if !slices.Equal(seen, want) {
		t.Fatal(seen)
	}
	for _, sight := range c.bakery.Notebooks[outside] {
		if sight.Target == spoon {
			t.Fatal("a witness behind the wall saw the spoon", sight)
		}
	}

	// Restore validates the remembered destination even after the item moved.
	b = c.bakery
	raw, err := b.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreBakery(raw, c.scope.WorldID, c.clock); err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(b.Notebooks[player], func(e NotebookEntry) bool { return e.Action == ActionPut })
	b.Notebooks[player][i].Container = baker
	if err := b.validateNotebooks(c.clock.Tick()); err == nil {
		t.Fatal("accepted an actor as the destination container")
	}
}

func TestPutNeedsAVisibleActorAndDestination(t *testing.T) {
	const player, baker, crate, spoon = "bakery/player", "bakery/baker", "bakery/crate", "bakery/spoon"
	for _, atDoor := range []bool{true, false} {
		c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
		if err != nil {
			t.Fatal(err)
		}
		b := c.bakery
		door := b.Entities["bakery/door"]
		door.Open = true
		b.Entities["bakery/door"] = door
		observer, actor, box, item := b.Entities[player], b.Entities[baker], b.Entities[crate], b.Entities[spoon]
		observer.Placement.Cell.X, observer.Placement.Cell.Y = 0, 3
		actor.Placement, box.Placement = door.Placement, door.Placement
		if atDoor {
			box.Placement.Cell.X++
		} else {
			actor.Placement.Cell.X++
		}
		box.Open = true
		item.Placement = Placement{HolderID: baker}
		b.Entities[player], b.Entities[baker], b.Entities[crate], b.Entities[spoon] = observer, actor, box, item
		view, err := c.Observe(player)
		if err != nil || view.hasTarget(baker) != atDoor || view.hasTarget(crate) == atDoor || view.hasTarget(spoon) {
			t.Fatal("unexpected corner sight line", view, err)
		}
		r := c.Submit(baker, spatialProposal(c, "put", ActionPut, ActionArgs{Target: spoon, Container: crate}))
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		for _, e := range c.bakery.Notebooks[player] {
			if e.Action == ActionPut {
				t.Fatal("inferred who put the spoon away from a partial view", e)
			}
		}
	}
}
