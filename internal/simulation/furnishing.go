// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed bakery-supplies.en.json
var bakerySupplies []byte

// Furnish only at creation. Both the names and the selected cells go into
// the snapshot so resuming a town never has to consult this recipe again.
func (b *Bakery) furnish(clock *Timeline, prefix string) error {
	if clock.draws["furnishing"] != 0 {
		return fmt.Errorf("bakery furnishings require an unused stream")
	}
	door := b.Entities[prefix+"door"].Placement.Cell
	prep, _ := door.neighbor(East)
	shopfront, _ := door.neighbor(West)
	used := make(map[CellPosition]bool)
	if b.GeneratorVersion >= 3 {
		used[b.Entities[prefix+"workroom-door"].Placement.Cell] = true
	}
	for _, slot := range []struct{ id, space string }{
		{"bakery-phone", b.InteriorID}, {"crate", b.InteriorID},
		{"street-phone", b.StreetID}, {"mint", b.StreetID},
	} {
		var choices []CellPosition
		for y, row := range b.Cells {
			for x, cell := range row {
				at := CellPosition{b.FloorID, x, y}
				if cell.Wall || cell.SpaceID != slot.space || used[at] {
					continue
				}
				// Leave the street walking lane, doorway, prep and rest stations clear.
				if slot.space == b.StreetID && x >= shopfront.X ||
					slot.space == b.InteriorID && (x <= door.X || (y != door.Y-1 && y != door.Y+1)) {
					continue
				}
				if slot.id == "bakery-phone" && !inReach(prep, at) {
					continue
				}
				choices = append(choices, at)
			}
		}
		if len(choices) == 0 {
			return fmt.Errorf("no available %s location", slot.id)
		}
		draw, err := clock.Draw("furnishing")
		if err != nil {
			return err
		}
		at := choices[draw%uint64(len(choices))]
		used[at] = true
		entity := b.Entities[prefix+slot.id]
		entity.Placement = Placement{Cell: at}
		b.Entities[prefix+slot.id] = entity
	}
	var supplies []struct{ ID, Name, Detail string }
	if err := json.Unmarshal(bakerySupplies, &supplies); err != nil {
		return err
	}
	if len(supplies) == 0 {
		return fmt.Errorf("no bakery supplies")
	}
	draw, err := clock.Draw("furnishing")
	if err != nil {
		return err
	}
	supply := supplies[draw%uint64(len(supplies))]
	if _, exists := b.Entities[prefix+supply.ID]; exists || !validID(supply.ID) || supply.Name == "" || supply.Detail == "" {
		return fmt.Errorf("invalid bakery supply %q", supply.ID)
	}
	b.Entities[prefix+supply.ID] = SpatialEntity{Name: supply.Name, Kind: Item, Portable: true,
		Placement: Placement{HolderID: prefix + "crate"}, CloseDetail: supply.Detail, DropSound: Clatter}
	if b.GeneratorVersion == 4 {
		b.Entities[prefix+"order"] = SpatialEntity{
			Name: "supply order", Kind: Item, Placement: b.Entities[prefix+"workbench"].Placement,
			RequestedItem: prefix + supply.ID,
			CloseDetail:   "a handwritten request to leave the " + supply.Name + " in the workbench drawer",
		}
	}
	if b.GeneratorVersion == 1 {
		b.GeneratorVersion = 2
	}
	if err := b.Validate(); err != nil {
		return err
	}
	return b.checkFurnishingAccess(prefix)
}

// At creation require an aisle to every resident and an adjacent place to
// stand at every fixture, even when all furniture is treated as impassable.
// The front door opens from the aisle. Its square connects the two spaces.
func (b *Bakery) checkFurnishingAccess(prefix string) error {
	var occupied, reached cellMask
	for _, e := range b.Entities {
		if e.Kind == Telephone || e.Kind == Planter || e.Kind == Container {
			at := e.Placement.Cell
			if occupied[at.Y][at.X] {
				return fmt.Errorf("fixtures overlap at (%d,%d)", at.X, at.Y)
			}
			occupied[at.Y][at.X] = true
		}
	}
	start := b.Entities[prefix+"player"].Placement.Cell
	if occupied[start.Y][start.X] {
		return fmt.Errorf("player starts on furniture")
	}
	queue := []CellPosition{start}
	reached[start.Y][start.X] = true
	for i := 0; i < len(queue); i++ {
		for _, dir := range []Direction{North, East, South, West} {
			next, _ := queue[i].neighbor(dir)
			if b.isFloorCell(next) && !occupied[next.Y][next.X] && !reached[next.Y][next.X] {
				reached[next.Y][next.X] = true
				queue = append(queue, next)
			}
		}
	}
	for id, e := range b.Entities {
		if e.Placement.HolderID != "" {
			continue // Contents have the same approach as their container.
		}
		at := e.Placement.Cell
		if reached[at.Y][at.X] {
			continue
		}
		accessible := false
		for _, dir := range []Direction{North, East, South, West} {
			next, _ := at.neighbor(dir)
			if b.isFloorCell(next) && reached[next.Y][next.X] {
				accessible = true
				break
			}
		}
		if !accessible || e.Kind == Actor || e.Kind == Door {
			return fmt.Errorf("no free aisle to %s", id)
		}
	}
	return nil
}
