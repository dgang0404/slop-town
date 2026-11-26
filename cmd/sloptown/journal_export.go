// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

// The in-game control also works for an in-memory visit. Reserve a new name
// on every export so two sessions can write in the same directory.
func exportJournal(entries []simulation.JournalEntry) (string, error) {
	for copyNumber := 1; ; copyNumber++ {
		name := "slop-town-journal.txt"
		if copyNumber > 1 {
			name = fmt.Sprintf("slop-town-journal-%d.txt", copyNumber)
		}
		file, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		writeErr := writeJournal(file, entries)
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return "", errors.Join(err, os.Remove(name))
		}
		return name, nil
	}
}

// Print only the player's journal, so a saved visit can be exported from a shell.
func runJournal(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("sloptown journal", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("db", "", "existing bakery save")
	timeline := flags.String("timeline", "main", "saved bakery timeline")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *path == "" || *timeline == "" {
		flags.Usage()
		return 2
	}
	scope := simulation.Scope{WorldID: "bakery", TimelineID: *timeline}
	entries, err := storage.BakeryJournal(*path, scope, "bakery/player")
	if err == nil {
		err = writeJournal(stdout, entries)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
