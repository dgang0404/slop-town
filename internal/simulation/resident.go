// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "net/url"

// Workers take at most one spatial action in this fixed order, after the
// primary actor. A break decision is logged immediately before its effect.
func (b *Bakery) advanceResidents(clock *Timeline, primary Event) ([]Event, error) {
	if b.ContentVersion < residentContentVersion {
		return nil, nil
	}
	roles := []string{"baker"}
	if b.ContentVersion >= personalityContentVersion {
		roles = append(roles, "porter")
	}
	var events []Event
	for _, role := range roles {
		id := url.PathEscape(b.WorldID) + "/" + role
		if id == primary.ActorID {
			continue
		}
		effect, choice, err := b.residentTurn(primary.Scope, role, clock)
		if err != nil {
			return nil, err
		}
		if effect.kind == "" {
			continue
		}
		parent := primary
		if choice != nil {
			parent = primary.child(primary.Sequence+uint64(len(events))+1, id, "personality", "rest_place_chosen", *choice)
			events = append(events, parent)
		}
		action := parent.child(primary.Sequence+uint64(len(events))+1, id, "spatial", effect.kind, effect.payload)
		b.applySpatial(action, effect)
		if choice != nil && effect.kind != "actor_rested" {
			worker := b.Entities[id]
			worker.SeekingRestSpot = true
			b.Entities[id] = worker
		}
		events = append(events, action)
	}
	return events, nil
}

// Workers know the route between the shopfront and the prep area. Each
// four-tick period sets the destination. Opening a door costs a turn.
func (b *Bakery) residentTurn(scope Scope, role string, clock *Timeline) (spatialEffect, *EventPayload, error) {
	id := url.PathEscape(scope.WorldID) + "/" + role
	doorID := url.PathEscape(scope.WorldID) + "/door"
	door := b.Entities[doorID].Placement.Cell
	goal := door
	goal.X--
	if (clock.Tick()-1)/4%2 == 1 {
		goal.X = door.X + 1
	}
	actor := b.Entities[id]
	if b.ContentVersion < restContentVersion || actor.Fatigue < fatigueLimit {
		return b.stepToward(scope, id, goal, doorID), nil, nil
	}
	goal.X = door.X + 2 // The resting place behind the prep area.
	return b.restTurn(scope, id, role, clock, goal, b.stepToward(scope, id, goal, doorID))
}

func (b *Bakery) restTurn(scope Scope, id, role string, clock *Timeline, goal CellPosition, step spatialEffect) (spatialEffect, *EventPayload, error) {
	actor := b.Entities[id]
	rest, failure := b.resolveSpatial(id, Proposal{Scope: scope, Action: ActionRest})
	if failure != nil {
		return step, nil, nil
	}
	if step.kind == "" && actor.Placement.Cell != goal && b.ContentVersion < personalityContentVersion {
		return step, nil, nil // Historical routine waits for its route to clear.
	}
	if step.kind == "" {
		return rest, nil, nil
	}
	if b.ContentVersion < personalityContentVersion || actor.SeekingRestSpot {
		return step, nil, nil
	}
	// Both resting here and taking a step toward the back are possible.
	stream := "rest-place/" + role
	draw, err := clock.Draw(stream)
	if err != nil {
		return spatialEffect{}, nil, err
	}
	choice := &EventPayload{Stream: stream, Value: draw % 100, Destination: goal}
	if int(choice.Value) < actor.RestInPlacePercent {
		choice.Destination = actor.Placement.Cell
		return rest, choice, nil
	}
	return step, choice, nil
}

func (b *Bakery) stepToward(scope Scope, id string, goal CellPosition, doorID string) spatialEffect {
	at := b.Entities[id].Placement.Cell
	var dir Direction
	switch {
	case at.Y < goal.Y:
		dir = South
	case at.Y > goal.Y:
		dir = North
	case at.X < goal.X:
		dir = East
	case at.X > goal.X:
		dir = West
	default:
		return spatialEffect{}
	}
	p := Proposal{Scope: scope, Action: ActionMove, Args: ActionArgs{Direction: dir}}
	effect, failure := b.resolveSpatial(id, p)
	if failure != nil && failure.code == "movement_blocked" {
		if next, _ := at.neighbor(dir); next == b.Entities[doorID].Placement.Cell {
			p.Action, p.Args = ActionOpen, ActionArgs{Target: doorID}
			effect, _ = b.resolveSpatial(id, p)
		}
	}
	return effect
}
