// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "slices"

type plantGrowth struct {
	WateredTick     int64
	WateredSequence uint64
	NewLeaves       bool
}

// The mint starts with a pair of folded leaves. After watering, individual
// plants take four to six more town turns to unfold them.
func (b *Bakery) leafDelay(id string) int64 {
	return 4 + int64(seededSample(b.Seed, "growth/mint/"+id, 0)%3)
}

func (g plantGrowth) valid(tick, delay int64) bool {
	if g.WateredTick == 0 {
		return g == (plantGrowth{})
	}
	return g.WateredTick > 0 && g.WateredTick <= tick && g.WateredSequence > 0 &&
		g.NewLeaves == (tick-g.WateredTick >= delay)
}

func (b *Bakery) water(actor SpatialEntity, ref string) (spatialEffect, *actionFailure) {
	plant := b.Entities[ref]
	if b.ContentVersion < plantContentVersion || plant.Kind != Planter || !inReach(actor.Placement.Cell, plant.Placement.Cell) {
		return spatialEffect{}, targetUnavailable()
	}
	if plant.Growth.WateredTick != 0 {
		return spatialEffect{}, &actionFailure{"already_in_state", "The soil is already damp."}
	}
	return spatialEffect{kind: "plant_watered", payload: EventPayload{Target: ref}, entityID: ref, entity: plant,
		message: "You water the " + plant.Name + " using its spout."}, nil
}

// Publish due growth after the workers move and before the end-of-turn view.
// The watering remains the causal parent even after a save and restart.
func (b *Bakery) advancePlants(events []Event) []Event {
	turn := events[0]
	var due []string
	for id, e := range b.Entities {
		g := e.Growth
		if e.Kind == Planter && g.WateredTick > 0 && !g.NewLeaves && turn.Tick-g.WateredTick >= b.leafDelay(id) {
			due = append(due, id)
		}
	}
	slices.Sort(due)
	for _, id := range due {
		plant := b.Entities[id]
		plant.Growth.NewLeaves = true
		b.Entities[id] = plant
		growth := turn.child(events[len(events)-1].Sequence+1, id, "plants", "leaves_unfolded", EventPayload{Target: id})
		growth.ParentID = eventID(turn.Scope, plant.Growth.WateredSequence)
		events = append(events, growth)
	}
	return events
}

const newLeavesLabel = " with new leaves"

// PlantCondition is the soil or foliage a character can see.
type PlantCondition string

const (
	soilDry   PlantCondition = "dry_soil"
	soilDamp  PlantCondition = "damp_soil"
	newLeaves PlantCondition = "new_leaves"
)

func (g plantGrowth) condition() PlantCondition {
	switch {
	case g.NewLeaves:
		return newLeaves
	case g.WateredTick != 0:
		return soilDamp
	default:
		return soilDry
	}
}

func (condition PlantCondition) appearance(name string) (label, glyph, state string) {
	switch condition {
	case newLeaves:
		return name + newLeavesLabel, "Y", "putting out new leaves"
	case soilDamp:
		return "damp " + name, "y", "damp"
	default:
		return "dry " + name, "y", "dry"
	}
}
