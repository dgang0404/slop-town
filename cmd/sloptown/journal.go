// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/dgang0404/slop-town/internal/simulation"
)

// Use one plain-text edition for the prompt, the exported file and stdout.
func writeJournal(dst io.Writer, entries []simulation.JournalEntry) error {
	out := bufio.NewWriter(dst)
	fmt.Fprintln(out, "My journal")
	if len(entries) == 0 {
		fmt.Fprintln(out, "I haven't recorded anything yet.")
		return out.Flush()
	}
	chapters := journalChapters{met: make(map[string]bool)}
	var continuingTo string
	quietWaits := 0
	flushWaits := func() {
		switch quietWaits {
		case 0:
		case 1:
			fmt.Fprintln(out, "I waited.")
		default:
			fmt.Fprintln(out, "I waited a while.")
		}
		quietWaits = 0
	}
	for _, entry := range entries {
		if title := chapters.begin(entry); title != "" {
			flushWaits()
			fmt.Fprintf(out, "\nChapter %d — %s\n\n", chapters.number, title)
		}
		continuation := entry.Action == simulation.ActionApproach && continuingTo != "" && entry.WalkTarget == continuingTo
		continuingTo = ""
		if entry.ContinueWalk {
			continuingTo = entry.WalkTarget
		}
		if entry.Action == simulation.ActionWait && len(entry.Witnessed) == 0 && len(entry.Observations) == 0 {
			quietWaits++
			continue
		}
		flushWaits()
		var lines []string
		if entry.Outcome != "" && !continuation {
			lines = append(lines, remembered(entry.Outcome))
		}
		if speech := entry.Conversation; speech.With != "" {
			if speech.IStarted {
				lines = append(lines, "I said to the "+speech.With+", “"+speech.Words+"”", "The "+speech.With+" replied, “"+speech.Reply+"”")
			} else {
				lines = append(lines, "The "+speech.With+" said to me, “"+speech.Words+"”", "I replied, “"+speech.Reply+"”")
			}
		}
		for _, sight := range entry.Witnessed {
			who := "the " + sight.Actor.Name
			switch sight.Action {
			case simulation.ActionIntroduce:
				lines = append(lines, "The "+sight.Actor.Name+" introduced themselves to me.")
			case simulation.ActionRest:
				lines = append(lines, "I saw "+who+" sit down.")
			case simulation.ActionOpen, simulation.ActionClose:
				lines = append(lines, "I saw "+who+" "+string(sight.Action)+" the "+sight.Target.Name+".")
			}
		}
		if entry.WalkEnd != "" {
			lines = append(lines, remembered(entry.WalkEnd))
		}
		for _, sight := range entry.Observations {
			lines = append(lines, remembered(sight))
		}
		if len(lines) != 0 {
			fmt.Fprintln(out, strings.Join(lines, " "))
		}
	}
	flushWaits()
	return out.Flush()
}

type journalChapters struct {
	number int
	inside bool
	met    map[string]bool
}

// A chapter starts on the event itself. Later visits and greetings can
// extend the account without moving an earlier chapter boundary.
func (c *journalChapters) begin(entry simulation.JournalEntry) string {
	var title string
	if c.number == 0 {
		title = "The visit begins"
	}
	if !c.inside && slices.Contains(entry.Observations, "You are in the bakery.") {
		c.inside = true
		title = "Inside the bakery"
	}
	var names []string
	var sameName bool
	firstIntroduction := func(person simulation.ObservedTarget) {
		if person.Ref != "" && !c.met[person.Ref] {
			c.met[person.Ref] = true
			name := "the " + person.Name
			sameName = sameName || slices.Contains(names, name)
			names = append(names, name)
		}
	}
	firstIntroduction(entry.IntroducedTo)
	for _, sight := range entry.Witnessed {
		if sight.Action == simulation.ActionIntroduce {
			firstIntroduction(sight.Actor)
		}
	}
	if sameName {
		title = "Introductions"
	} else if len(names) > 0 {
		title = "Meeting " + proseList(names)
	}
	if title != "" {
		c.number++
	}
	return title
}

// These are the sentences in a saved action outcome or observation. Change
// their perspective while keeping the original visible names and locations.
func remembered(sentence string) string {
	switch sentence {
	case "You pause your walk.":
		return "I paused my walk."
	case "Your destination moved. You pause.":
		return "My destination moved, so I paused."
	case "You lose sight of your destination.":
		return "I lost sight of my destination."
	case "Your legs feel tired.":
		return "My legs felt tired."
	case "A little of that relief lingers.":
		return "A little of that relief lingered."
	case "You rest for a turn.":
		return "I rested."
	}
	for _, verb := range []struct{ present, past string }{
		{"introduce yourself", "introduced myself"}, {"walk", "walked"}, {"move", "moved"},
		{"open", "opened"}, {"close", "closed"}, {"take", "took"}, {"put", "put"}, {"drop", "dropped"},
		{"feel", "felt"}, {"wait", "waited"}, {"are", "was"}, {"see", "saw"}, {"carry", "carried"},
	} {
		if tail, ok := strings.CutPrefix(sentence, "You "+verb.present); ok && (tail == "." || strings.HasPrefix(tail, " ")) {
			return "I " + verb.past + tail
		}
	}
	return sentence
}
