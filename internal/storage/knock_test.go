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

func TestBakeryKnockAcrossRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "knock.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 43, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "knock")
	start := checkpointAtHead(t, path, scope)
	if menu := play(path, "actions\nknock on\n", ""); !strings.Contains(menu, "  knock on the bakery door\n") || !strings.Contains(menu, "Use knock on NAME.") {
		t.Fatal(menu)
	}
	if checkpointAtHead(t, path, scope) != start {
		t.Fatal("browsing or entering an incomplete command spent a turn")
	}
	const knock = "knock on the bakery door\n"
	town, history := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
	execSQL(t, path, "CREATE TRIGGER fail_knock BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted knock'); END")
	play(path, knock, "interrupted knock")
	if !bytes.Equal(town, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(history, inspectScope(t, path, scope)) {
		t.Fatal("interrupted knock changed the town, memories or history")
	}
	execSQL(t, path, "DROP TRIGGER fail_knock")
	if out := play(path, knock, ""); !strings.Contains(out, "You knock on the bakery door.") || strings.Contains(out, "You hear a knock.") {
		t.Fatal(out)
	}
	knocked := checkpointAtHead(t, path, scope)
	const visit = knock + "talk to porter\n3\ncancel\ntalk to baker\n8\ncancel\njournal\njournal accounts\n"
	out := play(path, visit, "")
	const sight = "The porter told me they saw me knock on the bakery door."
	const sound = "The baker told me they heard a knock."
	if !strings.Contains(out, "The porter replies, “I saw you knock on the bakery door.”") ||
		!strings.Contains(out, "The baker replies, “I heard a knock.”") || strings.Contains(out, "The baker told me they saw") ||
		strings.Count(out, "I knocked on the bakery door.") != 1 || !strings.Contains(out, sight) || !strings.HasSuffix(out, sound+"\n") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	if retry := play(path, visit, ""); strings.Count(retry, "already completed") != 3 || !strings.HasSuffix(retry, sound+"\n") ||
		checkpointAtHead(t, path, scope) != head {
		t.Fatal(retry)
	}
	for _, from := range []uint64{start.Sequence, knocked.Sequence} {
		if replay, err := storage.Replay(path, scope, from); err != nil || replay != head || replay.Tick != 3 {
			t.Fatal(replay, err)
		}
	}
	t.Logf("replay from %d and %d reached tick %d, sequence %d, sha256 %s", start.Sequence, knocked.Sequence, head.Tick, head.Sequence, head.Hash)
}
