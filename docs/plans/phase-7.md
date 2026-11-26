# Phase 7 — emergent narrative and deterministic dialogue

Section 17 of the master design calls for a consistent narrator, adjustable narration length, and a chronicle of what the player saw, heard, and did. Sections 38 and 47 ground that writing in the events and observations available to the character. Phase 7 is the roadmap home for this work. The first presentation checkpoint runs in the Phase 5 bakery. [Master design, revision 7](../design/Slop_Town_Master_Design_Revised_v7.docx)

## Checkpoint 1 — readable terminal turns

Open a session with the full scene so the player can get their bearings. After each command, give the action’s outcome, then describe what changed for the player. **Bold the new or changed detail** where the terminal supports it. Unchanged surroundings can drop out of the prose. Keep `look` available for the full description and grid at any time.

Suppose the player has already seen the crate. Opening it reveals what is inside.

> You open the crate. Inside lies **a wooden spoon**.

On the following quiet turn, “You wait.” is enough. If the porter approaches and speaks during that wait, give the introduction its own sentence. If the porter slips out of view, “You lose sight of **the porter**” says what the player knows.

The terminal should compare the previous and current player observations and include any interaction the player witnessed between them. Identify people and objects by their stable references so two wooden spoons with the same name remain distinct. During a named walk, combine uneventful steps into a short passage. An interruption ends the passage immediately and returns the prompt. Keep the grid available as a view of the current scene, with a fresh copy in the transcript when the player asks for it.

### Acceptance walkthrough

| In the bakery | What the reader should get |
| --- | --- |
| Open the crate, then wait nearby. | The opening and newly visible spoon appear once. A quiet wait gets a brief acknowledgement. |
| Walk toward a visible destination and receive an introduction en route. | The passage stops at the encounter. The speaker and what the player heard are clear before the next choice. |
| Lose sight of a resident, then use `look`. | The turn describes the loss of sight. The full scene shows what is visible now. The resident’s unseen destination stays private. |
| Browse actions, cancel, retry an acknowledged command, or reopen a save. | Browsing and cancelling spend no turns. A retry is acknowledged once. Reopening establishes the current scene as the starting point for later changes. |
| Read the same walkthrough in a terminal with bold disabled or redirect it to a file. | The sentences still convey every change. Plain output contains no terminal styling codes. Both runs produce the same simulation replay hash. |

### Checkpoint 1 results

On 26 November 2026, a seed-42 bakery session opened the door, walked east twice, opened the crate, waited, and used `look`. The spoon appeared in the opening sentence. The quiet wait took one line. The grid appeared twice, on arrival and on request. Reopening the save showed the spoon in the full scene. The following wait had no reason to mention it again. [Try the demo](../../README.md#look-inside-the-bakery)

The same commands ran with terminal bold, with `NO_COLOR=1`, with `TERM=dumb`, and with output redirected. All four transcripts had identical wording. Replay from sequence zero reached tick 5, sequence 10, with SHA-256 `2740f2ebd8f991dd568b3398c274d18da59cbfa525efa5c0e15070a152afca7e` in each save. Down and Enter also selected and opened the door in an interactive Linux terminal.

A three-step walk to the crate mentioned the approach once and arrived on the same tick as the three directional moves. A worker’s introduction on the second step of a separate walk paused the passage at tick 4. A player who lost sight of the baker learned only that the baker had passed out of view. A test with two spoons distinguished them by their visible locations. The offline build, test suite, both fixture smoke runs, and three race-detector runs passed. [Transcript tests](../../cmd/sloptown/narration_test.go) · [Interrupted walk and replay test](../../cmd/sloptown/walk_test.go)

## Checkpoint 2 — adjustable narration

The bakery starts with brief turn descriptions. Choose `narration story` at the prompt for fuller passages, or launch the demo with `go run ./cmd/sloptown bakery -play -narration story`. The arrow menu offers the other style. The choice applies to this session, including when playing from a SQLite save. [Play the bakery](../../README.md#look-inside-the-bakery)

| Seed-42 command | Brief | Story |
| --- | --- | --- |
| Step east from the bakery doorway | You move east. **You are in the bakery.** | You step east **into the bakery**. The closed crate is **beside you now**. |
| Open the crate | You open the crate. Inside lies **a wooden spoon**. | You open the crate and find **a wooden spoon** inside. |
| Take the spoon | You **take** the wooden spoon. | You **pick up** the wooden spoon you saw in the crate. |
| Wait when nothing else changes | You wait. | You wait. |

The story templates use the player’s current and previous observations. They can recall where the player saw an object, or describe two residents disappearing together from the same part of the street. A spoken introduction still interrupts a walk. Asking about the setting or changing it during a numbered choice preserves that choice.

### Checkpoint 2 results

A seed-42 visit opened the door, stepped east twice, opened the crate, took the spoon, put it back, waited, and used `look`. Brief, story and a mid-visit switch produced identical receipts for all seven actions. Retrying each visit from the beginning acknowledged those seven actions. Replay from sequence zero reached tick 7, sequence 14, with SHA-256 `600f08f071d98edc8191e6aaacd48ede82254884a5c491c723b6490efe985d17`.

In an interactive Linux terminal, selecting `narration story` with the arrows and Enter left the opening save at tick zero. The walkthrough then showed the new spoon in bold and reached the same replay hash as both plain-text runs. The full offline build and test suite, both seed-42 smoke runs, and three race-detector runs passed. [Walkthrough and interruption tests](../../cmd/sloptown/narration_test.go)

## Checkpoint 3 — the player journal

`journal` opens a first-person account of the selected bakery timeline. Each committed player turn can contribute an action, a change of place or feeling, or a description of a person or object the player could see. The description preserves where the observer saw the object at the time. The journal also records personally witnessed interactions and introductions addressed to the player, including on another actor’s turn. A run of waits can share a sentence. A named walk records its approach once and keeps its arrival or interruption. [Read your journal in the demo](../../README.md#read-your-journal)

The simulation assembles the account from the player’s accepted receipts, addressed introductions and firsthand notebook sightings. The terminal writes the prose. SQLite reads one committed snapshot when the journal opens. The same projection serves in-memory play and existing bakery saves. Looking, browsing, reading, cancelling and rejected commands add no entries.

### Checkpoint 3 results

A seed-42 save created by the checkpoint 2 executable at `3ac9947` yielded the original seven-action visit. The journal remembered seeing the wooden spoon in the crate, picking it up and putting it back. Reading twice and retrying the visit preserved the entries. The old and new executables replayed to tick 7, sequence 14, with SHA-256 `600f08f071d98edc8191e6aaacd48ede82254884a5c491c723b6490efe985d17`. A failed receipt write added no entry. Opening a separate timeline in the same file produced an empty journal.

In the practice bakery, the player who waited at the shopfront remembered both workers’ greetings. The player who walked down the street missed those greetings and the baker’s break. A separate visit verified that hearing the porter introduce themselves twice produced two memories, even though the player learned to recognize the porter only once. Up and Enter opened the journal in a Linux terminal without advancing the save. The offline build, test suite, both seed-42 smoke runs and three race-detector runs passed. [Journal walkthroughs](../../cmd/sloptown/journal_test.go)

## Checkpoint 4 — chapters and plain-text export

The bakery clock counts turns. This prototype uses significant firsthand events for chapter boundaries. The first recorded visit opens the book. The first observed turn inside the bakery and a first personal introduction can each start a chapter. A player can introduce themselves or hear the other person speak first. A later greeting between the same two people stays in the account within the current chapter. When several introductions happen on the same turn, they share a heading. A meeting on the first turn inside also gets a single heading.

The journal on screen, the in-game `export journal` command and the standalone `sloptown journal -db PATH` command use the same plain-text rendering. The in-game command creates a new file for each copy. The standalone command reads the selected timeline from an existing save and writes the story to standard output. Both use the player’s journal projection. [Play and export a visit](../../README.md#read-your-journal)

### Checkpoint 4 results

Four waits at the seed-42 shopfront produced “The visit begins” followed by “Meeting the baker and the porter.” The player who went down the street missed the greetings and kept one chapter. On another visit, the player introduced themselves to the porter twice. The journal recorded both greetings under one heading. Returning through the bakery doorway also preserved the earlier chapter breaks. [Chapter and export tests](../../cmd/sloptown/journal_export_test.go)

A save created with the previous executable at `42c51f3` produced byte-identical text through shell export, Up and Enter in a Linux terminal, and a second export after retrying the original visit. Both executables replayed to tick 4, sequence 16, with SHA-256 `dad6bae76d36397648b43387efa41ff743efd54a3b42cfcc05eaacc6c223fe97`. A save deliberately named `slop-town-journal.txt` kept its replay hash while two exports took the next available names. The full offline build and test suite, both seed-42 smoke runs and three race-detector runs passed.

The CLI also cross-built for Windows amd64 and macOS arm64. Interactive export was exercised on Linux.

## Checkpoint 5 — conversations in the practice bakery

Use `talk to NAME` within reach of a worker. The menu offers up to eleven distinct lines drawn from the speaker’s observations, firsthand memories and reports they heard from this listener. The player sees the words they will say. After each exchange, nearby residents take their usual turns. If the listener is still within reach, the menu refreshes. [Try the seed-42 conversation](../../README.md#talk-to-a-worker)

Players and residents can submit the same speech action with a listener, topic, intent and tone. A question about a spoon in someone else’s pocket gets an answer from the listener’s own view. A worker seeking a seat can say they need one. A report preserves the person who told it and which event they witnessed. Both participants can read their exchange in their journal.

### Checkpoint 5 results

Three waits brought the seed-42 baker to the shopfront. The player asked what the baker had seen and learned that the porter had introduced themselves to the baker. A new line let the player confirm what they had heard. The journal quoted the baker. A separate simulation test had the baker volunteer the news to the player. The porter was nearby and received no journal entry for that exchange. A scene stocked with extra visible objects offered eleven distinct lines. Hiding the selected object before submission invalidated its question without spending a turn. [Simulation checks](../../internal/simulation/dialogue_test.go)

An interrupted SQLite receipt write preserved the pre-conversation checkpoint. Reopening the save restored the follow-up question. Retrying both lines by number preserved the journal, and substituting a different number produced a request conflict. [Saved conversation walkthrough](../../cmd/sloptown/dialogue_test.go)

A save made by checkpoint 4 at `4a1d626` replayed to the same tick-three hash in both executables. Continuing that save with typed choices and making the same choices in a fresh Linux terminal using Down and Enter both reached tick 5, sequence 20, with SHA-256 `5e58b9675cdff165c41eef113171ed6e4e21c8317483782b28280899294ffc66`. The full offline build and tests, both smoke runs and three race-detector runs passed.

## Later narrative work

The next conversation checkpoint is for residents to start exchanges during their own routines. Two residents can meet while the player is elsewhere. A later conversation can bring news of that meeting back to the player. [Master design, section 14.9](../design/Slop_Town_Master_Design_Revised_v7.docx)

A world calendar can supply day and season boundaries for longer chronicles. Illustrated export follows the plain-text edition.
