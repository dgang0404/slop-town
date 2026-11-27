// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/dgang0404/slop-town/internal/simulation"
)

type personMemory struct {
	name, meeting, detail, words, lastSeen, address string
}

// Build the book from the reader's introductions, examinations, conversations
// and saved sightings. Two people with the same name keep separate entries.
func writePeople(dst io.Writer, entries []simulation.JournalEntry, query string) error {
	out := bufio.NewWriter(dst)
	fmt.Fprintln(out, "People I’ve met")
	people := make(map[string]*personMemory)
	var order []string
	meet := func(person simulation.ObservedTarget, date simulation.BakeryDate, meeting string) {
		if person.Ref == "" || people[person.Ref] != nil {
			return
		}
		people[person.Ref] = &personMemory{name: person.Name, meeting: meeting + " on " + strings.ToLower(formatBakeryDate(date)) + "."}
		order = append(order, person.Ref)
	}
	for _, page := range entries {
		meet(page.IntroducedTo, page.Date, "I introduced myself")
		for _, sight := range page.Witnessed {
			if sight.Action == simulation.ActionIntroduce {
				meet(sight.Actor, page.Date, "They introduced themselves to me")
			}
		}
	}
	// A detail noticed before the introduction can be recognized afterward.
	for _, page := range entries {
		if p := people[page.ExaminedRef]; p != nil && page.ExaminedDetail != "" {
			p.detail = page.ExaminedDetail
		}
		talk := page.Conversation
		if p := people[talk.WithRef]; p != nil {
			if talk.Address != "" {
				p.address = "Mailing address: " + talk.Address + ". Shared on " + strings.ToLower(formatBakeryDate(page.Date)) + "."
			}
			words := talk.Words
			if talk.IStarted {
				words = talk.Reply
			}
			if words != "" && talk.Address == "" {
				p.words = words
			}
		}
		for _, sight := range page.LastSeen {
			if p := people[sight.Target.Ref]; p != nil {
				p.lastSeen = rememberedSighting(sight, journalPhrases.LastSeen)
				if sight.Landmark == "" && page.Place.Text != "" {
					p.lastSeen = rememberedObservation(page.Place) + " " + p.lastSeen
				}
			}
		}
	}
	printed := false
	for _, ref := range order {
		p := people[ref]
		if query != "" && !strings.Contains(" "+strings.ToLower(p.name)+" ", " "+query+" ") {
			continue
		}
		printed = true
		fmt.Fprintf(out, "\nThe %s\n%s\n", p.name, p.meeting)
		if p.address != "" {
			fmt.Fprintln(out, p.address)
		}
		if p.detail != "" {
			fmt.Fprintf(out, "I noticed %s.\n", p.detail)
		}
		if p.words != "" {
			fmt.Fprintf(out, "They said, “%s”\n", p.words)
		}
		if p.lastSeen != "" {
			fmt.Fprintln(out, p.lastSeen)
		}
	}
	if len(order) == 0 {
		fmt.Fprintln(out, "I haven’t exchanged introductions with anyone yet.")
	} else if !printed {
		fmt.Fprintln(out, "I haven’t met anyone by that name.")
	}
	return out.Flush()
}
