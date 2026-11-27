// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"strconv"
	"testing"
)

func TestWorkroomRoundTripAcrossSeeds(t *testing.T) {
	const player = "bakery/player"
	for seed := int64(0); seed < 100; seed++ {
		c, err := (BakerySetup{Workroom: true}).NewCoordinator(Scope{"bakery", "main"}, seed)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		i := 0
		act := func(kind ActionKind, args ActionArgs) {
			t.Helper()
			i++
			r := c.Submit(player, spatialProposal(c, strconv.Itoa(i), kind, args))
			if r.Status != "accepted" {
				t.Fatalf("seed %d, %s: %v", seed, kind, r)
			}
		}
		walk := func(dir Direction) { act(ActionMove, ActionArgs{Direction: dir}) }
		toward, back := South, North
		if c.bakery.Entities["bakery/door"].Placement.Cell.Y == 2 {
			toward, back = North, South
		}
		act(ActionOpen, ActionArgs{Target: "bakery/door"})
		for range 3 {
			walk(East)
		}
		view, err := c.Scene(player)
		if err != nil || view.hasTarget("bakery/workbench") {
			t.Fatalf("seed %d: closed room leaked", seed)
		}
		act(ActionOpen, ActionArgs{Target: "bakery/workroom-door"})
		walk(toward)
		walk(toward)
		view, err = c.Scene(player)
		if err != nil || view.Self.Place != InWorkroom || !view.hasTarget("bakery/workbench") {
			t.Fatal(seed, view, err)
		}
		act(ActionClose, ActionArgs{Target: "bakery/workroom-door"})
		act(ActionOpen, ActionArgs{Target: "bakery/workbench"})
		act(ActionRest, ActionArgs{})
		act(ActionOpen, ActionArgs{Target: "bakery/workroom-door"})
		walk(back)
		walk(back)
		for range 3 {
			walk(West)
		}
		view, err = c.Scene(player)
		if err != nil || view.Self.Place != OnStreet {
			t.Fatal(seed, err)
		}
	}
}

func TestWorkroomRoundLeavesOtherItemsInPlace(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Workroom: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	baker := b.Entities["bakery/baker"]
	baker.Placement = b.Entities["bakery/workroom-door"].Placement
	b.Entities["bakery/baker"] = baker
	drawer := b.Entities["bakery/workbench"]
	drawer.Open = true
	b.Entities["bakery/workbench"] = drawer
	spoon := b.Entities["bakery/spoon"]
	spoon.Placement = Placement{HolderID: "bakery/workbench"}
	b.Entities["bakery/spoon"] = spoon
	if action := b.workroomRound(scope, "bakery/baker", "bakery/door"); action.kind != "" {
		t.Fatal(action)
	}
}

func TestBakerAnswersFromOwnKnowledge(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Workroom: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	const baker, player, drawer = "bakery/baker", "bakery/player", "bakery/workbench"
	actor := b.Entities[player]
	actor.Placement = b.Entities[baker].Placement
	b.Entities[player] = actor
	view, err := b.Observe(scope, player)
	if err != nil {
		t.Fatal(err)
	}
	line := SpeechLine{Topic: "workroom", Intent: "ask_supplies", Tone: "curious"}
	ask := func() string {
		t.Helper()
		effect, fail := b.speak(view, baker, line)
		if fail != nil {
			t.Fatal(fail)
		}
		return effect.payload.Dialogue.Reply
	}
	toolID := b.Entities["bakery/order"].RequestedItem
	tool := b.Entities[toolID]
	before := ask()
	tool.Placement = Placement{HolderID: drawer}
	b.Entities[toolID] = tool
	if got := ask(); got != before {
		t.Fatal("unseen delivery changed the answer", got)
	}
	tool.Placement = Placement{HolderID: baker}
	b.Entities[toolID] = tool
	collected := ask()
	if collected != "I have the dough scraper I need for the workroom." {
		t.Fatal(collected)
	}
	b.Notebooks = map[string][]NotebookEntry{baker: {{Subject: "bakery/porter", Action: ActionPut, Target: toolID, Container: drawer}}}
	if got := ask(); got != collected {
		t.Fatal("credited wrong speaker", got)
	}
	b.Notebooks[baker][0].Subject = player
	if got := ask(); got != collected+" I saw you put it in the drawer. Thank you." {
		t.Fatal(got)
	}
	b.ContentVersion = workroomRoundsVersion
	if _, fail := b.speak(view, baker, line); fail == nil {
		t.Fatal("old timeline accepted new line")
	}
}

func TestHelperGreetingRequiresRecognitionAndWitnessedDelivery(t *testing.T) {
	scope := Scope{"bakery", "main"}
	c, err := (BakerySetup{Workroom: true}).NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	b := c.bakery
	const baker, helper = "bakery/baker", "bakery/porter"
	toolID := b.Entities["bakery/order"].RequestedItem
	tool := b.Entities[toolID]
	intro := NotebookEntry{Action: ActionIntroduce, Subject: helper, Target: baker}
	sight := NotebookEntry{Action: ActionPut, Subject: helper, Target: toolID, Container: "bakery/workbench"}
	view, err := b.Observe(scope, baker)
	if err != nil {
		t.Fatal(err)
	}
	greet := func(want bool) {
		t.Helper()
		reply := b.answer(helper, view, DialogueChoice{Line: SpeechLine{Intent: "greet"}, Words: "Hello."})
		if (reply.Number != "") != want {
			t.Fatal(reply)
		}
	}
	b.Notebooks = map[string][]NotebookEntry{baker: {intro, sight}}
	greet(false)
	tool.Placement = Placement{HolderID: baker}
	b.Entities[toolID] = tool
	b.Notebooks[baker] = []NotebookEntry{sight}
	greet(false)
	b.Notebooks[baker] = []NotebookEntry{intro}
	greet(false)
	b.Notebooks[baker] = append(b.Notebooks[baker], sight)
	greet(true)
	b.ContentVersion = supplyQuestionVersion
	greet(false)
	b.ContentVersion = helperGreetingVersion
	b.Contacts = []sharedContact{{Owner: helper, Person: baker, Number: "201"}}
	greet(false)
}
