// SPDX-License-Identifier: GPL-3.0-or-later
// Command packcheck validates a local pack's metadata without activating it.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"unicode/utf8"
)

type manifest struct {
	Format       int
	ContentAPI   int
	ID           string
	Version      int
	Dependencies []string
}

var packID = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

func validate(data []byte) (manifest, error) {
	var m manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	open, err := dec.Token()
	if !utf8.Valid(data) || err != nil || open != json.Delim('{') {
		return m, fmt.Errorf("expected a UTF-8 JSON object")
	}
	// Removing each field as it is read enforces exact names and one value per key.
	fields := map[string]any{
		"manifest_version": &m.Format, "content_api": &m.ContentAPI,
		"id": &m.ID, "version": &m.Version, "dependencies": &m.Dependencies,
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return m, err
		}
		name, ok := key.(string)
		if !ok || fields[name] == nil {
			return m, fmt.Errorf("unknown or repeated manifest field %q", key)
		}
		if err := dec.Decode(fields[name]); err != nil {
			return m, fmt.Errorf("invalid %s: %w", name, err)
		}
		delete(fields, name)
	}
	if _, err := dec.Token(); err != nil {
		return m, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return m, fmt.Errorf("unexpected data after manifest")
	}
	switch {
	case len(fields) != 0:
		return m, fmt.Errorf("missing required manifest field")
	case m.Format != 1:
		return m, fmt.Errorf("unsupported manifest_version %d", m.Format)
	case m.ContentAPI != 1:
		return m, fmt.Errorf("unsupported content_api %d", m.ContentAPI)
	case !packID.MatchString(m.ID):
		return m, fmt.Errorf("invalid pack ID %q", m.ID)
	case m.Version < 1:
		return m, fmt.Errorf("version must be a positive integer")
	case m.Dependencies == nil:
		return m, fmt.Errorf("dependencies must be an array")
	}
	seen := map[string]bool{m.ID: true}
	for _, id := range m.Dependencies {
		if !packID.MatchString(id) || seen[id] {
			return m, fmt.Errorf("invalid, repeated or self dependency %q", id)
		}
		seen[id] = true
	}
	return m, nil
}

func check(dir string) (manifest, error) {
	var empty manifest
	root, err := os.OpenRoot(dir)
	if err != nil {
		return empty, err
	}
	defer root.Close()
	const name = "manifest.json"
	info, err := root.Lstat(name)
	if err != nil {
		return empty, err
	}
	if !info.Mode().IsRegular() {
		return empty, fmt.Errorf("%s must be a regular file", name)
	}
	f, err := root.Open(name)
	if err != nil {
		return empty, err
	}
	defer f.Close()
	const limit = 64 << 10
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return empty, err
	}
	if len(data) > limit {
		return empty, fmt.Errorf("%s exceeds 64 KiB", name)
	}
	return validate(data)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: packcheck PACK_DIRECTORY")
		os.Exit(2)
	}
	m, err := check(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("valid manifest: %s version %d (content API %d)\n", m.ID, m.Version, m.ContentAPI)
}
