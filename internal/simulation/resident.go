// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "net/url"

// The baker knows the route between the shopfront and the prep area. Each
// four-tick period sets a destination; the current position determines the
// next action. Time spent opening the door is time spent on this route.
func (b *Bakery) bakerTurn(scope Scope, bakerID string, tick int64) (spatialEffect, bool) {
	doorID := url.PathEscape(scope.WorldID) + "/door"
	door := b.Entities[doorID].Placement.Cell
	goalX := door.X - 1
	if (tick-1)/4%2 == 1 {
		goalX = door.X + 1
	}
	baker := b.Entities[bakerID]
	takingBreak := b.ContentVersion >= restContentVersion && baker.Fatigue == fatigueLimit
	if takingBreak {
		goalX = door.X + 2 // A quiet place at the back of the bakery.
	}
	at := baker.Placement.Cell
	var dir Direction
	switch {
	case at.Y < door.Y:
		dir = South
	case at.Y > door.Y:
		dir = North
	case at.X < goalX:
		dir = East
	case at.X > goalX:
		dir = West
	case takingBreak:
		effect, failure := b.resolveSpatial(bakerID, Proposal{Scope: scope, Action: ActionRest})
		return effect, failure == nil
	default:
		return spatialEffect{}, false
	}
	p := Proposal{Scope: scope, Action: ActionMove, Args: ActionArgs{Direction: dir}}
	effect, failure := b.resolveSpatial(bakerID, p)
	if failure != nil && failure.code == "movement_blocked" {
		if next, _ := at.neighbor(dir); next == door {
			p.Action, p.Args = ActionOpen, ActionArgs{Target: doorID}
			effect, failure = b.resolveSpatial(bakerID, p)
		}
	}
	return effect, failure == nil
}
