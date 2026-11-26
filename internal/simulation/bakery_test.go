// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"reflect"
	"testing"
)

func makeBakery(t *testing.T, world string, timeline *Timeline) *Bakery {
	t.Helper()
	b, err := GenerateBakery(world, timeline)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBakeryReferenceLayouts(t *testing.T) {
	for _, tc := range []struct {
		seed int64
		row  int
	}{{42, 1}, {43, 2}} {
		first, second := NewTimeline(tc.seed), NewTimeline(tc.seed)
		b := makeBakery(t, "town", first)
		repeated := makeBakery(t, "town", second)
		if !reflect.DeepEqual(b, repeated) || !bytes.Equal(snapshot(t, first), snapshot(t, second)) {
			t.Fatalf("seed %d did not reproduce the same state", tc.seed)
		}
		if b.Seed != tc.seed || b.GeneratorVersion != 1 || b.ContentVersion != 1 || first.Tick() != 0 ||
			!reflect.DeepEqual(first.draws, map[string]uint64{"geography": 1}) {
			t.Fatalf("unexpected generation state for seed %d", tc.seed)
		}
		if b.FloorID != "town/ground" || len(b.Entities) != 4 || len(b.Cells) != 4 || len(b.Cells[0]) != 5 {
			t.Fatal("unexpected floor, entity count or dimensions")
		}
		for y, row := range b.Cells {
			for x, cell := range row {
				space := "town/street"
				if x >= 2 {
					space = "town/bakery"
				}
				if cell.SpaceID != space || cell.Wall != (x == 2 && y != tc.row) {
					t.Fatalf("seed %d: wrong cell (%d,%d): %+v", tc.seed, x, y, cell)
				}
			}
		}
		entities := []struct {
			id   string
			kind EntityKind
			x, y int
		}{
			{"town/player", Actor, 1, tc.row},
			{"town/door", Door, 2, tc.row},
			{"town/crate", Container, 3, tc.row - 1},
			{"town/spoon", Item, 3, tc.row - 1},
		}
		for _, want := range entities {
			entity := b.Entities[want.id]
			pos, err := b.PositionOf(want.id)
			if err != nil || pos != (CellPosition{"town/ground", want.x, want.y}) || entity.Kind != want.kind || entity.Open || entity.Portable != (want.kind == Item) {
				t.Fatalf("seed %d, %s: %+v at %+v, %v", tc.seed, want.id, entity, pos, err)
			}
		}
		if p := b.Entities["town/spoon"].Placement; p != (Placement{HolderID: "town/crate"}) {
			t.Fatalf("spoon must start solely inside the crate: %+v", p)
		}
	}
}

func TestBakeryStreamIsolation(t *testing.T) {
	clean, busy := NewTimeline(42), NewTimeline(42)
	draw(t, busy, "weather")
	a, b := makeBakery(t, "one/two", clean), makeBakery(t, "one/two", busy)
	if !reflect.DeepEqual(a, b) || busy.draws["weather"] != 1 || busy.draws["geography"] != 1 {
		t.Fatal("geography and weather interfered")
	}
	other := makeBakery(t, "one%2Ftwo", NewTimeline(42))
	for id := range a.Entities {
		if _, found := other.Entities[id]; found {
			t.Fatalf("ID shared by distinct worlds: %s", id)
		}
	}
	before := snapshot(t, busy)
	if _, err := GenerateBakery("one/two", busy); err == nil || !bytes.Equal(before, snapshot(t, busy)) {
		t.Fatal("regeneration should fail without consuming a draw")
	}
	late := NewTimeline(42)
	if err := late.Advance(1); err != nil {
		t.Fatal(err)
	}
	before = snapshot(t, late)
	if _, err := GenerateBakery("town", late); err == nil || !bytes.Equal(before, snapshot(t, late)) {
		t.Fatal("late generation changed state or succeeded")
	}
}

func TestBakeryEffectivePosition(t *testing.T) {
	b := makeBakery(t, "town", NewTimeline(42))
	spoon := b.Entities["town/spoon"]
	spoon.Placement = Placement{HolderID: "town/player"}
	b.Entities["town/spoon"] = spoon
	player := b.Entities["town/player"]
	player.Placement = Placement{Cell: CellPosition{b.FloorID, 0, 3}}
	b.Entities["town/player"] = player
	pos, err := b.PositionOf("town/spoon")
	if err != nil || pos != player.Placement.Cell {
		t.Fatalf("carried spoon at %+v, %v", pos, err)
	}
}

func TestBakeryInvalidPlacements(t *testing.T) {
	for _, tc := range []struct {
		name, entity string
		parent       Placement
	}{
		{"two parents", "town/spoon", Placement{Cell: CellPosition{"town/ground", 0, 0}, HolderID: "town/crate"}},
		{"missing parent", "town/spoon", Placement{}},
		{"missing holder", "town/spoon", Placement{HolderID: "town/missing"}},
		{"wrong holder kind", "town/spoon", Placement{HolderID: "town/door"}},
		{"cycle", "town/crate", Placement{HolderID: "town/crate"}},
		{"out of bounds", "town/player", Placement{Cell: CellPosition{"town/ground", -1, 2}}},
		{"foreign floor", "town/player", Placement{Cell: CellPosition{"other/ground", 0, 2}}},
		{"wall", "town/player", Placement{Cell: CellPosition{"town/ground", 2, 0}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := makeBakery(t, "town", NewTimeline(42))
			entity := b.Entities[tc.entity]
			entity.Placement = tc.parent
			b.Entities[tc.entity] = entity
			if err := b.Validate(); err == nil {
				t.Fatal("accepted invalid placement")
			}
		})
	}
}

func TestBakeryInvalidReferences(t *testing.T) {
	for _, id := range []string{"elsewhere/spoon", "town/", "town/ground", "town/street", "town/bakery"} {
		b := makeBakery(t, "town", NewTimeline(42))
		b.Entities[id] = b.Entities["town/spoon"]
		if err := b.Validate(); err == nil {
			t.Fatalf("accepted invalid entity ID %q", id)
		}
	}
	b := makeBakery(t, "town", NewTimeline(42))
	b.Cells[0][0].SpaceID = "town/unknown"
	if err := b.Validate(); err == nil {
		t.Fatal("accepted a cell in an unknown space")
	}
}
