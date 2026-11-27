// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const mailTurns = 3
const invitation = "Please write back when you have a moment."

type letter struct {
	Post                    uint64 // Sequence of the posting event, unique on this timeline.
	From, To, Address, Body string
	Tick, ReadTick          int64
	Delivered               bool
	ReplyTo                 uint64 `json:",omitempty"`
}

func (b *Bakery) addressOf(person string) string {
	role, ok := strings.CutPrefix(person, url.PathEscape(b.WorldID)+"/")
	if !ok || b.ContentVersion < mailContentVersion || b.Entities[person].Kind != Actor {
		return ""
	}
	switch role {
	case "player":
		return "Box 1, Town Post"
	case "baker":
		return "Box 2, Town Post"
	case "porter":
		return "Box 3, Town Post"
	}
	return ""
}

func validLetterBody(body string) bool {
	return body != "" && strings.TrimSpace(body) == body && utf8.ValidString(body) && utf8.RuneCountInString(body) <= 500 &&
		!strings.ContainsFunc(body, func(r rune) bool { return !unicode.IsPrint(r) })
}

func (b *Bakery) postLetter(actor string, p Proposal) (spatialEffect, *actionFailure) {
	args := p.Args
	if args != (ActionArgs{Target: args.Target, Body: args.Body}) || !validID(args.Target) || !validLetterBody(args.Body) {
		return spatialEffect{}, &actionFailure{"invalid_request", "Write one line of up to 500 printable characters."}
	}
	targets := Observation{Targets: b.contactBook(actor, ActionPost).People}
	ref, failure := targets.selectTarget(args.Target, p.ByName, p.Observed.Target)
	if failure != nil {
		if failure.code == "ambiguous_target" {
			return spatialEffect{}, &actionFailure{failure.code, "That name matches several people in your address book."}
		}
		return spatialEffect{}, &actionFailure{"address_unknown", "You need a mailing address they shared with you."}
	}
	c := b.contact(actor, ref, ActionPost)
	return spatialEffect{kind: "letter_posted", entityID: actor, entity: b.Entities[actor],
		payload: EventPayload{Target: ref, Address: c.Address, Body: args.Body},
		message: "You post a letter to the " + b.Entities[ref].Name + " at " + c.Address + "."}, nil
}

func (b *Bakery) readLetter(actor string) (spatialEffect, *actionFailure) {
	for _, l := range b.Letters {
		if l.To == actor && l.Delivered && l.ReadTick == 0 {
			return spatialEffect{kind: "letter_read", entityID: actor, entity: b.Entities[actor], payload: EventPayload{LetterSequence: l.Post, Target: l.From, Body: l.Body},
				message: "You read a letter from the " + b.Entities[l.From].Name + ". “" + l.Body + "”"}, nil
		}
	}
	return spatialEffect{}, &actionFailure{"no_letter", "You have no unread letters."}
}

// Workers can open their post or answer it during a free turn. A reply never asks for another acknowledgment.
func (b *Bakery) residentMail(actor string) spatialEffect {
	if effect, failure := b.readLetter(actor); failure == nil {
		return effect
	}
	answered := make(map[uint64]bool)
	for _, l := range b.Letters {
		if l.From == actor && l.ReplyTo != 0 {
			answered[l.ReplyTo] = true
		}
	}
	for _, l := range b.Letters {
		if l.To != actor || l.ReadTick == 0 || l.ReplyTo != 0 || answered[l.Post] || b.contact(actor, l.From, ActionPost).Address == "" {
			continue
		}
		effect, failure := b.postLetter(actor, Proposal{Args: ActionArgs{Target: l.From, Body: "Your letter reached me. Thank you for writing."}})
		if failure == nil {
			effect.payload.LetterSequence = l.Post
			return effect
		}
	}
	return spatialEffect{}
}

func (b *Bakery) deliverMail(events []Event) []Event {
	turn := events[0]
	for i, l := range b.Letters {
		if l.Delivered || turn.Tick-l.Tick < mailTurns {
			continue
		}
		b.Letters[i].Delivered = true
		e := turn.child(events[len(events)-1].Sequence+1, l.To, "post", "letter_delivered", EventPayload{LetterSequence: l.Post})
		e.ParentID = eventID(turn.Scope, l.Post)
		events = append(events, e)
	}
	return events
}

func (b *Bakery) validateCorrespondence(tick int64) error {
	seen := make(map[[2]string]sharedContact)
	seenNumbers := make(map[[2]string]bool)
	var lastShare uint64
	for _, c := range b.Contacts {
		key := [2]string{c.Owner, c.Person}
		if c.Owner == c.Person || b.Entities[c.Owner].Kind != Actor || c.Tick <= 0 || c.Tick > tick || c.Sequence <= lastShare {
			return fmt.Errorf("invalid contact disclosure")
		}
		lastShare = c.Sequence
		if c.Number != "" {
			number, _ := b.phoneLine(c.Person)
			if c.Number != number || c.Address != "" || seenNumbers[key] {
				return fmt.Errorf("invalid shared telephone number")
			}
			seenNumbers[key] = true
		} else {
			if c.Address == "" || c.Address != b.addressOf(c.Person) || seen[key].Sequence != 0 {
				return fmt.Errorf("invalid shared mailing address")
			}
			seen[key] = c
		}
	}
	posted := make(map[uint64]letter)
	answered := make(map[uint64]bool)
	var previous uint64
	for _, l := range b.Letters {
		c := seen[[2]string{l.From, l.To}]
		if l.Post <= previous || l.Tick <= 0 || l.Tick > tick || c.Address == "" || c.Address != l.Address || c.Tick > l.Tick || c.Sequence >= l.Post ||
			!validLetterBody(l.Body) || l.Delivered != (tick-l.Tick >= mailTurns) ||
			(l.ReadTick != 0 && (!l.Delivered || l.ReadTick-l.Tick <= mailTurns || l.ReadTick > tick)) {
			return fmt.Errorf("invalid saved letter at sequence %d", l.Post)
		}
		if l.ReplyTo != 0 {
			first, ok := posted[l.ReplyTo]
			if !ok || first.To != l.From || first.From != l.To || first.ReplyTo != 0 || answered[l.ReplyTo] || first.ReadTick == 0 || first.ReadTick >= l.Tick {
				return fmt.Errorf("invalid reply at sequence %d", l.Post)
			}
			answered[l.ReplyTo] = true
		}
		posted[l.Post], previous = l, l.Post
	}
	return nil
}
