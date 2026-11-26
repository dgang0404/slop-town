// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgang0404/slop-town/internal/simulation"
	"github.com/dgang0404/slop-town/internal/storage"
)

func TestDialogueAcrossVisits(t *testing.T) {
	scope := simulation.Scope{WorldID: "bakery", TimelineID: "main"}
	path := filepath.Join(t.TempDir(), "conversation.sqlite")
	play := func(input, batch string) string {
		t.Helper()
		var out, diag bytes.Buffer
		if code := runBakery([]string{"-practice", "-play", "-db", path, "-batch", batch}, strings.NewReader(input), &out, &diag); code != 0 {
			t.Fatalf("exit %d: %s", code, &diag)
		}
		return out.String()
	}
	play("quit\n", "arrival")
	if _, err := storage.SaveCheckpoint(path, scope); err != nil {
		t.Fatal(err)
	}
	const waits = "wait\nwait\nwait\n"
	const opening = "talk to the baker\n"
	const report = "The porter introduced themselves to me."
	const question = "So the porter introduced themselves to you?"
	first := play(waits+"actions\n"+opening+"99\ncancel\n", "arrival")
	if !strings.Contains(first, "talk to the baker") || !strings.Contains(first, "Enter a number") ||
		strings.Contains(first, report) || strings.Contains(first, question) {
		t.Fatal(first)
	}
	before, err := storage.SaveCheckpoint(path, scope)
	if err != nil || before.Tick != 3 {
		t.Fatal(before, err)
	}

	// Neither participant remembers a conversation interrupted before commit.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TRIGGER fail_speech BEFORE INSERT ON request_receipt BEGIN SELECT RAISE(FAIL,'interrupted conversation'); END"); err != nil {
		t.Fatal(err)
	}
	var diag bytes.Buffer
	if code := runBakery([]string{"-play", "-db", path, "-batch", "converse"}, strings.NewReader(opening+"3\n"), &bytes.Buffer{}, &diag); code == 0 || !strings.Contains(diag.String(), "interrupted conversation") {
		t.Fatal(code, &diag)
	}
	if _, err := db.Exec("DROP TRIGGER fail_speech"); err != nil {
		t.Fatal(err)
	}
	if head, err := storage.Replay(path, scope, 0); err != nil || head != before {
		t.Fatal(head, err)
	}

	const visit = opening + "3\n6\ncancel\njournal\n"
	read := play(visit, "converse")
	spoken, journal, ok := strings.Cut(read, "My journal\n")
	if !ok || !strings.Contains(spoken, "6. "+question) ||
		!strings.Contains(journal, "The baker replied, “"+report+"”") ||
		!strings.Contains(journal, "I said to the baker, “"+question+"”") ||
		strings.Contains(journal, "I saw the porter introduce") {
		t.Fatal(read)
	}
	want, err := storage.Replay(path, scope, 0)
	if err != nil || want.Tick != 5 {
		t.Fatal(want, err)
	}
	if head, err := storage.Replay(path, scope, before.Sequence); err != nil || head != want {
		t.Fatal(head, err)
	}
	retry := play(visit, "converse")
	_, again, _ := strings.Cut(retry, "My journal\n")
	if strings.Count(retry, "already completed") != 2 || again != journal {
		t.Fatal(retry)
	}
	wrong := play(opening+"1\n", "converse")
	if !strings.Contains(wrong, "different action") {
		t.Fatal(wrong)
	}
	if head, err := storage.Replay(path, scope, 0); err != nil || head != want {
		t.Fatal(head, err)
	}
	reopened := play(opening+"cancel\n", "return")
	if !strings.Contains(reopened, question) {
		t.Fatal(reopened)
	}

	// The original cancel is harmless when retry has already advanced to a
	// different kind of receipt. The following command still occupies its slot.
	play(visit+"wait\n", "converse")
	extended := play(visit+"wait\n", "converse")
	if strings.Contains(extended, "don't understand") || strings.Count(extended, "already completed") != 3 {
		t.Fatal(extended)
	}

}
