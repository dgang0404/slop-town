// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/dgang0404/slop-town/internal/simulation"
)

//go:embed journal.en.json
var journalEnglish []byte

var journalPhrases struct {
	Perceptions    map[simulation.PerceptionKind]string
	Discoveries    map[simulation.DiscoveryLocation]string
	DiscoveryIndex struct{ Title, Empty, NoMatch string }
}

func init() {
	if err := json.Unmarshal(journalEnglish, &journalPhrases); err != nil {
		panic(err)
	}
}

func rememberedObservation(sight simulation.JournalObservation) string {
	if phrase := journalPhrases.Perceptions[sight.Kind]; phrase != "" {
		return phrase
	}
	return remembered(sight.Text)
}

func rememberedDiscovery(sight simulation.Discovery) string {
	phrase := journalPhrases.Discoveries[sight.Location]
	if phrase == "" || (sight.Location == simulation.LocationUnspecified && sight.Text != "") {
		return remembered(sight.Text)
	}
	return strings.NewReplacer("{object}", sight.Label, "{bearing}", sight.Bearing, "{container}", sight.Container).
		Replace(phrase)
}
