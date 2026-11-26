// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func smoke(path string, seed int64) (string, error) {
	want := simulation.FixtureValue(seed)
	if err := storage.SaveFixture(path, seed, want); err != nil {
		return "", err
	}
	return verifyFixture(path, seed)
}

func verifyFixture(path string, seed int64) (string, error) {
	got, err := storage.LoadFixture(path, seed)
	if err != nil {
		return "", err
	}
	if want := simulation.FixtureValue(seed); got != want {
		return "", fmt.Errorf("saved fixture for seed %d differs: got %d, expected %d", seed, got, want)
	}
	return simulation.FixtureHash(seed, got), nil
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
