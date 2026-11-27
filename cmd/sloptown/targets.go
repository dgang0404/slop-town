// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"fmt"
	"io"

	"github.com/dgang0404/slop-town/internal/simulation"
)

type promptOption struct {
	label, ref string
	line       simulation.SpeechLine
}

type promptChoice struct {
	phrase       string
	ref          string
	options      []promptOption
	number       int
	repeatNumber int
}

// Both nouns come from the same actor view. An answer selects the reference
// that was displayed even if another session has since moved an object.
type bakeryCommand struct {
	simulation.Proposal
	target, container, speech promptChoice
	receipt                   *simulation.Receipt
	continuation              bool
}

func (s bakerySession) command(action simulation.ActionKind, args simulation.ActionArgs, last string) (bakeryCommand, string, error) {
	if s.recall != nil {
		receipt, err := s.recall()
		if err != nil {
			return bakeryCommand{}, "", err
		}
		if cmd, ok := recordedCommand(receipt, action, args, last); ok {
			return cmd, "", nil
		}
	}
	var scene simulation.Scene
	if args.Target != "" {
		var err error
		scene, err = s.observe()
		if err != nil {
			return bakeryCommand{}, "", err
		}
	}
	cmd, rejection := prepareCommand(action, args, scene, last)
	return cmd, rejection, nil
}

// A replayed transcript uses the original answer numbers. The prompt never
// needs names or positions from the historical scene.
func recordedCommand(r *simulation.Receipt, action simulation.ActionKind, args simulation.ActionArgs, last string) (bakeryCommand, bool) {
	if r == nil || r.Proposal.Action != action {
		return bakeryCommand{}, false
	}
	p := r.Proposal
	cmd := bakeryCommand{Proposal: p, receipt: r}
	if input := r.Input; input != nil {
		if input.Target != args.Target || input.Container != args.Container ||
			(args.Target == "it" && last != p.Args.Target) || (args.Container == "it" && last != p.Args.Container) {
			return bakeryCommand{}, false
		}
		cmd.target = promptChoice{phrase: args.Target, ref: p.Args.Target, repeatNumber: input.TargetChoice}
		cmd.container = promptChoice{phrase: args.Container, ref: p.Args.Container, repeatNumber: input.ContainerChoice}
		cmd.speech = promptChoice{phrase: "what to say", repeatNumber: input.LineChoice}
		return cmd, true
	}
	// Older reference receipts only record the resolved objects. Leave them
	// to the normal visible-choice and exact-proposal retry paths.
	return cmd, (p.ByName || args.Target == "") && p.Args == args
}

func (cmd *bakeryCommand) input() *simulation.CommandInput {
	if cmd.ByName || cmd.target.phrase == "" {
		return nil
	}
	return &simulation.CommandInput{Target: cmd.target.phrase, Container: cmd.container.phrase,
		TargetChoice: cmd.target.number, ContainerChoice: cmd.container.number, LineChoice: cmd.speech.number}
}

func prepareCommand(action simulation.ActionKind, args simulation.ActionArgs, scene simulation.Scene, last string) (bakeryCommand, string) {
	cmd := bakeryCommand{Proposal: simulation.Proposal{Action: action, Args: args, ByName: args.Target != "" && action != simulation.ActionSpeak}}
	match := func(phrase string) promptChoice {
		n := promptChoice{phrase: phrase}
		if phrase == "it" {
			n.ref = last
		} else {
			for _, t := range scene.MatchingTargets(phrase) {
				n.options = append(n.options, promptOption{ref: t.Ref, label: t.Name + " " + scene.ChoiceLocation(t.Ref)})
			}
			if len(n.options) == 1 {
				n.ref = n.options[0].ref
			}
		}
		return n
	}
	cmd.target, cmd.container = match(args.Target), match(args.Container)
	for _, n := range []promptChoice{cmd.target, cmd.container} {
		if n.phrase == "it" {
			if n.ref == "" {
				return bakeryCommand{}, "Name the object you mean by “it”."
			}
			cmd.ByName = false
		}
		if len(n.options) > 1 {
			cmd.ByName = false
		}
	}
	if !cmd.ByName {
		for _, n := range []promptChoice{cmd.target, cmd.container} {
			if n.phrase != "" && n.ref == "" && len(n.options) == 0 {
				return bakeryCommand{}, "You can't reach that target."
			}
		}
	}
	return cmd, ""
}

func (cmd *bakeryCommand) question() *promptChoice {
	for _, n := range []*promptChoice{&cmd.target, &cmd.container, &cmd.speech} {
		if n.number == 0 && (len(n.options) > 1 || n.repeatNumber > 0) {
			return n
		}
	}
	return nil
}

func (cmd *bakeryCommand) prompt(out io.Writer, n *promptChoice) {
	if n.repeatNumber > 0 {
		fmt.Fprintf(out, "Repeat your original choice number for %s, or type cancel.\n", n.phrase)
		return
	}
	if n == &cmd.speech {
		fmt.Fprintln(out, "What do you say? Enter a number or type cancel.")
	} else {
		fmt.Fprintf(out, "Which %s do you mean? Enter a number or type cancel.\n", n.phrase)
	}
	for i, t := range n.options {
		fmt.Fprintf(out, "  %d. %s\n", i+1, t.label)
	}
}

func (cmd *bakeryCommand) proposal() simulation.Proposal {
	p := cmd.Proposal
	if !p.ByName {
		p.Args.Target, p.Args.Container = cmd.target.ref, cmd.container.ref
	} else if cmd.target.ref != "" {
		p.Observed = simulation.NameBindings{Target: cmd.target.ref, Container: cmd.container.ref}
	}
	return p
}

// A walk commits one step per request. Each receipt records whether this
// command should take another step, including when replaying a batch prefix.
func (s bakerySession) execute(cmd bakeryCommand) (simulation.Result, error) {
	for {
		r, err := s.act(cmd)
		if err != nil || !r.ContinueWalk {
			return r, err
		}
		cmd.receipt = nil
		cmd.continuation = true
		cmd.Observed = simulation.NameBindings{}
		cmd.ByName = false
		cmd.target.ref = r.SelectedTarget
	}
}

func (s bakerySession) loadLines(cmd *bakeryCommand) error {
	choices, err := s.dialogue(cmd.target.ref)
	if err != nil {
		return err
	}
	cmd.speech = promptChoice{phrase: "what to say"}
	for _, choice := range choices {
		cmd.speech.options = append(cmd.speech.options, promptOption{label: choice.Words, line: choice.Line})
	}
	return nil
}

// Continue with the same person, even when a namesake has moved closer.
// A batch retry asks for the recorded number without redisplaying an old scene.
func (s bakerySession) nextExchange(last bakeryCommand) (bakeryCommand, error) {
	args := simulation.ActionArgs{Target: last.target.phrase}
	if s.recall != nil {
		r, err := s.recall()
		if err != nil {
			return bakeryCommand{}, err
		}
		if r != nil {
			if next, ok := recordedCommand(r, simulation.ActionSpeak, args, last.target.ref); ok && next.target.ref == last.target.ref {
				next.target.repeatNumber = 0
				return next, nil
			}
			return bakeryCommand{}, nil
		}
	}
	next := bakeryCommand{Proposal: simulation.Proposal{Action: simulation.ActionSpeak, Args: args},
		target: promptChoice{phrase: last.target.phrase, ref: last.target.ref}}
	if err := s.loadLines(&next); err != nil {
		return bakeryCommand{}, err
	}
	if len(next.speech.options) == 0 {
		return bakeryCommand{}, nil
	}
	return next, nil
}
