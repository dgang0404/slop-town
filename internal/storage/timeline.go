// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/dgang0404/slop-town/internal/simulation"
)

// Submit resolves one proposal against the latest committed state on disk.
// seed initializes a new timeline; existing timelines keep their saved seed.
func Submit(path string, scope simulation.Scope, seed int64, actor string, p simulation.Proposal) (simulation.Result, error) {
	result, _, err := submit(path, scope, seed, actor, p, false)
	return result, err
}

// SubmitBakery requires an initialized bakery. The boolean identifies an old
// receipt, so a client need not display its historical scene as a new action.
func SubmitBakery(path string, scope simulation.Scope, actor string, p simulation.Proposal) (simulation.Result, bool, error) {
	return submit(path, scope, 0, actor, p, true)
}

func submit(path string, scope simulation.Scope, seed int64, actor string, p simulation.Proposal, requireBakery bool) (simulation.Result, bool, error) {
	var empty simulation.Result
	db, err := openTimelineDB(path, false)
	if err != nil {
		return empty, false, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return empty, false, err
	}
	defer tx.Rollback()
	if err := migrate(tx); err != nil {
		return empty, false, fmt.Errorf("migrate save: %w", err)
	}
	state, err := committedState(tx, scope, 4)
	if !requireBakery && errors.Is(err, sql.ErrNoRows) {
		state.Clock = simulation.NewTimeline(seed)
		err = nil
	}
	if err != nil {
		return empty, false, err
	}
	if requireBakery && state.Bakery == nil {
		return empty, false, fmt.Errorf("timeline contains a different fixture")
	}
	prior, err := loadReceipt(tx, scope, actor, p.RequestID)
	if err != nil {
		return empty, false, err
	}
	result, step := state.Resolve(scope, state.sequence, actor, p, prior)
	if step == nil {
		return result, prior != nil && result.Status == "accepted", nil
	}
	if err := writeStep(tx, scope, actor, step, simulation.Receipt{Proposal: p, Result: result}); err != nil {
		return empty, false, fmt.Errorf("save step: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return empty, false, err
	}
	return result, false, nil
}

type timelineState struct {
	simulation.State
	sequence uint64
}

func committedState(tx *sql.Tx, scope simulation.Scope, schema int) (timelineState, error) {
	var state timelineState
	var clock, bakery []byte
	query := "SELECT clock,last_sequence,bakery FROM timeline_state WHERE world=? AND timeline=?"
	if schema == 3 {
		// Read historical Phase 1 saves without migrating them during replay.
		query = "SELECT clock,last_sequence,NULL FROM timeline_state WHERE world=? AND timeline=?"
	}
	err := tx.QueryRow(query, scope.WorldID, scope.TimelineID).Scan(&clock, &state.sequence, &bakery)
	if err != nil {
		return state, fmt.Errorf("read committed timeline: %w", err)
	}
	state.Clock, err = simulation.RestoreTimeline(clock)
	if err == nil && bakery != nil {
		state.Bakery, err = simulation.RestoreBakery(bakery, scope.WorldID, state.Clock)
	}
	return state, err
}

// OpenBakery initializes a fresh timeline once. Each later open restores its
// saved layout and geography stream before projecting the current actor view.
func OpenBakery(path string, scope simulation.Scope, seed int64, actor string) (simulation.Observation, error) {
	scene, err := OpenBakeryScene(path, scope, seed, actor)
	return scene.Observation, err
}

func OpenBakeryScene(path string, scope simulation.Scope, seed int64, actor string) (simulation.Scene, error) {
	var empty simulation.Scene
	if !scope.Valid() {
		return empty, fmt.Errorf("invalid world or timeline ID")
	}
	db, err := openTimelineDB(path, false)
	if err != nil {
		return empty, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	if err := migrate(tx); err != nil {
		return empty, err
	}
	state, err := committedState(tx, scope, 4)
	if errors.Is(err, sql.ErrNoRows) {
		state.Clock = simulation.NewTimeline(seed)
		state.Bakery, err = simulation.GenerateBakery(scope.WorldID, state.Clock)
		if err == nil {
			err = writeState(tx, scope, state)
		}
	}
	if err != nil {
		return empty, err
	}
	if state.Bakery == nil {
		return empty, fmt.Errorf("timeline contains a different fixture")
	}
	view, err := state.Bakery.Scene(scope, actor)
	if err != nil {
		return empty, err
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	return view, nil
}

// BakeryActions reads the current committed scene, including actions by other sessions.
func BakeryActions(path string, scope simulation.Scope, actor string) ([]simulation.ActionOption, error) {
	db, err := openTimelineDB(path, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	state, err := committedState(tx, scope, 4)
	if err != nil {
		return nil, err
	}
	return state.Bakery.AvailableActions(scope, actor)
}

func loadReceipt(tx *sql.Tx, scope simulation.Scope, actor, request string) (*simulation.Receipt, error) {
	var data []byte
	err := tx.QueryRow("SELECT receipt FROM request_receipt WHERE world=? AND timeline=? AND actor=? AND request=?", scope.WorldID, scope.TimelineID, actor, request).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var receipt simulation.Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func writeStep(tx *sql.Tx, scope simulation.Scope, actor string, step *simulation.Transition, receipt simulation.Receipt) error {
	if err := writeState(tx, scope, timelineState{step.State, step.Events[len(step.Events)-1].Sequence}); err != nil {
		return err
	}
	for _, event := range step.Events {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO timeline_event(world,timeline,seq,event) VALUES (?,?,?,?)", scope.WorldID, scope.TimelineID, event.Sequence, data); err != nil {
			return err
		}
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO request_receipt(world,timeline,actor,request,receipt) VALUES (?,?,?,?,?)", scope.WorldID, scope.TimelineID, actor, receipt.Proposal.RequestID, data)
	return err
}

func writeState(tx *sql.Tx, scope simulation.Scope, state timelineState) error {
	clock, err := state.Clock.Snapshot()
	if err != nil {
		return err
	}
	var bakery []byte
	if state.Bakery != nil {
		bakery, err = state.Bakery.Snapshot()
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(`INSERT INTO timeline_state(world,timeline,clock,last_sequence,bakery) VALUES (?,?,?,?,?)
		ON CONFLICT(world,timeline) DO UPDATE SET clock=excluded.clock, last_sequence=excluded.last_sequence, bakery=excluded.bakery`,
		scope.WorldID, scope.TimelineID, clock, state.sequence, bakery)
	return err
}

func openTimelineDB(path string, readOnly bool) (*sql.DB, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	name := filepath.ToSlash(abs)
	if !strings.HasPrefix(name, "/") {
		name = "/" + name // Windows drive letter in a file URI
	}
	// Acquire the writer lock before reading the clock or looking for a receipt.
	uri := url.URL{Scheme: "file", Path: name, RawQuery: "_txlock=immediate&_pragma=busy_timeout(10000)"}
	if readOnly {
		uri.RawQuery = "mode=ro&_pragma=busy_timeout(10000)"
	}
	return sql.Open("sqlite", uri.String())
}
