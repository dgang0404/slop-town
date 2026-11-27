// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"strings"
	"testing"
)

func TestDoorwaySightingsComeFromTheObserversView(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player, porter, door, crate, spoon = "bakery/player", "bakery/porter", "bakery/door", "bakery/crate", "bakery/spoon"
	entities := c.bakery.Entities
	position := func(ref string, x, y int) {
		e := entities[ref]
		e.Placement.Cell.X, e.Placement.Cell.Y = x, y
		entities[ref] = e
	}
	open := func(ref string, value bool) {
		e := entities[ref]
		e.Open = value
		entities[ref] = e
	}
	position(player, 3, 1)
	position(porter, 1, 1)
	closed, err := c.Scene(player)
	if err != nil || closed.hasTarget(porter) || closed.Landmarks[porter] != "" {
		t.Fatal("closed door exposed the porter", closed, err)
	}
	open(door, true)
	// An interior door is not the threshold between the shop and the street.
	entities["bakery/office-door"] = SpatialEntity{Kind: Door, Name: "office door", Open: true, Placement: entities[player].Placement}
	inside, err := c.Scene(player)
	if err != nil || inside.Self.Place != InBakery || inside.Landmarks[porter] != OnStreet || !strings.Contains(inside.Text, "You see the porter outside on the street.") ||
		inside.Landmarks[crate] != "" || inside.ChoiceLocation(porter) != "outside on the street, to the west" {
		t.Fatal(inside, err)
	}
	// Move the crate into the open doorway’s sightline. Its contents still require an open lid.
	position(crate, 3, 1)
	open(crate, true)
	outside, err := c.Scene(porter)
	if err != nil || outside.Landmarks[player] != InBakery || outside.Landmarks[crate] != InBakery ||
		outside.Details[spoon].Location != "in the crate inside the bakery" {
		t.Fatal(outside, err)
	}
	sight := locatedDiscovery(ObservedTarget{Ref: spoon, Name: "wooden spoon"}, "You see the wooden spoon in the crate inside the bakery.", outside.Landmarks[spoon])
	if sight.Location != LocationContainer || sight.Container != "crate" || sight.Landmark != "inside the bakery" {
		t.Fatal(sight)
	}
	open(crate, false)
	position(player, 2, 1)
	threshold, err := c.Scene(porter)
	if err != nil || threshold.Landmarks[player] != InBakeryDoorway || threshold.hasTarget(spoon) || threshold.Landmarks[spoon] != "" {
		t.Fatal(threshold, err)
	}
	usePracticeVersion(c.bakery, harvestContentVersion)
	old, err := c.Scene(porter)
	if err != nil || len(old.Landmarks) != 0 || !strings.Contains(old.Text, "You see the player to the east.") {
		t.Fatal("earlier scene lost its bearings", old, err)
	}
}
