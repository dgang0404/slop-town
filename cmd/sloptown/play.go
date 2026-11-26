// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/dgang0404/slop-town/internal/simulation"
)

const bakeryHelp = "Type a direction, or use open NAME, close NAME, take NAME, drop NAME, or put ITEM in CONTAINER. Use the names in your view. Type look to see it again or quit to leave."

func playBakery(in io.Reader, out io.Writer, look func() error, act func(simulation.ActionKind, simulation.ActionArgs) (string, error)) error {
	fmt.Fprintln(out, "Enter one command per line. Type help for commands or quit to leave.")
	lines := bufio.NewScanner(in)
	for lines.Scan() {
		line := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(lines.Text())), ".")
		switch line {
		case "":
			continue
		case "quit", "exit":
			return nil
		case "help", "?":
			fmt.Fprintln(out, bakeryHelp)
			continue
		case "look", "l":
			if err := look(); err != nil {
				return err
			}
			continue
		}
		action, args, err := parseBakeryAction(strings.Fields(line))
		if err != nil {
			fmt.Fprintln(out, err)
			continue
		}
		rejection, err := act(action, args)
		if err != nil {
			return err
		}
		if rejection != "" {
			fmt.Fprintln(out, rejection)
		}
	}
	return lines.Err()
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

// Parse syntax only. The proposal retains the player's noun phrases so the
// simulation resolves both targets against the acting character's current view.
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
