// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"maps"
	"slices"
)

// SightHandMovement is a perception, never a submitted action. Its memory
// carries neither the actor's identity nor the identity of the item.
const SightHandMovement ActionKind = "hand_movement"

func (s WitnessedAction) HandMovementPhrase() string {
	if s.Container.Ref != "" {
		return "a hand move near the " + s.Container.Name
	}
	return "a hand move"
}

// Initial aptitudes are fixed for each character in this seeded town. Only
// an eligible witness's own perception stream advances when a pickup occurs.
func (b *Bakery) observeDiscreetTake(event Event, clock *Timeline) error {
	taker := b.Entities[event.ActorID]
	item := b.Entities[event.Payload.Target]
	stealth := int(seededSample(b.Seed, "aptitude/stealth/"+event.ActorID, 0) % 21)
	for _, id := range slices.Sorted(maps.Keys(b.Entities)) {
		witness := b.Entities[id]
		if witness.Kind != Actor || id == event.ActorID {
			continue
		}
		view, err := b.Observe(event.Scope, id)
		if err != nil {
			return err
		}
		if !view.hasTarget(event.ActorID) {
			continue
		}
		roll, err := clock.Draw("perception/" + id)
		if err != nil {
			return err
		}
		distance := manhattanDistance(witness.Placement.Cell, taker.Placement.Cell)
		perception := int(seededSample(b.Seed, "aptitude/perception/"+id, 0) % 21)
		score := int(roll%100) + perception - stealth - 8*distance
		entry := NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence}
		switch {
		case score >= 55 && view.hasTarget(event.Payload.Target):
			entry.Action, entry.Subject, entry.Target = ActionTake, event.ActorID, event.Payload.Target
		case score >= 22:
			entry.Action = SightHandMovement
			if holder := item.Placement.HolderID; holder != "" && view.hasTarget(holder) {
				entry.Container = holder
			}
		default:
			continue
		}
		b.appendSighting(id, entry)
	}
	return nil
}
