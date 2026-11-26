// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"os/exec"
	"strings"
	"testing"
)

func TestFixtureV1(t *testing.T) {
	// Recorded with the original terminal runner before the package split.
	const want = "60261db115f851c52ad8e47ec6c6b787992eab8518bf7656591bd22b923e96b5"
	if got := FixtureHash(42, FixtureValue(42)); got != want {
		t.Fatalf("seed 42 hash = %s, want %s", got, want)
	}
}

func TestSimulationDependencies(t *testing.T) {
	// Inspect the production dependency graph, including simulation subpackages.
	const format = "{{if or (not .Standard) (eq .ImportPath \"database/sql\")}}{{.ImportPath}}{{end}}"
	out, err := exec.Command("go", "list", "-deps", "-f", format, "./...").CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	const core = "github.com/dgang0404/slop-town/internal/simulation"
	for _, pkg := range strings.Fields(string(out)) {
		if pkg != core && !strings.HasPrefix(pkg, core+"/") {
			t.Errorf("dependency %s is outside the Phase 0 simulation boundary", pkg)
		}
	}
}
