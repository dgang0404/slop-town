// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"unicode/utf8"
)

// Timeline holds the coordinator's clock and deterministic random streams.
// Its single owner advances it in simulation order.
type Timeline struct {
	seed  int64
	tick  int64
	draws map[string]uint64
}

func NewTimeline(seed int64) *Timeline {
	return &Timeline{seed: seed, draws: make(map[string]uint64)}
}

func (t *Timeline) Tick() int64 { return t.tick }

func (t *Timeline) Advance(delta int64) error {
	if delta <= 0 || delta > math.MaxInt64-t.tick {
		return fmt.Errorf("invalid tick advance %d from %d", delta, t.tick)
	}
	t.tick += delta
	return nil
}

// Draw computes a sample from the seed, the stream's zero-based draw index,
// and its name. Only this stream's counter changes.
func (t *Timeline) Draw(name string) (uint64, error) {
	if name == "" || !utf8.ValidString(name) {
		return 0, fmt.Errorf("stream name must be nonempty UTF-8")
	}
	index := t.draws[name]
	if index == math.MaxUint64 {
		return 0, fmt.Errorf("stream %q exhausted", name)
	}
	input := binary.BigEndian.AppendUint64(nil, uint64(t.seed))
	input = binary.BigEndian.AppendUint64(input, index)
	sum := sha256.Sum256(append(input, name...))
	if t.draws == nil {
		t.draws = make(map[string]uint64)
	}
	t.draws[name] = index + 1
	return binary.BigEndian.Uint64(sum[:8]), nil
}

// Pointers distinguish missing clock and seed fields from legitimate zeroes.
type clockSnapshot struct {
	Version int               `json:"version"`
	Seed    *int64            `json:"seed"`
	Tick    *int64            `json:"tick"`
	Draws   map[string]uint64 `json:"draws"`
}

func (t *Timeline) Snapshot() ([]byte, error) {
	draws := t.draws
	if draws == nil {
		draws = map[string]uint64{}
	}
	return json.Marshal(clockSnapshot{Version: 1, Seed: &t.seed, Tick: &t.tick, Draws: draws})
}

func RestoreTimeline(data []byte) (*Timeline, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("clock snapshot must be UTF-8")
	}
	var s clockSnapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Version != 1 {
		return nil, fmt.Errorf("unsupported clock snapshot version %d", s.Version)
	}
	if s.Seed == nil || s.Tick == nil || *s.Tick < 0 || s.Draws == nil {
		return nil, fmt.Errorf("invalid clock snapshot")
	}
	for name, count := range s.Draws {
		if name == "" || count == 0 {
			return nil, fmt.Errorf("invalid stream entry %q", name)
		}
	}
	return &Timeline{seed: *s.Seed, tick: *s.Tick, draws: s.Draws}, nil
}
