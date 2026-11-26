// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"testing"
)

func TestNotebookRecordsVisibleChanges(t *testing.T) {
	const player, baker, porter = "town/player", "town/baker", "town/porter"
	c := notebookCoordinator(t, 42)
	for tick := 1; tick <= 12; tick++ {
		p := spatialProposal(c, fmt.Sprint(tick), ActionWait, ActionArgs{})
		if tick <= 2 {
			p.Action = ActionRest
		}
		if r := c.Submit(player, p); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	// The porter saw the player sit; the baker opened the door later on
	// that tick. Resting again while already seated adds no new observation.
	// At tick 12 the player can see the baker sit, from beyond the porter's view.
	rest := c.events[len(c.events)-1]
	if rest.ActorID != baker || rest.Kind != "actor_rested" || rest.Tick != 12 {
		t.Fatal(rest)
	}
	for owner, want := range map[string][]NotebookEntry{
		porter: {{Tick: 1, SourceSequence: 2, Subject: player, Action: ActionRest}},
		baker:  nil,
		player: {{Tick: 12, SourceSequence: rest.Sequence, Subject: baker, Action: ActionRest}},
	} {
		got, err := c.bakery.Notebook(owner)
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("%s: got %+v, want %+v, %v", owner, got, want, err)
		}
		if len(got) > 0 {
			got[0].Tick = 999
			again, _ := c.bakery.Notebook(owner)
			if !slices.Equal(again, want) {
				t.Fatal("reading a notebook exposed the original entries")
			}
		}
	}
}

func TestNotebookRestoreChecksSightings(t *testing.T) {
	const player, baker = "town/player", "town/baker"
	c, err := (BakerySetup{Resident: true}).NewCoordinator(Scope{"town", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	if r := c.Submit(player, spatialProposal(c, "first", ActionRest, ActionArgs{})); r.Status != "accepted" {
		t.Fatal(r)
	}
	good := NotebookEntry{Tick: 1, SourceSequence: 2, Subject: player, Action: ActionRest}
	for _, tc := range []struct {
		owner   string
		entries []NotebookEntry
	}{
		{"town/spoon", []NotebookEntry{good}},
		{player, []NotebookEntry{good}},
		{baker, []NotebookEntry{good, good}},
		{baker, []NotebookEntry{{Tick: 2, SourceSequence: 2, Subject: player, Action: ActionRest}}},
		{baker, []NotebookEntry{{Tick: 1, SourceSequence: 2, Subject: player, Action: ActionClose, Target: "town/spoon"}}},
	} {
		c.bakery.Notebooks = map[string][]NotebookEntry{tc.owner: tc.entries}
		data, err := c.bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(data, "town", c.clock); err == nil {
			t.Fatalf("accepted invalid notebook %+v", tc)
		}
	}
}
