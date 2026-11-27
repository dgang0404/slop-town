// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestSecondHarvestSurvivesRestartAndRetry(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "regrowth.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	playFirst := newAcceptancePlayer(t, "regrowth-first")
	playWait := newAcceptancePlayer(t, "regrowth-wait")
	playRetry := newAcceptancePlayer(t, "regrowth-retry")
	start := checkpointAtHead(t, path, scope)
	ready := "water mint\n" + strings.Repeat("wait\n", 6) + "harvest planter\n"
	regrow := strings.Repeat("wait\n", 6)
	if out := playFirst(path, ready, ""); !strings.Contains(out, "cut back to short stems") {
		t.Fatal(out)
	}
	firstCut := checkpointAtHead(t, path, scope)
	if out := playWait(path, regrow, ""); strings.Count(out, "putting out new leaves") != 1 {
		t.Fatal("missing regrowth", out)
	}
	execSQL(t, path, "CREATE TRIGGER fail_second BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted second harvest'); END")
	failHarvest := newAcceptancePlayer(t, "regrowth-fail")
	failHarvest(path, "harvest planter\n", "interrupted second harvest")
	execSQL(t, path, "DROP TRIGGER fail_second")
	out := playRetry(path, "harvest planter\nlook\n", "")
	if strings.Count(out, "You harvest a mint sprig from the mint planter.") != 1 {
		t.Fatal(out)
	}
	if strings.Count(out, "You carry the mint sprig.") < 2 {
		t.Fatal("expected both sprigs in hand", out)
	}
	head := checkpointAtHead(t, path, scope)
	for _, seq := range []uint64{start.Sequence, firstCut.Sequence} {
		if result, err := storage.Replay(path, scope, seq); err != nil || result != head {
			t.Fatal(result, err)
		}
	}
	t.Logf("from %d and %d: tick %d, sequence %d, sha256 %s", start.Sequence, firstCut.Sequence, head.Tick, head.Sequence, head.Hash)
}