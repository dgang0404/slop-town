// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"strings"
	"testing"
)

func TestKneadAtThePrepBench(t *testing.T) {
	const player, baker = "bakery/player", "bakery/baker"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.ContentVersion < bakingContentVersion {
		t.Fatalf("content version %d", c.bakery.ContentVersion)
	}
	if c.bakery.Entities[baker].BakingBouts != 1 {
		t.Fatal("baker should start with baking experience")
	}
	knead := spatialProposal(c, "knead", ActionKnead, ActionArgs{})
	if r := submitUnchanged(t, c, player, knead); r.Code != "action_unavailable" {
		t.Fatal("player kneaded from the street", r)
	}
	walk := func(id, label string, dir Direction) {
		t.Helper()
		if r := c.Submit(id, spatialProposal(c, label, ActionMove, ActionArgs{Direction: dir})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	if r := c.Submit(player, spatialProposal(c, "open", ActionOpen, ActionArgs{Target: "bakery/door"})); r.Status != "accepted" {
		t.Fatal(r)
	}
	walk(player, "east", East)
	walk(player, "east-2", East)
	if c.bakery.Entities[player].Placement.Cell != c.bakery.PrepCell {
		t.Fatal("player not at prep")
	}
	if r := c.Submit(player, knead); r.Status != "accepted" || !strings.Contains(r.Message, "dense") {
		t.Fatal("first knead", r)
	}
	if r := c.Submit(player, spatialProposal(c, "knead-2", ActionKnead, ActionArgs{})); r.Status != "accepted" || !strings.Contains(r.Message, "dense") {
		t.Fatal("second knead", r)
	}
	if c.bakery.Entities[player].BakingBouts != 1 {
		t.Fatal("missing baking improvement")
	}
	var practice []Event
	for _, e := range c.History() {
		if e.Kind == "dough_kneaded" && e.Payload.BakingPractice {
			practice = append(practice, e)
		}
	}
	if len(practice) != 1 {
		t.Fatal(practice)
	}
	if r := c.Submit(player, spatialProposal(c, "knead-3", ActionKnead, ActionArgs{})); r.Status != "accepted" || !strings.Contains(r.Message, "springy") {
		t.Fatal("trained knead", r)
	}
	worker := c.bakery.Entities[baker]
	worker.Placement = Placement{Cell: c.bakery.PrepCell}
	c.bakery.Entities[baker] = worker
	if r := c.Submit(baker, spatialProposal(c, "baker-knead", ActionKnead, ActionArgs{})); r.Status != "accepted" || !strings.Contains(r.Message, "springy") {
		t.Fatal("baker knead", r)
	}
}

func TestOlderPracticeDisallowsKneading(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	usePracticeVersion(c.bakery, regrowthContentVersion)
	player := "bakery/player"
	if r := c.Submit(player, spatialProposal(c, "open", ActionOpen, ActionArgs{Target: "bakery/door"})); r.Status != "accepted" {
		t.Fatal(r)
	}
	for _, label := range []string{"east", "east-2"} {
		if r := c.Submit(player, spatialProposal(c, label, ActionMove, ActionArgs{Direction: East})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	if r := submitUnchanged(t, c, "bakery/player", spatialProposal(c, "knead", ActionKnead, ActionArgs{})); r.Status != "rejected" {
		t.Fatal("old town allowed kneading", r)
	}
}
