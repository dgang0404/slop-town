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

func TestMissedAndAnsweredCallsAcrossRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "phone.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "phone")
	start := checkpointAtHead(t, path, scope)
	interrupt := func(script string) {
		t.Helper()
		before, town := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
		execSQL(t, path, "CREATE TRIGGER fail_call BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted call'); END")
		play(path, script, "interrupted call")
		if !bytes.Equal(before, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(town, inspectScope(t, path, scope)) {
			t.Fatal("failed receipt changed town or history")
		}
		execSQL(t, path, "DROP TRIGGER fail_call")
	}
	const ask = "wait\nwait\nintroduce yourself to porter\ntalk to porter\n"
	const known = ask + "9\ncancel\n"
	const missed = known + "call 202\n"
	const street = missed + "south\nsouth\n"
	const answered = street + "call porter\n"
	out := play(path, ask, "")
	if !strings.Contains(out, "9. What number can I reach you on?") {
		t.Fatal(out)
	}
	interrupt(known)
	out = play(path, known+"journal people porter\n", "")
	if !strings.Contains(out, "Telephone: 202. Shared on day 1") || strings.Contains(out, "Mailing address:") {
		t.Fatal(out)
	}
	learned := checkpointAtHead(t, path, scope)
	interrupt(missed)
	out = play(path, missed+"journal\n", "")
	if !strings.Contains(out, "I got no answer on the porter’s line.") {
		t.Fatal(out)
	}
	rang := checkpointAtHead(t, path, scope)
	play(path, street, "")
	out = play(path, street+"look\nactions\n", "")
	if !strings.Contains(out, "  call the porter\n") || strings.Contains(out, "  talk to the porter\n") {
		t.Fatal("offscreen contact", out)
	}
	ready := checkpointAtHead(t, path, scope)
	interrupt(answered)
	out = play(path, answered+"journal people porter\njournal\n", "")
	if !strings.Contains(out, "The porter answers your call.") || strings.Count(out, "I called the porter and asked") != 1 ||
		!strings.Contains(out, "I last saw the porter in the bakery doorway.") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	if out = play(path, answered, ""); strings.Count(out, "already completed") != 8 || checkpointAtHead(t, path, scope) != head {
		t.Fatal("retry rang again", out)
	}
	counts := map[string]int{}
	town := inspectScope(t, path, scope)
	for _, e := range town.events {
		if e.Kind == "call_missed" || e.Kind == "call_answered" {
			counts[e.Kind]++
			if e.Payload.Call.Number != "202" || e.Payload.Call.From != "bakery/street-phone" || e.Payload.Call.To != "bakery/bakery-phone" ||
				(e.Kind == "call_missed" && e.Payload.Dialogue != (simulation.Exchange{})) {
				t.Fatal(e)
			}
		}
		if e.Tick == head.Tick && e.ActorID == "bakery/porter" {
			t.Fatal("recipient took another action while answering", e)
		}
	}
	if counts["call_missed"] != 1 || counts["call_answered"] != 1 || rang.Tick != learned.Tick+1 || head.Tick != ready.Tick+1 {
		t.Fatal(counts, rang, head)
	}
	for _, cp := range []storage.ReplayReport{start, learned, rang, ready} {
		if replay, err := storage.Replay(path, scope, cp.Sequence); err != nil || replay != head {
			t.Fatal(cp, replay, err)
		}
	}
	t.Logf("replayed sequences %d, %d, %d, %d to tick %d sequence %d sha256 %s", start.Sequence, learned.Sequence, rang.Sequence, ready.Sequence, head.Tick, head.Sequence, head.Hash)
}

func TestDirectNumberOfferKeepsItsIdentityOnRetry(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "contacts.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "give")
	met := strings.Repeat("wait\n", 4)
	offer := met + "give my number to porter\ncancel\n"
	if out := play(path, met+"actions\n", ""); !strings.Contains(out, "  give my number to the porter\n") {
		t.Fatal("crowded conversation lost the direct offer", out)
	}
	out := play(path, offer, "")
	if !strings.Contains(out, "You can reach me on 101 at the street phone.") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	if out = play(path, offer, ""); strings.Count(out, "already completed") != 5 {
		t.Fatal(out)
	}
	out = play(path, met+"ask porter for their number\n", "")
	if !strings.Contains(out, "different action") || checkpointAtHead(t, path, scope) != head {
		t.Fatal("changed disclosure reused the previous receipt", out)
	}
}
