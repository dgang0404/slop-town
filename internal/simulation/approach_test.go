// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"strings"
	"testing"
)

func TestWalkArrivalUsesTheSelectedVisibleDestination(t *testing.T) {
	const prefix = "You walk toward the spoon.\nAnother sound."
	const message = prefix + "\nYou are within reach of the spoon."
	second := ObservedTarget{Ref: "second", Name: "spoon"}
	r := Result{Message: message, SelectedTarget: second.Ref,
		Observation: Observation{Targets: []ObservedTarget{{Ref: "first", Name: "spoon"}, second}}}
	if rest, reached := r.SplitWalkArrival(); rest != prefix || reached != second {
		t.Fatal(rest, reached)
	}
	// A namesake in the final view cannot stand in for the selected spoon.
	r.Observation.Targets = r.Observation.Targets[:1]
	if rest, reached := r.SplitWalkArrival(); rest != message || reached.Ref != "" {
		t.Fatal(rest, reached)
	}
}

func TestWalkInterruptedAtShopfront(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player, target = "bakery/player", "bakery/crate"
	// Put the destination across the street, beyond the workers' meeting.
	crate := c.bakery.Entities[target]
	crate.Placement.Cell.X, crate.Placement.Cell.Y = 0, 0
	c.bakery.Entities[target] = crate
	for i := range 2 {
		r := c.Submit(player, spatialProposal(c, fmt.Sprint(i), ActionMove, ActionArgs{Direction: South}))
		if r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	walk := func(id string) Result {
		return c.Submit(player, spatialProposal(c, id, ActionApproach, ActionArgs{Target: target}))
	}
	if r := walk("first"); !r.ContinueWalk {
		t.Fatal(r)
	}
	r := walk("meeting")
	at := c.bakery.Entities[player].Placement.Cell
	if r.Status != "accepted" || r.ContinueWalk || !strings.Contains(r.Message, "introduces themselves to you") || !strings.Contains(r.Message, "pause your walk") || inReach(at, crate.Placement.Cell) || c.clock.Tick() != 4 {
		t.Fatalf("missed encounter: %+v, at %+v", r, at)
	}
	if r = walk("continue"); r.Status != "accepted" || !inReach(c.bakery.Entities[player].Placement.Cell, crate.Placement.Cell) || c.clock.Tick() != 5 {
		t.Fatal(r)
	}
}

func TestWalkStopsOnLosingSight(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player = "bakery/player"
	for i, dir := range []Direction{"", "", "", West, North} {
		p := spatialProposal(c, fmt.Sprint(i), ActionMove, ActionArgs{Direction: dir})
		if dir == "" {
			p.Action = ActionWait
		}
		if r := c.Submit(player, p); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	r := c.Submit(player, spatialProposal(c, "walk", ActionApproach, ActionArgs{Target: "bakery/baker"}))
	if r.Status != "accepted" || r.ContinueWalk || r.Observation.hasTarget("bakery/baker") || !strings.Contains(r.Message, "lose sight") || c.clock.Tick() != 6 {
		t.Fatal(r)
	}
}
