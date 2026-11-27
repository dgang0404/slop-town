// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"cmp"
	"fmt"
	"slices"
)

type plantGrowth struct {
	WateredTick     int64
	WateredSequence uint64
	NewLeaves       bool
	Harvested       bool   `json:",omitempty"`
	HarvestTick     int64  `json:",omitempty"`
	HarvestSequence uint64 `json:",omitempty"`
	HarvestCount    int    `json:",omitempty"`
}

// The mint starts with a pair of folded leaves. After watering, individual
// plants take four to six more town turns to unfold them.
func (b *Bakery) leafDelay(id string) int64 {
	return 4 + int64(seededSample(b.Seed, "growth/mint/"+id, 0)%3)
}

func (b *Bakery) regrowthDelay(id string) int64 {
	return 4 + int64(seededSample(b.Seed, "growth/mint-regrow/"+id, 0)%3)
}

func (g plantGrowth) valid(tick, leafDelay, regrowthDelay int64, regrowth bool) bool {
	if g.WateredTick == 0 {
		return g == (plantGrowth{})
	}
	if g.WateredTick <= 0 || g.WateredTick > tick || g.WateredSequence == 0 {
		return false
	}
	if g.HarvestCount == 0 {
		unfolded := tick-g.WateredTick >= leafDelay
		return g.NewLeaves == unfolded && (!g.Harvested || g.NewLeaves)
	}
	if !regrowth {
		return g.Harvested && g.HarvestCount <= 1
	}
	if g.HarvestCount > 2 || g.HarvestTick <= 0 || g.HarvestSequence == 0 {
		return false
	}
	if g.HarvestCount == 2 {
		return !g.NewLeaves
	}
	regrown := tick-g.HarvestTick >= regrowthDelay
	return g.NewLeaves == regrown
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

// Each cutting keeps its own ID through inventory and storage.
func sprigID(planter string, harvest int) string {
	if harvest <= 1 {
		return planter + "/sprig"
	}
	return fmt.Sprintf("%s/sprig-%d", planter, harvest)
}

func (b *Bakery) harvest(actorID, ref string) (spatialEffect, *actionFailure) {
	plant := b.Entities[ref]
	if b.ContentVersion < harvestContentVersion || plant.Kind != Planter || !inReach(b.Entities[actorID].Placement.Cell, plant.Placement.Cell) {
		return spatialEffect{}, targetUnavailable()
	}
	switch plant.Growth.condition(b.ContentVersion >= regrowthContentVersion) {
	case clippedStems:
		return spatialEffect{}, &actionFailure{"already_in_state", "Only clipped stems remain."}
	case newLeaves:
		count := plant.Growth.HarvestCount + 1
		if b.ContentVersion >= regrowthContentVersion {
			if count > 2 {
				return spatialEffect{}, &actionFailure{"already_in_state", "Only clipped stems remain."}
			}
			plant.Growth.HarvestCount = count
			plant.Growth.Harvested = true
			plant.Growth.NewLeaves = false
		} else {
			plant.Growth.Harvested = true
		}
		return spatialEffect{kind: "plant_harvested", payload: EventPayload{Target: ref, ItemID: sprigID(ref, count), HolderID: actorID},
			entityID: ref, entity: plant, message: "You harvest a mint sprig from the " + plant.Name + "."}, nil
	default:
		return spatialEffect{}, &actionFailure{"action_unavailable", "There are no new leaves to harvest yet."}
	}
}

// Publish due growth after the workers move and before the end-of-turn view.
// The watering remains the causal parent even after a save and restart.
func (b *Bakery) advancePlants(events []Event) []Event {
	turn := events[0]
	type unfold struct {
		id      string
		regrown bool
	}
	var due []unfold
	for id, e := range b.Entities {
		if e.Kind != Planter {
			continue
		}
		g := e.Growth
		if g.NewLeaves {
			continue
		}
		if g.HarvestCount == 0 && g.WateredTick > 0 && turn.Tick-g.WateredTick >= b.leafDelay(id) {
			due = append(due, unfold{id: id})
			continue
		}
		if b.ContentVersion >= regrowthContentVersion && g.HarvestCount == 1 && g.HarvestTick > 0 &&
			turn.Tick-g.HarvestTick >= b.regrowthDelay(id) {
			due = append(due, unfold{id: id, regrown: true})
		}
	}
	slices.SortFunc(due, func(a, b unfold) int { return cmp.Compare(a.id, b.id) })
	for _, job := range due {
		plant := b.Entities[job.id]
		plant.Growth.NewLeaves = true
		b.Entities[job.id] = plant
		growth := turn.child(events[len(events)-1].Sequence+1, job.id, "plants", "leaves_unfolded", EventPayload{Target: job.id})
		if job.regrown {
			growth.ParentID = eventID(turn.Scope, plant.Growth.HarvestSequence)
		} else {
			growth.ParentID = eventID(turn.Scope, plant.Growth.WateredSequence)
		}
		events = append(events, growth)
	}
	return events
}

const newLeavesLabel = " with new leaves"
const clippedStemsLabel = " with clipped stems"

// PlantCondition is the soil or foliage a character can see.
type PlantCondition string

const (
	soilDry      PlantCondition = "dry_soil"
	soilDamp     PlantCondition = "damp_soil"
	newLeaves    PlantCondition = "new_leaves"
	clippedStems PlantCondition = "clipped_stems"
)

func (g plantGrowth) condition(regrowth bool) PlantCondition {
	if g.NewLeaves {
		if g.Harvested && !regrowth {
			return clippedStems
		}
		return newLeaves
	}
	if regrowth && g.HarvestCount > 0 {
		return clippedStems
	}
	if !regrowth && g.Harvested {
		return clippedStems
	}
	if g.WateredTick != 0 {
		return soilDamp
	}
	return soilDry
}

func (condition PlantCondition) appearance(name string) (label, glyph, state string) {
	switch condition {
	case clippedStems:
		return name + clippedStemsLabel, "y", "cut back to short stems"
	case newLeaves:
		return name + newLeavesLabel, "Y", "putting out new leaves"
	case soilDamp:
		return "damp " + name, "y", "damp"
	default:
		return "dry " + name, "y", "dry"
	}
}
