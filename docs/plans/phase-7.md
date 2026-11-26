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

## Later narrative work

Use deterministic prose templates to give connected actions a consistent voice offline. A player who prefers brief narration can get a sentence, while a player who wants more description can read a fuller passage grounded in the same facts. The chronicle can group witnessed events into chapters by day, season, or a significant occurrence. Dialogue and journal entries draw on each participant’s knowledge, so a missed meeting can become a later discovery.
