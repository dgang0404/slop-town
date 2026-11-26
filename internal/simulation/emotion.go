// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

const reliefDuration = 3

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
