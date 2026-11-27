// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"reflect"
	"testing"
)

func TestAccountsKeepDistinctWitnessesAndSightings(t *testing.T) {
	const player, baker, porter, other = "bakery/player", "bakery/baker", "bakery/porter", "bakery/other"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	b.Entities[other] = b.Entities[porter] // Two witnesses with the same visible name.
	rest := NotebookEntry{Tick: 1, SourceSequence: 2, Action: ActionRest, Subject: player}
	again := rest
	again.Tick, again.SourceSequence = 2, 4
	report := func(sequence uint64, tick int64, teller string, source uint64) NotebookEntry {
		return NotebookEntry{Tick: tick, SourceSequence: sequence, Subject: teller, Reported: source, Action: ActionSpeak}
	}
	b.Notebooks = map[string][]NotebookEntry{
		porter: {rest, again},
		other:  {rest},
		baker:  {report(6, 3, porter, 2), report(8, 4, other, 2), report(10, 5, porter, 4)},
		player: {
			report(12, 6, porter, 2), // Heard directly before the baker mentioned it.
			report(48, 24, baker, 6), report(50, 25, baker, 8), report(52, 26, baker, 10),
			report(54, 27, baker, 6), // A repeat of the first story.
		},
	}
	if err := b.validateNotebooks(27); err != nil {
		t.Fatal(err)
	}
	first := AccountTelling{Teller: "porter", FromWitness: true, Date: bakeryDateAt(6)}
	later := AccountTelling{Teller: "baker", Date: bakeryDateAt(24)}
	want := []HeardAccount{
		{Witness: "porter", Act: "me sit down", Tellings: []AccountTelling{first, later}},
		{Witness: "porter", Act: "me sit down", Tellings: []AccountTelling{later}},
		{Witness: "porter", Act: "me sit down", Tellings: []AccountTelling{later}},
	}
	got, err := c.Accounts(player)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("accounts: %+v, %v", got, err)
	}

	// Someone can repeat a witness's own account back to them.
	b.appendSighting(porter, report(56, 28, baker, 6))
	got, err = c.Accounts(porter)
	if err != nil || len(got) != 1 || !got[0].OwnSighting || got[0].Act != "the player sit down" {
		t.Fatalf("witness as reader: %+v, %v", got, err)
	}
}
