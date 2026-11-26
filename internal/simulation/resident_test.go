// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Checkpoint 3 timelines have one scheduled worker and a stationary porter.
func notebookCoordinator(t *testing.T, seed int64) *Coordinator {
	t.Helper()
	c, err := NewCoordinator(Scope{"town", "main"}, seed)
	if err != nil {
		t.Fatal(err)
	}
	c.bakery, err = generateNotebookBakery(c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBakerWorkRoutine(t *testing.T) {
	const player, baker = "town/player", "town/baker"
	for _, seed := range []int64{42, 43} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			c := notebookCoordinator(t, seed)
			start, err := c.Observe(player)
			if err != nil || start.hasTarget(baker) {
				t.Fatalf("baker should start behind a closed door: %+v, %v", start, err)
			}
			if r := submitUnchanged(t, c, player, spatialProposal(c, "east", ActionMove, ActionArgs{Direction: East})); r.Code != "movement_blocked" {
				t.Fatal(r)
			}
			if r := submitUnchanged(t, c, player, spatialProposal(c, "bad-wait", ActionWait, ActionArgs{Target: baker})); r.Code != "invalid_request" {
				t.Fatal(r)
			}
			steps := []struct {
				x     int
				event string
			}{
				{3, "door_opened"}, {2, "actor_moved"}, {1, "actor_moved"}, {1, ""},
				{2, "actor_moved"}, {3, "actor_moved"}, {3, ""}, {3, ""},
				{2, "actor_moved"}, {3, "actor_moved"}, {4, "actor_moved"}, {4, "actor_rested"},
				{3, "actor_moved"}, {3, ""}, {3, ""}, {3, ""}, {2, "actor_moved"}, {1, "actor_moved"},
			}
			door := c.bakery.Entities["town/door"].Placement.Cell
			var first Result
			for i, step := range steps {
				tick := int64(i + 1)
				before := len(c.events)
				p := spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})
				r := c.Submit(player, p)
				if r.Status != "accepted" || r.Message != "You wait." || !r.Observation.hasTarget(baker) || c.clock.Tick() != tick {
					t.Fatalf("tick %d: %+v", tick, r)
				}
				at := c.bakery.Entities[baker].Placement.Cell
				if sitting := strings.Contains(r.Observation.Text, "You see the baker sitting to the east."); sitting != (tick == 12) {
					t.Fatalf("tick %d: unexpected posture: %s", tick, r.Observation.Text)
				}
				if at != (CellPosition{door.FloorID, step.x, door.Y}) || !c.bakery.Entities["town/door"].Open {
					t.Fatalf("tick %d: got baker at %+v, door open %v", tick, at, c.bakery.Entities["town/door"].Open)
				}
				events := c.events[before:]
				if len(events) < 2 || events[0].Kind != "tick_advanced" || events[1].Kind != "actor_waited" || events[1].ActorID != player {
					t.Fatalf("tick %d: missing player action: %+v", tick, events)
				}
				if i == 0 {
					first = r
				}
				if step.event == "" {
					if len(events) != 2 {
						t.Fatalf("tick %d: unexpected action at work station: %+v", tick, events)
					}
					continue
				}
				if len(events) != 3 {
					t.Fatalf("tick %d: got %d events", tick, len(events))
				}
				e := events[2]
				if e.Kind != step.event || e.ActorID != baker || e.ParentID != events[1].ID ||
					e.Tick != tick || e.Sequence != uint64(before+3) || e.RequestID != p.RequestID {
					t.Fatalf("tick %d: wrong resident event: %+v", tick, e)
				}
			}
			retry := submitUnchanged(t, c, player, spatialProposal(c, "1", ActionWait, ActionArgs{}))
			if !reflect.DeepEqual(retry, first) {
				t.Fatalf("retry returned a different observation: %+v", retry)
			}
		})
	}
}

func TestBakerUsesSharedDoor(t *testing.T) {
	c, err := (BakerySetup{Resident: true}).NewCoordinator(Scope{"town", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player, baker = "town/player", "town/baker"
	door := ActionArgs{Target: "town/door"}
	if r := c.Submit(player, spatialProposal(c, "open", ActionOpen, door)); r.Status != "accepted" {
		t.Fatal(r)
	}
	if c.bakery.Entities[baker].Placement != c.bakery.Entities[door.Target].Placement {
		t.Fatal("the baker should step through the door the player opened")
	}
	if r := submitUnchanged(t, c, player, spatialProposal(c, "close", ActionClose, door)); r.Code != "doorway_occupied" {
		t.Fatalf("player closed the door on the baker: %+v", r)
	}
	before := len(c.events)
	if r := c.Submit(baker, spatialProposal(c, "baker-wait", ActionWait, ActionArgs{})); r.Status != "accepted" ||
		len(c.events) != before+3 || c.events[before+2].ActorID != "town/porter" || c.bakery.Entities[baker].Placement != c.bakery.Entities[door.Target].Placement {
		t.Fatal("an explicitly acting resident should only take one turn", r)
	}
}

func TestBakerWorksOutOfSight(t *testing.T) {
	c := notebookCoordinator(t, 42)
	// Walk south along the street, then stay there as the baker returns indoors.
	var r Result
	for tick := 1; tick <= 12; tick++ {
		p := spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})
		if tick <= 2 {
			p.Action, p.Args = ActionMove, ActionArgs{Direction: South}
		}
		r = c.Submit("town/player", p)
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		if tick == 6 {
			last := c.events[len(c.events)-1]
			if last.ActorID != "town/baker" || last.Kind != "actor_moved" || last.Payload.Destination.X != 3 || r.Observation.hasTarget(last.ActorID) {
				t.Fatalf("expected the baker to return to prep out of sight: %+v, %+v", last, r.Observation)
			}
		}
	}
	last := c.events[len(c.events)-1]
	if last.ActorID != "town/baker" || last.Kind != "actor_rested" || !c.bakery.Entities[last.ActorID].Seated ||
		r.Observation.hasTarget(last.ActorID) || strings.Contains(r.Observation.Text, "sitting") {
		t.Fatalf("expected the baker to rest out of sight: %+v, %+v", last, r.Observation)
	}
}

func TestRestChangesTheBakersNextChoice(t *testing.T) {
	c, err := (BakerySetup{Resident: true}).NewCoordinator(Scope{"town", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player, baker = "town/player", "town/baker"
	// The baker normally leaves the route at tick 10. An early rest changes
	// that choice on the same schedule. The player can rest independently.
	for tick := 1; tick <= 10; tick++ {
		p := spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})
		actor := player
		switch tick {
		case 1:
			p.Action, p.Args.Direction = ActionMove, West
		case 2:
			p.Action = ActionRest
		case 8:
			actor, p.Action = baker, ActionRest
		}
		r := c.Submit(actor, p)
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		self := c.bakery.Entities[actor]
		if p.Action == ActionRest && (self.Fatigue != 0 || !self.Seated || !strings.Contains(r.Observation.Text, "You are sitting down.")) {
			t.Fatalf("rest did not recover %s: %+v, %+v", actor, self, r)
		}
		if tick == 7 && c.bakery.Entities[baker].Fatigue != 4 {
			t.Fatal("the player's rest affected the baker's fatigue")
		}
	}
	b := c.bakery.Entities[baker]
	if b.Seated || b.Fatigue != 2 || b.Placement.Cell.X != 1 {
		t.Fatalf("expected the rested baker at the shopfront: %+v", b)
	}
	if r := submitUnchanged(t, c, player, spatialProposal(c, "bad-rest", ActionRest, ActionArgs{Target: baker})); r.Code != "invalid_request" {
		t.Fatal(r)
	}
}
