// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"testing"
)

func TestResidentKnocksOnEitherDoor(t *testing.T) {
	const player, baker = "bakery/player", "bakery/baker"
	const front, office = "bakery/door", "bakery/office-door"
	for _, tc := range []struct {
		name, target           string
		frontOpen, hears, sees bool
	}{
		{"opposite faces of the closed front door", front, false, true, false},
		{"closed front door between listener and office", office, false, false, false},
		{"open front door between listener and office", office, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 43)
			if err != nil {
				t.Fatal(err)
			}
			b := c.bakery
			door := b.Entities[front]
			door.Open = tc.frontOpen
			b.Entities[front] = door
			cell := b.Entities[baker].Placement.Cell
			cell.X++
			b.Entities[office] = SpatialEntity{Kind: Door, Name: "office door", Placement: Placement{Cell: cell}}
			p := spatialProposal(c, "knock", ActionKnock, ActionArgs{Target: tc.target})
			result := c.Submit(baker, p)
			if result.Status != "accepted" || result.Message != "You knock on the "+b.Entities[tc.target].Name+"." ||
				c.clock.Tick() != 1 || c.bakery.Entities[front].Open != tc.frontOpen || c.bakery.Entities[office].Open {
				t.Fatal(result)
			}
			submitUnchanged(t, c, baker, p)
			impact := c.History()[1]
			if impact.Kind != "door_knocked" || impact.ActorID != baker || impact.Payload != (EventPayload{Target: tc.target}) {
				t.Fatal(impact)
			}
			sight, saw := c.bakery.firsthandAt(player, impact.Sequence)
			if saw != tc.sees || (saw && (sight.Action != ActionKnock || sight.Subject != baker || sight.Target != tc.target)) {
				t.Fatal("street sighting", sight, saw)
			}
			heard := false
			for _, e := range c.History() {
				if e.Kind != "sound_heard" || e.ActorID != player {
					continue
				}
				if heard {
					t.Fatal("one knock produced two sound memories", e)
				}
				heard = true
				memory, ok := c.bakery.firsthandAt(player, e.Sequence)
				if e.ParentID != impact.ID || !ok || memory != (NotebookEntry{Tick: 1, SourceSequence: e.Sequence, Action: ActionListen, Sound: Knock}) {
					t.Fatal("sound identified an unseen cause or lost its source", e, memory)
				}
			}
			if heard != tc.hears {
				t.Fatalf("heard knock = %t, want %t", heard, tc.hears)
			}
			pages, err := c.Journal(player)
			if err != nil {
				t.Fatal(err)
			}
			var mentions []string
			for _, page := range pages {
				for _, sight := range page.Witnessed {
					if sight.Action == ActionKnock {
						mentions = append(mentions, sight.Actor.Name+" knocked "+sight.ObjectPhrase())
					}
				}
				for _, observation := range page.Observations {
					if observation.Kind == Knock {
						mentions = append(mentions, observation.Text)
					}
				}
			}
			var want []string
			if tc.sees {
				want = []string{"baker knocked on the office door"}
			} else if tc.hears {
				want = []string{"You hear a knock."}
			}
			if !slices.Equal(mentions, want) {
				t.Fatalf("journal mentions %q, want %q", mentions, want)
			}
		})
	}
}

func TestKnockRequiresAReachableDoorAndCurrentRules(t *testing.T) {
	const player, door = "bakery/player", "bakery/door"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 43)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	p := spatialProposal(c, "knock", ActionKnock, ActionArgs{Target: door})
	hasKnock := func() bool {
		t.Helper()
		menu, err := c.AvailableActions(player)
		if err != nil {
			t.Fatal(err)
		}
		return slices.Contains(menu, ActionOption{ActionKnock, ActionArgs{Target: b.Entities[door].Name}})
	}
	old := b.Entities[player]
	away := old
	away.Placement.Cell.X--
	b.Entities[player] = away
	if view, err := c.Observe(player); err != nil || !view.hasTarget(door) {
		t.Fatal("test requires a visible but distant door", view, err)
	}
	if hasKnock() || submitUnchanged(t, c, player, p).Code != "target_unavailable" {
		t.Fatal("could knock from out of reach")
	}
	b.Entities[player] = old
	if !hasKnock() || submitUnchanged(t, c, player, spatialProposal(c, "porter", ActionKnock, ActionArgs{Target: "bakery/porter"})).Code != "target_unavailable" {
		t.Fatal("expected only a nearby door to be eligible")
	}
	usePracticeVersion(b, coverContentVersion)
	if hasKnock() || submitUnchanged(t, c, player, p).Code != "action_unavailable" {
		t.Fatal("older practice timeline offered knocking")
	}
}
