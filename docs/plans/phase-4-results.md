# Phase 4 resident acceptance

On 26 November 2026, the seed-42 player sat outside the bakery, walked two steps south, waited through the workers' breaks and came back to the shopfront. The same fourteen commands ran in two saves. One terminal session played them straight through. The other restarted before the baker's break, encountered a failed save, and retried the turn. [Acceptance test](../../internal/storage/phase4_acceptance_test.go)

| Tick | Event | Who saw it |
| ---: | ---: | --- |
| 1 | 2 | The porter saw the player sit on the street. The closed door kept the baker from seeing it. |
| 10 | 35 | The baker saw the porter sit in the doorway. The player was down the street. |
| 12 | 42 | The porter saw the baker sit at the back of the bakery. The player's view showed the porter up the street and the wall blocked the view of the baker. |

On tick 14 the player could see the baker again. The player's notebook was still empty. In both saves the baker had one entry and the porter had two. Each entry pointed to the witnessed action in the event history.

## Failed save and replay

The test interrupted the SQLite receipt write on tick 12. Reopening the save recovered tick 11, event 39, with the same clock, bakery state, notebooks, events and receipts as before the attempt. A fresh terminal session acknowledged the eleven completed commands and committed the break once. Repeating the full walk after the player's return acknowledged all fourteen commands.

The two saves finished with identical events, receipts, bakery state and views for all three characters. Replaying the interrupted save from sequence 0 or sequence 39 reached tick 14, sequence 49, with SHA-256 `785f8746f60c3184a71ba325dc40b9ee5c787aa43af0cb19701ea8a16a31dcf2`.

## Verification

Run the walkthrough from the repository root with

```sh
go test -count=1 -v -run '^TestPhase4Acceptance$' ./internal/storage
```

The full offline check passed on Linux amd64 with Go 1.27.4, including vet, all tests, the build, pack validation and both seed-42 smoke runs. The terminal, simulation and storage packages also passed three race-detector runs. [Offline verification script](../../scripts/check-phase-0.sh)

Hosted Linux verification passed for checkpoint 4 at `2c7567b`. The checkpoint 5 hosted result is pending its push. The workflow runs the acceptance test and prints both replay hashes in the log. [Checkpoint 4 hosted run](https://github.com/dgang0404/slop-town/actions/runs/45841597354) · [Linux workflow](../../.github/workflows/phase-0.yml)
