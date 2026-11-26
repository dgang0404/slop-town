# Phase 0 acceptance report

**25 November 2026 · verified revision `b076d5c` · ready for owner review.** A clean checkout can build two Go commands, validate a local pack and reopen a deterministic SQLite save. The local verification script and the GitHub-hosted run both passed. The approved Go decision replaced the original C# deliverable and moved Godot compatibility testing to the future client milestone. [Go decision](../decisions/0001-go-simulation.md) · [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45752309419)

## Reproduce the checks

Use Git and Go 1.25.0 or newer on Linux. From the repository root, run:

```sh
go mod download
sh scripts/check-phase-0.sh
```

The restore step needs access to the pinned modules. Once cached, the script disables Go downloads, runs vet and the tests, builds both commands, checks the bundled pack, and runs the fixture twice against one temporary save. On the laptop the script exited 0. All 11 top-level tests passed, including their manifest-validation subtests. The final lines were:

```text
valid manifest: smoke version 1 (content API 1)
Smoke run 1: seed=42 sha256=60261db115f851c52ad8e47ec6c6b787992eab8518bf7656591bd22b923e96b5
Smoke run 2: seed=42 sha256=60261db115f851c52ad8e47ec6c6b787992eab8518bf7656591bd22b923e96b5
```

The runner reports exit 1 for a failed save or verification and 2 for invalid arguments. A successful fixture run exits 0. The [checkpoint log](phase-0.md) records the tamper, rollback, invalid-input and trace checks.

## Environment and dependencies

The development laptop reported Ubuntu 22.04.5 LTS with the Lubuntu desktop installed, LXQt, x86-64, 8 logical CPUs, 15 GiB of RAM and 449 GiB available on the workspace filesystem. The local tools were Git 2.56.1 and Go 1.27.4. The embedded driver returned SQLite 3.53.3 from `SELECT sqlite_version()`. The hosted job used the Ubuntu 24.04 runner and the Go version selected from the module file. These measurements describe the build machines. Player hardware targets have a [separate evidence table](../system-requirements.md).

The sole direct Go module dependency is `modernc.org/sqlite v1.56.0`. Its indirect modules are go-humanize, uuid, go-isatty, go-strftime, bigfft, x/sys, libc, mathutil and memory. Exact versions and checksums are in [go.mod](../../go.mod) and [go.sum](../../go.sum). Simulation code uses the Go standard library. The fixture runs without a model, service account or separate database server.

## Acceptance decisions

| Original Phase 0 requirement | Result | Evidence or disposition |
| --- | --- | --- |
| Lubuntu-compatible environment | PASS | Local verification completed on the laptop described above. |
| Git repository and structure | PASS | Tracked commands under `cmd/`, core and storage under `internal/`, a bundled pack under `content/`, and phase plans under `docs/`. The CI script uses temporary outputs. |
| C#/.NET simulation solution | PASS under revised scope | The Go module built. `TestSimulationDependencies` checked the production dependency boundary. |
| Godot integration compatibility | DEFERRED by approved decision | The Go decision removes the Godot build from Phase 0. Revisit compatibility when selecting the graphical client. |
| SQLite persistence foundation | PASS | `TestSmokePersistence`, `TestLegacyMigrationRunsTwice` and `TestFailedSaveRollsBack` passed. Recovery evidence appears below. |
| Automated build and tests | PASS | [Verification script](../../scripts/check-phase-0.sh) exited 0 locally and in CI. |
| Contributor instructions | PASS | [AGENTS.md](../../AGENTS.md) defines ownership, verification and phase boundaries. |
| Master design revision 7 indexed | PASS | The [README](../../README.md) links both original design documents. |
| Architecture decision records | PASS | The [Go decision](../decisions/0001-go-simulation.md) and [simulation contracts](../decisions/0002-simulation-contracts.md) record stack and authority choices. |
| Engine interface conventions | PASS for design | The contracts specify proposal, result, event and observation fields, owners, versioning and error handling. Runtime enforcement belongs to Phase 1. |
| Logging and diagnostics | PASS | `TestTraceOutput` and `TestTraceSaveFailure` passed. Records carry run, seed, world and timeline context. |
| Headless simulation entry point | PASS | The `sloptown` binary reopened the seed-42 fixture and printed the reference hash twice. |
| Content pack conventions | PASS | `packcheck` accepted the smoke manifest. Tests rejected malformed, incompatible, linked and oversized inputs. |
| Free-first dependencies and licensing | PASS | The [README](../../README.md#license) records GPL-3.0-or-later for Go code and CC BY-SA 4.0 for documentation. The [pack guide](../content-packs.md) covers third-party notices. |
| Initial CI pipeline | PASS | GitHub Actions run [45752309419](https://github.com/dgang0404/slop-town/actions/runs/45752309419) completed successfully for `b076d5c`. |
| Phase 1 implementation specification | PASS for handoff | The [draft Phase 1 plan](phase-1.md) defines a first checkpoint, dependencies and acceptance tests. Implementation awaits owner approval. |

## Save recovery and remaining limits

For a backup, use SQLite's backup API to copy the source into a fresh destination. Keep the original while checking the copy. In the recovery exercise, Python's `sqlite3.Connection.backup` read the source in read-only mode. On the copy, `PRAGMA integrity_check` returned `ok`, `user_version` was 1, the saved seed/value rows matched the source, and the metadata pair was `smoke/main`. Running the fixture against the copy then returned the reference hash and exited 0. Checking the rows **before** running the command matters because the runner can insert a missing fixture.

Phase 0 established deterministic fixture storage. Full-world replay, crash recovery during gameplay, and save growth require later tests. The Windows and Apple silicon runners have build evidence. Device launches, mobile/browser saves and playable hardware minimums remain open. Gameplay tuning, personality formulas, town size, art direction and final mod APIs remain design work for later phases.

## Change history

| Commits | Delivered files and behavior |
| --- | --- |
| `71bfb30`, `b5f5531`, `8d33c80` | Design sources, license files, Go decision, module and first SQLite runner. |
| `c2692ae`, `78290d1` | Atomic fixture migration, rollback tests, `internal/simulation`, `internal/storage` and repository instructions. |
| `17800aa`, `5e94546` | Simulation contracts, portability evidence, player requirements and trace output. |
| `66a793e`, `b076d5c` | Pack checker and sample manifest, Linux verification script and GitHub workflow. |
| This handoff | Acceptance report, hosted CI result and draft Phase 1 specification. |
