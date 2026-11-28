// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
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
	var heardAnswer bool
	for _, note := range c.bakery.Notebooks[porter] {
		if note.Heard && note.Subject == player && note.Condition == resting {
			heardAnswer = true
		}
		if note.SourceSequence >= open.SourceSequence && note.Condition != "" && !note.Heard {
			t.Fatal("the porter answered", note)
		}
	}
	if !heardAnswer {
		t.Fatal(c.bakery.Notebooks[porter])
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
	place()
	if r := c.Submit(baker, spatialProposal(c, "own-answer", ActionSpeak, ActionArgs{Target: player, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I said the porter was taking a breather.") || strings.Contains(r.Message, "checked in on me") || strings.Contains(r.Message, "I told them") {
		t.Fatal(r)
	}
	place()
	if r := c.Submit(porter, spatialProposal(c, "asker", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "The player said you were taking a breather.") || strings.Contains(r.Message, "I told") || strings.Contains(r.Message, "felt fine") {
		t.Fatal(r)
	}
	place()
	if r := c.Submit(baker, spatialProposal(c, "stand", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "The player said I was taking a breather.") || strings.Contains(r.Message, "I told") {
		t.Fatal(r)
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
	place()
	if r := c.Submit(baker, spatialProposal(c, "still-own", ActionSpeak, ActionArgs{Target: player, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I said the porter was taking a breather.") || strings.Contains(r.Message, "I heard the baker check in") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestUnheardAnswerLeavesTheQuestionOpen(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	const baker, porter, player = "bakery/baker", "bakery/porter", "bakery/player"
	spot := c.bakery.Entities[baker].Placement
	c.bakery.ContentVersion = answerFirstVersion
	together := func() {
		b := c.bakery
		for _, id := range []string{baker, porter, player} {
			actor := b.Entities[id]
			actor.Placement = spot
			b.Entities[id] = actor
		}
	}
	together()
	seated := c.bakery.Entities[porter]
	seated.Seated = true
	c.bakery.Entities[porter] = seated
	if r := c.Submit(baker, spatialProposal(c, "ask", ActionSpeak, ActionArgs{Target: porter, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	together()
	away := c.bakery.Entities[porter]
	away.Seated = false
	doorY := c.bakery.Entities["bakery/door"].Placement.Cell.Y
	hidden := false
	for y := 0; y < bakeryHeight; y++ {
		if y == doorY {
			continue
		}
		away.Placement = c.bakery.Entities["bakery/door"].Placement
		away.Placement.Cell.X, away.Placement.Cell.Y = 0, y
		c.bakery.Entities[porter] = away
		view, err := c.bakery.Observe(scope, porter)
		if err == nil && !view.hasTarget(baker) && !view.hasTarget(player) {
			hidden = true
			break
		}
	}
	if !hidden {
		t.Fatal("no cell hides the other conversation")
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[player] {
		if note.Overheard && note.Open && note.Subject == baker && note.Target == porter {
			open = note
		}
	}
	choices, err := c.DialogueChoices(player, baker)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The porter is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Intent != "answer_first" || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(player, spatialProposal(c, "answer", ActionSpeak, ActionArgs{Target: baker, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	for _, note := range c.bakery.Notebooks[porter] {
		if note.Heard {
			t.Fatal("the porter heard an answer given out of sight", note)
		}
	}
	together()
	if r := c.Submit(baker, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You asked how I was holding up. I haven't answered.") || strings.Contains(r.Message, "player") || strings.Contains(r.Message, "breather") {
		t.Fatal(r)
	}
	together()
	ready := c.bakery.Entities[porter]
	ready.Seated, ready.SeekingRestSpot, ready.Fatigue, ready.ReliefTurns = false, false, 0, 0
	c.bakery.Entities[porter] = ready
	choices, err = c.DialogueChoices(porter, baker)
	if err != nil {
		t.Fatal(err)
	}
	var respond SpeechLine
	const own = "Doing all right, thank you."
	for _, choice := range choices {
		if choice.Words == own {
			respond = choice.Line
		}
	}
	if respond.Intent != "respond" || respond.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(porter, spatialProposal(c, "respond", ActionSpeak, ActionArgs{Target: baker, Speech: respond})); r.Status != "accepted" ||
		!strings.Contains(r.Message, own) || !strings.Contains(r.Message, "The baker replies, “All right.”") {
		t.Fatal(r)
	}
	together()
	if r := c.Submit(baker, spatialProposal(c, "own-reply", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You checked in on me. I told you I felt fine.") || strings.Contains(r.Message, "haven't answered") || strings.Contains(r.Message, "player") {
		t.Fatal(r)
	}
	var differed bool
	for _, note := range c.bakery.Notebooks[player] {
		if note.Overheard && note.Condition == doingFine && note.Target == porter {
			differed = true
		}
	}
	if !differed {
		t.Fatal(c.bakery.Notebooks[player])
	}
	together()
	if r := c.Submit(baker, spatialProposal(c, "different", ActionSpeak, ActionArgs{Target: player, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I said the porter was taking a breather.") || strings.Contains(r.Message, "felt fine") {
		t.Fatal(r)
	}
	playerJournal, err := c.Journal(player)
	porterJournal, err2 := c.Journal(porter)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	playerText, porterText := journalSpeech(playerJournal), journalSpeech(porterJournal)
	if !strings.Contains(playerText, "I said the porter was taking a breather.") || strings.Contains(playerText, "felt fine") ||
		!strings.Contains(porterText, "Doing all right, thank you.") || strings.Contains(porterText, "taking a breather") {
		t.Fatal(playerText, porterText)
	}
}

func TestOverlappingStartIsMissed(t *testing.T) {
	c := separateCircle(t, true, false)
	const baker, porter, player, customer = "bakery/baker", "bakery/porter", "bakery/player", "bakery/customer"
	playerJournal, err := c.Journal(player)
	besideJournal, err2 := c.Journal(porter)
	speakerJournal, err3 := c.Journal(customer)
	if err != nil || err2 != nil || err3 != nil {
		t.Fatal(err, err2, err3)
	}
	playerText, besideText, speakerText := journalSpeech(playerJournal), journalSpeech(besideJournal), journalSpeech(speakerJournal)
	if strings.Contains(playerText, "spoke at once") || strings.Contains(playerText, "other table") ||
		strings.Contains(besideText, "other table") || strings.Contains(besideText, "let them") ||
		!strings.Contains(besideText, "The customer and the neighbor spoke at once. I missed what they said.") ||
		!strings.Contains(speakerText, "Hello from the other table.") {
		t.Fatal(playerText, besideText, speakerText)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
	at := c.bakery.Entities[baker].Placement
	who := c.bakery.Entities[porter]
	who.Placement = at
	c.bakery.Entities[porter] = who
	if r := c.Submit(baker, spatialProposal(c, "retell", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "The customer and the neighbor spoke at once. I missed what they said.") ||
		strings.Contains(r.Message, "other table") || strings.Contains(r.Message, "breather") || strings.Contains(r.Message, "checked in") {
		t.Fatal(r)
	}
}

func TestSeparateCircleSpeaksDuringTheSameTurn(t *testing.T) {
	c := separateCircle(t, false, false)
	const baker, porter, player = "bakery/baker", "bakery/porter", "bakery/player"
	playerJournal, err := c.Journal(player)
	besideJournal, err2 := c.Journal(porter)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	playerText, besideText := journalSpeech(playerJournal), journalSpeech(besideJournal)
	if strings.Contains(playerText, "other table") || strings.Contains(playerText, "let them") ||
		!strings.Contains(besideText, "I heard the customer speak from the other table.") || !strings.Contains(besideText, "The neighbor let them go first.") {
		t.Fatal(playerText, besideText)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
	at := c.bakery.Entities[baker].Placement
	who := c.bakery.Entities[porter]
	who.Placement = at
	c.bakery.Entities[porter] = who
	if r := c.Submit(baker, spatialProposal(c, "retell", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") || !strings.Contains(r.Message, "The neighbor let them go first.") ||
		strings.Contains(r.Message, "breather") || strings.Contains(r.Message, "checked in") {
		t.Fatal(r)
	}
}

func TestRestedListenerAnswersBothCircles(t *testing.T) {
	c := separateCircle(t, false, true)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	pages, err := c.Journal(guest)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if !strings.Contains(text, "I heard the baker ask the porter how they were holding up.") || !strings.Contains(text, "I heard the customer speak from the other table.") {
		t.Fatal(text)
	}
	ask := func(who, id string) Result {
		at := c.bakery.Entities[who].Placement
		listener := c.bakery.Entities[guest]
		listener.Placement = at
		c.bakery.Entities[guest] = listener
		return c.Submit(who, spatialProposal(c, id, ActionSpeak, ActionArgs{Target: guest, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "check-in"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the baker ask the porter how they were holding up.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if r := ask(customer, "greeting"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
		strings.Contains(r.Message, "holding up") || strings.Contains(r.Message, "breather") {
		t.Fatal(r)
	}
	if r := ask(baker, "still-open"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the baker ask the porter how they were holding up.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if r := ask(customer, "still-greeting"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "holding up") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestNoonWeighsOnTheBaker(t *testing.T) {
	c := residentBetween(t, false, false, false)
	const baker, porter = "bakery/baker", "bakery/porter"
	if c.bakery.Entities[porter].Occupied || !c.bakery.Entities[baker].Occupied {
		t.Fatal(c.bakery.Entities[porter].Occupied, c.bakery.Entities[baker].Occupied)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestNoonMakesTheBakerMiss(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if c.bakery.Entities[porter].Occupied || !c.bakery.Entities[baker].Occupied {
		t.Fatal(c.bakery.Entities[porter].Occupied, c.bakery.Entities[baker].Occupied)
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	before, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(journalSpeech(before), "other table") {
		t.Fatal(journalSpeech(before))
	}
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	porterAt := c.bakery.Entities[porter]
	porterAt.Placement = heard
	c.bakery.Entities[porter] = porterAt
	for _, id := range []string{player, guest} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	bakerJournal, err := c.Journal(baker)
	porterJournal, err2 := c.Journal(porter)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if c.bakery.Entities[porter].Occupied || !strings.Contains(journalSpeech(bakerJournal), "I missed what they said.") || strings.Contains(journalSpeech(bakerJournal), "other table") ||
		!strings.Contains(journalSpeech(porterJournal), "I heard the customer speak from the other table.") {
		t.Fatal(journalSpeech(bakerJournal), journalSpeech(porterJournal))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestNoonDrawsTheBakersAttention(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	const guestFocus = "bakery/guest"
	if c.bakery.Entities[baker].Attention != guestFocus || c.bakery.Entities[porter].Attention != "" {
		t.Fatal(c.bakery.Entities[baker].Attention, c.bakery.Entities[porter].Attention)
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, porter} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, guest} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.Entities[baker].Attention != "" || !strings.Contains(journalSpeech(pages), "I was following another conversation. I missed what they said.") || strings.Contains(journalSpeech(pages), "other table") {
		t.Fatal(c.bakery.Entities[baker].Attention, journalSpeech(pages))
	}
}

func TestRoutineAnswerLetsTheAttentionGo(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if c.bakery.Entities[baker].Attention != guest {
		t.Fatal(c.bakery.Entities[baker].Attention)
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "routine", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || !strings.Contains(r.Message, "Doing all right, thank you.") || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, porter} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, guest} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if c.bakery.Entities[baker].Attention != "" || !strings.Contains(text, "I heard the customer speak from the other table.") || strings.Contains(text, "I was following another conversation.") {
		t.Fatal(c.bakery.Entities[baker].Attention, text)
	}
}

func TestBringerStillFollowsTheHeavyAnswer(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare, clerk = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare", "bakery/clerk"
	if c.bakery.Entities[guest].Attention != baker || c.bakery.Entities[baker].Attention != guest {
		t.Fatal(c.bakery.Entities[guest].Attention, c.bakery.Entities[baker].Attention)
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "routine", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[guest].Attention != baker {
		t.Fatal(r, c.bakery.Entities[baker].Attention, c.bakery.Entities[guest].Attention)
	}
	c.bakery.Entities[clerk] = SpatialEntity{Name: "clerk", Kind: Actor, Placement: station}
	for _, id := range []string{player, spare, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestJournal, err := c.Journal(guest)
	bakerJournal, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	guestText, bakerText := journalSpeech(guestJournal), journalSpeech(bakerJournal)
	if !strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "other table") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
}

func TestReleasedAttentionSurvivesReopen(t *testing.T) {
	c := releasedBringerHearsTheCircle(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	guestText, bakerText := journalSpeech(afterGuest), journalSpeech(afterBaker)
	if journalSpeech(beforeGuest) != guestText || journalSpeech(beforeBaker) != bakerText ||
		!strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[baker].Attention != "" {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestBothRepeatTheHeardLine(t *testing.T) {
	c := releasedBringerHearsTheCircle(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
			strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestRepeatedLineSurvivesReopen(t *testing.T) {
	c := releasedBringerHearsTheCircle(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "reopened-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
			strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestRoutineAnswerReleasesDrawnAttention(t *testing.T) {
	c, heard, station := attentionDrawnBack(t)
	const baker, player = "bakery/baker", "bakery/player"
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "release", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || !strings.Contains(r.Message, "Doing all right, thank you.") || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	finishDrawnAside(t, c, heard, station)
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if c.bakery.Entities[baker].Attention != "" || !strings.Contains(text, "I heard the customer speak from the other table.") || strings.Contains(text, "I was following another conversation.") {
		t.Fatal(c.bakery.Entities[baker].Attention, text)
	}
}

func TestFadedFreshAnswerSurvivesReopen(t *testing.T) {
	c := fadedFreshAnswerHearsTheCircle(t)
	const baker = "bakery/baker"
	before, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	after, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(after)
	if journalSpeech(before) != text || !strings.Contains(text, "I heard the customer speak from the other table.") || strings.Contains(text, "I was following another conversation.") || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(text, restored.Entities[baker].Attention, restored.Entities[baker].Occupied)
	}
}

func TestFadedBringerRepeatsTheHeardLine(t *testing.T) {
	c := fadedBringerHearsTheCircle(t)
	if r := askFadedBringer(c, "ask-bringer"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
}

func TestFadedBringerReplySurvivesReopen(t *testing.T) {
	c := fadedBringerHearsTheCircle(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	if r := askFadedBringer(c, "reopened-bringer"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
}

func TestDrawnBringerSplitSurvivesReopen(t *testing.T) {
	c := freshAnswerDrawsTheBringerBack(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestDrawnBringerSplitIsWhatTheyRepeat(t *testing.T) {
	c := freshAnswerDrawsTheBringerBack(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "ask-baker"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "ask-guest"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestDrawnBringerRepliesSurviveReopen(t *testing.T) {
	c := freshAnswerDrawsTheBringerBack(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "reopened-baker"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "reopened-guest"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestBothRepeatTheLineAfterTheMiss(t *testing.T) {
	c := nextCircleAfterTheMissIsHeard(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestRepeatedLineAfterTheMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheMissIsHeard(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "reopened-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestDrawnAgainSplitIsWhatTheyRepeat(t *testing.T) {
	c := freshAnswerDrawsAttentionBackAgain(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "ask-baker-again"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "ask-guest-again"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestDrawnAgainRepliesSurviveReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackAgain(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "reopened-baker-again"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "reopened-guest-again"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestBothRepeatTheLineAfterTheSecondMiss(t *testing.T) {
	c := nextCircleAfterTheSecondMissIsHeard(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "second-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestRepeatedLineAfterTheSecondMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheSecondMissIsHeard(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "reopened-second-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestDrawnOnceMoreSplitIsWhatTheyRepeat(t *testing.T) {
	c := freshAnswerDrawsAttentionBackOnceMore(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "ask-baker-once-more"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "ask-guest-once-more"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestDrawnOnceMoreRepliesSurviveReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackOnceMore(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "reopened-baker-once-more"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "reopened-guest-once-more"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestBothRepeatTheLineAfterTheLatestMiss(t *testing.T) {
	c := nextCircleAfterTheLatestMissIsHeard(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "latest-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestRepeatedLineAfterTheLatestMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheLatestMissIsHeard(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "reopened-latest-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestDrawnYetAgainSplitIsWhatTheyRepeat(t *testing.T) {
	c := freshAnswerDrawsAttentionBackYetAgain(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "ask-baker-yet-again"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "ask-guest-yet-again"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestDrawnYetAgainRepliesSurviveReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackYetAgain(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "reopened-baker-yet-again"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "reopened-guest-yet-again"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestBothRepeatTheLineAfterTheFurtherMiss(t *testing.T) {
	c := nextCircleAfterTheFurtherMissIsHeard(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "further-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestRepeatedLineAfterTheFurtherMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheFurtherMissIsHeard(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "reopened-further-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestBothRepeatTheLineAfterTheAnewMiss(t *testing.T) {
	c := nextCircleAfterTheAnewMissIsHeard(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	for _, id := range []string{baker, guest} {
		if r := ask(id, "anew-"+id); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
			t.Fatal(id, r)
		}
	}
}

func TestHeardCircleAfterTheAnewMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheAnewMissIsHeard(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestNextCircleAfterTheAnewMissIsHeard(t *testing.T) {
	nextCircleAfterTheAnewMissIsHeard(t)
}

func nextCircleAfterTheAnewMissIsHeard(t *testing.T) *Coordinator {
	t.Helper()
	c := freshAnswerDrawsAttentionBackAnew(t)
	const baker, guest, player, envoy, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/envoy", "bakery/clerk"
	if c.bakery.Entities[guest].Occupied || c.bakery.Entities[guest].Attention != "" || c.bakery.Entities[baker].Occupied || c.bakery.Entities[baker].Attention != "" {
		t.Fatal(c.bakery.Entities[guest].Occupied, c.bakery.Entities[guest].Attention, c.bakery.Entities[baker].Occupied, c.bakery.Entities[baker].Attention)
	}
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[envoy] = SpatialEntity{Name: "envoy", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, envoy, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == envoy
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-envoy", ActionSpeak, ActionArgs{Target: envoy, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == envoy {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The envoy is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "after-the-anew-miss", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestDrawnAnewRepliesSurviveReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackAnew(t)
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "reopened-baker-anew"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "reopened-guest-anew"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestDrawnAnewSplitIsWhatTheyRepeat(t *testing.T) {
	c := freshAnswerDrawsAttentionBackAnew(t)
	const baker, guest, customer = "bakery/baker", "bakery/guest", "bakery/customer"
	ask := func(id, name string) Result {
		who := c.bakery.Entities[id]
		who.Placement = c.bakery.Entities[customer].Placement
		c.bakery.Entities[id] = who
		return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: id, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "ask-baker-anew"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(baker, r)
	}
	if r := ask(guest, "ask-guest-anew"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I was following another conversation.") {
		t.Fatal(guest, r)
	}
}

func TestDrawnAnewSplitSurvivesReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackAnew(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestFreshAnswerDrawsAttentionBackAnew(t *testing.T) {
	freshAnswerDrawsAttentionBackAnew(t)
}

func freshAnswerDrawsAttentionBackAnew(t *testing.T) *Coordinator {
	t.Helper()
	c := nextCircleAfterTheFurtherMissIsHeard(t)
	const baker, guest, player, pilot, guard, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/pilot", "bakery/guard", "bakery/clerk"
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[pilot] = SpatialEntity{Name: "pilot", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{baker, guest, player, pilot} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == pilot
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-pilot", ActionSpeak, ActionArgs{Target: pilot, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == pilot {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var heavy SpeechLine
	for _, choice := range choices {
		if choice.Words == "The pilot is taking a breather." && choice.Line.Source == open.SourceSequence {
			heavy = choice.Line
		}
	}
	if heavy.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "draw-anew", ActionSpeak, ActionArgs{Target: player, Speech: heavy})); r.Status != "accepted" || c.bakery.Entities[guest].Attention != player || !c.bakery.Entities[guest].Occupied || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[guest].Attention, c.bakery.Entities[guest].Occupied, c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	c.bakery.Entities[guard] = SpatialEntity{Name: "guard", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, guard, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == guard
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-guard", ActionSpeak, ActionArgs{Target: guard, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	open = NotebookEntry{}
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == guard {
			open = note
		}
	}
	choices, err = c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The guard is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "circle-anew", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestHeardCircleAfterTheFurtherMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheFurtherMissIsHeard(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestNextCircleAfterTheFurtherMissIsHeard(t *testing.T) {
	nextCircleAfterTheFurtherMissIsHeard(t)
}

func nextCircleAfterTheFurtherMissIsHeard(t *testing.T) *Coordinator {
	t.Helper()
	c := freshAnswerDrawsAttentionBackYetAgain(t)
	const baker, guest, player, scout, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/scout", "bakery/clerk"
	if c.bakery.Entities[guest].Occupied || c.bakery.Entities[guest].Attention != "" || c.bakery.Entities[baker].Occupied || c.bakery.Entities[baker].Attention != "" {
		t.Fatal(c.bakery.Entities[guest].Occupied, c.bakery.Entities[guest].Attention, c.bakery.Entities[baker].Occupied, c.bakery.Entities[baker].Attention)
	}
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[scout] = SpatialEntity{Name: "scout", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, scout, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == scout
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-scout", ActionSpeak, ActionArgs{Target: scout, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == scout {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The scout is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "after-the-further-miss", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestDrawnYetAgainSplitSurvivesReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackYetAgain(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestFreshAnswerDrawsAttentionBackYetAgain(t *testing.T) {
	freshAnswerDrawsAttentionBackYetAgain(t)
}

func freshAnswerDrawsAttentionBackYetAgain(t *testing.T) *Coordinator {
	t.Helper()
	c := nextCircleAfterTheLatestMissIsHeard(t)
	const baker, guest, player, guide, host, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/guide", "bakery/host", "bakery/clerk"
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[guide] = SpatialEntity{Name: "guide", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{baker, guest, player, guide} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == guide
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-guide", ActionSpeak, ActionArgs{Target: guide, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == guide {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var heavy SpeechLine
	for _, choice := range choices {
		if choice.Words == "The guide is taking a breather." && choice.Line.Source == open.SourceSequence {
			heavy = choice.Line
		}
	}
	if heavy.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "draw-yet-again", ActionSpeak, ActionArgs{Target: player, Speech: heavy})); r.Status != "accepted" || c.bakery.Entities[guest].Attention != player || !c.bakery.Entities[guest].Occupied || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[guest].Attention, c.bakery.Entities[guest].Occupied, c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	c.bakery.Entities[host] = SpatialEntity{Name: "host", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, host, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == host
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-host", ActionSpeak, ActionArgs{Target: host, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	open = NotebookEntry{}
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == host {
			open = note
		}
	}
	choices, err = c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The host is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "circle-yet-again", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestHeardCircleAfterTheLatestMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheLatestMissIsHeard(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestNextCircleAfterTheLatestMissIsHeard(t *testing.T) {
	nextCircleAfterTheLatestMissIsHeard(t)
}

func nextCircleAfterTheLatestMissIsHeard(t *testing.T) *Coordinator {
	t.Helper()
	c := freshAnswerDrawsAttentionBackOnceMore(t)
	const baker, guest, player, rider, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/rider", "bakery/clerk"
	if c.bakery.Entities[guest].Occupied || c.bakery.Entities[guest].Attention != "" || c.bakery.Entities[baker].Occupied || c.bakery.Entities[baker].Attention != "" {
		t.Fatal(c.bakery.Entities[guest].Occupied, c.bakery.Entities[guest].Attention, c.bakery.Entities[baker].Occupied, c.bakery.Entities[baker].Attention)
	}
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[rider] = SpatialEntity{Name: "rider", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, rider, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == rider
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-rider", ActionSpeak, ActionArgs{Target: rider, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == rider {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The rider is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "after-the-latest-miss", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestDrawnOnceMoreSplitSurvivesReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackOnceMore(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestFreshAnswerDrawsAttentionBackOnceMore(t *testing.T) {
	freshAnswerDrawsAttentionBackOnceMore(t)
}

func freshAnswerDrawsAttentionBackOnceMore(t *testing.T) *Coordinator {
	t.Helper()
	c := nextCircleAfterTheSecondMissIsHeard(t)
	const baker, guest, player, miller, page, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/miller", "bakery/page", "bakery/clerk"
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[miller] = SpatialEntity{Name: "miller", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{baker, guest, player, miller} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == miller
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-miller", ActionSpeak, ActionArgs{Target: miller, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == miller {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var heavy SpeechLine
	for _, choice := range choices {
		if choice.Words == "The miller is taking a breather." && choice.Line.Source == open.SourceSequence {
			heavy = choice.Line
		}
	}
	if heavy.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "draw-once-more", ActionSpeak, ActionArgs{Target: player, Speech: heavy})); r.Status != "accepted" || c.bakery.Entities[guest].Attention != player || !c.bakery.Entities[guest].Occupied || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[guest].Attention, c.bakery.Entities[guest].Occupied, c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	c.bakery.Entities[page] = SpatialEntity{Name: "page", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, page, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == page
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-page", ActionSpeak, ActionArgs{Target: page, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	open = NotebookEntry{}
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == page {
			open = note
		}
	}
	choices, err = c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The page is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "circle-once-more", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestHeardCircleAfterTheSecondMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheSecondMissIsHeard(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestNextCircleAfterTheSecondMissIsHeard(t *testing.T) {
	nextCircleAfterTheSecondMissIsHeard(t)
}

func nextCircleAfterTheSecondMissIsHeard(t *testing.T) *Coordinator {
	t.Helper()
	c := freshAnswerDrawsAttentionBackAgain(t)
	const baker, guest, player, courier, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/courier", "bakery/clerk"
	if c.bakery.Entities[guest].Occupied || c.bakery.Entities[guest].Attention != "" || c.bakery.Entities[baker].Occupied || c.bakery.Entities[baker].Attention != "" {
		t.Fatal(c.bakery.Entities[guest].Occupied, c.bakery.Entities[guest].Attention, c.bakery.Entities[baker].Occupied, c.bakery.Entities[baker].Attention)
	}
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[courier] = SpatialEntity{Name: "courier", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, courier, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == courier
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-courier", ActionSpeak, ActionArgs{Target: courier, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == courier {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The courier is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "after-the-second-miss", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestDrawnAgainSplitSurvivesReopen(t *testing.T) {
	c := freshAnswerDrawsAttentionBackAgain(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestFreshAnswerDrawsAttentionBackAgain(t *testing.T) {
	freshAnswerDrawsAttentionBackAgain(t)
}

func freshAnswerDrawsAttentionBackAgain(t *testing.T) *Coordinator {
	t.Helper()
	c := nextCircleAfterTheMissIsHeard(t)
	const baker, guest, player, patron, regular, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/patron", "bakery/regular", "bakery/clerk"
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[patron] = SpatialEntity{Name: "patron", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{baker, guest, player, patron} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == patron
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-patron", ActionSpeak, ActionArgs{Target: patron, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == patron {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var heavy SpeechLine
	for _, choice := range choices {
		if choice.Words == "The patron is taking a breather." && choice.Line.Source == open.SourceSequence {
			heavy = choice.Line
		}
	}
	if heavy.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "draw-again", ActionSpeak, ActionArgs{Target: player, Speech: heavy})); r.Status != "accepted" || c.bakery.Entities[guest].Attention != player || !c.bakery.Entities[guest].Occupied || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[guest].Attention, c.bakery.Entities[guest].Occupied, c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	c.bakery.Entities[regular] = SpatialEntity{Name: "regular", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, regular, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == regular
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-regular", ActionSpeak, ActionArgs{Target: regular, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	open = NotebookEntry{}
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == regular {
			open = note
		}
	}
	choices, err = c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The regular is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "circle-again", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestHeardCircleAfterTheMissSurvivesReopen(t *testing.T) {
	c := nextCircleAfterTheMissIsHeard(t)
	const baker, guest = "bakery/baker", "bakery/guest"
	beforeGuest, err := c.Journal(guest)
	beforeBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	afterGuest, err := c.Journal(guest)
	afterBaker, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(afterGuest) == 0 || len(afterBaker) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(afterGuest[len(afterGuest)-1:])
	bakerText := journalSpeech(afterBaker[len(afterBaker)-1:])
	if journalSpeech(beforeGuest) != journalSpeech(afterGuest) || journalSpeech(beforeBaker) != journalSpeech(afterBaker) ||
		!strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") ||
		restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied || restored.Entities[baker].Attention != "" || restored.Entities[baker].Occupied {
		t.Fatal(guestText, bakerText, restored.Entities[guest].Attention, restored.Entities[baker].Attention)
	}
}

func TestNextCircleAfterTheMissIsHeard(t *testing.T) {
	nextCircleAfterTheMissIsHeard(t)
}

func nextCircleAfterTheMissIsHeard(t *testing.T) *Coordinator {
	t.Helper()
	c := freshAnswerDrawsTheBringerBack(t)
	const baker, guest, player, zeal, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/zeal", "bakery/clerk"
	if c.bakery.Entities[guest].Occupied || c.bakery.Entities[guest].Attention != "" || c.bakery.Entities[baker].Occupied || c.bakery.Entities[baker].Attention != "" {
		t.Fatal(c.bakery.Entities[guest].Occupied, c.bakery.Entities[guest].Attention, c.bakery.Entities[baker].Occupied, c.bakery.Entities[baker].Attention)
	}
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[zeal] = SpatialEntity{Name: "zeal", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, zeal, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == zeal
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-zeal", ActionSpeak, ActionArgs{Target: zeal, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == zeal {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The zeal is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "after-the-miss", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestFreshAnswerDrawsTheBringerBack(t *testing.T) {
	freshAnswerDrawsTheBringerBack(t)
}

func freshAnswerDrawsTheBringerBack(t *testing.T) *Coordinator {
	t.Helper()
	c := fadedBringerHearsTheCircle(t)
	const baker, guest, player, visitor, calm, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/visitor", "bakery/calm", "bakery/clerk"
	heard := c.bakery.Entities[guest].Placement
	station := c.bakery.Entities[player].Placement
	c.bakery.Entities[visitor] = SpatialEntity{Name: "visitor", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{baker, guest, player, visitor} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == visitor
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-visitor", ActionSpeak, ActionArgs{Target: visitor, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == visitor {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var heavy SpeechLine
	for _, choice := range choices {
		if choice.Words == "The visitor is taking a breather." && choice.Line.Source == open.SourceSequence {
			heavy = choice.Line
		}
	}
	if heavy.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "draw-back", ActionSpeak, ActionArgs{Target: player, Speech: heavy})); r.Status != "accepted" || c.bakery.Entities[guest].Attention != player || !c.bakery.Entities[guest].Occupied || c.bakery.Entities[baker].Attention != "" || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[guest].Attention, c.bakery.Entities[guest].Occupied, c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	c.bakery.Entities[calm] = SpatialEntity{Name: "calm", Kind: Actor, Placement: station, Seated: true}
	for _, id := range []string{player, calm, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == calm
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-calm", ActionSpeak, ActionArgs{Target: calm, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	open = NotebookEntry{}
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == calm {
			open = note
		}
	}
	choices, err = c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The calm is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "again-later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestPages, err := c.Journal(guest)
	bakerPages, err2 := c.Journal(baker)
	if err != nil || err2 != nil || len(guestPages) == 0 || len(bakerPages) == 0 {
		t.Fatal(err, err2)
	}
	guestText := journalSpeech(guestPages[len(guestPages)-1:])
	bakerText := journalSpeech(bakerPages[len(bakerPages)-1:])
	if !strings.Contains(guestText, "I was following another conversation. I missed what they said.") || strings.Contains(guestText, "I heard the customer speak from the other table.") || !strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func askFadedBringer(c *Coordinator, name string) Result {
	const guest, customer = "bakery/guest", "bakery/customer"
	who := c.bakery.Entities[guest]
	who.Placement = c.bakery.Entities[customer].Placement
	c.bakery.Entities[guest] = who
	return c.Submit(customer, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: guest, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
}

func TestFadedBringerSurvivesReopen(t *testing.T) {
	c := fadedBringerHearsTheCircle(t)
	const guest = "bakery/guest"
	before, err := c.Journal(guest)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	after, err := c.Journal(guest)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(after)
	if journalSpeech(before) != text || !strings.Contains(text, "I heard the customer speak from the other table.") || strings.Contains(text, "I was following another conversation.") || restored.Entities[guest].Attention != "" || restored.Entities[guest].Occupied {
		t.Fatal(text, restored.Entities[guest].Attention, restored.Entities[guest].Occupied)
	}
}

func TestFadedBringerHearsTheCircle(t *testing.T) {
	fadedBringerHearsTheCircle(t)
}

func fadedBringerHearsTheCircle(t *testing.T) *Coordinator {
	t.Helper()
	c := fadedFreshAnswerHearsTheCircle(t)
	const guest = "bakery/guest"
	if c.bakery.Entities[guest].Occupied || c.bakery.Entities[guest].Attention != "" {
		t.Fatal(c.bakery.Entities[guest].Occupied, c.bakery.Entities[guest].Attention)
	}
	pages, err := c.Journal(guest)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if !strings.Contains(text, "I heard the customer speak from the other table.") || strings.Contains(text, "I was following another conversation.") {
		t.Fatal(text)
	}
	return c
}

func TestFadedFreshAnswerReleasesAttention(t *testing.T) {
	fadedFreshAnswerHearsTheCircle(t)
}

func fadedFreshAnswerHearsTheCircle(t *testing.T) *Coordinator {
	t.Helper()
	c, heard, station := attentionDrawnBack(t)
	const baker, player = "bakery/baker", "bakery/player"
	if c.bakery.Entities[baker].Attention == "" || !c.bakery.Entities[baker].Occupied {
		t.Fatal(c.bakery.Entities[baker].Attention, c.bakery.Entities[baker].Occupied)
	}
	for _, name := range []string{"quiet-a", "quiet-b", "quiet-c", "quiet-d"} {
		if !c.bakery.Entities[baker].Occupied {
			break
		}
		for _, id := range []string{baker, player} {
			who := c.bakery.Entities[id]
			who.Placement = station
			who.Seated = false
			c.bakery.Entities[id] = who
		}
		if r := c.Submit(player, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "news", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" || strings.Contains(r.Message, "Doing all right") {
			t.Fatal(r)
		}
	}
	if c.bakery.Entities[baker].Occupied || c.bakery.Entities[baker].Attention != "" {
		t.Fatal(c.bakery.Entities[baker].Occupied, c.bakery.Entities[baker].Attention)
	}
	finishDrawnAside(t, c, heard, station)
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if !strings.Contains(text, "I heard the customer speak from the other table.") || strings.Contains(text, "I was following another conversation.") {
		t.Fatal(text)
	}
	return c
}

func TestFreshHeavyAnswerDrawsAttentionBack(t *testing.T) {
	c, heard, station := attentionDrawnBack(t)
	finishDrawnAside(t, c, heard, station)
	pages, err := c.Journal("bakery/baker")
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.Entities["bakery/baker"].Attention != "" || !strings.Contains(journalSpeech(pages), "I was following another conversation. I missed what they said.") {
		t.Fatal(c.bakery.Entities["bakery/baker"].Attention, journalSpeech(pages))
	}
}

func attentionDrawnBack(t *testing.T) (*Coordinator, Placement, Placement) {
	t.Helper()
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare, clerk = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare", "bakery/clerk"
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "routine", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || c.bakery.Entities[baker].Attention != "" {
		t.Fatal(r, c.bakery.Entities[baker].Attention)
	}
	for _, id := range []string{baker, player, guest} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == player
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(baker, spatialProposal(c, "ask-again", ActionSpeak, ActionArgs{Target: player, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == player {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, baker)
	if err != nil {
		t.Fatal(err)
	}
	var heavy SpeechLine
	for _, choice := range choices {
		if choice.Words == "The player is taking a breather." && choice.Line.Source == open.SourceSequence {
			heavy = choice.Line
		}
	}
	if heavy.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "again", ActionSpeak, ActionArgs{Target: baker, Speech: heavy})); r.Status != "accepted" || c.bakery.Entities[baker].Attention != guest {
		t.Fatal(r, c.bakery.Entities[baker].Attention)
	}
	return c, heard, station
}

func finishDrawnAside(t *testing.T, c *Coordinator, heard, station Placement) {
	t.Helper()
	const baker, guest, player, spare, clerk = "bakery/baker", "bakery/guest", "bakery/player", "bakery/spare", "bakery/clerk"
	c.bakery.Entities[clerk] = SpatialEntity{Name: "clerk", Kind: Actor, Placement: station}
	for _, id := range []string{player, spare, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		person := c.bakery.Entities[id]
		person.Placement = station
		person.Seated = false
		c.bakery.Entities[id] = person
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
}

func TestRoutineAnswerReleasesTheBringer(t *testing.T) {
	releasedBringerHearsTheCircle(t)
}

func releasedBringerHearsTheCircle(t *testing.T) *Coordinator {
	t.Helper()
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare, clerk = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare", "bakery/clerk"
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "routine", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || c.bakery.Entities[guest].Attention != baker {
		t.Fatal(r, c.bakery.Entities[guest].Attention)
	}
	for _, id := range []string{guest, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "guest-routine", ActionSpeak, ActionArgs{Target: guest, Speech: checkInQuestion})); r.Status != "accepted" || !strings.Contains(r.Message, "Doing all right, thank you.") || c.bakery.Entities[guest].Attention != "" || c.bakery.Entities[guest].Occupied {
		t.Fatal(r, c.bakery.Entities[guest].Attention, c.bakery.Entities[guest].Occupied)
	}
	c.bakery.Entities[clerk] = SpatialEntity{Name: "clerk", Kind: Actor, Placement: station}
	for _, id := range []string{player, spare, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, guest} {
		who := c.bakery.Entities[id]
		who.Placement = heard
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	for _, id := range []string{player, clerk} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[clerk] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(clerk, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(clerk, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestJournal, err := c.Journal(guest)
	bakerJournal, err2 := c.Journal(baker)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	guestText, bakerText := journalSpeech(guestJournal), journalSpeech(bakerJournal)
	if !strings.Contains(guestText, "I heard the customer speak from the other table.") || strings.Contains(guestText, "I was following another conversation.") ||
		!strings.Contains(bakerText, "I heard the customer speak from the other table.") || strings.Contains(bakerText, "I was following another conversation.") {
		t.Fatal(guestText, bakerText)
	}
	return c
}

func TestUnrestoredMissSurvivesReopen(t *testing.T) {
	c := weightLiftedAfterTheMiss(t)
	const baker, customer = "bakery/baker", "bakery/customer"
	before, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	after, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(after)
	if journalSpeech(before) != text || !strings.Contains(text, "I missed what they said.") || strings.Contains(text, "other table") || restored.Entities[baker].Occupied {
		t.Fatal(text, restored.Entities[baker].Occupied)
	}
	if r := c.Submit(customer, spatialProposal(c, "reopened", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I missed what you said.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
}

func TestRoutineAskerStillHearsTheCheckIn(t *testing.T) {
	c := weightLiftedAfterTheMiss(t)
	const baker, player = "bakery/baker", "bakery/player"
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[player].Placement
	who.Seated = false
	c.bakery.Entities[baker] = who
	if r := c.Submit(player, spatialProposal(c, "own-check-in", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You checked in on me. I told you I felt fine.") ||
		strings.Contains(r.Message, "I missed what you said.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(journalSpeech(pages))
	}
}

func TestHeardLineIsWhatGetsRepeated(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, customer = "bakery/baker", "bakery/customer"
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[customer].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(customer, spatialProposal(c, "retell", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(journalSpeech(pages))
	}
}

func TestRoutineAskerDoesNotHearTheOtherCircle(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, player = "bakery/baker", "bakery/player"
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[player].Placement
	who.Seated = false
	c.bakery.Entities[baker] = who
	if r := c.Submit(player, spatialProposal(c, "own-check-in", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You checked in on me. I told you I felt fine.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if !strings.Contains(text, "I missed what they said.") || !strings.Contains(text, "I heard the customer speak from the other table.") {
		t.Fatal(text)
	}
}

func TestSplitAccountSurvivesReopen(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, player, customer = "bakery/baker", "bakery/player", "bakery/customer"
	before, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	after, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(after)
	if journalSpeech(before) != text || !strings.Contains(text, "I missed what they said.") || !strings.Contains(text, "I heard the customer speak from the other table.") {
		t.Fatal(text)
	}
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[player].Placement
	who.Seated = false
	c.bakery.Entities[baker] = who
	if r := c.Submit(player, spatialProposal(c, "own-check-in", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You checked in on me. I told you I felt fine.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	who = c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[customer].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(customer, spatialProposal(c, "retell", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "I missed what you said.") || strings.Contains(r.Message, "felt fine") {
		t.Fatal(r)
	}
}

func TestYieldedListenerHearsThatTheyLetThemGoFirst(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, neighbor = "bakery/baker", "bakery/neighbor"
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[neighbor].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(neighbor, spatialProposal(c, "yield", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
		!strings.Contains(r.Message, "The neighbor let them go first.") ||
		strings.Contains(r.Message, "felt fine") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(journalSpeech(pages))
	}
}

func TestYieldSurvivesReopen(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, neighbor = "bakery/baker", "bakery/neighbor"
	before, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	after, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(after)
	if journalSpeech(before) != text || !strings.Contains(text, "I missed what they said.") || !strings.Contains(text, "I heard the customer speak from the other table.") {
		t.Fatal(text)
	}
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[neighbor].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(neighbor, spatialProposal(c, "yield", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
		!strings.Contains(r.Message, "The neighbor let them go first.") ||
		strings.Contains(r.Message, "felt fine") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
}

func TestOutsiderHearsTheYield(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, spare = "bakery/baker", "bakery/spare"
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[spare].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(spare, spatialProposal(c, "outside", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
		!strings.Contains(r.Message, "The neighbor let them go first.") ||
		strings.Contains(r.Message, "felt fine") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(journalSpeech(pages))
	}
}

func TestCheckInReplyLeavesTheOutsiderAccount(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, player, spare = "bakery/baker", "bakery/player", "bakery/spare"
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[player].Placement
	who.Seated = false
	c.bakery.Entities[baker] = who
	if r := c.Submit(player, spatialProposal(c, "own-check-in", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You checked in on me. I told you I felt fine.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	who = c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[spare].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(spare, spatialProposal(c, "outside", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
		!strings.Contains(r.Message, "The neighbor let them go first.") ||
		strings.Contains(r.Message, "felt fine") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(journalSpeech(pages))
	}
}

func TestCheckInThenOutsiderSurvivesReopen(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	const baker, player, spare = "bakery/baker", "bakery/player", "bakery/spare"
	before, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	after, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(after)
	if journalSpeech(before) != text || !strings.Contains(text, "I missed what they said.") || !strings.Contains(text, "I heard the customer speak from the other table.") {
		t.Fatal(text)
	}
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[player].Placement
	who.Seated = false
	c.bakery.Entities[baker] = who
	if r := c.Submit(player, spatialProposal(c, "own-check-in", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "You checked in on me. I told you I felt fine.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	who = c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[spare].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(spare, spatialProposal(c, "outside", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard the customer speak from the other table.") ||
		!strings.Contains(r.Message, "The neighbor let them go first.") ||
		strings.Contains(r.Message, "felt fine") || strings.Contains(r.Message, "I missed what you said.") {
		t.Fatal(r)
	}
}

func TestNextWordsGetThroughAfterTheLift(t *testing.T) {
	c := nextWordsAfterTheLift(t)
	pages, err := c.Journal("bakery/baker")
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if !strings.Contains(text, "I missed what they said.") || !strings.Contains(text, "I heard the customer speak from the other table.") {
		t.Fatal(text)
	}
}

func nextWordsAfterTheLift(t *testing.T) *Coordinator {
	t.Helper()
	c := weightLiftedAfterTheMiss(t)
	const baker, player, guest, spare, customer, neighbor = "bakery/baker", "bakery/player", "bakery/guest", "bakery/spare", "bakery/customer", "bakery/neighbor"
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-again", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	hide := c.bakery.Entities[customer].Placement
	found := false
	for y := 0; y < bakeryHeight && !found; y++ {
		for x := 0; x < bakeryWidth; x++ {
			heard := hide
			heard.Cell.X, heard.Cell.Y = x, y
			if heard == hide || !c.bakery.isFloorCell(heard.Cell) {
				continue
			}
			who := c.bakery.Entities[baker]
			who.Placement, who.Seated = heard, false
			c.bakery.Entities[baker] = who
			view, err := c.bakery.Observe(c.scope, baker)
			if err == nil && view.hasTarget(customer) && view.hasTarget(neighbor) && view.hasTarget(player) && view.hasTarget(guest) &&
				c.bakery.soundReaches(hide.Cell, heard.Cell, conversationReach) {
				found = true
				break
			}
		}
	}
	if !found || c.bakery.Entities[baker].Occupied {
		t.Fatal(found, c.bakery.Entities[baker].Occupied)
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." && choice.Line.Source == open.SourceSequence {
			answer = choice.Line
		}
	}
	if answer.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "next", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(pages)
	if !strings.Contains(text, "I missed what they said.") || !strings.Contains(text, "I heard the customer speak from the other table.") {
		t.Fatal(text)
	}
	return c
}

func TestLiftedWeightLeavesTheMiss(t *testing.T) {
	c := weightLiftedAfterTheMiss(t)
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func weightLiftedAfterTheMiss(t *testing.T) *Coordinator {
	t.Helper()
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare, customer = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare", "bakery/customer"
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	for _, id := range []string{player, guest} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	missed, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(missed), "I missed what they said.") || strings.Contains(journalSpeech(missed), "other table") {
		t.Fatal(journalSpeech(missed))
	}
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "routine", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || !strings.Contains(r.Message, "Doing all right, thank you.") || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[baker].Occupied)
	}
	who := c.bakery.Entities[baker]
	who.Placement = c.bakery.Entities[customer].Placement
	c.bakery.Entities[baker] = who
	if r := c.Submit(customer, spatialProposal(c, "again", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I missed what you said.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	return c
}

func TestRoutineAnswerLiftsTheDeliveryWeight(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if !c.bakery.Entities[baker].Occupied {
		t.Fatal("expected the delivery time to linger")
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "routine", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || !strings.Contains(r.Message, "Doing all right, thank you.") || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[baker].Occupied)
	}
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	porterAt := c.bakery.Entities[porter]
	porterAt.Placement = heard
	c.bakery.Entities[porter] = porterAt
	for _, id := range []string{player, guest} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.Entities[baker].Occupied || !strings.Contains(journalSpeech(pages), "I heard the customer speak from the other table.") || strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(c.bakery.Entities[baker].Occupied, journalSpeech(pages))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestResidentAnswersBothCircles(t *testing.T) {
	c := residentBetween(t, false, false, false)
	const baker, porter, customer = "bakery/baker", "bakery/porter", "bakery/customer"
	var said []string
	for _, e := range c.History() {
		if e.ActorID == porter && e.Payload.Dialogue.Statement.Intent == "both" {
			said = append(said, e.Payload.Target+" "+e.Payload.Dialogue.Words)
		}
	}
	if len(said) != 2 || !strings.Contains(said[0], baker) || !strings.Contains(said[0], "Two trays.") ||
		!strings.Contains(said[1], customer) || !strings.Contains(said[1], "At noon.") || strings.Contains(said[0], "At noon.") || strings.Contains(said[1], "Two trays.") {
		t.Fatal(said)
	}
	ask := func(who, id string) Result {
		at := c.bakery.Entities[who].Placement
		listener := c.bakery.Entities[porter]
		listener.Placement = at
		c.bakery.Entities[porter] = listener
		return c.Submit(who, spatialProposal(c, id, ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "check-in"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the baker ask the player how they were holding up.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if r := ask(customer, "greeting"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the customer speak from the other table.") || strings.Contains(r.Message, "holding up") {
		t.Fatal(r)
	}
	again := 0
	for _, e := range c.History() {
		if e.ActorID == porter && e.Payload.Dialogue.Statement.Intent == "both" {
			again++
		}
	}
	if again != 2 {
		t.Fatal(again)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestResidentAsksToRepeatOverlap(t *testing.T) {
	c := residentBetween(t, true, false, false)
	const baker, porter, customer = "bakery/baker", "bakery/porter", "bakery/customer"
	var said []string
	for _, e := range c.History() {
		if e.ActorID == porter && e.Payload.Dialogue.Statement.Intent == "both" {
			said = append(said, e.Payload.Target+" "+e.Payload.Dialogue.Words)
		}
	}
	if len(said) != 2 || !strings.Contains(said[0], baker) || !strings.Contains(said[0], "Two trays.") ||
		!strings.Contains(said[1], customer) || !strings.Contains(said[1], "Could you say that again? I missed what you said.") ||
		strings.Contains(said[1], "other table") || strings.Contains(said[1], "Hello from") || strings.Contains(said[1], "At noon.") {
		t.Fatal(said)
	}
	ask := func(who, id string) Result {
		at := c.bakery.Entities[who].Placement
		listener := c.bakery.Entities[porter]
		listener.Placement = at
		c.bakery.Entities[porter] = listener
		return c.Submit(who, spatialProposal(c, id, ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}}))
	}
	if r := ask(baker, "check-in"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard the baker ask the player how they were holding up.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if r := ask(customer, "again"); r.Status != "accepted" || !strings.Contains(r.Message, "Could you say that again? I missed what you said.") ||
		strings.Contains(r.Message, "other table") || strings.Contains(r.Message, "Hello from") {
		t.Fatal(r)
	}
	again := 0
	for _, e := range c.History() {
		if e.ActorID == porter && e.Payload.Dialogue.Statement.Intent == "both" {
			again++
		}
	}
	if again != 2 {
		t.Fatal(again)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyAnswerMissesTheOtherCircle(t *testing.T) {
	c := residentBetween(t, false, true, false)
	const baker, porter, customer, calm = "bakery/baker", "bakery/porter", "bakery/customer", "bakery/calm"
	var said []string
	for _, e := range c.History() {
		if e.ActorID == porter && e.Payload.Dialogue.Statement.Intent == "both" {
			said = append(said, e.Payload.Target+" "+e.Payload.Dialogue.Words)
		}
	}
	if len(said) != 2 || !strings.Contains(said[0], baker) || !strings.Contains(said[0], "Two trays.") ||
		!strings.Contains(said[1], customer) || !strings.Contains(said[1], "Could you say that again? I missed what you said.") ||
		strings.Contains(said[1], "other table") || strings.Contains(said[1], "At noon.") {
		t.Fatal(said)
	}
	calmJournal, err := c.Journal(calm)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(calmJournal), "I heard the customer speak from the other table.") {
		t.Fatal(journalSpeech(calmJournal))
	}
	at := c.bakery.Entities[customer].Placement
	who := c.bakery.Entities[porter]
	who.Placement = at
	c.bakery.Entities[porter] = who
	if r := c.Submit(customer, spatialProposal(c, "again", ActionSpeak, ActionArgs{Target: porter, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "Could you say that again? I missed what you said.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyAnswerEases(t *testing.T) {
	c := residentBetween(t, false, true, true)
	const baker, porter, spare = "bakery/baker", "bakery/porter", "bakery/spare"
	if c.bakery.Entities[porter].Occupied {
		t.Fatal("still occupied")
	}
	pages, err := c.Journal(porter)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I missed what they said.") || strings.Contains(journalSpeech(pages), "other table") {
		t.Fatal(journalSpeech(pages))
	}
	station := c.bakery.Entities["bakery/player"].Placement
	for _, id := range []string{baker, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[spare] {
		if note.Overheard && note.Open && note.Target == "bakery/player" {
			open = note
		}
	}
	choices, err := c.DialogueChoices(spare, baker)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The player is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(spare, spatialProposal(c, "ease", ActionSpeak, ActionArgs{Target: baker, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	after, err := c.Journal(porter)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.Entities[porter].Occupied || !strings.Contains(journalSpeech(after), "I heard the customer speak from the other table.") || !strings.Contains(journalSpeech(after), "I missed what they said.") {
		t.Fatal(journalSpeech(after))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyAnswerOccupiesTheSpeaker(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, guest, spare = "bakery/baker", "bakery/porter", "bakery/guest", "bakery/spare"
	if !c.bakery.Entities[guest].Occupied || c.bakery.Entities[porter].Occupied {
		t.Fatal(c.bakery.Entities[guest].Occupied, c.bakery.Entities[porter].Occupied)
	}
	who := c.bakery.Entities[guest]
	who.Placement = c.bakery.Entities[porter].Placement
	c.bakery.Entities[guest] = who
	station := c.bakery.Entities["bakery/player"].Placement
	for _, id := range []string{baker, spare} {
		at := c.bakery.Entities[id]
		at.Placement = station
		c.bakery.Entities[id] = at
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[spare] {
		if note.Overheard && note.Open && note.Target == "bakery/player" {
			open = note
		}
	}
	choices, err := c.DialogueChoices(spare, baker)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The player is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(spare, spatialProposal(c, "occupy", ActionSpeak, ActionArgs{Target: baker, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	guestJournal, err := c.Journal(guest)
	porterJournal, err2 := c.Journal(porter)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if c.bakery.Entities[guest].Occupied || !strings.Contains(journalSpeech(guestJournal), "I missed what they said.") || strings.Contains(journalSpeech(guestJournal), "other table") ||
		!strings.Contains(journalSpeech(porterJournal), "I heard the customer speak from the other table.") {
		t.Fatal(journalSpeech(guestJournal), journalSpeech(porterJournal))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyNewsOccupiesTheListener(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if !c.bakery.Entities[baker].Occupied || c.bakery.Entities[porter].Occupied {
		t.Fatal(c.bakery.Entities[baker].Occupied, c.bakery.Entities[porter].Occupied)
	}
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		if id == spare {
			who.Seated = true
		}
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	heard := c.bakery.Entities[porter].Placement
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		if id == spare {
			who.Seated = true
		}
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "told", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	bakerJournal, err := c.Journal(baker)
	porterJournal, err2 := c.Journal(porter)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if c.bakery.Entities[baker].Occupied || !strings.Contains(journalSpeech(bakerJournal), "I missed what they said.") || strings.Contains(journalSpeech(bakerJournal), "other table") ||
		!strings.Contains(journalSpeech(porterJournal), "I heard the customer speak from the other table.") {
		t.Fatal(journalSpeech(bakerJournal), journalSpeech(porterJournal))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestRoutineAnswerLeavesTheCircleOpen(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if !c.bakery.Entities[baker].Occupied {
		t.Fatal("expected the heavy news to linger")
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "routine", ActionSpeak, ActionArgs{Target: baker, Speech: checkInQuestion})); r.Status != "accepted" || !strings.Contains(r.Message, "Doing all right, thank you.") || c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[baker].Occupied)
	}
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	porterAt := c.bakery.Entities[porter]
	porterAt.Placement = heard
	c.bakery.Entities[porter] = porterAt
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.Entities[baker].Occupied || !strings.Contains(journalSpeech(pages), "I heard the customer speak from the other table.") || strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(c.bakery.Entities[baker].Occupied, journalSpeech(pages))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyFeelingOutlastsTheSubject(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if !c.bakery.Entities[baker].Occupied {
		t.Fatal("expected the heavy news to linger")
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, id := range []string{baker, player} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "other-subject", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "news", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" || strings.Contains(r.Message, "Doing all right") || !c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[baker].Occupied)
	}
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	porterAt := c.bakery.Entities[porter]
	porterAt.Placement = heard
	c.bakery.Entities[porter] = porterAt
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	bakerJournal, err := c.Journal(baker)
	porterJournal, err2 := c.Journal(porter)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if !strings.Contains(journalSpeech(bakerJournal), "I missed what they said.") || strings.Contains(journalSpeech(bakerJournal), "other table") ||
		!strings.Contains(journalSpeech(porterJournal), "I heard the customer speak from the other table.") {
		t.Fatal(journalSpeech(bakerJournal), journalSpeech(porterJournal))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyFeelingFades(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if !c.bakery.Entities[baker].Occupied {
		t.Fatal("expected the heavy news to linger")
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	for _, name := range []string{"quiet-a", "quiet-b", "quiet-c", "quiet-d"} {
		if !c.bakery.Entities[baker].Occupied {
			break
		}
		for _, id := range []string{baker, player} {
			who := c.bakery.Entities[id]
			who.Placement = station
			who.Seated = false
			c.bakery.Entities[id] = who
		}
		if r := c.Submit(player, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "news", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	if c.bakery.Entities[baker].Occupied {
		t.Fatal("still occupied")
	}
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	porterAt := c.bakery.Entities[porter]
	porterAt.Placement = heard
	c.bakery.Entities[porter] = porterAt
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	pages, err := c.Journal(baker)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.Entities[baker].Occupied || !strings.Contains(journalSpeech(pages), "I heard the customer speak from the other table.") || strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(c.bakery.Entities[baker].Occupied, journalSpeech(pages))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyAnswerReinforcesTheFeeling(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, porter, player, guest, spare = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/spare"
	if !c.bakery.Entities[baker].Occupied {
		t.Fatal("expected the heavy news to linger")
	}
	heard := c.bakery.Entities[porter].Placement
	station := c.bakery.Entities[player].Placement
	news := func(name string) {
		t.Helper()
		for _, id := range []string{baker, player} {
			who := c.bakery.Entities[id]
			who.Placement = station
			who.Seated = false
			c.bakery.Entities[id] = who
		}
		if r := c.Submit(player, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "news", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" || strings.Contains(r.Message, "Doing all right") {
			t.Fatal(r)
		}
	}
	news("before")
	if !c.bakery.Entities[baker].Occupied {
		t.Fatal("faded before the fresh answer")
	}
	for _, id := range []string{player, guest, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = id == spare
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(player, spatialProposal(c, "ask-spare", ActionSpeak, ActionArgs{Target: spare, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	choices, err := c.DialogueChoices(spare, baker)
	if err != nil {
		t.Fatal(err)
	}
	var reinforce SpeechLine
	for _, choice := range choices {
		if choice.Words == "The player is taking a breather." {
			reinforce = choice.Line
		}
	}
	if reinforce.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(spare, spatialProposal(c, "reinforce", ActionSpeak, ActionArgs{Target: baker, Speech: reinforce})); r.Status != "accepted" || !c.bakery.Entities[baker].Occupied {
		t.Fatal(r, c.bakery.Entities[baker].Occupied)
	}
	news("after-a")
	news("after-b")
	if !c.bakery.Entities[baker].Occupied {
		t.Fatal("faded despite the fresh answer")
	}
	bakerAt := c.bakery.Entities[baker]
	bakerAt.Placement = heard
	c.bakery.Entities[baker] = bakerAt
	porterAt := c.bakery.Entities[porter]
	porterAt.Placement = heard
	c.bakery.Entities[porter] = porterAt
	for _, id := range []string{player, guest} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == spare {
			open = note
		}
	}
	again, err := c.DialogueChoices(guest, player)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range again {
		if choice.Words == "The spare is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source == 0 || answer.Source != open.SourceSequence {
		t.Fatal(again)
	}
	if r := c.Submit(guest, spatialProposal(c, "later", ActionSpeak, ActionArgs{Target: player, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	bakerJournal, err := c.Journal(baker)
	porterJournal, err2 := c.Journal(porter)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if !strings.Contains(journalSpeech(bakerJournal), "I missed what they said.") || strings.Contains(journalSpeech(bakerJournal), "other table") ||
		!strings.Contains(journalSpeech(porterJournal), "I heard the customer speak from the other table.") {
		t.Fatal(journalSpeech(bakerJournal), journalSpeech(porterJournal))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestHeavyNewsStaysWithTheAsker(t *testing.T) {
	c := residentBetween(t, false, false, true)
	const baker, player, spare = "bakery/baker", "bakery/player", "bakery/spare"
	heardNews := false
	for _, note := range c.bakery.Notebooks[player] {
		if note.Heard && note.Condition == resting {
			heardNews = true
		}
	}
	if !heardNews || c.bakery.Entities[player].Occupied || !c.bakery.Entities[baker].Occupied {
		t.Fatal(heardNews, c.bakery.Entities[player].Occupied, c.bakery.Entities[baker].Occupied)
	}
	heard := c.bakery.Entities["bakery/porter"].Placement
	station := c.bakery.Entities[player].Placement
	playerAt := c.bakery.Entities[player]
	playerAt.Placement = heard
	playerAt.Seated = false
	c.bakery.Entities[player] = playerAt
	for _, id := range []string{baker, spare} {
		who := c.bakery.Entities[id]
		who.Placement = station
		who.Seated = false
		c.bakery.Entities[id] = who
	}
	choices, err := c.DialogueChoices(spare, baker)
	if err != nil {
		t.Fatal(err)
	}
	var told SpeechLine
	for _, choice := range choices {
		if choice.Words == "The player is taking a breather." {
			told = choice.Line
		}
	}
	if told.Source == 0 {
		t.Fatal(choices)
	}
	if r := c.Submit(spare, spatialProposal(c, "again", ActionSpeak, ActionArgs{Target: baker, Speech: told})); r.Status != "accepted" {
		t.Fatal(r)
	}
	pages, err := c.Journal(player)
	if err != nil {
		t.Fatal(err)
	}
	if c.bakery.Entities[player].Occupied || !c.bakery.Entities[baker].Occupied || !strings.Contains(journalSpeech(pages), "I heard the customer speak from the other table.") || strings.Contains(journalSpeech(pages), "I missed what they said.") {
		t.Fatal(c.bakery.Entities[player].Occupied, c.bakery.Entities[baker].Occupied, journalSpeech(pages))
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func residentBetween(t *testing.T, overlap, weighted, spare bool) *Coordinator {
	t.Helper()
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	const baker, porter, player, guest, customer = "bakery/baker", "bakery/porter", "bakery/player", "bakery/guest", "bakery/customer"
	c.bakery.ContentVersion = answerFirstVersion
	spot := c.bakery.Entities[baker].Placement
	for _, id := range []string{porter, player} {
		actor := c.bakery.Entities[id]
		actor.Placement = spot
		c.bakery.Entities[id] = actor
	}
	seated := c.bakery.Entities[player]
	seated.Seated = true
	c.bakery.Entities[player] = seated
	c.bakery.Entities[guest] = SpatialEntity{Name: "guest", Kind: Actor, Placement: spot}
	if spare {
		c.bakery.Entities["bakery/spare"] = SpatialEntity{Name: "spare", Kind: Actor, Placement: spot}
	}
	if r := c.Submit(baker, spatialProposal(c, "ask", ActionSpeak, ActionArgs{Target: player, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, player, guest} {
		actor := c.bakery.Entities[id]
		actor.Placement = spot
		if id == player {
			actor.Seated = true
		}
		c.bakery.Entities[id] = actor
	}
	doorY := c.bakery.Entities["bakery/door"].Placement.Cell.Y
	hidden := false
	var hide Placement
	for y := 0; y < bakeryHeight; y++ {
		if y == doorY {
			continue
		}
		hide = c.bakery.Entities["bakery/door"].Placement
		hide.Cell.X, hide.Cell.Y = 0, y
		away := c.bakery.Entities[porter]
		away.Placement, away.Seated = hide, false
		c.bakery.Entities[porter] = away
		view, err := c.bakery.Observe(scope, porter)
		if err == nil && !view.hasTarget(baker) && !view.hasTarget(player) {
			hidden = true
			break
		}
	}
	if !hidden {
		t.Fatal("no cell hides the other conversation")
	}
	c.bakery.Entities[customer] = SpatialEntity{Name: "customer", Kind: Actor, Placement: hide}
	const neighbor = "bakery/neighbor"
	c.bakery.Entities[neighbor] = SpatialEntity{Name: "neighbor", Kind: Actor, Placement: hide, Seated: overlap}
	found := false
	for y := 0; y < bakeryHeight && !found; y++ {
		for x := 0; x < bakeryWidth; x++ {
			at := hide
			at.Cell.X, at.Cell.Y = x, y
			if at == hide || !c.bakery.isFloorCell(at.Cell) {
				continue
			}
			who := c.bakery.Entities[porter]
			who.Placement, who.Seated, who.Fatigue = at, false, 0
			c.bakery.Entities[porter] = who
			view, err := c.bakery.Observe(scope, porter)
			if err == nil && view.hasTarget(baker) && view.hasTarget(player) && view.hasTarget(customer) && view.hasTarget(neighbor) &&
				c.bakery.soundReaches(hide.Cell, at.Cell, conversationReach) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("no cell sees both circles")
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[guest] {
		if note.Overheard && note.Open && note.Target == player {
			open = note
		}
	}
	choices, err := c.DialogueChoices(guest, baker)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The player is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if weighted {
		who := c.bakery.Entities[porter]
		who.Occupied = true
		c.bakery.Entities[porter] = who
		c.bakery.Entities["bakery/calm"] = SpatialEntity{Name: "calm", Kind: Actor, Placement: who.Placement}
	}
	if r := c.Submit(guest, spatialProposal(c, "answer", ActionSpeak, ActionArgs{Target: baker, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	return c
}

func TestSpeechBeyondEarshotStaysMissing(t *testing.T) {
	c := separateCircleFar(t, false, false, true, false, false, false)
	const porter, customer, zeal = "bakery/porter", "bakery/customer", "bakery/zeal"
	nearJournal, err := c.Journal(porter)
	farJournal, err2 := c.Journal(zeal)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if !strings.Contains(journalSpeech(nearJournal), "I heard the customer speak from the other table.") || strings.Contains(journalSpeech(farJournal), "other table") {
		t.Fatal(journalSpeech(nearJournal), journalSpeech(farJournal))
	}
	at := c.bakery.Entities[customer].Placement
	who := c.bakery.Entities[zeal]
	who.Placement = at
	c.bakery.Entities[zeal] = who
	if r := c.Submit(customer, spatialProposal(c, "closer", ActionSpeak, ActionArgs{Target: zeal, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" || strings.Contains(r.Message, "other table") || strings.Contains(r.Message, "customer speak") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestMissedLineSurvivesReopen(t *testing.T) {
	c := separateCircleFar(t, false, false, false, true, false, false)
	const zeal, baker = "bakery/zeal", "bakery/baker"
	before, err := c.Journal(zeal)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	after, err := c.Journal(zeal)
	if err != nil {
		t.Fatal(err)
	}
	text := journalSpeech(after)
	if journalSpeech(before) != text || !strings.Contains(text, "I was following another conversation. I missed what they said.") || strings.Contains(text, "other table") {
		t.Fatal(journalSpeech(before), text)
	}
	if restored.Entities[zeal].Attention != baker {
		t.Fatal(restored.Entities[zeal].Attention)
	}
}

func separateCircle(t *testing.T, neighborSeated, between bool) *Coordinator {
	t.Helper()
	return separateCircleFar(t, neighborSeated, between, false, false, false, false)
}

func TestAttentionSwitchLeavesTheMiss(t *testing.T) {
	c := separateCircleFar(t, false, false, false, true, false, false)
	const porter, customer, zeal = "bakery/porter", "bakery/customer", "bakery/zeal"
	nearJournal, err := c.Journal(porter)
	awayJournal, err2 := c.Journal(zeal)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if !strings.Contains(journalSpeech(nearJournal), "I heard the customer speak from the other table.") ||
		!strings.Contains(journalSpeech(awayJournal), "I was following another conversation. I missed what they said.") ||
		strings.Contains(journalSpeech(awayJournal), "other table") {
		t.Fatal(journalSpeech(nearJournal), journalSpeech(awayJournal))
	}
	who := c.bakery.Entities[zeal]
	who.Attention = customer
	who.Placement = c.bakery.Entities[customer].Placement
	c.bakery.Entities[zeal] = who
	if r := c.Submit(customer, spatialProposal(c, "switch", ActionSpeak, ActionArgs{Target: zeal, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I missed what you said.") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestFragmentHidesTheSpeaker(t *testing.T) {
	c := separateCircleFar(t, false, false, false, false, true, false)
	const baker, visitor = "bakery/baker", "bakery/visitor"
	pages, err := c.Journal(visitor)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I heard “after closing.” I don't know who spoke.") || strings.Contains(journalSpeech(pages), "customer") || strings.Contains(journalSpeech(pages), "other table") {
		t.Fatal(journalSpeech(pages))
	}
	who := c.bakery.Entities[visitor]
	who.Placement = c.bakery.Entities[baker].Placement
	c.bakery.Entities[visitor] = who
	if r := c.Submit(baker, spatialProposal(c, "fragment", ActionSpeak, ActionArgs{Target: visitor, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard “after closing.” I don't know who spoke.") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestMurmurIsNotWords(t *testing.T) {
	c := separateCircleFar(t, false, false, false, false, false, true)
	const baker, listener = "bakery/baker", "bakery/murmur"
	pages, err := c.Journal(listener)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I heard a murmur.") || strings.Contains(journalSpeech(pages), "after closing") || strings.Contains(journalSpeech(pages), "customer") {
		t.Fatal(journalSpeech(pages))
	}
	who := c.bakery.Entities[listener]
	who.Placement = c.bakery.Entities[baker].Placement
	c.bakery.Entities[listener] = who
	if r := c.Submit(baker, spatialProposal(c, "murmur", ActionSpeak, ActionArgs{Target: listener, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		strings.Contains(r.Message, "after closing") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestFragmentRetellingKeepsTheSpeakerUnknown(t *testing.T) {
	c := separateCircleFar(t, false, false, false, false, true, false)
	const visitor, clerk = "bakery/visitor", "bakery/clerk"
	c.bakery.Entities[clerk] = SpatialEntity{Name: "clerk", Kind: Actor, Placement: c.bakery.Entities[visitor].Placement}
	before, err := c.Journal(clerk)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(journalSpeech(before), "after closing") || strings.Contains(journalSpeech(before), "customer") {
		t.Fatal(journalSpeech(before))
	}
	ask := func(name, from, to, topic string) Result {
		t.Helper()
		return c.Submit(from, spatialProposal(c, name, ActionSpeak, ActionArgs{Target: to, Speech: SpeechLine{Topic: topic, Intent: "ask", Tone: "curious"}}))
	}
	if r := ask("news", visitor, clerk, "news"); r.Status != "accepted" || strings.Contains(r.Message, "after closing") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if r := ask("hearsay", visitor, clerk, "hearsay"); r.Status != "accepted" || strings.Contains(r.Message, "after closing") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if r := ask("retell", clerk, visitor, "conversation"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard “after closing.” I don't know who spoke.") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	pages, err := c.Journal(clerk)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(journalSpeech(pages), "I heard “after closing.” I don't know who spoke.") || strings.Contains(journalSpeech(pages), "customer") || strings.Contains(journalSpeech(pages), "other table") {
		t.Fatal(journalSpeech(pages))
	}
	if r := ask("again", visitor, clerk, "conversation"); r.Status != "accepted" || !strings.Contains(r.Message, "I heard “after closing.” I don't know who spoke.") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestResidentKeepsTheCircleTheyJoined(t *testing.T) {
	c := separateCircleFar(t, false, false, false, false, true, false)
	const baker, porter, clerk = "bakery/baker", "bakery/porter", "bakery/clerk"
	spot := c.bakery.Entities[baker].Placement
	for _, id := range []string{porter, clerk} {
		who := c.bakery.Entities[id]
		if id == clerk {
			who = SpatialEntity{Name: "clerk", Kind: Actor}
		}
		who.Placement = spot
		c.bakery.Entities[id] = who
	}
	if r := c.Submit(porter, spatialProposal(c, "joined", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "The player said you were taking a breather.") || strings.Contains(r.Message, "after closing") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if r := c.Submit(clerk, spatialProposal(c, "outside", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard “after closing.” I don't know who spoke.") || strings.Contains(r.Message, "checked in") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
	if err := c.bakery.validateNotebooks(c.clock.Tick()); err != nil {
		t.Fatal(err)
	}
}

func TestFragmentSurvivesReopen(t *testing.T) {
	c := separateCircleFar(t, false, false, false, false, true, false)
	const visitor, clerk = "bakery/visitor", "bakery/clerk"
	c.bakery.Entities[clerk] = SpatialEntity{Name: "clerk", Kind: Actor, Placement: c.bakery.Entities[visitor].Placement}
	if r := c.Submit(clerk, spatialProposal(c, "retell", ActionSpeak, ActionArgs{Target: visitor, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard “after closing.” I don't know who spoke.") || strings.Contains(r.Message, "customer") {
		t.Fatal(r)
	}
	heard, err := c.Journal(visitor)
	passed, err2 := c.Journal(clerk)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	raw, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreBakery(raw, c.scope.WorldID, c.clock)
	if err != nil {
		t.Fatal(err)
	}
	again, err := restored.Snapshot()
	if err != nil || !bytes.Equal(raw, again) {
		t.Fatal(err)
	}
	c.bakery = restored
	heardAgain, err := c.Journal(visitor)
	passedAgain, err2 := c.Journal(clerk)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	heardText, passedText := journalSpeech(heardAgain), journalSpeech(passedAgain)
	if journalSpeech(heard) != heardText || journalSpeech(passed) != passedText ||
		!strings.Contains(heardText, "I heard “after closing.” I don't know who spoke.") ||
		!strings.Contains(passedText, "I heard “after closing.” I don't know who spoke.") ||
		strings.Contains(heardText, "customer") || strings.Contains(passedText, "customer") ||
		strings.Contains(heardText, "other table") || strings.Contains(passedText, "other table") {
		t.Fatal(heardText, passedText)
	}
	if r := c.Submit(visitor, spatialProposal(c, "again", ActionSpeak, ActionArgs{Target: clerk, Speech: SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}})); r.Status != "accepted" ||
		!strings.Contains(r.Message, "I heard “after closing.” I don't know who spoke.") || strings.Contains(r.Message, "customer") || strings.Contains(r.Message, "other table") {
		t.Fatal(r)
	}
}

func separateCircleFar(t *testing.T, neighborSeated, between, far, attend, fragment, murmur bool) *Coordinator {
	t.Helper()
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Practice: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	const baker, porter, player, customer = "bakery/baker", "bakery/porter", "bakery/player", "bakery/customer"
	c.bakery.ContentVersion = answerFirstVersion
	spot := c.bakery.Entities[baker].Placement
	for _, id := range []string{porter, player} {
		actor := c.bakery.Entities[id]
		actor.Placement = spot
		c.bakery.Entities[id] = actor
	}
	seated := c.bakery.Entities[porter]
	seated.Seated = true
	c.bakery.Entities[porter] = seated
	if between {
		c.bakery.Entities["bakery/guest"] = SpatialEntity{Name: "guest", Kind: Actor, Placement: spot}
	}
	if r := c.Submit(baker, spatialProposal(c, "ask", ActionSpeak, ActionArgs{Target: porter, Speech: checkInQuestion})); r.Status != "accepted" || strings.Contains(r.Message, "replies") {
		t.Fatal(r)
	}
	for _, id := range []string{baker, player} {
		actor := c.bakery.Entities[id]
		actor.Placement = spot
		c.bakery.Entities[id] = actor
	}
	doorY := c.bakery.Entities["bakery/door"].Placement.Cell.Y
	hidden := false
	var hide Placement
	for y := 0; y < bakeryHeight; y++ {
		if y == doorY {
			continue
		}
		hide = c.bakery.Entities["bakery/door"].Placement
		hide.Cell.X, hide.Cell.Y = 0, y
		away := c.bakery.Entities[porter]
		away.Placement, away.Seated = hide, false
		c.bakery.Entities[porter] = away
		view, err := c.bakery.Observe(scope, porter)
		if err == nil && !view.hasTarget(baker) && !view.hasTarget(player) {
			hidden = true
			break
		}
	}
	if !hidden {
		t.Fatal("no cell hides the other conversation")
	}
	c.bakery.Entities[customer] = SpatialEntity{Name: "customer", Kind: Actor, Placement: hide}
	const neighbor = "bakery/neighbor"
	c.bakery.Entities[neighbor] = SpatialEntity{Name: "neighbor", Kind: Actor, Placement: hide, Seated: neighborSeated}
	if between {
		const guest = "bakery/guest"
		found := false
		var at Placement
		for y := 0; y < bakeryHeight && !found; y++ {
			for x := 0; x < bakeryWidth; x++ {
				at = hide
				at.Cell.X, at.Cell.Y = x, y
				if at == hide || !c.bakery.isFloorCell(at.Cell) {
					continue
				}
				who := c.bakery.Entities[guest]
				who.Placement = at
				c.bakery.Entities[guest] = who
				view, err := c.bakery.Observe(scope, guest)
				if err == nil && view.hasTarget(baker) && view.hasTarget(player) && view.hasTarget(customer) && view.hasTarget(neighbor) &&
					c.bakery.soundReaches(hide.Cell, at.Cell, conversationReach) {
					found = true
					break
				}
			}
		}
		if !found {
			t.Fatal("no cell sees both circles")
		}
	}
	if far {
		const zeal = "bakery/zeal"
		c.bakery.Entities[zeal] = SpatialEntity{Name: "zeal", Kind: Actor, Placement: hide}
		found := false
		for y := 0; y < bakeryHeight && !found; y++ {
			for x := 0; x < bakeryWidth; x++ {
				at := hide
				at.Cell.X, at.Cell.Y = x, y
				if at == hide || !c.bakery.isFloorCell(at.Cell) {
					continue
				}
				who := c.bakery.Entities[zeal]
				who.Placement = at
				c.bakery.Entities[zeal] = who
				view, err := c.bakery.Observe(scope, zeal)
				if err != nil || !view.hasTarget(customer) || !view.hasTarget(neighbor) || c.bakery.soundReaches(hide.Cell, at.Cell, conversationReach+1) {
					continue
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatal("no cell sees the other table from beyond earshot")
		}
	}
	if attend {
		const zeal = "bakery/zeal"
		c.bakery.Entities[zeal] = SpatialEntity{Name: "zeal", Kind: Actor, Placement: hide, Attention: baker}
		found := false
		for y := 0; y < bakeryHeight && !found; y++ {
			for x := 0; x < bakeryWidth; x++ {
				at := hide
				at.Cell.X, at.Cell.Y = x, y
				if at == hide || !c.bakery.isFloorCell(at.Cell) {
					continue
				}
				who := c.bakery.Entities[zeal]
				who.Placement = at
				c.bakery.Entities[zeal] = who
				view, err := c.bakery.Observe(scope, zeal)
				if err != nil || !view.hasTarget(customer) || !view.hasTarget(neighbor) || !c.bakery.soundReaches(hide.Cell, at.Cell, conversationReach) {
					continue
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatal("no cell hears the other table")
		}
	}
	if fragment {
		const visitor = "bakery/visitor"
		c.bakery.Entities[visitor] = SpatialEntity{Name: "visitor", Kind: Actor, Placement: hide}
		found := false
		for y := 0; y < bakeryHeight && !found; y++ {
			for x := 0; x < bakeryWidth; x++ {
				at := hide
				at.Cell.X, at.Cell.Y = x, y
				if at == hide || !c.bakery.isFloorCell(at.Cell) {
					continue
				}
				who := c.bakery.Entities[visitor]
				who.Placement = at
				c.bakery.Entities[visitor] = who
				view, err := c.bakery.Observe(scope, visitor)
				if err != nil || view.hasTarget(customer) || !c.bakery.soundReaches(hide.Cell, at.Cell, fragmentReach) {
					continue
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatal("no cell hears the speaker without seeing them")
		}
	}
	if murmur {
		const listener = "bakery/murmur"
		c.bakery.Entities[listener] = SpatialEntity{Name: "murmur", Kind: Actor, Placement: hide}
		found := false
		for y := 0; y < bakeryHeight && !found; y++ {
			for x := 0; x < bakeryWidth; x++ {
				at := hide
				at.Cell.X, at.Cell.Y = x, y
				if at == hide || !c.bakery.isFloorCell(at.Cell) {
					continue
				}
				who := c.bakery.Entities[listener]
				who.Placement = at
				c.bakery.Entities[listener] = who
				view, err := c.bakery.Observe(scope, listener)
				if err != nil || view.hasTarget(customer) || c.bakery.soundReaches(hide.Cell, at.Cell, fragmentReach) || !c.bakery.soundReaches(hide.Cell, at.Cell, murmurReach) {
					continue
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatal("no cell hears only a murmur")
		}
	}
	var open NotebookEntry
	for _, note := range c.bakery.Notebooks[player] {
		if note.Overheard && note.Open && note.Target == porter {
			open = note
		}
	}
	choices, err := c.DialogueChoices(player, baker)
	if err != nil {
		t.Fatal(err)
	}
	var answer SpeechLine
	for _, choice := range choices {
		if choice.Words == "The porter is taking a breather." {
			answer = choice.Line
		}
	}
	if answer.Source != open.SourceSequence {
		t.Fatal(choices)
	}
	if r := c.Submit(player, spatialProposal(c, "answer", ActionSpeak, ActionArgs{Target: baker, Speech: answer})); r.Status != "accepted" {
		t.Fatal(r)
	}
	return c
}

func journalSpeech(pages []JournalEntry) string {
	var text strings.Builder
	for _, page := range pages {
		text.WriteString(page.Conversation.Words)
		text.WriteString("\n")
		text.WriteString(page.Conversation.Reply)
		text.WriteString("\n")
		for _, obs := range page.Observations {
			text.WriteString(obs.Text)
			text.WriteString("\n")
		}
	}
	return text.String()
}
