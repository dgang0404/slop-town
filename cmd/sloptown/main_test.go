// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestSmokePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.sqlite")
	run := func(seed int64) string {
		t.Helper()
		hash, err := smoke(path, seed)
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
	if _, err := smoke(path, 17); err == nil {
		t.Fatal("failed to detect modified saved fixture")
	}
}
