// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"strings"
	"testing"
)

func TestCrateScreensAResidentFromOneSide(t *testing.T) {
	const player, baker, customer = "bakery/player", "bakery/baker", "bakery/customer"
	const crate, door, spoon = "bakery/crate", "bakery/door", "bakery/spoon"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	box, entrance := b.Entities[crate], b.Entities[door]
	entrance.Open = true
	b.Entities[door] = entrance
	across, hider, item := b.Entities[player], b.Entities[baker], b.Entities[spoon]
	across.Placement = entrance.Placement
	hider.Placement = box.Placement
	hider.Placement.Cell.X++
	item.Placement = Placement{HolderID: baker}
	b.Entities[player], b.Entities[baker], b.Entities[spoon] = across, hider, item
	side := hider.Placement
	side.Cell.Y++
	b.Entities[customer] = SpatialEntity{Kind: Actor, Name: "customer", Placement: side}
	if before, err := c.Observe(player); err != nil || !before.hasTarget(baker) || !before.hasTarget(crate) {
		t.Fatal(before, err)
	}
	hide := spatialProposal(c, "duck", ActionHide, ActionArgs{Target: crate})
	r := c.Submit(baker, hide)
	if r.Status != "accepted" || r.Observation.hasTarget(player) || !r.Observation.hasTarget(customer) ||
		!r.Observation.hasTarget(crate) || !r.Observation.hasTarget(spoon) {
		t.Fatal(r)
	}
	submitUnchanged(t, c, baker, hide)
	for _, id := range []string{player, customer} {
		scene, err := c.Scene(id)
		if err != nil || scene.hasTarget(baker) != (id == customer) || !scene.hasTarget(crate) {
			t.Fatal(id, scene, err)
		}
		if id == customer && scene.Details[baker].State != "crouching" {
			t.Fatal(scene.Details[baker])
		}
		entry, ok := c.bakery.firsthandAt(id, 2)
		if !ok || entry.Action != ActionHide || entry.Subject != baker || entry.Target != crate {
			t.Fatal("lost the sighting before the baker ducked", id, entry)
		}
	}
	menu, err := c.AvailableActions(player)
	if err != nil || slices.ContainsFunc(menu, func(o ActionOption) bool { return o.Args.Target == "baker" }) {
		t.Fatal(menu, err)
	}
	if lost := submitUnchanged(t, c, player, spatialProposal(c, "follow", ActionApproach, ActionArgs{Target: baker})); lost.Code != "target_unavailable" {
		t.Fatal(lost)
	}
	if again := submitUnchanged(t, c, baker, spatialProposal(c, "again", ActionHide, ActionArgs{Target: crate})); again.Code != "already_in_state" {
		t.Fatal(again)
	}
	// The crate screens the crouching person. Sound can still reach the
	// doorway and the standing customer can see the spoon change hands.
	if drop := c.Submit(baker, spatialProposal(c, "drop", ActionDrop, ActionArgs{Target: spoon})); drop.Status != "accepted" {
		t.Fatal(drop)
	}
	for _, id := range []string{player, customer} {
		notes, err := c.bakery.Notebook(id)
		saw := slices.ContainsFunc(notes, func(n NotebookEntry) bool { return n.Action == ActionDrop && n.Subject == baker && n.Target == spoon })
		heard := slices.ContainsFunc(notes, func(n NotebookEntry) bool { return n.Sound == Clatter })
		if err != nil || saw != (id == customer) || !heard {
			t.Fatal(id, notes, err)
		}
	}
	stand := c.Submit(baker, spatialProposal(c, "stand", ActionEmerge, ActionArgs{}))
	view, err := c.Scene(player)
	if stand.Status != "accepted" || err != nil || !view.hasTarget(baker) || view.Details[baker].State != "standing" {
		t.Fatal(stand, view, err)
	}
	if again := submitUnchanged(t, c, baker, spatialProposal(c, "stand again", ActionEmerge, ActionArgs{})); again.Status != "rejected" {
		t.Fatal(again)
	}
}

func TestCoverRequiresAnAdjacentSuitableObject(t *testing.T) {
	const player, crate = "bakery/player", "bakery/crate"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	actor, box := b.Entities[player], b.Entities[crate]
	actor.Placement = box.Placement
	b.Entities[player] = actor
	p := spatialProposal(c, "hide", ActionHide, ActionArgs{Target: crate})
	if r := submitUnchanged(t, c, player, p); r.Status != "rejected" {
		t.Fatal("no side to hide on in the crate's own square", r)
	}
	actor.Placement.Cell.X++
	b.Entities[player] = actor
	box.ProvidesCover = false
	b.Entities["bakery/small-box"] = box
	if r := submitUnchanged(t, c, player, spatialProposal(c, "small", ActionHide, ActionArgs{Target: "bakery/small-box"})); r.Status != "rejected" {
		t.Fatal("same name supplied the missing cover", r)
	}
	for _, next := range []ActionKind{ActionRest, ActionMove} {
		if r := c.Submit(player, spatialProposal(c, "hide-before-"+string(next), ActionHide, ActionArgs{Target: crate})); r.Status != "accepted" {
			t.Fatal(r)
		}
		args := ActionArgs{}
		if next == ActionMove {
			args.Direction = South
		}
		if r := c.Submit(player, spatialProposal(c, string(next), next, args)); r.Status != "accepted" || strings.Contains(r.Observation.Text, crouchingBehind) || c.bakery.Entities[player].HiddenBehind != "" {
			t.Fatal(r)
		}
	}
}

func TestRestoreChecksCover(t *testing.T) {
	const player, crate, spoon = "bakery/player", "bakery/crate", "bakery/spoon"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	actor := b.Entities[player]
	actor.Placement = b.Entities[crate].Placement
	actor.Placement.Cell.X++
	actor.HiddenBehind = crate
	b.Entities[player] = actor
	restore := func() error {
		t.Helper()
		raw, err := b.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		_, err = RestoreBakery(raw, "bakery", c.clock)
		return err
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*SpatialEntity){
		func(e *SpatialEntity) { e.Seated = true },
		func(e *SpatialEntity) { e.HiddenBehind = spoon },
		func(e *SpatialEntity) { e.Placement.Cell.Y += 2 },
	} {
		e := actor
		change(&e)
		b.Entities[player] = e
		if err := restore(); err == nil {
			t.Fatal("restored invalid hiding place", e)
		}
	}
	b.Entities[player] = actor
	usePracticeVersion(b, stealthPracticeContentVersion)
	if err := restore(); err == nil {
		t.Fatal("restored a crouching actor under older rules")
	}
}
