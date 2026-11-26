# Phase 4 — autonomous residents

The first resident is a baker who follows a work routine. Later checkpoints give residents needs that can interrupt a shift, private notebooks and individual preferences. The terminal shows the people and objects the player can observe. The simulation chooses and validates resident actions offline. [Master design, sections 9 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 3 acceptance](phase-3-results.md)

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
