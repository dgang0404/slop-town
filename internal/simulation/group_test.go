// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"strings"
	"testing"
)

func TestFreshTownCheckInIsOverheard(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.ContentVersion < groupConversationVersion {
		t.Fatalf("new practice town is content %d", c.bakery.ContentVersion)
	}
	const baker, porter, player = "bakery/baker", "bakery/porter", "bakery/player"
	spot := c.bakery.Entities[baker].Placement
	for _, id := range []string{porter, player} {
		actor := c.bakery.Entities[id]
		actor.Placement = spot
		c.bakery.Entities[id] = actor
	}
	if r := c.Submit(baker, spatialProposal(c, "check-in", ActionSpeak, ActionArgs{Target: porter, Speech: checkInQuestion})); r.Status != "accepted" {
		t.Fatal(r)
	}
	var overheard bool
	for _, note := range c.bakery.Notebooks[player] {
		if note.Overheard && note.Subject == baker && note.Target == porter {
			overheard = true
		}
	}
	if !overheard {
		t.Fatal(c.bakery.Notebooks[player])
	}
}

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
	b = c.bakery
	askPorter := b.Entities[player]
	askPorter.Placement = b.Entities[porter].Placement
	b.Entities[player] = askPorter
	const secondhand = "The baker told me you heard the baker check in on me. They said they felt fine."
	if r := c.Submit(player, spatialProposal(c, "porter-hearsay", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "hearsay", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, secondhand) {
		t.Fatal(r)
	}
	b = c.bakery
	askPorter = b.Entities[player]
	askPorter.Placement = b.Entities[porter].Placement
	b.Entities[player] = askPorter
	if r := c.Submit(player, spatialProposal(c, "porter-talk", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "The baker checked in on me. I told them I felt fine.") || strings.Contains(r.Message, "you heard") {
		t.Fatal(r)
	}
}

func TestBystanderAnswersBeforeThePersonAsked(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	const baker, porter, player = "bakery/baker", "bakery/porter", "bakery/player"
	place := func() {
		b := c.bakery
		b.ContentVersion = answerFirstVersion
		spot := b.Entities[baker].Placement
		for _, id := range []string{porter, player} {
			actor := b.Entities[id]
			actor.Placement = spot
			b.Entities[id] = actor
		}
	}
	place()
	if r := c.Submit(baker, spatialProposal(c, "standing", ActionSpeak, ActionArgs{Target: porter, Speech: checkInQuestion})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "Doing all right, thank you.") {
		t.Fatal(r)
	}
	place()
	seated := c.bakery.Entities[porter]
	seated.Seated = true
	c.bakery.Entities[porter] = seated
	if r := c.Submit(baker, spatialProposal(c, "sitting", ActionSpeak, ActionArgs{Target: porter, Speech: checkInQuestion})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "How are you holding up?") || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	place()
	seated = c.bakery.Entities[porter]
	seated.Seated = true
	c.bakery.Entities[porter] = seated
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[player] {
		if note.Overheard && note.Open && note.Subject == baker && note.Target == porter && note.Condition == "" {
			open = note
		}
	}
	if open.SourceSequence == 0 {
		t.Fatal(c.bakery.Notebooks[player])
	}
	for _, note := range c.bakery.Notebooks[porter] {
		if note.Open && note.Condition != "" {
			t.Fatal(note)
		}
	}
	choices, err := c.DialogueChoices(player, baker)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	const words = "The porter is taking a breather."
	for _, choice := range choices {
		if choice.Words == words {
			answer = choice.Line
		}
	}
	if answer.Intent != "answer_first" || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(player, spatialProposal(c, "answer", ActionSpeak, ActionArgs{Target: baker, Speech: answer})); r.Status != "accepted" ||
		!strings.Contains(r.Message, words) || !strings.Contains(r.Message, "The baker replies, “All right.”") {
		t.Fatal(r)
	}
	var answered bool
	for _, note := range c.bakery.Notebooks[player] {
		if note.Answered && note.Subject == porter && note.Condition == resting {
			answered = true
		}
	}
	if !answered {
		t.Fatal(c.bakery.Notebooks[player])
	}
	for _, note := range c.bakery.Notebooks[porter] {
		if note.SourceSequence >= open.SourceSequence && note.Condition != "" {
			t.Fatal("the porter answered", note)
		}
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
	place()
	seated = c.bakery.Entities[porter]
	seated.Seated = true
	c.bakery.Entities[porter] = seated
	choices, err = c.DialogueChoices(porter, baker)
	if err != nil {
		t.Fatal(err)
	}
	var pickUp SpeechLine
	const agreed = "Yes. I was taking a breather."
	for _, choice := range choices {
		if choice.Words == agreed {
			pickUp = choice.Line
		}
	}
	if pickUp.Intent != "pick_up" || pickUp.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(porter, spatialProposal(c, "pick-up", ActionSpeak, ActionArgs{Target: baker, Speech: pickUp})); r.Status != "accepted" ||
		!strings.Contains(r.Message, agreed) || !strings.Contains(r.Message, "The baker replies, “All right.”") {
		t.Fatal(r)
	}
	var own bool
	for _, note := range c.bakery.Notebooks[porter] {
		if note.Condition == resting && !note.Asked && !note.Answered && note.Subject == baker && note.SourceSequence > open.SourceSequence {
			own = true
		}
	}
	if !own {
		t.Fatal(c.bakery.Notebooks[porter])
	}
	var heard bool
	for _, note := range c.bakery.Notebooks[player] {
		if note.Overheard && !note.Open && note.Subject == baker && note.Target == porter && note.Condition == resting {
			heard = true
		}
	}
	if !heard {
		t.Fatal(c.bakery.Notebooks[player])
	}
	place()
	if r := c.Submit(baker, spatialProposal(c, "after", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You checked in on me. I told you I had stopped for a breather.") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}
