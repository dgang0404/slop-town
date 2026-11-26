// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/dgang0404/slop-town/internal/simulation"
)

const bakeryHelp = "Type a direction, or use open NAME, close NAME, take NAME, drop NAME, or put ITEM in CONTAINER. Short names work. Use it for the last object you acted on. Type actions for choices here, look to see the scene again or quit to leave."

func playBakery(in io.Reader, out io.Writer, observe func() (simulation.Scene, error), available func() ([]simulation.ActionOption, error), act func(simulation.Proposal) (simulation.Result, error)) error {
	fmt.Fprintln(out, "Enter one command per line. Ask “What can I do here?” for choices, type help for commands or quit to leave.")
	var cmd bakeryCommand
	var last string
	lines := bufio.NewScanner(in)
	for lines.Scan() {
		line := strings.TrimSuffix(strings.Join(strings.Fields(strings.ToLower(lines.Text())), " "), ".")
		switch line {
		case "":
			continue
		case "quit", "exit":
			return nil
		case "help", "?":
			fmt.Fprintln(out, bakeryHelp)
			continue
		case "actions", "what can i do here", "what can i do here?":
			options, err := available()
			if err != nil {
				return err
			}
			if len(options) == 0 {
				fmt.Fprintln(out, "There are no actions available here.")
				continue
			}
			fmt.Fprintln(out, "Available actions")
			for _, option := range options {
				fmt.Fprintln(out, "  "+actionCommand(option))
			}
			continue
		case "look", "l":
			scene, err := observe()
			if err != nil {
				return err
			}
			fmt.Fprintln(out, scene.Text+"\n"+scene.Grid)
			continue
		}
		if n := cmd.question(); n != nil {
			if line == "cancel" {
				cmd = bakeryCommand{}
				fmt.Fprintln(out, "Cancelled.")
				continue
			}
			i, err := strconv.Atoi(line)
			if err != nil || i < 1 || i > len(n.options) {
				fmt.Fprintf(out, "Enter a number from 1 to %d or type cancel.\n", len(n.options))
				continue
			}
			n.ref = n.options[i-1].Ref
		} else {
			action, args, err := parseBakeryAction(strings.Fields(line))
			if err != nil {
				fmt.Fprintln(out, err)
				continue
			}
			var scene simulation.Scene
			if args.Target != "" {
				scene, err = observe()
				if err != nil {
					return err
				}
			}
			cmd, err = prepareCommand(action, args, scene, last)
			if err != nil {
				fmt.Fprintln(out, err)
				cmd = bakeryCommand{}
				continue
			}
		}
		if n := cmd.question(); n != nil {
			cmd.prompt(out, n)
			continue
		}
		result, err := act(cmd.proposal())
		if err != nil {
			return err
		}
		if result.Status != "accepted" {
			fmt.Fprintln(out, result.Message)
		} else if cmd.Args.Target != "" {
			last = cmd.target.ref
			if result.SelectedTarget != "" {
				last = result.SelectedTarget
			} else if cmd.ByName {
				// Older receipts identify their subject through the recorded view.
				if matches := result.Observation.MatchingTargets(cmd.Args.Target); len(matches) == 1 {
					last = matches[0].Ref
				}
			}
		}
		cmd = bakeryCommand{}
	}
	return lines.Err()
}

func actionCommand(option simulation.ActionOption) string {
	switch option.Action {
	case simulation.ActionMove:
		return "go " + string(option.Args.Direction)
	case simulation.ActionPut:
		return "put the " + option.Args.Target + " in the " + option.Args.Container
	default:
		return string(option.Action) + " the " + option.Args.Target
	}
}

func direction(word string) simulation.Direction {
	switch word {
	case "north", "n":
		return simulation.North
	case "south", "s":
		return simulation.South
	case "east", "e":
		return simulation.East
	case "west", "w":
		return simulation.West
	}
	return ""
}

func noun(words []string) string {
	if len(words) > 0 && (words[0] == "the" || words[0] == "a" || words[0] == "an") {
		words = words[1:]
	}
	return strings.Join(words, " ")
}

// Parse syntax before resolving noun phrases against the actor's view.
func parseBakeryAction(words []string) (simulation.ActionKind, simulation.ActionArgs, error) {
	var args simulation.ActionArgs
	if len(words) == 0 {
		return "", args, errors.New("Enter a command or type help.")
	}
	verb, rest := words[0], words[1:]
	if len(rest) == 0 {
		if d := direction(verb); d != "" {
			return simulation.ActionMove, simulation.ActionArgs{Direction: d}, nil
		}
	}
	switch verb {
	case "go", "walk", "move":
		if len(rest) != 1 || direction(rest[0]) == "" {
			return "", args, errors.New("Choose north, south, east or west.")
		}
		return simulation.ActionMove, simulation.ActionArgs{Direction: direction(rest[0])}, nil
	case "put":
		i := slices.IndexFunc(rest, func(w string) bool { return w == "in" || w == "into" })
		if i >= 0 {
			args.Target, args.Container = noun(rest[:i]), noun(rest[i+1:])
		}
		if args.Target == "" || args.Container == "" {
			return "", simulation.ActionArgs{}, errors.New("Use put ITEM in CONTAINER.")
		}
		return simulation.ActionPut, args, nil
	case "pick":
		if len(rest) == 0 || rest[0] != "up" {
			return "", args, errors.New("Use pick up NAME.")
		}
		verb, rest = "take", rest[1:]
	case "get":
		verb = "take"
	case "open", "close", "take", "drop":
	default:
		return "", args, errors.New("I don't understand that command. Type help for commands.")
	}
	args.Target = noun(rest)
	if args.Target == "" {
		return "", args, fmt.Errorf("What do you want to %s?", verb)
	}
	return simulation.ActionKind(verb), args, nil
}
