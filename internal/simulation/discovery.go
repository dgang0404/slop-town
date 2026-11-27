// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "strings"

// Discovery describes a target as it appeared in one recorded observation.
// A container's visible name is all a version-one receipt establishes about it.
type Discovery struct {
	Target    ObservedTarget
	Label     string // Includes visible details such as "closed" or "sitting".
	Location  DiscoveryLocation
	Bearing   string
	Container string
	Text      string // Original observation, including any unfamiliar wording.
}

type DiscoveryLocation string

const (
	LocationUnspecified DiscoveryLocation = "unspecified"
	LocationHere        DiscoveryLocation = "here"
	LocationBearing     DiscoveryLocation = "bearing"
	LocationContainer   DiscoveryLocation = "container"
	LocationInventory   DiscoveryLocation = "inventory"
)

// Decode the descriptions emitted by the version-one observer. Match the
// complete target name before interpreting any location words inside it.
func discoveryFromReceipt(target ObservedTarget, text string) Discovery {
	d := Discovery{Target: target, Label: target.Name, Location: LocationUnspecified, Text: text}
	if text == "You carry the "+target.Name+"." {
		d.Location = LocationInventory
		return d
	}
	line, ok := strings.CutPrefix(text, "You see the ")
	if !ok {
		return d
	}
	line, ok = strings.CutSuffix(line, ".")
	if !ok {
		return d
	}
	for _, label := range []string{target.Name, "closed " + target.Name, "open " + target.Name, target.Name + " sitting", target.Name + " crouching"} {
		where, ok := strings.CutPrefix(line, label+" ")
		if !ok {
			continue
		}
		switch where {
		case "here":
			d.Location = LocationHere
		case "to the north", "to the northeast", "to the east", "to the southeast",
			"to the south", "to the southwest", "to the west", "to the northwest":
			d.Location, d.Bearing = LocationBearing, strings.TrimPrefix(where, "to the ")
		default:
			holder, ok := strings.CutPrefix(where, "in the ")
			if !ok || holder == "" {
				continue
			}
			d.Location, d.Container = LocationContainer, holder
		}
		d.Label = label
		break
	}
	return d
}
