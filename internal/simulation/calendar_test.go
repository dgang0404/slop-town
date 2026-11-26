// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"math"
	"testing"
)

func TestBakeryDateBoundaries(t *testing.T) {
	for _, tc := range []struct {
		tick int64
		want BakeryDate
	}{
		{0, BakeryDate{1, "spring", 1}},
		{23, BakeryDate{1, "spring", 1}},
		{24, BakeryDate{2, "spring", 1}},
		{167, BakeryDate{7, "spring", 1}},
		{168, BakeryDate{1, "summer", 1}},
		{336, BakeryDate{1, "autumn", 1}},
		{671, BakeryDate{7, "winter", 1}},
		{672, BakeryDate{1, "spring", 2}},
		{math.MaxInt64, BakeryDate{1, "winter", 13725256007224369}},
	} {
		if got := bakeryDateAt(tc.tick); got != tc.want {
			t.Errorf("tick %d: got %+v, want %+v", tc.tick, got, tc.want)
		}
	}
}

func TestACharacterCanMissWholeDays(t *testing.T) {
	const player, visitor = "bakery/player", "bakery/visitor"
	c, err := (BakerySetup{}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	// A second actor can spend turns in the quiet bakery. Waiting gives
	// the other character nothing to record.
	person := c.bakery.Entities[player]
	person.Name = "visitor"
	c.bakery.Entities[visitor] = person
	wait := func(who string, tick int) {
		t.Helper()
		if r := c.Submit(who, spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	wait(player, 1)
	for tick := 2; tick <= 168; tick++ {
		wait(visitor, tick)
	}
	scene, err := c.Scene(player)
	if err != nil || scene.Date != (BakeryDate{1, "summer", 1}) {
		t.Fatal(scene.Date, err)
	}
	book, err := c.Journal(player)
	if err != nil || len(book) != 1 || book[0].Date != (BakeryDate{1, "spring", 1}) {
		t.Fatalf("journal after other actor's turns: %+v, %v", book, err)
	}
	wait(player, 169)
	book, err = c.Journal(player)
	if err != nil || len(book) != 2 || book[1].Date != scene.Date {
		t.Fatalf("journal on return: %+v, %v", book, err)
	}
}
