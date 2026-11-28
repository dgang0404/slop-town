// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"net/url"
	"slices"
)

// Workers take at most one spatial action in this fixed order, after the
// primary actor. A break decision is logged immediately before its effect.
func (b *Bakery) advanceResidents(clock *Timeline, primary Event) ([]Event, []heardSound, error) {
	if b.ContentVersion < residentContentVersion {
		return nil, nil, nil
	}
	roles := []string{"baker"}
	if b.ContentVersion >= personalityContentVersion {
		roles = append(roles, "porter")
	}
	var events []Event
	heard := b.soundsDuringAction(nil, primary, primary)
	if effect := b.secondCircle(primary); effect.kind != "" {
		action := primary.child(primary.Sequence+1, effect.entityID, "spatial", effect.kind, effect.payload)
		if err := b.applySpatial(&action, effect, clock); err != nil {
			return nil, nil, err
		}
		events = append(events, action)
	}
	for _, role := range roles {
		id := url.PathEscape(b.WorldID) + "/" + role
		if id == primary.ActorID || (primary.Kind == "call_answered" && primary.Payload.Target == id) {
			continue
		}
		effect, choice, err := b.residentTurn(primary.Scope, role, clock)
		if err != nil {
			return nil, nil, err
		}
		if effect.kind == "" && len(b.Letters) > 0 {
			effect = b.residentMail(id)
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
		if err := b.applySpatial(&action, effect, clock); err != nil {
			return nil, nil, err
		}
		heard = b.soundsDuringAction(heard, primary, action)
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
			if err := b.applySpatial(&action, effect, clock); err != nil {
				return nil, nil, err
			}
			events = append(events, action)
		}
	}
	return events, heard, nil
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
		// The baker checks the workbench during the last eight turns of each
		// 32-turn shift. The visit happens even if nobody filled the order.
		if role == "baker" && b.GeneratorVersion >= 4 && b.ContentVersion >= workroomRoundsVersion && (clock.Tick()-1)%32 >= 24 {
			return b.workroomRound(scope, id, doorID, clock), nil, nil
		}
		if b.ContentVersion >= relationshipContentVersion && actor.Placement.Cell == goal && goal.X == door.X-1 && !actor.Seated {
			return b.shopfrontIntroduction(scope, id), nil, nil
		}
		return b.stepToward(scope, id, goal, doorID), nil, nil
	}
	goal.X = door.X + 2 // The resting place behind the prep area.
	return b.restTurn(scope, id, role, clock, goal, b.stepToward(scope, id, goal, doorID))
}

// Approach along the main aisle before turning toward the inner door.
func (b *Bakery) workroomRound(scope Scope, baker, frontDoor string, clock *Timeline) spatialEffect {
	prefix := url.PathEscape(scope.WorldID) + "/"
	inner := prefix + "workroom-door"
	drawer := prefix + "workbench"
	tool := b.Entities[prefix+"order"].RequestedItem
	tick := clock.Tick()

	if b.ContentVersion >= workroomToolUseVersion {
		if b.Entities[tool].Placement.HolderID == baker {
			return b.workroomWithTool(scope, baker, frontDoor, inner, drawer, tool, tick)
		}
		if b.bakerUsedToolThisShift(baker, tick) && b.Entities[tool].Placement.HolderID == drawer {
			if b.actorInWorkroom(baker, prefix) {
				entry := b.Entities[inner].Placement.Cell
				at := b.Entities[baker].Placement.Cell
				if at != entry {
					return b.stepToward(scope, baker, entry, inner)
				}
				p := Proposal{Scope: scope, Action: ActionOpen, Args: ActionArgs{Target: inner}}
				if !b.Entities[inner].Open {
					effect, _ := b.resolveSpatial(baker, p)
					return effect
				}
				aisle := b.Entities[frontDoor].Placement.Cell
				aisle.X = entry.X
				if at != aisle {
					return b.stepToward(scope, baker, aisle, frontDoor)
				}
			}
			return spatialEffect{}
		}
	}

	entry := b.Entities[inner].Placement.Cell
	at := b.Entities[baker].Placement.Cell
	if at.X != entry.X {
		aisle := b.Entities[frontDoor].Placement.Cell
		aisle.X = entry.X
		return b.stepToward(scope, baker, aisle, frontDoor)
	}
	if at != entry {
		return b.stepToward(scope, baker, entry, inner)
	}
	p := Proposal{Scope: scope, Action: ActionOpen, Args: ActionArgs{Target: drawer}}
	if b.Entities[drawer].Open {
		if b.Entities[tool].Placement.HolderID != drawer {
			return spatialEffect{}
		}
		p.Action, p.Args.Target = ActionTake, tool
	}
	effect, _ := b.resolveSpatial(baker, p)
	return effect
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
func (b *Bakery) secondCircle(primary Event) spatialEffect {
	if b.ContentVersion < answerFirstVersion || primary.Kind != "actors_conversed" || primary.Payload.Dialogue.Statement.Intent != "answer_first" {
		return spatialEffect{}
	}
	busy := map[string]bool{primary.ActorID: true, primary.Payload.Target: true}
	var idle []string
	for id, e := range b.Entities {
		if e.Kind != Actor || busy[id] {
			continue
		}
		idle = append(idle, id)
	}
	slices.Sort(idle)
	var speaker, listener string
	found := false
	for i := 0; i < len(idle) && !found; i++ {
		for j := i + 1; j < len(idle); j++ {
			if b.apartFrom(primary, idle[i], idle[j]) {
				speaker, listener, found = idle[i], idle[j], true
				break
			}
		}
	}
	if !found {
		return spatialEffect{}
	}
	x := Exchange{Words: "Hello from the other table.", Reply: "Hello.", Statement: SpeechLine{Intent: "aside"}}
	return spatialEffect{kind: "actors_conversed", payload: EventPayload{Target: listener, Dialogue: x},
		message: x.message(b.Entities[listener].Name), entityID: speaker, entity: b.Entities[speaker]}
}

func (b *Bakery) apartFrom(primary Event, speaker, listener string) bool {
	speakerView, err := b.Observe(primary.Scope, speaker)
	if err != nil || !speakerView.hasTarget(listener) || speakerView.hasTarget(primary.ActorID) || speakerView.hasTarget(primary.Payload.Target) {
		return false
	}
	listenerView, err := b.Observe(primary.Scope, listener)
	if err != nil || !listenerView.hasTarget(speaker) || listenerView.hasTarget(primary.ActorID) || listenerView.hasTarget(primary.Payload.Target) {
		return false
	}
	return true
}

func (b *Bakery) residentConversation(primary Event) spatialEffect {
	prefix := url.PathEscape(b.WorldID) + "/"
	baker, porter := prefix+"baker", prefix+"porter"
	if primary.ActorID == baker || primary.ActorID == porter {
		return spatialEffect{}
	}
	if (primary.Kind == "actors_conversed" || primary.Kind == "call_answered" || primary.Kind == "actor_introduced") &&
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
		if e.Subject == other && !e.Overheard && (e.Condition != "" || (b.ContentVersion >= reportContentVersion && e.Reported != 0)) {
			return true
		}
	}
	return false
}
