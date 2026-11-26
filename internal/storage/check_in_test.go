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

func TestMissedCheckInAcrossRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "meeting.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	checkpoint := func() storage.ReplayReport {
		t.Helper()
		head, err := storage.SaveCheckpoint(path, scope)
		if err != nil {
			t.Fatal(err)
		}
		return head
	}
	genesis := checkpoint()
	play := newAcceptancePlayer(t, "visit")
	const approach = "wait\neast\neast\n"
	const missed = approach + "south\n"
	play(path, approach, "")
	before := checkpoint()
	ledger, actors := inspectScope(t, path, scope), bakeryBytes(t, path, scope)
	execSQL(t, path, "CREATE TRIGGER fail_meeting BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'missed write'); END")
	play(path, missed, "missed write")
	if !reflect.DeepEqual(ledger, inspectScope(t, path, scope)) || !bytes.Equal(actors, bakeryBytes(t, path, scope)) {
		t.Fatal("the failed write left a meeting behind")
	}
	execSQL(t, path, "DROP TRIGGER fail_meeting")
	if out := play(path, missed+"journal\n", ""); strings.Contains(out, "checked in") || strings.Contains(out, "holding up") {
		t.Fatal(out)
	}
	after := checkpoint()
	const report = "The baker checked in on me. I told them I felt fine."
	const visit = missed + "north\ntalk to porter\n4\n4\ncancel\njournal\n"
	out := play(path, visit, "")
	_, book, found := strings.Cut(out, "My journal\n")
	if !found || !strings.Contains(book, "The porter replied, “"+report+"”") || !strings.Contains(book, "So the baker checked in on you?") {
		t.Fatal(out)
	}
	retried := play(path, visit, "")
	_, same, _ := strings.Cut(retried, "My journal\n")
	if strings.Count(retried, "already completed") != 7 || same != book {
		t.Fatal(retried)
	}
	head := checkpoint()
	for _, from := range []uint64{genesis.Sequence, before.Sequence, after.Sequence} {
		replayed, err := storage.Replay(path, scope, from)
		if err != nil || replayed != head || head.Tick != 7 {
			t.Fatal(from, replayed, err)
		}
	}
	t.Logf("replayed from %d, %d and %d to tick %d, sequence %d, sha256 %s", genesis.Sequence, before.Sequence, after.Sequence, head.Tick, head.Sequence, head.Hash)
}
