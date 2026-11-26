// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

// runBakery shows the actor view after each committed action.
func runBakery(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown bakery", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Int64("seed", 42, "bakery layout seed")
	path := flags.String("db", "", "optional SQLite save file")
	timeline := flags.String("timeline", "main", "bakery timeline")
	batch := flags.String("batch", "", "repeat the same batch ID to retry its ordered requests")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sloptown bakery [-seed N] [-db PATH] [-timeline NAME] [-batch ID] [move DIRECTION | open NAME | close NAME | take NAME | drop NAME | put ITEM CONTAINER] ...")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	scope := simulation.Scope{WorldID: "bakery", TimelineID: *timeline}
	const player = "bakery/player"
	var view simulation.Observation
	var submit func(simulation.Proposal) (simulation.Result, bool, error)
	var err error
	if *path == "" {
		c, err := simulation.NewBakeryCoordinator(scope, *seed)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		view, err = c.Observe(player)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		submit = func(p simulation.Proposal) (simulation.Result, bool, error) { return c.Submit(player, p), false, nil }
	} else {
		view, err = storage.OpenBakery(*path, scope, *seed, player)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		submit = func(p simulation.Proposal) (simulation.Result, bool, error) {
			return storage.SubmitBakery(*path, scope, player, p)
		}
	}
	if *batch == "" {
		*batch = rand.Text()
	}
	fmt.Fprintln(stdout, view.Text+"\n"+view.Grid)
	words := flags.Args()
	for request := 1; len(words) > 0; request++ {
		action := simulation.ActionKind(words[0])
		count := 2
		switch action {
		case simulation.ActionPut:
			count = 3
		case simulation.ActionMove, simulation.ActionOpen, simulation.ActionClose, simulation.ActionTake, simulation.ActionDrop:
		default:
			flags.Usage()
			return 2
		}
		if len(words) < count {
			flags.Usage()
			return 2
		}
		var args simulation.ActionArgs
		if action == simulation.ActionMove {
			args.Direction = simulation.Direction(words[1])
		} else {
			args.Target = words[1]
			if action == simulation.ActionPut {
				args.Container = words[2]
			}
		}
		result, repeated, err := submit(simulation.Proposal{Scope: scope, Version: 1, RequestID: *batch + "/" + strconv.Itoa(request), Action: action, ByName: action != simulation.ActionMove, Args: args})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if result.Status != "accepted" {
			fmt.Fprintln(stderr, result.Message)
			return 1
		}
		if repeated {
			fmt.Fprintln(stdout, "This action was already completed.")
		} else {
			view = result.Observation
			fmt.Fprintf(stdout, "\n%s\n%s\n%s\n", result.Message, view.Text, view.Grid)
		}
		words = words[count:]
	}
	return 0
}
