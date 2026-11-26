// SPDX-License-Identifier: GPL-3.0-or-later
//go:build !linux && !darwin && !windows

package main

import "io"

func keyboardReader(io.Reader, io.Writer) (readMenuLine, func()) { return nil, nil }
