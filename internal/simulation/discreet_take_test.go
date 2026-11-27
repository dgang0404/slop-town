// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"strings"
	"testing"
)

func TestDiscreetTakeWitnesses(t *testing.T) {
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	const crate, spoon = "bakery/crate", "bakery/spoon"
	for _, taker := range []string{player, baker} {
		t.Run(taker, func(t *testing.T) {
			c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
			if err != nil {
				t.Fatal(err)
			}
			b := c.bakery
			door := b.Entities["bakery/door"]
			door.Open = true
			b.Entities["bakery/door"] = door
			box := b.Entities[crate]
			box.Open = true
			b.Entities[crate] = box
			at := box.Placement
			near := at
			near.Cell.X++
			for _, id := range []string{player, baker, porter} {
				a := b.Entities[id]
				a.Placement = at
				b.Entities[id] = a
			}
			b.Entities["bakery/clerk"] = SpatialEntity{Name: "clerk", Kind: Actor, Placement: near}
			b.Entities["bakery/visitor"] = SpatialEntity{Name: "visitor", Kind: Actor, Placement: near}
			b.Entities["bakery/miller"] = SpatialEntity{Name: "miller", Kind: Actor, Placement: Placement{Cell: CellPosition{FloorID: b.FloorID}}}
			landmark := crate
			if taker == baker {
				item := b.Entities[spoon]
				item.Placement = at
				b.Entities[spoon] = item
				landmark = ""
			}
			p := spatialProposal(c, "pickup", ActionDiscreetTake, ActionArgs{Target: spoon})
			r := c.Submit(taker, p)
			if r.Status != "accepted" || c.clock.Tick() != 1 || c.bakery.Entities[spoon].Placement != (Placement{HolderID: taker}) {
				t.Fatal(r)
			}
			full := NotebookEntry{Tick: 1, SourceSequence: 2, Action: ActionTake, Subject: taker, Target: spoon}
			partial := NotebookEntry{Tick: 1, SourceSequence: 2, Action: SightHandMovement, Container: landmark}
			other, otherSaw := baker, full
			if taker == baker {
				other, otherSaw = player, partial
			}
			for _, tc := range []struct {
				id    string
				draws uint64
				entry NotebookEntry
			}{
				{"bakery/clerk", 1, full}, {"bakery/visitor", 1, partial}, {other, 1, otherSaw},
				{porter, 1, NotebookEntry{}}, {"bakery/miller", 0, NotebookEntry{}}, {taker, 0, NotebookEntry{}},
			} {
				e, found := c.bakery.firsthandAt(tc.id, 2)
				if found != (tc.entry.Action != "") || e != tc.entry || c.clock.draws["perception/"+tc.id] != tc.draws {
					t.Errorf("%s recorded %+v, present %v, draws %d", tc.id, e, found, c.clock.draws["perception/"+tc.id])
				}
			}
			journal, err := c.Journal(other)
			if err != nil || len(journal) != 1 || !slices.Contains(journal[0].Witnessed, c.bakery.witnessedAction(otherSaw)) {
				t.Fatal(journal, err)
			}
			submitUnchanged(t, c, taker, p)
			// Even a saved partial account cannot carry a hidden identity.
			entry := &c.bakery.Notebooks["bakery/visitor"][0]
			for _, bad := range []NotebookEntry{{Subject: taker}, {Target: spoon}, {Container: taker}} {
				*entry = partial
				entry.Subject, entry.Target, entry.Container = bad.Subject, bad.Target, bad.Container
				if err := c.bakery.validateNotebooks(1); err == nil {
					t.Fatal("accepted invalid partial sighting", *entry)
				}
			}
		})
	}
}

func TestDiscreetTakeAtTheDoorway(t *testing.T) {
	const baker, crate, spoon, witness = "bakery/baker", "bakery/crate", "bakery/spoon", "bakery/cobbler"
	for _, actorAtDoor := range []bool{true, false} {
		c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
		if err != nil {
			t.Fatal(err)
		}
		b := c.bakery
		door := b.Entities["bakery/door"]
		door.Open = true
		b.Entities["bakery/door"] = door
		actor, box := b.Entities[baker], b.Entities[crate]
		box.Open = true
		actor.Placement, box.Placement = door.Placement, door.Placement
		if actorAtDoor {
			box.Placement.Cell.X++
		} else {
			actor.Placement.Cell.X++
		}
		b.Entities[baker], b.Entities[crate] = actor, box
		b.Entities[witness] = SpatialEntity{Name: "cobbler", Kind: Actor, Placement: Placement{Cell: CellPosition{FloorID: b.FloorID, X: 0, Y: 3}}}
		view, err := c.Observe(witness)
		if err != nil || view.hasTarget(baker) != actorAtDoor || view.hasTarget(spoon) == actorAtDoor {
			t.Fatal(view, err)
		}
		r := c.Submit(baker, spatialProposal(c, "reach", ActionDiscreetTake, ActionArgs{Target: spoon}))
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		e, found := c.bakery.firsthandAt(witness, 2)
		if actorAtDoor {
			// This witness would identify the pickup if the item were in sight.
			if !found || e != (NotebookEntry{Tick: 1, SourceSequence: 2, Action: SightHandMovement}) {
				t.Fatal(e)
			}
		} else if found || c.clock.draws["perception/"+witness] != 0 {
			t.Fatal("seeing the item disappear identified an unseen actor", e)
		}
	}
}

func TestResidentPickupUsesLightAtTheTimeOfTheReach(t *testing.T) {
	const baker, player, porter = "bakery/baker", "bakery/player", "bakery/porter"
	const doorID, spoon = "bakery/door", "bakery/spoon"
	for _, tc := range []struct {
		name       string
		version    int
		open       bool
		action     ActionKind
		wantDim    bool
		wantMemory ActionKind
	}{
		{"daylight", daylightContentVersion, true, ActionDiscreetTake, false, SightHandMovement},
		{"closed until the porter opens it", daylightContentVersion, false, ActionDiscreetTake, true, ""},
		{"ordinary pickup in dim light", daylightContentVersion, false, ActionTake, true, ActionTake},
		{"older bakery with its door closed", dropSoundContentVersion, false, ActionDiscreetTake, false, SightHandMovement},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
			if err != nil {
				t.Fatal(err)
			}
			b := c.bakery
			b.ContentVersion = tc.version
			door := b.Entities[doorID]
			door.Open = tc.open
			b.Entities[doorID] = door
			// From the street the original player cannot sense indoor light.
			if outside, err := c.Scene(player); err != nil || outside.Self.Light != "" || (!tc.open && outside.hasTarget(baker)) {
				t.Fatal(outside, err)
			}
			prep := door.Placement
			prep.Cell.X++
			for _, id := range []string{baker, player, porter, spoon} {
				entity := b.Entities[id]
				entity.Placement = prep
				b.Entities[id] = entity
			}
			view, err := c.Scene(player)
			if err != nil || (view.Self.Light == DimLight) != tc.wantDim || !view.hasTarget(baker) || !view.hasTarget(spoon) {
				t.Fatal(view, err)
			}
			p := spatialProposal(c, "reach", tc.action, ActionArgs{Target: spoon})
			r := c.Submit(baker, p)
			if r.Status != "accepted" || !c.bakery.Entities[doorID].Open || strings.Contains(r.Observation.Text, DimLight.Description()) {
				t.Fatal(r)
			}
			memory, found := c.bakery.firsthandAt(player, 2)
			if found != (tc.wantMemory != "") || memory.Action != tc.wantMemory {
				t.Fatalf("player remembered %+v, want %q", memory, tc.wantMemory)
			}
			if tc.action == ActionDiscreetTake && c.clock.draws["perception/"+player] != 1 {
				t.Fatal("the pickup should use one perception draw in either light")
			}
			submitUnchanged(t, c, baker, p)
		})
	}
}

func TestDiscreetTakeNeedsAnAvailableItemAndNewPracticeTimeline(t *testing.T) {
	const player, spoon, crate = "bakery/player", "bakery/spoon", "bakery/crate"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 4)
	if err != nil {
		t.Fatal(err)
	}
	actor := c.bakery.Entities[player]
	box := c.bakery.Entities[crate]
	actor.Placement = box.Placement
	c.bakery.Entities[player] = actor
	option := ActionOption{Action: ActionDiscreetTake, Args: ActionArgs{Target: "wooden spoon"}}
	check := func(available bool) {
		t.Helper()
		menu, err := c.AvailableActions(player)
		if err != nil || slices.Contains(menu, option) != available {
			t.Fatal(menu, err)
		}
	}
	check(false)
	p := spatialProposal(c, "try", ActionDiscreetTake, ActionArgs{Target: spoon})
	if r := submitUnchanged(t, c, player, p); r.Status != "rejected" {
		t.Fatal(r)
	}
	box.Open = true
	c.bakery.Entities[crate] = box
	check(true)
	c.bakery.ContentVersion = itemWitnessContentVersion
	check(false)
	if r := submitUnchanged(t, c, player, p); r.Code != "action_unavailable" {
		t.Fatal(r)
	}
}
