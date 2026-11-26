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

// Exchange keeps player-facing lines or the intent of a background conversation.
// Each report source points into the notebook of the person who spoke.
type Exchange struct {
	Words         string        `json:",omitempty"`
	Reply         string        `json:",omitempty"`
	SpeakerSource uint64        `json:",omitempty"`
	ReplySource   uint64        `json:",omitempty"`
	Question      SpeechLine    `json:",omitzero"`
	Statement     SpeechLine    `json:",omitzero"`
	Condition     workCondition `json:",omitempty"`
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
	add := func(line SpeechLine, words string) {
		if words != "" && len(choices) < limit && !seen[words] {
			choices = append(choices, DialogueChoice{line, words})
			seen[words] = true
		}
	}
	add(SpeechLine{Topic: "greeting", Intent: "greet", Tone: "warm"}, "Hello.")
	add(checkInQuestion, checkInWords)
	add(SpeechLine{Topic: "news", Intent: "ask", Tone: "curious"}, "What have you seen around here?")

	// Most recent memories first. Only a report addressed to this speaker
	// can unlock a question about what the listener told them.
	entries := b.Notebooks[view.ObserverID]
	memoryLimit := limit
	if b.ContentVersion >= reportContentVersion {
		memoryLimit -= 2 // Keep the questions about conversations and secondhand news available.
	}
	for i := len(entries) - 1; i >= 0 && len(choices) < memoryLimit; i-- {
		add(b.memoryLine(entries[i], view.ObserverID, listener))
	}
	if b.ContentVersion >= conversationContentVersion {
		add(SpeechLine{Topic: "conversation", Intent: "ask", Tone: "curious"}, "Have you had a chance to talk with anyone lately?")
	}
	if b.ContentVersion >= reportContentVersion {
		add(SpeechLine{Topic: "hearsay", Intent: "ask", Tone: "curious"}, "What have you heard from other people?")
	}
	for _, obj := range view.Targets {
		if obj.Ref == listener {
			continue
		}
		if b.Entities[obj.Ref].Kind == Actor {
			add(SpeechLine{Topic: obj.Ref, Intent: "ask", Tone: "curious"}, "Have you met the "+obj.Name+"?")
		} else {
			add(SpeechLine{Topic: obj.Ref, Intent: "notice", Tone: "curious"}, "Can you see the "+obj.Name+" from there?")
		}
	}
	return choices
}

// A confirmation gives the same event back to its source in the hearer's voice.
func (b *Bakery) sightingWords(fact NotebookEntry, hearer string, confirm bool) string {
	name := b.Entities[fact.Subject].Name
	if fact.Condition != "" {
		return fact.checkInAccount(name, fact.Subject == hearer, confirm)
	}
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
	if confirm {
		return "You saw " + b.observedAct(fact, hearer, "") + "?"
	}
	return "I saw " + b.observedAct(fact, "", hearer) + "."
}

func (b *Bakery) speak(view Observation, listener string, line SpeechLine) (spatialEffect, *actionFailure) {
	if line == checkInQuestion {
		effect, failure := b.checkIn(view, listener)
		if failure == nil {
			x := &effect.payload.Dialogue
			x.Words, x.Reply = x.lines()
			effect.message = x.message(b.Entities[listener].Name)
		}
		return effect, failure
	}
	choices := b.dialogueChoices(view, listener)
	if len(choices) == 0 {
		return spatialEffect{}, targetUnavailable()
	}
	for _, choice := range choices {
		if choice.Line != line {
			continue
		}
		if line.Intent == "share" || line.Intent == "relay" {
			effect, failure := b.transferReport(view, listener, line)
			if failure == nil {
				x := &effect.payload.Dialogue
				x.Words, x.Reply = choice.Words, reportThanks
				effect.message = x.message(b.Entities[listener].Name)
			}
			return effect, failure
		}
		listening, err := b.Observe(view.Scope, listener)
		if err != nil {
			return spatialEffect{}, targetUnavailable()
		}
		exchange := b.answer(view.ObserverID, listening, choice)
		return spatialEffect{kind: "actors_conversed", payload: EventPayload{Target: listener, Dialogue: exchange},
			message:  exchange.message(b.Entities[listener].Name),
			entityID: view.ObserverID, entity: b.Entities[view.ObserverID]}, nil
	}
	return spatialEffect{}, &actionFailure{"line_unavailable", "That line is no longer available."}
}

func (b *Bakery) answer(speaker string, view Observation, choice DialogueChoice) Exchange {
	listener := view.ObserverID
	line := choice.Line
	x := Exchange{Words: choice.Words}
	switch line.Intent {
	case "confirm_source":
		x.Reply = "Yes. That's who told me."
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
		case "hearsay":
			x.Reply = "No one's told me what they saw."
		case "news":
			x.Reply = "Nothing comes to mind."
		case "conversation":
			x.Reply = "I don't remember anyone checking in lately."
		default:
			x.Reply = "I don't remember an introduction."
		}
		entries := b.Notebooks[listener]
		for i := len(entries) - 1; i >= 0; i-- {
			fact := entries[i]
			if line.Topic == "hearsay" {
				if sight, ok := b.reportedObservation(fact); ok {
					x.Reply = b.relayWords(fact, sight, listener, speaker, false)
					x.ReplySource = fact.SourceSequence
					break
				}
				continue
			}
			if (line.Topic == "news" && fact.Reported == 0) || (line.Topic == "conversation" && fact.Condition != "") ||
				(fact.Action == ActionIntroduce && fact.Subject == line.Topic) {
				x.Reply = b.sightingWords(fact, speaker, false)
				x.ReplySource = fact.SourceSequence
				break
			}
		}
	}
	return x
}
