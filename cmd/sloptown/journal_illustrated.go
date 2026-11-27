// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"embed"
	"encoding/base64"
	"html/template"
	"io"

	"github.com/dgang0404/slop-town/internal/simulation"
)

//go:embed journal.html.tmpl illustrations/*.webp
var journalFiles embed.FS

// Each portrait belongs to one observed object in the starter bakery.
// Other objects can share its name. Only bundled bytes become a data URL.
func journalSketch(target simulation.ObservedTarget) (template.URL, error) {
	var file string
	switch target {
	case simulation.ObservedTarget{Ref: "bakery/door", Name: "bakery door"}:
		file = "door.webp"
	case simulation.ObservedTarget{Ref: "bakery/crate", Name: "crate"}:
		file = "crate.webp"
	case simulation.ObservedTarget{Ref: "bakery/spoon", Name: "wooden spoon"}:
		file = "spoon.webp"
	default:
		return "", nil
	}
	image, err := journalFiles.ReadFile("illustrations/" + file)
	if err != nil {
		return "", err
	}
	return template.URL("data:image/webp;base64," + base64.StdEncoding.EncodeToString(image)), nil
}

var illustratedJournal = template.Must(template.New("journal.html.tmpl").Funcs(template.FuncMap{
	"sketch": journalSketch,
}).ParseFS(journalFiles, "journal.html.tmpl"))

func writeIllustratedJournal(out io.Writer, entries []simulation.JournalEntry) error {
	return illustratedJournal.Execute(out, struct {
		Chapters []journalChapter
		Empty    string
	}{composeJournal(entries), emptyJournal})
}
