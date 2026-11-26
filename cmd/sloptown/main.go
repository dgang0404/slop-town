// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"crypto/sha256"
	"database/sql"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"

	_ "modernc.org/sqlite"
)

// Each seed owns one repeatable smoke-test fixture.
func fixture(seed int64) int64 {
	rng := rand.New(rand.NewPCG(uint64(seed), 0))
	return rng.Int64()
}

func save(path string, seed, value int64) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("CREATE TABLE IF NOT EXISTS smoke_fixture (seed INTEGER PRIMARY KEY, value INTEGER NOT NULL)"); err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO smoke_fixture(seed, value) VALUES (?, ?) ON CONFLICT(seed) DO NOTHING", seed, value); err != nil {
		return err
	}
	return tx.Commit()
}

func smoke(path string, seed int64) (string, error) {
	want := fixture(seed)
	if err := save(path, seed, want); err != nil {
		return "", err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return "", err
	}
	defer db.Close()
	var got int64
	if err := db.QueryRow("SELECT value FROM smoke_fixture WHERE seed = ?", seed).Scan(&got); err != nil {
		return "", err
	}
	if got != want {
		return "", fmt.Errorf("saved fixture for seed %d differs: got %d, expected %d", seed, got, want)
	}
	state := fmt.Sprintf("fixture-v1/%d/%d", seed, got)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(state))), nil
}

func main() {
	path := flag.String("db", "slop-town-smoke.sqlite", "local SQLite fixture file")
	seed := flag.Int64("seed", 1, "fixture seed")
	flag.Parse()
	if flag.NArg() != 0 || *path == "" {
		flag.Usage()
		os.Exit(2)
	}
	hash, err := smoke(*path, *seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("seed=%d sha256=%s\n", *seed, hash)
}
