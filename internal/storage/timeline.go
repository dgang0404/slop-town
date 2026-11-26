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
	var empty simulation.Result
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
	clock, seq, err := loadClock(tx, scope, seed)
	if err != nil {
		return empty, err
	}
	prior, err := loadReceipt(tx, scope, actor, p.RequestID)
	if err != nil {
		return empty, err
	}
	result, step := simulation.Resolve(scope, clock, seq, actor, p, prior)
	if step == nil {
		return result, nil
	}
	if err := writeStep(tx, scope, actor, step, simulation.Receipt{Proposal: p, Result: result}); err != nil {
		return empty, fmt.Errorf("save step: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	return result, nil
}

func committedClock(tx *sql.Tx, scope simulation.Scope) (*simulation.Timeline, uint64, error) {
	var data []byte
	var seq uint64
	err := tx.QueryRow("SELECT clock,last_sequence FROM timeline_state WHERE world=? AND timeline=?", scope.WorldID, scope.TimelineID).Scan(&data, &seq)
	if err != nil {
		return nil, 0, fmt.Errorf("read committed timeline: %w", err)
	}
	clock, err := simulation.RestoreTimeline(data)
	return clock, seq, err
}

func loadClock(tx *sql.Tx, scope simulation.Scope, seed int64) (*simulation.Timeline, uint64, error) {
	clock, seq, err := committedClock(tx, scope)
	if errors.Is(err, sql.ErrNoRows) {
		return simulation.NewTimeline(seed), 0, nil
	}
	return clock, seq, err
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
	clock, err := step.Clock.Snapshot()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO timeline_state(world,timeline,clock,last_sequence) VALUES (?,?,?,?)
		ON CONFLICT(world,timeline) DO UPDATE SET clock=excluded.clock, last_sequence=excluded.last_sequence`,
		scope.WorldID, scope.TimelineID, clock, step.Events[len(step.Events)-1].Sequence)
	if err != nil {
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
