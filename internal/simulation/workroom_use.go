// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "net/url"

func (b *Bakery) workbenchStation(prefix string) CellPosition {
	at, step := b.workroomOrigin(prefix)
	entry := CellPosition{FloorID: b.FloorID, X: at.X + 1, Y: at.Y}
	return CellPosition{FloorID: b.FloorID, X: entry.X, Y: entry.Y + step}
}

func workShiftIndex(tick int64) int64 {
	return (tick - 1) / 32
}

func (b *Bakery) bakerUsedToolThisShift(baker string, tick int64) bool {
	actor := b.Entities[baker]
	if actor.WorkroomToolUseTick <= 0 {
		return false
	}
	return workShiftIndex(actor.WorkroomToolUseTick) == workShiftIndex(tick)
}

func (b *Bakery) actorInWorkroom(actor, prefix string) bool {
	cell, err := b.PositionOf(actor)
	if err != nil {
		return false
	}
	return b.Cells[cell.Y][cell.X].SpaceID == prefix+"workroom"
}

func (b *Bakery) useWorkroomTool(scope Scope, baker, tool string, tick int64) spatialEffect {
	item := b.Entities[tool]
	actor := b.Entities[baker]
	actor.WorkroomToolUseTick = tick
	b.Entities[baker] = actor
	return spatialEffect{
		kind:     "workroom_tool_used",
		payload:  EventPayload{Target: tool},
		entityID: baker,
		entity:   actor,
		message:  "You work the " + item.Name + " along the bench.",
	}
}

func (b *Bakery) workroomWithTool(scope Scope, baker, frontDoor, inner, drawer, tool string, tick int64) spatialEffect {
	prefix := url.PathEscape(scope.WorldID) + "/"
	station := b.workbenchStation(prefix)
	at := b.Entities[baker].Placement.Cell
	if at != station {
		return b.stepToward(scope, baker, station, inner)
	}
	if !b.bakerUsedToolThisShift(baker, tick) {
		return b.useWorkroomTool(scope, baker, tool, tick)
	}
	bench := b.Entities[drawer]
	p := Proposal{Scope: scope, Action: ActionPut, Args: ActionArgs{Target: tool, Container: drawer}}
	if !bench.Open {
		p.Action, p.Args = ActionOpen, ActionArgs{Target: drawer}
		effect, _ := b.resolveSpatial(baker, p)
		return effect
	}
	effect, _ := b.resolveSpatial(baker, p)
	return effect
}