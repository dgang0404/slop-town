# Phase 3 — player commands and contextual actions

The bakery already resolves typed proposals and persists their outcomes. Phase 3 lets a player express the same intentions in ordinary command phrases, discover what they can do, and clarify which visible object they mean. The terminal interprets input. The simulation binds the actor, checks the current scene and resolves the action. The fixture runs offline. [Master design, sections 12.4, 14.7, 33, 47 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 2 acceptance](phase-2-results.md)

## Walkthrough

On seed 42, type `open the bakery door`, walk east twice, and open the crate. Type `pick up the wooden spoon`. The spoon enters inventory at tick 5. Close the crate, walk west twice, close the bakery door and drop the spoon. The save should match the Phase 2 reference at tick 10. Asking for help, looking around, or entering an unrecognized verb between actions should leave the clock, random streams and event history intact.

## Checkpoints

| Checkpoint | Deliverable | Acceptance |
| --- | --- | --- |
| 1. One line per command | Add a terminal play session for the existing bakery verbs, cardinal directions, optional articles, common aliases, and `put ITEM in CONTAINER`. Support `look`, `help` and `quit`. | The walkthrough reaches the reference digest. A bad command or a blocked action gives feedback and the player can keep typing. A hidden and a nonexistent noun give the same response. |
| 2. Discover available actions | Answer “what can I do here?” from the acting character's current observations and the simulation's interaction rules. | A closed door offers opening. Opening the crate makes the spoon eligible for taking. Returning outside removes the crate from the suggestions. Asking costs no time. |
| 3. Clarify targets | Resolve short noun phrases, ambiguous matches and `it` against visible choices and the actor's recent selections. | Two visible objects with the same noun prompt a choice. An ambiguous request spends no time. Revalidate the chosen reference at submission, including when the scene has changed. |
| 4. Replay and handoff | Verify parsed and explicit commands across process restarts, including a clarification and a repeated request. | Equivalent inputs produce the same authoritative outcomes and observations. Record the local evidence and the hosted Linux run before the Phase 4 handoff. |

The bakery supplies movement, door and container actions for this phase. Resident routines and conversations have later roadmap milestones. Command history and undo have wider implications for saved timelines. Review those mechanics against the branching rules in section 44 of the master design before adding them to the terminal.

## Checkpoint 1 results

On 26 November 2026, the terminal played the seed-42 walkthrough one line at a time. `look`, `help`, an unknown verb, a blocked move and two unavailable targets left the initial checkpoint at tick 0 and sequence 0. Across sessions using the same SQLite save, five equivalent pickup commands under the same batch ID returned their original receipts. After the player dropped the spoon on the street, replay from sequence 0 reached tick 10 and sequence 20 with the Phase 2 digest `9f421909302942c47ba2ac7c46c627591ef95360749cff544ce23d206bafe58b`.

The full offline Linux verification script and three race-detector runs of the terminal package passed locally. Use `go test -v -count=1 -run '^(TestParseBakeryAction|TestBakeryPlaySession)$' ./cmd/sloptown` to repeat the focused checks. [Play the bakery](../../README.md#look-inside-the-bakery)

## Checkpoint 2 results

On 26 November 2026, `What can I do here?` and `actions` listed commands using the simulation's movement, visibility, reach and inventory rules. In the seed-42 doorway the player could see the crate, but had to step closer to open it. Opening the lid offered `take the wooden spoon`. Picking it up offered `drop the wooden spoon` and `put the wooden spoon in the crate`. Walking back to the street removed the crate from the list.

The in-memory and SQLite sessions produced the same menus. Repeated queries on the street left the save at tick 0 and event sequence 0. When a second actor took the spoon, the player's next query omitted it and the simulation rejected the previously offered command. A SQLite save created by checkpoint 1 reopened with the correct choices at tick 5. Replay from sequence 0 matched the reference digests at ticks 5 and 10.

The offline Linux verification script passed, as did three race-detector runs of the simulation and terminal packages. Run the focused checks with

```sh
go test -v -count=1 -run '^(TestBakeryActionDiscovery|TestBakeryActionsAreActorScoped|TestBakeryPlaySession)$' ./cmd/sloptown ./internal/simulation
```

## Checkpoint 3 results

On 26 November 2026 the terminal accepted short names such as `open door` and `take spoon`. A complete visible name takes priority over a shorter match. After a successful interaction, `it` refers to the primary object for the rest of the session. For example, after `put it in the crate`, `take it` picks the same spoon back up.

A SQLite test scene placed two wooden spoons on the street and kept a third behind the bakery door. The prompt listed the two visible spoons by location. Asking, entering an invalid number, cancelling and leaving the prompt kept the tick-zero replay hash. The player could choose both a spoon and a destination for `put spoon in box`. When the baker took the selected spoon before the player answered, the simulation rejected the stale choice even though the other spoon was still within reach. Swapping two spoons also left `it` pointing to the original object.

The seed-42 walkthrough reached the Phase 2 tick-10 digest using short names. The current runner reopened a save written by checkpoint 2, reused five receipts, and replayed a new `put it in crate` at tick 6. The offline Linux checks and three race-detector runs of the simulation and terminal packages passed. Repeat the focused checks with

```sh
go test -v -count=1 -run '^(TestMatchingTargets|TestNamedActionKeepsObservedReference|TestBakeryClarification|TestBakeryPlaySession)$' ./internal/simulation ./cmd/sloptown
```

## Checkpoint 4 results

On 26 November 2026 the local Linux acceptance test reproduced the same events and observations with parsed commands and explicit commands in separate SQLite saves. After restart, the player repeated five completed actions, including choosing a spoon and a destination crate. Appending `open it` advanced the save exactly once. The final replay reached tick 10 and sequence 20 in both saves. [Acceptance results and hosted verification status](phase-3-results.md)

The Phase 4 handoff awaits the hosted Linux run for checkpoint 4 after push. The next roadmap milestone introduces autonomous residents, needs, schedules, notebooks and personality. The baker in the Phase 3 tests acts at times selected by the test.
