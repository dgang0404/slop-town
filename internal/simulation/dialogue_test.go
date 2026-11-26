// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestDialogueKnowledgeAndAttribution(t *testing.T) {
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		if r := c.Submit(player, spatialProposal(c, fmt.Sprint(i), ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}

	// The porter introduced themselves to the baker on tick three.
	lines := func(from, to string) []DialogueChoice {
		t.Helper()
		choices, err := c.DialogueChoices(from, to)
		if err != nil {
			t.Fatal(err)
		}
		return choices
	}
	find := func(choices []DialogueChoice, words string) SpeechLine {
		t.Helper()
		for _, choice := range choices {
			if choice.Words == words {
				return choice.Line
			}
		}
		t.Fatalf("missing %q in %v", words, choices)
		return SpeechLine{}
	}
	const report = "The porter introduced themselves to me."
	known := find(lines(baker, player), report)
	if known.Source == 0 {
		t.Fatal(known)
	}
	for _, attempted := range []SpeechLine{known, {Topic: porter, Intent: "confirm", Tone: "careful", Source: known.Source},
		{Topic: "greeting", Intent: "greet", Tone: "hostile"}} {
		bad := spatialProposal(c, "private", ActionSpeak, ActionArgs{Target: baker, Speech: attempted})
		if r := submitUnchanged(t, c, player, bad); r.Code != "line_unavailable" {
			t.Fatal(r)
		}
	}
	// A resident can choose to tell a player using the same speech action.
	say := spatialProposal(c, "tell", ActionSpeak, ActionArgs{Target: player, Speech: known})
	if r := c.Submit(baker, say); r.Status != "accepted" || !strings.Contains(r.Message, report) {
		t.Fatal(r)
	}
	if r := submitUnchanged(t, c, baker, say); r.Status != "accepted" {
		t.Fatal(r)
	}
	for _, tc := range []struct {
		actor          string
		started, heard bool
	}{{baker, true, true}, {player, false, true}, {porter, false, false}} {
		book, err := c.Journal(tc.actor)
		if err != nil {
			t.Fatal(err)
		}
		var exchanges []JournalSpeech
		for _, page := range book {
			if page.Conversation.With != "" {
				exchanges = append(exchanges, page.Conversation)
			}
		}
		if (len(exchanges) == 1) != tc.heard || (tc.heard && (exchanges[0].IStarted != tc.started || exchanges[0].Words != report)) {
			t.Fatal(tc, exchanges)
		}
	}
	confirm := find(lines(player, baker), "So the porter introduced themselves to you?")
	if confirm.Source != known.Source {
		t.Fatal(confirm)
	}
	bad := spatialProposal(c, "wrong-source", ActionSpeak, ActionArgs{Target: porter, Speech: confirm})
	if r := submitUnchanged(t, c, player, bad); r.Code != "line_unavailable" {
		t.Fatal(r)
	}
	say = spatialProposal(c, "confirm", ActionSpeak, ActionArgs{Target: baker, Speech: confirm})
	if r := c.Submit(player, say); r.Status != "accepted" || !strings.Contains(r.Message, "Yes.") {
		t.Fatal(r)
	}

	// A changed save cannot point a report back to the exchange that created it.
	for i, entry := range c.bakery.Notebooks[player] {
		if entry.Action == ActionSpeak {
			c.bakery.Notebooks[player][i].Reported = entry.SourceSequence
			raw, err := c.bakery.Snapshot()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RestoreBakery(raw, c.scope.WorldID, c.clock); err == nil {
				t.Fatal("restored circular report")
			}
			return
		}
	}
	t.Fatal("listener did not retain the report")
}

func TestDialogueVisibleChoicesAndStaleSelection(t *testing.T) {
	const player, porter = "bakery/player", "bakery/porter"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	wait := spatialProposal(c, "wait", ActionWait, ActionArgs{})
	if r := c.Submit(player, wait); r.Status != "accepted" {
		t.Fatal(r)
	}
	for i := range 14 {
		e := SpatialEntity{Name: fmt.Sprintf("parcel %02d", i), Kind: Item, Placement: c.bakery.Entities[player].Placement, Portable: true}
		c.bakery.Entities["bakery/"+e.Name] = e
	}
	choices, err := c.DialogueChoices(player, porter)
	if err != nil || len(choices) != 11 {
		t.Fatal(choices, err)
	}
	again, err := c.DialogueChoices(player, porter)
	if err != nil || !slices.Equal(choices, again) {
		t.Fatal(again, err)
	}
	seen := map[string]bool{}
	for _, c := range choices {
		if seen[c.Words] || strings.Contains(c.Words, "spoon") {
			t.Fatal(c)
		}
		seen[c.Words] = true
	}
	choice := choices[10]
	p := spatialProposal(c, "ask", ActionSpeak, ActionArgs{Target: porter, Speech: choice.Line})
	hidden := c.bakery.Entities[choice.Line.Topic]
	hidden.Placement = Placement{HolderID: "bakery/crate"}
	c.bakery.Entities[choice.Line.Topic] = hidden
	if r := submitUnchanged(t, c, player, p); r.Code != "line_unavailable" {
		t.Fatal(r)
	}
	worker := c.bakery.Entities[porter]
	worker.Placement.Cell.X, worker.Placement.Cell.Y = 0, 3
	c.bakery.Entities[porter] = worker
	p.Args.Speech = choices[0].Line
	if r := submitUnchanged(t, c, player, p); r.Code != "target_unavailable" {
		t.Fatal(r)
	}
}

func TestDialogueAnswersFromListenersOwnView(t *testing.T) {
	const player, porter, spoon = "bakery/player", "bakery/porter", "bakery/spoon"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	worker := b.Entities[porter]
	worker.Placement = b.Entities[player].Placement
	worker.SeekingRestSpot = true
	b.Entities[porter] = worker
	item := b.Entities[spoon]
	item.Placement = Placement{HolderID: player}
	b.Entities[spoon] = item
	ask := func(line SpeechLine, want string) {
		t.Helper()
		p := spatialProposal(c, "ask", ActionSpeak, ActionArgs{Target: porter, Speech: line})
		effect, failure := b.resolveSpatial(player, p)
		if failure != nil || effect.payload.Dialogue.Reply != want {
			t.Fatal(failure, effect)
		}
	}
	ask(SpeechLine{Topic: spoon, Intent: "notice", Tone: "curious"}, "I can't see it from here.")
	condition := SpeechLine{Topic: "condition", Intent: "ask", Tone: "gentle"}
	ask(condition, "I could use a place to sit.")
	worker.SeekingRestSpot, worker.ReliefTurns = false, reliefDuration
	b.Entities[porter] = worker
	ask(condition, "Better. I needed that rest.")
	ask(SpeechLine{Topic: "news", Intent: "ask", Tone: "curious"}, "Nothing comes to mind.")
}
