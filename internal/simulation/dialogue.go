// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "net/url"

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
	ContactGiver  string        `json:"AddressGiver,omitempty"`
	Number        string        `json:",omitempty"`
	Address       string        `json:",omitempty"`
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
	return b.dialogueChoices(view, listener, true), nil
}

func (b *Bakery) canTalk(view Observation, listener string) bool {
	return b.ContentVersion >= relationshipContentVersion && view.hasTarget(listener) &&
		b.Entities[listener].Kind == Actor && inReach(b.Entities[view.ObserverID].Placement.Cell, b.Entities[listener].Placement.Cell)
}

func (b *Bakery) dialogueChoices(view Observation, listener string, includeWitnessQuestions bool) []DialogueChoice {
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

	if b.GeneratorVersion >= 4 && b.ContentVersion >= supplyQuestionVersion && listener == url.PathEscape(b.WorldID)+"/baker" {
		add(SpeechLine{Topic: "workroom", Intent: "ask_supplies", Tone: "curious"}, "Do you need anything for the workroom?")
	}

	// Most recent memories first. Only a report addressed to this speaker
	// can unlock a question about what the listener told them.
	entries := b.Notebooks[view.ObserverID]
	contactLines := b.contactChoices(view.ObserverID, listener)
	memoryLimit := limit - min(2, len(contactLines)) // Preserve room for recent testimony as more contact methods become available.
	if b.ContentVersion >= reportContentVersion {
		memoryLimit -= 2 // Keep the questions about conversations and secondhand news available.
		if b.ContentVersion >= soundAccountContentVersion {
			memoryLimit-- // Leave room to ask about firsthand sounds.
		}
		if includeWitnessQuestions {
			for _, choice := range b.witnessQuestions(view.ObserverID, listener, memoryLimit-len(choices)) {
				add(choice.Line, choice.Words)
			}
		}
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
	if b.ContentVersion >= soundAccountContentVersion {
		add(SpeechLine{Topic: "sounds", Intent: "ask", Tone: "curious"}, "What sounds have you heard lately?")
	}
	if len(contactLines) > limit-len(choices) {
		// A crowded exchange still offers one line for each contact method.
		for _, topic := range []string{"mail", "phone"} {
			for _, choice := range contactLines {
				if choice.Line.Topic == topic {
					add(choice.Line, choice.Words)
					break
				}
			}
		}
	}
	for _, choice := range contactLines {
		add(choice.Line, choice.Words)
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
	if fact.Overheard {
		return fact.overheardAccount(name, b.Entities[fact.Target].Name, fact.Subject == hearer, fact.Target == hearer, confirm)
	}
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
		return "You " + b.perceivedAct(fact, hearer, "") + "?"
	}
	return "I " + b.perceivedAct(fact, "", hearer) + "."
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
	// Check older proposals against the choices available before witness
	// questions existed. A new menu slot must not invalidate a saved turn.
	var choices []DialogueChoice
	if line == AskPhoneNumber || line == OfferPhoneNumber {
		// An explicit contact request can select a line omitted from a crowded general menu.
		if !b.canTalk(view, listener) {
			return spatialEffect{}, targetUnavailable()
		}
		choices = b.contactChoices(view.ObserverID, listener)
	} else {
		choices = b.dialogueChoices(view, listener, line.Intent == "ask_witness")
	}
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
	case "ask_supplies":
		prefix := url.PathEscape(b.WorldID) + "/"
		toolID := b.Entities[prefix+"order"].RequestedItem
		tool := b.Entities[toolID]
		x.Reply = "I could use the " + tool.Name + ". Please leave it in the workbench drawer."
		if tool.Placement.HolderID == listener {
			x.Reply = "I have the " + tool.Name + " I need for the workroom."
		}
		if b.sawSupplyDelivery(listener, speaker, toolID, prefix+"workbench") {
			if tool.Placement.HolderID != listener {
				x.Reply = "I asked for the " + tool.Name + "."
			}
			x.Reply += " I saw you put it in the drawer. Thank you."
		}
	case "ask_number":
		x.ContactGiver = listener
		x.Number, _ = b.phoneLine(listener)
		x.Reply = b.shareNumberWords(listener)
	case "offer_number":
		x.ContactGiver = speaker
		x.Number, _ = b.phoneLine(speaker)
		x.Reply = "Thank you. I’ll keep your number."
	case "ask_address":
		x.ContactGiver, x.Address = listener, b.addressOf(listener)
		x.Reply = "You can write to me at " + x.Address + "."
	case "offer_address":
		x.ContactGiver, x.Address = speaker, b.addressOf(speaker)
		x.Reply = "Thank you. I’ll keep your address."
	case "ask_witness":
		account, _ := b.entryAt(speaker, line.Source)
		fact, ok := b.originalWitness(account, listener)
		if !ok {
			x.Reply = "I don't remember seeing that."
			break
		}
		x.Reply = b.sightingWords(fact, speaker, false)
		x.ReplySource = fact.SourceSequence
	case "confirm_source":
		x.Reply = "Yes. That's who told me."
		if source, ok := b.entryAt(listener, line.Source); ok {
			if fact, ok := b.overheardReport(source); ok {
				x.Reply = "Yes. " + b.overheardRelay(source, fact, listener, speaker, false)
			}
		}
	case "confirm":
		x.Reply = "Yes. That's what I said."
		if fact, ok := b.entryAt(listener, line.Source); ok && fact.Overheard {
			x.Reply = "Yes. " + fact.overheardAccount(b.Entities[fact.Subject].Name, b.Entities[fact.Target].Name, fact.Subject == speaker, fact.Target == speaker, false)
		}
	case "greet":
		x.Reply = "Hello."
		if b.recognizes(listener, speaker) {
			x.Reply = "Good to see you again."
			if b.GeneratorVersion >= 4 && b.ContentVersion >= helperGreetingVersion {
				b.offerHelperNumber(&x, listener, speaker)
			}
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
			if b.ContentVersion >= soundAccountContentVersion {
				x.Reply = "I haven't heard a firsthand account I can pass on."
			}
		case "news", "sounds":
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
				if overheard, ok := b.overheardReport(fact); ok {
					x.Reply = b.overheardRelay(fact, overheard, listener, speaker, false)
					x.ReplySource = fact.SourceSequence
					break
				}
				if source, overheard, ok := b.passedOverheard(fact); ok {
					x.Reply = b.passedOverheardWords(fact, source, overheard, listener, speaker)
					x.ReplySource = fact.SourceSequence
					break
				}
				continue
			}
			if (line.Topic == "sounds" && fact.Action == ActionListen) ||
				(line.Topic == "news" && fact.Reported == 0 && fact.Action != ActionListen) || (line.Topic == "conversation" && fact.Condition != "") ||
				(fact.Action == ActionIntroduce && fact.Subject == line.Topic) {
				x.Reply = b.sightingWords(fact, speaker, false)
				x.ReplySource = fact.SourceSequence
				break
			}
		}
	}
	return x
}

func (b *Bakery) sawSupplyDelivery(witness, helper, tool, drawer string) bool {
	for _, note := range b.Notebooks[witness] {
		if note.Action == ActionPut && note.Subject == helper && note.Target == tool && note.Container == drawer {
			return true
		}
	}
	return false
}

// The baker can follow up on a favor they personally witnessed. This passes
// one person's number to one listener through the ordinary disclosure event.
func (b *Bakery) offerHelperNumber(x *Exchange, baker, helper string) {
	prefix := url.PathEscape(b.WorldID) + "/"
	if baker != prefix+"baker" || b.contact(helper, baker, ActionCall).Number != "" {
		return
	}
	tool := b.Entities[prefix+"order"].RequestedItem
	if !b.sawSupplyDelivery(baker, helper, tool, prefix+"workbench") {
		return
	}
	if b.Entities[tool].Placement.HolderID != baker && b.Entities[baker].WorkroomToolUseTick == 0 {
		return
	}
	x.ContactGiver = baker
	x.Number, _ = b.phoneLine(baker)
	x.Reply += " Thanks for bringing the " + b.Entities[tool].Name + ". " + b.shareNumberWords(baker)
}
