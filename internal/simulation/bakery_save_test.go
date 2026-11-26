// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

import (
	"bytes"
	"reflect"
	"testing"
)

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
