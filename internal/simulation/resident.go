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
	// A free spell belongs to the workers only when both finished their other
	// business. Neither gives up movement, an introduction or a needed rest.
	if b.ContentVersion >= conversationContentVersion && len(events) == 0 {
		if effect := b.residentConversation(primary); effect.kind != "" {
			action := primary.child(primary.Sequence+1, effect.entityID, "spatial", effect.kind, effect.payload)
			b.applySpatial(action, effect)
			events = append(events, action)
		}
	}
	return events, nil
}

// Workers know the route between the shopfront and the prep area. Each
// four-tick period sets the destination. Opening a door costs a turn.
func (b *Bakery) residentTurn(scope Scope, role string, clock *Timeline) (spatialEffect, *EventPayload, error) {
	id := url.PathEscape(scope.WorldID) + "/" + role
	doorID := url.PathEscape(scope.WorldID) + "/door"
	door := b.Entities[doorID].Placement.Cell
	goal := workStation(door, clock.Tick())
	actor := b.Entities[id]
	if b.ContentVersion < restContentVersion || actor.Fatigue < fatigueLimit {
		if b.ContentVersion >= relationshipContentVersion && actor.Placement.Cell == goal && goal.X == door.X-1 && !actor.Seated {
			return b.shopfrontIntroduction(scope, id), nil, nil
		}
		return b.stepToward(scope, id, goal, doorID), nil, nil
	}
	goal.X = door.X + 2 // The resting place behind the prep area.
	return b.restTurn(scope, id, role, clock, goal, b.stepToward(scope, id, goal, doorID))
}

func workStation(door CellPosition, tick int64) CellPosition {
	if (tick-1)/4%2 == 1 {
		door.X++
	} else {
		door.X--
	}
	return door
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

// A worker with a free turn at the shopfront can introduce themselves to
// someone sharing the spot. Work, travel and fatigue keep their priority.
func (b *Bakery) shopfrontIntroduction(scope Scope, speaker string) spatialEffect {
	at := b.Entities[speaker].Placement
	var listener string
	for id, e := range b.Entities {
		if id == speaker || e.Kind != Actor || e.Placement != at || e.Seated || b.recognizes(speaker, id) || b.recognizes(id, speaker) {
			continue
		}
		if listener == "" || id < listener {
			listener = id
		}
	}
	if listener == "" {
		return spatialEffect{}
	}
	effect, _ := b.resolveSpatial(speaker, Proposal{Scope: scope, Action: ActionIntroduce, Args: ActionArgs{Target: listener}})
	return effect
}

// Familiar coworkers can talk during a lull. One conversation per pair in
// eight turns leaves room for the two four-turn legs of their work routine.
func (b *Bakery) residentConversation(primary Event) spatialEffect {
	prefix := url.PathEscape(b.WorldID) + "/"
	baker, porter := prefix+"baker", prefix+"porter"
	if primary.ActorID == baker || primary.ActorID == porter {
		return spatialEffect{}
	}
	if (primary.Kind == "actors_conversed" || primary.Kind == "actor_introduced") &&
		(primary.Payload.Target == baker || primary.Payload.Target == porter) {
		return spatialEffect{}
	}
	station := workStation(b.Entities[prefix+"door"].Placement.Cell, primary.Tick)
	if b.Entities[baker].Placement.Cell != station || b.Entities[porter].Placement.Cell != station {
		return spatialEffect{}
	}
	speaker, listener := baker, porter
	if !b.recognizes(speaker, listener) {
		speaker, listener = listener, speaker
		if !b.recognizes(speaker, listener) {
			return spatialEffect{}
		}
	}
	if b.recentConversation(speaker, listener, primary.Tick) || b.recentConversation(listener, speaker, primary.Tick) {
		return spatialEffect{}
	}
	if b.ContentVersion >= reportContentVersion {
		if effect := b.residentReport(primary.Scope, speaker, listener); effect.kind != "" {
			return effect
		}
		if effect := b.residentReport(primary.Scope, listener, speaker); effect.kind != "" {
			return effect
		}
	}
	view, err := b.Observe(primary.Scope, speaker)
	if err != nil {
		return spatialEffect{}
	}
	effect, _ := b.checkIn(view, listener)
	return effect
}

func (b *Bakery) recentConversation(actor, other string, tick int64) bool {
	entries := b.Notebooks[actor]
	for i := len(entries) - 1; i >= 0 && tick-entries[i].Tick < 8; i-- {
		e := entries[i]
		if e.Subject == other && (e.Condition != "" || (b.ContentVersion >= reportContentVersion && e.Reported != 0)) {
			return true
		}
	}
	return false
}
