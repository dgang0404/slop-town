// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"strings"
	"testing"
)

func TestReliefDuringResidentsBreaks(t *testing.T) {
	for _, leave := range []bool{false, true} {
		c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
		if err != nil {
			t.Fatal(err)
		}
		const player, porter, baker = "bakery/player", "bakery/porter", "bakery/baker"
		for tick := 1; tick <= 15; tick++ {
			p := spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})
			if leave && tick <= 2 {
				p.Action, p.Args.Direction = ActionMove, South
			}
			r := c.Submit(player, p)
			if r.Status != "accepted" {
				t.Fatal(r)
			}
			for _, worker := range []struct {
				id                   string
				restTick, preference int
			}{{porter, 10, 87}, {baker, 12, 40}} {
				want := 0
				if tick >= worker.restTick {
					want = max(0, reliefDuration-(tick-worker.restTick))
				}
				e := c.bakery.Entities[worker.id]
				if e.ReliefTurns != want || e.RestInPlacePercent != worker.preference {
					t.Fatalf("tick %d: %s: %+v", tick, worker.id, e)
				}
				obs, err := c.Observe(worker.id)
				if err != nil || strings.Contains(obs.Text, "You feel relieved.") != (want == reliefDuration) || strings.Contains(obs.Text, "A little of that relief lingers.") != (want > 0 && want < reliefDuration) {
					t.Fatalf("tick %d: %s: %v, %s", tick, worker.id, err, obs.Text)
				}
			}
			if tick == 12 && strings.Contains(r.Observation.Text, "The baker's shoulders look relaxed.") == leave {
				t.Fatalf("player view on the baker's break, left=%t: %s", leave, r.Observation.Text)
			}
			if c.bakery.Entities[player].ReliefTurns != 0 {
				t.Fatal("witness acquired another actor's emotion")
			}
		}
	}
}

func TestReliefFromPlayersOwnRest(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player = "bakery/player"
	// Back-and-forth pacing produces fatigue without qualifying for endurance.
	for i, dir := range []Direction{West, East, West, East, West} {
		if r := c.Submit(player, spatialProposal(c, fmt.Sprint(i), ActionMove, ActionArgs{Direction: dir})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	p := spatialProposal(c, "recover", ActionRest, ActionArgs{})
	r := c.Submit(player, p)
	if r.Status != "accepted" || !strings.Contains(r.Observation.Text, "You feel relieved.") || c.bakery.Entities[player].EnduranceBouts != 0 {
		t.Fatal(r)
	}
	submitUnchanged(t, c, player, p)
	bad := spatialProposal(c, "bad", ActionWait, ActionArgs{Target: "bakery/door"})
	if r := submitUnchanged(t, c, player, bad); r.Status != "rejected" {
		t.Fatal(r)
	}
	for i := 1; i <= reliefDuration; i++ {
		p.RequestID = fmt.Sprintf("sit-%d", i)
		if r := c.Submit(player, p); r.Status != "accepted" || c.bakery.Entities[player].ReliefTurns != reliefDuration-i {
			t.Fatal("resting again renewed relief", r)
		}
	}
}

func TestRestoreInvalidRelief(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id             string
		version, turns int
	}{
		{"bakery/porter", practiceContentVersion, 1}, {"bakery/crate", emotionContentVersion, 1}, {"bakery/player", emotionContentVersion, -1}, {"bakery/baker", emotionContentVersion, reliefDuration + 1},
	} {
		b := c.bakery
		usePracticeVersion(b, tc.version)
		e := b.Entities[tc.id]
		original := e
		e.ReliefTurns = tc.turns
		b.Entities[tc.id] = e
		raw, err := b.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(raw, b.WorldID, c.clock); err == nil {
			t.Fatalf("accepted invalid relief %+v", tc)
		}
		b.Entities[tc.id] = original
	}
}
