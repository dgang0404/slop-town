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

func TestBakeryCoverAcrossRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "behind-crate.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 2, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "cover")
	start := checkpointAtHead(t, path, scope)
	const opening = "wait\ne\ne\ne\nn\nwait\nwait\nwait\n"
	const hide = opening + "hide behind the crate\n"
	play(path, opening, "")
	before := checkpointAtHead(t, path, scope)
	town, history := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
	execSQL(t, path, "CREATE TRIGGER fail_cover BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted hiding turn'); END")
	play(path, hide, "interrupted hiding turn")
	if !bytes.Equal(town, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(history, inspectScope(t, path, scope)) {
		t.Fatal("interrupted hide changed the posture, witnesses, workers or clock")
	}
	execSQL(t, path, "DROP TRIGGER fail_cover")
	if out := play(path, hide, ""); !strings.Contains(out, "You crouch behind the crate.") ||
		!strings.Contains(out, "From here you can no longer see") {
		t.Fatal(out)
	}
	hidden := checkpointAtHead(t, path, scope)
	look := play(path, "look\nactions\n", "")
	if !strings.Contains(look, "You are crouching behind the crate.") || !strings.Contains(look, "  come out\n") ||
		strings.Contains(look, "the baker ") || strings.Contains(look, "the porter") || strings.Contains(look, "hide behind the crate") {
		t.Fatal(look)
	}
	if checkpointAtHead(t, path, scope) != hidden {
		t.Fatal("reading the scene or browsing actions changed the hiding turn")
	}
	const visit = hide + "listen\ncome out\ntalk to baker\n3\njournal\njournal accounts\n"
	out := play(path, visit, "")
	const report = "The baker told me they saw me hide behind the crate."
	if !strings.Contains(out, "You hear footsteps.") || !strings.Contains(out, "The baker replies, “I saw you hide behind the crate.”") ||
		strings.Count(out, "I crouched behind the crate.") != 1 || strings.Contains(out, "I was crouching behind the crate.") ||
		!strings.Contains(out, "I stood up beside the crate.") || !strings.HasSuffix(out, report+"\n") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	if retry := play(path, visit, ""); strings.Count(retry, "already completed") != 12 || !strings.HasSuffix(retry, report+"\n") {
		t.Fatal(retry)
	}
	for _, from := range []uint64{start.Sequence, before.Sequence, hidden.Sequence} {
		if replay, err := storage.Replay(path, scope, from); err != nil || replay != head || replay.Tick != 12 {
			t.Fatal(replay, err)
		}
	}
	t.Logf("replay from %d, %d and %d reached tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, hidden.Sequence, head.Tick, head.Sequence, head.Hash)
}
