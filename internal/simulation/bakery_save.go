// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

// The version 1 encoding uses row-major cells and JSON's sorted entity keys.
// Keep the encoded fields and their order stable within this format version.
type bakerySnapshot struct {
	Version int     `json:"version"`
	Bakery  *Bakery `json:"bakery"`
}

func (b *Bakery) Snapshot() ([]byte, error) {
	return json.Marshal(bakerySnapshot{1, b})
}

func RestoreBakery(data []byte, world string, clock *Timeline) (*Bakery, error) {
	var saved bakerySnapshot
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, err
	}
	b := saved.Bakery
	if saved.Version != 1 || b == nil || b.GeneratorVersion != 1 || (b.ContentVersion != 1 && b.ContentVersion != residentContentVersion) {
		return nil, fmt.Errorf("unsupported bakery snapshot or content version")
	}
	if clock == nil || b.WorldID != world || b.Seed != clock.seed || clock.draws["geography"] != 1 {
		return nil, fmt.Errorf("bakery does not match the committed world and clock")
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	prefix := url.PathEscape(world) + "/"
	if b.ContentVersion == residentContentVersion {
		baker := b.Entities[prefix+"baker"]
		if baker.Kind != Actor || !b.isFloorCell(baker.Placement.Cell) {
			return nil, fmt.Errorf("invalid saved baker")
		}
	}
	for name, kind := range map[string]EntityKind{"player": Actor, "door": Door, "crate": Container, "spoon": Item} {
		if b.Entities[prefix+name].Kind != kind {
			return nil, fmt.Errorf("invalid saved bakery entity %q", name)
		}
	}
	// Unmarshal alone accepts duplicate keys and pads or truncates fixed arrays.
	canonical, err := b.Snapshot()
	if err != nil || !bytes.Equal(data, canonical) {
		return nil, fmt.Errorf("invalid canonical bakery snapshot")
	}
	return b, nil
}
