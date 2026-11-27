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
	style    string
	bold     bool
	walking  bool // A live step of this command has already been narrated.
}

func validNarration(style string) bool {
	return style == "brief" || style == "story"
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
	target := current.ActionTarget
	prior := n.previous
	message, news, _ := strings.Cut(turn.Message, "\n")
	continuing := cmd.continuation && n.walking
	var lines []string
	if !continuing {
		if opening := n.actionOpening(cmd, target, current, message); opening != "" {
			lines = append(lines, opening)
		}
	}
	for line := range strings.SplitSeq(news, "\n") {
		// Name a lost destination once below, using its last visible description.
		if line != "" && !(line == "You lose sight of your destination." && prior.Details[target].Label != "") {
			lines = append(lines, n.mark(line))
		}
	}
	if place := current.Self.Place; place != "" && place != prior.Self.Place {
		switch place {
		case simulation.InBakery:
			lines = append(lines, "You enter "+n.mark("the bakery")+".")
		case simulation.InBakeryDoorway:
			lines = append(lines, "You reach "+n.mark("the bakery doorway")+".")
		case simulation.OnStreet:
			lines = append(lines, "You step onto "+n.mark("the street")+".")
		default:
			lines = append(lines, n.mark(place.Description()))
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
	var revealed []string
	for _, obj := range current.Targets {
		now := current.Details[obj.Ref]
		was, seen := prior.Details[obj.Ref]
		if !seen && opened != "" && now.Holder == opened {
			item := n.mark(indefiniteName(obj.Name))
			if n.style == "story" {
				revealed = append(revealed, item)
			} else {
				lines = append(lines, "Inside lies "+item+".")
			}
		} else if detail := n.describeTarget(obj, current, explained[obj.Ref]); detail != "" {
			lines = append(lines, detail)
		}
		if now.Recognition != "" && now.Recognition != was.Recognition && !introduced[obj.Ref] {
			lines = append(lines, n.mark(now.Recognition))
		}
		if now.Expression != "" && now.Expression != was.Expression {
			lines = append(lines, n.mark(now.Expression))
		}
	}
	if len(revealed) > 0 {
		// Opening a container is a single action. Keep the discovery beside its cause.
		lines[0] = strings.TrimSuffix(message, ".") + " and find " + proseList(revealed) + " inside."
	}
	lines = append(lines, n.lostFromView(current)...)
	if len(lines) == 0 && !continuing {
		if cmd.Action == simulation.ActionMove {
			lines = append(lines, "One step "+n.mark(string(cmd.Args.Direction))+".")
		} else {
			lines = append(lines, message)
		}
	}
	if len(lines) != 0 {
		if !continuing {
			fmt.Fprintln(out)
		}
		if len(lines) == 1 && lines[0] == message && !continuing && cmd.Action != simulation.ActionWait && cmd.Action != simulation.ActionRest {
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
	if was.Condition == simulation.SittingDown && now.Condition != simulation.SittingDown {
		lines = append(lines, n.mark("You get back on your feet."))
	}
	if now.Condition == simulation.SittingDown && was.Condition != simulation.SittingDown {
		lines = append(lines, n.mark("You sit down."))
	} else if now.Condition != "" && now.Condition != was.Condition {
		lines = append(lines, n.mark(now.Condition.Description()))
	}
	if now.Feeling != was.Feeling {
		feeling := now.Feeling.Description()
		if feeling == "" {
			feeling = "The feeling of relief passes."
		}
		lines = append(lines, n.mark(feeling))
	}
	return lines
}

func (n *turnNarrator) describeTarget(obj simulation.ObservedTarget, scene simulation.Scene, explained bool) string {
	now := scene.Details[obj.Ref]
	was, seen := n.previous.Details[obj.Ref]
	subject := "The " + obj.Name
	switch {
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
		if direction, ok := strings.CutPrefix(now.Location, "to the "); n.style == "story" && ok {
			subject = "To the " + direction + ", the " + obj.Name
		}
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
	case n.style == "story" && now.Holder == "" && was.Holder == "" &&
		squaredDistance(n.previous.Self.Cell, was.Cell) > 1 && squaredDistance(scene.Self.Cell, now.Cell) == 1:
		// The observer has stepped alongside something they could already see.
		return "The " + now.Label + " is " + n.mark("beside you now") + "."
	}
	return ""
}

func (n *turnNarrator) lostFromView(scene simulation.Scene) []string {
	var lines []string
	var names []string
	var lastPlace string
	flush := func() {
		if len(names) > 0 {
			lines = append(lines, "You lose sight of "+proseList(names)+", last seen "+lastPlace+".")
			names = names[:0]
		}
	}
	for _, obj := range n.previous.Targets {
		if _, visible := scene.Details[obj.Ref]; visible {
			continue
		}
		name := "the " + obj.Name
		where := n.previous.Details[obj.Ref].Location
		ambiguous := len(n.previous.MatchingTargets(obj.Name)) > 1 || len(scene.MatchingTargets(obj.Name)) > 0
		if n.style == "story" && scene.Self.Cell != n.previous.Self.Cell {
			switch {
			case where == "here":
				where = "where you stood before your last step"
			case strings.HasPrefix(where, "to the "):
				where += " before your last step"
			}
		}
		if n.style == "story" && !ambiguous {
			if where != lastPlace {
				flush()
				lastPlace = where
			}
			names = append(names, n.mark(name))
			continue
		}
		flush()
		if ambiguous {
			name += " " + where
		}
		lines = append(lines, "You lose sight of "+n.mark(name)+".")
	}
	flush()
	return lines
}

func indefiniteName(name string) string {
	first, _ := utf8.DecodeRuneInString(name)
	if strings.ContainsRune("aeiou", unicode.ToLower(first)) {
		return "an " + name
	}
	return "a " + name
}

func proseList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

// Movement and waiting need a separate acknowledgement only on a quiet turn.
func (n *turnNarrator) actionOpening(cmd bakeryCommand, target string, scene simulation.Scene, outcome string) string {
	switch cmd.Action {
	case simulation.ActionMove, simulation.ActionWait:
		return ""
	case simulation.ActionRest:
		if scene.Self.Condition == simulation.SittingDown && n.previous.Self.Condition != simulation.SittingDown {
			return ""
		}
	case simulation.ActionTake:
		was := n.previous.Details[target]
		now := scene.Details[target]
		if n.style == "story" && now.Holder == scene.ObserverID && was.Holder != "" && was.Holder != scene.ObserverID {
			return "You " + n.mark("pick up") + " the " + now.Label + " you saw " + was.Location + "."
		}
	}
	return outcome
}

func squaredDistance(a, b simulation.CellPosition) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}
