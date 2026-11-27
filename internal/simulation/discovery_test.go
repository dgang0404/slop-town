// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"testing"
)

func TestJournalRemembersWhereEachSpoonWasFirstSeen(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const streetID = "town/street-spoon"
	spoon := c.bakery.Entities[spoonID]
	spoon.Placement = c.bakery.Entities[playerID].Placement
	c.bakery.Entities[streetID] = spoon
	commitBakeryAction(t, c, playerID, ActionWait, ActionArgs{})
	enterBakery(t, c, playerID)
	commitBakeryAction(t, c, playerID, ActionOpen, ActionArgs{Target: crateID})
	commitBakeryAction(t, c, playerID, ActionTake, ActionArgs{Target: spoonID})
	commitBakeryAction(t, c, playerID, ActionDrop, ActionArgs{Target: spoonID})
	book, err := c.Journal(playerID)
	if err != nil {
		t.Fatal(err)
	}
	var spoons, latest []Discovery
	for _, entry := range book {
		for _, d := range entry.FirstSeen {
			if d.Target.Name == spoon.Name {
				spoons = append(spoons, d)
			}
		}
		for _, d := range entry.LastSeen {
			if d.Target.Name == spoon.Name {
				latest = append(latest, d)
			}
		}
	}
	want := []Discovery{
		{Target: ObservedTarget{Ref: streetID, Name: spoon.Name}, Label: spoon.Name, Location: LocationHere, Text: "You see the wooden spoon here."},
		{Target: ObservedTarget{Ref: spoonID, Name: spoon.Name}, Label: spoon.Name, Location: LocationContainer, Container: "crate", Text: "You see the wooden spoon in the crate."},
	}
	if !slices.Equal(spoons, want) {
		t.Fatalf("first sightings %+v, want %+v", spoons, want)
	}
	wantLatest := []Discovery{
		{Target: want[1].Target, Label: spoon.Name, Location: LocationHere, Text: "You see the wooden spoon here."},
		{Target: want[0].Target, Label: spoon.Name, Location: LocationBearing, Bearing: "west", Text: "You see the wooden spoon to the west."},
	}
	if !slices.Equal(latest, wantLatest) {
		t.Fatalf("latest sightings %+v, want %+v", latest, wantLatest)
	}
}

func TestLastSeenSpoonFollowsTheObserver(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const baker = "town/baker"
	c.bakery.Entities[baker] = SpatialEntity{Name: "baker", Kind: Actor, Placement: c.bakery.Entities[crateID].Placement}
	enterBakery(t, c, playerID)
	crate, spoon := ActionArgs{Target: crateID}, ActionArgs{Target: spoonID}
	commitBakeryAction(t, c, playerID, ActionOpen, crate)
	commitBakeryAction(t, c, playerID, ActionClose, crate)
	commitBakeryAction(t, c, playerID, ActionMove, ActionArgs{Direction: West})
	commitBakeryAction(t, c, playerID, ActionMove, ActionArgs{Direction: West})
	commitBakeryAction(t, c, playerID, ActionClose, ActionArgs{Target: doorID})
	commitBakeryAction(t, c, baker, ActionOpen, crate)
	commitBakeryAction(t, c, baker, ActionTake, spoon)

	check := func(actor string, want DiscoveryLocation) {
		t.Helper()
		entries, err := c.Journal(actor)
		if err != nil {
			t.Fatal(err)
		}
		var found []Discovery
		for _, entry := range entries {
			for _, d := range entry.LastSeen {
				if d.Target.Ref == spoonID {
					found = append(found, d)
				}
			}
		}
		if len(found) != 1 || found[0].Location != want || (want == LocationContainer && found[0].Container != "crate") {
			t.Fatalf("%s recalls %+v, want %s", actor, found, want)
		}
	}
	check(playerID, LocationContainer)
	check(baker, LocationInventory)
	enterBakery(t, c, playerID)
	check(playerID, LocationContainer)
	commitBakeryAction(t, c, baker, ActionDrop, spoon)
	commitBakeryAction(t, c, playerID, ActionWait, ActionArgs{})
	check(playerID, LocationBearing)
}

func TestDiscoveryReadsOnlyTheRecordedLocation(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		location   DiscoveryLocation
		label      string
		bearing    string
		container  string
	}{
		{"crate", "You see the open crate to the southeast.", LocationBearing, "open crate", "southeast", ""},
		{"baker", "You see the baker sitting here.", LocationHere, "baker sitting", "", ""},
		{"key in the lock", "You see the key in the lock in the box in the corner.", LocationContainer, "key in the lock", "", "box in the corner"},
		{"key", "You carry the key.", LocationInventory, "key", "", ""},
		{"key", "You see the key beside a shelf.", LocationUnspecified, "key", "", ""},
	} {
		target := ObservedTarget{Ref: "town/object", Name: tc.name}
		d := discoveryFromReceipt(target, tc.text)
		if d.Target != target || d.Text != tc.text || d.Location != tc.location || d.Label != tc.label || d.Bearing != tc.bearing || d.Container != tc.container {
			t.Errorf("%q: %+v", tc.text, d)
		}
	}
}

func TestUnpairedDescriptionsLeaveLocationsUnspecified(t *testing.T) {
	memory := journalMemory{latest: make(map[string]recordedSighting)}
	view := Observation{
		Targets: []ObservedTarget{{Ref: "town/one", Name: "spoon"}, {Ref: "town/two", Name: "spoon"}},
		Text:    "You are on the street.\nYou see the spoon in the box.",
	}
	_, _, discoveries := memory.observe(view, 0)
	if len(discoveries) != len(view.Targets) {
		t.Fatal(discoveries)
	}
	for i, d := range discoveries {
		if d.Target != view.Targets[i] || d.Location != LocationUnspecified || d.Container != "" || d.Text != "" {
			t.Fatal(d)
		}
	}
}
