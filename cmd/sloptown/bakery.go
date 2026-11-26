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
	resident := flags.Bool("resident", false, "include the baker's work routine when creating a new timeline")
	practice := flags.Bool("practice", false, "create a resident bakery with use-based endurance")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sloptown bakery [-seed N] [-resident] [-practice] [-db PATH] [-timeline NAME] [-batch ID] [-play | ACTION ...]")
		fmt.Fprintln(stderr, "Actions: wait | rest | move DIRECTION | open NAME | close NAME | take NAME | drop NAME | put ITEM CONTAINER | introduce PERSON | approach NAME")
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
	setup := simulation.BakerySetup{Resident: *resident, Practice: *practice}
	const player = "bakery/player"
	var session bakerySession
	var submit func(simulation.Proposal, *simulation.CommandInput) (simulation.Result, bool, error)
	request := 1
	requestID := func() string { return *batch + "/" + strconv.Itoa(request) }
	if *path == "" {
		c, err := setup.NewCoordinator(scope, *seed)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		session.observe = func() (simulation.Scene, error) { return c.Scene(player) }
		session.available = func() ([]simulation.ActionOption, error) { return c.AvailableActions(player) }
		submit = func(p simulation.Proposal, _ *simulation.CommandInput) (simulation.Result, bool, error) {
			return c.Submit(player, p), false, nil
		}
	} else {
		session.observe = func() (simulation.Scene, error) {
			return storage.OpenBakerySceneWithSetup(*path, scope, *seed, player, setup)
		}
		session.available = func() ([]simulation.ActionOption, error) { return storage.BakeryActions(*path, scope, player) }
		submit = func(p simulation.Proposal, input *simulation.CommandInput) (simulation.Result, bool, error) {
			return storage.SubmitBakeryInput(*path, scope, player, p, input)
		}
		if *batch != "" && *play {
			session.recall = func() (*simulation.Receipt, error) { return storage.BakeryReceipt(*path, scope, player, requestID()) }
		}
	}
	if *batch == "" {
		*batch = rand.Text()
	}
	scene, err := session.observe()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, scene.Text+"\n"+scene.Grid)
	session.act = func(cmd bakeryCommand) (simulation.Result, error) {
		var result simulation.Result
		repeated := cmd.receipt != nil
		if repeated {
			result = cmd.receipt.Result
		} else {
			p := cmd.proposal()
			p.Scope, p.Version, p.RequestID = scope, 1, requestID()
			var err error
			result, repeated, err = submit(p, cmd.input())
			if err != nil || result.Status != "accepted" {
				return result, err
			}
		}
		// A rejected action has no receipt. Its correction takes the same slot
		// so retrying the transcript cannot insert an extra action before it.
		request++
		if repeated {
			fmt.Fprintln(stdout, "This action was already completed.")
		} else {
			fmt.Fprintf(stdout, "\n%s\n%s\n%s\n", result.Message, result.Observation.Text, result.Observation.Grid)
		}
		return result, nil
	}
	if *play {
		if err := session.play(stdin, stdout); err != nil {
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
		case simulation.ActionWait, simulation.ActionRest:
			count = 1
		case simulation.ActionPut:
			count = 3
		case simulation.ActionMove, simulation.ActionOpen, simulation.ActionClose, simulation.ActionTake, simulation.ActionDrop, simulation.ActionIntroduce, simulation.ActionApproach:
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
		} else if count > 1 {
			args.Target = words[1]
			if action == simulation.ActionPut {
				args.Container = words[2]
			}
		}
		result, err := session.execute(bakeryCommand{Proposal: simulation.Proposal{Action: action, Args: args, ByName: args.Target != ""}})
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if result.Status != "accepted" {
			fmt.Fprintln(stderr, result.Message)
			return 1
		}
		words = words[count:]
	}
	return 0
}
