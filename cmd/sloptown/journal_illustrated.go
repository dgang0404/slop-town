// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	_ "embed"
	"html/template"
	"io"
	"slices"

	"github.com/dgang0404/slop-town/internal/simulation"
)

//go:embed journal.html.tmpl
var journalTemplate string

var illustratedJournal = template.Must(template.New("journal").Parse(journalTemplate))

func writeIllustratedJournal(out io.Writer, entries []simulation.JournalEntry) error {
	hasMaps := slices.ContainsFunc(entries, func(e simulation.JournalEntry) bool { return e.DiscoveryGrid != "" })
	return illustratedJournal.Execute(out, struct {
		Chapters []journalChapter
		Empty    string
		HasMaps  bool
	}{composeJournal(entries), emptyJournal, hasMaps})
}
