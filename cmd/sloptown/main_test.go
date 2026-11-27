// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
)

var quiet = slog.New(slog.DiscardHandler)

func TestBakeryCommand(t *testing.T) {
	call := func(wantExit int, actions ...string) (string, string) {
		t.Helper()
		var out, diag bytes.Buffer
		if code := run(append([]string{"bakery", "-seed", "42"}, actions...), &out, &diag); code != wantExit {
			t.Fatalf("exit %d for %v: %s", code, actions, diag.String())
		}
		if strings.Contains(out.String(), "bakery/") || strings.Contains(out.String(), "tick") {
			t.Fatalf("terminal printed internal identifiers: %s", out.String())
		}
		return out.String(), diag.String()
	}
	initial, _ := call(0)
	if strings.Contains(initial, "crate") || strings.Contains(initial, "spoon") || !strings.Contains(initial, "@ + ?") {
		t.Fatalf("unexpected street view: %s", initial)
	}
	_, hidden := call(1, "open", "crate")
	_, missing := call(1, "open", "missing")
	if hidden == "" || hidden != missing {
		t.Fatal("name selection disclosed a hidden object")
	}
	out, _ := call(0, "-full", "open", "bakery door", "move", "east", "move", "east", "open", "crate", "take", "wooden spoon", "put", "wooden spoon", "crate")
	if !strings.Contains(out, "You carry the wooden spoon.") || !strings.Contains(out, "You put the wooden spoon in the crate.") ||
		!strings.Contains(out, "c*") {
		t.Fatalf("visible targets did not work through the terminal: %s", out)
	}
}

func TestStepCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timeline.sqlite")
	args := []string{"step", "-db", path, "-request", "r1"}
	for _, tc := range []struct {
		stream string
		exit   int
		status string
		code   string
	}{
		{"weather", 0, "accepted", ""},
		{"weather", 0, "accepted", ""},
		{"movement", 1, "rejected", "request_conflict"},
	} {
		var stdout, stderr bytes.Buffer
		exit := run(append(args, "-stream", tc.stream), &stdout, &stderr)
		var result simulation.Result
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("step returned invalid JSON: %v (stderr %q)", err, stderr.String())
		}
		if exit != tc.exit || result.RequestID != "r1" || result.Status != tc.status || result.Code != tc.code {
			t.Fatalf("unexpected step response: exit %d, %+v", exit, result)
		}
	}
}

func TestTraceOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-save.sqlite")
	args := []string{"-db", path, "-seed", "42"}
	var plain, traced, diagnostic bytes.Buffer
	if code := run(args, &plain, &diagnostic); code != 0 || diagnostic.Len() != 0 {
		t.Fatalf("plain run exited %d: %s", code, diagnostic.String())
	}
	if code := run(append(args, "-trace"), &traced, &diagnostic); code != 0 || traced.String() != plain.String() {
		t.Fatalf("traced run exited %d with stdout %q", code, traced.String())
	}
	lines := strings.Split(strings.TrimSpace(diagnostic.String()), "\n")
	if len(lines) != 2 || strings.Contains(diagnostic.String(), path) {
		t.Fatalf("unexpected trace: %s", diagnostic.String())
	}
	var runID string
	for i, msg := range []string{"fixture_committed", "fixture_verified"} {
		var record map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &record); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			runID, _ = record["run_id"].(string)
		}
		if runID == "" || record["run_id"] != runID || record["msg"] != msg || record["seed"] != float64(42) || record["world_id"] != "smoke" || record["timeline_id"] != "main" {
			t.Fatalf("unexpected %s record: %v", msg, record)
		}
	}
}

func TestTraceSaveFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	path := filepath.Join(t.TempDir(), "missing", "private-save.sqlite")
	code := run([]string{"-trace", "-db", path}, &stdout, &stderr)
	var record map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if code != 1 || stdout.Len() != 0 || record["msg"] != "fixture_failed" || record["stage"] != "save" || strings.Contains(stderr.String(), path) {
		t.Fatalf("unexpected failure report: exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestSmokePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.sqlite")
	run := func(seed int64) string {
		t.Helper()
		hash, err := smoke(path, seed, quiet)
		if err != nil {
			t.Fatal(err)
		}
		return hash
	}
	first := run(17)
	if run(18) == first || run(17) != first {
		t.Fatal("unexpected fixture hashes across runs")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE smoke_fixture SET value = -1 WHERE seed = 17"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := smoke(path, 17, quiet); err == nil {
		t.Fatal("failed to detect modified saved fixture")
	}
}

func legacyFixture(t *testing.T) (string, *sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("CREATE TABLE smoke_fixture (seed INTEGER PRIMARY KEY, value INTEGER NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO smoke_fixture VALUES (42, ?)", simulation.FixtureValue(42)); err != nil {
		t.Fatal(err)
	}
	hash, err := verifyFixture(path, 42)
	if err != nil {
		t.Fatal(err)
	}
	return path, db, hash
}

func readInt(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLegacyMigrationRunsTwice(t *testing.T) {
	path, db, original := legacyFixture(t)
	var firstSchema int
	for run := 1; run <= 2; run++ {
		hash, err := smoke(path, 42, quiet)
		if err != nil {
			t.Fatal(err)
		}
		if hash != original {
			t.Fatalf("run %d changed fixture hash to %s", run, hash)
		}
		if version := readInt(t, db, "PRAGMA user_version"); version != 4 {
			t.Fatalf("run %d stored schema version %d, want 3", run, version)
		}
		if n := readInt(t, db, "SELECT count(*) FROM fixture_metadata WHERE world_id = 'smoke' AND timeline_id = 'main'"); n != 1 {
			t.Fatalf("run %d found %d matching metadata rows, want 1", run, n)
		}
		// SQLite changes this counter when DDL modifies the schema.
		schema := readInt(t, db, "PRAGMA schema_version")
		if run == 2 && schema != firstSchema {
			t.Fatal("second run modified the schema")
		}
		firstSchema = schema
	}
}

func TestFailedSaveRollsBack(t *testing.T) {
	path, db, original := legacyFixture(t)
	// FAIL preserves earlier changes until the surrounding transaction rolls back.
	_, err := db.Exec(`CREATE TRIGGER fail_save AFTER INSERT ON smoke_fixture BEGIN
		UPDATE smoke_fixture SET value = -1 WHERE seed = 42;
		SELECT RAISE(FAIL, 'injected save failure');
	END`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := smoke(path, 43, quiet); err == nil || !strings.Contains(err.Error(), "injected save failure") {
		t.Fatalf("expected injected save failure, got %v", err)
	}
	hash, err := verifyFixture(path, 42)
	if err != nil {
		t.Fatal(err)
	}
	if hash != original || readInt(t, db, "SELECT count(*) FROM smoke_fixture") != 1 {
		t.Fatal("failed transaction changed saved fixtures")
	}
	if version := readInt(t, db, "PRAGMA user_version"); version != 0 {
		t.Fatalf("failed transaction left schema version %d, want 0", version)
	}
	if n := readInt(t, db, "SELECT count(*) FROM sqlite_schema WHERE name = 'fixture_metadata'"); n != 0 {
		t.Fatal("failed transaction left metadata table behind")
	}
	if _, err := db.Exec("DROP TRIGGER fail_save"); err != nil {
		t.Fatal(err)
	}
	if _, err := smoke(path, 43, quiet); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
}

func TestReplayCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.sqlite")
	call := func(want int, args ...string) string {
		t.Helper()
		var out, diagnostic bytes.Buffer
		if code := run(append(args, "-db", path), &out, &diagnostic); code != want {
			t.Fatalf("%v exited %d: %s", args, code, diagnostic.String())
		}
		return out.String()
	}
	call(2, "replay")
	call(0, "step", "-seed", "42", "-request", "r1")
	first := call(0, "checkpoint")
	if got := call(0, "replay", "-from", "2"); got != first {
		t.Fatal("empty replay differs from saved state")
	}
	call(0, "step", "-request", "r2")
	var report struct {
		Sequence uint64
		Tick     int64
		Hash     string `json:"sha256"`
	}
	if err := json.Unmarshal([]byte(call(0, "replay", "-from", "2")), &report); err != nil {
		t.Fatal(err)
	}
	if report.Tick != 2 || report.Sequence != 4 || report.Hash != "d508c6d2e9416be9d60c32bbc51cb36d2c2be7e13c146057d74dad10c525554d" {
		t.Fatalf("unexpected replay report %+v", report)
	}
	call(1, "replay", "-from", "3")
}
