// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

// Both workers reach the same open doorway, equally tired, with both
// break sites available. No choice stream should be used on the approach.
func workersAtBreak(t *testing.T) *Coordinator {
	t.Helper()
	c, err := (BakerySetup{Resident: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	for tick := 1; tick <= 9; tick++ {
		if r := c.Submit("bakery/player", spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	door := c.bakery.Entities["bakery/door"]
	for _, role := range []string{"baker", "porter"} {
		worker := c.bakery.Entities["bakery/"+role]
		if !door.Open || worker.Placement != door.Placement || worker.Fatigue != fatigueLimit || worker.Seated || worker.SeekingRestSpot || c.clock.draws["rest-place/"+role] != 0 {
			t.Fatalf("unexpected %s before break: %+v", role, worker)
		}
	}
	return c
}

func TestPersonalityChangesTheChoice(t *testing.T) {
	for _, swap := range []bool{false, true} {
		c := workersAtBreak(t)
		baker, porter := c.bakery.Entities["bakery/baker"], c.bakery.Entities["bakery/porter"]
		if baker.RestInPlacePercent != 40 || porter.RestInPlacePercent != 87 {
			t.Fatalf("seed 42 profiles: %+v, %+v", baker, porter)
		}
		if swap {
			baker.RestInPlacePercent, porter.RestInPlacePercent = porter.RestInPlacePercent, baker.RestInPlacePercent
			c.bakery.Entities["bakery/baker"], c.bakery.Entities["bakery/porter"] = baker, porter
		}
		first := len(c.events)
		wait := spatialProposal(c, "break", ActionWait, ActionArgs{})
		r := c.Submit("bakery/player", wait)
		if r.Status != "accepted" || len(c.events) != first+6 {
			t.Fatal(r, c.events[first:])
		}
		for i, role := range []string{"baker", "porter"} {
			choice, action := c.events[first+2+2*i], c.events[first+3+2*i]
			id, stream := "bakery/"+role, "rest-place/"+role
			worker := c.bakery.Entities[id]
			sit := (role == "porter") != swap
			at := worker.Placement.Cell
			if choice.ActorID != id || choice.Kind != "rest_place_chosen" || choice.Engine != "personality" || choice.Tick != 10 ||
				choice.ParentID != c.events[first+1].ID || choice.Payload.Stream != stream || choice.Payload.Value != []uint64{58, 48}[i] ||
				action.ParentID != choice.ID || action.ActorID != id || c.clock.draws[stream] != 1 {
				t.Fatalf("wrong decision or effect: %+v, %+v", choice, action)
			}
			if sit {
				if !worker.Seated || worker.SeekingRestSpot || worker.Fatigue != 0 || at.X != 2 || action.Kind != "actor_rested" || choice.Payload.Destination != at {
					t.Fatalf("%s should sit in place: %+v, %+v", id, worker, choice)
				}
			} else if worker.Seated || !worker.SeekingRestSpot || at.X != 3 || action.Kind != "actor_moved" || choice.Payload.Destination.X != 4 {
				t.Fatalf("%s should walk to the back: %+v, %+v", id, worker, choice)
			}
		}
		if got := submitUnchanged(t, c, "bakery/player", wait); !reflect.DeepEqual(got, r) {
			t.Fatal("retry changed the outcome")
		}
		for tick := 11; tick <= 12; tick++ {
			if got := c.Submit("bakery/player", spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})); got.Status != "accepted" {
				t.Fatal(got)
			}
		}
		traveller := "baker"
		if swap {
			traveller = "porter"
		}
		worker := c.bakery.Entities["bakery/"+traveller]
		if !worker.Seated || worker.SeekingRestSpot || worker.Placement.Cell.X != 4 || c.clock.draws["rest-place/"+traveller] != 1 {
			t.Fatalf("did not finish the chosen break: %+v", worker)
		}
	}
}

func TestPersonalityDrawFailureIsAtomic(t *testing.T) {
	c := workersAtBreak(t)
	c.clock.draws["rest-place/porter"] = math.MaxUint64
	r := submitUnchanged(t, c, "bakery/player", spatialProposal(c, "break", ActionWait, ActionArgs{}))
	if r.Code != "resolution_failed" || c.clock.draws["rest-place/baker"] != 0 {
		t.Fatal(r)
	}
}

func TestRestPreferenceNeedsTwoAvailablePlaces(t *testing.T) {
	for _, atBack := range []bool{false, true} {
		c := workersAtBreak(t)
		b := c.bakery
		actor := b.Entities["bakery/baker"]
		at := actor.Placement.Cell
		if atBack {
			actor.Placement.Cell.X = 4
			b.Entities["bakery/baker"] = actor
		} else {
			b.Cells[at.Y][at.X+1].Wall = true
		}
		effect, choice, err := b.residentTurn(c.scope, "baker", c.clock)
		if err != nil || effect.kind != "actor_rested" || choice != nil || c.clock.draws["rest-place/baker"] != 0 {
			t.Fatalf("expected rest without a draw, got %+v, %+v, %v", effect, choice, err)
		}
	}
}

func TestRestoreInvalidRestPreference(t *testing.T) {
	c := workersAtBreak(t)
	for _, n := range []int{0, 100} {
		worker := c.bakery.Entities["bakery/porter"]
		worker.RestInPlacePercent = n
		c.bakery.Entities["bakery/porter"] = worker
		data, err := c.bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(data, c.scope.WorldID, c.clock); err == nil {
			t.Fatalf("accepted invalid preference %d", n)
		}
	}
}
