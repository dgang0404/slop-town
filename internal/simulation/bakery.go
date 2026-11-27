// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"maps"
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
	Telephone EntityKind = "telephone"
	Actor     EntityKind = "actor"
	Door      EntityKind = "door"
	Container EntityKind = "container"
	Item      EntityKind = "item"
	Planter   EntityKind = "planter" // A fixed outdoor planter with its own watering spout.
)

type SpatialEntity struct {
	Name               string
	Kind               EntityKind
	Placement          Placement
	Open               bool
	Portable           bool
	Fatigue            int             `json:",omitempty"`
	Seated             bool            `json:",omitempty"`
	RestInPlacePercent int             `json:",omitempty"`
	SeekingRestSpot    bool            `json:",omitempty"`
	WalkedCells        uint32          `json:",omitempty"`
	EnduranceBouts     int             `json:",omitempty"`
	ReliefTurns        int             `json:",omitempty"`
	DropSound          PerceptionKind  `json:",omitempty"` // Sound made when this item hits the floor.
	StealthPractice    stealthPractice `json:",omitzero"`
	ProvidesCover      bool            `json:",omitempty"`
	HiddenBehind       string          `json:",omitempty"` // Reference to the cover beside a crouching actor.
	CloseDetail        string          `json:",omitempty"` // A visible distinguishing mark within arm's reach.
	Disconnected       bool            `json:",omitempty"` // A fixed telephone whose line is out of service.
	Growth             plantGrowth     `json:",omitzero"`
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
	Contacts         []sharedContact            `json:"Addresses,omitempty"` // Original save field for contact disclosures.
	Letters          []letter                   `json:",omitempty"`
	Notebooks        map[string][]NotebookEntry `json:",omitempty"`
}

// BakerySetup selects the content of a new timeline. The save supplies the
// content version when resuming an existing timeline.
type BakerySetup struct {
	Resident bool
	Practice bool // Includes both workers in a bakery with use-based endurance.
}

const (
	residentContentVersion        = 2 // The original work routine.
	restContentVersion            = 3
	notebookContentVersion        = 4
	personalityContentVersion     = 5
	practiceContentVersion        = 6
	emotionContentVersion         = 7
	relationshipContentVersion    = 8
	conversationContentVersion    = 9
	reportContentVersion          = 10
	itemWitnessContentVersion     = 11
	discreetTakeContentVersion    = 12
	listenContentVersion          = 13
	soundAccountContentVersion    = 14
	dropSoundContentVersion       = 15
	daylightContentVersion        = 16
	stealthPracticeContentVersion = 17
	coverContentVersion           = 18
	knockContentVersion           = 19
	examineContentVersion         = 20
	plantContentVersion           = 21
	harvestContentVersion         = 22
	landmarkContentVersion        = 23
	mailContentVersion            = 24
	phoneContentVersion           = 25
	fatigueLimit                  = 5 // Steps since the last rest, capped during a break.
)

func (setup BakerySetup) Generate(worldID string, clock *Timeline) (*Bakery, error) {
	if (!setup.Resident && !setup.Practice) || clock == nil {
		return GenerateBakery(worldID, clock)
	}
	// Publish the geography and both profile draws together.
	staged := *clock
	staged.draws = maps.Clone(clock.draws)
	b, err := generateNotebookBakery(worldID, &staged)
	if err != nil {
		return nil, err
	}
	for _, role := range []string{"baker", "porter"} {
		draw, err := staged.Draw("personality/" + role)
		if err != nil {
			return nil, err
		}
		id := url.PathEscape(worldID) + "/" + role
		resident := b.Entities[id]
		resident.RestInPlacePercent = 1 + int(draw%99)
		b.Entities[id] = resident
	}
	b.ContentVersion = personalityContentVersion
	if setup.Practice {
		b.ContentVersion = phoneContentVersion
		prefix := url.PathEscape(worldID) + "/"
		for localID, detail := range bakeryCloseDetails {
			id := prefix + localID
			entity := b.Entities[id]
			entity.CloseDetail = detail
			b.Entities[id] = entity
		}
		spoon := b.Entities[prefix+"spoon"]
		spoon.DropSound = Clatter
		b.Entities[prefix+"spoon"] = spoon
		crate := b.Entities[prefix+"crate"]
		crate.ProvidesCover = true
		b.Entities[prefix+"crate"] = crate
		streetPhone := b.Entities[prefix+"player"].Placement
		streetPhone.Cell.Y++
		insidePhone := b.Entities[prefix+"baker"].Placement
		insidePhone.Cell.X++
		b.Entities[prefix+"street-phone"] = SpatialEntity{Name: "street phone", Kind: Telephone, Placement: streetPhone}
		b.Entities[prefix+"bakery-phone"] = SpatialEntity{Name: "bakery phone", Kind: Telephone, Placement: insidePhone}
		at := b.Entities[prefix+"player"].Placement
		at.Cell.X--
		b.Entities[prefix+"mint"] = SpatialEntity{Name: "mint planter", Kind: Planter, Placement: at,
			CloseDetail: "a brass watering spout over the soil"}
	}
	*clock = staged
	return b, nil
}

func generateNotebookBakery(worldID string, clock *Timeline) (*Bakery, error) {
	b, err := GenerateBakery(worldID, clock)
	if err != nil {
		return nil, err
	}
	prefix := url.PathEscape(worldID) + "/"
	station := b.Entities[prefix+"door"].Placement
	station.Cell.X++
	b.ContentVersion = notebookContentVersion
	b.Entities[prefix+"baker"] = SpatialEntity{Name: "baker", Kind: Actor, Placement: station}
	street := b.Entities[prefix+"player"].Placement
	street.Cell.Y = bakeryHeight - 1
	b.Entities[prefix+"porter"] = SpatialEntity{Name: "porter", Kind: Actor, Placement: street}
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
		if e.Kind == Telephone && (b.ContentVersion < phoneContentVersion || e.Portable || e.Placement.HolderID != "") {
			return fmt.Errorf("invalid telephone %q", id)
		}
		if e.Disconnected && e.Kind != Telephone {
			return fmt.Errorf("unexpected telephone state for %q", id)
		}
		if e.Kind == Planter {
			if b.ContentVersion < plantContentVersion || e.Portable || e.Placement.HolderID != "" ||
				b.Cells[e.Placement.Cell.Y][e.Placement.Cell.X].SpaceID != b.StreetID {
				return fmt.Errorf("invalid outdoor planter %q", id)
			}
			if b.ContentVersion >= harvestContentVersion {
				item, exists := b.Entities[sprigID(id)]
				if exists != e.Growth.Harvested || (exists && (item.Kind != Item || !item.Portable || item.Name != "mint sprig")) {
					return fmt.Errorf("invalid harvested sprig for %q", id)
				}
			} else if e.Growth.Harvested {
				return fmt.Errorf("unexpected harvest for %q", id)
			}
		} else if e.Growth != (plantGrowth{}) {
			return fmt.Errorf("unexpected plant growth for %q", id)
		}
		if e.CloseDetail != "" && b.ContentVersion < examineContentVersion {
			return fmt.Errorf("unexpected examination detail for %q", id)
		}
		if e.ProvidesCover && (b.ContentVersion < coverContentVersion || e.Kind != Container || e.Portable || e.Placement.HolderID != "") {
			return fmt.Errorf("invalid cover for %q", id)
		}
		if e.HiddenBehind != "" && (e.Kind != Actor || e.Seated || !b.canHideBehind(e, e.HiddenBehind)) {
			return fmt.Errorf("invalid hiding place for %q", id)
		}
		if e.DropSound != "" && (b.ContentVersion < dropSoundContentVersion || e.Kind != Item || !e.Portable || e.DropSound != Clatter) {
			return fmt.Errorf("invalid drop sound for %q", id)
		}
		if e.StealthPractice != (stealthPractice{}) && (e.Kind != Actor || b.ContentVersion < stealthPracticeContentVersion) {
			return fmt.Errorf("unexpected stealth practice for %q", id)
		}
		if e.ReliefTurns != 0 && (e.Kind != Actor || b.ContentVersion < emotionContentVersion || e.ReliefTurns < 0 || e.ReliefTurns > reliefDuration) {
			return fmt.Errorf("invalid relief for %q", id)
		}
		if err := b.validatePractice(id, e); err != nil {
			return err
		}
		hasProfile := b.ContentVersion >= personalityContentVersion && (id == prefix+"baker" || id == prefix+"porter")
		if hasProfile {
			if e.Kind != Actor || e.RestInPlacePercent < 1 || e.RestInPlacePercent > 99 {
				return fmt.Errorf("invalid rest preference for %q", id)
			}
		} else if e.RestInPlacePercent != 0 || e.SeekingRestSpot {
			return fmt.Errorf("unexpected rest preference for %q", id)
		}
		if e.SeekingRestSpot && e.Fatigue != fatigueLimit {
			return fmt.Errorf("invalid break in progress for %q", id)
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
