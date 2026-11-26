// SPDX-License-Identifier: GPL-3.0-or-later
//go:build linux || darwin || windows

package main

import (
	"io"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/eiannone/keyboard"
)

func TestBrowseWalkAndRestoreDraft(t *testing.T) {
	scene := &simulation.Scene{}
	menu := func() ([]menuChoice, *simulation.Scene, error) {
		return []menuChoice{{"walk to the crate", "walk to the crate"}, {"rest", "rest"}}, scene, nil
	}
	for _, tc := range []struct {
		keys     []keyboard.Key
		want     string
		selected bool
	}{
		{[]keyboard.Key{keyboard.KeyArrowDown, keyboard.KeyArrowDown, keyboard.KeyArrowUp, keyboard.KeyEnter}, "walk to the crate", true},
		{[]keyboard.Key{keyboard.KeyArrowUp, keyboard.KeyArrowUp, keyboard.KeyEsc, keyboard.KeyEnter}, "look", false},
	} {
		keys := make(chan keyboard.KeyEvent, 16)
		for _, r := range "look" {
			keys <- keyboard.KeyEvent{Rune: r}
		}
		for _, k := range tc.keys {
			keys <- keyboard.KeyEvent{Key: k}
		}
		close(keys)
		line, selected, err := keyLine(keys, io.Discard, menu, false)
		if err != nil || line != tc.want || (selected == scene) != tc.selected {
			t.Fatal(line, selected, err)
		}
	}
}
