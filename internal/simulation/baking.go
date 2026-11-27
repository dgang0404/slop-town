// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "fmt"

// Two qualifying kneads earn one lasting improvement in this bakery.
const trainedKneads = 2

func (b *Bakery) knead(actor SpatialEntity, actorID string) (spatialEffect, *actionFailure) {
	if b.ContentVersion < bakingContentVersion || actor.Kind != Actor {
		return spatialEffect{}, &actionFailure{"action_unavailable", "You can't knead dough here."}
	}
	cell := actor.Placement.Cell
	if cell != b.PrepCell || !b.isFloorCell(cell) || b.Cells[cell.Y][cell.X].SpaceID != b.InteriorID {
		return spatialEffect{}, &actionFailure{"action_unavailable", "You need to stand at the prep bench to knead dough."}
	}
	light := actor.BakingBouts >= 1
	message := "You knead the dough into a dense loaf."
	if light {
		message = "You knead the dough until it feels light and springy."
	}
	practice := false
	if actor.BakingBouts < 1 {
		actor.BakingKneads++
		if actor.BakingKneads >= trainedKneads {
			actor.BakingBouts = 1
			practice = true
		}
	}
	return spatialEffect{
		kind:     "dough_kneaded",
		payload:  EventPayload{BakingPractice: practice},
		entityID: actorID,
		entity:   actor,
		message:  message,
	}, nil
}

func (b *Bakery) validateBaking(id string, actor SpatialEntity) error {
	if b.ContentVersion < bakingContentVersion {
		if actor.BakingBouts != 0 || actor.BakingKneads != 0 {
			return fmt.Errorf("unexpected baking practice for %q", id)
		}
		return nil
	}
	if actor.Kind != Actor {
		if actor.BakingBouts != 0 || actor.BakingKneads != 0 {
			return fmt.Errorf("unexpected baking practice for %q", id)
		}
		return nil
	}
	if actor.BakingBouts < 0 || actor.BakingBouts > 1 || actor.BakingKneads < 0 || actor.BakingKneads > trainedKneads {
		return fmt.Errorf("invalid baking practice for %q", id)
	}
	if actor.BakingKneads == trainedKneads && actor.BakingBouts != 1 {
		return fmt.Errorf("missing baking skill for %q", id)
	}
	if actor.BakingBouts == 0 && actor.BakingKneads == trainedKneads {
		return fmt.Errorf("invalid baking skill for %q", id)
	}
	return nil
}