// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func smoke(path string, seed int64, log *slog.Logger) (string, error) {
	want := simulation.FixtureValue(seed)
	if err := storage.SaveFixture(path, seed, want); err != nil {
		log.Error("fixture_failed", "stage", "save")
		return "", err
	}
	log.Info("fixture_committed")
	hash, err := verifyFixture(path, seed)
	if err != nil {
		log.Error("fixture_failed", "stage", "verify")
		return "", err
	}
	log.Info("fixture_verified", "sha256", hash)
	return hash, nil
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

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("db", "slop-town-smoke.sqlite", "local SQLite fixture file")
	seed := flags.Int64("seed", 1, "fixture seed")
	trace := flags.Bool("trace", false, "write fixture diagnostics as JSON lines to stderr")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *path == "" {
		flags.Usage()
		return 2
	}
	log := slog.New(slog.DiscardHandler)
	if *trace {
		log = slog.New(slog.NewJSONHandler(stderr, nil)).With(
			"run_id", rand.Text(), "seed", *seed, "world_id", "smoke", "timeline_id", "main",
		)
	}
	hash, err := smoke(*path, *seed, log)
	if err != nil {
		if !*trace {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	fmt.Fprintf(stdout, "seed=%d sha256=%s\n", *seed, hash)
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
