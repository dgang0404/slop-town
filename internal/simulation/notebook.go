// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"cmp"
	"fmt"
	"slices"
)

// NotebookEntry records firsthand observations and reports received in conversation.
// SourceSequence identifies the observation or exchange on the owner's timeline.
type NotebookEntry struct {
	Tick           int64          `json:"tick"`
	SourceSequence uint64         `json:"source_sequence"`
	Subject        string         `json:"subject"`
	Action         ActionKind     `json:"action"`
	Target         string         `json:"target,omitempty"`
	Reported       uint64         `json:"reported,omitempty"`  // Entry in the immediate teller's notebook.
	Condition      workCondition  `json:"condition,omitempty"` // Listener's answer during this owner's own exchange.
	Overheard      bool           `json:"overheard,omitempty"` // This owner saw the exchange without taking part.
	Open           bool           `json:"open,omitempty"`      // The question was heard and has not been answered by the person asked.
	Answered       bool           `json:"answered,omitempty"`  // This owner answered a question addressed to someone else.
	Asked          bool           `json:"asked,omitempty"`
	Container      string         `json:"container,omitempty"` // Visible destination of a put or landmark for a hand movement.
	Sound          PerceptionKind `json:"sound,omitempty"`     // The listener recalls the sound alone.
}

// Observe a take before the item enters a private inventory. A drop or put
// becomes visible on placement, before anyone else can act on the same tick.
// A qualifying discreet take records the actor's practice on its event.
func (b *Bakery) applySpatial(event *Event, effect spatialEffect, clock *Timeline) error {
	placesItem := event.Kind == "item_put" || event.Kind == "item_dropped"
	if event.Kind == "item_taken_discreetly" {
		if err := b.observeDiscreetTake(event, clock); err != nil {
			return err
		}
	} else if b.ContentVersion >= notebookContentVersion && !placesItem {
		b.recordWitnesses(*event)
	}
	switch event.Kind {
	case "actors_conversed":
		x := event.Payload.Dialogue
		if x.ContactGiver != "" {
			owner := event.ActorID
			if owner == x.ContactGiver {
				owner = event.Payload.Target
			}
			b.Contacts = append(slices.Clip(b.Contacts), sharedContact{Owner: owner, Person: x.ContactGiver, Address: x.Address, Number: x.Number, Tick: event.Tick, Sequence: event.Sequence})
		}
	case "letter_posted":
		p := event.Payload
		b.Letters = append(b.Letters, letter{Post: event.Sequence, From: event.ActorID, To: p.Target, Address: p.Address, Body: p.Body, Tick: event.Tick, ReplyTo: p.LetterSequence})
	case "letter_read":
		for i := range b.Letters {
			if b.Letters[i].Post == event.Payload.LetterSequence {
				b.Letters[i].ReadTick = event.Tick
				break
			}
		}
	case "plant_harvested":
		id := event.Payload.ItemID
		if _, exists := b.Entities[id]; exists {
			return fmt.Errorf("harvested item %q already exists", id)
		}
		b.Entities[id] = SpatialEntity{Kind: Item, Name: "mint sprig", Portable: true, Placement: Placement{HolderID: event.Payload.HolderID}}
		if b.ContentVersion >= regrowthContentVersion {
			plant := effect.entity
			plant.Growth.HarvestTick = event.Tick
			plant.Growth.HarvestSequence = event.Sequence
			effect.entity = plant
		}
	case "plant_watered":
		effect.entity.Growth = plantGrowth{WateredTick: event.Tick, WateredSequence: event.Sequence}
	}
	b.Entities[effect.entityID] = effect.entity
	if placesItem && b.ContentVersion >= itemWitnessContentVersion {
		b.recordWitnesses(*event)
	}
	return nil
}

func (b *Bakery) recordOpenCheckIn(event Event) {
	entry := NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence, Action: ActionSpeak, Open: true, Subject: event.Payload.Target, Asked: true}
	b.appendSighting(event.ActorID, entry)
	entry.Subject, entry.Asked = event.ActorID, false
	b.appendSighting(event.Payload.Target, entry)
	heard := entry
	heard.Overheard, heard.Asked = true, false
	heard.Subject, heard.Target = event.ActorID, event.Payload.Target
	for id, e := range b.Entities {
		if e.Kind != Actor || id == event.ActorID || id == event.Payload.Target {
			continue
		}
		view, err := b.Observe(event.Scope, id)
		if err != nil || !view.hasTarget(event.ActorID) || !view.hasTarget(event.Payload.Target) {
			continue
		}
		b.appendSighting(id, heard)
	}
}

func (b *Bakery) recordAnswerFirst(event Event) {
	about := event.Payload.Dialogue.Statement.Topic
	b.appendSighting(event.ActorID, NotebookEntry{
		Tick: event.Tick, SourceSequence: event.Sequence, Subject: about, Action: ActionSpeak, Condition: resting, Answered: true,
	})
	b.appendSighting(event.Payload.Target, NotebookEntry{
		Tick: event.Tick, SourceSequence: event.Sequence, Subject: event.ActorID, Action: ActionSpeak, Reported: event.Payload.Dialogue.Statement.Source,
	})
}

func (b *Bakery) recordPickedUp(event Event) {
	entry := NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence, Action: ActionSpeak, Condition: resting, Subject: event.ActorID, Asked: true}
	b.appendSighting(event.Payload.Target, entry)
	entry.Subject, entry.Asked = event.Payload.Target, false
	b.appendSighting(event.ActorID, entry)
	heard := entry
	heard.Overheard, heard.Asked = true, false
	heard.Subject, heard.Target = event.Payload.Target, event.ActorID
	for id, e := range b.Entities {
		if e.Kind != Actor || id == event.ActorID || id == event.Payload.Target {
			continue
		}
		view, err := b.Observe(event.Scope, id)
		if err != nil || !view.hasTarget(event.ActorID) || !view.hasTarget(event.Payload.Target) {
			continue
		}
		b.appendSighting(id, heard)
	}
}

func (b *Bakery) openCheckInAbout(owner, about string, before uint64) (NotebookEntry, bool) {
	for i := len(b.Notebooks[owner]) - 1; i >= 0; i-- {
		e := b.Notebooks[owner][i]
		if e.SourceSequence >= before {
			continue
		}
		if e.Overheard && e.Open && e.Target == about {
			return e, true
		}
	}
	return NotebookEntry{}, false
}

func (b *Bakery) overhearCheckIn(event Event, heard NotebookEntry) {
	heard.Overheard = true
	heard.Asked = false
	heard.Target = event.Payload.Target
	heard.Subject = event.ActorID
	for id, e := range b.Entities {
		if e.Kind != Actor || id == event.ActorID || id == event.Payload.Target {
			continue
		}
		view, err := b.Observe(event.Scope, id)
		if err != nil || !view.hasTarget(event.ActorID) || !view.hasTarget(event.Payload.Target) {
			continue
		}
		b.appendSighting(id, heard)
	}
}

func (b *Bakery) recordWitnesses(event Event) {
	entry := NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence, Subject: event.ActorID, Target: event.Payload.Target}
	switch event.Kind {
	case "actors_conversed", "call_answered":
		x := event.Payload.Dialogue
		if x.Open {
			b.recordOpenCheckIn(event)
			return
		}
		if x.Statement.Intent == "answer_first" {
			b.recordAnswerFirst(event)
			return
		}
		if x.Statement.Intent == "pick_up" {
			b.recordPickedUp(event)
			return
		}
		if x.Condition != "" {
			entry.Subject, entry.Target, entry.Action = event.Payload.Target, "", ActionSpeak
			entry.Condition, entry.Asked = x.Condition, true
			b.appendSighting(event.ActorID, entry)
			entry.Subject, entry.Asked = event.ActorID, false
			b.appendSighting(event.Payload.Target, entry)
			if b.ContentVersion >= groupConversationVersion {
				b.overhearCheckIn(event, entry)
			}
			return
		}
		heard := func(owner, speaker string, source uint64) {
			if source != 0 {
				b.appendSighting(owner, NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence, Subject: speaker, Action: ActionSpeak, Reported: source})
			}
		}
		heard(entry.Target, entry.Subject, x.SpeakerSource)
		heard(entry.Subject, entry.Target, x.ReplySource)
		return
	case "actor_introduced":
		// Familiarity starts with the person addressed. Keep the source of
		// their first introduction even if the speaker repeats it later.
		if !b.recognizes(entry.Target, entry.Subject) {
			entry.Action = ActionIntroduce
			b.appendSighting(entry.Target, entry)
		}
		return
	case "actor_rested":
		if b.Entities[event.ActorID].Seated {
			return // Only sitting down is observable.
		}
		entry.Action = ActionRest
	case "actor_hid":
		entry.Action = ActionHide // The witness sees the actor duck before cover blocks the view.
	case "plant_harvested":
		entry.Action = ActionHarvest
	case "door_knocked":
		entry.Action = ActionKnock
	case "door_opened", "container_opened":
		entry.Action = ActionOpen
	case "door_closed", "container_closed":
		entry.Action = ActionClose
	case "item_taken":
		if b.ContentVersion < itemWitnessContentVersion {
			return
		}
		entry.Action = ActionTake
	case "item_dropped":
		entry.Action = ActionDrop
	case "item_put":
		entry.Action, entry.Container = ActionPut, event.Payload.HolderID
	default:
		return
	}
	for id, e := range b.Entities {
		if e.Kind != Actor || id == event.ActorID {
			continue
		}
		view, err := b.Observe(event.Scope, id)
		if err != nil || !view.hasTarget(entry.Subject) || (entry.Target != "" && !view.hasTarget(entry.Target)) ||
			(entry.Container != "" && !view.hasTarget(entry.Container)) {
			continue
		}
		b.appendSighting(id, entry)
	}
}

func (b *Bakery) appendSighting(owner string, entry NotebookEntry) {
	if b.Notebooks == nil {
		b.Notebooks = make(map[string][]NotebookEntry)
	}
	// A staged timeline shares its old pages with the committed one.
	b.Notebooks[owner] = append(slices.Clip(b.Notebooks[owner]), entry)
}

func (b *Bakery) entryAt(owner string, sequence uint64) (NotebookEntry, bool) {
	entries := b.Notebooks[owner]
	i, found := slices.BinarySearchFunc(entries, sequence, func(entry NotebookEntry, seq uint64) int {
		return cmp.Compare(entry.SourceSequence, seq)
	})
	if !found {
		return NotebookEntry{}, false
	}
	return entries[i], true
}

// firsthandAt looks up a participant or eyewitness memory by event sequence.
func (b *Bakery) firsthandAt(owner string, sequence uint64) (NotebookEntry, bool) {
	e, found := b.entryAt(owner, sequence)
	if e.Reported != 0 {
		return NotebookEntry{}, false
	}
	return e, found
}

// The listener's first introduction is the durable source of familiarity.
func (b *Bakery) recognizes(listener, speaker string) bool {
	return slices.ContainsFunc(b.Notebooks[listener], func(e NotebookEntry) bool {
		return e.Action == ActionIntroduce && e.Subject == speaker
	})
}

// Notebook returns one actor's own observations for read-only inspection.
func (b *Bakery) Notebook(actor string) ([]NotebookEntry, error) {
	if b == nil || b.ContentVersion < notebookContentVersion {
		return nil, fmt.Errorf("this timeline has no notebooks")
	}
	if b.Entities[actor].Kind != Actor {
		return nil, fmt.Errorf("unknown notebook owner %q", actor)
	}
	return slices.Clone(b.Notebooks[actor]), nil
}

func (b *Bakery) validateNotebooks(tick int64) error {
	for owner, entries := range b.Notebooks {
		if b.ContentVersion < notebookContentVersion || b.Entities[owner].Kind != Actor || len(entries) == 0 {
			return fmt.Errorf("invalid saved notebook for %q", owner)
		}
		introduced := make(map[string]bool)
		var prev NotebookEntry
		for _, e := range entries {
			if e.Tick <= 0 || e.Tick < prev.Tick || e.Tick > tick || e.SourceSequence <= prev.SourceSequence ||
				(e.Action != ActionSpeak && (e.Reported != 0 || e.Condition != "")) || (e.Asked && e.Condition == "" && !e.Open) ||
				(e.Container != "" && e.Action != ActionPut && e.Action != SightHandMovement) || e.Subject == owner ||
				(e.Action != ActionListen && e.Sound != "") ||
				(e.Action != ActionListen && e.Action != SightHandMovement && b.Entities[e.Subject].Kind != Actor) {
				return fmt.Errorf("invalid notebook sighting for %q at sequence %d", owner, e.SourceSequence)
			}
			switch e.Action {
			case ActionListen:
				if b.ContentVersion < soundAccountContentVersion || e.Subject != "" || e.Target != "" || e.Sound.SoundPhrase() == "" ||
					(e.Sound == Clatter && b.ContentVersion < dropSoundContentVersion) ||
					(e.Sound == Knock && b.ContentVersion < knockContentVersion) {
					return fmt.Errorf("invalid sound memory for %q at sequence %d", owner, e.SourceSequence)
				}
			case SightHandMovement:
				if b.ContentVersion < discreetTakeContentVersion || e.Subject != "" || e.Target != "" ||
					(e.Container != "" && b.Entities[e.Container].Kind != Container) {
					return fmt.Errorf("invalid hand movement sighting for %q at sequence %d", owner, e.SourceSequence)
				}
			case ActionSpeak:
				if e.Open {
					if b.ContentVersion < answerFirstVersion || e.Condition != "" || e.Reported != 0 || e.Answered {
						return fmt.Errorf("invalid open check-in for %q at sequence %d", owner, e.SourceSequence)
					}
					if e.Overheard {
						asker, heardAsker := b.firsthandAt(e.Subject, e.SourceSequence)
						listener, heardListener := b.firsthandAt(e.Target, e.SourceSequence)
						if e.Asked || e.Target == "" || !heardAsker || !heardListener || !asker.Open || !asker.Asked || asker.Overheard || asker.Subject != e.Target ||
							!listener.Open || listener.Asked || listener.Overheard || listener.Subject != e.Subject {
							return fmt.Errorf("invalid open check-in for %q at sequence %d", owner, e.SourceSequence)
						}
						break
					}
					other, found := b.firsthandAt(e.Subject, e.SourceSequence)
					if e.Target != "" || !found || !other.Open || other.Overheard || other.Asked == e.Asked || other.Subject != owner {
						return fmt.Errorf("invalid open check-in for %q at sequence %d", owner, e.SourceSequence)
					}
					break
				}
				if e.Answered {
					_, valid := checkInAnswers[e.Condition]
					_, heard := b.openCheckInAbout(owner, e.Subject, e.SourceSequence)
					if b.ContentVersion < answerFirstVersion || !valid || e.Condition != resting || e.Asked || e.Overheard || e.Reported != 0 || e.Target != "" || !heard {
						return fmt.Errorf("invalid early answer for %q at sequence %d", owner, e.SourceSequence)
					}
					break
				}
				if e.Overheard {
					asker, heardAsker := b.firsthandAt(e.Subject, e.SourceSequence)
					listener, heardListener := b.firsthandAt(e.Target, e.SourceSequence)
					_, valid := checkInAnswers[e.Condition]
					if b.ContentVersion < groupConversationVersion || !valid || e.Asked || e.Reported != 0 || e.Target == "" ||
						!heardAsker || !heardListener || !asker.Asked || asker.Subject != e.Target || listener.Asked || listener.Subject != e.Subject ||
						asker.Condition != e.Condition || listener.Condition != e.Condition {
						return fmt.Errorf("invalid overheard check-in for %q at sequence %d", owner, e.SourceSequence)
					}
					break
				}
				if e.Condition != "" {
					other, found := b.firsthandAt(e.Subject, e.SourceSequence)
					_, valid := checkInAnswers[e.Condition]
					if b.ContentVersion < conversationContentVersion || !valid || e.Reported != 0 || e.Target != "" ||
						!found || other.Subject != owner || other.Tick != e.Tick || other.Condition != e.Condition || other.Asked == e.Asked {
						return fmt.Errorf("invalid check-in for %q at sequence %d", owner, e.SourceSequence)
					}
					break
				}
				fact, ok := b.entryAt(e.Subject, e.Reported)
				if ok && fact.Reported != 0 {
					_, ok = b.reportedObservation(fact)
				}
				if !ok || fact.Tick > e.Tick || e.Reported >= e.SourceSequence || e.Target != "" || b.ContentVersion < relationshipContentVersion {
					return fmt.Errorf("invalid reported sighting for %q at sequence %d", owner, e.SourceSequence)
				}
			case ActionIntroduce:
				if b.ContentVersion < relationshipContentVersion || e.Target != owner || introduced[e.Subject] {
					return fmt.Errorf("invalid introduction for %q at sequence %d", owner, e.SourceSequence)
				}
				introduced[e.Subject] = true
			case ActionRest:
				if e.Target != "" {
					return fmt.Errorf("unexpected target in a rest sighting")
				}
			case ActionHide:
				if b.ContentVersion < coverContentVersion || !b.Entities[e.Target].ProvidesCover {
					return fmt.Errorf("invalid hiding place in a notebook sighting")
				}
			case ActionHarvest:
				if b.ContentVersion < harvestContentVersion || !b.Entities[e.Target].Growth.Harvested || b.Entities[e.Target].Kind != Planter {
					return fmt.Errorf("invalid harvested planter in a notebook sighting")
				}
			case ActionKnock:
				if b.ContentVersion < knockContentVersion || b.Entities[e.Target].Kind != Door {
					return fmt.Errorf("invalid door in a notebook sighting")
				}
			case ActionOpen, ActionClose:
				if kind := b.Entities[e.Target].Kind; kind != Door && kind != Container {
					return fmt.Errorf("invalid target in a notebook sighting")
				}
			case ActionTake, ActionDrop, ActionPut:
				item := b.Entities[e.Target]
				if b.ContentVersion < itemWitnessContentVersion || item.Kind != Item || !item.Portable ||
					(e.Action == ActionPut && b.Entities[e.Container].Kind != Container) {
					return fmt.Errorf("invalid item transfer sighting for %q at sequence %d", owner, e.SourceSequence)
				}
			default:
				return fmt.Errorf("unknown notebook action %q", e.Action)
			}
			prev = e
		}
	}
	return nil
}
