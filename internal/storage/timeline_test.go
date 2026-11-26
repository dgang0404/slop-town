// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

var scope = simulation.Scope{WorldID: "w", TimelineID: "t"}

func proposal(request string) simulation.Proposal {
	return simulation.Proposal{Scope: scope, Version: 1, RequestID: request, Action: "fixture_step", Args: simulation.FixtureArgs{Stream: "weather"}}
}

func accept(t *testing.T, path, request string) simulation.Result {
	t.Helper()
	r, err := storage.Submit(path, scope, 42, "actor", proposal(request))
	if err != nil || r.Status != "accepted" {
		t.Fatalf("submit %s: %+v, %v", request, r, err)
	}
	return r
}

func execSQL(t *testing.T, path, statement string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(statement); err != nil {
		t.Fatal(err)
	}
}

type diskState struct {
	clock    string
	seq      int64
	events   []simulation.Event
	receipts []simulation.Receipt
}

func readRecords[T any](t *testing.T, db *sql.DB, scope simulation.Scope, query string) []T {
	t.Helper()
	rows, err := db.Query(query, scope.WorldID, scope.TimelineID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var records []T
	for rows.Next() {
		var data []byte
		var record T
		if err := rows.Scan(&data); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return records
}

// Read each assertion through a fresh database handle, including after faults.
func inspect(t *testing.T, path string) diskState {
	t.Helper()
	return inspectScope(t, path, scope)
}

func inspectScope(t *testing.T, path string, scope simulation.Scope) diskState {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var s diskState
	err = db.QueryRow("SELECT clock, last_sequence FROM timeline_state WHERE world=? AND timeline=?", scope.WorldID, scope.TimelineID).Scan(&s.clock, &s.seq)
	if err != nil {
		t.Fatal(err)
	}
	s.events = readRecords[simulation.Event](t, db, scope, "SELECT event FROM timeline_event WHERE world=? AND timeline=? ORDER BY seq")
	s.receipts = readRecords[simulation.Receipt](t, db, scope, "SELECT receipt FROM request_receipt WHERE world=? AND timeline=? ORDER BY actor, request")
	return s
}

func TestDurableStepRollbackAndRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save #1.sqlite")
	first := accept(t, path, "first")
	before := inspect(t, path)
	execSQL(t, path, `CREATE TRIGGER fail_receipt BEFORE INSERT ON request_receipt BEGIN
		SELECT RAISE(FAIL, 'injected at receipt'); END`)
	r, err := storage.Submit(path, scope, 42, "actor", proposal("second"))
	if err == nil || !strings.Contains(err.Error(), "injected at receipt") || r.Status == "accepted" {
		t.Fatalf("expected unpublished failure, got %+v, %v", r, err)
	}
	if after := inspect(t, path); !reflect.DeepEqual(after, before) {
		t.Fatalf("partial command survived rollback: %+v", after)
	}
	execSQL(t, path, "DROP TRIGGER fail_receipt")
	accept(t, path, "second")
	after := inspect(t, path)
	control := filepath.Join(t.TempDir(), "control.sqlite")
	accept(t, control, "first")
	accept(t, control, "second")
	if !reflect.DeepEqual(after, inspect(t, control)) {
		t.Fatal("retry diverged from a run without the fault")
	}
	memory, err := simulation.NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	memory.Submit("actor", proposal("first"))
	memory.Submit("actor", proposal("second"))
	if !reflect.DeepEqual(after.events, memory.History()) || after.seq != 4 || len(after.receipts) != 2 {
		t.Fatal("committed events or receipts differ from expected history")
	}
	if accept(t, path, "first") != first {
		t.Fatal("late retry lost original receipt")
	}
	p := proposal("first")
	p.Args.Stream = "different"
	r, err = storage.Submit(path, scope, 42, "actor", p)
	if err != nil || r.Code != "request_conflict" || !reflect.DeepEqual(after, inspect(t, path)) {
		t.Fatalf("conflicting retry changed state or result: %+v, %v", r, err)
	}
}

func TestConcurrentDurableRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contended.sqlite")
	start := make(chan struct{})
	const workers = 12
	var results [workers]simulation.Result
	var errs [workers]error
	done := make(chan struct{}, workers)
	for i := range workers {
		go func() {
			<-start
			results[i], errs[i] = storage.Submit(path, scope, 42, "actor", proposal("race"))
			done <- struct{}{}
		}()
	}
	close(start)
	for range workers {
		<-done
	}
	for i, r := range results {
		if errs[i] != nil || r.Status != "accepted" || r != results[0] {
			t.Fatalf("concurrent command %d: %+v, %v", i, r, errs[i])
		}
	}
	s := inspect(t, path)
	if s.seq != 2 || len(s.events) != 2 || s.events[1].Tick != 1 || len(s.receipts) != 1 {
		t.Fatalf("expected one committed step, got %+v", s)
	}
}

func TestVersionOneUpgrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.sqlite")
	execSQL(t, path, `CREATE TABLE smoke_fixture(seed INTEGER PRIMARY KEY, value INTEGER NOT NULL);
		CREATE TABLE fixture_metadata(id INTEGER PRIMARY KEY CHECK (id=1),world_id TEXT NOT NULL,timeline_id TEXT NOT NULL);
		INSERT INTO smoke_fixture VALUES(-5,1234);
		INSERT INTO fixture_metadata VALUES(1,'smoke','main');
		PRAGMA user_version=1;`)
	accept(t, path, "first")
	got, err := storage.LoadFixture(path, -5)
	if err != nil || got != 1234 {
		t.Fatalf("migration changed existing fixture: %d, %v", got, err)
	}
}

func TestDurableReceiptScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scopes.sqlite")
	accept(t, path, "one")
	for _, other := range []simulation.Scope{
		{WorldID: scope.WorldID, TimelineID: "branch"},
		{WorldID: "another-world", TimelineID: scope.TimelineID},
	} {
		p := proposal("one")
		p.Scope, p.Args.Stream = other, "movement"
		r, err := storage.Submit(path, other, 7, "actor", p)
		if err != nil || r.Status != "accepted" || r.Observation.Scope != other {
			t.Fatalf("failed to isolate scope %+v: %+v, %v", other, r, err)
		}
	}
	p := proposal("one")
	p.Args.Stream = "movement"
	r, err := storage.Submit(path, scope, 7, "another-actor", p)
	if err != nil || r.Status != "accepted" || r.Observation.ObserverID != "another-actor" {
		t.Fatalf("failed to isolate actor receipt: %+v, %v", r, err)
	}
	state := inspect(t, path)
	if state.seq != 4 || len(state.receipts) != 2 || len(state.events) != 4 || state.events[3].Payload.Value != 0xc952c2ba4351a4a3 {
		t.Fatal("original timeline should retain seed 42 and only its own two steps")
	}
}
