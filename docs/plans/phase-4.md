# Phase 4 — autonomous residents

The baker and porter follow work routines, choose where to take breaks and keep private notebooks of what they witness. Their individual preferences influence their choices. The terminal shows the people and objects the player can observe. The simulation chooses and validates resident actions offline. [Master design, sections 9, 11, 40–41 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 3 acceptance](phase-3-results.md)

## Checkpoints

| Checkpoint | Behavior to verify |
| --- | --- |
| 1. Work routine | The baker opens the door, walks to the shopfront and returns to the prep area. A player can wait, help by opening the door, or walk away. The baker continues out of sight. |
| 2. A need interrupts work | One modeled need can take priority over a scheduled activity. After the resident meets the need, they can return to work. Restarting mid-interruption preserves the choice. |
| 3. Private notebook | A resident records something they personally observed with its source and time. Another resident who missed the event has no such entry. |
| 4. Personality influences a choice | Two residents with different preferences can choose different actions in the same circumstances. Replaying either decision reproduces it. |
| 5. Replay and handoff | Follow multiple residents through a missed encounter, restart and failed save. Compare the resulting histories and actor observations, and record the local and hosted Linux results. |

## Checkpoint 1 results

On 26 November 2026 the seed-42 baker started in the prep area behind the closed door. The prototype alternates four ticks at a time between aiming for the shopfront and aiming for the prep area. The baker can move one square or open the door in a turn. After arriving, the baker stays there until the destination changes. When the player opened the door first, the baker stepped into the doorway on the same tick. The occupied doorway then prevented the player from closing it.

The player can type `wait` to spend a turn where they stand. Successful actions advance the simulation once. Looking, listing actions, rejected commands and repeated requests spend no time. Both characters use the same rules for movement and doors. In a second walk the player headed south. The baker returned indoors on tick 6, out of the player's sight. [Try the terminal session](../../README.md#meet-the-baker) · [Simulation tests](../../internal/simulation/resident_test.go)

New baker timelines use bakery content version 2. A save keeps its original content version on reopen. Each accepted turn writes the player action, any baker action, the clock, the scene and the receipt in one SQLite transaction. A test failed the write of the baker's first action and reopened the save. The original closed door, tick-zero checkpoint, event history and receipts all survived. Retrying committed one turn. [SQLite test](../../internal/storage/resident_test.go)

With ten waits on seed 42, separate runner processes reproduced the following states. Replay from sequence 0 and from sequence 3 reached the same final digest.

| Point | Tick | Event sequence | SHA-256 |
| --- | ---: | ---: | --- |
| The baker opens the door | 1 | 3 | `496544a61702f551091cbdbd605b1d7cb97f29f495832bfad3cbe36b1ea270ca` |
| The baker reaches the shopfront a second time | 10 | 27 | `07d0038bf8e137f1431b101ed11ff48519186a2b315b23aabd8de22ad64d591c` |

The full offline Linux check passed with Go 1.27.4, including both seed-42 smoke runs. The terminal, simulation and storage packages also passed three race-detector runs. A separate save written by the final Phase 3 executable reopened with the same tick-2 replay digest and accepted a new move at tick 3. [Offline verification script](../../scripts/check-phase-0.sh)

Repeat the focused checks from the repository root.

```sh
go test -count=1 -v -run '^TestBaker(WorkRoutine|WorksOutOfSight|UsesSharedDoor|RollbackAndReplay|TerminalSession)$' ./internal/simulation ./internal/storage ./cmd/sloptown
```

The hosted Linux verification passed for checkpoint 1 at `c8d26f9`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45765257972)

## Checkpoint 2 results

Walking builds fatigue. In this prototype, after five successful steps the baker chooses the back of the bakery as a place to rest. Opening a closed door costs a turn, and a blocked move adds no fatigue. The baker stays on the way to the break even if the shift changes. On arrival the baker spends one turn sitting down, recovers, and follows the current shift on the next turn. Players use the same movement and rest rules and choose their own resting place.

On 26 November 2026, a player waiting outside on seed 42 saw the baker head for the shopfront and then turn back in the doorway. At tick 12 the player could see the baker sitting at the back of the bakery. At tick 13 the baker moved to the prep area. A player who walked down the street missed the break. Giving the baker an early rest at tick 8 changed the outcome at tick 10, when the baker reached the shopfront. [Work routine and observation tests](../../internal/simulation/resident_test.go)

| Seed-42 save with the player waiting outside | Tick | Event sequence | SHA-256 |
| --- | ---: | ---: | --- |
| Baker has turned back and is walking to the resting place | 10 | 27 | `89920a3c8181e5eaf21b9a1cf316e5e591199aa4fdba63b3aa9edbf09d036b9e` |
| Baker has rested and returned to prep | 13 | 36 | `8e2b1876a8e108382a11b3319e77e24f01d54220ba15932e9ab74f3f497ce535` |

Reopening mid-break reproduced the uninterrupted observations and events. Replay from sequences 0 and 27 produced the same tick-13 digest. Forcing SQLite to fail while writing the rest event preserved both actors, the clock, the receipt history and the events. Retrying committed the break once. [Restart and rollback test](../../internal/storage/resident_test.go)

New `-resident` timelines use content version 3. A separate compatibility run reopened a version 2 save made by checkpoint 1. Both executables reproduced the original tick-10 digest. After three new commands both also reproduced the tick-13 digest for that work-only routine. The full offline Linux checks and three race-detector runs passed locally.

Repeat the focused checks from the repository root.

```sh
go test -count=1 -v -run '^(TestBaker(WorkRoutine|WorksOutOfSight|RestAcrossRestart|RollbackAndReplay|TerminalSession)|TestRestChangesTheBakersNextChoice)$' ./internal/simulation ./internal/storage ./cmd/sloptown
```

The hosted Linux verification passed for checkpoint 2 at `a555e60`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45766498455)

## Checkpoint 3 results

New resident bakeries include a porter on the street. For this checkpoint each character keeps firsthand sightings of someone else sitting down or opening or closing a door or container. At each action the simulation checks whether the witness can see the person and, for object interactions, the object. Each entry stores the observed action and references, the tick, and the source event's sequence within that timeline. The developer notebook command reads one owner's entries. Player scenes, suggestions and receipts continue to use the character's current view. [Notebook rules](../../internal/simulation/notebook.go) · [Terminal walkthrough](../../README.md#inspect-the-residents-notebooks)

On 26 November 2026, the following seed-42 walkthrough ran across process restarts.

| Action | Source event | What the residents saw |
| --- | ---: | --- |
| Player sat outside on tick 1 | 2 | The porter saw the player sit. The baker was behind the closed door and opened it later that tick. |
| Player sat inside on tick 4 | 11 | The baker saw the player sit. The bakery wall blocked the porter's view. |
| Player opened the crate on tick 5 | 13 | The baker could see the player but could not see the crate until moving later that tick. Neither resident recorded the opening. |
| Player closed the crate on tick 6 | 16 | From the doorway, the baker saw both the player and the crate. The porter missed it. |

Resting a second time while already seated added no sighting. In a separate run the player saw the baker sit down at the back of the bakery on tick 12. The porter missed the break. [Observation tests](../../internal/simulation/notebook_test.go)

Replaying the six-turn walkthrough from sequences 0 and 3 reproduced the same tick-6, sequence-17 digest, `b6ba766b3ae5505f6bc231b035a5a53873aefd2843c05791095b9b446d495a56`. Failing the receipt write rolled back the crate closure, its event and the baker's entry together. A retry recorded one sighting. A second timeline in the same save had an empty notebook. [Restart test](../../internal/storage/notebook_test.go)

Notebook timelines use bakery content version 4. A version 3 save from checkpoint 2 reopened in the new runner, matched its tick-10 replay, and continued to the original tick-13 digest. The full offline Linux checks passed, along with three race-detector runs of the simulation, storage and terminal packages. Run the notebook checks with

```sh
go test -count=1 -v -run '^TestNotebook' ./internal/simulation ./internal/storage
```

The hosted Linux verification passed for checkpoint 3 at `920caf0`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45839368340)

## Checkpoint 4 results

New resident bakeries use content version 5. The porter now joins the baker on the route between the shopfront and the prep area. At generation, each worker draws a stable preference between 1 and 99 percent for resting where they stand. When fatigue calls for a break and both locations are available, a worker draws once from their own stream. The simulation saves the decision, its source request and the chosen destination. A worker heading to the back continues there across turns and restarts. When the worker is already there or the next step is blocked with no door they can open, they can sit immediately.

In the seed-42 walkthrough on 26 November 2026, the player waited outside. At tick 9 the baker and porter stood together in the open doorway. Each had taken five steps since their last rest. Both chose a resting place at tick 10.

| Worker | Chance of sitting in place | Draw (0–99) | Observable action |
| --- | ---: | ---: | --- |
| Baker | 40% | 58 | Stepped into prep, reached the back at tick 11 and sat down at tick 12. |
| Porter | 87% | 48 | Sat down in the doorway and returned to the shopfront at tick 11. |

A test exchanged just the two preferences. With the same starting positions, clock and random draws, the baker sat down in the doorway and the porter walked to the back. The ordinary terminal shows the workers' actions. The developer `profile` command reads the saved percentage without advancing the timeline. [Simulation tests](../../internal/simulation/personality_test.go) · [Play the walkthrough](../../README.md#meet-the-baker)

Replay from sequence 0, sequence 29 (just before the choices) and sequence 35 (after both choices) reached tick 13, sequence 46, with SHA-256 `42fc440f92d381c50c0b96b11c6e72456cef5116399ec811914e5ec38b9fb24d`. A failed SQLite write during the porter's decision rolled back both workers' turns, including the random draws. The retry committed each decision once. A save written by checkpoint 3 reopened, matched the published tick-6 digest, and continued to tick 13 with the same result in both executables. [SQLite restart and rollback test](../../internal/storage/personality_test.go)

The full offline Linux checks passed, including both seed-42 smoke runs. Three race-detector runs passed for the simulation, terminal and storage packages. Repeat the focused checks with

```sh
go test -count=1 -v -run '^(TestPersonality|TestRestPreference|TestRestoreInvalidRestPreference)' ./internal/simulation ./internal/storage
```

The hosted Linux verification passed for checkpoint 4 at `2c7567b`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45841597354)

## Checkpoint 5 results

The porter saw the baker take a break while the player was down the street. Interrupting the save on that turn and retrying produced the same event history and the same three private notebooks as playing straight through. Both replays reached tick 14, sequence 49. The local offline and race checks passed. Hosted Linux verification passed at `a463d2a`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45842625342) [Phase 4 acceptance results](phase-4-results.md)
