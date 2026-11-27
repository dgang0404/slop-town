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

const bakeryHelp = "Type a direction to walk, wait to spend a turn here, or rest to sit down when available. Use walk to NAME to approach a visible destination. Use open NAME, close NAME, take NAME, drop NAME, or put ITEM in CONTAINER. Use introduce yourself to NAME or talk to NAME when someone is nearby. Short names work. Use it for the last object you acted on. Type actions for choices here, look for the current scene, calendar for today's date, journal to read your memories or quit to leave. Use journal accounts to recall who told you about a sighting, or journal leads to see which witnesses you could ask. Use journal discoveries for first sightings or journal last seen for the latest. Add a name to search either list. Use export journal for a text copy or export illustrated journal for a book you can open in a browser. Choose narration brief or narration story to change how turns read."

const leadsCommand = "journal leads"
const firstSightingsCommand = "journal discoveries"
const lastSightingsCommand = "journal last seen"

func sightingQuery(line string) (command, name string) {
	for _, cmd := range []string{firstSightingsCommand, lastSightingsCommand} {
		if line == cmd {
			return cmd, ""
		}
		if query, ok := strings.CutPrefix(line, cmd+" "); ok {
			if words := strings.Fields(query); len(words) > 1 {
				query = noun(words)
			}
			return cmd, query
		}
	}
	return "", ""
}

type bakerySession struct {
	narrator  *turnNarrator
	dialogue  func(string) ([]simulation.DialogueChoice, error)
	journal   func() ([]simulation.JournalEntry, error)
	accounts  func() ([]simulation.HeardAccount, error)
	observe   func() (simulation.Scene, error)
	available func() ([]simulation.ActionOption, error)
	recall    func() (*simulation.Receipt, error)
	act       func(bakeryCommand) (simulation.Result, error)
}

func (s bakerySession) play(in io.Reader, out io.Writer) error {
	fmt.Fprintln(out, "Enter one command per line. Ask “What can I do here?” for choices, type help for commands or quit to leave.")
	var cmd bakeryCommand
	var last string
	readKeys, closeKeys := keyboardReader(in, out)
	if readKeys != nil {
		defer closeKeys()
		fmt.Fprintln(out, "Up/Down: browse · Enter: choose · Esc: return to typing")
	}
	lines := bufio.NewScanner(in)
	for {
		var input string
		var selectedScene *simulation.Scene
		if readKeys != nil {
			var err error
			question := cmd.question()
			input, selectedScene, err = readKeys(func() ([]menuChoice, *simulation.Scene, error) { return s.menu(cmd) }, question != nil && question.repeatNumber > 0)
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
		} else {
			if !lines.Scan() {
				return lines.Err()
			}
			input = lines.Text()
		}
		line := strings.TrimSuffix(strings.Join(strings.Fields(strings.ToLower(input)), " "), ".")
		if verb, style, _ := strings.Cut(line, " "); verb == "narration" {
			switch {
			case style == "":
				fmt.Fprintf(out, "Narration is %s. Choose narration brief or narration story.\n", s.narrator.style)
			case validNarration(style):
				s.narrator.style = style
				fmt.Fprintf(out, "Narration is %s.\n", style)
			default:
				fmt.Fprintln(out, "Choose narration brief or narration story.")
			}
			continue
		}
		if command, name := sightingQuery(line); command != "" {
			entries, err := s.journal()
			if err != nil {
				return err
			}
			if err := writeSightingIndex(out, entries, name, command == lastSightingsCommand); err != nil {
				return err
			}
			continue
		}
		switch line {
		case "":
			continue
		case "cancel":
			// A retried batch may already have reached the end of this prompt.
			cmd = bakeryCommand{}
			fmt.Fprintln(out, "Cancelled.")
			continue
		case "quit", "exit":
			return nil
		case "help", "?":
			fmt.Fprintln(out, bakeryHelp)
			continue
		case "actions", "what can i do here", "what can i do here?":
			options, err := s.available()
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
		case "journal accounts", leadsCommand:
			accounts, err := s.accounts()
			if err != nil {
				return err
			}
			if line == leadsCommand {
				err = writeLeads(out, accounts)
			} else {
				err = writeAccounts(out, accounts)
			}
			if err != nil {
				return err
			}
			continue
		case "journal", "export journal", "export illustrated journal":
			entries, err := s.journal()
			if err != nil {
				return err
			}
			if line != "journal" {
				path, err := exportJournal(entries, line == "export illustrated journal")
				if err != nil {
					fmt.Fprintln(out, "Could not export your journal.", err)
				} else {
					fmt.Fprintf(out, "Saved your journal to %s.\n", path)
				}
			} else if err := writeJournal(out, entries); err != nil {
				return err
			}
			continue
		case "look", "l", "calendar":
			scene, err := s.observe()
			if err != nil {
				return err
			}
			if line == "calendar" {
				fmt.Fprintln(out, formatBakeryDate(scene.Date)+".")
			} else {
				s.showScene(out, scene)
			}
			continue
		}
		if n := cmd.question(); n != nil {
			i, err := strconv.Atoi(line)
			if n.repeatNumber > 0 {
				if err != nil || i < 1 {
					cmd.prompt(out, n)
					continue
				}
				if i != n.repeatNumber {
					fmt.Fprintln(out, "That request ID was already used for a different action.")
					cmd = bakeryCommand{}
					continue
				}
			} else if err != nil || i < 1 || i > len(n.options) {
				fmt.Fprintf(out, "Enter a number from 1 to %d or type cancel.\n", len(n.options))
				continue
			} else {
				selected := n.options[i-1]
				n.ref = selected.ref
				if n == &cmd.speech {
					cmd.Args.Speech = selected.line
				}
			}
			n.number = i
		} else {
			action, args, err := parseBakeryAction(strings.Fields(line))
			if err != nil {
				fmt.Fprintln(out, err)
				continue
			}
			var rejection string
			if selectedScene != nil {
				cmd, rejection = prepareCommand(action, args, *selectedScene, last)
			} else {
				cmd, rejection, err = s.command(action, args, last)
			}
			if err != nil {
				return err
			}
			if rejection != "" {
				fmt.Fprintln(out, rejection)
				continue
			}
		}
		if cmd.Action == simulation.ActionSpeak && cmd.question() == nil && cmd.Args.Speech == (simulation.SpeechLine{}) {
			if err := s.loadLines(&cmd); err != nil {
				return err
			}
			if len(cmd.speech.options) == 0 {
				fmt.Fprintln(out, "There is nobody you can talk to by that name here.")
				cmd = bakeryCommand{}
				continue
			}
		}
		if n := cmd.question(); n != nil {
			cmd.prompt(out, n)
			continue
		}
		result, err := s.execute(cmd)
		if err != nil {
			return err
		}
		if result.Status != "accepted" {
			fmt.Fprintln(out, result.Message)
			if result.Code == "within_reach" {
				last = cmd.target.ref
			}
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
		if result.Status == "accepted" && cmd.Action == simulation.ActionSpeak {
			next, err := s.nextExchange(cmd)
			if err != nil {
				return err
			}
			cmd = next
			if n := cmd.question(); n != nil {
				cmd.prompt(out, n)
			} else {
				fmt.Fprintln(out, "The conversation ends.")
			}
		} else {
			cmd = bakeryCommand{}
		}
	}
}

func (s bakerySession) showScene(out io.Writer, scene simulation.Scene) {
	fmt.Fprintln(out, scene.Text+"\n"+scene.Grid)
	if s.narrator != nil {
		s.narrator.previous = scene
	}
}

func actionCommand(option simulation.ActionOption) string {
	switch option.Action {
	case simulation.ActionWait, simulation.ActionRest:
		return string(option.Action)
	case simulation.ActionMove:
		return "go " + string(option.Args.Direction)
	case simulation.ActionApproach:
		return "walk to the " + option.Args.Target
	case simulation.ActionSpeak:
		return "talk to the " + option.Args.Target
	case simulation.ActionIntroduce:
		return "introduce yourself to the " + option.Args.Target
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
	case "wait", "rest":
		if len(rest) != 0 {
			return "", args, fmt.Errorf("Type %s to spend one turn here.", verb)
		}
		return simulation.ActionKind(verb), args, nil
	case "go", "walk", "move":
		if verb == "walk" && len(rest) > 0 && rest[0] == "to" {
			target := noun(rest[1:])
			if target == "" {
				return "", args, errors.New("Walk to whom or what?")
			}
			return simulation.ActionApproach, simulation.ActionArgs{Target: target}, nil
		}
		if len(rest) != 1 || direction(rest[0]) == "" {
			return "", args, errors.New("Choose north, south, east or west.")
		}
		return simulation.ActionMove, simulation.ActionArgs{Direction: direction(rest[0])}, nil
	case "talk":
		if len(rest) < 2 || rest[0] != "to" || noun(rest[1:]) == "" {
			return "", args, errors.New("Use talk to NAME.")
		}
		return simulation.ActionSpeak, simulation.ActionArgs{Target: noun(rest[1:])}, nil
	case "introduce":
		if len(rest) < 3 || rest[0] != "yourself" || rest[1] != "to" || noun(rest[2:]) == "" {
			return "", args, errors.New("Use introduce yourself to NAME.")
		}
		return simulation.ActionIntroduce, simulation.ActionArgs{Target: noun(rest[2:])}, nil
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
