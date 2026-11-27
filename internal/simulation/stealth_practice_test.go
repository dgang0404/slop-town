// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "testing"

func TestResidentLearnsFromDiscreetPickups(t *testing.T) {
	const baker, player, spoon, door = "bakery/baker", "bakery/player", "bakery/spoon", "bakery/door"
	container := stealthPractice{Container: true}
	both := stealthPractice{Floor: true, Container: true}
	for _, tc := range []struct {
		name         string
		prior, after stealthPractice
		witness      string
		action       ActionKind
		version      int
		credit       string
		memory       ActionKind
	}{
		{"identified while learning", container, both, "here", ActionDiscreetTake, stealthPracticeContentVersion, "floor", ActionTake},
		{"already practiced", both, both, "here", ActionDiscreetTake, stealthPracticeContentVersion, "", SightHandMovement},
		{"distant witness", container, container, "two steps away", ActionDiscreetTake, stealthPracticeContentVersion, "", SightHandMovement},
		{"nobody in sight", container, container, "outside", ActionDiscreetTake, stealthPracticeContentVersion, "", ""},
		{"ordinary pickup", container, container, "here", ActionTake, stealthPracticeContentVersion, "", ActionTake},
		{"older bakery", stealthPractice{}, stealthPractice{}, "here", ActionDiscreetTake, daylightContentVersion, "", ActionTake},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 5)
			if err != nil {
				t.Fatal(err)
			}
			b := c.bakery
			b.ContentVersion = tc.version
			actor, item, observer, entrance := b.Entities[baker], b.Entities[spoon], b.Entities[player], b.Entities[door]
			actor.StealthPractice = tc.prior
			item.Placement = actor.Placement
			if tc.witness != "outside" {
				entrance.Open = true
				observer.Placement = actor.Placement
				if tc.witness == "two steps away" {
					observer.Placement.Cell.X++
					observer.Placement.Cell.Y++
				}
			}
			b.Entities[baker], b.Entities[spoon], b.Entities[player], b.Entities[door] = actor, item, observer, entrance
			p := spatialProposal(c, "pickup", tc.action, ActionArgs{Target: spoon})
			r := c.Submit(baker, p)
			if r.Status != "accepted" || c.bakery.Entities[baker].StealthPractice != tc.after {
				t.Fatal(r, c.bakery.Entities[baker])
			}
			event := c.History()[1]
			if event.Payload.StealthPractice != tc.credit {
				t.Fatalf("pickup credited %q, want %q", event.Payload.StealthPractice, tc.credit)
			}
			entry, found := c.bakery.firsthandAt(player, event.Sequence)
			if found != (tc.memory != "") || entry.Action != tc.memory {
				t.Fatal("unexpected eyewitness account", entry)
			}
			submitUnchanged(t, c, baker, p)
		})
	}
}

func TestRestoreRejectsStealthPracticeOnItemsAndOldActors(t *testing.T) {
	for _, tc := range []struct {
		id      string
		version int
	}{{"bakery/spoon", stealthPracticeContentVersion}, {"bakery/player", daylightContentVersion}} {
		c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 5)
		if err != nil {
			t.Fatal(err)
		}
		c.bakery.ContentVersion = tc.version
		entity := c.bakery.Entities[tc.id]
		entity.StealthPractice.Floor = true
		c.bakery.Entities[tc.id] = entity
		data, err := c.bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(data, "bakery", c.clock); err == nil {
			t.Fatalf("accepted unexpected practice for %s in version %d", tc.id, tc.version)
		}
	}
}
