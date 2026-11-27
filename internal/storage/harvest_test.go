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

func TestHarvestSurvivesRestartAndRetry(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "harvest.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "mint")
	start := checkpointAtHead(t, path, scope)
	ready := "water mint\n" + strings.Repeat("wait\n", 6)
	pick := ready + "harvest planter\n"
	if out := play(path, ready+"actions\n", ""); !strings.Contains(out, "  harvest the mint planter\n") {
		t.Fatal(out)
	}
	leafy := checkpointAtHead(t, path, scope)
	before, records := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
	execSQL(t, path, "CREATE TRIGGER fail_harvest BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted harvest'); END")
	play(path, pick, "interrupted harvest")
	if !bytes.Equal(before, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(records, inspectScope(t, path, scope)) {
		t.Fatal("failed receipt changed the plant, inventory, workers, clock or history")
	}
	execSQL(t, path, "DROP TRIGGER fail_harvest")
	out := play(path, pick+"look\njournal\n", "")
	for _, expected := range []string{"You harvest a mint sprig from the mint planter.", "You carry the mint sprig.", "I saw clipped stems on the mint planter."} {
		if strings.Count(out, expected) != 1 {
			t.Fatalf("wanted one %q after restart: %s", expected, out)
		}
	}
	cut := checkpointAtHead(t, path, scope)
	store := pick + "examine it\ndrop it\ntake sprig\neast\nwalk to crate\nopen crate\nput sprig in crate\n"
	out = play(path, store+"look\njournal\n", "")
	for _, expected := range []string{"You find the mint sprig in hand.", "You see the mint sprig in the crate.", "I harvested a mint sprig from the mint planter."} {
		if strings.Count(out, expected) != 1 {
			t.Fatalf("wanted one %q: %s", expected, out)
		}
	}
	head := checkpointAtHead(t, path, scope)
	out = play(path, store+"look\n", "")
	if strings.Count(out, "already completed") != int(head.Tick) || checkpointAtHead(t, path, scope) != head ||
		!strings.Contains(out, "You see the mint sprig in the crate.") {
		t.Fatal("retry failed to preserve the stowed sprig", out)
	}
	count := 0
	for _, event := range inspectScope(t, path, scope).events {
		if event.Kind == "plant_harvested" {
			count++
		}
	}
	if count != 1 || cut.Tick != leafy.Tick+1 {
		t.Fatal("harvest count or turn cost", count, leafy, cut)
	}
	for _, seq := range []uint64{start.Sequence, leafy.Sequence, cut.Sequence} {
		if result, err := storage.Replay(path, scope, seq); err != nil || result != head {
			t.Fatal(result, err)
		}
	}
	t.Logf("from %d, %d and %d: tick %d, sequence %d, sha256 %s", start.Sequence, leafy.Sequence, cut.Sequence, head.Tick, head.Sequence, head.Hash)
}
