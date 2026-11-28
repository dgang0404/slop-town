// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

const reliefDuration = 3

// A heavy answer fades across later turns unless a new one resets it.
const weightDuration = 4

// Once per committed turn, age existing feelings and then apply this turn's
// personal triggers. A worker can recover even while the player is elsewhere.
func (b *Bakery) advanceRelief(events []Event) {
	for id, actor := range b.Entities {
		if actor.ReliefTurns > 0 {
			actor.ReliefTurns--
			b.Entities[id] = actor
		}
	}
	for _, event := range events {
		if event.Payload.ReliefStarted {
			actor := b.Entities[event.ActorID]
			actor.ReliefTurns = reliefDuration
			b.Entities[event.ActorID] = actor
		}
	}
}

// A heavy answer lingers for a few turns, then fades when nothing reinforces it.
func (b *Bakery) advanceWeight() {
	if b.ContentVersion < answerFirstVersion {
		return
	}
	for id, actor := range b.Entities {
		if actor.WeightTurns <= 0 {
			continue
		}
		actor.WeightTurns--
		if actor.WeightTurns == 0 {
			actor.Occupied = false
			actor.Attention = ""
		}
		b.Entities[id] = actor
	}
}
