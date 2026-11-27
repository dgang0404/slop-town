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

func TestMintGrowsWhilePlayerIsInside(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "mint.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "mint")
	start := checkpointAtHead(t, path, scope)
	interrupt := func(commands string) {
		t.Helper()
		before, events := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
		execSQL(t, path, "CREATE TRIGGER fail_plant BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted plant turn'); END")
		play(path, commands, "interrupted plant turn")
		if !bytes.Equal(before, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(events, inspectScope(t, path, scope)) {
			t.Fatal("interrupted plant turn changed the town or its history")
		}
		execSQL(t, path, "DROP TRIGGER fail_plant")
	}
	const water = "water mint\n"
	const inside = water + "east\neast\nsouth\nwait\nwait\n"
	const growth = inside + "wait\n"
	interrupt(water)
	if out := play(path, water, ""); !strings.Contains(out, "You water the mint planter using its spout.") {
		t.Fatal(out)
	}
	watered := checkpointAtHead(t, path, scope)
	if out := play(path, water+"actions\nlook\nwater mint\njournal\n", ""); !strings.Contains(out, "The soil is already damp.") ||
		strings.Contains(out, "  water the mint planter\n") || checkpointAtHead(t, path, scope) != watered {
		t.Fatal("looking or trying to water again changed the town", out)
	}
	play(path, inside, "")
	beforeGrowth := checkpointAtHead(t, path, scope)
	interrupt(growth)
	const hidden = growth + "look\nactions\njournal\njournal last seen mint\n"
	out := play(path, hidden, "")
	if strings.Contains(out, "new leaves") || strings.Contains(out, "  examine the mint planter\n") ||
		!strings.Contains(out, "I last saw the damp mint planter") {
		t.Fatal("growth behind the wall leaked into the player's view", out)
	}
	leafy := checkpointAtHead(t, path, scope)
	town := inspectScope(t, path, scope)
	var wateredID string
	var count int
	for _, e := range town.events {
		switch e.Kind {
		case "plant_watered":
			wateredID = e.ID
		case "leaves_unfolded":
			count++
			if e.Tick != 7 || e.ParentID != wateredID || wateredID == "" {
				t.Fatal("growth lost its cause or its due turn", e)
			}
		}
	}
	if count != 1 {
		t.Fatal("growth events", count)
	}
	const visit = growth + "north\nlook\njournal\njournal last seen mint\n"
	out = play(path, visit, "")
	if strings.Count(out, "I saw new leaves on the mint planter.") != 1 || !strings.Contains(out, "I last saw the mint planter with new leaves") {
		t.Fatal("player did not remember finding the leaves on return", out)
	}
	head := checkpointAtHead(t, path, scope)
	if retry := play(path, visit, ""); strings.Count(retry, "already completed") != 8 || checkpointAtHead(t, path, scope) != head {
		t.Fatal(retry)
	}
	for _, seq := range []uint64{start.Sequence, watered.Sequence, beforeGrowth.Sequence, leafy.Sequence} {
		if replay, err := storage.Replay(path, scope, seq); err != nil || replay != head || replay.Tick != 8 {
			t.Fatal(replay, err)
		}
	}
	t.Logf("from %d, %d, %d and %d: tick %d, sequence %d, sha256 %s", start.Sequence, watered.Sequence, beforeGrowth.Sequence, leafy.Sequence, head.Tick, head.Sequence, head.Hash)
}
