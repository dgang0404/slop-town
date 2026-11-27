// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
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
	ActionWait         ActionKind = "wait"
	ActionListen       ActionKind = "listen"
	ActionRest         ActionKind = "rest"
	ActionHide         ActionKind = "hide"
	ActionEmerge       ActionKind = "emerge"
	ActionMove         ActionKind = "move"
	ActionOpen         ActionKind = "open"
	ActionKnock        ActionKind = "knock"
	ActionExamine      ActionKind = "examine"
	ActionWater        ActionKind = "water"
	ActionHarvest      ActionKind = "harvest"
	ActionClose        ActionKind = "close"
	ActionTake         ActionKind = "take"
	ActionDiscreetTake ActionKind = "discreet_take"
	ActionPut          ActionKind = "put"
	ActionDrop         ActionKind = "drop"
	ActionIntroduce    ActionKind = "introduce"
	ActionApproach     ActionKind = "approach"
	ActionSpeak        ActionKind = "speak"
)

type ActionArgs struct {
	Stream    string     `json:"stream,omitempty"`
	Direction Direction  `json:"direction,omitempty"`
	Target    string     `json:"target,omitempty"`
	Container string     `json:"container,omitempty"` // Destination of a put.
	Speech    SpeechLine `json:"speech,omitzero"`
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
	Version    int                       `json:"version"`
	ObserverID string                    `json:"observer_id"`
	Text       string                    `json:"text"`
	Grid       string                    `json:"grid,omitempty"`
	Targets    []ObservedTarget          `json:"targets,omitempty"`
	Heard      []PerceptionKind          `json:"heard,omitempty"`     // Sounds heard during this turn, in order.
	Plants     map[string]PlantCondition `json:"plants,omitempty"`    // Only planters visible to this observer.
	Landmarks  map[string]PerceptionKind `json:"landmarks,omitempty"` // Visible targets across a threshold.
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
	ContinueWalk   bool        `json:"continue_walk,omitempty"`
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
	Destination       CellPosition   `json:",omitzero"`
	Target            string         `json:",omitempty"`
	HolderID          string         `json:",omitempty"`
	ItemID            string         `json:",omitempty"` // The cutting produced when Target is harvested.
	PersonDetail      string         `json:",omitempty"` // Distinguishing detail actually seen during an examination.
	EndurancePractice bool           `json:",omitempty"`
	ReliefStarted     bool           `json:",omitempty"`
	Dialogue          Exchange       `json:",omitzero"`
	Sound             PerceptionKind `json:",omitempty"`
	StealthPractice   string         `json:",omitempty"` // First qualifying reach from the floor or a container.
}

type requestKey struct{ actorID, requestID string }

// CommandInput preserves the player's noun phrases and one-based answers for
// a reference proposal. The simulation replays the resolved proposal itself.
type CommandInput struct {
	Target          string `json:"target"`
	Container       string `json:"container,omitempty"`
	TargetChoice    int    `json:"target_choice,omitempty"`
	ContainerChoice int    `json:"container_choice,omitempty"`
	LineChoice      int    `json:"line_choice,omitempty"`
}

type Receipt struct {
	Proposal Proposal
	Result   Result
	Input    *CommandInput `json:"input,omitempty"`
}

// AcceptedFor checks that this receipt belongs to the requested bakery observer.
func (r *Receipt) AcceptedFor(scope Scope, actor, request string) bool {
	return r != nil && r.Proposal.Scope == scope && r.Proposal.RequestID == request &&
		r.Result.Status == "accepted" && r.Result.RequestID == request &&
		r.Result.Observation.Scope == scope && r.Result.Observation.ObserverID == actor
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
	return c.SubmitTurn(actorID, p).Result
}

// Turn carries the receipt result and the live actor view from the same commit.
// A retry supplies the original result with an empty scene.
type Turn struct {
	Result
	Scene    Scene
	Repeated bool
}

func (c *Coordinator) SubmitTurn(actorID string, p Proposal) Turn {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := requestKey{actorID, p.RequestID}
	result, step := (State{Clock: c.clock, Bakery: c.bakery}).Resolve(c.scope, uint64(len(c.events)), actorID, p, c.receipts[key])
	turn := Turn{Repeated: step == nil && result.Status == "accepted"}
	if step != nil {
		turn.Scene = step.Scene
		c.clock = step.Clock
		c.bakery = step.Bakery
		c.events = append(c.events, step.Events...)
		c.receipts[key] = &Receipt{Proposal: p, Result: result}
	}
	result.Observation.Targets = slices.Clone(result.Observation.Targets)
	result.Observation.Heard = slices.Clone(result.Observation.Heard)
	result.Observation.Plants = maps.Clone(result.Observation.Plants)
	result.Observation.Landmarks = maps.Clone(result.Observation.Landmarks)
	turn.Result = result
	if step != nil && step.Bakery != nil {
		turn.Scene.Observation = result.Observation
	}
	return turn
}

// State is the authoritative simulation state at one point in a timeline.
type State struct {
	Clock  *Timeline
	Bakery *Bakery
}

// Transition holds the staged state and events for a single commit.
type Transition struct {
	State
	Scene  Scene
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
	events := []Event{advanced, advanced.child(advanced.Sequence+1, actorID, engine, kind, payload)}
	result.Status = "accepted"
	var scene Scene
	if nextBakery == nil {
		result.Observation = Observation{Scope: scope, Version: contractVersion, ObserverID: actorID, Text: message}
	} else {
		action := &events[1]
		if err := nextBakery.applySpatial(action, primary, &next); err != nil {
			return reject("resolution_failed", "The action could not be completed."), nil
		}
		residents, heard, err := nextBakery.advanceResidents(&next, *action)
		if err != nil {
			return reject("resolution_failed", "The resident could not finish the turn."), nil
		}
		events = append(events, residents...)
		if nextBakery.ContentVersion >= plantContentVersion {
			events = nextBakery.advancePlants(events)
		}
		if nextBakery.ContentVersion >= emotionContentVersion {
			nextBakery.advanceRelief(events)
		}
		scene, err = (State{Clock: &next, Bakery: nextBakery}).Scene(scope, actorID)
		if err != nil {
			return reject("resolution_failed", "You cannot observe your surroundings."), nil
		}
		var soundLines []string
		for _, sound := range heard {
			if sound.listener == actorID {
				scene.Heard = append(scene.Heard, sound.kind)
				soundLines = append(soundLines, sound.kind.Description())
			}
			if nextBakery.ContentVersion >= soundAccountContentVersion {
				memory := sound.cause.child(events[len(events)-1].Sequence+1, sound.listener, "perception", "sound_heard", EventPayload{Sound: sound.kind})
				events = append(events, memory)
				nextBakery.appendSighting(sound.listener, NotebookEntry{Tick: memory.Tick, SourceSequence: memory.Sequence, Action: ActionListen, Sound: sound.kind})
			}
		}
		if len(soundLines) != 0 {
			if p.Action != ActionListen {
				soundLines = append([]string{message}, soundLines...)
			}
			message = strings.Join(soundLines, "\n")
		}
		result.Observation = scene.Observation
		scene.ActionTarget = primary.payload.Target
		// The notebook records visibility at the instant of each effect, even
		// if a later effect moves its subject out of the final scene.
		for _, entry := range nextBakery.Notebooks[actorID][len(bakery.Notebooks[actorID]):] {
			if entry.Action != ActionSpeak && entry.Action != ActionListen {
				scene.Witnessed = append(scene.Witnessed, nextBakery.witnessedAction(entry))
			}
		}
		result.Message = message
		interrupted := false
		for _, e := range residents {
			if e.Kind == "actor_introduced" && e.Payload.Target == actorID {
				interrupted = true
				result.Message += "\nThe " + nextBakery.Entities[e.ActorID].Name + " introduces themselves to you."
			}
		}
		result.SelectedTarget = p.Observed.Target
		if p.Action == ActionHarvest {
			result.SelectedTarget = primary.payload.ItemID
		}
		if p.Action == ActionApproach {
			ref := primary.payload.Target
			result.SelectedTarget = ref
			// An addressed introduction leaves the next decision to the walker.
			switch {
			case interrupted:
				result.Message += "\nYou pause your walk."
			case !scene.hasTarget(ref):
				result.Message += "\nYou lose sight of your destination."
			default:
				dest, _ := nextBakery.PositionOf(ref)
				was, _ := bakery.PositionOf(ref)
				switch {
				case inReach(nextBakery.Entities[actorID].Placement.Cell, dest):
					result.Message += "\n" + walkArrivalMessage(nextBakery.Entities[ref].Name)
				case dest != was:
					result.Message += "\nYour destination moved. You pause."
				default:
					result.ContinueWalk = true
				}
			}
		}
	}

	return result, &Transition{State: State{Clock: &next, Bakery: nextBakery}, Scene: scene, Events: events}
}
