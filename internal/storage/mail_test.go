// SPDX-License-Identifier: GPL-3.0-or-later
package storage_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestLettersSurviveInterruptedDeliveryAndRestart(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "post.sqlite")
	if _, err := storage.OpenBakerySceneWithSetup(path, scope, 42, "bakery/player", simulation.BakerySetup{Practice: true}); err != nil {
		t.Fatal(err)
	}
	play := newAcceptancePlayer(t, "mail")
	start := checkpointAtHead(t, path, scope)
	interrupt := func(script string) {
		t.Helper()
		before, records := bakeryBytes(t, path, scope), inspectScope(t, path, scope)
		execSQL(t, path, "CREATE TRIGGER interrupt_mail BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted post'); END")
		play(path, script, "interrupted post")
		if !bytes.Equal(before, bakeryBytes(t, path, scope)) || !reflect.DeepEqual(records, inspectScope(t, path, scope)) {
			t.Fatal("interrupted mail changed town or history")
		}
		execSQL(t, path, "DROP TRIGGER interrupt_mail")
	}
	const met = "wait\nwait\nintroduce yourself to porter\ntalk to porter\n"
	const route = met + "7\n8\ncancel\nsouth\nsouth\n"
	const post = route + "write to porter: Did your shift go well?\n"
	const answer = "Your letter reached me. Thank you for writing."
	play(path, met, "")
	interrupt(met + "7\n")
	out := play(path, route+"journal people porter\nactions\n", "")
	if !strings.Contains(out, "Box 3, Town Post. Shared on day 1") || !strings.Contains(out, "write to the porter:") {
		t.Fatal(out)
	}
	interrupt(post)
	play(path, post, "")
	stamped := checkpointAtHead(t, path, scope)
	if out := play(path, route+"write to porter: Different text.\n", ""); !strings.Contains(out, "different action") || checkpointAtHead(t, path, scope) != stamped {
		t.Fatal("a changed note reused the original receipt", out)
	}
	beforeDelivery := post + strings.Repeat("wait\n", 2)
	play(path, beforeDelivery, "")
	transit := checkpointAtHead(t, path, scope)
	interrupt(beforeDelivery + "wait\n")
	// Reopening from the save delivers the original envelope once.
	throughReply := post + strings.Repeat("wait\n", 10)
	out = play(path, throughReply+"look\njournal\n", "")
	if strings.Contains(out, answer) || !strings.Contains(out, "You have an unread letter.") || !strings.Contains(out, "A letter arrived for me.") {
		t.Fatal("sealed reply or delivery notice", out)
	}
	unopened := checkpointAtHead(t, path, scope)
	const read = "read letter\n"
	interrupt(throughReply + read)
	if out = play(path, throughReply+read+"journal\n", ""); !strings.Contains(out, "You read a letter from the porter. “"+answer+"”") || !strings.Contains(out, "I read a letter from the porter.") {
		t.Fatal(out)
	}
	head := checkpointAtHead(t, path, scope)
	out = play(path, throughReply+read+read, "")
	if !strings.Contains(out, "You have no unread letters.") || checkpointAtHead(t, path, scope) != head {
		t.Fatal(out)
	}
	town := inspectScope(t, path, scope)
	var posts, deliveries, openings []simulation.Event
	for _, e := range town.events {
		switch e.Kind {
		case "letter_posted":
			posts = append(posts, e)
		case "letter_delivered":
			deliveries = append(deliveries, e)
		case "letter_read":
			openings = append(openings, e)
		}
	}
	if len(posts) != 2 || len(deliveries) != 2 || len(openings) != 2 {
		t.Fatal(posts, deliveries, openings)
	}
	for i, p := range posts {
		d, r := deliveries[i], openings[i]
		if d.ParentID != p.ID || d.Tick != p.Tick+3 || d.ActorID != p.Payload.Target || r.ActorID != d.ActorID || r.Tick <= d.Tick || r.Payload.LetterSequence != p.Sequence {
			t.Fatal(p, d, r)
		}
	}
	if posts[1].Payload.LetterSequence != posts[0].Sequence {
		t.Fatal("reply lost its original envelope", posts)
	}
	for _, cp := range []storage.ReplayReport{start, stamped, transit, unopened} {
		if replay, err := storage.Replay(path, scope, cp.Sequence); err != nil || replay != head {
			t.Fatal(replay, err)
		}
	}
	t.Logf("posted %d, delivered %d, opened %d; reply posted %d, delivered %d, opened %d. Replay from %d, %d, %d, %d: tick %d, sequence %d, hash %s", posts[0].Tick, deliveries[0].Tick, openings[0].Tick, posts[1].Tick, deliveries[1].Tick, openings[1].Tick, start.Sequence, stamped.Sequence, transit.Sequence, unopened.Sequence, head.Tick, head.Sequence, head.Hash)
}
