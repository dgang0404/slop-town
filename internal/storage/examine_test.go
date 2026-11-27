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

func TestBakeryExaminationAcrossRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "examine.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "examine")
	start := checkpointAtHead(t, path, scope)
	if out := play(path, "look\nactions\nexamine spoon\n", ""); !strings.Contains(out, "  examine the bakery door\n") ||
		strings.Contains(out, "wooden spoon") || !strings.Contains(out, "You can't see that here.") || checkpointAtHead(t, path, scope) != start {
		t.Fatal(out)
	}
	const atCrate = "examine door\ne\nwalk to crate\nexamine crate\n"
	out := play(path, atCrate, "")
	if !strings.Contains(out, "You find the bakery door closed, the brass worn bright around the latch.") ||
		!strings.Contains(out, "The bakery door is now open.") || !strings.Contains(out, "You find the crate closed, a pale scrape running across its lid.") ||
		strings.Contains(out, "wooden spoon") {
		t.Fatal(out)
	}
	const opened = atCrate + "open it\n"
	play(path, opened, "")
	before := checkpointAtHead(t, path, scope)
	town, history := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
	execSQL(t, path, "CREATE TRIGGER fail_examine BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted examination'); END")
	play(path, opened+"examine it\n", "interrupted examination")
	if !bytes.Equal(town, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(history, inspectScope(t, path, scope)) {
		t.Fatal("interruption changed the town, clock or saved observations")
	}
	execSQL(t, path, "DROP TRIGGER fail_examine")
	const visit = opened + "examine it\nexamine spoon\ntake it\nexamine crate\njournal\n"
	out = play(path, visit, "")
	const full = "I found the crate open with the wooden spoon inside, a pale scrape running across its lid."
	const empty = "I found the crate open with nothing visible inside, a pale scrape running across its lid."
	if strings.Count(out, full) != 1 || strings.Count(out, empty) != 1 || strings.Index(out, full) > strings.Index(out, empty) ||
		!strings.Contains(out, "I found the wooden spoon in the crate, a shallow notch cut into its handle.") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	if retry := play(path, visit, ""); strings.Count(retry, "already completed") != 9 || strings.Count(retry, full) != 1 ||
		checkpointAtHead(t, path, scope) != head {
		t.Fatal(retry)
	}
	for _, from := range []uint64{start.Sequence, before.Sequence} {
		if replay, err := storage.Replay(path, scope, from); err != nil || replay != head || replay.Tick != 9 {
			t.Fatal(replay, err)
		}
	}
	t.Logf("replay from %d and %d reached tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, head.Tick, head.Sequence, head.Hash)
}
