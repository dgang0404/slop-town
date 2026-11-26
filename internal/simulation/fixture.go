// SPDX-License-Identifier: GPL-3.0-or-later
// Package simulation owns the deterministic Phase 0 smoke fixture.
package simulation

import (
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
)

// FixtureValue returns the repeatable value for a smoke-test seed.
func FixtureValue(seed int64) int64 {
	rng := rand.New(rand.NewPCG(uint64(seed), 0))
	return rng.Int64()
}

// FixtureHash hashes the versioned, decimal encoding of a saved fixture.
func FixtureHash(seed, value int64) string {
	state := fmt.Sprintf("fixture-v1/%d/%d", seed, value)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(state)))
}
