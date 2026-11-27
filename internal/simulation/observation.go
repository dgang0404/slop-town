// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"errors"
	"maps"
	"net/url"
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
	MailBook      ContactBook
	PhoneBook     ContactBook
	UnreadLetters int
	ActionTarget  string // Resolved object of the observer's action. Empty on a look.
	Date          BakeryDate
	Self          ObserverDetail
	Details       map[string]VisibleDetail
	Witnessed     []WitnessedAction // During this turn. Empty on a look.
}

// ObserverDetail contains only the surroundings and sensations described to the actor.
type ObserverDetail struct {
	Cell                             CellPosition
	Place, Condition, Feeling, Light PerceptionKind
	Cover                            string // Reference to the cover the observer is using.
}

// VisibleDetail describes a target in the observer's field of view.
type VisibleDetail struct {
	Label, State, Location, Holder string
	Recognition, Expression        string
	Cell                           CellPosition
}

// ChoiceLocation keeps a bearing when disambiguating several visible namesakes.
func (s Scene) ChoiceLocation(ref string) string {
	detail := s.Details[ref]
	if s.Landmarks[ref] != "" {
		return detail.Location + ", " + relativeLocation(s.Self.Cell, detail.Cell)
	}
	return detail.Location
}

// WitnessedAction identifies a firsthand sighting at the moment it occurred.
type WitnessedAction struct {
	Actor, Target, Container ObservedTarget
	Action                   ActionKind
}

// ObjectPhrase names the visible object and, for a put, its observed destination.
func (s WitnessedAction) ObjectPhrase() string {
	objects := "the " + s.Target.Name
	if s.Action == ActionHarvest {
		return "a sprig from " + objects
	}
	if s.Action == ActionKnock {
		return "on " + objects
	}
	if s.Action == ActionHide {
		return "behind " + objects
	}
	if s.Action == ActionPut {
		objects += " in the " + s.Container.Name
	}
	return objects
}

func (c *Coordinator) Scene(actorID string) (Scene, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return (State{Clock: c.clock, Bakery: c.bakery}).Scene(c.scope, actorID)
}

func (s State) Scene(scope Scope, actorID string) (Scene, error) {
	if s.Clock == nil {
		return Scene{}, errors.New("scene requires a timeline")
	}
	scene := Scene{Details: make(map[string]VisibleDetail), Date: bakeryDateAt(s.Clock.Tick())}
	view, err := s.Bakery.observe(scope, actorID, &scene)
	scene.Observation = view
	return scene, err
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

// A standing character can see over the crate. Trace sight at crouching
// height only when it is the observer or the person observed who uses cover.
func (b *Bakery) withCover(opaque cellMask, coverID string) cellMask {
	if coverID != "" {
		cell := b.Entities[coverID].Placement.Cell
		opaque[cell.Y][cell.X] = true
	}
	return opaque
}

func (b *Bakery) Observe(scope Scope, actorID string) (Observation, error) {
	return b.observe(scope, actorID, nil)
}

func (b *Bakery) observe(scope Scope, actorID string, scene *Scene) (Observation, error) {
	if b == nil || !scope.Valid() || scope.WorldID != b.WorldID {
		return Observation{}, errors.New("there is no spatial scene")
	}
	actor := b.Entities[actorID]
	at := actor.Placement.Cell
	if actor.Kind != Actor || actor.Placement.HolderID != "" || !b.isFloorCell(at) {
		return Observation{}, errors.New("invalid observing character")
	}
	opaque := b.withCover(b.opaqueCells(), actor.HiddenBehind)
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
	doorway := b.Entities[url.PathEscape(b.WorldID)+"/door"].Placement.Cell
	self := ObserverDetail{Cell: at, Place: b.visiblePlace(at, doorway)}
	if actor.Seated {
		self.Condition = SittingDown
	} else if b.ContentVersion >= practiceContentVersion && actor.Fatigue == fatigueLimit {
		self.Condition = TiredLegs
	}
	switch actor.ReliefTurns {
	case reliefDuration:
		self.Feeling = FeelingRelieved
	case 1, 2:
		self.Feeling = LingeringRelief
	}
	lines := []string{self.Place.Description()}
	if actor.HiddenBehind != "" {
		self.Cover = actor.HiddenBehind
		lines = append(lines, crouchingBehind+b.Entities[self.Cover].Name+".")
	}
	if b.dimAt(at) {
		self.Light = DimLight
		lines = append(lines, self.Light.Description())
	}
	if self.Condition != "" {
		lines = append(lines, self.Condition.Description())
	}
	if self.Feeling != "" {
		lines = append(lines, self.Feeling.Description())
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
		if e.HiddenBehind != "" {
			sight := b.withCover(opaque, e.HiddenBehind)
			if !sight.canSee(at.X, at.Y, pos.X, pos.Y) {
				continue
			}
		}
		if b.ContentVersion < landmarkContentVersion && pos == at && e.Kind == Door {
			self.Place = InBakeryDoorway
			lines[0] = self.Place.Description()
		}
		view.Targets = append(view.Targets, ObservedTarget{Ref: id, Name: e.Name})
		if e.Kind == Planter {
			if view.Plants == nil {
				view.Plants = make(map[string]PlantCondition)
			}
			view.Plants[id] = e.Growth.condition(b.ContentVersion >= regrowthContentVersion)
		}
		if e.Placement.HolderID == actorID {
			if scene != nil {
				scene.Details[id] = VisibleDetail{Label: e.Name, Location: "in your inventory", Holder: actorID}
			}
			lines = append(lines, "You carry the "+e.Name+".")
			continue
		}
		label, glyph, state := appearance(e, b.ContentVersion >= regrowthContentVersion)
		location := relativeLocation(at, pos)
		if holder := e.Placement.HolderID; holder != "" {
			location = "in the " + b.Entities[holder].Name
		}
		if b.ContentVersion >= landmarkContentVersion && e.Kind != Door {
			place := b.visiblePlace(pos, doorway)
			if place != self.Place {
				if view.Landmarks == nil {
					view.Landmarks = make(map[string]PerceptionKind)
				}
				view.Landmarks[id] = place
				if e.Placement.HolderID == "" {
					location = place.landmark()
				} else {
					location += " " + place.landmark()
				}
			}
		}
		lines = append(lines, "You see the "+label+" "+location+".")
		detail := VisibleDetail{Label: label, State: state, Location: location, Holder: e.Placement.HolderID, Cell: pos}
		if b.ContentVersion >= relationshipContentVersion && e.Kind == Actor && b.recognizes(actorID, id) {
			detail.Recognition = "You recognize the " + e.Name + " from their introduction."
			lines = append(lines, detail.Recognition)
		}
		if e.ReliefTurns == reliefDuration {
			detail.Expression = "The " + e.Name + "'s shoulders look relaxed."
			lines = append(lines, detail.Expression)
		}
		if scene != nil {
			scene.Details[id] = detail
		}
		tile := &tiles[pos.Y][pos.X]
		*tile = strings.TrimPrefix(*tile, ".") + glyph
	}
	if scene != nil {
		scene.Self = self
		scene.MailBook = b.contactBook(actorID, ActionPost)
		scene.PhoneBook = b.contactBook(actorID, ActionCall)
		for _, l := range b.Letters {
			if l.To == actorID && l.Delivered && l.ReadTick == 0 {
				scene.UnreadLetters++
			}
		}
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

func appearance(e SpatialEntity, regrowth bool) (label, glyph, state string) {
	switch e.Kind {
	case Telephone:
		if e.Disconnected {
			return e.Name, "T", "disconnected"
		}
		return e.Name, "T", "in working order"
	case Planter:
		return e.Growth.condition(regrowth).appearance(e.Name)
	case Door:
		if e.Open {
			return "open " + e.Name, "/", "open"
		}
		return "closed " + e.Name, "+", "closed"
	case Container:
		if e.Open {
			return "open " + e.Name, "c", "open"
		}
		return "closed " + e.Name, "C", "closed"
	case Actor:
		if e.HiddenBehind != "" {
			return e.Name + " crouching", "a", "crouching"
		}
		if e.Seated {
			return e.Name + " sitting", "a", "sitting"
		}
		return e.Name, "a", "standing"
	default:
		return e.Name, "*", ""
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
