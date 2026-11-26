// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/dgang0404/slop-town/internal/simulation"
)

type nounChoice struct {
	phrase  string
	ref     string
	options []simulation.ObservedTarget
}

// Both nouns come from the same actor view. An answer selects the reference
// that was displayed even if another session has since moved an object.
type bakeryCommand struct {
	simulation.Proposal
	target, container nounChoice
	locations         map[string]string
}

func prepareCommand(action simulation.ActionKind, args simulation.ActionArgs, scene simulation.Scene, last string) (bakeryCommand, error) {
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
				return cmd, errors.New("Name the object you mean by “it”.")
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
				return cmd, errors.New("You can't reach that target.")
			}
		}
	}
	return cmd, nil
}

func (cmd *bakeryCommand) question() *nounChoice {
	for _, n := range []*nounChoice{&cmd.target, &cmd.container} {
		if n.ref == "" && len(n.options) > 1 {
			return n
		}
	}
	return nil
}

func (cmd *bakeryCommand) prompt(out io.Writer, n *nounChoice) {
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
