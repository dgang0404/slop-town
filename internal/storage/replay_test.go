// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func checkpointAtHead(t *testing.T, path string, s simulation.Scope) storage.ReplayReport {
	t.Helper()
	cp, err := storage.SaveCheckpoint(path, s)
	if err != nil {
		t.Fatal(err)
	}
	return cp
}

func TestReplayCheckpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save #1.sqlite")
	accept(t, path, "one")
	start := checkpointAtHead(t, path, scope)
	if start.Sequence != 2 || start.Tick != 1 || len(start.Hash) != 64 {
		t.Fatalf("bad checkpoint %+v", start)
	}
	if same := checkpointAtHead(t, path, scope); same != start {
		t.Fatalf("repeated checkpoint changed: %+v", same)
	}
	got, err := storage.Replay(path, scope, start.Sequence)
	if err != nil || got != start {
		t.Fatalf("empty suffix: %+v, %v", got, err)
	}
	for _, id := range []string{"two", "three"} {
		accept(t, path, id)
	}
	head := checkpointAtHead(t, path, scope)
	before := inspect(t, path)
	got, err = storage.Replay(path, scope, start.Sequence)
	const reference = "1165d6ea2e4db3b4c28814d815befc157947f358c725de78206803fd37c17ee0"
	if err != nil || got != head || got.Tick != 3 || got.Sequence != 6 || got.Hash != reference {
		t.Fatalf("replay %+v, want %+v, %v", got, head, err)
	}
	if !reflect.DeepEqual(before, inspect(t, path)) {
		t.Fatal("replay changed the save")
	}

	live, err := simulation.NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"one", "two", "three"} {
		live.Submit("actor", proposal(id))
	}
	if !reflect.DeepEqual(before.events, live.History()) {
		t.Fatal("verified history differs from uninterrupted execution")
	}
}

func TestReplayIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "branches.sqlite")
	accept(t, path, "one")
	first := checkpointAtHead(t, path, scope)
	other := simulation.Scope{WorldID: scope.WorldID, TimelineID: "other"}
	p := proposal("one")
	p.Scope = other
	r, err := storage.Submit(path, other, 42, "actor", p)
	if err != nil || r.Status != "accepted" {
		t.Fatalf("setup: %+v, %v", r, err)
	}
	second := checkpointAtHead(t, path, other)
	if second.Hash == first.Hash {
		t.Fatal("state digest omits identity")
	}
	p.RequestID = "two"
	p.Args.Stream = "movement"
	if r, err := storage.Submit(path, other, 42, "actor", p); err != nil || r.Status != "accepted" {
		t.Fatalf("second timeline step: %+v, %v", r, err)
	}
	want := checkpointAtHead(t, path, other)
	accept(t, path, "two")
	execSQL(t, path, "DELETE FROM timeline_event WHERE timeline='t' AND seq=3")
	got, err := storage.Replay(path, other, second.Sequence)
	if err != nil || got != want || got.Sequence != 4 {
		t.Fatalf("other timeline affected: %+v, %v", got, err)
	}
}

func TestReplayReportsDivergence(t *testing.T) {
	for _, tc := range []struct{ name, statement, diagnostic string }{
		{"missing", "DELETE FROM timeline_event WHERE seq=3", "sequence 3"},
		{"missing tail", "DELETE FROM timeline_event WHERE seq=4", "sequence 4"},
		{"reordered", `UPDATE timeline_event SET seq=10 WHERE seq=3; UPDATE timeline_event SET seq=3 WHERE seq=4; UPDATE timeline_event SET seq=4 WHERE seq=10`, "sequence 3"},
		{"changed draw", `UPDATE timeline_event SET event=json_set(event,'$.Payload.Value',0) WHERE seq=4`, "sequence 4"},
		{"receipt", "DELETE FROM request_receipt WHERE request='two'", "sequence 3"},
		{"event version", `UPDATE timeline_event SET event=json_set(event,'$.Version',7) WHERE seq=4`, "event version 7"},
		{"engine version", `UPDATE timeline_checkpoint SET data=json_set(data,'$.engine_version',8)`, "engine 8"},
		{"content version", `UPDATE timeline_checkpoint SET data=json_set(data,'$.content_version',9)`, "content 9"},
		{"scope", `UPDATE timeline_checkpoint SET data=json_set(data,'$.world_id','other')`, "checkpoint identity or hash mismatch"},
		{"head", `UPDATE timeline_state SET clock=json_set(clock,'$.tick',77)`, "sequence 4: replayed state hash differs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "save.sqlite")
			accept(t, path, "one")
			cp := checkpointAtHead(t, path, scope)
			accept(t, path, "two")
			execSQL(t, path, tc.statement)
			_, err := storage.Replay(path, scope, cp.Sequence)
			if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
				t.Fatalf("expected %q, got %v", tc.diagnostic, err)
			}
		})
	}
}

func TestCheckpointUpgradesVersionTwo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version2.sqlite")
	accept(t, path, "one")
	execSQL(t, path, "DROP TABLE timeline_checkpoint; PRAGMA user_version=2")
	before := inspect(t, path)
	cp := checkpointAtHead(t, path, scope)
	if cp.Sequence != 2 || !reflect.DeepEqual(before, inspect(t, path)) {
		t.Fatal("checkpoint migration changed timeline data")
	}
	if r, err := storage.Replay(path, scope, cp.Sequence); err != nil || r != cp {
		t.Fatalf("upgraded replay: %+v, %v", r, err)
	}
}

func TestReplayRejectsRepeatedCommittedRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "duplicate.sqlite")
	accept(t, path, "one")
	cp := checkpointAtHead(t, path, scope)
	accept(t, path, "two")
	accept(t, path, "three")
	execSQL(t, path, `UPDATE timeline_event SET event=json_set(event,'$.RequestID','two') WHERE seq>=5`)
	_, err := storage.Replay(path, scope, cp.Sequence)
	if err == nil || !strings.Contains(err.Error(), "sequence 5: repeated committed request") {
		t.Fatalf("expected duplicate request diagnostic, got %v", err)
	}
}
