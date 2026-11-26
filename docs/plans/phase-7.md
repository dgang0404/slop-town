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

## Later narrative work

Build a player journal from what the character saw, heard and did. The chronicle can group witnessed events into chapters by day, season, or a significant occurrence. Dialogue and journal entries draw on each participant’s knowledge, so a missed meeting can become a later discovery.
