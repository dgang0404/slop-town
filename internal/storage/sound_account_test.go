// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestFollowingASoundAccountAcrossRestart(t *testing.T) {
	const player, porter, baker = "bakery/player", "bakery/porter", "bakery/baker"
	path := filepath.Join(t.TempDir(), "bakery.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, player, simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	start := checkpointAtHead(t, path, scope)
	submit := func(actor, request string, action simulation.ActionKind, args simulation.ActionArgs) simulation.Result {
		t.Helper()
		r, repeated, err := storage.SubmitBakery(path, scope, actor, simulation.Proposal{Scope: scope, Version: 1, RequestID: request, Action: action, Args: args})
		if err != nil || repeated || r.Status != "accepted" {
			t.Fatal(r, err)
		}
		return r
	}
	if r := submit(porter, "listen", simulation.ActionListen, simulation.ActionArgs{}); !slices.Equal(r.Observation.Heard, []simulation.PerceptionKind{simulation.DoorCreak}) {
		t.Fatal(r)
	}
	submit(player, "wait-1", simulation.ActionWait, simulation.ActionArgs{})
	submit(player, "wait-2", simulation.ActionWait, simulation.ActionArgs{})
	sounds := simulation.SpeechLine{Topic: "sounds", Intent: "ask", Tone: "curious"}
	if r := submit(baker, "ask-porter", simulation.ActionSpeak, simulation.ActionArgs{Target: porter, Speech: sounds}); !strings.Contains(r.Message, "I heard a door creak.") {
		t.Fatal(r.Message)
	}
	before := checkpointAtHead(t, path, scope)
	menuNumber := func(person, words string) int {
		t.Helper()
		options, err := storage.BakeryDialogueChoices(path, scope, player, person)
		if err != nil || len(options) > 11 {
			t.Fatal(options, err)
		}
		for i, opt := range options {
			if opt.Words == words {
				return i + 1
			}
		}
		t.Fatalf("%q missing from %v", words, options)
		return 0
	}
	play := newAcceptancePlayer(t, "sound-report")
	ask := fmt.Sprintf("talk to baker\n%d\ncancel\njournal leads\n", menuNumber(baker, "What have you heard from other people?"))
	town, history := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
	execSQL(t, path, "CREATE TRIGGER fail_relay BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted sound report'); END")
	play(path, ask, "interrupted sound report")
	if !bytes.Equal(town, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(history, inspectScope(t, path, scope)) {
		t.Fatal("interrupted exchange changed the town, memories or history")
	}
	execSQL(t, path, "DROP TRIGGER fail_relay")
	if out := play(path, ask, ""); !strings.Contains(out, "The porter told me they heard a door creak.") ||
		!strings.Contains(out, "Did the porter hear a door creak, as the baker said?") {
		t.Fatal(out)
	}
	const verify = "The baker said you heard a door creak. Is that right?"
	follow := ask + fmt.Sprintf("talk to porter\n%d\ncancel\njournal accounts\njournal leads\n", menuNumber(porter, verify))
	out := play(path, follow, "")
	_, accounts, ok := strings.Cut(out, "Accounts I've heard\n")
	if !ok || !strings.Contains(accounts, "The baker said the porter heard a door creak.") ||
		!strings.Contains(accounts, "The porter told me they heard a door creak.") || strings.Contains(accounts, "saw") ||
		!strings.HasSuffix(accounts, "I have no accounts to follow up.\n") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	again := play(path, follow, "")
	_, reopened, _ := strings.Cut(again, "Accounts I've heard\n")
	if reopened != accounts || strings.Count(again, "already completed") != 2 {
		t.Fatal(again)
	}
	for _, from := range []uint64{start.Sequence, before.Sequence} {
		if r, err := storage.Replay(path, scope, from); err != nil || r != head || r.Tick != 6 {
			t.Fatal(r, err)
		}
	}
	t.Logf("replay from %d and %d reached tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, head.Tick, head.Sequence, head.Hash)
}
