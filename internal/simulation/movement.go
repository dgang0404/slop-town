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

func targetUnavailable() *actionFailure {
	return &actionFailure{"target_unavailable", "You can't reach that target."}
}

// applySpatial operates on the staged entities owned by the coordinator.
func (b *Bakery) applySpatial(actorID string, p Proposal) (spatialEffect, *actionFailure) {
	actor := b.Entities[actorID]
	if actor.Kind != Actor || actor.Placement.HolderID != "" || !b.isFloorCell(actor.Placement.Cell) {
		return spatialEffect{}, &actionFailure{"invalid_request", "Invalid acting character."}
	}
	args := p.Args
	switch p.Action {
	case ActionMove:
		if p.ByName || args != (ActionArgs{Direction: args.Direction}) {
			return spatialEffect{}, &actionFailure{"invalid_request", "Movement takes one direction."}
		}
		return b.move(actorID, actor, args.Direction)
	case ActionPut:
		if !validID(args.Target) || !validID(args.Container) || args != (ActionArgs{Target: args.Target, Container: args.Container}) {
			return spatialEffect{}, &actionFailure{"invalid_request", "Choose an item and a destination container."}
		}
	case ActionOpen, ActionClose, ActionTake, ActionDrop:
		if !validID(args.Target) || args != (ActionArgs{Target: args.Target}) {
			return spatialEffect{}, &actionFailure{"invalid_request", "Choose one target."}
		}
	default:
		return spatialEffect{}, &actionFailure{"invalid_request", "Unknown spatial action."}
	}
	view, err := b.Observe(p.Scope, actorID)
	if err != nil {
		return spatialEffect{}, targetUnavailable()
	}
	var failure *actionFailure
	p.Args.Target, failure = view.selectTarget(args.Target, p.ByName)
	if failure != nil {
		return spatialEffect{}, failure
	}
	if p.Action == ActionPut {
		p.Args.Container, failure = view.selectTarget(args.Container, p.ByName)
		if failure != nil {
			return spatialEffect{}, failure
		}
	}
	if p.Action == ActionOpen || p.Action == ActionClose {
		return b.setOpen(actor.Placement.Cell, p.Args.Target, p.Action == ActionOpen)
	}
	return b.transferItem(actorID, actor.Placement.Cell, p)
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

func (b *Bakery) setOpen(actorCell CellPosition, target string, open bool) (spatialEffect, *actionFailure) {
	entity := b.Entities[target]
	if (entity.Kind != Door && entity.Kind != Container) || !inReach(actorCell, entity.Placement.Cell) {
		return spatialEffect{}, targetUnavailable()
	}
	name := entity.Name
	if entity.Kind == Door {
		name = "door"
	}
	if entity.Open == open {
		return spatialEffect{}, &actionFailure{"already_in_state", "The " + name + " is already in that position."}
	}
	if entity.Kind == Door && !open {
		for id, occupant := range b.Entities {
			if id != target && occupant.Placement.Cell == entity.Placement.Cell {
				return spatialEffect{}, &actionFailure{"doorway_occupied", "The doorway is occupied."}
			}
		}
	}
	entity.Open = open
	b.Entities[target] = entity
	if open {
		return spatialEffect{string(entity.Kind) + "_opened", EventPayload{Target: target}, "You open the " + name + "."}, nil
	}
	return spatialEffect{string(entity.Kind) + "_closed", EventPayload{Target: target}, "You close the " + name + "."}, nil
}
