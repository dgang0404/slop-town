// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "fmt"

const reportThanks = "Thanks for telling me."

func (e NotebookEntry) isObservation() bool {
	return e.Action == ActionRest || e.Action == ActionOpen || e.Action == ActionClose
}

func (b *Bakery) observedAct(fact NotebookEntry, me, you string) string {
	who := "the " + b.Entities[fact.Subject].Name
	switch fact.Subject {
	case me:
		who = "me"
	case you:
		who = "you"
	}
	if fact.Action == ActionRest {
		return who + " sit down"
	}
	return who + " " + string(fact.Action) + " the " + b.Entities[fact.Target].Name
}

func (e NotebookEntry) speechLine(intent, tone string) SpeechLine {
	return SpeechLine{Topic: e.Subject, Intent: intent, Tone: tone, Source: e.SourceSequence}
}

// A memory offers either the owner's account or a follow-up to the person
// who told it. A relay keeps its eyewitness even when the teller changes.
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

// Only a report heard directly from an eyewitness can be passed on. The
// reference goes through that specific witness even when others saw the act.
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
		act := b.observedAct(fact, hearer, teller)
		if report.Subject == hearer {
			return "Did I tell you I saw " + act + "?"
		}
		return "Did the " + witness + " tell you they saw " + act + "?"
	}
	act := b.observedAct(fact, teller, hearer)
	if report.Subject == hearer {
		return "You told me you saw " + act + "."
	}
	return "The " + witness + " told me they saw " + act + "."
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
			return "", "", fmt.Errorf("missing eyewitness account at sequence %d", e.Sequence)
		}
		return b.sightingWords(fact, e.Payload.Target, false), reportThanks, nil
	}
	words, reply := x.lines()
	return words, reply, nil
}

// The worker offers the latest physical act they saw and have not already
// described to this coworker. Only earlier exchanges between this pair count.
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
