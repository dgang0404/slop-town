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
	kind     string
	payload  EventPayload
	message  string
	entityID string
	entity   SpatialEntity
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
	view, err := b.Observe(p.Scope, actorID)
	if err != nil {
		return spatialEffect{}, &actionFailure{"invalid_request", "Invalid acting character."}
	}
	effect, failure := b.planSpatial(view, p.Action, p.Args, p.ByName)
	if failure == nil {
		b.Entities[effect.entityID] = effect.entity
	}
	return effect, failure
}

// planSpatial computes the effect without applying it. Discovery and submission
// use the same target selection, movement and interaction rules.
func (b *Bakery) planSpatial(view Observation, action ActionKind, args ActionArgs, byName bool) (spatialEffect, *actionFailure) {
	actor := b.Entities[view.ObserverID]
	switch action {
	case ActionMove:
		if byName || args != (ActionArgs{Direction: args.Direction}) {
			return spatialEffect{}, &actionFailure{"invalid_request", "Movement takes one direction."}
		}
		return b.move(view.ObserverID, actor, args.Direction)
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
	var failure *actionFailure
	args.Target, failure = view.selectTarget(args.Target, byName)
	if failure != nil {
		return spatialEffect{}, failure
	}
	if action == ActionPut {
		args.Container, failure = view.selectTarget(args.Container, byName)
		if failure != nil {
			return spatialEffect{}, failure
		}
	}
	if action == ActionOpen || action == ActionClose {
		return b.setOpen(actor.Placement.Cell, args.Target, action == ActionOpen)
	}
	return b.transferItem(view.ObserverID, actor.Placement.Cell, action, args)
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
	return spatialEffect{kind: "actor_moved", payload: EventPayload{Destination: dest},
		message: "You move " + string(dir) + ".", entityID: actorID, entity: actor}, nil
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
	verb, kind := "close", string(entity.Kind)+"_closed"
	if open {
		verb, kind = "open", string(entity.Kind)+"_opened"
	}
	return spatialEffect{kind: kind, payload: EventPayload{Target: target},
		message: "You " + verb + " the " + name + ".", entityID: target, entity: entity}, nil
}
