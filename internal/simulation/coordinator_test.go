// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"math"
	"reflect"
	"slices"
	"testing"
)

func fixtureCommand(t *testing.T, scope Scope) (*Coordinator, Proposal) {
	t.Helper()
	c, err := NewCoordinator(scope, 42)
	if err != nil {
		t.Fatal(err)
	}
	return c, Proposal{Scope: scope, Version: 1, RequestID: "r1", Action: "fixture_step", Args: FixtureArgs{Stream: "weather"}}
}

func TestCommandEventOrder(t *testing.T) {
	scope := Scope{"world/1", "main"}
	var first []Event
	for run := 0; run < 2; run++ {
		c, p := fixtureCommand(t, scope)
		for i, requestID := range []string{"r1", "r2"} {
			p.RequestID = requestID
			r := c.Submit("resident", p)
			if r.Status != "accepted" || r.Version != 1 || r.RequestID != requestID ||
				r.Observation.Scope != scope || r.Observation.ObserverID != "resident" || r.Observation.Text == "" {
				t.Fatalf("unexpected result: %+v", r)
			}
			events := c.History()
			if len(events) != 2*(i+1) {
				t.Fatalf("unexpected history: %v", events)
			}
			step, draw := events[2*i], events[2*i+1]
			for j, e := range []Event{step, draw} {
				if e.Scope != scope || e.Version != 1 || e.Engine != "fixture" || e.PayloadVersion != 1 ||
					e.Tick != int64(i+1) || e.Sequence != uint64(2*i+j+1) || e.ActorID != "resident" || e.RequestID != requestID {
					t.Fatalf("incorrect event envelope: %+v", e)
				}
			}
			wantDraw := []uint64{0x9b3e36a2b0e88c3e, 0xee873f3f5e94da17}[i]
			if step.Kind != "tick_advanced" || step.Payload.Delta != 1 || step.ParentID != "" ||
				draw.Kind != "stream_drawn" || draw.ParentID != step.ID || draw.Payload != (EventPayload{Stream: "weather", Value: wantDraw}) {
				t.Fatalf("incorrect causal pair: %+v, %+v", step, draw)
			}
		}
		history := c.History()
		ids := make(map[string]bool)
		for _, e := range history {
			if ids[e.ID] || e.ID == "" {
				t.Fatalf("invalid event ID %q", e.ID)
			}
			ids[e.ID] = true
		}
		if run == 0 {
			first = slices.Clone(history)
		} else if !slices.Equal(first, history) {
			t.Fatal("repeating the commands changed the event history")
		}
		history[0].Kind = "modified"
		if c.History()[0].Kind != "tick_advanced" {
			t.Fatal("history exposes mutable state")
		}
	}
}

func TestScopedCommandRetry(t *testing.T) {
	c, p := fixtureCommand(t, Scope{"a/b", "c"})
	first := c.Submit("player", p)
	if first.Status != "accepted" || c.Submit("resident", p).Status != "accepted" {
		t.Fatal("each actor should be able to use the same request ID")
	}
	history, state := c.History(), snapshot(t, c.clock)
	if got := c.Submit("player", p); !reflect.DeepEqual(got, first) {
		t.Fatalf("retry returned %+v, want %+v", got, first)
	}
	p.Args.Stream = "movement"
	if got := c.Submit("player", p); got.Code != "request_conflict" {
		t.Fatalf("conflicting retry: %+v", got)
	}
	if !slices.Equal(history, c.History()) || !bytes.Equal(state, snapshot(t, c.clock)) || len(c.receipts) != 2 {
		t.Fatal("retry changed committed state")
	}
	for _, scope := range []Scope{{"a/b", "other"}, {"a", "b/c"}} {
		other, q := fixtureCommand(t, scope)
		if other.Submit("player", q).Status != "accepted" || other.History()[0].ID == history[0].ID {
			t.Fatal("event or receipt identity collided across scopes")
		}
	}
}

func TestCommandRejections(t *testing.T) {
	c, p := fixtureCommand(t, Scope{"world", "main"})
	for _, tc := range []struct {
		field string
		code  string
	}{
		{"version", "unsupported_version"}, {"actor", "invalid_request"}, {"request", "invalid_request"},
		{"world", "invalid_request"}, {"timeline", "invalid_request"}, {"action", "invalid_request"}, {"stream", "invalid_request"},
		{"container", "invalid_request"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			q, actor := p, "player"
			switch tc.field {
			case "version":
				q.Version++
			case "actor":
				actor = ""
			case "request":
				q.RequestID = ""
			case "world":
				q.WorldID = "elsewhere"
			case "timeline":
				q.TimelineID = "other"
			case "action":
				q.Action = "unknown"
			case "stream":
				q.Args.Stream = ""
			case "container":
				q.Args.Container = "world/crate"
			}
			before := snapshot(t, c.clock)
			got := c.Submit(actor, q)
			if got.Status != "rejected" || got.Code != tc.code || got.Message == "" ||
				len(c.History()) != 0 || len(c.receipts) != 0 || !bytes.Equal(before, snapshot(t, c.clock)) {
				t.Fatalf("rejection leaked state or returned the wrong result: %+v", got)
			}
		})
	}
	// Draw fails after Advance succeeds on the staged copy.
	c.clock.draws["weather"] = math.MaxUint64
	before := snapshot(t, c.clock)
	if got := c.Submit("player", p); got.Code != "resolution_failed" ||
		!bytes.Equal(before, snapshot(t, c.clock)) || len(c.receipts) != 0 || len(c.History()) != 0 {
		t.Fatalf("partially applied a failed action: %+v", got)
	}
}

func TestConcurrentCommandRetry(t *testing.T) {
	c, p := fixtureCommand(t, Scope{"world", "main"})
	const count = 12
	start := make(chan struct{})
	results := make(chan Result, count)
	for range count {
		go func() {
			<-start
			results <- c.Submit("player", p)
		}()
	}
	close(start)
	first := <-results
	for i := 1; i < count; i++ {
		if got := <-results; !reflect.DeepEqual(got, first) {
			t.Fatalf("concurrent retry returned %+v, want %+v", got, first)
		}
	}
	if first.Status != "accepted" || len(c.History()) != 2 || c.clock.Tick() != 1 {
		t.Fatal("concurrent submissions did not commit exactly once")
	}
}
