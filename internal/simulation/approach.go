// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import "strings"

func walkArrivalMessage(name string) string {
	return "You are within reach of the " + name + "."
}

// SplitWalkArrival separates the destination reached from the other news in
// a version-one walk receipt. Use the selected reference and the name visible
// on that turn, including when two destinations share a name.
func (r Result) SplitWalkArrival() (string, ObservedTarget) {
	i := strings.LastIndexByte(r.Message, '\n')
	if i < 0 || r.SelectedTarget == "" || r.ContinueWalk {
		return r.Message, ObservedTarget{}
	}
	for _, target := range r.Observation.Targets {
		if target.Ref == r.SelectedTarget && r.Message[i+1:] == walkArrivalMessage(target.Name) {
			return r.Message[:i], target
		}
	}
	return r.Message, ObservedTarget{}
}

// Resolve one ordinary step on a shortest visible route. Each subsequent
// step must be submitted against the new state after the residents act.
func (b *Bakery) approach(actorID, target string) (spatialEffect, *actionFailure) {
	actor := b.Entities[actorID]
	start := actor.Placement.Cell
	dest, err := b.PositionOf(target)
	if err != nil {
		return spatialEffect{}, targetUnavailable()
	}
	if inReach(start, dest) {
		return spatialEffect{}, &actionFailure{"within_reach", "You are already within reach of the " + b.Entities[target].Name + "."}
	}
	type step struct {
		at    CellPosition
		first Direction
	}
	queue := []step{{at: start}}
	seen := cellMask{}
	seen[start.Y][start.X] = true
	opaque := b.opaqueCells()
	sight := b.withCover(opaque, actor.HiddenBehind)
	for i := 0; i < len(queue); i++ {
		curr := queue[i]
		for _, dir := range []Direction{North, East, South, West} {
			next, _ := curr.at.neighbor(dir)
			if !b.isFloorCell(next) || seen[next.Y][next.X] || opaque[next.Y][next.X] || !sight.canSee(start.X, start.Y, next.X, next.Y) {
				continue
			}
			seen[next.Y][next.X] = true
			first := curr.first
			if first == "" {
				first = dir
			}
			if inReach(next, dest) {
				effect, failure := b.move(actorID, actor, first)
				effect.payload.Target = target
				effect.message = "You walk toward the " + b.Entities[target].Name + "."
				return effect, failure
			}
			queue = append(queue, step{next, first})
		}
	}
	return spatialEffect{}, &actionFailure{"route_blocked", "You can't see a clear route there."}
}
