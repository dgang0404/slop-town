// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"math"
	"testing"
)

func snapshot(t *testing.T, timeline *Timeline) []byte {
	t.Helper()
	data, err := timeline.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func draw(t *testing.T, timeline *Timeline, stream string) uint64 {
	t.Helper()
	value, err := timeline.Draw(stream)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestTimelineAdvance(t *testing.T) {
	timeline := NewTimeline(42)
	if timeline.Tick() != 0 {
		t.Fatal("timeline must start at tick zero")
	}
	for _, delta := range []int64{1, 7, math.MaxInt64 - 8} {
		want := timeline.Tick() + delta
		if err := timeline.Advance(delta); err != nil || timeline.Tick() != want {
			t.Fatalf("advance %d: tick %d, want %d, err %v", delta, timeline.Tick(), want, err)
		}
	}
	draw(t, timeline, "movement")
	before := snapshot(t, timeline)
	for _, delta := range []int64{0, -1, 1} {
		if err := timeline.Advance(delta); err == nil || !bytes.Equal(before, snapshot(t, timeline)) {
			t.Fatalf("advance %d should fail without changing the timeline", delta)
		}
	}
}

func TestIndependentStreams(t *testing.T) {
	a, b := NewTimeline(42), NewTimeline(42)
	for _, want := range []uint64{0x9b3e36a2b0e88c3e, 0xee873f3f5e94da17, 0xc06b1f68506096bd} {
		draw(t, b, "movement")
		if draw(t, a, "weather") != want || draw(t, b, "weather") != want {
			t.Fatal("weather diverged from the version 1 sequence")
		}
	}
	if draw(t, a, "movement") != 0xc952c2ba4351a4a3 {
		t.Fatal("movement did not start at its first draw")
	}
	if draw(t, NewTimeline(-1), "movement") == 0xc952c2ba4351a4a3 {
		t.Fatal("changing the seed did not change the result")
	}
}

func TestResumeTimeline(t *testing.T) {
	live := NewTimeline(-17)
	draw(t, live, "movement")
	draw(t, live, "weather")
	if err := live.Advance(1); err != nil {
		t.Fatal(err)
	}
	saved := snapshot(t, live)
	resumed, err := RestoreTimeline(saved)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"weather", "movement", "new-stream", "weather"} {
		if got, want := draw(t, resumed, name), draw(t, live, name); got != want {
			t.Fatalf("restored %q draw %x, want %x", name, got, want)
		}
	}
	for _, timeline := range []*Timeline{live, resumed} {
		if err := timeline.Advance(1); err != nil || timeline.Tick() != 2 {
			t.Fatalf("expected tick 2, got %d, err %v", timeline.Tick(), err)
		}
	}
	if !bytes.Equal(snapshot(t, live), snapshot(t, resumed)) {
		t.Fatal("resumed state diverged")
	}
}

func TestInvalidClockSnapshots(t *testing.T) {
	for _, data := range []string{
		`{`,
		`{"version":2,"tick":0,"seed":42,"draws":{}}`,
		`{"version":1,"tick":-1,"seed":42,"draws":{}}`,
		`{"version":1,"tick":0,"draws":{}}`,
		`{"version":1,"seed":42,"draws":{}}`,
		`{"version":1,"tick":0,"seed":42}`,
		`{"version":1,"tick":0,"seed":42,"draws":{"weather":0}}`,
	} {
		if _, err := RestoreTimeline([]byte(data)); err == nil {
			t.Errorf("accepted invalid snapshot %s", data)
		}
	}
}

func TestRejectedDrawPreservesState(t *testing.T) {
	timeline, err := RestoreTimeline([]byte(`{"version":1,"tick":3,"seed":7,"draws":{"full":18446744073709551615}}`))
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, timeline)
	for _, name := range []string{"full", "", "\xff"} {
		if _, err := timeline.Draw(name); err == nil || !bytes.Equal(before, snapshot(t, timeline)) {
			t.Fatalf("draw from %q should fail without changing state", name)
		}
	}
}
