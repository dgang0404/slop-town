// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dgang0404/slop-town/internal/simulation"
)

// The field order and tags define the version 1 fixture state hash. Clock
// snapshots use deterministic JSON, including sorted stream names.
type checkpoint struct {
	Version int `json:"version"`
	simulation.Scope
	Sequence       uint64          `json:"sequence"`
	EngineVersion  int             `json:"engine_version"`
	ContentVersion int             `json:"content_version"`
	Clock          json.RawMessage `json:"clock"`
}

type ReplayReport struct {
	simulation.Scope
	Sequence uint64 `json:"sequence"`
	Tick     int64  `json:"tick"`
	Hash     string `json:"sha256"`
}

func encodeState(scope simulation.Scope, seq uint64, clock *simulation.Timeline) ([]byte, ReplayReport, error) {
	snapshot, err := clock.Snapshot()
	if err != nil {
		return nil, ReplayReport{}, err
	}
	data, err := json.Marshal(checkpoint{Version: 1, Scope: scope, Sequence: seq, EngineVersion: 1, ContentVersion: 1, Clock: snapshot})
	if err != nil {
		return nil, ReplayReport{}, err
	}
	return data, ReplayReport{scope, seq, clock.Tick(), fmt.Sprintf("%x", sha256.Sum256(data))}, nil
}

// SaveCheckpoint bookmarks the current committed position. Existing checkpoints
// at the same position must agree byte-for-byte.
func SaveCheckpoint(path string, scope simulation.Scope) (ReplayReport, error) {
	var empty ReplayReport
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
	clock, seq, err := committedClock(tx, scope)
	if err != nil {
		return empty, err
	}
	data, report, err := encodeState(scope, seq, clock)
	if err != nil {
		return empty, err
	}
	_, err = tx.Exec("INSERT INTO timeline_checkpoint(world,timeline,seq,data,digest) VALUES(?,?,?,?,?) ON CONFLICT(world,timeline,seq) DO NOTHING", scope.WorldID, scope.TimelineID, seq, data, report.Hash)
	if err != nil {
		return empty, err
	}
	var saved []byte
	var digest string
	err = tx.QueryRow("SELECT data,digest FROM timeline_checkpoint WHERE world=? AND timeline=? AND seq=?", scope.WorldID, scope.TimelineID, seq).Scan(&saved, &digest)
	if err != nil {
		return empty, err
	}
	if !bytes.Equal(data, saved) || report.Hash != digest {
		return empty, fmt.Errorf("checkpoint differs at sequence %d", seq)
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	return report, nil
}

func restoreCheckpoint(tx *sql.Tx, scope simulation.Scope, seq uint64) (*simulation.Timeline, error) {
	var data []byte
	var digest string
	err := tx.QueryRow("SELECT data,digest FROM timeline_checkpoint WHERE world=? AND timeline=? AND seq=?", scope.WorldID, scope.TimelineID, seq).Scan(&data, &digest)
	if err != nil {
		return nil, fmt.Errorf("checkpoint sequence %d: %w", seq, err)
	}
	var cp checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}
	if cp.Version != 1 || cp.EngineVersion != 1 || cp.ContentVersion != 1 {
		return nil, fmt.Errorf("checkpoint sequence %d: incompatible versions (format %d, engine %d, content %d)", seq, cp.Version, cp.EngineVersion, cp.ContentVersion)
	}
	if cp.Scope != scope || cp.Sequence != seq || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
		return nil, fmt.Errorf("checkpoint identity or hash mismatch at sequence %d", seq)
	}
	return simulation.RestoreTimeline(cp.Clock)
}

// Replay verifies the suffix against one consistent read-only view of the save.
func Replay(path string, scope simulation.Scope, from uint64) (ReplayReport, error) {
	var empty ReplayReport
	db, err := openTimelineDB(path, true)
	if err != nil {
		return empty, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return empty, err
	}
	defer tx.Rollback()
	var schema int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&schema); err != nil {
		return empty, err
	}
	if schema != 3 {
		return empty, fmt.Errorf("replay requires save schema 3, found %d", schema)
	}
	clock, err := restoreCheckpoint(tx, scope, from)
	if err != nil {
		return empty, err
	}
	head, last, err := committedClock(tx, scope)
	if err != nil {
		return empty, err
	}
	if from > last {
		return empty, fmt.Errorf("checkpoint sequence %d exceeds head %d", from, last)
	}
	clock, err = replaySuffix(tx, scope, clock, from, last)
	if err != nil {
		return empty, err
	}
	_, got, err := encodeState(scope, last, clock)
	if err != nil {
		return empty, err
	}
	_, want, err := encodeState(scope, last, head)
	if err != nil {
		return empty, err
	}
	if got != want {
		return empty, fmt.Errorf("sequence %d: replayed state hash differs from committed head", last)
	}
	return got, nil
}

func replaySuffix(tx *sql.Tx, scope simulation.Scope, clock *simulation.Timeline, from, last uint64) (*simulation.Timeline, error) {
	rows, err := tx.Query("SELECT seq,event FROM timeline_event WHERE world=? AND timeline=? AND seq>? ORDER BY seq", scope.WorldID, scope.TimelineID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seq := from
	var pending []simulation.Event
	type requestKey struct{ actor, request string }
	resolved := make(map[requestKey]bool)
	for rows.Next() {
		var position uint64
		var data []byte
		if err := rows.Scan(&position, &data); err != nil {
			return nil, err
		}
		seq++
		if position != seq || seq > last {
			return nil, fmt.Errorf("sequence %d: unexpected stored position %d (head %d)", seq, position, last)
		}
		var event simulation.Event
		if err := json.Unmarshal(data, &event); err != nil {
			return nil, fmt.Errorf("sequence %d: %w", seq, err)
		}
		if event.Version != 1 || event.PayloadVersion != 1 {
			return nil, fmt.Errorf("sequence %d: unsupported event version %d or payload version %d", seq, event.Version, event.PayloadVersion)
		}
		if len(pending) == 0 {
			key := requestKey{event.ActorID, event.RequestID}
			if resolved[key] {
				return nil, fmt.Errorf("sequence %d: repeated committed request", seq)
			}
			resolved[key] = true
			receipt, err := loadReceipt(tx, scope, event.ActorID, event.RequestID)
			if err != nil {
				return nil, fmt.Errorf("sequence %d: read receipt: %w", seq, err)
			}
			if receipt == nil {
				return nil, fmt.Errorf("sequence %d: missing request receipt", seq)
			}
			result, step := simulation.Resolve(scope, clock, seq-1, event.ActorID, receipt.Proposal, nil)
			if step == nil {
				return nil, fmt.Errorf("sequence %d: proposal version %d rejected (%s)", seq, receipt.Proposal.Version, result.Code)
			}
			if result != receipt.Result {
				return nil, fmt.Errorf("sequence %d: recorded result differs from resolved result", seq)
			}
			clock, pending = step.Clock, step.Events
		}
		if event != pending[0] {
			return nil, fmt.Errorf("sequence %d: recorded event differs from resolved event", seq)
		}
		pending = pending[1:]
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if seq != last || len(pending) != 0 {
		return nil, fmt.Errorf("sequence %d: missing event", seq+1)
	}
	return clock, nil
}
