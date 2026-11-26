// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestWalkToCrateAcrossRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	run := func(path, input string, fail bool) string {
		t.Helper()
		var out, diag bytes.Buffer
		code := runBakery([]string{"-db", path, "-batch", "visit", "-practice", "-play"}, strings.NewReader(input), &out, &diag)
		if (!fail && code != 0) || (fail && (code != 1 || !strings.Contains(diag.String(), "interrupted walk"))) {
			t.Fatalf("exit %d: %s", code, diag.String())
		}
		return out.String()
	}
	const prefix = "wait\neast\neast\neast\nsouth\nsouth\n"
	const named = prefix + "walk to the crate\nopen it\n"
	clean, manual, interrupted := filepath.Join(t.TempDir(), "clean.sqlite"), filepath.Join(t.TempDir(), "manual.sqlite"), filepath.Join(t.TempDir(), "interrupted.sqlite")
	for _, path := range []string{clean, manual, interrupted} {
		run(path, "", false)
		if _, err := storage.SaveCheckpoint(path, scope); err != nil {
			t.Fatal(err)
		}
	}
	out := run(clean, named, false)
	if strings.Count(out, "You walk toward the crate.") != 3 || !strings.Contains(out, "You open the crate.") {
		t.Fatal(out)
	}
	run(manual, prefix+"north\nnorth\nnorth\nopen crate\n", false)
	head := func(path string) storage.ReplayReport {
		t.Helper()
		r, err := storage.Replay(path, scope, 0)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	want := head(clean)
	if want.Tick != 10 || head(manual) != want {
		t.Fatalf("individual steps diverged: %+v vs %+v", head(manual), want)
	}
	db, err := sql.Open("sqlite", interrupted)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TRIGGER fail_walk BEFORE INSERT ON request_receipt WHEN NEW.request='visit/8' BEGIN SELECT RAISE(FAIL,'interrupted walk'); END`); err != nil {
		t.Fatal(err)
	}
	run(interrupted, named, true)
	if got := head(interrupted); got.Tick != 7 {
		t.Fatalf("wanted one committed walking step: %+v", got)
	}
	if _, err = db.Exec("DROP TRIGGER fail_walk"); err != nil {
		t.Fatal(err)
	}
	if out := run(interrupted, named, false); strings.Count(out, "You walk toward the crate.") != 2 || head(interrupted) != want {
		t.Fatal("resumed walk diverged", out)
	}
	if out := run(interrupted, named+"walk to crate\n", false); strings.Count(out, "already completed") != 10 || !strings.Contains(out, "already within reach") || head(interrupted) != want {
		t.Fatal(out)
	}
	t.Logf("both paths reached tick %d, sequence %d, sha256 %s", want.Tick, want.Sequence, want.Hash)
}
