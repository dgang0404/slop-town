// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

// Sample the room when the effect occurs. Another worker may move or open a
// door later on the same turn. Only the kind of sound enters the receipt.
func (b *Bakery) audibleAction(listener string, event Event) PerceptionKind {
	if listener == event.ActorID {
		return ""
	}
	var sound PerceptionKind
	var source CellPosition
	var reach int
	switch event.Kind {
	case "actor_moved":
		sound, source, reach = Footsteps, event.Payload.Destination, 3
	case "door_opened", "door_closed":
		sound, source, reach = DoorCreak, b.Entities[event.Payload.Target].Placement.Cell, 4
	default:
		return ""
	}
	if perceptionAptitude(b.Seed, listener) >= 14 {
		reach++
	}
	if b.soundReaches(source, b.Entities[listener].Placement.Cell, reach) {
		return sound
	}
	return ""
}

// Sound travels through adjacent floor cells and around corners. A closed
// door blocks a sound passing through it. Its own creak starts on the door.
func (b *Bakery) soundReaches(source, listener CellPosition, reach int) bool {
	if !b.isFloorCell(source) || !b.isFloorCell(listener) || manhattanDistance(source, listener) > reach {
		return false
	}
	blocked := b.opaqueCells()
	blocked[source.Y][source.X] = true // Mark visited cells as we enqueue them.
	type step struct {
		cell     CellPosition
		distance int
	}
	queue := []step{{cell: source}}
	for i := 0; i < len(queue); i++ {
		current := queue[i]
		if current.cell == listener {
			return true
		}
		if current.distance == reach {
			continue
		}
		for _, direction := range []Direction{North, East, South, West} {
			next, _ := current.cell.neighbor(direction)
			if b.isFloorCell(next) && !blocked[next.Y][next.X] {
				blocked[next.Y][next.X] = true
				queue = append(queue, step{next, current.distance + 1})
			}
		}
	}
	return false
}
