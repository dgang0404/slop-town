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
	full := flags.Bool("full", false, "show a full scene and grid after each positional action")
	var style string
	flags.Func("narration", "choose brief or story narration in play", func(value string) error {
		if !validNarration(value) {
			return errors.New("choose brief or story")
		}
		style = value
		return nil
	})
	resident := flags.Bool("resident", false, "include the baker's work routine when creating a new timeline")
	practice := flags.Bool("practice", false, "create a resident bakery with use-based endurance")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sloptown bakery [-seed N] [-resident] [-practice] [-db PATH] [-timeline NAME] [-batch ID] [-play [-narration brief|story] | [-full] ACTION ...]")
		fmt.Fprintln(stderr, "Actions: wait | rest | listen | move DIRECTION | open NAME | close NAME | knock NAME | examine NAME | water NAME | harvest NAME | take NAME | discreet_take NAME | drop NAME | put ITEM CONTAINER | introduce PERSON | approach NAME | hide NAME | emerge | post PERSON MESSAGE | read_letter | call NAME_OR_NUMBER")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *play && (flags.NArg() != 0 || *full) {
		flags.Usage()
		return 2
	}
	if style != "" && !*play {
		fmt.Fprintln(stderr, "-narration requires -play")
		return 2
	}
	if style == "" {
		style = "brief"
	}
	scope := simulation.Scope{WorldID: "bakery", TimelineID: *timeline}
	setup := simulation.BakerySetup{Resident: *resident, Practice: *practice}
	const player = "bakery/player"
	var session bakerySession
	if !*full {
		session.narrator = &turnNarrator{bold: terminalBold(stdout), style: style}
	}
	var submit func(simulation.Proposal, *simulation.CommandInput) (simulation.Turn, error)
	request := 1
	requestID := func() string { return *batch + "/" + strconv.Itoa(request) }
	if *path == "" {
		c, err := setup.NewCoordinator(scope, *seed)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		session.observe = func() (simulation.Scene, error) { return c.Scene(player) }
		session.dialogue = func(listener string) ([]simulation.DialogueChoice, error) { return c.DialogueChoices(player, listener) }
		session.journal = func() ([]simulation.JournalEntry, error) { return c.Journal(player) }
		session.accounts = func() ([]simulation.HeardAccount, error) { return c.Accounts(player) }
		session.available = func() ([]simulation.ActionOption, error) { return c.AvailableActions(player) }
		submit = func(p simulation.Proposal, _ *simulation.CommandInput) (simulation.Turn, error) {
			return c.SubmitTurn(player, p), nil
		}
	} else {
		session.observe = func() (simulation.Scene, error) {
			return storage.OpenBakerySceneWithSetup(*path, scope, *seed, player, setup)
		}
		session.dialogue = func(listener string) ([]simulation.DialogueChoice, error) {
			return storage.BakeryDialogueChoices(*path, scope, player, listener)
		}
		session.journal = func() ([]simulation.JournalEntry, error) { return storage.BakeryJournal(*path, scope, player) }
		session.accounts = func() ([]simulation.HeardAccount, error) { return storage.BakeryAccounts(*path, scope, player) }
		session.available = func() ([]simulation.ActionOption, error) { return storage.BakeryActions(*path, scope, player) }
		submit = func(p simulation.Proposal, input *simulation.CommandInput) (simulation.Turn, error) {
			return storage.SubmitBakeryTurn(*path, scope, player, p, input)
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
	session.showScene(stdout, scene)
	session.act = func(cmd bakeryCommand) (simulation.Result, error) {
		var turn simulation.Turn
		if cmd.receipt != nil {
			turn = simulation.Turn{Result: cmd.receipt.Result, Repeated: true}
		} else {
			p := cmd.proposal()
			p.Scope, p.Version, p.RequestID = scope, 1, requestID()
			var err error
			turn, err = submit(p, cmd.input())
			if err != nil || turn.Status != "accepted" {
				return turn.Result, err
			}
		}
		// A rejected action has no receipt. Its correction takes the same slot
		// so retrying the transcript cannot insert an extra action before it.
		request++
		if session.narrator != nil {
			session.narrator.write(stdout, cmd, turn)
		} else if turn.Repeated {
			fmt.Fprintln(stdout, "This action was already completed.")
		} else {
			fmt.Fprintf(stdout, "\n%s\n%s\n%s\n", turn.Message, turn.Observation.Text, turn.Observation.Grid)
		}
		return turn.Result, nil
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
		case simulation.ActionWait, simulation.ActionRest, simulation.ActionListen, simulation.ActionEmerge, simulation.ActionReadLetter:
			count = 1
		case simulation.ActionPut, simulation.ActionPost:
			count = 3
		case simulation.ActionCall, simulation.ActionMove, simulation.ActionOpen, simulation.ActionClose, simulation.ActionKnock, simulation.ActionExamine, simulation.ActionWater, simulation.ActionHarvest, simulation.ActionTake, simulation.ActionDiscreetTake, simulation.ActionDrop, simulation.ActionIntroduce, simulation.ActionApproach, simulation.ActionHide:
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
			} else if action == simulation.ActionPost {
				args.Body = words[2]
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
