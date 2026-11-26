// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func currentView(t *testing.T, c *Coordinator, actor string) Observation {
	t.Helper()
	clock, history, receipts := snapshot(t, c.clock), c.History(), maps.Clone(c.receipts)
	view, err := c.Observe(actor)
	if err != nil || view.ObserverID != actor || view.Scope != c.scope {
		t.Fatalf("invalid observation %+v, %v", view, err)
	}
	if !bytes.Equal(clock, snapshot(t, c.clock)) || !reflect.DeepEqual(history, c.History()) || !maps.Equal(receipts, c.receipts) {
		t.Fatal("observing changed time, randomness, events or receipts")
	}
	return view
}

func TestMatchingTargets(t *testing.T) {
	wood := ObservedTarget{Ref: "wood", Name: "polished Wooden Spoon"}
	metal := ObservedTarget{Ref: "metal", Name: "metal spoon"}
	plain := ObservedTarget{Ref: "plain", Name: "spoon"}
	view := Observation{Targets: []ObservedTarget{wood, metal}}
	for query, want := range map[string][]ObservedTarget{
		"wooden spoon": {wood}, "spoon": {wood, metal}, "woo": nil, "oo": nil, "": nil,
	} {
		if got := view.MatchingTargets(query); !slices.Equal(got, want) {
			t.Errorf("%q matched %v, want %v", query, got, want)
		}
	}
	view.Targets = append(view.Targets, plain)
	if got := view.MatchingTargets("spoon"); !slices.Equal(got, []ObservedTarget{plain}) {
		t.Fatalf("exact name did not take priority: %v", got)
	}
}

func TestNamedActionKeepsObservedReference(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const other = "town/other-spoon"
	spoon := c.bakery.Entities[spoonID]
	spoon.Name = "metal spoon"
	spoon.Placement = c.bakery.Entities[playerID].Placement
	c.bakery.Entities[other] = spoon
	p := spatialProposal(c, "pickup", ActionTake, ActionArgs{Target: "spoon"})
	p.ByName = true
	// The originally observed spoon is now inside a closed crate. Only the
	// other spoon matches the name at submission.
	p.Observed = NameBindings{Target: spoonID}
	if r := submitUnchanged(t, c, playerID, p); r.Code != "target_unavailable" {
		t.Fatalf("named action switched targets: %+v", r)
	}
	p.Observed.Target = other
	r := c.Submit(playerID, p)
	if r.Status != "accepted" || r.SelectedTarget != other {
		t.Fatalf("unexpected selected target: %+v", r)
	}
	p.Observed = NameBindings{}
	if retry := submitUnchanged(t, c, playerID, p); !reflect.DeepEqual(retry, r) {
		t.Fatalf("retry lost the original selection: %+v", retry)
	}
	enterBakery(t, c, playerID)
	commitBakeryAction(t, c, playerID, ActionOpen, ActionArgs{Target: crateID})
	put := spatialProposal(c, "put", ActionPut, ActionArgs{Target: "metal spoon", Container: "crate"})
	put.ByName = true
	put.Observed = NameBindings{Target: other, Container: "town/old-crate"}
	if r := submitUnchanged(t, c, playerID, put); r.Code != "target_unavailable" {
		t.Fatalf("named action switched containers: %+v", r)
	}
}

func TestBakeryObservationWalkthrough(t *testing.T) {
	for _, tc := range []struct {
		seed int64
		row  int
	}{{42, 1}, {43, 2}} {
		t.Run(fmt.Sprint(tc.seed), func(t *testing.T) {
			c := bakeryCoordinator(t, tc.seed)
			door, crate, spoon := ActionArgs{Target: doorID}, ActionArgs{Target: crateID}, ActionArgs{Target: spoonID}
			view := currentView(t, c, playerID)
			tile := func(x, y int) string {
				return strings.Fields(strings.Split(view.Grid, "\n")[y])[x]
			}
			if view.hasTarget(crateID) || view.hasTarget(spoonID) || !view.hasTarget(doorID) ||
				strings.Contains(view.Text, "crate") || strings.Contains(view.Text, "spoon") ||
				!strings.Contains(view.Text, "closed bakery door") || tile(2, tc.row) != "+" || tile(3, tc.row-1) != "?" {
				t.Fatalf("unexpected street scene %+v", view)
			}
			step := func(action ActionKind, args ActionArgs) {
				t.Helper()
				_, r := commitBakeryAction(t, c, playerID, action, args)
				view = r.Observation
				if fresh := currentView(t, c, playerID); !reflect.DeepEqual(view, fresh) {
					t.Fatalf("action and observation disagree: %+v, %+v", view, fresh)
				}
			}
			step(ActionOpen, door)
			if tile(3, tc.row) != "." {
				t.Fatalf("open doorway did not reveal the floor: %s", view.Grid)
			}
			step(ActionMove, ActionArgs{Direction: East})
			if !strings.Contains(view.Text, "in the bakery doorway") {
				t.Fatalf("unexpected doorway view: %s", view.Text)
			}
			step(ActionMove, ActionArgs{Direction: East})
			if !view.hasTarget(crateID) || view.hasTarget(spoonID) || tile(3, tc.row-1) != "C" ||
				!strings.Contains(view.Text, "closed crate to the north") || strings.Contains(view.Text, "spoon") {
				t.Fatalf("unexpected view beside the crate %+v", view)
			}
			step(ActionOpen, crate)
			if !view.hasTarget(spoonID) || tile(3, tc.row-1) != "c*" || !strings.Contains(view.Text, "wooden spoon in the crate") {
				t.Fatalf("opening the lid did not reveal the spoon %+v", view)
			}
			step(ActionTake, spoon)
			step(ActionClose, crate)
			step(ActionMove, ActionArgs{Direction: West})
			step(ActionMove, ActionArgs{Direction: West})
			step(ActionClose, door)
			if !view.hasTarget(spoonID) || !strings.Contains(view.Text, "carry the wooden spoon") ||
				view.hasTarget(crateID) || strings.Contains(view.Text, "crate") || tile(3, tc.row-1) != "?" ||
				strings.Contains(view.Grid, "*") {
				t.Fatalf("unexpected return to the street %+v", view)
			}
			hidden := rejectBakeryAction(t, c, playerID, ActionOpen, crate, "target_unavailable")
			missing := rejectBakeryAction(t, c, playerID, ActionOpen, ActionArgs{Target: "town/absent"}, "target_unavailable")
			if !reflect.DeepEqual(hidden, missing) {
				t.Fatal("stale visible reference disclosed a hidden crate")
			}
			step(ActionDrop, spoon)
			if !view.hasTarget(spoonID) || tile(1, tc.row) != "@*" || !strings.Contains(view.Text, "wooden spoon here") {
				t.Fatalf("unexpected dropped spoon %+v", view)
			}
		})
	}
}

func TestBakeryObservationIsActorScoped(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const bakerID = "town/baker"
	c.bakery.Entities[bakerID] = SpatialEntity{Name: "baker", Kind: Actor, Placement: Placement{Cell: CellPosition{c.bakery.FloorID, 3, 1}}}
	_, opened := commitBakeryAction(t, c, bakerID, ActionOpen, ActionArgs{Target: crateID})
	outside := currentView(t, c, playerID)
	if !opened.Observation.hasTarget(spoonID) || outside.hasTarget(spoonID) || outside.hasTarget(bakerID) {
		t.Fatal("actors on opposite sides of the closed door shared a view")
	}
	// The returned slice must not give a caller a way to rewrite the saved receipt.
	p, taken := commitBakeryAction(t, c, bakerID, ActionTake, ActionArgs{Target: spoonID})
	taken.Observation.Targets[0].Ref = "corrupt"
	if retry := c.Submit(bakerID, p); retry.Observation.hasTarget("corrupt") {
		t.Fatal("caller changed the original receipt")
	}
	enterBakery(t, c, playerID)
	inside := currentView(t, c, playerID)
	if !inside.hasTarget(bakerID) || inside.hasTarget(spoonID) || strings.Contains(inside.Text, "spoon") {
		t.Fatalf("another actor's inventory leaked %+v", inside)
	}
	if _, err := c.Observe(crateID); err == nil {
		t.Fatal("a crate observed the scene")
	}
}

func TestBakeryActionsAreActorScoped(t *testing.T) {
	c := bakeryCoordinator(t, 42)
	const baker = "town/baker"
	b := c.bakery
	b.Entities[baker] = SpatialEntity{Name: "baker", Kind: Actor, Placement: b.Entities[crateID].Placement}
	commitBakeryAction(t, c, baker, ActionOpen, ActionArgs{Target: crateID})
	actions := func(actor string) []ActionOption {
		t.Helper()
		got, err := c.AvailableActions(actor)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	take := ActionOption{ActionTake, ActionArgs{Target: "wooden spoon"}}
	if !slices.Contains(actions(baker), take) || slices.Contains(actions(playerID), take) {
		t.Fatal("the baker alone can reach the spoon from inside the closed bakery")
	}
	enterBakery(t, c, playerID)
	if !slices.Contains(actions(playerID), take) {
		t.Fatal("the player beside the open crate should be able to take the spoon")
	}
	commitBakeryAction(t, c, baker, ActionTake, ActionArgs{Target: spoonID})
	drop := ActionOption{ActionDrop, take.Args}
	if !slices.Contains(actions(baker), drop) {
		t.Fatal("the baker cannot drop the carried spoon")
	}
	for _, option := range actions(playerID) {
		if option.Args.Target == take.Args.Target {
			t.Fatalf("another actor's inventory appeared in the player's choices: %+v", option)
		}
	}
	// The previously offered command is validated against the new scene.
	p := spatialProposal(c, "take-after-baker", take.Action, take.Args)
	p.ByName = true
	if r := submitUnchanged(t, c, playerID, p); r.Code != "target_unavailable" {
		t.Fatalf("stale command returned %+v", r)
	}
}

func TestBakeryObservationSightLines(t *testing.T) {
	var opaque cellMask
	opaque[0][1], opaque[1][0] = true, true
	if opaque.canSee(0, 0, 1, 1) || opaque.canSee(1, 1, 0, 0) || opaque.canSee(0, 0, 2, 2) {
		t.Fatal("sight crossed touching opaque corners")
	}
	opaque[1][0] = false
	if !opaque.canSee(0, 0, 1, 1) || !opaque.canSee(1, 1, 0, 0) {
		t.Fatal("one opaque corner hid the open diagonal")
	}
	if !opaque.canSee(0, 0, 1, 0) || opaque.canSee(0, 0, 3, 0) || opaque.canSee(0, 0, 3, 1) {
		t.Fatal("opaque cell should be visible and block sight beyond it")
	}
}
