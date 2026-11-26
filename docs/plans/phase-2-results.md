# Phase 2 bakery acceptance

On 25 November 2026, the seed-42 bakery passed the local Linux restart and replay walkthrough. A fresh runner process performed each action. After each restart, the clock and streams, spatial snapshot, ordered events, request receipts and actor observation matched the uninterrupted in-memory simulation. [Acceptance test](../../internal/storage/phase2_acceptance_test.go)

| Point in the walkthrough | Tick | Event sequence | SHA-256 of the canonical state |
| --- | ---: | ---: | --- |
| The spoon is in the player's inventory, beside the open crate | 5 | 10 | `c35a6eb2830ac7c08e171b564543528af3952f11cef5102eab379aca54285812` |
| The spoon is on the street at (1,1), with the door and crate closed | 10 | 20 | `9f421909302942c47ba2ac7c46c627591ef95360749cff544ce23d206bafe58b` |

Replay from either sequence 0 or sequence 10 produced the final digest. The digest covers world and timeline identity, sequence, clock, random streams and the versioned bakery state. The bakery encoding writes cells in row-major order and entities by sorted ID. Replay also checks each event and each command's recorded result, including the observation and its target references.

## Recovery and compatibility

- A SQLite trigger failed the receipt write for the pickup, after the transaction had staged the placement and events. Reopening showed the spoon inside the open crate at tick 4. Removing the trigger and retrying moved it into inventory once. A later retry of a committed action returned its original receipt.
- A second timeline in the same database started with identical seed-42 geometry and placements. Its player walked west while the first timeline retained the completed walkthrough.
- SQLite schema 4 adds an optional bakery snapshot to each timeline head. Version 1 checkpoints retain their Phase 1 encoding and hashes. Version 2 checkpoints include the bakery. The compatibility tests read a schema-3 save before migration and verified the same replay hash after upgrade.

## Verification

Local verification passed on Linux with Go 1.27.4. The runner, storage and simulation packages passed three race-detector runs. The bakery simulation tests, including round trips of both reference layouts, also passed under Node.js with WebAssembly.

Run `sh scripts/check-phase-0.sh` from the repository root after restoring Go dependencies. The script builds and tests offline, runs both process-restart acceptance cases, and checks the original seed-42 smoke hash twice. To repeat just the new acceptance case, run `go test -count=1 -v -run '^TestPhase2Acceptance$' ./internal/storage`. [Terminal walkthrough](../../README.md#save-and-replay-the-bakery)

On 26 November 2026 the hosted Linux job passed at `405b5f6`. Its log records the failed pickup and successful reopen at tick 4, the tick-5 checkpoint digest and the matching tick-10 digest for both replays. [GitHub Actions run 45758907911](https://github.com/dgang0404/slop-town/actions/runs/45758907911/job/139703985542)
