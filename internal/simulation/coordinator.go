// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"sync"
	"unicode/utf8"
)

const contractVersion = 1

// Scope identifies a single history of a generated world.
type Scope struct {
	WorldID    string `json:"world_id"`
	TimelineID string `json:"timeline_id"`
}

type ActionKind string

const (
	ActionMove  ActionKind = "move"
	ActionOpen  ActionKind = "open"
	ActionClose ActionKind = "close"
)

type ActionArgs struct {
	Stream    string    `json:"stream,omitempty"`
	Direction Direction `json:"direction,omitempty"`
	Target    string    `json:"target,omitempty"`
}

type FixtureArgs = ActionArgs

type Proposal struct {
	Scope
	Version   int        `json:"version"`
	RequestID string     `json:"request_id"`
	Action    ActionKind `json:"action"`
	Args      ActionArgs `json:"args"`
}

type Observation struct {
	Scope
	Version    int    `json:"version"`
	ObserverID string `json:"observer_id"`
	Text       string `json:"text"`
}

type Result struct {
	Version     int         `json:"version"`
	RequestID   string      `json:"request_id"`
	Status      string      `json:"status"`
	Code        string      `json:"code,omitempty"`
	Message     string      `json:"message,omitempty"`
	Observation Observation `json:"observation,omitzero"`
}

// Event is a developer-facing record. ParentID points to an earlier event
// in the same history. A root event is caused directly by its request.
type Event struct {
	Scope
	ID             string
	Sequence       uint64
	Tick           int64
	Version        int
	RequestID      string
	ActorID        string
	Engine         string
	Kind           string
	ParentID       string
	PayloadVersion int
	Payload        EventPayload
}

// Spatial events record the destination cell or the door's stable ID.
type EventPayload struct {
	Delta       int64
	Stream      string
	Value       uint64
	Destination CellPosition `json:",omitzero"`
	Target      string       `json:",omitempty"`
}

type requestKey struct{ actorID, requestID string }

type Receipt struct {
	Proposal Proposal
	Result   Result
}

// Coordinator serializes submissions and owns a timeline's in-memory state.
type Coordinator struct {
	mu       sync.Mutex
	scope    Scope
	clock    *Timeline
	bakery   *Bakery
	events   []Event
	receipts map[requestKey]*Receipt
}

func validID(id string) bool { return id != "" && utf8.ValidString(id) }

func NewCoordinator(scope Scope, seed int64) (*Coordinator, error) {
	if !validID(scope.WorldID) || !validID(scope.TimelineID) {
		return nil, fmt.Errorf("world and timeline IDs must be nonempty UTF-8")
	}
	return &Coordinator{scope: scope, clock: NewTimeline(seed), receipts: make(map[requestKey]*Receipt)}, nil
}

// History returns a detached copy for diagnostics.
func (c *Coordinator) History() []Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.events)
}

// Submit binds the actor from the caller's context. A committed request can
// be retried after later submissions and still returns its original result.
func (c *Coordinator) Submit(actorID string, p Proposal) Result {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := requestKey{actorID, p.RequestID}
	result, step := resolve(c.scope, c.clock, c.bakery, uint64(len(c.events)), actorID, p, c.receipts[key])
	if step != nil {
		c.clock = step.Clock
		c.bakery = step.Bakery
		c.events = append(c.events, step.Events...)
		c.receipts[key] = &Receipt{p, result}
	}
	return result
}

// Transition holds all state and events staged for a single commit.
type Transition struct {
	Clock  *Timeline
	Bakery *Bakery
	Events []Event
}

// Resolve stages a Phase 1 fixture command. A nil transition denotes
// a rejection or an identical retry. Callers publish only after commit.
func Resolve(scope Scope, clock *Timeline, lastSequence uint64, actorID string, p Proposal, saved *Receipt) (Result, *Transition) {
	return resolve(scope, clock, nil, lastSequence, actorID, p, saved)
}

func resolve(scope Scope, clock *Timeline, bakery *Bakery, lastSequence uint64, actorID string, p Proposal, saved *Receipt) (Result, *Transition) {
	result := Result{Version: contractVersion, RequestID: p.RequestID}
	reject := func(code, message string) Result {
		result.Status, result.Code, result.Message = "rejected", code, message
		return result
	}
	if p.Version != contractVersion {
		return reject("unsupported_version", "Unsupported action version."), nil
	}
	if !validID(scope.WorldID) || !validID(scope.TimelineID) || p.Scope != scope || !validID(actorID) || !validID(p.RequestID) {
		return reject("invalid_request", "Invalid request context."), nil
	}
	if saved != nil {
		if saved.Proposal != p {
			return reject("request_conflict", "That request ID was already used for a different action."), nil
		}
		return saved.Result, nil
	}
	if bakery == nil && (p.Action != "fixture_step" || !validID(p.Args.Stream) || p.Args.Direction != "" || p.Args.Target != "") {
		return reject("invalid_request", "Expected a fixture step with a named stream."), nil
	}
	next := *clock
	if err := next.Advance(1); err != nil {
		return reject("resolution_failed", "The fixture could not advance."), nil
	}
	var nextBakery *Bakery
	var engine, kind, message string
	var payload EventPayload
	if bakery == nil {
		next.draws = maps.Clone(clock.draws)
		value, err := next.Draw(p.Args.Stream)
		if err != nil {
			return reject("resolution_failed", "The fixture could not draw a value."), nil
		}
		engine, kind, message = "fixture", "stream_drawn", "The fixture advanced."
		payload = EventPayload{Stream: p.Args.Stream, Value: value}
	} else {
		staged := *bakery
		staged.Entities = maps.Clone(bakery.Entities)
		effect, failure := staged.applySpatial(actorID, p)
		if failure != nil {
			return reject(failure.code, failure.message), nil
		}
		nextBakery = &staged
		engine, kind, payload, message = "spatial", effect.kind, effect.payload, effect.message
	}
	event := func(seq uint64, kind string, payload EventPayload) Event {
		id := fmt.Sprintf("%s/%s/%d", url.PathEscape(scope.WorldID), url.PathEscape(scope.TimelineID), seq)
		return Event{Scope: scope, ID: id, Sequence: seq, Tick: next.Tick(), Version: contractVersion,
			RequestID: p.RequestID, ActorID: actorID, Engine: engine, Kind: kind, PayloadVersion: 1, Payload: payload}
	}
	advanced := event(lastSequence+1, "tick_advanced", EventPayload{Delta: 1})
	action := event(advanced.Sequence+1, kind, payload)
	action.ParentID = advanced.ID
	result.Status = "accepted"
	result.Observation = Observation{Scope: scope, Version: contractVersion, ObserverID: actorID, Text: message}

	return result, &Transition{Clock: &next, Bakery: nextBakery, Events: []Event{advanced, action}}
}
