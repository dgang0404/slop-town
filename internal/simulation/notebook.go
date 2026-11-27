// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"cmp"
	"fmt"
	"slices"
)

// NotebookEntry records an act the owner witnessed or a report they heard.
// SourceSequence identifies that sighting or exchange on the owner's timeline.
type NotebookEntry struct {
	Tick           int64         `json:"tick"`
	SourceSequence uint64        `json:"source_sequence"`
	Subject        string        `json:"subject"`
	Action         ActionKind    `json:"action"`
	Target         string        `json:"target,omitempty"`
	Reported       uint64        `json:"reported,omitempty"`  // Entry in the immediate teller's notebook.
	Condition      workCondition `json:"condition,omitempty"` // Listener's answer during this owner's own exchange.
	Asked          bool          `json:"asked,omitempty"`
	Container      string        `json:"container,omitempty"` // Visible destination of a put or landmark for a hand movement.
}

// Observe a take before the item enters a private inventory. A drop or put
// becomes visible on placement, before anyone else can act on the same tick.
func (b *Bakery) applySpatial(event Event, effect spatialEffect, clock *Timeline) error {
	placesItem := event.Kind == "item_put" || event.Kind == "item_dropped"
	if event.Kind == "item_taken_discreetly" {
		if err := b.observeDiscreetTake(event, clock); err != nil {
			return err
		}
	} else if b.ContentVersion >= notebookContentVersion && !placesItem {
		b.recordWitnesses(event)
	}
	b.Entities[effect.entityID] = effect.entity
	if placesItem && b.ContentVersion >= itemWitnessContentVersion {
		b.recordWitnesses(event)
	}
	return nil
}

func (b *Bakery) recordWitnesses(event Event) {
	entry := NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence, Subject: event.ActorID, Target: event.Payload.Target}
	switch event.Kind {
	case "actors_conversed":
		x := event.Payload.Dialogue
		if x.Condition != "" {
			entry.Subject, entry.Target, entry.Action = event.Payload.Target, "", ActionSpeak
			entry.Condition, entry.Asked = x.Condition, true
			b.appendSighting(event.ActorID, entry)
			entry.Subject, entry.Asked = event.ActorID, false
			b.appendSighting(event.Payload.Target, entry)
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
				(e.Action != ActionSpeak && (e.Reported != 0 || e.Condition != "")) || (e.Asked && e.Condition == "") ||
				(e.Container != "" && e.Action != ActionPut && e.Action != SightHandMovement) || e.Subject == owner ||
				(e.Action != SightHandMovement && b.Entities[e.Subject].Kind != Actor) {
				return fmt.Errorf("invalid notebook sighting for %q at sequence %d", owner, e.SourceSequence)
			}
			switch e.Action {
			case SightHandMovement:
				if b.ContentVersion < discreetTakeContentVersion || e.Subject != "" || e.Target != "" ||
					(e.Container != "" && b.Entities[e.Container].Kind != Container) {
					return fmt.Errorf("invalid hand movement sighting for %q at sequence %d", owner, e.SourceSequence)
				}
			case ActionSpeak:
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
