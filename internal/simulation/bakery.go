// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"net/url"
	"strings"
)

const bakeryWidth, bakeryHeight = 5, 4

// CellPosition addresses one cell on a world-scoped floor.
type CellPosition struct {
	FloorID string
	X, Y    int
}

type BakeryCell struct {
	SpaceID string
	Wall    bool
}

func (b *Bakery) isFloorCell(cell CellPosition) bool {
	return cell.FloorID == b.FloorID &&
		cell.X >= 0 && cell.X < bakeryWidth && cell.Y >= 0 && cell.Y < bakeryHeight &&
		!b.Cells[cell.Y][cell.X].Wall
}

// Placement has one direct parent. A holder is either an inventory owner or
// a container. Its contents derive their effective position from that holder.
type Placement struct {
	Cell     CellPosition
	HolderID string
}

type EntityKind string

const (
	Actor     EntityKind = "actor"
	Door      EntityKind = "door"
	Container EntityKind = "container"
	Item      EntityKind = "item"
)

type SpatialEntity struct {
	Name      string
	Kind      EntityKind
	Placement Placement
	Open      bool
	Portable  bool
	Fatigue   int  `json:",omitempty"`
	Seated    bool `json:",omitempty"`
}

// Bakery is authoritative fixture state. Cells are in row-major order and
// entity IDs are independent of position, inventory and timeline.
type Bakery struct {
	WorldID          string
	FloorID          string
	StreetID         string
	InteriorID       string
	Seed             int64
	GeneratorVersion int
	ContentVersion   int
	Cells            [bakeryHeight][bakeryWidth]BakeryCell
	Entities         map[string]SpatialEntity
}

// BakerySetup selects the content of a new timeline. The save supplies the
// content version when resuming an existing timeline.
type BakerySetup struct {
	Resident bool
}

const (
	residentContentVersion = 2 // The original work routine.
	restContentVersion     = 3
	fatigueLimit           = 5 // Steps since the last rest, capped during a break.
)

func (setup BakerySetup) Generate(worldID string, clock *Timeline) (*Bakery, error) {
	b, err := GenerateBakery(worldID, clock)
	if err != nil || !setup.Resident {
		return b, err
	}
	prefix := url.PathEscape(worldID) + "/"
	station := b.Entities[prefix+"door"].Placement
	station.Cell.X++
	b.ContentVersion = restContentVersion
	b.Entities[prefix+"baker"] = SpatialEntity{Name: "baker", Kind: Actor, Placement: station}
	return b, nil
}

// GenerateBakery initializes a world at tick zero using the first geography
// draw. Rejecting a late or repeated initialization consumes no randomness.
func GenerateBakery(worldID string, timeline *Timeline) (*Bakery, error) {
	if !validID(worldID) || timeline == nil {
		return nil, fmt.Errorf("bakery generation requires a world ID and a timeline")
	}
	if timeline.Tick() != 0 || timeline.draws["geography"] != 0 {
		return nil, fmt.Errorf("bakery generation requires tick zero and an unused geography stream")
	}
	draw, err := timeline.Draw("geography")
	if err != nil {
		return nil, err
	}
	row := 1 + int(draw&1)
	prefix := url.PathEscape(worldID) + "/"
	b := &Bakery{
		WorldID: worldID, FloorID: prefix + "ground",
		StreetID: prefix + "street", InteriorID: prefix + "bakery",
		Seed: timeline.seed, GeneratorVersion: 1, ContentVersion: 1,
	}
	for y := range b.Cells {
		for x := range b.Cells[y] {
			space := b.InteriorID
			if x < 2 {
				space = b.StreetID
			}
			b.Cells[y][x] = BakeryCell{SpaceID: space, Wall: x == 2 && y != row}
		}
	}
	onFloor := func(x, y int) Placement {
		return Placement{Cell: CellPosition{FloorID: b.FloorID, X: x, Y: y}}
	}
	b.Entities = map[string]SpatialEntity{
		prefix + "player": {Name: "player", Kind: Actor, Placement: onFloor(1, row)},
		prefix + "door":   {Name: "bakery door", Kind: Door, Placement: onFloor(2, row)},
		prefix + "crate":  {Name: "crate", Kind: Container, Placement: onFloor(3, row-1)},
		prefix + "spoon":  {Name: "wooden spoon", Kind: Item, Portable: true, Placement: Placement{HolderID: prefix + "crate"}},
	}
	return b, nil
}

// PositionOf follows the placement chain to its floor cell, checking each
// reference. The holder's position determines the position of its contents.
func (b *Bakery) PositionOf(id string) (CellPosition, error) {
	seen := make(map[string]bool)
	for {
		entity, ok := b.Entities[id]
		if !ok {
			return CellPosition{}, fmt.Errorf("unknown entity %q", id)
		}
		if seen[id] {
			return CellPosition{}, fmt.Errorf("placement cycle at %q", id)
		}
		seen[id] = true
		p := entity.Placement
		if p.HolderID == "" {
			cell := p.Cell
			if !b.isFloorCell(cell) {
				return CellPosition{}, fmt.Errorf("invalid floor cell for %q", id)
			}
			return cell, nil
		}
		if p.Cell != (CellPosition{}) {
			return CellPosition{}, fmt.Errorf("entity %q has both a cell and a holder", id)
		}
		holder, ok := b.Entities[p.HolderID]
		if !ok || (holder.Kind != Container && holder.Kind != Actor) {
			return CellPosition{}, fmt.Errorf("invalid holder %q for %q", p.HolderID, id)
		}
		id = p.HolderID
	}
}

// Validate checks space references and every entity's chain of parents.
func (b *Bakery) Validate() error {
	prefix := url.PathEscape(b.WorldID) + "/"
	if !validID(b.WorldID) || b.FloorID != prefix+"ground" || b.StreetID != prefix+"street" || b.InteriorID != prefix+"bakery" {
		return fmt.Errorf("invalid bakery space identities")
	}
	for y, row := range b.Cells {
		for x, cell := range row {
			if cell.SpaceID != b.StreetID && cell.SpaceID != b.InteriorID {
				return fmt.Errorf("cell (%d,%d) references unknown space %q", x, y, cell.SpaceID)
			}
		}
	}
	for id, e := range b.Entities {
		localID, inWorld := strings.CutPrefix(id, prefix)
		if !inWorld || !validID(localID) {
			return fmt.Errorf("invalid entity ID %q for world %q", id, b.WorldID)
		}
		if id == b.FloorID || id == b.StreetID || id == b.InteriorID {
			return fmt.Errorf("entity ID %q is already used by a space", id)
		}
		if _, err := b.PositionOf(id); err != nil {
			return err
		}
		if (e.Fatigue != 0 || e.Seated) && (e.Kind != Actor || b.ContentVersion < restContentVersion) {
			return fmt.Errorf("unexpected rest state for %q", id)
		}
		if e.Fatigue < 0 || e.Fatigue > fatigueLimit || (e.Seated && e.Fatigue != 0) {
			return fmt.Errorf("invalid fatigue for %q", id)
		}
	}
	return nil
}
