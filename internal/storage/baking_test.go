// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestKneadingSurvivesRestartAndRetry(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "knead.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "knead")
	enter := "open door\ne\ne\n"
	trained := enter + "knead\nknead\nknead\n"
	start := checkpointAtHead(t, path, scope)
	before := checkpointAtHead(t, path, scope)
	execSQL(t, path, "CREATE TRIGGER fail_knead BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted knead'); END")
	play(path, enter+"knead\n", "interrupted knead")
	execSQL(t, path, "DROP TRIGGER fail_knead")
	retry := newAcceptancePlayer(t, "knead-retry")
	out := retry(path, trained, "")
	for _, want := range []string{"dense loaf", "light and springy"} {
		if strings.Count(out, want) < 1 {
			t.Fatal(out)
		}
	}
	head := checkpointAtHead(t, path, scope)
	if result, err := storage.Replay(path, scope, start.Sequence); err != nil || result != head {
		t.Fatal(result, err)
	}
	if result, err := storage.Replay(path, scope, before.Sequence); err != nil || result != head {
		t.Fatal(result, err)
	}
	t.Logf("from %d and %d: tick %d, sequence %d, sha256 %s", start.Sequence, before.Sequence, head.Tick, head.Sequence, head.Hash)
}