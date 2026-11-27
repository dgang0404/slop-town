// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"net/url"
	"strings"
)

var AskPhoneNumber = SpeechLine{Topic: "phone", Intent: "ask_number", Tone: "curious"}
var OfferPhoneNumber = SpeechLine{Topic: "phone", Intent: "offer_number", Tone: "warm"}

// The two workers have individual extensions at the bakery handset.
func (b *Bakery) phoneLine(person string) (number, handset string) {
	prefix := url.PathEscape(b.WorldID) + "/"
	role, ok := strings.CutPrefix(person, prefix)
	if !ok || b.ContentVersion < phoneContentVersion || b.Entities[person].Kind != Actor {
		return "", ""
	}
	switch role {
	case "player":
		return "101", prefix + "street-phone"
	case "baker":
		return "201", prefix + "bakery-phone"
	case "porter":
		return "202", prefix + "bakery-phone"
	}
	return "", ""
}

func (b *Bakery) shareNumberWords(person string) string {
	number, handset := b.phoneLine(person)
	return "You can reach me on " + number + " at the " + b.Entities[handset].Name + "."
}

func (b *Bakery) canUsePhone(view Observation, ref string) bool {
	phone := b.Entities[ref]
	return phone.Kind == Telephone && !phone.Disconnected && view.hasTarget(ref) &&
		inReach(b.Entities[view.ObserverID].Placement.Cell, phone.Placement.Cell)
}

func (b *Bakery) outgoingPhone(view Observation, destination string) string {
	for _, target := range view.Targets {
		if target.Ref != destination && b.canUsePhone(view, target.Ref) {
			return target.Ref
		}
	}
	return ""
}

func (b *Bakery) phoneCall(view Observation, p Proposal) (spatialEffect, *actionFailure) {
	if b.ContentVersion < phoneContentVersion {
		return spatialEffect{}, &actionFailure{"action_unavailable", "There is no telephone service here."}
	}
	if !validID(p.Args.Target) || p.Args != (ActionArgs{Target: p.Args.Target}) {
		return spatialEffect{}, &actionFailure{"invalid_request", "Choose one person or telephone number to call."}
	}
	caller := view.ObserverID
	book := b.contactBook(caller, ActionCall)
	ref, failure := book.targets(p.Args.Target, ActionCall).selectTarget(p.Args.Target, p.ByName, p.Observed.Target)
	if failure != nil {
		if failure.code == "ambiguous_target" {
			return spatialEffect{}, &actionFailure{failure.code, "That name matches several people in your phone book."}
		}
		return spatialEffect{}, &actionFailure{"number_unknown", "You need a telephone number they shared with you."}
	}
	number, destination := b.phoneLine(ref)
	from := b.outgoingPhone(view, destination)
	if from == "" {
		if b.canUsePhone(view, destination) {
			return spatialEffect{}, &actionFailure{"same_phone", "That number rings here. Use another phone to call it."}
		}
		return spatialEffect{}, &actionFailure{"phone_unavailable", "You need a working phone within reach."}
	}
	name := b.Entities[ref].Name
	payload := EventPayload{Target: ref, Call: DialedLine{From: from, To: destination, Number: number}}
	effect := spatialEffect{kind: "call_missed", entityID: caller, entity: b.Entities[caller], payload: payload,
		message: "You get no answer on the " + name + "’s line."}
	listening, err := b.Observe(view.Scope, ref)
	receiver := b.Entities[ref]
	if err != nil || !b.canUsePhone(listening, destination) || receiver.Seated || receiver.HiddenBehind != "" || receiver.Fatigue == fatigueLimit {
		return effect, nil
	}

	// A check-in over a working line uses the same words and firsthand knowledge as one in person.
	effect.kind = "call_answered"
	x := Exchange{Question: checkInQuestion, Condition: receiver.workCondition()}
	question, reply := x.lines()
	x.Words, x.Reply = "It’s the "+b.Entities[caller].Name+". "+question, reply
	effect.payload.Dialogue = x
	effect.message = "The " + name + " answers your call. You ask, “" + x.Words + "” The " + name + " replies, “" + reply + "”"
	return effect, nil
}

// DialedLine identifies the two physical handsets and the extension dialed for replay.
type DialedLine struct{ From, To, Number string }
