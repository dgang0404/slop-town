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
func runBakery(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown bakery", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Int64("seed", 42, "bakery layout seed")
	path := flags.String("db", "", "optional SQLite save file")
	timeline := flags.String("timeline", "main", "bakery timeline")
	batch := flags.String("batch", "", "repeat the same batch ID to retry its ordered requests")
	play := flags.Bool("play", false, "read one player command per line")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sloptown bakery [-seed N] [-db PATH] [-timeline NAME] [-batch ID] [-play | ACTION ...]")
		fmt.Fprintln(stderr, "Actions: move DIRECTION | open NAME | close NAME | take NAME | drop NAME | put ITEM CONTAINER")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *play && flags.NArg() != 0 {
		flags.Usage()
		return 2
	}
	scope := simulation.Scope{WorldID: "bakery", TimelineID: *timeline}
	const player = "bakery/player"
	var observe func() (simulation.Observation, error)
	var submit func(simulation.Proposal) (simulation.Result, bool, error)
	if *path == "" {
		c, err := simulation.NewBakeryCoordinator(scope, *seed)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		observe = func() (simulation.Observation, error) { return c.Observe(player) }
		submit = func(p simulation.Proposal) (simulation.Result, bool, error) { return c.Submit(player, p), false, nil }
	} else {
		observe = func() (simulation.Observation, error) {
			return storage.OpenBakery(*path, scope, *seed, player)
		}
		submit = func(p simulation.Proposal) (simulation.Result, bool, error) {
			return storage.SubmitBakery(*path, scope, player, p)
		}
	}
	if *batch == "" {
		*batch = rand.Text()
	}
	look := func() error {
		view, err := observe()
		if err == nil {
			fmt.Fprintln(stdout, view.Text+"\n"+view.Grid)
		}
		return err
	}
	if err := look(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	request := 1
	act := func(action simulation.ActionKind, args simulation.ActionArgs) (rejection string, err error) {
		result, repeated, err := submit(simulation.Proposal{
			Scope: scope, Version: 1, RequestID: *batch + "/" + strconv.Itoa(request),
			Action: action, ByName: action != simulation.ActionMove, Args: args,
		})
		if err != nil {
			return "", err
		}
		if result.Status != "accepted" {
			return result.Message, nil
		}
		// A rejected action has no receipt. Its correction takes the same slot
		// so retrying the transcript cannot insert an extra action before it.
		request++
		if repeated {
			fmt.Fprintln(stdout, "This action was already completed.")
		} else {
			fmt.Fprintf(stdout, "\n%s\n%s\n%s\n", result.Message, result.Observation.Text, result.Observation.Grid)
		}
		return "", nil
	}
	if *play {
		if err := playBakery(stdin, stdout, look, act); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	words := flags.Args()
	for len(words) > 0 {
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
		rejection, err := act(action, args)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if rejection != "" {
			fmt.Fprintln(stderr, rejection)
			return 1
		}
		words = words[count:]
	}
	return 0
}
