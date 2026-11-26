// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundledPack(t *testing.T) {
	m, err := check(filepath.Join("..", "..", "content", "smoke"))
	if err != nil || m.ID != "smoke" || m.Version != 1 {
		t.Fatalf("check returned %+v, %v", m, err)
	}
}

func TestManifestValidation(t *testing.T) {
	const valid = `{"manifest_version":1,"content_api":1,"id":"demo","version":2,"dependencies":["base"]}`
	if _, err := validate([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, from, to string }{
		{"format", `"manifest_version":1`, `"manifest_version":9`},
		{"incompatible", `"content_api":1`, `"content_api":2`},
		{"version", `"version":2`, `"version":0`},
		{"unknown key", `"id"`, `"url"`},
		{"repeat key", `"id"`, `"id":"other","id"`},
		{"path", `"demo"`, `"../demo"`},
		{"self dependency", `["base"]`, `["demo"]`},
		{"dependency path", `["base"]`, `["../base"]`},
		{"repeated dependency", `["base"]`, `["base","base"]`},
		{"missing field", `"version":2,`, ""},
		{"null array", `["base"]`, "null"},
		{"truncated", "}", ""},
		{"second object", "}", "}{}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := validate([]byte(strings.Replace(valid, tc.from, tc.to, 1))); err == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
}

func TestOversizeManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), bytes.Repeat([]byte{' '}, (64<<10)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := check(dir); err == nil || !strings.Contains(err.Error(), "64 KiB") {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestManifestSymlink(t *testing.T) {
	dir := t.TempDir()
	target, err := filepath.Abs(filepath.Join("..", "..", "content", "smoke", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "manifest.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := check(dir); err == nil {
		t.Fatal("accepted linked manifest")
	}
}
