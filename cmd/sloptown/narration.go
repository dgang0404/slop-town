// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dgang0404/slop-town/internal/simulation"
)

type turnNarrator struct {
	previous simulation.Scene
	bold     bool
	walking  bool // A live step of this command has already been narrated.
}

func terminalBold(out io.Writer) bool {
	file, ok := out.(*os.File)
	return ok && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && supportsBold(file)
}

func (n *turnNarrator) mark(detail string) string {
	if n.bold {
		return "\x1b[1m" + detail + "\x1b[0m"
	}
	return detail
}

func (n *turnNarrator) write(out io.Writer, cmd bakeryCommand, turn simulation.Turn) {
	if turn.Repeated {
		if !cmd.continuation {
			fmt.Fprintln(out, "This action was already completed.")
		}
		n.walking = false
		return
	}
	current := turn.Scene
	target := turn.SelectedTarget
	if target == "" {
		target = cmd.target.ref
	}
	prior := n.previous
	message, news, _ := strings.Cut(turn.Message, "\n")
	continuing := cmd.continuation && n.walking
	var lines []string
	if !continuing {
		lines = append(lines, message)
	}
	for line := range strings.SplitSeq(news, "\n") {
		// Name a lost destination once below, using its last visible description.
		if line != "" && !(line == "You lose sight of your destination." && prior.Details[target].Label != "") {
			lines = append(lines, n.mark(line))
		}
	}
	lines = append(lines, n.describeSelf(prior.Self, current.Self)...)

	// An observed act accounts for the matching change in appearance.
	explained := make(map[string]bool)
	introduced := make(map[string]bool)
	switch cmd.Action {
	case simulation.ActionOpen, simulation.ActionClose, simulation.ActionPut, simulation.ActionDrop, simulation.ActionTake:
		explained[target] = true
	}
	for _, event := range current.Witnessed {
		actor := "The " + event.Actor.Name
		switch event.Action {
		case simulation.ActionIntroduce:
			// The addressed greeting is already in the action's outcome.
			introduced[event.Actor.Ref] = true
		case simulation.ActionRest:
			lines = append(lines, actor+" "+n.mark("sits down")+".")
			explained[event.Actor.Ref] = true
		case simulation.ActionOpen, simulation.ActionClose:
			lines = append(lines, actor+" "+n.mark(string(event.Action)+"s the "+event.Target.Name)+".")
			explained[event.Target.Ref] = true
		}
	}
	opened := ""
	if cmd.Action == simulation.ActionOpen {
		opened = target
	}
	for _, obj := range current.Targets {
		now := current.Details[obj.Ref]
		was := prior.Details[obj.Ref]
		if detail := n.describeTarget(obj, current, opened, explained[obj.Ref]); detail != "" {
			lines = append(lines, detail)
		}
		if now.Recognition != "" && now.Recognition != was.Recognition && !introduced[obj.Ref] {
			lines = append(lines, n.mark(now.Recognition))
		}
		if now.Expression != "" && now.Expression != was.Expression {
			lines = append(lines, n.mark(now.Expression))
		}
	}
	for _, obj := range prior.Targets {
		if _, visible := current.Details[obj.Ref]; visible {
			continue
		}
		description := "the " + obj.Name
		if len(prior.MatchingTargets(obj.Name)) > 1 || len(current.MatchingTargets(obj.Name)) != 0 {
			description += " " + prior.Details[obj.Ref].Location
		}
		lines = append(lines, "You lose sight of "+n.mark(description)+".")
	}
	if len(lines) != 0 {
		if !continuing {
			fmt.Fprintln(out)
		}
		if len(lines) == 1 && !continuing && cmd.Action != simulation.ActionWait && cmd.Action != simulation.ActionRest {
			// On an otherwise quiet turn the action itself can be the only change.
			if words, ok := strings.CutPrefix(message, "You "); ok {
				verb, rest, _ := strings.Cut(words, " ")
				lines[0] = "You " + n.mark(verb) + " " + rest
			}
		}
		fmt.Fprintln(out, strings.Join(lines, " "))
	}
	n.previous, n.walking = current, turn.ContinueWalk
}

func (n *turnNarrator) describeSelf(was, now simulation.ObserverDetail) []string {
	var lines []string
	if now.Place != was.Place {
		lines = append(lines, n.mark(now.Place))
	}
	if was.Seated && !now.Seated {
		lines = append(lines, n.mark("You get back on your feet."))
	}
	if now.Condition != "" && now.Condition != was.Condition {
		lines = append(lines, n.mark(now.Condition))
	}
	if now.Feeling != was.Feeling {
		feeling := now.Feeling
		if feeling == "" {
			feeling = "The feeling of relief passes."
		}
		lines = append(lines, n.mark(feeling))
	}
	return lines
}

func (n *turnNarrator) describeTarget(obj simulation.ObservedTarget, scene simulation.Scene, opened string, explained bool) string {
	now := scene.Details[obj.Ref]
	was, seen := n.previous.Details[obj.Ref]
	subject := "The " + obj.Name
	switch {
	case !seen && opened != "" && now.Holder == opened:
		article := "a "
		first, _ := utf8.DecodeRuneInString(obj.Name)
		if strings.ContainsRune("aeiou", unicode.ToLower(first)) {
			article = "an "
		}
		return "Inside lies " + n.mark(article+obj.Name) + "."
	case !seen && now.Holder == scene.ObserverID:
		return "You carry " + n.mark("the "+obj.Name) + "."
	case !seen:
		return "You see " + n.mark("the "+now.Label) + " " + now.Location + "."
	case !explained && was.Holder != now.Holder:
		return subject + " is now " + n.mark(now.Location) + "."
	case !explained && was.State != now.State && now.State != "":
		return subject + " is now " + n.mark(now.State) + "."
	case now.Holder == "" && was.Holder == "" && now.Cell != was.Cell:
		if now.Location != was.Location {
			return subject + " is now " + n.mark(now.Location) + "."
		}
		// A resident can move closer while staying in the same compass direction.
		before := squaredDistance(n.previous.Self.Cell, was.Cell)
		after := squaredDistance(scene.Self.Cell, now.Cell)
		switch {
		case after < before:
			return subject + " " + n.mark("draws closer") + "."
		case after > before:
			return subject + " " + n.mark("moves farther away") + "."
		default:
			return subject + " " + n.mark("takes a step") + "."
		}
	}
	return ""
}

func squaredDistance(a, b simulation.CellPosition) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}
