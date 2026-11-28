// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"maps"
	"slices"
)

// Sample each listener before the next actor moves. The cause stays in the
// replay history; the listener's notebook carries only the sound.
type heardSound struct {
	listener string
	kind     PerceptionKind
	cause    Event
}

func (b *Bakery) soundsDuringAction(heard []heardSound, primary, event Event) []heardSound {
	var sound PerceptionKind
	var source CellPosition
	var reach int
	switch event.Kind {
	case "door_knocked":
		sound, source, reach = Knock, b.Entities[event.Payload.Target].Placement.Cell, 4
	case "item_dropped":
		if b.ContentVersion >= dropSoundContentVersion {
			sound, source, reach = b.Entities[event.Payload.Target].DropSound, event.Payload.Destination, 3
		}
	default:
		if primary.Kind != "actor_listened" {
			return heard
		}
		sound = b.audibleAction(primary.ActorID, event)
		if sound != "" && !slices.ContainsFunc(heard, func(h heardSound) bool { return h.listener == primary.ActorID && h.kind == sound }) {
			heard = append(heard, heardSound{primary.ActorID, sound, primary})
		}
		return heard
	}
	if sound == "" {
		return heard
	}
	for _, id := range slices.Sorted(maps.Keys(b.Entities)) {
		// The knocker already recalls the deliberate action in their receipt.
		if b.Entities[id].Kind != Actor || (sound == Knock && id == event.ActorID) {
			continue
		}
		if b.withinEarshot(id, source, reach) {
			heard = append(heard, heardSound{id, sound, event})
		}
	}
	return heard
}

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
	if b.withinEarshot(listener, source, reach) {
		return sound
	}
	return ""
}

// Conversation carries a shorter way than a knock. Skill adds at most one step.
// A fragment can pass out of sight a little farther. Beyond that, only a murmur remains.
const (
	conversationReach = 2
	fragmentReach     = conversationReach + 3
	murmurReach       = conversationReach + 6
)

func (b *Bakery) withinEarshot(listener string, source CellPosition, reach int) bool {
	if perceptionAptitude(b.Seed, listener) >= 14 {
		reach++
	}
	return b.soundReaches(source, b.Entities[listener].Placement.Cell, reach)
}

// Sound travels through adjacent floor cells and around corners. A closed
// door blocks a sound passing through it. Creaks and knocks start on the door.
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
