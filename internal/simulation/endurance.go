// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"math/bits"
)

// This small bakery offers one attainable improvement. Each bit identifies a
// destination reached on this floor since the actor last sat down.
const trainedBouts = 2

func walkingCell(cell CellPosition) uint32 {
	return uint32(1) << (cell.Y*bakeryWidth + cell.X)
}

func (actor *SpatialEntity) walkStep(version int, dest CellPosition) {
	if version >= practiceContentVersion {
		if actor.EnduranceBouts < trainedBouts {
			actor.WalkedCells |= walkingCell(dest)
		} else if actor.Seated {
			return // The first step after recovery costs a trained walker no fatigue.
		}
	}
	actor.Fatigue = min(actor.Fatigue+1, fatigueLimit)
}

func (actor *SpatialEntity) recoverEndurance() bool {
	qualified := actor.Fatigue == fatigueLimit && bits.OnesCount32(actor.WalkedCells) >= 4
	actor.WalkedCells = 0
	if qualified {
		actor.EnduranceBouts++
	}
	return qualified
}

func (b *Bakery) validatePractice(id string, actor SpatialEntity) error {
	if b.ContentVersion < practiceContentVersion || actor.Kind != Actor {
		if actor.EnduranceBouts != 0 || actor.WalkedCells != 0 {
			return fmt.Errorf("unexpected endurance practice for %q", id)
		}
		return nil
	}
	if actor.EnduranceBouts < 0 || actor.EnduranceBouts > trainedBouts {
		return fmt.Errorf("invalid endurance practice for %q", id)
	}
	if actor.WalkedCells == 0 {
		if actor.EnduranceBouts < trainedBouts && actor.Fatigue != 0 {
			return fmt.Errorf("missing walking route for %q", id)
		}
		return nil
	}
	if actor.EnduranceBouts == trainedBouts || actor.Fatigue == 0 ||
		actor.WalkedCells>>(bakeryWidth*bakeryHeight) != 0 ||
		actor.WalkedCells&walkingCell(actor.Placement.Cell) == 0 ||
		(actor.Fatigue < fatigueLimit && bits.OnesCount32(actor.WalkedCells) > actor.Fatigue) {
		return fmt.Errorf("invalid walking route for %q", id)
	}
	for y, row := range b.Cells {
		for x, cell := range row {
			if cell.Wall && actor.WalkedCells&walkingCell(CellPosition{X: x, Y: y}) != 0 {
				return fmt.Errorf("walking route crosses a wall for %q", id)
			}
		}
	}
	return nil
}
