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

type FixtureArgs struct {
	Stream string `json:"stream"`
}

type Proposal struct {
	Scope
	Version   int         `json:"version"`
	RequestID string      `json:"request_id"`
	Action    string      `json:"action"`
	Args      FixtureArgs `json:"args"`
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

// tick_advanced carries Delta. stream_drawn carries Stream and Value.
type EventPayload struct {
	Delta  int64
	Stream string
	Value  uint64
}

type requestKey struct{ actorID, requestID string }

type receipt struct {
	proposal Proposal
	result   Result
}

// Coordinator serializes submissions and owns a timeline's in-memory state.
type Coordinator struct {
	mu       sync.Mutex
	scope    Scope
	clock    *Timeline
	events   []Event
	receipts map[requestKey]receipt
}

func validID(id string) bool { return id != "" && utf8.ValidString(id) }

func NewCoordinator(scope Scope, seed int64) (*Coordinator, error) {
	if !validID(scope.WorldID) || !validID(scope.TimelineID) {
		return nil, fmt.Errorf("world and timeline IDs must be nonempty UTF-8")
	}
	return &Coordinator{scope: scope, clock: NewTimeline(seed), receipts: make(map[requestKey]receipt)}, nil
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

	result := Result{Version: contractVersion, RequestID: p.RequestID}
	reject := func(code, message string) Result {
		result.Status, result.Code, result.Message = "rejected", code, message
		return result
	}
	if p.Version != contractVersion {
		return reject("unsupported_version", "Unsupported action version.")
	}
	if p.Scope != c.scope || !validID(actorID) || !validID(p.RequestID) {
		return reject("invalid_request", "Invalid request context.")
	}
	key := requestKey{actorID, p.RequestID}
	if saved, ok := c.receipts[key]; ok {
		if saved.proposal != p {
			return reject("request_conflict", "That request ID was already used for a different action.")
		}
		return saved.result
	}
	if p.Action != "fixture_step" || !validID(p.Args.Stream) {
		return reject("invalid_request", "Expected a fixture step with a named stream.")
	}

	next := *c.clock
	next.draws = maps.Clone(c.clock.draws)
	if err := next.Advance(1); err != nil {
		return reject("resolution_failed", "The fixture could not advance.")
	}
	value, err := next.Draw(p.Args.Stream)
	if err != nil {
		return reject("resolution_failed", "The fixture could not draw a value.")
	}

	event := func(seq uint64, kind string, payload EventPayload) Event {
		id := fmt.Sprintf("%s/%s/%d", url.PathEscape(c.scope.WorldID), url.PathEscape(c.scope.TimelineID), seq)
		return Event{Scope: c.scope, ID: id, Sequence: seq, Tick: next.Tick(), Version: contractVersion,
			RequestID: p.RequestID, ActorID: actorID, Engine: "fixture", Kind: kind, PayloadVersion: 1, Payload: payload}
	}
	advanced := event(uint64(len(c.events))+1, "tick_advanced", EventPayload{Delta: 1})
	drawn := event(advanced.Sequence+1, "stream_drawn", EventPayload{Stream: p.Args.Stream, Value: value})
	drawn.ParentID = advanced.ID
	result.Status = "accepted"
	result.Observation = Observation{Scope: c.scope, Version: contractVersion, ObserverID: actorID, Text: "The fixture advanced."}

	c.clock = &next
	c.events = append(c.events, advanced, drawn)
	c.receipts[key] = receipt{p, result}
	return result
}
