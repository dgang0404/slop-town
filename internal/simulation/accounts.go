// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "fmt"

// HeardAccount groups what one character has heard about one witness's sighting.
// Act is phrased from the reader's perspective, such as "me sit down".
type HeardAccount struct {
	Witness     string
	Act         string
	OwnSighting bool
	Tellings    []AccountTelling
}

type AccountTelling struct {
	Teller      string
	FromWitness bool
	Date        BakeryDate // When the reader first heard it from this person.
}

func (c *Coordinator) Accounts(actor string) ([]HeardAccount, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bakery.Accounts(actor)
}

func (b *Bakery) Accounts(actor string) ([]HeardAccount, error) {
	if b == nil || b.Entities[actor].Kind != Actor {
		return nil, fmt.Errorf("no accounts for this character")
	}
	type sighting struct {
		witness string
		event   uint64
	}
	type telling struct {
		sighting
		teller string
	}
	var accounts []HeardAccount
	index := make(map[sighting]int)
	heard := make(map[telling]bool)
	for _, memory := range b.Notebooks[actor] {
		if memory.Reported == 0 {
			continue
		}
		// Follow only the reference the teller gave this character. The
		// witness's other sightings remain private, even on the same turn.
		source, ok := b.entryAt(memory.Subject, memory.Reported)
		if !ok {
			return nil, fmt.Errorf("missing source for a heard account")
		}
		witness, fact := memory.Subject, source
		if source.Reported != 0 {
			witness = source.Subject
			fact, ok = b.reportedObservation(source)
			if !ok {
				return nil, fmt.Errorf("missing witness for a retold account")
			}
		}
		if !fact.isObservation() {
			continue // Greetings and check-ins belong in the ordinary journal.
		}
		key := sighting{witness, fact.SourceSequence}
		via := telling{key, memory.Subject}
		if heard[via] {
			continue
		}
		heard[via] = true
		i, exists := index[key]
		if !exists {
			i = len(accounts)
			index[key] = i
			accounts = append(accounts, HeardAccount{Witness: b.Entities[witness].Name,
				Act: b.observedAct(fact, actor, ""), OwnSighting: witness == actor})
		}
		accounts[i].Tellings = append(accounts[i].Tellings, AccountTelling{
			Teller: b.Entities[memory.Subject].Name, FromWitness: memory.Subject == witness, Date: bakeryDateAt(memory.Tick),
		})
	}
	return accounts, nil
}
