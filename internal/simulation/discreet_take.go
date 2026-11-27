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

type stealthPractice struct {
	Floor, Container bool
}

// Each kind of reach counts once, however many witnesses or attempts there are.
func (p *stealthPractice) record(from Placement) string {
	switch {
	case from.HolderID == "" && !p.Floor:
		p.Floor = true
		return "floor"
	case from.HolderID != "" && !p.Container:
		p.Container = true
		return "container"
	}
	return ""
}

// Initial aptitudes are fixed for each character in this seeded town. Only
// an eligible witness's own perception stream advances when a pickup occurs.
func (b *Bakery) observeDiscreetTake(event *Event, clock *Timeline) error {
	taker := b.Entities[event.ActorID]
	item := b.Entities[event.Payload.Target]
	stealth := int(seededSample(b.Seed, "aptitude/stealth/"+event.ActorID, 0) % 21)
	if taker.StealthPractice.Floor && taker.StealthPractice.Container {
		stealth += 10
	}
	// Judge the light where the hand reaches the item, before it is moved.
	at, err := b.PositionOf(event.Payload.Target)
	if err != nil {
		return err
	}
	lightPenalty := 0
	if b.dimAt(at) {
		lightPenalty = 25
	}
	var visibleUpClose bool
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
		seesItem := view.hasTarget(event.Payload.Target)
		if distance <= 1 && seesItem {
			visibleUpClose = true
		}
		perception := perceptionAptitude(b.Seed, id)
		score := int(roll%100) + perception - stealth - 8*distance - lightPenalty
		entry := NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence}
		switch {
		case score >= 55 && seesItem:
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
	// The reach can teach even when identified. Apply practice after every
	// witness has resolved this attempt with the actor's starting skill.
	if b.ContentVersion >= stealthPracticeContentVersion && visibleUpClose {
		event.Payload.StealthPractice = taker.StealthPractice.record(item.Placement)
		b.Entities[event.ActorID] = taker
	}
	return nil
}
