// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"strings"
	"testing"
)

func TestBystanderRemembersAVisibleCheckIn(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	b.ContentVersion = groupConversationVersion
	const baker, porter, player = "bakery/baker", "bakery/porter", "bakery/player"
	spot := b.Entities[baker].Placement
	for _, id := range []string{porter, player} {
		actor := b.Entities[id]
		actor.Placement = spot
		b.Entities[id] = actor
	}
	event := Event{Scope: scope, Tick: 2, Sequence: 4, Kind: "actors_conversed", ActorID: baker,
		Payload: EventPayload{Target: porter, Dialogue: Exchange{Condition: doingFine}}}
	b.recordWitnesses(event)
	notes := b.Notebooks[player]
	if len(notes) != 1 || !notes[0].Overheard || notes[0].Subject != baker || notes[0].Target != porter || notes[0].Condition != doingFine {
		t.Fatal(notes)
	}
	if err := b.validateNotebooks(event.Tick); err != nil {
		t.Fatal(err)
	}
	pages, err := b.Journal(scope, player, func(yield func(Event) error) error { return yield(event) }, func(string) (*Receipt, error) { return nil, nil })
	if err != nil || len(pages) != 1 || len(pages[0].Observations) != 1 ||
		!strings.Contains(pages[0].Observations[0].Text, "I heard the baker check in on the porter") {
		t.Fatal(pages, err)
	}
	view, err := b.Observe(scope, baker)
	if err != nil {
		t.Fatal(err)
	}
	asked, fail := b.speak(view, player, SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"})
	if fail != nil || !strings.Contains(asked.message, "I heard you check in on the porter. They said they felt fine.") {
		t.Fatal(asked, fail)
	}

	away := b.Entities[player]
	away.Placement = b.Entities["bakery/door"].Placement
	away.Placement.Cell.X = 0
	b.Entities[player] = away
	b.Notebooks[player] = nil
	b.recordWitnesses(event)
	if len(b.Notebooks[player]) != 0 {
		t.Fatal("a bystander out of sight remembered the check-in", b.Notebooks[player])
	}

	b.ContentVersion = conversationContentVersion
	b.Entities[player] = SpatialEntity{Name: "player", Kind: Actor, Placement: spot}
	b.Notebooks = nil
	b.recordWitnesses(event)
	if len(b.Notebooks[player]) != 0 {
		t.Fatal("older towns recorded an overheard check-in")
	}
}

func TestParticipantConfirmsWhatTheBystanderHeard(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	b.ContentVersion = groupConversationVersion
	const baker, porter, player = "bakery/baker", "bakery/porter", "bakery/player"
	spot := b.Entities[baker].Placement
	for _, id := range []string{porter, player} {
		actor := b.Entities[id]
		actor.Placement = spot
		b.Entities[id] = actor
	}
	event := Event{Scope: scope, Tick: 1, Sequence: 1, Kind: "actors_conversed", ActorID: baker,
		Payload: EventPayload{Target: porter, Dialogue: Exchange{Condition: doingFine}}}
	b.recordWitnesses(event)
	ask := SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}
	if r := c.Submit(baker, spatialProposal(c, "ask", ActionSpeak, ActionArgs{Target: player, Speech: ask})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard you check in on the porter. They said they felt fine.") {
		t.Fatal(r)
	}
	b = c.bakery
	heard := b.Entities[player]
	heard.Placement = b.Entities[baker].Placement
	b.Entities[player] = heard
	hearsay := SpeechLine{Topic: "hearsay", Intent: "ask", Tone: "curious"}
	if r := c.Submit(player, spatialProposal(c, "hearsay", ActionSpeak, ActionArgs{Target: baker, Speech: hearsay})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You told me you heard me check in on the porter. They said they felt fine.") {
		t.Fatal(r)
	}
	b = c.bakery
	heard = b.Entities[player]
	heard.Placement = b.Entities[baker].Placement
	b.Entities[player] = heard
	choices, err := c.DialogueChoices(baker, player)
	if err != nil {
		t.Fatal(err)
	}
	var confirm SpeechLine
	for _, choice := range choices {
		if choice.Words == "So you heard me check in on the porter?" {
			confirm = choice.Line
		}
	}
	if confirm.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(baker, spatialProposal(c, "confirm", ActionSpeak, ActionArgs{Target: player, Speech: confirm})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "Yes. I heard you check in on the porter. They said they felt fine.") {
		t.Fatal(r)
	}
	b = c.bakery
	beside := b.Entities[porter]
	beside.Placement = b.Entities[baker].Placement
	b.Entities[porter] = beside
	const passed = "The player told me they heard me check in on you. They said they felt fine."
	choices, err = c.DialogueChoices(baker, porter)
	if err != nil {
		t.Fatal(err)
	}
	var relay SpeechLine
	for _, choice := range choices {
		if choice.Words == passed {
			relay = choice.Line
		}
	}
	if relay.Intent != "relay" {
		t.Fatal(choices)
	}
	if r := c.Submit(baker, spatialProposal(c, "relay", ActionSpeak, ActionArgs{Target: porter, Speech: relay})); r.Status != "accepted" ||
		!strings.Contains(r.Message, passed) {
		t.Fatal(r)
	}
	b = c.bakery
	back := b.Entities[baker]
	back.Placement = b.Entities[porter].Placement
	b.Entities[baker] = back
	const check = "Did the player tell you they heard you check in on me?"
	choices, err = c.DialogueChoices(porter, baker)
	if err != nil {
		t.Fatal(err)
	}
	var follow SpeechLine
	for _, choice := range choices {
		if choice.Words == check {
			follow = choice.Line
		}
	}
	if follow.Intent != "confirm_source" {
		t.Fatal(choices)
	}
	if r := c.Submit(porter, spatialProposal(c, "follow", ActionSpeak, ActionArgs{Target: baker, Speech: follow})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "Yes. "+passed) {
		t.Fatal(r)
	}
}
