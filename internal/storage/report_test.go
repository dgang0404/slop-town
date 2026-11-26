// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestEyewitnessReportAcrossRestart(t *testing.T) {
	const player = "bakery/player"
	path := filepath.Join(t.TempDir(), "news.sqlite")
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, player, simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	initial := checkpointAtHead(t, path, scope)
	play := newAcceptancePlayer(t, "reports")
	const before = "rest\neast\neast\n"
	const after = before + "south\n"
	const ask = after + "north\ntalk to baker\n5\n"
	const route = ask + "4\ncancel\njournal\n"
	const account = "The porter told me they saw you sit down."

	// Interrupt first the off-screen account, then the player's discovery.
	interrupt := func(prefix, command string) storage.ReplayReport {
		t.Helper()
		play(path, prefix, "")
		checkpoint := checkpointAtHead(t, path, scope)
		state, actors := inspectScope(t, path, scope), bakeryBytes(t, path, scope)
		execSQL(t, path, "CREATE TRIGGER fail_report BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted report'); END")
		play(path, command, "interrupted report")
		if !reflect.DeepEqual(inspectScope(t, path, scope), state) || !bytes.Equal(actors, bakeryBytes(t, path, scope)) {
			t.Fatal("failed report write changed the town")
		}
		execSQL(t, path, "DROP TRIGGER fail_report")
		return checkpoint
	}
	first := interrupt(before, after)
	if out := play(path, after+"journal\n", ""); strings.Contains(out, "told me") || strings.Contains(out, "I saw the player") {
		t.Fatal(out)
	}
	second := interrupt(after+"north\n", ask)
	out := play(path, route, "")
	_, book, ok := strings.Cut(out, "My journal\n")
	if !ok || !strings.Contains(book, "The baker replied, “"+account+"”") || !strings.Contains(book, "Did the porter tell you they saw me sit down?") {
		t.Fatal(out)
	}
	events := inspectScope(t, path, scope).events
	if len(events) < 15 {
		t.Fatalf("only %d events in the visit", len(events))
	}
	offscreen := events[14]
	if offscreen.Tick != 4 || offscreen.ActorID != "bakery/porter" || offscreen.Payload.Target != "bakery/baker" ||
		offscreen.Payload.Dialogue != (simulation.Exchange{SpeakerSource: 2, Statement: simulation.SpeechLine{Topic: player, Intent: "share", Tone: "plain", Source: 2}}) {
		t.Fatal(offscreen)
	}
	for _, actor := range []string{"bakery/baker", "bakery/porter"} {
		pages, err := storage.BakeryJournal(path, scope, actor)
		found := false
		for _, page := range pages {
			if page.Conversation.Words == "I saw the player sit down." && page.Conversation.Reply == "Thanks for telling me." && page.Conversation.IStarted == (actor == "bakery/porter") {
				found = true
			}
		}
		if err != nil || !found {
			t.Fatal(actor, pages, err)
		}
	}
	head := checkpointAtHead(t, path, scope)
	retry := play(path, route, "")
	_, again, _ := strings.Cut(retry, "My journal\n")
	if strings.Count(retry, "already completed") != 7 || book != again {
		t.Fatal(retry)
	}
	for _, from := range []uint64{initial.Sequence, first.Sequence, second.Sequence} {
		replayed, err := storage.Replay(path, scope, from)
		if err != nil || replayed != head || head.Tick != 7 {
			t.Fatal(replayed, err)
		}
	}
	t.Logf("replay from %d, %d and %d reached tick %d, sequence %d, sha256 %s", initial.Sequence, first.Sequence, second.Sequence, head.Tick, head.Sequence, head.Hash)
	// Reopen the town and take the account to the named witness. A failed
	// write must leave the report available for the same question on retry.
	const witnessVisit = ask + "4\ncancel\ntalk to porter\n4\ncancel\njournal\n"
	beforeQuestion := interrupt(route, witnessVisit)
	witnessOut := play(path, witnessVisit, "")
	_, witnessBook, _ := strings.Cut(witnessOut, "My journal\n")
	const direct = "The porter replied, “I saw you sit down.”"
	if !strings.Contains(witnessBook, account) || strings.Count(witnessBook, direct) != 1 {
		t.Fatal(witnessOut)
	}
	final := checkpointAtHead(t, path, scope)
	for _, from := range []uint64{initial.Sequence, beforeQuestion.Sequence} {
		replayed, err := storage.Replay(path, scope, from)
		if err != nil || replayed != final || final.Tick != 8 {
			t.Fatal(replayed, err)
		}
	}
	retry = play(path, witnessVisit, "")
	_, again, _ = strings.Cut(retry, "My journal\n")
	if strings.Count(retry, "already completed") != 8 || again != witnessBook {
		t.Fatal(retry)
	}
	t.Logf("witness reply replayed to tick %d, sequence %d, sha256 %s", final.Tick, final.Sequence, final.Hash)

}
