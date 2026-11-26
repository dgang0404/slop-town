// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"

	"github.com/mattn/go-isatty"
	"golang.org/x/sys/windows"
)

func supportsBold(file *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(file.Fd()), &mode) == nil && mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 || isatty.IsCygwinTerminal(file.Fd())
}
