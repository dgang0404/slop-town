# Phase 5 acceptance — a visit to the bakery

On 26 November 2026, two fresh seed-42 practice bakeries received the same terminal commands. One visit ran straight through. During the other, SQLite failed to save the player’s second rest. Both visits ended on tick 23 with the same player stamina, private notebooks, views, and 80 events. [Acceptance test](../../internal/storage/phase5_acceptance_test.go)

| Tick | What happened |
| ---: | --- |
| 1–3 | The player walked up to the porter and introduced themselves. The porter later introduced themselves to the baker. The porter remembered the player from event 6, and the baker remembered the porter from event 12. |
| 6–8 | The player felt tired on their fifth step and sat down on tick 8. The rest brought relief and counted as their first endurance practice. |
| 14–16 | A single `walk to the crate` took three steps and three ticks. The player arrived within reach of the crate. |
| 17 | The player sat down again. Both workers saw it. The developer history linked the second qualifying recovery and the onset of relief to event 56. |
| 18–23 | Relief lingered for two turns and had faded by tick 20. After training, the player felt tired on step six of the next walk. |

At the end, the porter recognized the player and the baker recognized the porter. The player had received no introduction. Each worker’s notebook contained one sighting of the player sitting at the crate, and each worker’s introduction still pointed to its original speaker and event.

## What survived the failed save

The test interrupted the SQLite receipt write on tick 17. A fresh process reopened at tick 16, sequence 54, with the player beside the crate. The reopened actor state and history matched their values before the attempted rest. Retrying acknowledged the sixteen completed turns and committed the rest once. Repeating the full visit after the final step acknowledged all twenty-three turns.

The restarted save matched the uninterrupted one on the clock and random streams, bakery state, ordered events and command receipts. The test also reopened the town from the perspective of each of the three characters and compared what they could see. Replay from the beginning, immediately before the failed rest and immediately after its successful retry (sequences 0, 54 and 58) produced the same final SHA-256, `d67316bab6c1483be1c791318e5aed3d2e1002a401c171193d473af835375792`.

## Repeat the walkthrough

Run the acceptance test from the repository root to exercise both saves and print all three replay hashes.

```sh
go test -count=1 -v -run '^TestPhase5Acceptance$' ./internal/storage
```

The full offline Linux check passed on amd64 with Go 1.27.4, including vet, the package tests, the build, pack validation and both seed-42 smoke runs. The terminal, simulation and storage packages also passed three race-detector runs. [Offline verification script](../../scripts/check-phase-0.sh)

The hosted Linux build, tests and smoke check passed for the push containing the relationship and demo-controls checkpoints at `aac5b0b`. The new acceptance test will run on GitHub after its commit is pushed. [Hosted Linux job](https://github.com/dgang0404/slop-town/actions/runs/45872493285/job/140091194974) · [Workflow](../../.github/workflows/phase-0.yml)
