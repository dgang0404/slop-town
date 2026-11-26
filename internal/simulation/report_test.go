// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestEyewitnessReportAndRetelling(t *testing.T) {
	const player, baker, porter, miller = "bakery/player", "bakery/baker", "bakery/porter", "bakery/miller"
	const crate = "bakery/crate"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	// A closed bakery door separates the miller and porter from the player and baker.
	inside, outside := b.Entities[crate].Placement, b.Entities[player].Placement
	b.Entities[miller] = SpatialEntity{Name: "miller", Kind: Actor, Placement: inside}
	place := func(who string, at Placement) {
		e := b.Entities[who]
		e.Placement = at
		b.Entities[who] = e
	}
	place(porter, inside)
	place(baker, outside)
	say := func(who, to string, line SpeechLine) Result {
		t.Helper()
		p := spatialProposal(c, who+"/"+line.Intent, ActionSpeak, ActionArgs{Target: to, Speech: line})
		r := c.Submit(who, p)
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		return r
	}
	for _, action := range []ActionKind{ActionOpen, ActionClose} {
		r := c.Submit(miller, spatialProposal(c, string(action), action, ActionArgs{Target: crate}))
		if r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	b = c.bakery
	sight, ok := b.firsthandAt(porter, 2)
	if !ok || sight.Subject != miller || sight.Action != ActionOpen || sight.Target != crate || b.Entities[crate].Open {
		t.Fatal(sight)
	}
	if _, ok := b.firsthandAt(baker, 2); ok {
		t.Fatal("baker saw through the closed door")
	}
	place(baker, outside)
	place(porter, outside)
	const attributed = "The porter told me they saw the miller open the crate."
	find := func(from, to, words string) (SpeechLine, bool) {
		t.Helper()
		options, err := c.DialogueChoices(from, to)
		if err != nil || len(options) > 11 {
			t.Fatal(options, err)
		}
		for _, option := range options {
			if option.Words == words {
				return option.Line, true
			}
		}
		return SpeechLine{}, false
	}
	if _, ok := find(baker, player, attributed); ok {
		t.Fatal("private observation appeared before anyone spoke")
	}
	say(porter, baker, sight.speechLine("share", "plain"))
	b = c.bakery
	if effect := b.residentReport(c.scope, porter, baker); effect.payload.Dialogue.SpeakerSource == sight.SourceSequence {
		t.Fatal("repeated an observation already shared with this coworker")
	}
	news := spatialProposal(c, "own-sightings", ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "news", Intent: "ask", Tone: "curious"}})
	answer, failure := b.resolveSpatial(player, news)
	if failure != nil || strings.Contains(answer.payload.Dialogue.Reply, "open the crate") {
		t.Fatal("secondhand news appeared among firsthand sightings", answer, failure)
	}
	line, ok := find(baker, player, attributed)
	if !ok || line.Intent != "relay" {
		t.Fatal(line)
	}
	// Borrowing the origin event directly would lose the person who told the baker.
	forged := spatialProposal(c, "forged", ActionSpeak, ActionArgs{Target: player, Speech: sight.speechLine("share", "plain")})
	if r := submitUnchanged(t, c, baker, forged); r.Code != "line_unavailable" {
		t.Fatal(r)
	}
	r := say(baker, player, line)
	if !strings.Contains(r.Message, attributed) || strings.Contains(r.Message, "spoon") {
		t.Fatal(r.Message)
	}
	b = c.bakery
	i := slices.IndexFunc(b.Notebooks[player], func(e NotebookEntry) bool { return e.Reported == line.Source && e.Subject == baker })
	if i < 0 {
		t.Fatal("player did not retain the baker's account", b.Notebooks[player])
	}
	report := b.Notebooks[player][i]
	heard, ok := b.entryAt(baker, report.Reported)
	if !ok || report.Subject != baker || heard.Subject != porter || heard.Reported != sight.SourceSequence {
		t.Fatal(report, heard)
	}
	if _, ok := b.firsthandAt(player, report.SourceSequence); ok {
		t.Fatal("retelling became a firsthand observation")
	}
	_, ok = find(player, baker, "Did the porter tell you they saw the miller open the crate?")
	if !ok {
		t.Fatal("missing source confirmation")
	}
	choices, err := c.DialogueChoices(player, porter)
	if err != nil || slices.ContainsFunc(choices, func(o DialogueChoice) bool { return o.Line.Intent == "relay" && o.Line.Source == report.SourceSequence }) {
		t.Fatal(choices, err)
	}
}

func TestReportSourcesAcrossSave(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	const player, baker, porter = "bakery/player", "bakery/baker", "bakery/porter"
	steps := []struct {
		action ActionKind
		args   ActionArgs
	}{
		{ActionRest, ActionArgs{}}, {ActionMove, ActionArgs{Direction: East}}, {ActionMove, ActionArgs{Direction: East}},
		{ActionMove, ActionArgs{Direction: South}}, {ActionMove, ActionArgs{Direction: North}},
		{ActionSpeak, ActionArgs{Target: baker, Speech: SpeechLine{Topic: "hearsay", Intent: "ask", Tone: "curious"}}},
	}
	for i, step := range steps {
		if r := c.Submit(player, spatialProposal(c, fmt.Sprint(i), step.action, step.args)); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	b := c.bakery
	restore := func() error {
		t.Helper()
		raw, err := b.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		_, err = RestoreBakery(raw, c.scope.WorldID, c.clock)
		return err
	}
	if effect := b.residentReport(c.scope, porter, baker); effect.kind != "" {
		t.Fatal("the porter repeated the same observation to the baker", effect)
	}
	original := slices.Clone(b.Notebooks[player])
	if err := restore(); err != nil || len(original) != 1 || original[0].Reported == 0 {
		t.Fatal(original, err)
	}
	wrongTeller, circular := original[0], original[0]
	wrongTeller.Subject = porter
	circular.Reported = circular.SourceSequence
	for _, bad := range []NotebookEntry{wrongTeller, circular} {
		b.Notebooks[player] = []NotebookEntry{bad}
		if restore() == nil {
			t.Fatal("restored a missing or circular source", bad)
		}
	}
	b.Notebooks[player] = original
	b.ContentVersion = conversationContentVersion
	if restore() == nil {
		t.Fatal("restored a two-link report under historical conversation rules")
	}
}
