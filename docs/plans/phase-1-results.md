# Phase 1 acceptance report

**25 November 2026 · Phase 1 exit criteria passed at `4575e15`.** A seed-42 timeline reached the same tick, random stream position and ordered history after restarting the terminal runner as it did in an uninterrupted simulation. An identical retry returned the original result. A failed write left the previous save usable. The acceptance test exercised all three cases using the built `sloptown` executable and a temporary SQLite file. [Acceptance test](../../internal/storage/phase1_acceptance_test.go) · [Passing hosted run](https://github.com/dgang0404/slop-town/actions/runs/45755020507)

## Fixture and results

The fixture uses world `smoke`, timeline `main`, actor `player` and the `weather` stream. Each new `fixture_step` advances one tick, makes one draw and records two events. The test starts a new process for each CLI invocation and reads the committed state through a fresh database handle. In parallel with those steps, a reference execution resolves the three distinct requests in memory, without a restart or injected failure.

| Operation | Observed result |
| --- | --- |
| Submit `r1`, then checkpoint | Tick 1, sequence 2, one draw and one receipt. |
| Restart, submit `r2`, then replay from sequence 2 | Tick 2, sequence 4. The saved clock, stream counters, ordered events and receipts matched the reference execution. |
| Restart and submit `r2` again | The CLI returned the original result. The save still contained four events and two receipts. |
| Inject a SQLite failure at the final write for `r3` | The runner exited 1, reported the injected fault and emitted no success result. Reopening the file showed the same clock, stream counters, sequence, events and receipts. Replay reproduced the tick-2 hash. |
| Remove the fault and retry `r3` | Tick 3, sequence 6, three draws and three receipts. The saved state and event history matched the uninterrupted execution. |

The CLI reported these SHA-256 values. The digest covers scope, sequence, versions and the deterministic clock snapshot. The test compares the event and receipt records separately.

| State | SHA-256 |
| --- | --- |
| Tick 1 checkpoint | `d899d2cf3ae733c0ba0fce7d0f64a6ad5236654ae33a5284966006d373abe034` |
| Tick 2 replay, including after rollback | `d508c6d2e9416be9d60c32bbc51cb36d2c2be7e13c146057d74dad10c525554d` |
| Tick 3 replay after retry | `39dbddba99553b8b9754617c159317ff7627457297497b73dd3886997e9c4d4c` |
| Original Phase 0 seed-42 fixture, both runs | `60261db115f851c52ad8e47ec6c6b787992eab8518bf7656591bd22b923e96b5` |

## Reproduce the checks

From a clean checkout, with Go 1.25.0 or newer, restore the pinned modules and run the same [verification script](../../scripts/check-phase-0.sh) used by Linux CI:

```sh
go mod download
sh scripts/check-phase-0.sh
```

The script disables Go downloads after restore. Look for `TestPhase1Acceptance` in its output. To run only the process-restart demonstration with dependencies already cached:

```sh
GOPROXY=off GOTOOLCHAIN=local CGO_ENABLED=0 go test -mod=readonly -count=1 -v -run '^TestPhase1Acceptance$' ./internal/storage
```

## Verification status

- **Local.** A clean Linux checkout of `4575e15` passed the full suite with Go 1.27.4 after restoring dependencies from the local module archive. Vet, both command builds, the content fixture and both Phase 0 smoke runs passed. The runner, storage and simulation packages also passed three runs with the Go race detector.
- **Hosted.** GitHub Actions ran `TestPhase1Acceptance` on Ubuntu 24.04 with Go 1.25.0. It logged the tick-2 hash before and after the injected failure and the tick-3 hash after recovery. The complete verification job passed at `4575e15`. [Run 45755020507](https://github.com/dgang0404/slop-town/actions/runs/45755020507)

## Phase 2 review

Phase 1 covers a synthetic action and a deliberately failed database transaction. Device packaging, world generation, and recovery after an abrupt device shutdown remain separate work. For Phase 2, review a small seeded layout with stable space and object IDs, what can move between spaces or containers, and what the actor can see before and after moving. Agree on a fixture whose positions, contents and visible facts can be checked after save and replay. Phase 2 implementation requires owner approval. [Master roadmap, section 50](../design/Slop_Town_Master_Design_Revised_v7.docx)
