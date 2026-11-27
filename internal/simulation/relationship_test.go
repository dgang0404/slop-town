// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestShopfrontIntroduction(t *testing.T) {
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	for _, leave := range []bool{false, true} {
		c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
		if err != nil {
			t.Fatal(err)
		}
		for tick := 1; tick <= 4; tick++ {
			p := spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})
			if leave && tick <= 2 {
				p.Action, p.Args.Direction = ActionMove, South
			}
			r := c.Submit(player, p)
			if r.Status != "accepted" {
				t.Fatal(r)
			}
			if tick == 3 {
				if !c.bakery.recognizes(baker, porter) || c.bakery.recognizes(porter, baker) ||
					c.bakery.recognizes(player, porter) || strings.Contains(r.Message, "introduc") || strings.Contains(r.Observation.Text, "recognize") {
					t.Fatalf("the porter's introduction to the baker reached the wrong actor: %+v", r)
				}
				view, err := c.Observe(baker)
				if err != nil || !strings.Contains(view.Text, "You recognize the porter from their introduction.") {
					t.Fatal(view, err)
				}
			}
			if tick == 4 && (strings.Contains(r.Message, "The baker introduces themselves to you.") != !leave || c.bakery.recognizes(player, baker) != !leave) {
				t.Fatalf("player at shopfront=%t, got %+v", !leave, r)
			}
		}
	}
}

func TestDirectedIntroduction(t *testing.T) {
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	intro := spatialProposal(c, "hello", ActionIntroduce, ActionArgs{Target: porter})
	for _, target := range []string{porter, baker, player, "bakery/door"} {
		bad := intro
		bad.Args.Target = target
		if r := submitUnchanged(t, c, player, bad); r.Status != "rejected" {
			t.Fatal(r)
		}
	}
	if r := c.Submit(player, spatialProposal(c, "south", ActionMove, ActionArgs{Direction: South})); r.Status != "accepted" {
		t.Fatal(r)
	}
	if view, err := c.Observe(baker); err != nil || view.hasTarget(player) {
		t.Fatal("expected the baker to miss the introduction", view, err)
	}
	options, err := c.AvailableActions(player)
	if err != nil || !slices.Contains(options, ActionOption{ActionIntroduce, ActionArgs{Target: "porter"}}) {
		t.Fatal(options, err)
	}
	if r := c.Submit(player, intro); r.Status != "accepted" {
		t.Fatal(r)
	}
	submitUnchanged(t, c, player, intro)
	intro.RequestID = "again"
	if r := c.Submit(player, intro); r.Status != "accepted" {
		t.Fatal(r)
	}
	first := NotebookEntry{Tick: 2, SourceSequence: 6, Subject: player, Target: porter, Action: ActionIntroduce}
	var notes []NotebookEntry
	for _, n := range c.bakery.Notebooks[porter] {
		if n.Action == ActionIntroduce {
			notes = append(notes, n)
		}
	}
	if !slices.Equal(notes, []NotebookEntry{first}) || c.bakery.recognizes(player, porter) || c.bakery.recognizes(baker, player) {
		t.Fatal("unexpected familiarity after repeated introduction", notes)
	}
	// The porter can choose to return the introduction on a separate turn.
	back := spatialProposal(c, "reply", ActionIntroduce, ActionArgs{Target: player})
	if r := c.Submit(porter, back); r.Status != "accepted" || !c.bakery.recognizes(player, porter) {
		t.Fatal(r)
	}
}

func TestRestoreInvalidIntroduction(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.clock.Advance(1); err != nil {
		t.Fatal(err)
	}
	const player, porter = "bakery/player", "bakery/porter"
	first := NotebookEntry{Tick: 1, SourceSequence: 2, Action: ActionIntroduce, Subject: player, Target: porter}
	second := first
	second.SourceSequence++
	for _, tc := range []struct {
		owner   string
		version int
		notes   []NotebookEntry
	}{
		{"bakery/baker", relationshipContentVersion, []NotebookEntry{first}},
		{porter, emotionContentVersion, []NotebookEntry{first}},
		{porter, relationshipContentVersion, []NotebookEntry{first, second}},
	} {
		usePracticeVersion(c.bakery, tc.version)
		c.bakery.Notebooks = map[string][]NotebookEntry{tc.owner: tc.notes}
		raw, err := c.bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(raw, c.scope.WorldID, c.clock); err == nil {
			t.Fatalf("invalid introduction accepted: %+v", tc)
		}
	}
}
