// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"slices"
)

// NotebookEntry is a firsthand sighting of another actor. SourceSequence
// identifies the causing event within the notebook owner's timeline.
type NotebookEntry struct {
	Tick           int64      `json:"tick"`
	SourceSequence uint64     `json:"source_sequence"`
	Subject        string     `json:"subject"`
	Action         ActionKind `json:"action"`
	Target         string     `json:"target,omitempty"`
}

// applySpatial records witnesses before applying this individual effect. A
// later door opening or resident move on the same tick cannot add a witness.
func (b *Bakery) applySpatial(event Event, effect spatialEffect) {
	if b.ContentVersion >= notebookContentVersion {
		b.recordWitnesses(event)
	}
	b.Entities[effect.entityID] = effect.entity
}

func (b *Bakery) recordWitnesses(event Event) {
	entry := NotebookEntry{Tick: event.Tick, SourceSequence: event.Sequence, Subject: event.ActorID, Target: event.Payload.Target}
	switch event.Kind {
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
	default:
		return
	}
	for id, e := range b.Entities {
		if e.Kind != Actor || id == event.ActorID {
			continue
		}
		view, err := b.Observe(event.Scope, id)
		if err != nil || !view.hasTarget(entry.Subject) || (entry.Target != "" && !view.hasTarget(entry.Target)) {
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
				e.Subject == owner || b.Entities[e.Subject].Kind != Actor {
				return fmt.Errorf("invalid notebook sighting for %q at sequence %d", owner, e.SourceSequence)
			}
			switch e.Action {
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
			default:
				return fmt.Errorf("unknown notebook action %q", e.Action)
			}
			prev = e
		}
	}
	return nil
}
