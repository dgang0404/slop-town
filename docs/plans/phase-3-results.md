# Phase 3 command and replay acceptance

On 26 November 2026, two seed-42 test saves started with a wooden spoon and a brass spoon inside a closed wooden crate, and an open flour crate nearby. One runner read typed commands and numbered choices. The other received explicit verbs and full names. Each invocation started a new process. At ticks 4, 9 and 10 the saves agreed on the clock and random streams, object placements, ordered events, action messages and player observations, including visible target references. [Acceptance test](../../internal/storage/phase3_acceptance_test.go)

| Point in the walkthrough | Tick | Event sequence | SHA-256 of the canonical state |
| --- | ---: | ---: | --- |
| The wooden crate is closed again, with both spoons inside. Replay from sequence 0 or 8 reaches this state. | 9 | 18 | `4900c5f07b7ee6629c7a8bec660bd6564be9c01ac0a2fb496c9f3a2d62e55d3a` |
| After restarting the same batch, the player appends `open it` and reopens the wooden crate. | 10 | 20 | `f9d6411fd624963830cff8f08907e3eabac55eaf2ec4477967021698485c6fc9` |

## Retry after the scene changes

At tick 9 the terminal acknowledged all five repeated actions after the player supplied the original numbers. Neither hidden spoon appeared in the retry prompt. Changing the spoon number or the destination number produced a request conflict. The save retained its original events and receipts. The next new command used the wooden crate as the subject of `it`.

For a newly committed numbered choice, SQLite stores the original noun phrases and answers with the resolved request in one transaction. Replaying the world uses the resolved object references. Older receipts still replay. A numbered choice saved before checkpoint 4 has only its object references, so repeating it through the terminal depends on being able to select the same objects again. A separate compatibility run reopened a checkpoint 3 save, acknowledged its two door actions, matched its tick-2 replay hash, then appended one action at tick 3.

## Verification

Run `go test -count=1 -v -run '^TestPhase3Acceptance$' ./internal/storage` for the restart case. The full local offline script passed on Linux with Go 1.27.4, including the original seed-42 smoke hash on both runs. The terminal, simulation and storage also passed three race-detector runs. [Local verification script](../../scripts/check-phase-0.sh)

The hosted Linux jobs passed for checkpoint 3 at `fcee4a0` and checkpoint 4 at `92e67f6`. The final run passed `TestPhase3Acceptance`. It reproduced the tick-9 digest above from both sequence 0 and sequence 8, then the tick-10 digest after the appended command. [Checkpoint 3 hosted job](https://github.com/dgang0404/slop-town/actions/runs/45762689430/job/139716131166) · [Checkpoint 4 hosted job](https://github.com/dgang0404/slop-town/actions/runs/45763940069/job/139720209156)
