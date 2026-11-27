// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"testing"
)

func TestListeningUsesTheDoorAtTheMomentOfTheSound(t *testing.T) {
	const player, baker, porter, doorID = "bakery/player", "bakery/baker", "bakery/porter", "bakery/door"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	door := c.bakery.Entities[doorID]
	// The baker steps up behind the closed door, then the porter opens it.
	for id, offset := range map[string]int{baker: 2, porter: 1, player: -1} {
		a := c.bakery.Entities[id]
		a.Placement = door.Placement
		a.Placement.Cell.X += offset
		c.bakery.Entities[id] = a
	}
	p := spatialProposal(c, "listen", ActionListen, ActionArgs{})
	r := c.Submit(player, p)
	if r.Status != "accepted" || !slices.Equal(r.Observation.Heard, []PerceptionKind{DoorCreak}) || !c.bakery.Entities[doorID].Open {
		t.Fatal(r)
	}
	events := c.History()
	if len(events) != 5 || events[2].Kind != "actor_moved" || events[2].ActorID != baker || events[3].Kind != "door_opened" ||
		events[4].Kind != "sound_heard" || events[4].Payload.Sound != DoorCreak || events[4].ActorID != player {
		t.Fatal(events)
	}
	// Editing the client's view must not change the saved perception.
	r.Observation.Heard[0] = Footsteps
	if again := submitUnchanged(t, c, player, p); !slices.Equal(again.Observation.Heard, []PerceptionKind{DoorCreak}) {
		t.Fatal(again)
	}
	next := c.Submit(player, spatialProposal(c, "listen-again", ActionListen, ActionArgs{}))
	events = c.History()[len(events):]
	if len(events) != 5 || events[2].Kind != "actor_moved" || events[3].Kind != "actor_moved" ||
		events[4].Kind != "sound_heard" || events[4].Payload.Sound != Footsteps ||
		!slices.Equal(next.Observation.Heard, []PerceptionKind{Footsteps}) {
		t.Fatalf("two workers stepped on one turn: %v, heard %v", events, next.Observation.Heard)
	}
	if view, err := c.Observe(player); err != nil || len(view.Heard) != 0 {
		t.Fatal("a later look repeated the earlier sound", view, err)
	}
}

func TestAResidentCanListenAroundACorner(t *testing.T) {
	const baker, porter, doorID = "bakery/baker", "bakery/porter", "bakery/door"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	door := b.Entities[doorID]
	door.Open = true
	b.Entities[doorID] = door
	worker, listener := b.Entities[baker], b.Entities[porter]
	worker.Placement, listener.Placement = door.Placement, door.Placement
	worker.Placement.Cell.X += 2
	listener.Placement.Cell.X--
	listener.Placement.Cell.Y++
	b.Entities[baker], b.Entities[porter] = worker, listener
	r := c.Submit(porter, spatialProposal(c, "listen", ActionListen, ActionArgs{}))
	if r.Status != "accepted" || r.Message != "You hear footsteps." || r.Observation.hasTarget(baker) ||
		!slices.Equal(r.Observation.Heard, []PerceptionKind{Footsteps}) {
		t.Fatal(r)
	}
	// Sound must take the four-cell route through the open doorway even
	// though the cells on opposite sides of this wall are two tiles apart.
	source, ear := door.Placement.Cell, door.Placement.Cell
	source.X++
	ear.X--
	source.Y--
	ear.Y--
	if b.soundReaches(source, ear, 3) || !b.soundReaches(source, ear, 4) {
		t.Fatal("sound crossed the wall or failed to round the corner")
	}
}

func TestListenMenuAndOldTimelines(t *testing.T) {
	const player = "bakery/player"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []int{listenContentVersion, discreetTakeContentVersion} {
		c.bakery.ContentVersion = version
		menu, err := c.AvailableActions(player)
		if err != nil || slices.Contains(menu, ActionOption{Action: ActionListen}) != (version == listenContentVersion) {
			t.Fatal(version, menu, err)
		}
	}
	if r := submitUnchanged(t, c, player, spatialProposal(c, "unavailable", ActionListen, ActionArgs{})); r.Code != "action_unavailable" {
		t.Fatal(r)
	}
}
