// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"testing"
)

func bakeryCoordinator(t *testing.T, seed int64) *Coordinator {
	t.Helper()
	c, err := NewBakeryCoordinator(Scope{"town", "main"}, seed)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func spatialProposal(c *Coordinator, id string, action ActionKind, args ActionArgs) Proposal {
	return Proposal{Scope: c.scope, Version: 1, RequestID: id, Action: action, Args: args}
}

// Retries and rejections must preserve the entire committed state.
func submitUnchanged(t *testing.T, c *Coordinator, actor string, p Proposal) Result {
	t.Helper()
	clock, history, receipts := snapshot(t, c.clock), c.History(), maps.Clone(c.receipts)
	state := *c.bakery
	state.Entities = maps.Clone(c.bakery.Entities)
	r := c.Submit(actor, p)
	if !bytes.Equal(clock, snapshot(t, c.clock)) || !slices.Equal(history, c.History()) ||
		!reflect.DeepEqual(&state, c.bakery) || !maps.Equal(receipts, c.receipts) {
		t.Fatalf("%s changed committed state: %+v", p.RequestID, r)
	}
	return r
}

func TestBakeryDoorWalk(t *testing.T) {
	for _, tc := range []struct {
		seed int64
		row  int
	}{{42, 1}, {43, 2}} {
		t.Run(fmt.Sprint(tc.seed), func(t *testing.T) {
			c := bakeryCoordinator(t, tc.seed)
			const actor = "town/player"
			door := ActionArgs{Target: "town/door"}
			east, west := ActionArgs{Direction: East}, ActionArgs{Direction: West}
			open := spatialProposal(c, "open", ActionOpen, door)
			enter := spatialProposal(c, "enter", ActionMove, east)
			closeDoor := spatialProposal(c, "close", ActionClose, door)
			if r := submitUnchanged(t, c, actor, enter); r.Code != "movement_blocked" {
				t.Fatalf("walked through a closed door: %+v", r)
			}
			steps := []struct {
				proposal Proposal
				kind     string
				x        int
				open     bool
			}{
				{open, "door_opened", 1, true},
				{enter, "actor_moved", 2, true},
				{spatialProposal(c, "inside", ActionMove, east), "actor_moved", 3, true},
				{spatialProposal(c, "leave", ActionMove, west), "actor_moved", 2, true},
				{spatialProposal(c, "outside", ActionMove, west), "actor_moved", 1, true},
				{closeDoor, "door_closed", 1, false},
			}
			var openResult Result
			for i, s := range steps {
				r := c.Submit(actor, s.proposal)
				pos := CellPosition{c.bakery.FloorID, s.x, tc.row}
				if r.Status != "accepted" || r.Observation.ObserverID != actor || r.Observation.Text == "" ||
					c.clock.Tick() != int64(i+1) || c.bakery.Entities[actor].Placement.Cell != pos ||
					c.bakery.Entities[door.Target].Open != s.open || len(c.receipts) != i+1 {
					t.Fatalf("step %d: unexpected result or world state: %+v", i+1, r)
				}
				history := c.History()
				if len(history) != 2*(i+1) {
					t.Fatalf("step %d has %d accumulated events", i+1, len(history))
				}
				advance, event := history[2*i], history[2*i+1]
				payload := EventPayload{Target: door.Target}
				if s.kind == "actor_moved" {
					payload = EventPayload{Destination: pos}
				}
				if advance.Kind != "tick_advanced" || advance.Payload.Delta != 1 || advance.Sequence != uint64(2*i+1) ||
					event.Sequence != advance.Sequence+1 || event.ParentID != advance.ID || event.Kind != s.kind ||
					event.Tick != int64(i+1) || event.ActorID != actor || event.RequestID != s.proposal.RequestID ||
					event.Engine != "spatial" || event.Payload != payload {
					t.Fatalf("step %d: incorrect causal events: %+v, %+v", i+1, advance, event)
				}
				if i == 0 {
					openResult = r
				}
				if s.x == 2 {
					if got := submitUnchanged(t, c, actor, closeDoor); got.Code != "doorway_occupied" {
						t.Fatalf("closed door on actor: %+v", got)
					}
				}
			}
			if r := submitUnchanged(t, c, actor, open); !reflect.DeepEqual(r, openResult) {
				t.Fatalf("late retry returned %+v, want %+v", r, openResult)
			}
			enter.Args.Direction = West
			if r := submitUnchanged(t, c, actor, enter); r.Code != "request_conflict" {
				t.Fatalf("accepted changed direction on retry: %+v", r)
			}
			if !maps.Equal(c.clock.draws, map[string]uint64{"geography": 1}) {
				t.Fatalf("unexpected draws: %v", c.clock.draws)
			}
		})
	}
}

func TestBakeryMovementLimits(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const actor = "town/player"
	// From (1,0) the north edge, east wall and diagonally adjacent door are all unreachable.
	if r := c.Submit(actor, spatialProposal(c, "north", ActionMove, ActionArgs{Direction: North})); r.Status != "accepted" {
		t.Fatal(r)
	}
	for _, tc := range []struct {
		actor  string
		action ActionKind
		args   ActionArgs
		code   string
	}{
		{actor, ActionMove, ActionArgs{Direction: North}, "movement_blocked"},
		{actor, ActionMove, ActionArgs{Direction: East}, "movement_blocked"},
		{actor, ActionMove, ActionArgs{Direction: "southeast"}, "invalid_request"},
		{actor, ActionMove, ActionArgs{Direction: West, Target: "town/spoon"}, "invalid_request"},
		{actor, ActionOpen, ActionArgs{Target: "town/door"}, "target_unavailable"},
		{actor, ActionOpen, ActionArgs{Target: "town/absent"}, "target_unavailable"},
		{actor, ActionOpen, ActionArgs{Target: "town/spoon"}, "target_unavailable"},
		{actor, ActionClose, ActionArgs{Target: "town/door", Direction: West}, "invalid_request"},
		{"town/crate", ActionMove, ActionArgs{Direction: South}, "invalid_request"},
	} {
		r := submitUnchanged(t, c, tc.actor, spatialProposal(c, "rejected", tc.action, tc.args))
		if r.Status != "rejected" || r.Code != tc.code {
			t.Fatalf("%s %+v: %+v", tc.action, tc.args, r)
		}
	}
	south := spatialProposal(c, "south", ActionMove, ActionArgs{Direction: South})
	if r := c.Submit(actor, south); r.Status != "accepted" || c.bakery.Entities[actor].Placement.Cell.Y != 1 {
		t.Fatal(r)
	}
	c.clock.tick = math.MaxInt64
	if r := submitUnchanged(t, c, actor, spatialProposal(c, "overflow", ActionOpen, ActionArgs{Target: "town/door"})); r.Code != "resolution_failed" {
		t.Fatalf("expected clock overflow: %+v", r)
	}
}

func TestBakeryResidentAtDoor(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const resident = "town/baker"
	c.bakery.Entities[resident] = c.bakery.Entities["town/player"]
	door := ActionArgs{Target: "town/door"}
	closeDoor := spatialProposal(c, "close", ActionClose, door)
	if r := submitUnchanged(t, c, resident, closeDoor); r.Code != "already_in_state" {
		t.Fatal(r)
	}
	if r := c.Submit(resident, spatialProposal(c, "open", ActionOpen, door)); r.Status != "accepted" {
		t.Fatal(r)
	}
	if r := c.Submit(resident, spatialProposal(c, "enter", ActionMove, ActionArgs{Direction: East})); r.Status != "accepted" {
		t.Fatal(r)
	}
	if r := submitUnchanged(t, c, "town/player", closeDoor); r.Code != "doorway_occupied" {
		t.Fatalf("closed the door on another actor: %+v", r)
	}
}
