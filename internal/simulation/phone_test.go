// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestAResidentCanCallOnlyANumberTheyLearned(t *testing.T) {
	const player, porter, baker = "bakery/player", "bakery/porter", "bakery/baker"
	const street, shop = "bakery/street-phone", "bakery/bakery-phone"
	c, err := (BakerySetup{Practice: true}).NewCoordinator(Scope{"bakery", "main"}, 42)
	if err != nil {
		t.Fatal(err)
	}
	act := func(who string, action ActionKind, args ActionArgs) Result {
		t.Helper()
		r := c.Submit(who, spatialProposal(c, fmt.Sprint(c.clock.Tick()+1), action, args))
		if r.Status != "accepted" {
			t.Fatal(r)
		}
		return r
	}
	for range 4 {
		act(player, ActionWait, ActionArgs{})
	}
	p := spatialProposal(c, "ring", ActionCall, ActionArgs{Target: "101"})
	p.ByName = true
	if r := submitUnchanged(t, c, porter, p); r.Code != "number_unknown" {
		t.Fatal(r)
	}
	act(player, ActionSpeak, ActionArgs{Target: porter, Speech: OfferPhoneNumber})
	for _, who := range []string{player, porter, baker} {
		scene, err := c.Scene(who)
		if err != nil || (scene.PhoneBook.Details[player] == "101") != (who == porter) || len(scene.MailBook.People) != 0 || scene.PhoneBook.Details[porter] != "" {
			t.Fatal(who, scene, err)
		}
	}
	// The porter introduced themselves during their routine. The player later offered their number.
	pages, err := c.Journal(porter)
	if err != nil || !slices.ContainsFunc(pages, func(e JournalEntry) bool { return e.IntroducedTo.Ref == player }) ||
		!slices.ContainsFunc(pages, func(e JournalEntry) bool { return e.Conversation.Number == "101" && e.Conversation.WithRef == player }) {
		t.Fatal(pages, err)
	}
	setCell := func(id string, x, y int) {
		e := c.bakery.Entities[id]
		e.Placement.Cell.X, e.Placement.Cell.Y = x, y
		c.bakery.Entities[id] = e
	}
	setCell(porter, 1, 3)
	if r := submitUnchanged(t, c, porter, p); r.Code != "same_phone" {
		t.Fatal(r)
	}
	setCell(porter, 0, 0)
	if r := submitUnchanged(t, c, porter, p); r.Code != "phone_unavailable" {
		t.Fatal(r)
	}
	setCell(porter, 4, 1)
	switchLine := func(ref string, off bool) {
		e := c.bakery.Entities[ref]
		e.Disconnected = off
		c.bakery.Entities[ref] = e
	}
	switchLine(shop, true)
	if r := submitUnchanged(t, c, porter, p); r.Code != "phone_unavailable" {
		t.Fatal(r)
	}
	switchLine(shop, false)
	setCell(player, 1, 3)
	scene, _ := c.Scene(porter)
	if scene.hasTarget(player) {
		t.Fatal("expected the recipient to be out of sight", scene)
	}
	// The caller gets the same ring-out for an unavailable person and an unavailable receiving line.
	switchLine(street, true)
	miss, failure := c.bakery.resolveSpatial(porter, p)
	if failure != nil || miss.kind != "call_missed" {
		t.Fatal(miss, failure)
	}
	switchLine(street, false)
	recipient := c.bakery.Entities[player]
	recipient.Seated = true
	c.bakery.Entities[player] = recipient
	busy, failure := c.bakery.resolveSpatial(porter, p)
	if failure != nil || busy.kind != "call_missed" || busy.message != miss.message || busy.payload.Dialogue != (Exchange{}) {
		t.Fatal(busy, failure)
	}
	recipient.Seated = false
	c.bakery.Entities[player] = recipient
	p.Observed.Target = baker
	if r := submitUnchanged(t, c, porter, p); r.Code != "number_unknown" {
		t.Fatal("stale contact binding", r)
	}
	p.Observed.Target = player
	r := c.Submit(porter, p)
	if r.Status != "accepted" || !strings.Contains(r.Message, "answers your call") || r.Observation.hasTarget(player) {
		t.Fatal(r)
	}
	retry := submitUnchanged(t, c, porter, p)
	if retry.Message != r.Message {
		t.Fatal(retry)
	}
	for _, who := range []string{player, porter, baker} {
		j, err := c.Journal(who)
		if err != nil || slices.ContainsFunc(j, func(e JournalEntry) bool { return e.Conversation.ByPhone }) != (who != baker) {
			t.Fatal(who, j, err)
		}
	}
	// A forged number cannot be restored as somebody else's extension.
	c.bakery.Contacts[0].Number = "201"
	data, err := c.bakery.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreBakery(data, "bakery", c.clock); err == nil {
		t.Fatal("restored another person's number")
	}
}
