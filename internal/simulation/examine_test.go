// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"strings"
	"testing"
)

func TestExamineSamplesBeforeThePorterMoves(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player, porter = "bakery/player", "bakery/porter"
	examine := spatialProposal(c, "from-street", ActionExamine, ActionArgs{Target: porter})
	first := c.Submit(player, examine)
	if first.Status != "accepted" || first.Message != "You find the porter standing." ||
		!inReach(c.bakery.Entities[player].Placement.Cell, c.bakery.Entities[porter].Placement.Cell) {
		t.Fatal(first)
	}
	second := c.Submit(player, spatialProposal(c, "up-close", ActionExamine, examine.Args))
	if second.Status != "accepted" || !strings.HasPrefix(second.Message, "You find the porter standing, one sleeve frayed at the seam.") {
		t.Fatal(second)
	}
	if retry := submitUnchanged(t, c, player, examine); retry.Message != first.Message {
		t.Fatal("retry used the porter's later position", retry)
	}
}

func TestBakerExaminesOnlyVisibleContents(t *testing.T) {
	const baker, crate, door = "bakery/baker", "bakery/crate", "bakery/door"
	for _, tc := range []struct {
		name           string
		daylight, open bool
		finding        string
	}{
		{"shut lid", true, false, "closed, a pale scrape running across its lid"},
		{"open in daylight", true, true, "open with the flour tin and the wooden spoon inside, a pale scrape running across its lid"},
		{"open in dim light", false, true, "open with the flour tin and the wooden spoon inside"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
			if err != nil {
				t.Fatal(err)
			}
			b := c.bakery
			box, entrance := b.Entities[crate], b.Entities[door]
			box.Open, entrance.Open = tc.open, tc.daylight
			b.Entities[crate], b.Entities[door] = box, entrance
			b.Entities["bakery/flour-tin"] = SpatialEntity{Kind: Container, Name: "flour tin", Placement: Placement{HolderID: crate}}
			b.Entities["bakery/key"] = SpatialEntity{Kind: Item, Name: "brass key", Placement: Placement{HolderID: "bakery/flour-tin"}}
			b.Entities["bakery/pencil"] = SpatialEntity{Kind: Item, Name: "red pencil", Placement: Placement{HolderID: "bakery/player"}}
			for _, ref := range []string{"bakery/key", "bakery/pencil"} {
				if r := submitUnchanged(t, c, baker, spatialProposal(c, ref, ActionExamine, ActionArgs{Target: ref})); r.Code != "target_unavailable" {
					t.Fatal(r)
				}
			}
			menu, err := c.AvailableActions(baker)
			if err != nil {
				t.Fatal(err)
			}
			for _, o := range menu {
				if o.Action == ActionExamine && (o.Args.Target == "brass key" || o.Args.Target == "red pencil" || (!tc.open && o.Args.Target == "wooden spoon")) {
					t.Fatal("menu exposed a hidden target", o)
				}
			}
			result := c.Submit(baker, spatialProposal(c, "crate", ActionExamine, ActionArgs{Target: crate}))
			if result.Status != "accepted" || result.Message != "You find the crate "+tc.finding+"." {
				t.Fatal(result)
			}
		})
	}
}

func TestEarlierPracticeSaveKeepsItsActionMenu(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 43)
	if err != nil {
		t.Fatal(err)
	}
	usePracticeVersion(c.bakery, knockContentVersion)
	const player = "bakery/player"
	options, err := c.AvailableActions(player)
	if err != nil || slices.ContainsFunc(options, func(o ActionOption) bool { return o.Action == ActionExamine }) {
		t.Fatal(options, err)
	}
	p := spatialProposal(c, "try-examine", ActionExamine, ActionArgs{Target: "bakery/door"})
	if r := submitUnchanged(t, c, player, p); r.Code != "action_unavailable" {
		t.Fatal(r)
	}
}

func TestExamineKeepsTheChosenSpoon(t *testing.T) {
	const player, baker = "bakery/player", "bakery/baker"
	const first, second = "bakery/spoon", "bakery/other-spoon"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 43)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	item, worker := b.Entities[first], b.Entities[baker]
	worker.Placement = b.Entities[player].Placement
	item.Placement = worker.Placement
	b.Entities[baker], b.Entities[first] = worker, item
	item.CloseDetail = "its handle stained blue"
	b.Entities[second] = item
	choice := spatialProposal(c, "which-spoon", ActionExamine, ActionArgs{Target: "spoon"})
	choice.ByName = true
	if r := submitUnchanged(t, c, player, choice); r.Code != "ambiguous_target" {
		t.Fatal(r)
	}
	choice.Observed.Target = second
	take := c.Submit(baker, spatialProposal(c, "take-blue", ActionTake, ActionArgs{Target: second}))
	if take.Status != "accepted" {
		t.Fatal(take)
	}
	if r := submitUnchanged(t, c, player, choice); r.Code != "target_unavailable" || strings.Contains(r.Message, "blue") {
		t.Fatal("the other spoon or a private detail slipped into the answer", r)
	}
	person := c.Submit(player, spatialProposal(c, "see-baker", ActionExamine, ActionArgs{Target: baker}))
	if person.Status != "accepted" || strings.Contains(person.Message, "spoon") || strings.Contains(person.Message, "blue") {
		t.Fatal("examining the baker disclosed a private possession", person)
	}
	choice.Observed.Target = first
	if r := c.Submit(player, choice); r.Status != "accepted" || !strings.Contains(r.Message, "a shallow notch") || strings.Contains(r.Message, "blue") {
		t.Fatal("the remaining spoon lost its own mark", r)
	}
}
