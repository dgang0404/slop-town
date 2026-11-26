// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/dgang0404/slop-town/internal/simulation"
)

// runBakery executes explicit actions in one in-memory session. The terminal
// only receives actor observations and resolves names against their references.
func runBakery(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown bakery", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Int64("seed", 42, "bakery layout seed")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sloptown bakery [-seed N] [move DIRECTION | open NAME | close NAME | take NAME | drop NAME | put ITEM CONTAINER] ...")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	const player = "bakery/player"
	c, err := simulation.NewBakeryCoordinator(scope, *seed)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	view, err := c.Observe(player)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
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
			args.Target, err = visibleRef(view, words[1])
			if err == nil && action == simulation.ActionPut {
				args.Container, err = visibleRef(view, words[2])
			}
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
		result := c.Submit(player, simulation.Proposal{Scope: scope, Version: 1, RequestID: strconv.Itoa(request), Action: action, Args: args})
		if result.Status != "accepted" {
			fmt.Fprintln(stderr, result.Message)
			return 1
		}
		view = result.Observation
		fmt.Fprintf(stdout, "\n%s\n%s\n%s\n", result.Message, view.Text, view.Grid)
		words = words[count:]
	}
	return 0
}

func visibleRef(view simulation.Observation, name string) (string, error) {
	var ref string
	for _, target := range view.Targets {
		if target.Name == name {
			if ref != "" {
				return "", errors.New("That name matches more than one visible target.")
			}
			ref = target.Ref
		}
	}
	if ref == "" {
		return "", errors.New("That target is out of view.")
	}
	return ref, nil
}
