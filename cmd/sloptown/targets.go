// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"fmt"
	"io"

	"github.com/dgang0404/slop-town/internal/simulation"
)

type nounChoice struct {
	phrase       string
	ref          string
	options      []simulation.ObservedTarget
	number       int
	repeatNumber int
}

// Both nouns come from the same actor view. An answer selects the reference
// that was displayed even if another session has since moved an object.
type bakeryCommand struct {
	simulation.Proposal
	target, container nounChoice
	locations         map[string]string
	receipt           *simulation.Receipt
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
		cmd.target = nounChoice{phrase: args.Target, ref: p.Args.Target, repeatNumber: input.TargetChoice}
		cmd.container = nounChoice{phrase: args.Container, ref: p.Args.Container, repeatNumber: input.ContainerChoice}
		return cmd, true
	}
	// Older reference receipts only record the resolved objects. Leave them
	// to the normal visible-choice and exact-proposal retry paths.
	return cmd, (p.ByName || action == simulation.ActionMove) && p.Args == args
}

func (cmd *bakeryCommand) input() *simulation.CommandInput {
	if cmd.ByName || cmd.target.phrase == "" {
		return nil
	}
	return &simulation.CommandInput{Target: cmd.target.phrase, Container: cmd.container.phrase,
		TargetChoice: cmd.target.number, ContainerChoice: cmd.container.number}
}

func prepareCommand(action simulation.ActionKind, args simulation.ActionArgs, scene simulation.Scene, last string) (bakeryCommand, string) {
	cmd := bakeryCommand{Proposal: simulation.Proposal{Action: action, Args: args, ByName: args.Target != ""}, locations: scene.Locations}
	match := func(phrase string) nounChoice {
		n := nounChoice{phrase: phrase}
		if phrase == "it" {
			n.ref = last
		} else {
			n.options = scene.MatchingTargets(phrase)
			if len(n.options) == 1 {
				n.ref = n.options[0].Ref
			}
		}
		return n
	}
	cmd.target, cmd.container = match(args.Target), match(args.Container)
	for _, n := range []nounChoice{cmd.target, cmd.container} {
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
		for _, n := range []nounChoice{cmd.target, cmd.container} {
			if n.phrase != "" && n.ref == "" && len(n.options) == 0 {
				return bakeryCommand{}, "You can't reach that target."
			}
		}
	}
	return cmd, ""
}

func (cmd *bakeryCommand) question() *nounChoice {
	for _, n := range []*nounChoice{&cmd.target, &cmd.container} {
		if n.number == 0 && (len(n.options) > 1 || n.repeatNumber > 0) {
			return n
		}
	}
	return nil
}

func (cmd *bakeryCommand) prompt(out io.Writer, n *nounChoice) {
	if n.repeatNumber > 0 {
		fmt.Fprintf(out, "Repeat your original choice number for %s, or type cancel.\n", n.phrase)
		return
	}
	fmt.Fprintf(out, "Which %s do you mean? Enter a number or type cancel.\n", n.phrase)
	for i, t := range n.options {
		fmt.Fprintf(out, "  %d. %s %s\n", i+1, t.Name, cmd.locations[t.Ref])
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
