// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"maps"
	"testing"
)

const (
	playerID = "town/player"
	doorID   = "town/door"
	crateID  = "town/crate"
	spoonID  = "town/spoon"
)

func commitBakeryAction(t *testing.T, c *Coordinator, actor string, action ActionKind, args ActionArgs) (Proposal, Result) {
	t.Helper()
	tick := c.clock.Tick() + 1
	p := spatialProposal(c, fmt.Sprintf("step-%d", tick), action, args)
	r := c.Submit(actor, p)
	if r.Status != "accepted" || r.Observation.ObserverID != actor || c.clock.Tick() != tick {
		t.Fatalf("tick %d %s %+v: %+v", tick, action, args, r)
	}
	if err := c.bakery.Validate(); err != nil {
		t.Fatalf("tick %d: %v", tick, err)
	}
	return p, r
}

func enterBakery(t *testing.T, c *Coordinator, actor string) {
	t.Helper()
	commitBakeryAction(t, c, actor, ActionOpen, ActionArgs{Target: doorID})
	commitBakeryAction(t, c, actor, ActionMove, ActionArgs{Direction: East})
	commitBakeryAction(t, c, actor, ActionMove, ActionArgs{Direction: East})
}

func checkSpoon(t *testing.T, c *Coordinator, parent Placement, cell CellPosition) {
	t.Helper()
	pos, err := c.bakery.PositionOf(spoonID)
	if err != nil || pos != cell || c.bakery.Entities[spoonID].Placement != parent {
		t.Fatalf("spoon placement %+v at %+v, want %+v at %+v (error %v)", c.bakery.Entities[spoonID].Placement, pos, parent, cell, err)
	}
}

func rejectBakeryAction(t *testing.T, c *Coordinator, actor string, action ActionKind, args ActionArgs, code string) Result {
	t.Helper()
	r := submitUnchanged(t, c, actor, spatialProposal(c, "rejected", action, args))
	if r.Status != "rejected" || r.Code != code {
		t.Fatalf("%s %+v: got %+v, want %s", action, args, r, code)
	}
	return r
}

func TestBakeryInventoryWalkthrough(t *testing.T) {
	for _, tc := range []struct {
		seed int64
		row  int
	}{{42, 1}, {43, 2}} {
		t.Run(fmt.Sprint(tc.seed), func(t *testing.T) {
			c := bakeryCoordinator(t, tc.seed)
			street := CellPosition{c.bakery.FloorID, 1, tc.row}
			interior := CellPosition{c.bakery.FloorID, 3, tc.row}
			enterBakery(t, c, playerID)
			commitBakeryAction(t, c, playerID, ActionOpen, ActionArgs{Target: crateID})
			take, receipt := commitBakeryAction(t, c, playerID, ActionTake, ActionArgs{Target: spoonID})
			checkSpoon(t, c, Placement{HolderID: playerID}, interior)
			commitBakeryAction(t, c, playerID, ActionClose, ActionArgs{Target: crateID})
			commitBakeryAction(t, c, playerID, ActionMove, ActionArgs{Direction: West})
			commitBakeryAction(t, c, playerID, ActionMove, ActionArgs{Direction: West})
			commitBakeryAction(t, c, playerID, ActionClose, ActionArgs{Target: doorID})
			checkSpoon(t, c, Placement{HolderID: playerID}, street)
			commitBakeryAction(t, c, playerID, ActionDrop, ActionArgs{Target: spoonID})
			checkSpoon(t, c, Placement{Cell: street}, street)
			if c.clock.Tick() != 10 || c.bakery.Entities[doorID].Open || c.bakery.Entities[crateID].Open ||
				len(c.bakery.Entities) != 4 || !maps.Equal(c.clock.draws, map[string]uint64{"geography": 1}) {
				t.Fatal("unexpected final bakery state")
			}
			history := c.History()
			if len(history) != 20 {
				t.Fatalf("got %d events, want 20", len(history))
			}
			for _, want := range []struct {
				tick int
				kind string
				data EventPayload
			}{
				{4, "container_opened", EventPayload{Target: crateID}},
				{5, "item_taken", EventPayload{Target: spoonID, HolderID: playerID}},
				{6, "container_closed", EventPayload{Target: crateID}},
				{10, "item_dropped", EventPayload{Target: spoonID, Destination: street}},
			} {
				parent, e := history[2*want.tick-2], history[2*want.tick-1]
				if e.Tick != int64(want.tick) || e.Kind != want.kind || e.Payload != want.data ||
					parent.Kind != "tick_advanced" || e.ParentID != parent.ID || e.ActorID != playerID {
					t.Fatalf("tick %d: unexpected event %+v", want.tick, e)
				}
			}
			if got := submitUnchanged(t, c, playerID, take); got != receipt {
				t.Fatalf("late take retry returned %+v, want %+v", got, receipt)
			}
		})
	}
}

func TestBakeryInventoryPutBack(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	enterBakery(t, c, playerID)
	spoon, crate := ActionArgs{Target: spoonID}, ActionArgs{Target: crateID}
	put := ActionArgs{Target: spoonID, Container: crateID}
	rejectBakeryAction(t, c, playerID, ActionTake, crate, "not_portable")
	hidden := rejectBakeryAction(t, c, playerID, ActionTake, spoon, "target_unavailable")
	missing := rejectBakeryAction(t, c, playerID, ActionTake, ActionArgs{Target: "town/absent"}, "target_unavailable")
	if hidden != missing {
		t.Fatal("closed crate disclosed its contents")
	}
	commitBakeryAction(t, c, playerID, ActionOpen, crate)
	rejectBakeryAction(t, c, playerID, ActionPut, put, "target_unavailable")
	rejectBakeryAction(t, c, playerID, ActionDrop, spoon, "target_unavailable")
	commitBakeryAction(t, c, playerID, ActionTake, spoon)
	commitBakeryAction(t, c, playerID, ActionClose, crate)
	rejectBakeryAction(t, c, playerID, ActionPut, put, "target_unavailable")
	commitBakeryAction(t, c, playerID, ActionOpen, crate)
	commitBakeryAction(t, c, playerID, ActionMove, ActionArgs{Direction: East})
	rejectBakeryAction(t, c, playerID, ActionPut, put, "target_unavailable")
	commitBakeryAction(t, c, playerID, ActionMove, ActionArgs{Direction: West})
	p, receipt := commitBakeryAction(t, c, playerID, ActionPut, put)
	checkSpoon(t, c, Placement{HolderID: crateID}, c.bakery.Entities[crateID].Placement.Cell)
	history := c.History()
	if e := history[len(history)-1]; e.Kind != "item_put" || e.Payload != (EventPayload{Target: spoonID, HolderID: crateID}) {
		t.Fatalf("unexpected put event: %+v", e)
	}
	commitBakeryAction(t, c, playerID, ActionClose, crate)
	if got := submitUnchanged(t, c, playerID, p); got != receipt {
		t.Fatalf("late put retry: %+v", got)
	}
	p.Args.Container = doorID
	if got := submitUnchanged(t, c, playerID, p); got.Code != "request_conflict" {
		t.Fatalf("changed destination on retry: %+v", got)
	}
}

func TestBakeryInventoryReachAndOwnership(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const baker = "town/baker"
	c.bakery.Entities[baker] = c.bakery.Entities[playerID]
	enterBakery(t, c, baker)
	spoon := ActionArgs{Target: spoonID}
	commitBakeryAction(t, c, baker, ActionOpen, ActionArgs{Target: crateID})
	rejectBakeryAction(t, c, playerID, ActionTake, spoon, "target_unavailable")
	commitBakeryAction(t, c, baker, ActionTake, spoon)
	commitBakeryAction(t, c, baker, ActionDrop, spoon)
	rejectBakeryAction(t, c, playerID, ActionTake, spoon, "target_unavailable")
	commitBakeryAction(t, c, playerID, ActionMove, ActionArgs{Direction: East})
	commitBakeryAction(t, c, playerID, ActionTake, spoon)
	rejectBakeryAction(t, c, baker, ActionTake, spoon, "target_unavailable")
	rejectBakeryAction(t, c, baker, ActionPut, ActionArgs{Target: spoonID, Container: crateID}, "target_unavailable")
	rejectBakeryAction(t, c, baker, ActionDrop, spoon, "target_unavailable")
	rejectBakeryAction(t, c, playerID, ActionPut, spoon, "invalid_request")
	rejectBakeryAction(t, c, playerID, ActionDrop, ActionArgs{Target: spoonID, Container: crateID}, "invalid_request")
}
