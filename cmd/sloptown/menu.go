// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"strconv"

	"github.com/dgang0404/slop-town/internal/simulation"
)

type menuChoice struct{ label, value string }
type menuSource func() ([]menuChoice, *simulation.Scene, error)
type readMenuLine func(menuSource, bool) (string, *simulation.Scene, error)

func (s bakerySession) menu(cmd bakeryCommand) ([]menuChoice, *simulation.Scene, error) {
	var choices []menuChoice
	if question := cmd.question(); question != nil {
		for i, option := range question.options {
			n := strconv.Itoa(i + 1)
			choices = append(choices, menuChoice{n + ". " + option.Name + " " + cmd.details[option.Ref].Location, n})
		}
		return append(choices, menuChoice{"cancel", "cancel"}), nil, nil
	}
	scene, err := s.observe()
	if err != nil {
		return nil, nil, err
	}
	options, err := s.available()
	if err != nil {
		return nil, nil, err
	}
	var directions []menuChoice
	for _, option := range options {
		if option.Args.Target != "" && len(scene.MatchingTargets(option.Args.Target)) == 0 ||
			option.Args.Container != "" && len(scene.MatchingTargets(option.Args.Container)) == 0 {
			continue
		}
		command := actionCommand(option)
		choice := menuChoice{command, command}
		if option.Action == simulation.ActionMove {
			directions = append(directions, choice)
		} else {
			choices = append(choices, choice)
		}
	}
	choices = append(choices, directions...)
	choices = append(choices, menuChoice{"look", "look"})
	setting := "narration story"
	if s.narrator.style == "story" {
		setting = "narration brief"
	}
	return append(choices, menuChoice{setting, setting}, menuChoice{"quit", "quit"}), &scene, nil
}
