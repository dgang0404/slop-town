// SPDX-License-Identifier: GPL-3.0-or-later
//go:build !windows

package main

import (
	"os"

	"github.com/mattn/go-isatty"
)

func supportsBold(file *os.File) bool {
	return isatty.IsTerminal(file.Fd())
}
