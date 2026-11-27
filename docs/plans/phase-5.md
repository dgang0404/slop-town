# Phase 5 — hidden progression, emotion and relationships

The player and residents develop through what they do and experience. The simulation owns changes to capability, temporary emotion and directional relationships. Player views can show effort, expression and behavior. The developer history records the causes. This phase builds on the saved routines, notebooks and actor-specific observations from Phase 4. [Master design, sections 9–10, 13, 46 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 4 acceptance](phase-4-results.md)

| Checkpoint | Behavior to verify |
| --- | --- |
| 1. Endurance through practice | A demanding walk followed by recovery contributes to a small lasting improvement. Player and resident movement use the same rule. |
| 2. Temporary emotion | A personally experienced event changes an actor's temporary emotional state. Its effects subside with time while the actor's baseline personality persists. |
| 3. Directional relationships | A concrete interaction can change how one participant regards the other. Test both directions and a resident who missed it. |
| 4. Demo controls | Browse available actions with Up and Down. Walk up to a visible person or object by name. [Walkthrough and checks](#checkpoint-4-demo-controls) |
| 5. Acceptance | Follow the consequences through ordinary play, an interrupted save, restart and replay. Compare character views, developer history and local and hosted results. |

## Planned mechanic — professions and learnable skills

Players and residents can learn baking, reading, writing, music and woodworking through meaningful practice. A profession describes the skills a person has already had reason to develop. In a starting town, baking is the baker’s strongest skill, and the baker starts as the town’s most accomplished baker. The librarian reads fluently and writes well. The farmer has developed strength through years of lifting and hauling. Strength is a physical attribute that can help with many jobs. Knowing when dough has risen enough is learned baking skill.

Anyone with ingredients and access to an oven can try baking. The librarian can bake bread badly at first, learn from a dense loaf and eventually surpass the baker. The baker can learn a tune and the farmer can take up woodworking without either having to change jobs. The same opportunities apply to the player. Practice develops the skill the action actually exercises. Reading a letter and composing a reply exercise different skills, even when both happen at the same desk.

Show proficiency through the work itself. A practiced baker can judge the rise by touch, and a musician learning a tune may lose the rhythm at a difficult passage. For a future check, give the baker, the librarian and the player the same recipe and equipment. Verify that all three can attempt it, prior experience affects the outcome, and meaningful practice carries into later attempts after a save and reload.

## Checkpoint 1 results

Checkpoint 1 introduced practice bakeries with both Phase 4 workers in content version 6. The seed-42 player walked west, south, east, north, south and north, then sat down. The first two walks each took the player to four distinct destinations and left them tired. On the third walk the player felt tired on step six. On the first walk that happened on step five. The baker and porter also earned practice during their ordinary work routines. [Run the walkthrough](../../README.md#try-the-endurance-walk) · [Simulation checks](../../internal/simulation/endurance_test.go)

For this prototype a recovery counts after at least five steps and four distinct destinations since the last rest. The simulation marks the qualifying rest in the developer event history. After two qualifying recoveries, the first step from sitting costs no fatigue. This bakery supports one endurance improvement. Pacing back and forth between two cells and repeatedly sitting down both earned zero additional practice. Existing saves retain their original rules.

Failing the SQLite receipt write on the second recovery left the actor state and history intact. Retrying awarded the recovery once. Replay from sequence 0 and from sequence 46, immediately before that rest, reached tick 14, sequence 49, with SHA-256 `c972f354662c8587789538378f74ad13052dbb799c741fd833667c7d362a05f7`. [Restart and retry test](../../internal/storage/endurance_test.go)

Repeat the focused checks with

```sh
go test -count=1 -v -run 'WalkingPractice|EnduranceAcrossRestart' ./internal/simulation ./internal/storage
```

On 26 November 2026 the full offline Linux check passed with Go 1.27.4, including both seed-42 smoke runs. The terminal, simulation and storage packages passed three race-detector runs. Hosted Linux verification passed at `b233190`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45853342478) [Offline verification script](../../scripts/check-phase-0.sh)

## Checkpoint 2 results

Checkpoint 2 introduced content version 7 for practice bakeries. Sitting down after reaching the existing fatigue threshold brings the acting character relief. The developer history marks the rest that caused it. The character feels relieved on that turn and feels a little relief on the following two turns. Someone who can see the character on the first turn can notice relaxed shoulders. Looking, retrying a command and reopening the save spend no time. Sitting again while rested spends a turn and lets the feeling fade. The baker and porter keep their original break preferences throughout. [Simulation checks](../../internal/simulation/emotion_test.go)

On seed 42, the porter rested in the doorway on tick 10 and the baker rested at the back on tick 12. A player waiting at the shopfront could see both expressions. A player who walked two steps south missed the baker’s expression. The workers experienced the same relief in both walks. The player could also feel relief after pacing until tired, even though that route earned no endurance practice. [Terminal walkthrough](../../README.md#try-the-endurance-walk)

A failed SQLite receipt write on tick 12 rolled back the baker’s reaction and the fading of the porter’s earlier feeling. Retrying recorded one trigger per break. Both feelings had passed by tick 15. Replay from sequences 0, 35 and 42 reproduced sequence 51 with SHA-256 `4987e8bb6acaef449feeeb317673a2d3aab4f887b4b2c978db8978fc382d6112`. A version 6 save from the previous executable reopened at its published tick-14 digest. Both executables continued that save to the same tick-18 digest. [SQLite restart and rollback test](../../internal/storage/emotion_test.go)

The full offline Linux check and three race-detector runs passed. Hosted Linux verification passed at `7b7d5e2`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45867590635) Repeat the focused checks with

```sh
go test -count=1 -v -run Relief ./internal/simulation ./internal/storage
```

## Checkpoint 3 results

Content version 8 adds directional familiarity to fresh practice bakeries. An actor can introduce themselves to someone they can see on the same or an adjacent square. The listener’s notebook records who spoke, when, and the source event. Recognition appears in the listener’s view on seeing that person again. Repeating an introduction keeps its original source. The reverse direction requires a separate introduction. [Terminal walkthrough](../../README.md#introduce-yourself) · [Simulation checks](../../internal/simulation/relationship_test.go)

A worker with a free turn at the shopfront can introduce themselves to a person sharing the square. Travel, fatigue and scheduled work take priority. Workers skip seated people and pairs who have already had an introduction in either direction. At most one introduction uses that worker’s turn.

| Seed-42 turn | Interaction | Who became familiar with whom |
| ---: | --- | --- |
| 2 | After walking south, the player introduced themselves to the porter. The bakery wall blocked the baker’s view. | The porter recognized the player. The baker missed it. |
| 3 | The porter introduced themselves to the baker at the shopfront. | The baker recognized the porter. The porter still had only the player’s introduction. |

Staying at the shopfront gave a different result. On turn 4 the player received separate introductions from both workers. Walking two steps south at the start still allowed the two workers to meet each other on turn 3. [Simulation checks](../../internal/simulation/relationship_test.go)

The SQLite test interrupted the receipt write on turn 3. The player’s earlier introduction survived and the attempted introduction to the baker rolled back with the rest of the turn. After restarting, retrying and repeating the entire fifteen-turn transcript, both listeners retained exactly one introduction, tied to events 6 and 12 respectively. Replay from sequences 0 and 8 reached tick 15, sequence 52, with SHA-256 `3af00083bdcb72e16a28e3dc3273bd488f203e39188b39e8066c6031875b546a`. A version 7 save from checkpoint 2 reproduced its published tick-15 digest and continued to the same tick-18 state in both executables. [Restart test](../../internal/storage/relationship_test.go)

Repeat the focused checks with

```sh
go test -count=1 -v -run Introduction ./internal/simulation ./internal/storage
```

The full offline Linux check passed with Go 1.27.4, including both seed-42 smoke runs. The terminal, simulation and storage packages passed three race-detector runs. Checkpoint 3 passed hosted verification in the same push as checkpoint 4. [Offline verification script](../../scripts/check-phase-0.sh)

## Checkpoint 4 demo controls

This checkpoint brings the master design’s named destinations and interruptible movement into the terminal bakery. [Master design](../design/Slop_Town_Master_Design_Revised_v7.docx)

**Browse actions.** At the interactive prompt, Up and Down cycle through actions the player can take in the current scene, including approaching a visible target. The prompt shows the selected command. Enter submits it once. Escape returns to the previous input. Browsing, cancelling and asking for actions take no game time. Use the same choices and validation as the typed prompt so a selection still refers to the person or object the player picked. Keep typed commands and plain line input for players using text-only terminals and for saved walkthroughs.

**Walk to a noun.** The player can type `walk to the porter` or `walk to the crate`. Resolve names against the player’s current view and use the existing numbered choices when several visible targets match. Stop where the player can interact with the selected target. If it is already within reach, say so without advancing time. Each step uses ordinary movement rules and advances the town. A closed door, a lost target or an encounter can interrupt the walk. Report what the player can observe and leave them at the last completed step.

**When directions help.** In a room or on the street, offer people, objects and landmarks by name. Prefer descriptions such as “the crate beside the bakery door” when that is enough to act. Keep compass directions where they communicate a meaningful choice, such as a forest trail leading west and a road to the library leading north. Both locations are examples for future town navigation.

| Acceptance walkthrough | Expected result |
| --- | --- |
| Browse from a blank prompt, cancel, then browse and select an action. | Both arrow keys change the selection. Cancelling preserves the replay hash. Enter executes exactly one selected command. |
| Walk to a visible crate, then open it. Repeat the approach while beside it. | The player arrives within reach and can open the crate. The second approach costs no turn. |
| Choose between two visible objects with the same name. During another walk, let the selected resident leave sight. | Only visible matches appear. The walk keeps the chosen identity and stops when the player loses the resident. |
| Compare a named walk with the same individual steps, including a stop partway through. | Time, worker routines and positions agree for the steps completed. Reopening the save retains the stopping position. |


### Verification

A Linux terminal session used Up, Down and Escape to browse and cancel. Its tick-zero replay hash stayed the same. Enter on “walk to the porter” committed one step. On a separate three-square walk to the crate, named travel and three individual northward steps followed by opening the crate both ended at tick 10, sequence 36, SHA-256 `d790f982bcc900b02f0408bf021660595ba8cfbf561a54a499e315987426d20c`. Failing the second step’s SQLite receipt write left the first step saved. Resuming the batch finished the two outstanding steps. [Terminal and restart test](../../cmd/sloptown/walk_test.go)

In a seed-42 test with the crate placed across the street, a worker at the shopfront introduced themselves on the second step. The walk paused there on tick 4. The player chose the crate again and arrived on tick 5. In the ordinary bakery a player who lost sight of the moving baker stopped on tick 6. [Simulation checks](../../internal/simulation/approach_test.go) · [Playable walkthrough](../../README.md#browse-actions-and-walk-to-a-destination)

Longer travel to known town destinations, such as a bakery down the road, follows the same design. Each leg advances the simulation and an en-route event can return control to the player. The current demo covers destinations visible on the bakery map. Elapsed time in this demo is measured in simulation turns.

The offline Linux suite and three race-detector runs passed. The Windows amd64 and macOS arm64 executables cross-built successfully. Interactive arrow input was exercised on Linux. The hosted Linux job passed for the push containing checkpoints 3 and 4 at `aac5b0b`. [Hosted Linux job](https://github.com/dgang0404/slop-town/actions/runs/45872493285/job/140091194974)

## Checkpoint 5 results

A seed-42 player introduced themselves to the porter, walked to the crate, and rested in sight of both workers. A failed save rolled back the second endurance improvement, the feeling of relief and both witnesses’ notebook entries together. After retry, the player walked six steps before feeling tired. The uninterrupted and restarted visits ended with matching views for all three characters and identical event histories. Three replays reached tick 23, sequence 80. [Phase 5 acceptance results](phase-5-results.md)
