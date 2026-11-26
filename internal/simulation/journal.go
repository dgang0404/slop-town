// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
	"strings"
)

// JournalSpeech preserves the words heard by a participant.
type JournalSpeech struct {
	With         string
	IStarted     bool
	Words, Reply string
}

// JournalEntry holds an actor's own action, audible dialogue, firsthand sightings
// and discoveries in the action's saved observation. Entries are in turn order.
type JournalEntry struct {
	Date         BakeryDate
	Conversation JournalSpeech
	Action       ActionKind
	Outcome      string
	WalkEnd      string
	WalkTarget   string
	ContinueWalk bool
	IntroducedTo ObservedTarget
	Witnessed    []WitnessedAction
	Observations []string
}

// Journal reads existing history on demand. Both backends visit committed events
// in sequence and retrieve receipts only for the observer's own accepted turns.
func (b *Bakery) Journal(scope Scope, actor string, visit func(func(Event) error) error, receipt func(string) (*Receipt, error)) ([]JournalEntry, error) {
	if b == nil || b.WorldID != scope.WorldID || b.Entities[actor].Kind != Actor {
		return nil, fmt.Errorf("no journal for this character")
	}
	var entries []JournalEntry
	var lastTick int64
	entryAt := func(tick int64) *JournalEntry {
		if len(entries) == 0 || tick != lastTick {
			entries = append(entries, JournalEntry{Date: bakeryDateAt(tick)})
			lastTick = tick
		}
		return &entries[len(entries)-1]
	}
	memory := journalMemory{seen: make(map[string]bool)}
	sightings := b.Notebooks[actor]
	err := visit(func(e Event) error {
		if e.Scope != scope {
			return fmt.Errorf("mismatched journal history")
		}
		if e.Tick < 0 {
			return fmt.Errorf("invalid journal tick %d", e.Tick)
		}
		if e.Kind == "tick_advanced" && e.ActorID == actor {
			r, err := receipt(e.RequestID)
			if err != nil {
				return err
			}
			if !r.AcceptedFor(scope, actor, e.RequestID) {
				return fmt.Errorf("missing or mismatched journal receipt")
			}
			page := entryAt(e.Tick)
			page.Action = r.Proposal.Action
			outcome, rest, _ := strings.Cut(r.Result.Message, "\n")
			page.Outcome = outcome
			page.WalkTarget, page.ContinueWalk = r.Result.SelectedTarget, r.Result.ContinueWalk
			if page.Action == ActionApproach && !page.ContinueWalk && rest != "" {
				// The last line records arrival or interruption. Addressed speech
				// has its own event and is included below in the order it was heard.
				page.WalkEnd = rest[strings.LastIndexByte(rest, '\n')+1:]
			}
			page.Observations = memory.observe(r.Result.Observation)
		}

		if e.Kind == "actors_conversed" && (e.ActorID == actor || e.Payload.Target == actor) {
			page := entryAt(e.Tick)
			other := e.ActorID
			started := e.ActorID == actor
			if started {
				other = e.Payload.Target
				page.Outcome = "" // The exchange includes the actor's own words.
			}
			words, reply, err := b.exchangeLines(e)
			if err != nil {
				return err
			}
			page.Conversation = JournalSpeech{With: b.Entities[other].Name, IStarted: started,
				Words: words, Reply: reply}
		}

		// The notebook retains the first introduction for familiarity. Every
		// introduction addressed to this actor, including a repeat, was audible.
		if e.Kind == "actor_introduced" {
			switch {
			case e.Payload.Target == actor:
				page := entryAt(e.Tick)
				page.Witnessed = append(page.Witnessed, WitnessedAction{
					Action: ActionIntroduce, Actor: ObservedTarget{Ref: e.ActorID, Name: b.Entities[e.ActorID].Name},
				})
			case e.ActorID == actor && e.Tick == lastTick && len(entries) > 0 && entries[len(entries)-1].Action == ActionIntroduce:
				target := e.Payload.Target
				entries[len(entries)-1].IntroducedTo = ObservedTarget{Ref: target, Name: b.Entities[target].Name}
			}
		}
		if len(sightings) > 0 && sightings[0].SourceSequence == e.Sequence {
			s := sightings[0]
			sightings = sightings[1:]
			if s.Action != ActionIntroduce && s.Action != ActionSpeak {
				page := entryAt(e.Tick)
				page.Witnessed = append(page.Witnessed, b.witnessedAction(s))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(sightings) != 0 {
		return nil, fmt.Errorf("missing history for a journal sighting")
	}
	return entries, nil
}

type journalMemory struct {
	seen map[string]bool
	self []string
}

// A version-one receipt pairs its ordered selectable targets with one visible
// description each. Preserve the wording and location the observer saw then.
func (m *journalMemory) observe(v Observation) []string {
	var self, descriptions, observations []string
	for line := range strings.SplitSeq(v.Text, "\n") {
		if strings.HasPrefix(line, "You see the ") || strings.HasPrefix(line, "You carry the ") {
			descriptions = append(descriptions, line)
		} else if len(descriptions) == 0 && line != "" {
			// Location and personal sensations precede the visible targets.
			self = append(self, line)
			if !slices.Contains(m.self, line) {
				observations = append(observations, line)
			}
		}
	}
	m.self = self
	for i, target := range v.Targets {
		if m.seen[target.Ref] {
			continue
		}
		m.seen[target.Ref] = true
		if len(descriptions) == len(v.Targets) {
			observations = append(observations, descriptions[i])
		} else {
			observations = append(observations, "You see the "+target.Name+".")
		}
	}
	return observations
}

func (b *Bakery) witnessedAction(s NotebookEntry) WitnessedAction {
	w := WitnessedAction{Action: s.Action, Actor: ObservedTarget{Ref: s.Subject, Name: b.Entities[s.Subject].Name}}
	if s.Target != "" {
		w.Target = ObservedTarget{Ref: s.Target, Name: b.Entities[s.Target].Name}
	}
	return w
}

func (c *Coordinator) Journal(actor string) ([]JournalEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	visit := func(record func(Event) error) error {
		for _, e := range c.events {
			if err := record(e); err != nil {
				return err
			}
		}
		return nil
	}
	return c.bakery.Journal(c.scope, actor, visit, func(request string) (*Receipt, error) {
		return c.receipts[requestKey{actor, request}], nil
	})
}
