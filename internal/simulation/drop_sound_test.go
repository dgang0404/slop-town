// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"testing"
)

func TestResidentDropsSpoonAroundCorner(t *testing.T) {
	const baker, porter, player, far = "bakery/baker", "bakery/porter", "bakery/player", "bakery/customer"
	const doorID, spoon = "bakery/door", "bakery/spoon"
	for _, tc := range []struct {
		name string
		open bool
	}{{"open doorway", true}, {"door opens after drop", false}} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
			if err != nil {
				t.Fatal(err)
			}
			b := c.bakery
			door := b.Entities[doorID]
			door.Open = tc.open
			b.Entities[doorID] = door
			b.Entities[far] = SpatialEntity{Kind: Actor, Name: "customer"}
			for id, xy := range map[string][2]int{baker: {1, 2}, player: {3, 1}, porter: {3, 1}, far: {4, 3}} {
				a := b.Entities[id]
				a.Placement = Placement{Cell: CellPosition{FloorID: b.FloorID, X: xy[0], Y: xy[1]}}
				b.Entities[id] = a
			}
			item := b.Entities[spoon]
			item.Placement = Placement{HolderID: baker}
			b.Entities[spoon] = item
			if view, err := c.Observe(player); err != nil || view.hasTarget(baker) || view.hasTarget(spoon) {
				t.Fatal(view, err)
			}
			p := spatialProposal(c, "drop", ActionDrop, ActionArgs{Target: spoon})
			r := c.Submit(baker, p)
			if r.Status != "accepted" || !slices.Equal(r.Observation.Heard, []PerceptionKind{Clatter}) || !c.bakery.Entities[doorID].Open {
				t.Fatal(r)
			}
			drop := c.History()[1]
			var listeners []string
			for _, e := range c.History() {
				if e.Kind != "sound_heard" {
					continue
				}
				if e.ParentID != drop.ID || e.Payload != (EventPayload{Sound: Clatter}) {
					t.Fatal(e)
				}
				entry, ok := c.bakery.firsthandAt(e.ActorID, e.Sequence)
				if !ok || entry != (NotebookEntry{Tick: 1, SourceSequence: e.Sequence, Action: ActionListen, Sound: Clatter}) {
					t.Fatal(e.ActorID, entry)
				}
				listeners = append(listeners, e.ActorID)
			}
			want := []string{baker}
			if tc.open {
				want = []string{baker, player, porter}
			}
			if !slices.Equal(listeners, want) {
				t.Fatalf("heard by %v, want %v", listeners, want)
			}
			pages, err := c.Journal(player)
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, page := range pages {
				for _, o := range page.Observations {
					if o.Kind == Clatter {
						count++
					}
				}
				for _, sight := range page.Witnessed {
					if sight.Action == ActionDrop {
						t.Fatal("the listener identified an unseen drop", sight)
					}
				}
			}
			wantCount := 0
			if tc.open {
				wantCount = 1
			}
			if count != wantCount {
				t.Fatalf("journal recorded %d clatters, want %d", count, wantCount)
			}
			submitUnchanged(t, c, baker, p)
		})
	}
}

func TestImpactBelongsToTheDroppedItem(t *testing.T) {
	const player, spoon, other = "bakery/player", "bakery/spoon", "bakery/other"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	item := c.bakery.Entities[spoon]
	item.Placement = Placement{HolderID: player}
	c.bakery.Entities[spoon] = item
	item.DropSound = ""
	c.bakery.Entities[other] = item // Same visible name; only the original spoon clatters.
	for _, id := range []string{other, spoon} {
		r := c.Submit(player, spatialProposal(c, id, ActionDrop, ActionArgs{Target: id}))
		if r.Status != "accepted" || slices.Contains(r.Observation.Heard, Clatter) != (id == spoon) {
			t.Fatal(id, r)
		}
	}
}
