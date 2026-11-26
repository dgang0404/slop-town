// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

type Direction string

const (
	North Direction = "north"
	South Direction = "south"
	East  Direction = "east"
	West  Direction = "west"
)

// NewBakeryCoordinator starts the spatial fixture with its own clock and geography draw.
func NewBakeryCoordinator(scope Scope, seed int64) (*Coordinator, error) {
	c, err := NewCoordinator(scope, seed)
	if err != nil {
		return nil, err
	}
	c.bakery, err = GenerateBakery(scope.WorldID, c.clock)
	if err != nil {
		return nil, err
	}
	return c, nil
}

type spatialEffect struct {
	kind    string
	payload EventPayload
	message string
}

type actionFailure struct {
	code    string
	message string
}

// applySpatial operates on the staged entities owned by the coordinator.
func (b *Bakery) applySpatial(actorID string, p Proposal) (spatialEffect, *actionFailure) {
	actor := b.Entities[actorID]
	if actor.Kind != Actor || actor.Placement.HolderID != "" || !b.isFloorCell(actor.Placement.Cell) {
		return spatialEffect{}, &actionFailure{"invalid_request", "Invalid acting character."}
	}
	switch p.Action {
	case ActionMove:
		if p.Args.Stream != "" || p.Args.Target != "" {
			return spatialEffect{}, &actionFailure{"invalid_request", "Movement takes one direction."}
		}
		return b.move(actorID, actor, p.Args.Direction)
	case ActionOpen, ActionClose:
		if p.Args.Stream != "" || p.Args.Direction != "" || !validID(p.Args.Target) {
			return spatialEffect{}, &actionFailure{"invalid_request", "Choose one target."}
		}
		return b.setDoorOpen(actor.Placement.Cell, p.Args.Target, p.Action == ActionOpen)
	default:
		return spatialEffect{}, &actionFailure{"invalid_request", "Unknown spatial action."}
	}
}

func (b *Bakery) move(actorID string, actor SpatialEntity, dir Direction) (spatialEffect, *actionFailure) {
	dest := actor.Placement.Cell
	switch dir {
	case North:
		dest.Y--
	case South:
		dest.Y++
	case East:
		dest.X++
	case West:
		dest.X--
	default:
		return spatialEffect{}, &actionFailure{"invalid_request", "Choose north, south, east or west."}
	}
	blocked := &actionFailure{"movement_blocked", "You can't move there."}
	if !b.isFloorCell(dest) {
		return spatialEffect{}, blocked
	}
	for _, entity := range b.Entities {
		if entity.Kind == Door && entity.Placement.Cell == dest && !entity.Open {
			return spatialEffect{}, blocked
		}
	}
	actor.Placement.Cell = dest
	b.Entities[actorID] = actor
	return spatialEffect{"actor_moved", EventPayload{Destination: dest}, "You move " + string(dir) + "."}, nil
}

func inReach(a, b CellPosition) bool {
	dx := max(a.X, b.X) - min(a.X, b.X)
	dy := max(a.Y, b.Y) - min(a.Y, b.Y)
	return a.FloorID == b.FloorID && dx+dy <= 1
}

func (b *Bakery) setDoorOpen(actorCell CellPosition, target string, open bool) (spatialEffect, *actionFailure) {
	door := b.Entities[target]
	if door.Kind != Door || !inReach(actorCell, door.Placement.Cell) {
		return spatialEffect{}, &actionFailure{"target_unavailable", "You can't reach that target."}
	}
	if door.Open == open {
		return spatialEffect{}, &actionFailure{"already_in_state", "The door is already in that position."}
	}
	kind, message := "door_opened", "You open the door."
	if !open {
		for id, entity := range b.Entities {
			if id != target && entity.Placement.Cell == door.Placement.Cell {
				return spatialEffect{}, &actionFailure{"doorway_occupied", "The doorway is occupied."}
			}
		}
		kind, message = "door_closed", "You close the door."
	}
	door.Open = open
	b.Entities[target] = door
	return spatialEffect{kind, EventPayload{Target: target}, message}, nil
}
