// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestResidentCheckInDuringWork(t *testing.T) {
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	// At tick four the player turns the corner inside the bakery. Both
	// workers have finished their trip to the shopfront and are out of view.
	for i, dir := range []Direction{"", East, East, South, "", "", "", ""} {
		p := spatialProposal(c, fmt.Sprint(i+1), ActionWait, ActionArgs{})
		if dir != "" {
			p.Action, p.Args.Direction = ActionMove, dir
		}
		r := c.Submit(player, p)
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		if i == 3 && (r.Observation.hasTarget(baker) || r.Observation.hasTarget(porter) || strings.Contains(r.Message, "holding up")) {
			t.Fatal(r)
		}
	}
	var meetings []Event
	history := c.History()
	for _, event := range history {
		if event.Kind == "actors_conversed" {
			meetings = append(meetings, event)
		}
	}
	if len(meetings) != 1 {
		t.Fatal(meetings)
	}
	meet := meetings[0]
	if meet.Tick != 4 || meet.ActorID != baker || meet.Payload.Target != porter || meet.Payload.Dialogue != (Exchange{Question: checkInQuestion, Condition: doingFine}) ||
		meet.ParentID != history[meet.Sequence-2].ID || history[meet.Sequence-2].Kind != "actor_moved" {
		t.Fatal(meet)
	}

	for _, id := range []string{player, baker, porter} {
		note, found := c.bakery.firsthandAt(id, meet.Sequence)
		book, err := c.Journal(id)
		if err != nil {
			t.Fatal(err)
		}
		hasSpeech := slices.ContainsFunc(book, func(page JournalEntry) bool {
			return page.Conversation.Words == checkInWords && page.Conversation.Reply == "Doing all right, thank you." && page.Conversation.IStarted == (id == baker)
		})
		if found != (id != player) || hasSpeech != found || (found && (note.Condition != doingFine || note.Asked != (id == baker))) {
			t.Fatal(id, note, book)
		}
	}
	// A notebook may not turn the other person's answer into a different one.
	notes := c.bakery.Notebooks[porter]
	for i, e := range notes {
		if e.SourceSequence == meet.Sequence {
			notes[i].Condition = needsSeat
		}
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreBakery(raw, c.scope.WorldID, c.clock); err == nil {
		t.Fatal("restored conflicting accounts of one check-in")
	}
}

func TestResidentCheckInRequiresBothWorkers(t *testing.T) {
	const baker, porter = "bakery/baker", "bakery/porter"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	front := b.Entities["bakery/player"].Placement
	for _, id := range []string{baker, porter} {
		e := b.Entities[id]
		e.Placement = front
		b.Entities[id] = e
	}
	turn := Event{Scope: c.scope, Tick: 4, ActorID: "bakery/player", Kind: "actor_waited"}
	if got := b.residentConversation(turn); got.kind != "" {
		t.Fatal("strangers began a familiar check-in", got)
	}
	// Only the porter recognizes the baker, so the porter initiates.
	b.appendSighting(porter, NotebookEntry{Tick: 1, SourceSequence: 2, Action: ActionIntroduce, Subject: baker, Target: porter})
	if got := b.residentConversation(turn); got.entityID != porter || got.payload.Target != baker {
		t.Fatal(got)
	}
	turn.Kind, turn.Payload.Target = "actors_conversed", porter
	if got := b.residentConversation(turn); got.kind != "" {
		t.Fatal("interrupted a player's exchange", got)
	}
	turn.Kind, turn.Tick = "actor_waited", 5
	if got := b.residentConversation(turn); got.kind != "" {
		t.Fatal("skipped the trip to the prep area", got)
	}
}
