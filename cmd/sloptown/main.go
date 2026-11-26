// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"crypto/rand"
	"encoding/json"
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
	if len(args) > 0 {
		switch args[0] {
		case "bakery":
			return runBakery(args[1:], os.Stdin, stdout, stderr)
		case "notebook", "profile":
			return runActorInspection(args[0], args[1:], stdout, stderr)
		case "step":
			return runStep(args[1:], stdout, stderr)
		case "checkpoint", "replay":
			return runCheckpoint(args[0], args[1:], stdout, stderr)
		}
	}
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

// runStep exercises durable command resolution on the synthetic smoke/main timeline.
func runStep(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown step", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("db", "slop-town-smoke.sqlite", "local SQLite save")
	seed := flags.Int64("seed", 1, "seed for a new timeline")
	request := flags.String("request", "", "required request ID")
	stream := flags.String("stream", "weather", "fixture random stream")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *path == "" || *request == "" {
		flags.Usage()
		return 2
	}
	scope := simulation.Scope{WorldID: "smoke", TimelineID: "main"}
	proposal := simulation.Proposal{Scope: scope, Version: 1, RequestID: *request, Action: "fixture_step", Args: simulation.FixtureArgs{Stream: *stream}}
	result, err := storage.Submit(*path, scope, *seed, "player", proposal)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if result.Status != "accepted" {
		return 1
	}
	return 0
}

func runCheckpoint(command string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("db", "slop-town-smoke.sqlite", "local SQLite save")
	world := flags.String("world", "smoke", "saved world")
	timeline := flags.String("timeline", "main", "saved timeline")
	var from int64
	if command == "replay" {
		flags.Int64Var(&from, "from", -1, "required saved checkpoint sequence")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *path == "" || command == "replay" && from < 0 {
		flags.Usage()
		return 2
	}
	scope := simulation.Scope{WorldID: *world, TimelineID: *timeline}
	var report storage.ReplayReport
	var err error
	if command == "replay" {
		report, err = storage.Replay(*path, scope, uint64(from))
	} else {
		report, err = storage.SaveCheckpoint(*path, scope)
	}
	if err == nil {
		err = json.NewEncoder(stdout).Encode(report)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
