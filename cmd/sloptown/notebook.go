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

// Actor inspection is read-only and separate from the player's bakery session.
func runActorInspection(command string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("db", "", "existing SQLite save")
	world := flags.String("world", "bakery", "saved world")
	timeline := flags.String("timeline", "main", "saved timeline")
	actor := flags.String("actor", "", "full actor ID")
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
	if command == "profile" {
		percent, err := storage.BakeryRestPreference(*path, scope, *actor)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Profile of %s (timeline %s)\n%d%% chance of resting in place when both spots are available.\n", *actor, *timeline, percent)
		return 0
	}
	entries, err := storage.BakeryNotebook(*path, scope, *actor)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Notebook of %s (timeline %s)\n", *actor, *timeline)
	if len(entries) == 0 {
		fmt.Fprintln(stdout, "No notebook entries.")
	}
	for _, e := range entries {
		if e.Condition != "" {
			if e.Asked {
				fmt.Fprintf(stdout, "Tick %d, event %d — you checked in on %s. Their answer was %s.\n", e.Tick, e.SourceSequence, e.Subject, e.Condition)
			} else {
				fmt.Fprintf(stdout, "Tick %d, event %d — %s checked in on you. Your answer was %s.\n", e.Tick, e.SourceSequence, e.Subject, e.Condition)
			}
			continue
		}
		if e.Action == simulation.ActionSpeak {
			fmt.Fprintf(stdout, "Tick %d, event %d — %s told you about event %d.\n", e.Tick, e.SourceSequence, e.Subject, e.Reported)
			continue
		}
		if e.Action == simulation.ActionListen {
			fmt.Fprintf(stdout, "Tick %d, event %d — heard %s.\n", e.Tick, e.SourceSequence, e.Sound.SoundPhrase())
			continue
		}
		saw := "sit down"
		if e.Action == simulation.SightHandMovement {
			where := ""
			if e.Container != "" {
				where = " near " + e.Container
			}
			fmt.Fprintf(stdout, "Tick %d, event %d — saw a hand move%s.\n", e.Tick, e.SourceSequence, where)
			continue
		}
		if e.Action == simulation.ActionIntroduce {
			fmt.Fprintf(stdout, "Tick %d, event %d — %s introduced themselves to you.\n", e.Tick, e.SourceSequence, e.Subject)
			continue
		}
		if e.Action != simulation.ActionRest {
			saw = string(e.Action) + " " + e.Target
		}
		if e.Action == simulation.ActionPut {
			saw += " in " + e.Container
		}
		if e.Action == simulation.ActionHide {
			saw = "hide behind " + e.Target
		}
		fmt.Fprintf(stdout, "Tick %d, event %d — saw %s %s.\n", e.Tick, e.SourceSequence, e.Subject, saw)
	}
	return 0
}
