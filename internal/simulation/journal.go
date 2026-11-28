// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// JournalSpeech preserves the words heard by a participant.
type JournalSpeech struct {
	With         string
	WithRef      string
	Address      string // An address explicitly given to this reader.
	Number       string
	ByPhone      bool
	IStarted     bool
	Words, Reply string
}

// JournalEntry holds an actor's own action, audible dialogue, firsthand sightings
// and discoveries in the action's saved observation. Entries are in turn order.
type JournalEntry struct {
	Date           BakeryDate
	Place          JournalObservation // Location stated in the observer's own receipt, when present.
	Conversation   JournalSpeech
	Action         ActionKind
	Outcome        string
	WalkEnd        string
	WalkTarget     string
	WalkArrival    ObservedTarget
	ContinueWalk   bool
	IntroducedTo   ObservedTarget
	ExaminedRef    string // Resolved target of the character’s own examination.
	ExaminedDetail string
	OutcomeTarget  string // The action already introduces this object in the journal.
	Witnessed      []WitnessedAction
	Observations   []JournalObservation
	FirstSeen      []Discovery // From this character's saved observation.
	DiscoveryGrid  string      // The same receipt's view at the time of a first sighting.
	LastSeen       []Discovery // Latest recorded view of each target, on the turn it was seen.
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
	memory := journalMemory{latest: make(map[string]recordedSighting)}
	sightings := b.Notebooks[actor]
	seenKnocks := make(map[string]bool)
	err := visit(func(e Event) error {
		if e.Scope != scope {
			return fmt.Errorf("mismatched journal history")
		}
		if e.Tick < 0 {
			return fmt.Errorf("invalid journal tick %d", e.Tick)
		}
		if e.Kind == "tick_advanced" {
			clear(seenKnocks)
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
			message := r.Result.Message
			if page.Action == ActionApproach {
				message, page.WalkArrival = r.Result.SplitWalkArrival()
			}
			outcome, rest, _ := strings.Cut(message, "\n")
			page.Outcome = outcome
			page.WalkTarget, page.ContinueWalk = r.Result.SelectedTarget, r.Result.ContinueWalk
			if page.Action == ActionApproach && !page.ContinueWalk && page.WalkArrival.Ref == "" && rest != "" {
				// The last line records the interruption. Addressed speech
				// has its own event and is included below in the order it was heard.
				page.WalkEnd = rest[strings.LastIndexByte(rest, '\n')+1:]
			}
			page.Place, page.Observations, page.FirstSeen = memory.observe(r.Result.Observation, len(entries)-1)
			if len(page.FirstSeen) > 0 {
				page.DiscoveryGrid = r.Result.Observation.Grid
			}
			if page.Action == ActionListen {
				page.Outcome = "" // The journal keeps what the listener perceived that turn.
				if b.ContentVersion < dropSoundContentVersion {
					// Earlier timelines kept journal sounds in the listening receipt.
					for _, sound := range r.Result.Observation.Heard {
						page.Observations = append(page.Observations, JournalObservation{Kind: sound, Text: sound.Description()})
					}
				}
			}
		}

		if e.ActorID == actor {
			switch e.Kind {
			case "target_examined":
				if person := b.Entities[e.Payload.Target]; person.Kind == Actor {
					page := entryAt(e.Tick)
					page.ExaminedRef, page.ExaminedDetail = e.Payload.Target, e.Payload.PersonDetail
					if b.ContentVersion < landmarkContentVersion {
						// Older examinations kept the visible detail only in the reader's receipt.
						if finding, ok := strings.CutPrefix(page.Outcome, "You find the "+person.Name+" "); ok {
							_, detail, _ := strings.Cut(strings.TrimSuffix(finding, "."), ", ")
							page.ExaminedDetail = detail
						}
					}
				}
			case "letter_posted":
				page := entryAt(e.Tick)
				page.Action = ActionPost
				page.Outcome = "You post a letter to the " + b.Entities[e.Payload.Target].Name + " at " + e.Payload.Address + ". “" + e.Payload.Body + "”"
			case "letter_read":
				page := entryAt(e.Tick)
				page.Action = ActionReadLetter
				page.Outcome = "You read a letter from the " + b.Entities[e.Payload.Target].Name + ". “" + e.Payload.Body + "”"
			case "plant_watered":
				entryAt(e.Tick).OutcomeTarget = e.Payload.Target
			case "plant_harvested":
				entryAt(e.Tick).OutcomeTarget = e.Payload.ItemID
			}
		}
		if (e.Kind == "actors_conversed" || e.Kind == "call_answered") && (e.ActorID == actor || e.Payload.Target == actor) {
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
			page.Conversation = JournalSpeech{ByPhone: e.Kind == "call_answered", WithRef: other, With: b.Entities[other].Name, IStarted: started,
				Words: words, Reply: reply}
			if x := e.Payload.Dialogue; x.ContactGiver == other {
				page.Conversation.Address, page.Conversation.Number = x.Address, x.Number
			}
		}
		if e.Kind == "letter_delivered" && e.ActorID == actor {
			page := entryAt(e.Tick)
			page.Observations = append(page.Observations, JournalObservation{Text: "A letter arrived for me."})
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
			case e.ActorID == actor:
				target := e.Payload.Target
				entryAt(e.Tick).IntroducedTo = ObservedTarget{Ref: target, Name: b.Entities[target].Name}
			}
		}
		if len(sightings) > 0 && sightings[0].SourceSequence == e.Sequence {
			s := sightings[0]
			sightings = sightings[1:]
			if s.Action == ActionListen {
				if b.ContentVersion >= dropSoundContentVersion && !seenKnocks[e.ParentID] {
					// An impact may be heard during another character's turn.
					page := entryAt(e.Tick)
					page.Observations = append(page.Observations, JournalObservation{Kind: s.Sound, Text: s.Sound.Description()})
				}
			} else if s.Murmur {
				page := entryAt(e.Tick)
				page.Observations = append(page.Observations, JournalObservation{Text: b.murmurWords()})
			} else if s.Fragment {
				page := entryAt(e.Tick)
				page.Observations = append(page.Observations, JournalObservation{Text: b.fragmentWords()})
			} else if s.Aside {
				page := entryAt(e.Tick)
				page.Observations = append(page.Observations, JournalObservation{Text: b.asideWords(s)})
			} else if s.Overheard {
				page := entryAt(e.Tick)
				text := s.overheardCheckIn(b.Entities[s.Subject].Name, b.Entities[s.Target].Name)
				if own := b.ownAnswerJournal(actor, s); own != "" {
					text = own
				}
				page.Observations = append(page.Observations, JournalObservation{Text: text})
			} else if s.Action != ActionIntroduce && s.Action != ActionSpeak {
				page := entryAt(e.Tick)
				page.Witnessed = append(page.Witnessed, b.witnessedAction(s))
				if s.Action == ActionKnock {
					// Seeing the knock tells the whole incident. The sound
					// remains in the notebook for questions about hearing.
					seenKnocks[e.ID] = true
				}
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
	// Targets sharing a final observation keep the same reference order as
	// the saved view. Each entry supplies the setting and date of that view.
	for _, ref := range slices.Sorted(maps.Keys(memory.latest)) {
		last := memory.latest[ref]
		page := &entries[last.page]
		page.LastSeen = append(page.LastSeen, locatedDiscovery(last.target, last.description, last.place))
	}
	return entries, nil
}

type recordedSighting struct {
	page        int
	target      ObservedTarget
	description string
	plant       PlantCondition
	place       PerceptionKind
}

type journalMemory struct {
	latest map[string]recordedSighting
	self   []string
}

// A version-one receipt pairs its ordered selectable targets with one visible
// description each. Preserve the wording and location the observer saw then.
func (m *journalMemory) observe(v Observation, page int) (place JournalObservation, observations []JournalObservation, firstSeen []Discovery) {
	location, details, _ := strings.Cut(v.Text, "\n")
	place = personalObservation(location)
	var self, descriptions []string
	for line := range strings.SplitSeq(details, "\n") {
		if strings.HasPrefix(line, "You see the ") || strings.HasPrefix(line, "You carry the ") {
			descriptions = append(descriptions, line)
		} else if len(descriptions) == 0 && line != "" {
			// Personal sensations precede the visible targets.
			self = append(self, line)
			if !slices.Contains(m.self, line) {
				observations = append(observations, personalObservation(line))
			}
		}
	}
	m.self = self
	for i, target := range v.Targets {
		var description string
		if len(descriptions) == len(v.Targets) {
			description = descriptions[i]
		}
		previous, seen := m.latest[target.Ref]
		condition := v.Plants[target.Ref]
		if !seen {
			firstSeen = append(firstSeen, locatedDiscovery(target, description, v.Landmarks[target.Ref]))
		} else if condition == newLeaves && previous.plant != newLeaves {
			observations = append(observations, JournalObservation{Text: "You see new leaves on the " + target.Name + "."})
		} else if condition == clippedStems && previous.plant != clippedStems {
			observations = append(observations, JournalObservation{Text: "You see clipped stems on the " + target.Name + "."})
		}
		m.latest[target.Ref] = recordedSighting{page: page, target: target, description: description, plant: condition, place: v.Landmarks[target.Ref]}
	}
	return place, observations, firstSeen
}

func (b *Bakery) witnessedAction(s NotebookEntry) WitnessedAction {
	w := WitnessedAction{Action: s.Action, Actor: ObservedTarget{Ref: s.Subject, Name: b.Entities[s.Subject].Name}}
	if s.Target != "" {
		w.Target = ObservedTarget{Ref: s.Target, Name: b.Entities[s.Target].Name}
	}
	if s.Container != "" {
		w.Container = ObservedTarget{Ref: s.Container, Name: b.Entities[s.Container].Name}
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
