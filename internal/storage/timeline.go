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
	turn, err := submit(path, scope, seed, actor, p, false, nil)
	return turn.Result, err
}

// SubmitBakery requires an initialized bakery. The boolean identifies an old
// receipt, so a client need not display its historical scene as a new action.
func SubmitBakery(path string, scope simulation.Scope, actor string, p simulation.Proposal) (simulation.Result, bool, error) {
	return SubmitBakeryInput(path, scope, actor, p, nil)
}

// SubmitBakeryInput commits the accepted terminal input with its authoritative receipt.
func SubmitBakeryInput(path string, scope simulation.Scope, actor string, p simulation.Proposal, input *simulation.CommandInput) (simulation.Result, bool, error) {
	turn, err := SubmitBakeryTurn(path, scope, actor, p, input)
	return turn.Result, turn.Repeated, err
}

// SubmitBakeryTurn also returns the actor's presentation details from the committed step.
func SubmitBakeryTurn(path string, scope simulation.Scope, actor string, p simulation.Proposal, input *simulation.CommandInput) (simulation.Turn, error) {
	return submit(path, scope, 0, actor, p, true, input)
}

func submit(path string, scope simulation.Scope, seed int64, actor string, p simulation.Proposal, requireBakery bool, input *simulation.CommandInput) (simulation.Turn, error) {
	var empty simulation.Turn
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
		return empty, fmt.Errorf("migrate save: %w", err)
	}
	state, err := committedState(tx, scope, 4)
	if !requireBakery && errors.Is(err, sql.ErrNoRows) {
		state.Clock = simulation.NewTimeline(seed)
		err = nil
	}
	if err != nil {
		return empty, err
	}
	if requireBakery && state.Bakery == nil {
		return empty, fmt.Errorf("timeline contains a different fixture")
	}
	prior, err := loadReceipt(tx, scope, actor, p.RequestID)
	if err != nil {
		return empty, err
	}
	result, step := state.Resolve(scope, state.sequence, actor, p, prior)
	if step == nil {
		return simulation.Turn{Result: result, Repeated: prior != nil && result.Status == "accepted"}, nil
	}
	if err := writeStep(tx, scope, actor, step, simulation.Receipt{Proposal: p, Result: result, Input: input}); err != nil {
		return empty, fmt.Errorf("save step: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	return simulation.Turn{Result: result, Scene: step.Scene}, nil
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
	return OpenBakerySceneWithSetup(path, scope, seed, actor, simulation.BakerySetup{})
}

// OpenBakerySceneWithSetup applies the selected content only to a new timeline.
func OpenBakerySceneWithSetup(path string, scope simulation.Scope, seed int64, actor string, setup simulation.BakerySetup) (simulation.Scene, error) {
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
		state.Bakery, err = setup.Generate(scope.WorldID, state.Clock)
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
	view, err := state.Scene(scope, actor)
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
	bakery, err := readBakery(path, scope)
	if err != nil {
		return nil, err
	}
	return bakery.AvailableActions(scope, actor)
}

// BakeryNotebook is a developer read of the selected actor's saved sightings.
func BakeryNotebook(path string, scope simulation.Scope, actor string) ([]simulation.NotebookEntry, error) {
	bakery, err := readBakery(path, scope)
	if err != nil {
		return nil, err
	}
	return bakery.Notebook(actor)
}

// BakeryAccounts reads the named character's sources from one committed snapshot.
func BakeryAccounts(path string, scope simulation.Scope, actor string) ([]simulation.HeardAccount, error) {
	bakery, err := readBakery(path, scope)
	if err != nil {
		return nil, err
	}
	return bakery.Accounts(actor)
}

// BakeryDialogueChoices projects one speaker's lines at the committed head.
func BakeryDialogueChoices(path string, scope simulation.Scope, actor, listener string) ([]simulation.DialogueChoice, error) {
	bakery, err := readBakery(path, scope)
	if err != nil {
		return nil, err
	}
	return bakery.DialogueChoices(scope, actor, listener)
}

// BakeryRestPreference exposes one saved tendency to the developer inspector.
func BakeryRestPreference(path string, scope simulation.Scope, actor string) (int, error) {
	bakery, err := readBakery(path, scope)
	if err != nil {
		return 0, err
	}
	if bakery == nil || bakery.Entities[actor].RestInPlacePercent == 0 {
		return 0, fmt.Errorf("no saved personality profile for %q", actor)
	}
	return bakery.Entities[actor].RestInPlacePercent, nil
}

func readBakery(path string, scope simulation.Scope) (*simulation.Bakery, error) {
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
	return state.Bakery, nil
}

// BakeryReceipt looks up one committed request when resuming an explicit batch.
func BakeryReceipt(path string, scope simulation.Scope, actor, request string) (*simulation.Receipt, error) {
	db, err := openTimelineDB(path, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	receipt, err := loadReceipt(db, scope, actor, request)
	if err != nil || receipt == nil {
		return receipt, err
	}
	if !receipt.AcceptedFor(scope, actor, request) {
		return nil, fmt.Errorf("stored receipt does not match the requested actor and command")
	}
	return receipt, nil
}

func loadReceipt(reader interface{ QueryRow(string, ...any) *sql.Row }, scope simulation.Scope, actor, request string) (*simulation.Receipt, error) {
	var data []byte
	err := reader.QueryRow("SELECT receipt FROM request_receipt WHERE world=? AND timeline=? AND actor=? AND request=?", scope.WorldID, scope.TimelineID, actor, request).Scan(&data)
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
