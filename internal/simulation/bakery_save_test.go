// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"reflect"
	"testing"
)

// Use the item properties that shipped with the selected practice rules so a
// restore test can exercise the particular invalid field it installs.
func usePracticeVersion(b *Bakery, version int) {
	b.ContentVersion = version
	if version < bakingContentVersion {
		b.PrepCell = CellPosition{}
	}
	for id, e := range b.Entities {
		if (version < plantContentVersion && e.Kind == Planter) || (version < phoneContentVersion && e.Kind == Telephone) {
			delete(b.Entities, id)
			continue
		}
		if version < examineContentVersion {
			e.CloseDetail = ""
		}
		if version < coverContentVersion {
			e.ProvidesCover = false
		}
		if version < dropSoundContentVersion {
			e.DropSound = ""
		}
		if version < bakingContentVersion {
			e.BakingBouts, e.BakingKneads = 0, 0
		}
		b.Entities[id] = e
	}
}

func TestBakerySnapshotRoundTrip(t *testing.T) {
	for _, seed := range []int64{42, 43} {
		clock := NewTimeline(seed)
		original := makeBakery(t, "town", clock)
		data, err := original.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		restored, err := RestoreBakery(data, "town", clock)
		if err != nil || !reflect.DeepEqual(restored, original) {
			t.Fatalf("seed %d: %v", seed, err)
		}
		for _, tc := range []struct{ old, new string }{
			{`"version":1`, `"version":2`},
			{`"GeneratorVersion":1`, `"GeneratorVersion":99`},
			{`"ContentVersion":1`, `"ContentVersion":99`},
			{`"town/spoon":`, `"town/missing":`},
			{`"HolderID":"town/crate"`, `"HolderID":"town/missing"`},
			{`"version":1`, `"version":1,"version":1`},
		} {
			damaged := bytes.Replace(data, []byte(tc.old), []byte(tc.new), 1)
			if _, err := RestoreBakery(damaged, "town", clock); err == nil {
				t.Fatalf("accepted %s", tc.new)
			}
		}
		if _, err := RestoreBakery(data, "other", clock); err == nil {
			t.Fatal("accepted foreign world")
		}
		if _, err := RestoreBakery(data, "town", NewTimeline(seed)); err == nil {
			t.Fatal("accepted unused geography stream")
		}
	}
}

func TestRestoreInvalidRestState(t *testing.T) {
	for _, tc := range []struct {
		id      string
		version int
		fatigue int
		seated  bool
	}{
		{"town/baker", restContentVersion, -1, false},
		{"town/player", restContentVersion, fatigueLimit + 1, false},
		{"town/baker", restContentVersion, 1, true},
		{"town/spoon", restContentVersion, 0, true},
		{"town/baker", residentContentVersion, 1, false},
	} {
		clock := NewTimeline(42)
		b, err := generateNotebookBakery("town", clock)
		if err != nil {
			t.Fatal(err)
		}
		b.ContentVersion = tc.version
		e := b.Entities[tc.id]
		e.Fatigue, e.Seated = tc.fatigue, tc.seated
		b.Entities[tc.id] = e
		data, err := b.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := RestoreBakery(data, "town", clock); err == nil {
			t.Fatalf("accepted invalid saved state: %+v", tc)
		}
	}
}

func TestFreshPracticeSnapshotRestores(t *testing.T) {
	clock := NewTimeline(42)
	b, err := BakerySetup{Practice: true}.Generate("bakery", clock)
	if err != nil {
		t.Fatal(err)
	}
	data, err := b.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = RestoreBakery(data, "bakery", clock); err != nil {
		t.Fatalf("restore: %v; version=%d gen=%d", err, b.ContentVersion, b.GeneratorVersion)
	}
}
