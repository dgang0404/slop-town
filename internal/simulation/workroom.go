// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	_ "embed"
	"fmt"
	"strings"
)

// Rows run from the main aisle toward the back wall. Mirroring the same
// footprint fits the workroom on either side of the bakery's work route.
//
//go:embed bakery-workroom.layout
var workroomPlan string

func (b *Bakery) workroomOrigin(prefix string) (CellPosition, int) {
	at := b.Entities[prefix+"door"].Placement.Cell
	step := 1
	if at.Y > 1 {
		step = -1
	}
	at.X++
	at.Y += step
	return at, step
}

func (b *Bakery) addWorkroom(prefix string) error {
	at, step := b.workroomOrigin(prefix)
	b.GeneratorVersion = 4
	b.ContentVersion = supplyQuestionVersion
	for row, line := range strings.Split(strings.TrimSpace(workroomPlan), "\n") {
		for x, tile := range line {
			cell := CellPosition{b.FloorID, at.X + x, at.Y + row*step}
			if !b.isFloorCell(cell) {
				return fmt.Errorf("workroom footprint overlaps a wall or map edge")
			}
			switch tile {
			case '.': // Retain the existing floor.
			case '#':
				b.Cells[cell.Y][cell.X].Wall = true
			case 'D':
				b.Entities[prefix+"workroom-door"] = SpatialEntity{Name: "workroom door", Kind: Door, Placement: Placement{Cell: cell}}
			case 'W':
				b.Cells[cell.Y][cell.X].SpaceID = prefix + "workroom"
				b.Entities[prefix+"workbench"] = SpatialEntity{Name: "workbench drawer", Kind: Container, Placement: Placement{Cell: cell}}
			default:
				return fmt.Errorf("unknown workroom tile %q", tile)
			}
		}
	}
	if !b.validWorkroom(prefix) {
		return fmt.Errorf("invalid workroom footprint")
	}
	return nil
}

func (b *Bakery) validWorkroom(prefix string) bool {
	at, step := b.workroomOrigin(prefix)
	door := b.Entities[prefix+"workroom-door"]
	bench := b.Entities[prefix+"workbench"]
	entry := CellPosition{b.FloorID, at.X + 1, at.Y}
	station := CellPosition{b.FloorID, entry.X, entry.Y + step}
	return b.isFloorCell(entry) && b.isFloorCell(station) && door.Kind == Door && door.Placement.Cell == entry &&
		bench.Kind == Container && bench.Placement.Cell == station && b.Cells[station.Y][station.X].SpaceID == prefix+"workroom" && b.Cells[station.Y][at.X].Wall
}
