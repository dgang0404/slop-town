// SPDX-License-Identifier: GPL-3.0-or-later
//go:build linux || darwin || windows

package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/eiannone/keyboard"
	"github.com/mattn/go-isatty"
)

// Redirected input/output keeps the line protocol used by scripts.
func keyboardReader(in io.Reader, out io.Writer) (readMenuLine, func()) {
	input, ok := in.(*os.File)
	output, okOut := out.(*os.File)
	if !ok || !okOut || !isatty.IsTerminal(input.Fd()) || !isatty.IsTerminal(output.Fd()) {
		return nil, nil
	}
	keys, err := keyboard.GetKeys(32)
	if err != nil {
		return nil, nil
	}
	return func(menu menuSource, repeat bool) (string, *simulation.Scene, error) {
		return keyLine(keys, out, menu, repeat)
	}, func() { keyboard.Close() }
}

func keyLine(keys <-chan keyboard.KeyEvent, out io.Writer, menu menuSource, repeat bool) (string, *simulation.Scene, error) {
	var draft []rune
	var choices []menuChoice
	var scene *simulation.Scene
	index := -1
	width := 0
	for {
		shown := string(draft)
		if index >= 0 {
			shown = choices[index].label
		}
		size := len([]rune(shown))
		fmt.Fprintf(out, "\r> %s%s\r> %s", shown, strings.Repeat(" ", max(0, width-size)), shown)
		width = size
		ev, ok := <-keys
		if !ok {
			return "", nil, io.EOF
		}
		if ev.Err != nil {
			return "", nil, ev.Err
		}
		switch ev.Key {
		case keyboard.KeyEnter, keyboard.KeyCtrlJ:
			fmt.Fprintln(out)
			if index >= 0 {
				return choices[index].value, scene, nil
			}
			return string(draft), nil, nil
		case keyboard.KeyCtrlC, keyboard.KeyCtrlD:
			fmt.Fprintln(out)
			return "", nil, io.EOF
		case keyboard.KeyArrowUp, keyboard.KeyArrowDown:
			if index < 0 {
				var err error
				choices, scene, err = menu()
				if err != nil {
					return "", nil, err
				}
				if len(choices) == 0 {
					continue
				}
				index = 0
				if repeat {
					// Offer numbers without disclosing the saved answer. The last
					// number extends the list as the player scrolls.
					choices = append(choices, menuChoice{"1", "1"})
					index = 1
				}
			} else if repeat && ev.Key == keyboard.KeyArrowDown && index == len(choices)-1 {
				n := strconv.Itoa(len(choices))
				choices = append(choices, menuChoice{n, n})
				index++
			} else if ev.Key == keyboard.KeyArrowDown {
				index = (index + 1) % len(choices)
			} else {
				index = (index + len(choices) - 1) % len(choices)
			}
		case keyboard.KeyEsc:
			index = -1
		case keyboard.KeyBackspace, keyboard.KeyBackspace2:
			index = -1
			if len(draft) > 0 {
				draft = draft[:len(draft)-1]
			}
		case keyboard.KeySpace:
			index = -1
			draft = append(draft, ' ')
		default:
			if unicode.IsPrint(ev.Rune) {
				index = -1
				draft = append(draft, ev.Rune)
			}
		}
	}
}
