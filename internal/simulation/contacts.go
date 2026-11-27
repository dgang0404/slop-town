// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

// Each disclosure records one way to reach a person, who heard it, and when.
type sharedContact struct {
	Owner, Person string
	Address       string `json:",omitempty"`
	Tick          int64
	Sequence      uint64
	Number        string `json:",omitempty"`
}

func (c sharedContact) value(action ActionKind) string {
	if action == ActionCall {
		return c.Number
	}
	return c.Address
}

func (b *Bakery) contact(owner, person string, action ActionKind) sharedContact {
	for _, c := range b.Contacts {
		if c.Owner == owner && c.Person == person && c.value(action) != "" {
			return c
		}
	}
	return sharedContact{}
}

// ContactBook contains only the names and destinations that were shared with the reader.
type ContactBook struct {
	People  []ObservedTarget
	Details map[string]string
}

func (b *Bakery) contactBook(owner string, action ActionKind) ContactBook {
	var book ContactBook
	for _, c := range b.Contacts {
		if c.Owner != owner || c.value(action) == "" {
			continue
		}
		if book.Details == nil {
			book.Details = make(map[string]string)
		}
		book.People = append(book.People, ObservedTarget{Ref: c.Person, Name: b.Entities[c.Person].Name})
		book.Details[c.Person] = c.value(action)
	}
	return book
}

// A known extension is an exact alias for its person. Names retain ordinary noun matching.
func (book ContactBook) targets(query string, action ActionKind) Observation {
	if action == ActionCall {
		for _, person := range book.People {
			if query == book.Details[person.Ref] {
				return Observation{Targets: []ObservedTarget{{Ref: person.Ref, Name: query}}}
			}
		}
	}
	return Observation{Targets: book.People}
}

func (book ContactBook) MatchingTargets(query string, action ActionKind) []ObservedTarget {
	return book.targets(query, action).MatchingTargets(query)
}

var askAddress = SpeechLine{Topic: "mail", Intent: "ask_address", Tone: "curious"}
var offerAddress = SpeechLine{Topic: "mail", Intent: "offer_address", Tone: "warm"}

func (b *Bakery) contactChoices(actor, listener string) []DialogueChoice {
	if b.ContentVersion < mailContentVersion || !(b.recognizes(actor, listener) || b.recognizes(listener, actor)) {
		return nil
	}
	var lines []DialogueChoice
	if b.addressOf(listener) != "" && b.contact(actor, listener, ActionPost).Address == "" {
		lines = append(lines, DialogueChoice{askAddress, "Where can I write to you?"})
	}
	if address := b.addressOf(actor); address != "" && b.contact(listener, actor, ActionPost).Address == "" {
		lines = append(lines, DialogueChoice{offerAddress, "You can write to me at " + address + "."})
	}
	if number, _ := b.phoneLine(listener); number != "" && b.contact(actor, listener, ActionCall).Number == "" {
		lines = append(lines, DialogueChoice{AskPhoneNumber, "What number can I reach you on?"})
	}
	if number, _ := b.phoneLine(actor); number != "" && b.contact(listener, actor, ActionCall).Number == "" {
		lines = append(lines, DialogueChoice{OfferPhoneNumber, b.shareNumberWords(actor)})
	}
	return lines
}
