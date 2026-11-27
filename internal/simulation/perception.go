// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

// PerceptionKind identifies a place or sensation the character can perceive.
type PerceptionKind string

// The same initial perception aptitude applies to hearing and sight.
func perceptionAptitude(seed int64, actor string) int {
	return int(seededSample(seed, "aptitude/perception/"+actor, 0) % 21)
}

const (
	OnStreet        PerceptionKind = "on_street"
	InBakeryDoorway PerceptionKind = "in_bakery_doorway"
	InBakery        PerceptionKind = "in_bakery"
	SittingDown     PerceptionKind = "sitting_down"
	TiredLegs       PerceptionKind = "tired_legs"
	FeelingRelieved PerceptionKind = "feeling_relieved"
	LingeringRelief PerceptionKind = "lingering_relief"
	Footsteps       PerceptionKind = "footsteps"
	DoorCreak       PerceptionKind = "door_creak"
)

// Version 1 receipts save these exact sentences. Live scenes use the kinds
// directly, and old journals recover them from the observer's own receipt.
var receiptDescriptions = map[PerceptionKind]string{
	OnStreet:        "You are on the street.",
	InBakeryDoorway: "You are in the bakery doorway.",
	InBakery:        "You are in the bakery.",
	SittingDown:     "You are sitting down.",
	TiredLegs:       "Your legs feel tired.",
	FeelingRelieved: "You feel relieved.",
	LingeringRelief: "A little of that relief lingers.",
	Footsteps:       "You hear footsteps.",
	DoorCreak:       "You hear a door creak.",
}

func (k PerceptionKind) Description() string { return receiptDescriptions[k] }

// JournalObservation preserves the observer's location or personal sensations,
// including any wording with no recognized perception kind.
type JournalObservation struct {
	Kind PerceptionKind
	Text string
}

func (o JournalObservation) SameAs(other JournalObservation) bool {
	if o.Kind != "" {
		return o.Kind == other.Kind
	}
	return o == other
}

func personalObservation(text string) JournalObservation {
	for kind, description := range receiptDescriptions {
		if text == description {
			return JournalObservation{Kind: kind, Text: text}
		}
	}
	return JournalObservation{Text: text}
}
