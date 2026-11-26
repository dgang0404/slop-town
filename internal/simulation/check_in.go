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
	}
	return spatialEffect{kind: "actors_conversed", payload: EventPayload{Target: listener, Dialogue: x},
		entityID: view.ObserverID, entity: b.Entities[view.ObserverID]}, nil
}

func (x Exchange) lines() (string, string) {
	if x.Words != "" || x.Condition == "" {
		return x.Words, x.Reply
	}
	return checkInWords, checkInAnswers[x.Condition].reply
}

func (x Exchange) message(listenerName string) string {
	words, reply := x.lines()
	return "You say, “" + words + "”\nThe " + listenerName + " replies, “" + reply + "”"
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
