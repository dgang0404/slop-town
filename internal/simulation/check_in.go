// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

const checkInWords = "How are you holding up?"

var checkInQuestion = SpeechLine{Topic: "condition", Intent: "ask", Tone: "gentle"}

type workCondition string

const (
	doingFine workCondition = "fine"
	needsSeat workCondition = "needs_rest"
	resting   workCondition = "resting"
	relieved  workCondition = "relieved"
)

var checkInAnswers = map[workCondition]struct{ reply, remembered string }{
	doingFine: {"Doing all right, thank you.", "felt fine"},
	needsSeat: {"I could use a place to sit.", "needed a seat"},
	resting:   {"I'm taking a breather.", "had stopped for a breather"},
	relieved:  {"Better. I needed that rest.", "felt better after a rest"},
}

func (actor SpatialEntity) workCondition() workCondition {
	switch {
	case actor.ReliefTurns > 0:
		return relieved
	case actor.SeekingRestSpot || actor.Fatigue == fatigueLimit:
		return needsSeat
	case actor.Seated:
		return resting
	default:
		return doingFine
	}
}

// The menu and the resident routine use the same reach, intent and reply rules.
// An unobserved meeting needs only its participants, question and answer.
func (b *Bakery) checkIn(view Observation, listener string) (spatialEffect, *actionFailure) {
	if !b.canTalk(view, listener) {
		return spatialEffect{}, targetUnavailable()
	}
	x := Exchange{Question: checkInQuestion, Condition: b.Entities[listener].workCondition()}
	if b.ContentVersion < conversationContentVersion {
		// Existing explicit check-ins predate structured conversation memories.
		words, reply := x.lines()
		x = Exchange{Words: words, Reply: reply}
	} else if b.holdsCheckIn(view, listener) {
		// Sitting is visible. Someone who can see both people may answer before the person asked.
		x.Condition, x.Open, x.Words = "", true, checkInWords
	}
	return spatialEffect{kind: "actors_conversed", payload: EventPayload{Target: listener, Dialogue: x},
		entityID: view.ObserverID, entity: b.Entities[view.ObserverID]}, nil
}

func (b *Bakery) holdsCheckIn(view Observation, listener string) bool {
	if b.ContentVersion < answerFirstVersion || !b.Entities[listener].Seated {
		return false
	}
	asker := view.ObserverID
	for id, e := range b.Entities {
		if e.Kind != Actor || id == asker || id == listener {
			continue
		}
		seen, err := b.Observe(view.Scope, id)
		if err == nil && seen.hasTarget(asker) && seen.hasTarget(listener) {
			return true
		}
	}
	return false
}

func (x Exchange) lines() (string, string) {
	if x.Words != "" || x.Condition == "" {
		return x.Words, x.Reply
	}
	return checkInWords, checkInAnswers[x.Condition].reply
}

func (x Exchange) message(listenerName string) string {
	words, reply := x.lines()
	said := "You say, “" + words + "”"
	if reply == "" {
		return said
	}
	return said + "\nThe " + listenerName + " replies, “" + reply + "”"
}

// The participant can report what was said even if the coworker's condition
// has changed since. A later listener can confirm who spoke to whom.
func (e NotebookEntry) checkInAccount(name string, aboutHearer, confirm bool) string {
	who := "the " + name
	if aboutHearer {
		who = "you"
	}
	if confirm {
		if aboutHearer {
			who = "me"
		}
		if e.Asked {
			return "So you checked in on " + who + "?"
		}
		if aboutHearer {
			return "So I checked in on you?"
		}
		return "So " + who + " checked in on you?"
	}
	state := checkInAnswers[e.Condition].remembered
	if e.Asked {
		attribution := "They said they "
		if aboutHearer {
			attribution = "You said you "
		}
		return "I checked in on " + who + ". " + attribution + state + "."
	}
	if aboutHearer {
		return "You checked in on me. I told you I " + state + "."
	}
	return "The " + name + " checked in on me. I told them I " + state + "."
}

func (b *Bakery) earlyAnswerWords(fact NotebookEntry, owner, speaker string) (string, bool) {
	if b.ContentVersion < answerFirstVersion || fact.Action != ActionSpeak || fact.Reported == 0 {
		return "", false
	}
	cited, ok := b.entryAt(fact.Subject, fact.Reported)
	if !ok || !cited.Overheard || !cited.Open || cited.Subject != owner {
		return "", false
	}
	var matched bool
	for _, e := range b.Notebooks[fact.Subject] {
		if e.Answered && e.Subject == cited.Target && e.Condition == resting && e.SourceSequence > cited.SourceSequence && e.SourceSequence <= fact.SourceSequence {
			matched = true
		}
	}
	if !matched {
		return "", false
	}
	who := "The " + b.Entities[fact.Subject].Name
	if fact.Subject == speaker {
		who = "You"
	}
	about, verb := "the "+b.Entities[cited.Target].Name, "was"
	if cited.Target == speaker {
		about, verb = "you", "were"
	}
	return who + " said " + about + " " + verb + " taking a breather.", true
}

func (b *Bakery) answeredWords(fact NotebookEntry, speaker string) string {
	about, verb := "the "+b.Entities[fact.Subject].Name, "was"
	if fact.Subject == speaker {
		about, verb = "you", "were"
	}
	return "I said " + about + " " + verb + " taking a breather."
}

func (b *Bakery) stoodWords(fact NotebookEntry, speaker string) string {
	who := "The " + b.Entities[fact.Subject].Name
	if fact.Subject == speaker {
		who = "You"
	}
	return who + " said I was taking a breather."
}

func (e NotebookEntry) overheardCheckIn(asker, listener string) string {
	if e.Open {
		return "I heard the " + asker + " ask the " + listener + " how they were holding up."
	}
	return e.overheardAccount(asker, listener, false, false, false)
}

func (e NotebookEntry) overheardAccount(asker, listener string, aboutAsker, aboutListener, confirm bool) string {
	whoAsked, whoAnswered := "the "+asker, "the "+listener
	state := checkInAnswers[e.Condition].remembered
	if confirm {
		if aboutAsker {
			whoAsked = "me"
		}
		if aboutListener {
			whoAnswered = "me"
		}
		return "So you heard " + whoAsked + " check in on " + whoAnswered + "?"
	}
	if aboutAsker {
		whoAsked = "you"
	}
	if aboutListener {
		whoAnswered = "you"
	}
	return "I heard " + whoAsked + " check in on " + whoAnswered + ". They said they " + state + "."
}
