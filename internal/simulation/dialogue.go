// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

// SpeechLine is the stable identity behind a displayed utterance. A memory
// source identifies one event, even if two events concern the same person.
type SpeechLine struct {
	Topic  string `json:"topic"`
	Intent string `json:"intent"`
	Tone   string `json:"tone"`
	Source uint64 `json:"source,omitempty"`
}

type DialogueChoice struct {
	Line  SpeechLine
	Words string
}

// Exchange records both audible lines. Sources refer to a firsthand sighting
// in the notebook of the participant who actually voiced that report.
type Exchange struct {
	Words         string
	Reply         string
	SpeakerSource uint64 `json:",omitempty"`
	ReplySource   uint64 `json:",omitempty"`
}

func (c *Coordinator) DialogueChoices(actor, listener string) ([]DialogueChoice, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bakery.DialogueChoices(c.scope, actor, listener)
}

func (b *Bakery) DialogueChoices(scope Scope, actor, listener string) ([]DialogueChoice, error) {
	view, err := b.Observe(scope, actor)
	if err != nil {
		return nil, err
	}
	return b.dialogueChoices(view, listener), nil
}

func (b *Bakery) canTalk(view Observation, listener string) bool {
	return b.ContentVersion >= relationshipContentVersion && view.hasTarget(listener) &&
		b.Entities[listener].Kind == Actor && inReach(b.Entities[view.ObserverID].Placement.Cell, b.Entities[listener].Placement.Cell)
}

func (b *Bakery) dialogueChoices(view Observation, listener string) []DialogueChoice {
	if !b.canTalk(view, listener) {
		return nil
	}
	const limit = 11
	var choices []DialogueChoice
	seen := make(map[string]bool)
	add := func(topic, intent, tone string, source uint64, words string) {
		if len(choices) < limit && !seen[words] {
			choices = append(choices, DialogueChoice{SpeechLine{topic, intent, tone, source}, words})
			seen[words] = true
		}
	}
	add("greeting", "greet", "warm", 0, "Hello.")
	add("condition", "ask", "gentle", 0, "How are you holding up?")
	add("news", "ask", "curious", 0, "What have you seen around here?")

	// Most recent memories first. Only a report addressed to this speaker
	// can unlock a question about what the listener told them.
	entries := b.Notebooks[view.ObserverID]
	for i := len(entries) - 1; i >= 0 && len(choices) < limit; i-- {
		e := entries[i]
		if e.Action == ActionSpeak {
			if e.Subject == listener {
				if fact, ok := b.reportedSighting(listener, e.Reported); ok {
					add(fact.Subject, "confirm", "careful", fact.SourceSequence, b.sightingWords(fact, view.ObserverID, true))
				}
			}
		} else {
			add(e.Subject, "share", "plain", e.SourceSequence, b.sightingWords(e, listener, false))
		}
	}
	for _, obj := range view.Targets {
		if obj.Ref == listener {
			continue
		}
		if b.Entities[obj.Ref].Kind == Actor {
			add(obj.Ref, "ask", "curious", 0, "Have you met the "+obj.Name+"?")
		} else {
			add(obj.Ref, "notice", "curious", 0, "Can you see the "+obj.Name+" from there?")
		}
	}
	return choices
}

// A confirmation gives the same event back to its source in the hearer's voice.
func (b *Bakery) sightingWords(fact NotebookEntry, hearer string, confirm bool) string {
	name := b.Entities[fact.Subject].Name
	if fact.Action == ActionIntroduce {
		if fact.Subject == hearer {
			if confirm {
				return "So I introduced myself to you?"
			}
			return "You introduced yourself to me."
		}
		if confirm {
			return "So the " + name + " introduced themselves to you?"
		}
		return "The " + name + " introduced themselves to me."
	}
	verb := "sit down"
	if fact.Action != ActionRest {
		verb = string(fact.Action) + " the " + b.Entities[fact.Target].Name
	}
	who := "the " + name
	if fact.Subject == hearer {
		if confirm {
			who = "me"
		} else {
			who = "you"
		}
	}
	if confirm {
		return "You saw " + who + " " + verb + "?"
	}
	return "I saw " + who + " " + verb + "."
}

func (b *Bakery) speak(view Observation, listener string, line SpeechLine) (spatialEffect, *actionFailure) {
	choices := b.dialogueChoices(view, listener)
	if len(choices) == 0 {
		return spatialEffect{}, targetUnavailable()
	}
	for _, choice := range choices {
		if choice.Line != line {
			continue
		}
		listening, err := b.Observe(view.Scope, listener)
		if err != nil {
			return spatialEffect{}, targetUnavailable()
		}
		exchange := b.answer(view.ObserverID, listening, choice)
		return spatialEffect{kind: "actors_conversed", payload: EventPayload{Target: listener, Dialogue: exchange},
			message:  "You say, “" + exchange.Words + "”\nThe " + b.Entities[listener].Name + " replies, “" + exchange.Reply + "”",
			entityID: view.ObserverID, entity: b.Entities[view.ObserverID]}, nil
	}
	return spatialEffect{}, &actionFailure{"line_unavailable", "That line is no longer available."}
}

func (b *Bakery) answer(speaker string, view Observation, choice DialogueChoice) Exchange {
	listener := view.ObserverID
	line := choice.Line
	x := Exchange{Words: choice.Words}
	switch line.Intent {
	case "share":
		x.SpeakerSource, x.Reply = line.Source, "Thanks for telling me."
	case "confirm":
		x.Reply = "Yes. That's what I said."
	case "greet":
		x.Reply = "Hello."
		if b.recognizes(listener, speaker) {
			x.Reply = "Good to see you again."
		}
	case "notice":
		x.Reply = "I can't see it from here."
		if view.hasTarget(line.Topic) {
			x.Reply = "Yes, I can see it from here."
		}
	case "ask":
		switch line.Topic {
		case "condition":
			actor := b.Entities[listener]
			switch {
			case actor.ReliefTurns > 0:
				x.Reply = "Better. I needed that rest."
			case actor.SeekingRestSpot || actor.Fatigue == fatigueLimit:
				x.Reply = "I could use a place to sit."
			case actor.Seated:
				x.Reply = "I'm taking a breather."
			default:
				x.Reply = "Doing all right, thank you."
			}
		default:
			x.Reply = "I don't remember an introduction."
			if line.Topic == "news" {
				x.Reply = "Nothing comes to mind."
			}
			entries := b.Notebooks[listener]
			for i := len(entries) - 1; i >= 0; i-- {
				fact := entries[i]
				if (line.Topic == "news" && fact.Action != ActionSpeak) || (fact.Action == ActionIntroduce && fact.Subject == line.Topic) {
					x.Reply = b.sightingWords(fact, speaker, false)
					x.ReplySource = fact.SourceSequence
					break
				}
			}
		}
	}
	return x
}
