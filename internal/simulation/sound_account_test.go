// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"slices"
	"strings"
	"testing"
)

func TestSoundChoicesBelongToTheListener(t *testing.T) {
	const player, porter = "bakery/player", "bakery/porter"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	r := c.SubmitTurn(player, spatialProposal(c, "listen", ActionListen, ActionArgs{}))
	if !slices.Equal(r.Observation.Heard, []PerceptionKind{DoorCreak, Footsteps}) {
		t.Fatal(r)
	}
	choices, err := c.DialogueChoices(player, porter)
	if err != nil || len(choices) > 11 {
		t.Fatal(choices, err)
	}
	sources := make(map[PerceptionKind]uint64)
	for _, choice := range choices {
		if choice.Line.Intent != "share" {
			continue
		}
		entry, ok := c.bakery.firsthandAt(player, choice.Line.Source)
		if !ok || entry.Action != ActionListen || choice.Words != "I heard "+entry.Sound.SoundPhrase()+"." ||
			entry.Subject != "" || entry.Target != "" || entry.Container != "" {
			t.Fatal(choice, entry)
		}
		sources[entry.Sound] = entry.SourceSequence
	}
	if sources[DoorCreak] == 0 || sources[DoorCreak] >= sources[Footsteps] {
		t.Fatal("the two sounds need independent, ordered sources", sources)
	}
	// The porter cannot claim the player's footfalls as a firsthand sound.
	line := SpeechLine{Topic: string(Footsteps), Intent: "share", Tone: "plain", Source: sources[Footsteps]}
	p := spatialProposal(c, "borrowed", ActionSpeak, ActionArgs{Target: player, Speech: line})
	if r := submitUnchanged(t, c, porter, p); r.Code != "line_unavailable" {
		t.Fatal(r)
	}
	// Nor can the player substitute the creak's source for the footsteps.
	p.Args.Target, p.Args.Speech.Source = porter, sources[DoorCreak]
	if r := submitUnchanged(t, c, player, p); r.Code != "line_unavailable" {
		t.Fatal(r)
	}
	for _, topic := range []string{"sounds", "news"} {
		p := spatialProposal(c, "ask-"+topic, ActionSpeak, ActionArgs{Target: player, Speech: SpeechLine{Topic: topic, Intent: "ask", Tone: "curious"}})
		answer, failure := c.bakery.resolveSpatial(porter, p)
		if failure != nil || (answer.payload.Dialogue.Reply == "I heard footsteps.") != (topic == "sounds") ||
			strings.Contains(answer.payload.Dialogue.Reply, "saw footsteps") {
			t.Fatal(topic, answer.payload.Dialogue, failure)
		}
	}
	original := c.bakery.Notebooks[player][len(c.bakery.Notebooks[player])-1]
	for _, invalid := range []NotebookEntry{
		{Action: ActionListen, Sound: OnStreet}, {Action: ActionListen, Sound: Footsteps, Subject: porter},
		{Action: ActionListen, Sound: Footsteps, Target: "bakery/door"}, {Action: ActionRest, Sound: Footsteps, Subject: porter},
	} {
		invalid.Tick, invalid.SourceSequence = original.Tick, original.SourceSequence
		c.bakery.Notebooks[player][len(c.bakery.Notebooks[player])-1] = invalid
		if err := c.bakery.validateNotebooks(1); err == nil {
			t.Fatal("accepted invalid sound memory", invalid)
		}
	}
}
