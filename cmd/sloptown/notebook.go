// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

// runNotebook is a developer command, separate from the player's bakery session.
func runNotebook(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown notebook", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("db", "", "existing SQLite save")
	world := flags.String("world", "bakery", "saved world")
	timeline := flags.String("timeline", "main", "saved timeline")
	actor := flags.String("actor", "", "notebook owner's full actor ID")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	scope := simulation.Scope{WorldID: *world, TimelineID: *timeline}
	if flags.NArg() != 0 || *path == "" || *actor == "" || !scope.Valid() {
		flags.Usage()
		return 2
	}
	entries, err := storage.BakeryNotebook(*path, scope, *actor)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Notebook of %s (timeline %s)\n", *actor, *timeline)
	if len(entries) == 0 {
		fmt.Fprintln(stdout, "No firsthand sightings.")
	}
	for _, e := range entries {
		saw := "sit down"
		if e.Action != simulation.ActionRest {
			saw = string(e.Action) + " " + e.Target
		}
		fmt.Fprintf(stdout, "Tick %d, event %d — saw %s %s.\n", e.Tick, e.SourceSequence, e.Subject, saw)
	}
	return 0
}
