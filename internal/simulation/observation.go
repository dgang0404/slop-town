// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"errors"
	"maps"
	"slices"
	"strings"
)

// Observe reads the current scene without submitting a request or advancing time.
func (c *Coordinator) Observe(actorID string) (Observation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bakery.Observe(c.scope, actorID)
}

// Scene carries live presentation details separately from the versioned observation in a receipt.
type Scene struct {
	Observation
	Locations map[string]string
}

func (c *Coordinator) Scene(actorID string) (Scene, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bakery.Scene(c.scope, actorID)
}

func (b *Bakery) Scene(scope Scope, actorID string) (Scene, error) {
	locations := make(map[string]string)
	view, err := b.observe(scope, actorID, locations)
	return Scene{Observation: view, Locations: locations}, err
}

type cellMask [bakeryHeight][bakeryWidth]bool

func sign(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}

// Trace between cell centers. Crossing a corner is possible if either side
// is clear. The first opaque cell itself is visible.
func (opaque *cellMask) canSee(x, y, destX, destY int) bool {
	stepX, stepY := sign(destX-x), sign(destY-y)
	dx, dy := (destX-x)*stepX, (destY-y)*stepY
	for crossedX, crossedY := 0, 0; x != destX || y != destY; {
		// Compare the distances to the next vertical and horizontal edges.
		order := (2*crossedX+1)*dy - (2*crossedY+1)*dx
		if order == 0 && opaque[y][x+stepX] && opaque[y+stepY][x] {
			return false
		}
		if order <= 0 {
			x += stepX
			crossedX++
		}
		if order >= 0 {
			y += stepY
			crossedY++
		}
		if (x != destX || y != destY) && opaque[y][x] {
			return false
		}
	}
	return true
}

func (b *Bakery) opaqueCells() cellMask {
	var opaque cellMask
	for y, row := range b.Cells {
		for x, cell := range row {
			opaque[y][x] = cell.Wall
		}
	}
	for _, e := range b.Entities {
		if e.Kind == Door && !e.Open && b.isFloorCell(e.Placement.Cell) {
			cell := e.Placement.Cell
			opaque[cell.Y][cell.X] = true
		}
	}
	return opaque
}

func (b *Bakery) Observe(scope Scope, actorID string) (Observation, error) {
	return b.observe(scope, actorID, nil)
}

func (b *Bakery) observe(scope Scope, actorID string, locations map[string]string) (Observation, error) {
	if b == nil || !scope.Valid() || scope.WorldID != b.WorldID {
		return Observation{}, errors.New("there is no spatial scene")
	}
	actor := b.Entities[actorID]
	at := actor.Placement.Cell
	if actor.Kind != Actor || actor.Placement.HolderID != "" || !b.isFloorCell(at) {
		return Observation{}, errors.New("invalid observing character")
	}
	opaque := b.opaqueCells()
	var visible cellMask
	var tiles [bakeryHeight][bakeryWidth]string
	for y, row := range b.Cells {
		for x, cell := range row {
			visible[y][x] = opaque.canSee(at.X, at.Y, x, y)
			switch {
			case !visible[y][x]:
				tiles[y][x] = "?"
			case cell.Wall:
				tiles[y][x] = "#"
			default:
				tiles[y][x] = "."
			}
		}
	}
	tiles[at.Y][at.X] = "@"
	view := Observation{Scope: scope, Version: contractVersion, ObserverID: actorID}
	lines := []string{"You are in the bakery."}
	if b.Cells[at.Y][at.X].SpaceID == b.StreetID {
		lines[0] = "You are on the street."
	}
	if actor.Seated {
		lines = append(lines, "You are sitting down.")
	} else if b.ContentVersion >= practiceContentVersion && actor.Fatigue == fatigueLimit {
		lines = append(lines, "Your legs feel tired.")
	}
	switch actor.ReliefTurns {
	case reliefDuration:
		lines = append(lines, "You feel relieved.")
	case 1, 2:
		lines = append(lines, "A little of that relief lingers.")
	}
	for _, id := range slices.Sorted(maps.Keys(b.Entities)) {
		if id == actorID {
			continue
		}
		pos, err := b.PositionOf(id)
		if err != nil || !visible[pos.Y][pos.X] || !b.contentsVisible(id, actorID) {
			continue
		}
		e := b.Entities[id]
		if pos == at && e.Kind == Door {
			lines[0] = "You are in the bakery doorway."
		}
		view.Targets = append(view.Targets, ObservedTarget{Ref: id, Name: e.Name})
		if e.Placement.HolderID == actorID {
			if locations != nil {
				locations[id] = "in your inventory"
			}
			lines = append(lines, "You carry the "+e.Name+".")
			continue
		}
		label, glyph := appearance(e)
		location := relativeLocation(at, pos)
		if holder := e.Placement.HolderID; holder != "" {
			location = "in the " + b.Entities[holder].Name
		}
		if locations != nil {
			locations[id] = location
		}
		lines = append(lines, "You see the "+label+" "+location+".")
		if b.ContentVersion >= relationshipContentVersion && e.Kind == Actor && b.recognizes(actorID, id) {
			lines = append(lines, "You recognize the "+e.Name+" from their introduction.")
		}
		if e.ReliefTurns == reliefDuration {
			lines = append(lines, "The "+e.Name+"'s shoulders look relaxed.")
		}
		tile := &tiles[pos.Y][pos.X]
		*tile = strings.TrimPrefix(*tile, ".") + glyph
	}
	view.Text = strings.Join(lines, "\n")
	view.Grid = renderTiles(tiles)
	return view, nil
}

// PositionOf has already validated this chain. Each enclosing lid must be
// open, and only the observer can inspect their own inventory.
func (b *Bakery) contentsVisible(id, actorID string) bool {
	for holder := b.Entities[id].Placement.HolderID; holder != ""; holder = b.Entities[holder].Placement.HolderID {
		if holder == actorID {
			return true
		}
		e := b.Entities[holder]
		if e.Kind != Container || !e.Open {
			return false
		}
	}
	return true
}

func appearance(e SpatialEntity) (label, glyph string) {
	switch e.Kind {
	case Door:
		if e.Open {
			return "open " + e.Name, "/"
		}
		return "closed " + e.Name, "+"
	case Container:
		if e.Open {
			return "open " + e.Name, "c"
		}
		return "closed " + e.Name, "C"
	case Actor:
		if e.Seated {
			return e.Name + " sitting", "a"
		}
		return e.Name, "a"
	default:
		return e.Name, "*"
	}
}

func relativeLocation(from, to CellPosition) string {
	if from == to {
		return "here"
	}
	ns := [...]string{"north", "", "south"}
	ew := [...]string{"west", "", "east"}
	return "to the " + ns[sign(to.Y-from.Y)+1] + ew[sign(to.X-from.X)+1]
}

// Keep all symbols on a shared tile, including a spoon dropped at the observer's feet.
func renderTiles(tiles [bakeryHeight][bakeryWidth]string) string {
	var widths [bakeryWidth]int
	for _, row := range tiles {
		for x, tile := range row {
			widths[x] = max(widths[x], len(tile))
		}
	}
	var lines []string
	for _, row := range tiles {
		var line strings.Builder
		for x, tile := range row {
			line.WriteString(tile)
			if x+1 < len(row) {
				line.WriteString(strings.Repeat(" ", widths[x]-len(tile)+1))
			}
		}
		lines = append(lines, line.String())
	}
	return strings.Join(lines, "\n")
}

func (v Observation) hasTarget(ref string) bool {
	return slices.ContainsFunc(v.Targets, func(t ObservedTarget) bool { return t.Ref == ref })
}

// MatchingTargets prefers a complete name to a sequence of whole words in a name.
// Targets retain the stable order of this actor's observation.
func (v Observation) MatchingTargets(query string) []ObservedTarget {
	if query == "" {
		return nil
	}
	folded := strings.ToLower(query)
	var exact, foldedExact, short []ObservedTarget
	for _, target := range v.Targets {
		name := strings.ToLower(target.Name)
		switch {
		case target.Name == query:
			exact = append(exact, target)
		case name == folded:
			foldedExact = append(foldedExact, target)
		case strings.Contains(" "+name+" ", " "+folded+" "):
			short = append(short, target)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	if len(foldedExact) > 0 {
		return foldedExact
	}
	return short
}

// selectTarget accepts one visible reference, or one unambiguous visible name.
func (v Observation) selectTarget(query string, byName bool, observed string) (string, *actionFailure) {
	if !byName {
		if v.hasTarget(query) {
			return query, nil
		}
		return "", targetUnavailable()
	}
	matches := v.MatchingTargets(query)
	if observed != "" {
		for _, t := range matches {
			if t.Ref == observed {
				return observed, nil
			}
		}
		return "", targetUnavailable()
	}
	switch len(matches) {
	case 0:
		return "", targetUnavailable()
	case 1:
		return matches[0].Ref, nil
	default:
		return "", &actionFailure{"ambiguous_target", "That name matches more than one visible target."}
	}
}
