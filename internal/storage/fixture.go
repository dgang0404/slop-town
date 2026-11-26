// SPDX-License-Identifier: GPL-3.0-or-later
// Package storage persists smoke fixtures and simulation transitions in SQLite.
package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func migrate(tx *sql.Tx) error {
	var version int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	switch version {
	case 4:
		return nil
	case 0:
		// The original runner created smoke_fixture without setting a version.
		_, err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS smoke_fixture (seed INTEGER PRIMARY KEY, value INTEGER NOT NULL);
			CREATE TABLE fixture_metadata (
				id INTEGER PRIMARY KEY CHECK (id = 1),
				world_id TEXT NOT NULL,
				timeline_id TEXT NOT NULL
			);
			INSERT INTO fixture_metadata (id, world_id, timeline_id) VALUES (1, 'smoke', 'main');
			PRAGMA user_version = 1;`)
		if err != nil {
			return err
		}
		fallthrough
	case 1:
		_, err := tx.Exec(`
			CREATE TABLE timeline_state (
				world TEXT NOT NULL, timeline TEXT NOT NULL, clock BLOB NOT NULL,
				last_sequence INTEGER NOT NULL CHECK(last_sequence >= 0),
				PRIMARY KEY(world, timeline)
			);
			CREATE TABLE timeline_event (
				world TEXT NOT NULL, timeline TEXT NOT NULL, seq INTEGER NOT NULL CHECK(seq > 0),
				event BLOB NOT NULL, PRIMARY KEY(world, timeline, seq)
			);
			CREATE TABLE request_receipt (
				world TEXT NOT NULL, timeline TEXT NOT NULL, actor TEXT NOT NULL,
				request TEXT NOT NULL, receipt BLOB NOT NULL, PRIMARY KEY(world, timeline, actor, request)
			);
			PRAGMA user_version = 2;`)
		if err != nil {
			return err
		}
		fallthrough
	case 2:
		_, err := tx.Exec(`
			CREATE TABLE timeline_checkpoint (
				world TEXT NOT NULL, timeline TEXT NOT NULL, seq INTEGER NOT NULL CHECK(seq >= 0),
				data BLOB NOT NULL, digest TEXT NOT NULL, PRIMARY KEY(world,timeline,seq)
			);
			PRAGMA user_version = 3;`)
		if err != nil {
			return err
		}
		fallthrough
	case 3:
		_, err := tx.Exec(`ALTER TABLE timeline_state ADD COLUMN bakery BLOB;
			PRAGMA user_version = 4;`)
		return err
	default:
		return fmt.Errorf("unsupported save schema version %d", version)
	}
}

// SaveFixture migrates the database and inserts a fixture if its seed is new.
func SaveFixture(path string, seed, value int64) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := migrate(tx); err != nil {
		return fmt.Errorf("migrate save: %w", err)
	}
	if _, err = tx.Exec("INSERT INTO smoke_fixture(seed, value) VALUES (?, ?) ON CONFLICT(seed) DO NOTHING", seed, value); err != nil {
		return err
	}
	return tx.Commit()
}

// LoadFixture reads the saved value for seed using a new connection.
func LoadFixture(path string, seed int64) (int64, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var got int64
	if err := db.QueryRow("SELECT value FROM smoke_fixture WHERE seed = ?", seed).Scan(&got); err != nil {
		return 0, err
	}
	return got, nil
}
