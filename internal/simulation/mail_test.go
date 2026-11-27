// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestAddressesAndSealedLettersBelongToTheirRecipient(t *testing.T) {
	const player, porter, baker = "bakery/player", "bakery/porter", "bakery/baker"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	act := func(who string, kind ActionKind, args ActionArgs) Result {
		t.Helper()
		r := c.Submit(who, spatialProposal(c, fmt.Sprint(c.clock.Tick()+1), kind, args))
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		return r
	}
	post := ActionArgs{Target: porter, Body: "The BLUE spoon: is it yours?"}
	ask := ActionArgs{Target: porter, Speech: askAddress}
	act(player, ActionWait, ActionArgs{})
	act(player, ActionWait, ActionArgs{})
	if r := submitUnchanged(t, c, player, spatialProposal(c, "stranger", ActionSpeak, ask)); r.Code != "line_unavailable" {
		t.Fatal(r)
	}
	act(player, ActionIntroduce, ActionArgs{Target: porter})
	if r := submitUnchanged(t, c, player, spatialProposal(c, "guess", ActionPost, post)); r.Code != "address_unknown" {
		t.Fatal(r)
	}
	act(player, ActionSpeak, ask)
	if r := submitUnchanged(t, c, player, spatialProposal(c, "ask-again", ActionSpeak, ask)); r.Code != "line_unavailable" {
		t.Fatal(r)
	}
	for _, who := range []string{player, porter, baker} {
		scene, err := c.Scene(who)
		if err != nil || (scene.MailBook.Details[porter] != "") != (who == player) || scene.MailBook.Details[player] != "" {
			t.Fatal(who, scene, err)
		}
	}
	for _, body := range []string{"", "  ", "bell\a", strings.Repeat("x", 501)} {
		if r := submitUnchanged(t, c, player, spatialProposal(c, "bad", ActionPost, ActionArgs{Target: porter, Body: body})); r.Code != "invalid_request" {
			t.Fatal(r)
		}
	}
	act(player, ActionPost, post)
	for range mailTurns {
		if r := submitUnchanged(t, c, porter, spatialProposal(c, "early", ActionReadLetter, ActionArgs{})); r.Code != "no_letter" {
			t.Fatal(r)
		}
		act(player, ActionWait, ActionArgs{})
	}
	for _, who := range []string{player, baker} {
		if r := submitUnchanged(t, c, who, spatialProposal(c, "wrong-envelope", ActionReadLetter, ActionArgs{})); r.Code != "no_letter" {
			t.Fatal(r)
		}
	}
	pages, err := c.Journal(porter)
	if err != nil || strings.Contains(fmt.Sprint(pages), post.Body) {
		t.Fatal("sealed contents reached the recipient's journal", pages, err)
	}
	if r := act(porter, ActionReadLetter, ActionArgs{}); !strings.Contains(r.Message, post.Body) {
		t.Fatal(r)
	}
	for range 20 {
		act(player, ActionWait, ActionArgs{})
	}
	if len(c.bakery.Letters) != 1 {
		t.Fatal("porter replied without a return address", c.bakery.Letters)
	}
	pages, err = c.Journal(porter)
	if err != nil || !strings.Contains(fmt.Sprint(pages), post.Body) {
		t.Fatal(pages, err)
	}
	third, err := c.Journal(baker)
	if err != nil || strings.Contains(fmt.Sprint(third), post.Body) {
		t.Fatal("a bystander learned the text", third, err)
	}
	// Restore must reconcile the envelope with the delivery schedule and the learned address.
	original := c.bakery.Letters[0]
	for _, damage := range []func(*letter){func(l *letter) { l.Delivered = false }, func(l *letter) { l.Address = "Unshared box" }, func(l *letter) { l.ReadTick = l.Tick + 1 }, func(l *letter) { l.ReplyTo = l.Post }} {
		c.bakery.Letters[0] = original
		damage(&c.bakery.Letters[0])
		data, err := c.bakery.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(data, "bakery", c.clock); err == nil {
			t.Fatal("restored impossible letter", c.bakery.Letters[0])
		}
	}
}

func TestEarlierTownKeepsItsConversationChoices(t *testing.T) {
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	usePracticeVersion(c.bakery, landmarkContentVersion)
	for i := range 4 {
		if r := c.Submit("bakery/player", spatialProposal(c, fmt.Sprint(i), ActionWait, ActionArgs{})); r.Status != "accepted" {
			t.Fatal(r)
		}
	}
	choices, err := c.DialogueChoices("bakery/player", "bakery/porter")
	if err != nil || len(choices) == 0 || slices.ContainsFunc(choices, func(c DialogueChoice) bool { return c.Line == askAddress || c.Line == offerAddress }) {
		t.Fatal(choices, err)
	}
	if r := submitUnchanged(t, c, "bakery/player", spatialProposal(c, "old", ActionReadLetter, ActionArgs{})); r.Code != "action_unavailable" {
		t.Fatal(r)
	}
}
