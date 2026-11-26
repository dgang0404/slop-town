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
	ActionWait      ActionKind = "wait"
	ActionRest      ActionKind = "rest"
	ActionMove      ActionKind = "move"
	ActionOpen      ActionKind = "open"
	ActionClose     ActionKind = "close"
	ActionTake      ActionKind = "take"
	ActionPut       ActionKind = "put"
	ActionDrop      ActionKind = "drop"
	ActionIntroduce ActionKind = "introduce"
)

type ActionArgs struct {
	Stream    string    `json:"stream,omitempty"`
	Direction Direction `json:"direction,omitempty"`
	Target    string    `json:"target,omitempty"`
	Container string    `json:"container,omitempty"` // Destination of a put.
}

type FixtureArgs = ActionArgs

// A named command may carry the references it matched in the caller's view.
// These guard its first execution. The name remains the request's retry identity.
type NameBindings struct {
	Target    string `json:"target,omitempty"`
	Container string `json:"container,omitempty"`
}

type Proposal struct {
	Scope
	Version   int          `json:"version"`
	RequestID string       `json:"request_id"`
	ByName    bool         `json:"by_name,omitempty"` // Resolve target names against the acting character's view.
	Action    ActionKind   `json:"action"`
	Args      ActionArgs   `json:"args"`
	Observed  NameBindings `json:"observed,omitzero"`
}

type Observation struct {
	Scope
	Version    int              `json:"version"`
	ObserverID string           `json:"observer_id"`
	Text       string           `json:"text"`
	Grid       string           `json:"grid,omitempty"`
	Targets    []ObservedTarget `json:"targets,omitempty"`
}

// ObservedTarget is a reference the observer may select in the current view.
type ObservedTarget struct {
	Ref  string `json:"ref"`
	Name string `json:"name"`
}

type Result struct {
	Version        int         `json:"version"`
	RequestID      string      `json:"request_id"`
	Status         string      `json:"status"`
	Code           string      `json:"code,omitempty"`
	Message        string      `json:"message,omitempty"`
	Observation    Observation `json:"observation,omitzero"`
	SelectedTarget string      `json:"selected_target,omitempty"`
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

func eventID(scope Scope, sequence uint64) string {
	return fmt.Sprintf("%s/%s/%d", url.PathEscape(scope.WorldID), url.PathEscape(scope.TimelineID), sequence)
}

// All effects of a turn share its scope, tick and causing request.
func (parent Event) child(sequence uint64, actor, engine, kind string, payload EventPayload) Event {
	e := parent
	e.ID, e.ParentID, e.Sequence = eventID(e.Scope, sequence), parent.ID, sequence
	e.ActorID, e.Engine, e.Kind, e.Payload = actor, engine, kind, payload
	return e
}

// A transfer identifies the item in Target and its new parent in HolderID or Destination.
type EventPayload struct {
	Delta             int64
	Stream            string
	Value             uint64
	Destination       CellPosition `json:",omitzero"`
	Target            string       `json:",omitempty"`
	HolderID          string       `json:",omitempty"`
	EndurancePractice bool         `json:",omitempty"`
	ReliefStarted     bool         `json:",omitempty"`
}

type requestKey struct{ actorID, requestID string }

// CommandInput preserves the player's noun phrases and one-based answers for
// a reference proposal. The simulation replays the resolved proposal itself.
type CommandInput struct {
	Target          string `json:"target"`
	Container       string `json:"container,omitempty"`
	TargetChoice    int    `json:"target_choice,omitempty"`
	ContainerChoice int    `json:"container_choice,omitempty"`
}

type Receipt struct {
	Proposal Proposal
	Result   Result
	Input    *CommandInput `json:"input,omitempty"`
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

func (s Scope) Valid() bool { return validID(s.WorldID) && validID(s.TimelineID) }

func NewCoordinator(scope Scope, seed int64) (*Coordinator, error) {
	if !scope.Valid() {
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
	result, step := (State{Clock: c.clock, Bakery: c.bakery}).Resolve(c.scope, uint64(len(c.events)), actorID, p, c.receipts[key])
	if step != nil {
		c.clock = step.Clock
		c.bakery = step.Bakery
		c.events = append(c.events, step.Events...)
		c.receipts[key] = &Receipt{Proposal: p, Result: result}
	}
	result.Observation.Targets = slices.Clone(result.Observation.Targets)
	return result
}

// State is the authoritative simulation state at one point in a timeline.
type State struct {
	Clock  *Timeline
	Bakery *Bakery
}

// Transition holds the staged state and events for a single commit.
type Transition struct {
	State
	Events []Event
}

// Resolve stages a Phase 1 fixture command. A nil transition denotes
// a rejection or an identical retry. Callers publish only after commit.
func Resolve(scope Scope, clock *Timeline, lastSequence uint64, actorID string, p Proposal, saved *Receipt) (Result, *Transition) {
	return (State{Clock: clock}).Resolve(scope, lastSequence, actorID, p, saved)
}

func (s State) Resolve(scope Scope, lastSequence uint64, actorID string, p Proposal, saved *Receipt) (Result, *Transition) {
	clock, bakery := s.Clock, s.Bakery
	result := Result{Version: contractVersion, RequestID: p.RequestID}
	reject := func(code, message string) Result {
		result.Status, result.Code, result.Message = "rejected", code, message
		return result
	}
	if p.Version != contractVersion {
		return reject("unsupported_version", "Unsupported action version."), nil
	}
	if !scope.Valid() || p.Scope != scope || !validID(actorID) || !validID(p.RequestID) ||
		clock == nil || (bakery != nil && bakery.WorldID != scope.WorldID) {
		return reject("invalid_request", "Invalid request context."), nil
	}
	if p.Observed != (NameBindings{}) && (!p.ByName || p.Observed.Target == "" || p.Args.Target == "" ||
		(p.Observed.Container != "" && p.Action != ActionPut)) {
		return reject("invalid_request", "Invalid observed targets."), nil
	}
	if saved != nil {
		original := saved.Proposal
		// A retry can arrive from a later view. Explicit numbered selections
		// are reference proposals, whose arguments must still match exactly.
		original.Observed, p.Observed = NameBindings{}, NameBindings{}
		if original != p {
			return reject("request_conflict", "That request ID was already used for a different action."), nil
		}
		return saved.Result, nil
	}
	if bakery == nil && (p.Action != "fixture_step" || p.ByName || !validID(p.Args.Stream) || p.Args != (ActionArgs{Stream: p.Args.Stream})) {
		return reject("invalid_request", "Expected a fixture step with a named stream."), nil
	}
	next := *clock
	next.draws = maps.Clone(clock.draws)
	if err := next.Advance(1); err != nil {
		return reject("resolution_failed", "The fixture could not advance."), nil
	}
	var nextBakery *Bakery
	var primary spatialEffect
	var engine, kind, message string
	var payload EventPayload
	if bakery == nil {
		value, err := next.Draw(p.Args.Stream)
		if err != nil {
			return reject("resolution_failed", "The fixture could not draw a value."), nil
		}
		engine, kind, message = "fixture", "stream_drawn", "The fixture advanced."
		payload = EventPayload{Stream: p.Args.Stream, Value: value}
	} else {
		var failure *actionFailure
		primary, failure = bakery.resolveSpatial(actorID, p)
		if failure != nil {
			return reject(failure.code, failure.message), nil
		}
		staged := *bakery
		staged.Entities = maps.Clone(bakery.Entities)
		staged.Notebooks = maps.Clone(bakery.Notebooks)
		nextBakery = &staged
		engine, kind, payload, message = "spatial", primary.kind, primary.payload, primary.message
	}
	advanced := Event{Scope: scope, ID: eventID(scope, lastSequence+1), Sequence: lastSequence + 1,
		Tick: next.Tick(), Version: contractVersion, RequestID: p.RequestID, ActorID: actorID,
		Engine: engine, Kind: "tick_advanced", PayloadVersion: 1, Payload: EventPayload{Delta: 1}}
	action := advanced.child(advanced.Sequence+1, actorID, engine, kind, payload)
	events := []Event{advanced, action}
	result.Status = "accepted"
	if nextBakery == nil {
		result.Observation = Observation{Scope: scope, Version: contractVersion, ObserverID: actorID, Text: message}
	} else {
		nextBakery.applySpatial(action, primary)
		residents, err := nextBakery.advanceResidents(&next, action)
		if err != nil {
			return reject("resolution_failed", "The resident could not finish the turn."), nil
		}
		events = append(events, residents...)
		if nextBakery.ContentVersion >= emotionContentVersion {
			nextBakery.advanceRelief(events)
		}
		view, err := nextBakery.Observe(scope, actorID)
		if err != nil {
			return reject("resolution_failed", "You cannot observe your surroundings."), nil
		}
		result.Observation = view
		result.Message = message
		for _, e := range residents {
			if e.Kind == "actor_introduced" && e.Payload.Target == actorID {
				result.Message += "\nThe " + nextBakery.Entities[e.ActorID].Name + " introduces themselves to you."
			}
		}
		result.SelectedTarget = p.Observed.Target
	}

	return result, &Transition{State: State{Clock: &next, Bakery: nextBakery}, Events: events}
}
