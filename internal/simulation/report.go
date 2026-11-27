// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "fmt"

const reportThanks = "Thanks for telling me."

func (e NotebookEntry) isObservation() bool {
	switch e.Action {
	case ActionRest, ActionOpen, ActionClose, ActionKnock, ActionTake, ActionPut, ActionDrop, ActionHide, SightHandMovement, ActionListen:
		return true
	}
	return false
}

func (b *Bakery) observedAct(fact NotebookEntry, me, you string) string {
	if fact.Action == ActionListen {
		return fact.Sound.SoundPhrase()
	}
	sight := b.witnessedAction(fact)
	if fact.Action == SightHandMovement {
		return sight.HandMovementPhrase()
	}
	who := "the " + sight.Actor.Name
	switch fact.Subject {
	case me:
		who = "me"
	case you:
		who = "you"
	}
	if fact.Action == ActionRest {
		return who + " sit down"
	}
	return who + " " + string(fact.Action) + " " + sight.ObjectPhrase()
}

func (b *Bakery) perceivedAct(fact NotebookEntry, me, you string) string {
	if fact.Action == ActionListen {
		return "heard " + fact.Sound.SoundPhrase()
	}
	return "saw " + b.observedAct(fact, me, you)
}

func (e NotebookEntry) speechLine(intent, tone string) SpeechLine {
	topic := e.Subject
	if e.Action == SightHandMovement {
		topic = string(SightHandMovement)
	}
	if e.Action == ActionListen {
		topic = string(e.Sound)
	}
	return SpeechLine{Topic: topic, Intent: intent, Tone: tone, Source: e.SourceSequence}
}

// A memory offers either the owner's account or a follow-up to the person
// who told it. A relay keeps its original observer even when the teller changes.
func (b *Bakery) memoryLine(e NotebookEntry, speaker, listener string) (SpeechLine, string) {
	if e.Reported == 0 {
		return e.speechLine("share", "plain"), b.sightingWords(e, listener, false)
	}
	if e.Subject == listener {
		source, ok := b.entryAt(listener, e.Reported)
		if ok && source.Reported == 0 {
			return source.speechLine("confirm", "careful"), b.sightingWords(source, speaker, true)
		}
		if sight, ok := b.reportedObservation(source); ok {
			return source.speechLine("confirm_source", "careful"), b.relayWords(source, sight, listener, speaker, true)
		}
	} else if sight, ok := b.reportedObservation(e); ok {
		return e.speechLine("relay", "plain"), b.relayWords(e, sight, speaker, listener, false)
	}
	return SpeechLine{}, ""
}

// Only a report heard directly from the person who perceived it can be passed
// on. Other people's memories of the same incident have their own sources.
func (b *Bakery) reportedObservation(report NotebookEntry) (NotebookEntry, bool) {
	if b.ContentVersion < reportContentVersion || report.Action != ActionSpeak || report.Reported == 0 {
		return NotebookEntry{}, false
	}
	fact, found := b.firsthandAt(report.Subject, report.Reported)
	return fact, found && fact.isObservation()
}

func (b *Bakery) relayWords(report, fact NotebookEntry, teller, hearer string, confirm bool) string {
	witness := b.Entities[report.Subject].Name
	if confirm {
		act := b.perceivedAct(fact, hearer, teller)
		if report.Subject == hearer {
			return "Did I tell you I " + act + "?"
		}
		return "Did the " + witness + " tell you they " + act + "?"
	}
	act := b.perceivedAct(fact, teller, hearer)
	if report.Subject == hearer {
		return "You told me you " + act + "."
	}
	return "The " + witness + " told me they " + act + "."
}

// Both a chosen line and an autonomous exchange must cite a memory belonging
// to the speaker. Background events retain the reference without a transcript.
func (b *Bakery) transferReport(view Observation, listener string, line SpeechLine) (spatialEffect, *actionFailure) {
	if !b.canTalk(view, listener) {
		return spatialEffect{}, targetUnavailable()
	}
	e, found := b.entryAt(view.ObserverID, line.Source)
	intent := "share"
	if e.Reported != 0 {
		intent = "relay"
		_, found = b.reportedObservation(e)
	}
	if !found || line != e.speechLine(intent, "plain") {
		return spatialEffect{}, &actionFailure{"line_unavailable", "That line is no longer available."}
	}
	x := Exchange{SpeakerSource: line.Source}
	if b.ContentVersion >= reportContentVersion {
		x.Statement = line
	}
	return spatialEffect{kind: "actors_conversed", payload: EventPayload{Target: listener, Dialogue: x},
		entityID: view.ObserverID, entity: b.Entities[view.ObserverID]}, nil
}

func (b *Bakery) exchangeLines(e Event) (string, string, error) {
	x := e.Payload.Dialogue
	if x.Statement.Intent == "share" && x.Words == "" {
		fact, ok := b.firsthandAt(e.ActorID, x.SpeakerSource)
		if !ok || !fact.isObservation() || x.Statement != fact.speechLine("share", "plain") {
			return "", "", fmt.Errorf("missing firsthand account at sequence %d", e.Sequence)
		}
		return b.sightingWords(fact, e.Payload.Target, false), reportThanks, nil
	}
	words, reply := x.lines()
	return words, reply, nil
}

// The worker offers their latest firsthand sight or sound they have not
// already described to this coworker.
func (b *Bakery) residentReport(scope Scope, speaker, listener string) spatialEffect {
	told := make(map[uint64]bool)
	for _, e := range b.Notebooks[listener] {
		if e.Subject == speaker && e.Reported != 0 {
			told[e.Reported] = true
		}
	}
	entries := b.Notebooks[speaker]
	for i := len(entries) - 1; i >= 0; i-- {
		fact := entries[i]
		if !fact.isObservation() || fact.Subject == listener || told[fact.SourceSequence] {
			continue
		}
		view, err := b.Observe(scope, speaker)
		if err != nil {
			return spatialEffect{}
		}
		effect, _ := b.transferReport(view, listener, fact.speechLine("share", "plain"))
		return effect
	}
	return spatialEffect{}
}

// Follow the particular account the asker heard through its teller to the
// named witness. Another observer of the same event cannot answer for them.
func (b *Bakery) originalWitness(account NotebookEntry, witness string) (NotebookEntry, bool) {
	if account.Reported == 0 || account.Subject == witness {
		return NotebookEntry{}, false
	}
	telling, ok := b.entryAt(account.Subject, account.Reported)
	if !ok || telling.Subject != witness {
		return NotebookEntry{}, false
	}
	return b.reportedObservation(telling)
}

func (b *Bakery) witnessQuestions(speaker, listener string, limit int) []DialogueChoice {
	var choices []DialogueChoice
	seen := make(map[string]bool)
	entries := b.Notebooks[speaker]
	lastDirect := make(map[uint64]uint64)
	for _, e := range entries {
		if e.Subject == listener && e.Reported != 0 {
			lastDirect[e.Reported] = e.SourceSequence
		}
	}
	for i := len(entries) - 1; i >= 0 && len(choices) < limit; i-- {
		e := entries[i]
		fact, ok := b.originalWitness(e, listener)
		if !ok {
			continue
		}
		// A later retelling cannot reopen a sighting we heard from its witness.
		// Version 10 retired only retellings that preceded the direct account.
		if direct := lastDirect[fact.SourceSequence]; direct != 0 && (b.ContentVersion >= itemWitnessContentVersion || direct > e.SourceSequence) {
			continue
		}
		words := "The " + b.Entities[e.Subject].Name + " said you " + b.perceivedAct(fact, speaker, listener) + ". Is that right?"
		if !seen[words] {
			choices = append(choices, DialogueChoice{Line: e.speechLine("ask_witness", "careful"), Words: words})
			seen[words] = true
		}
	}
	return choices
}
