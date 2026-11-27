// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"strings"
)

//go:embed bakery-close-details.en.json
var bakeryCloseDetailsJSON []byte

var bakeryCloseDetails map[string]string

func init() {
	if err := json.Unmarshal(bakeryCloseDetailsJSON, &bakeryCloseDetails); err != nil {
		panic(err)
	}
}

// The target has been resolved in the actor's view. Sample its appearance
// before the workers take their turns and save the finding in the receipt.
func (b *Bakery) examination(view Observation, ref string) (finding, detail string) {
	target := b.Entities[ref]
	observer := b.Entities[view.ObserverID].Placement.Cell
	at, _ := b.PositionOf(ref) // Observe already resolved the visible target's position.
	_, _, state := appearance(target)
	switch target.Kind {
	case Container:
		if target.Open {
			var contents []string
			for _, seen := range view.Targets {
				if b.Entities[seen.Ref].Placement.HolderID == ref {
					contents = append(contents, "the "+seen.Name)
				}
			}
			switch len(contents) {
			case 0:
				state += " with nothing visible inside"
			case 1:
				state += " with " + contents[0] + " inside"
			default:
				last := len(contents) - 1
				state += " with " + strings.Join(contents[:last], ", ") + " and " + contents[last] + " inside"
			}
		}
	case Item:
		if target.RequestedItem != "" {
			state = "pinned to the workbench"
			break
		}
		switch holder := target.Placement.HolderID; holder {
		case view.ObserverID:
			state = "in hand"
		case "":
			state = "on the ground " + relativeLocation(observer, at)
		default:
			state = "in the " + b.Entities[holder].Name
		}
	}
	if landmark := view.Landmarks[ref].landmark(); landmark != "" {
		state += " " + landmark
	}
	finding = "You find the " + target.Name + " " + state
	if target.CloseDetail != "" && inReach(observer, at) && !b.dimAt(at) {
		detail = target.CloseDetail
		finding += ", " + detail
	}
	if target.RequestedItem != "" && detail != "" {
		drawer := url.PathEscape(b.WorldID) + "/workbench"
		if view.hasTarget(drawer) && b.Entities[drawer].Open {
			if view.hasTarget(target.RequestedItem) && b.Entities[target.RequestedItem].Placement.HolderID == drawer {
				finding += ". The requested tool is in the drawer"
			} else {
				finding += ". The requested tool is missing from the open drawer"
			}
		}
	}
	return finding + ".", detail
}
