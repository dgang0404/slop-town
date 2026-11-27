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

type journalChapter struct {
	Number   int
	Title    string
	Passages []journalPassage
}

type journalPassage struct {
	Text      string
	FirstSeen []simulation.Discovery
}

const emptyJournal = "I haven't recorded anything yet."

// Both indexes use the date and setting of the character's own saved view.
func writeSightingIndex(dst io.Writer, entries []simulation.JournalEntry, name string, latest bool) error {
	out := bufio.NewWriter(dst)
	index, phrases := journalPhrases.DiscoveryIndex, journalPhrases.Discoveries
	if latest {
		index, phrases = journalPhrases.LastSeenIndex, journalPhrases.LastSeen
	}
	fmt.Fprintln(out, index.Title)
	var date simulation.BakeryDate
	var place simulation.JournalObservation
	printed := false
	for _, entry := range entries {
		sightings := entry.FirstSeen
		if latest {
			sightings = entry.LastSeen
		}
		for _, sight := range sightings {
			if name != "" && !strings.Contains(" "+strings.ToLower(sight.Target.Name)+" ", " "+name+" ") {
				continue
			}
			if !printed || entry.Date != date {
				fmt.Fprintln(out)
				fmt.Fprintln(out, formatBakeryDate(entry.Date))
				date, place = entry.Date, simulation.JournalObservation{}
			}
			if entry.Place.Text != "" && !entry.Place.SameAs(place) {
				fmt.Fprint(out, rememberedObservation(entry.Place), " ")
			}
			fmt.Fprintln(out, rememberedSighting(sight, phrases))
			printed, place = true, entry.Place
		}
	}
	if !printed {
		message := index.Empty
		if name != "" {
			message = index.NoMatch
		}
		fmt.Fprintln(out, message)
	}
	return out.Flush()
}

func writeLeads(dst io.Writer, accounts []simulation.HeardAccount) error {
	out := bufio.NewWriter(dst)
	phrases := journalPhrases.Leads
	fmt.Fprintln(out, phrases.Title)
	var date simulation.BakeryDate
	printed := false
	for _, account := range accounts {
		if !account.AwaitingWitness() {
			continue
		}
		first := account.Tellings[0]
		if !printed || date != first.Date {
			fmt.Fprintln(out)
			fmt.Fprintln(out, formatBakeryDate(first.Date))
			date = first.Date
		}
		sense := "see"
		if account.Sound != "" {
			sense = "hear"
		}
		question := strings.NewReplacer("{witness}", account.Witness, "{sense}", sense, "{act}", account.Act, "{teller}", first.Teller).Replace(phrases.Question)
		fmt.Fprintln(out, question)
		printed = true
	}
	if !printed {
		fmt.Fprintln(out, phrases.Empty)
	}
	return out.Flush()
}

func writeAccounts(dst io.Writer, accounts []simulation.HeardAccount) error {
	out := bufio.NewWriter(dst)
	fmt.Fprintln(out, "Accounts I've heard")
	if len(accounts) == 0 {
		fmt.Fprintln(out, "I haven't heard any accounts yet.")
	}
	for _, account := range accounts {
		fmt.Fprintln(out)
		verb := "saw"
		if account.Sound != "" {
			verb = "heard"
		}
		for _, note := range account.Tellings {
			fmt.Fprintf(out, "%s — The %s ", formatBakeryDate(note.Date), note.Teller)
			switch {
			case note.FromWitness:
				fmt.Fprintf(out, "told me they %s %s.\n", verb, account.Act)
			case account.OwnSighting:
				fmt.Fprintf(out, "said I'd told them I %s %s.\n", verb, account.Act)
			default:
				fmt.Fprintf(out, "said the %s %s %s.\n", account.Witness, verb, account.Act)
			}
		}
		if account.AwaitingWitness() {
			fmt.Fprintf(out, "I haven't heard it from the %s yet.\n", account.Witness)
		}
	}
	return out.Flush()
}

func writeJournal(dst io.Writer, entries []simulation.JournalEntry) error {
	out := bufio.NewWriter(dst)
	fmt.Fprintln(out, "My journal")
	if len(entries) == 0 {
		fmt.Fprintln(out, emptyJournal)
	}
	for _, chapter := range composeJournal(entries) {
		fmt.Fprintf(out, "\nChapter %d — %s\n\n", chapter.Number, chapter.Title)
		for _, passage := range chapter.Passages {
			fmt.Fprintln(out, passage.Text)
		}
	}
	return out.Flush()
}

// Both editions tell the same story. Only the illustrated one uses the
// subjects first seen in each passage to choose accompanying artwork.
func composeJournal(entries []simulation.JournalEntry) []journalChapter {
	var book []journalChapter
	addPassage := func(words string, firstSeen []simulation.Discovery) {
		chapter := &book[len(book)-1]
		chapter.Passages = append(chapter.Passages, journalPassage{words, firstSeen})
	}
	chapters := journalChapters{met: make(map[string]bool)}
	var continuingTo string
	var place, describedPlace simulation.JournalObservation
	quietWaits := 0
	flushWaits := func() {
		switch quietWaits {
		case 0:
		case 1:
			addPassage("I waited.", nil)
		default:
			addPassage("I waited a while.", nil)
		}
		quietWaits = 0
	}
	for _, entry := range entries {
		newDay := entry.Date != chapters.date
		if entry.Place.Text != "" {
			place = entry.Place
		}
		title, namesPlace := chapters.begin(entry)
		if title != "" {
			flushWaits()
			book = append(book, journalChapter{Number: chapters.number, Title: title})
			if namesPlace {
				describedPlace = place
			}
		}
		continuation := entry.Action == simulation.ActionApproach && continuingTo != "" && entry.WalkTarget == continuingTo
		continuingTo = ""
		if entry.ContinueWalk {
			continuingTo = entry.WalkTarget
		}
		if entry.Action == simulation.ActionWait && len(entry.Witnessed) == 0 && len(entry.Observations) == 0 && len(entry.FirstSeen) == 0 && entry.Conversation.With == "" {
			quietWaits++
			continue
		}
		flushWaits()
		var lines []string
		// A routine step or wait can bring news; the news carries the passage.
		if entry.WalkArrival.Ref != "" {
			lines = append(lines, strings.ReplaceAll(journalPhrases.WalkArrival.Journal, "{object}", entry.WalkArrival.Name))
		} else if entry.Outcome != "" && !continuation && entry.Action != simulation.ActionMove && entry.Action != simulation.ActionWait {
			lines = append(lines, remembered(entry.Outcome))
		} else if continuation && newDay {
			lines = append(lines, strings.Replace(remembered(entry.Outcome), "I walked toward ", "I continued toward ", 1))
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
			case simulation.SightHandMovement:
				lines = append(lines, "I saw "+sight.HandMovementPhrase()+".")
			case simulation.ActionIntroduce:
				lines = append(lines, "The "+sight.Actor.Name+" introduced themselves to me.")
			case simulation.ActionRest:
				lines = append(lines, "I saw "+who+" sit down.")
			case simulation.ActionOpen, simulation.ActionClose, simulation.ActionTake, simulation.ActionDrop, simulation.ActionPut, simulation.ActionHide:
				lines = append(lines, "I saw "+who+" "+string(sight.Action)+" "+sight.ObjectPhrase()+".")
			}
		}
		if entry.WalkEnd != "" {
			lines = append(lines, remembered(entry.WalkEnd))
		}
		for _, sight := range entry.Observations {
			if (entry.Action == simulation.ActionRest && sight.Kind == simulation.SittingDown) ||
				(entry.Action == simulation.ActionHide && sight.Kind == simulation.Crouching) {
				continue
			}
			lines = append(lines, rememberedObservation(sight))
		}
		for _, discovery := range entry.FirstSeen {
			lines = append(lines, rememberedSighting(discovery, journalPhrases.Discoveries))
		}
		if len(lines) != 0 {
			// Place the next incident, skipping uneventful steps between settings.
			if place.Text != "" && !place.SameAs(describedPlace) {
				lines = append([]string{rememberedObservation(place)}, lines...)
				describedPlace = place
			}
			addPassage(strings.Join(lines, " "), entry.FirstSeen)
		}
	}
	flushWaits()
	return book
}

type journalChapters struct {
	number int
	inside bool
	met    map[string]bool
	date   simulation.BakeryDate
}

func formatBakeryDate(date simulation.BakeryDate) string {
	return fmt.Sprintf("Day %d of %s, year %d", date.Day, date.Season, date.Year)
}

// A new date and a milestone on the same recorded turn share one heading.
func (c *journalChapters) begin(entry simulation.JournalEntry) (title string, namesPlace bool) {
	if c.number == 0 {
		title = "The visit begins"
	}
	if !c.inside && entry.Place.Kind == simulation.InBakery {
		c.inside = true
		title = "Inside the bakery"
		namesPlace = true
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
		namesPlace = false
	} else if len(names) > 0 {
		title = "Meeting " + proseList(names)
		namesPlace = false // The meeting heading leaves the arrival to the prose.
	}
	if entry.Date != c.date {
		c.date = entry.Date
		if title == "" {
			title = formatBakeryDate(entry.Date)
		} else {
			title = formatBakeryDate(entry.Date) + " · " + title
		}
	}
	if title != "" {
		c.number++
	}
	return title, namesPlace
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
	case "You rest for a turn.":
		return "I rested."
	}
	for _, verb := range []struct{ present, past string }{
		{"introduce yourself", "introduced myself"}, {"discreetly take", "discreetly took"}, {"walk", "walked"}, {"move", "moved"},
		{"open", "opened"}, {"close", "closed"}, {"take", "took"}, {"put", "put"}, {"drop", "dropped"},
		{"crouch", "crouched"}, {"stand", "stood"},
		{"feel", "felt"}, {"wait", "waited"}, {"are", "was"}, {"see", "saw"}, {"carry", "carried"},
	} {
		if tail, ok := strings.CutPrefix(sentence, "You "+verb.present); ok && (tail == "." || strings.HasPrefix(tail, " ")) {
			return "I " + verb.past + tail
		}
	}
	return sentence
}
