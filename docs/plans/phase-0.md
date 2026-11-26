# Phase 0 development plan

Use this plan for the implementation stack alongside the [original workstreams](../design/Slop_Town_Phase_0_Development_Foundation.docx). The [Go decision](../decisions/0001-go-simulation.md) updates the toolchain and client scope.

| Workstream | Revised deliverable |
| --- | --- |
| 00 Environment | Verify Go on Linux and record the offline build and test commands. |
| 01 Repository | Maintain the design documents, contribution instructions, and licensing. |
| 02 Project structure | Build a Go module with a terminal entry point. Keep future simulation rules independent of storage and presentation. |
| 03 Architecture | Define authority, commands, events, IDs, time, and persistence contracts before implementing gameplay. |
| 04 Persistence | Exercise SQLite initialization, reopening, transaction rollback, and migration safety. |
| 05 Runner | Run a seeded fixture and check its state hash after reopening the database. |
| 06 Content | Specify and validate a minimal content manifest and document safe loading. |
| 07 Verification | Run Go tests and the offline smoke command locally, then add Linux CI. |
| 08 Handoff | Review Phase 0 acceptance evidence and draft the Phase 1 specification. |

## Architecture checkpoint

The [simulation contracts](../decisions/0002-simulation-contracts.md) define the proposal, result, event and observation records for future clients. The simulation validates an actor's proposal and publishes only committed outcomes. The decision also covers engine ownership, execution order, timeline identity, deterministic time and randomness, and recovery after a failed write.

The first portability check uses the desktop runner for SQLite and the simulation fixture alone for WebAssembly. With the pinned modules cached, reproduce the builds from the repository root in a POSIX shell. The WebAssembly check also requires Node on the build machine.

```sh
out=$(mktemp -d)
export GOPROXY=off GOTOOLCHAIN=local CGO_ENABLED=0
go test -mod=readonly -count=1 ./...
go build -mod=readonly -o "$out/sloptown" ./cmd/sloptown
"$out/sloptown" -db "$out/save.sqlite" -seed 42
"$out/sloptown" -db "$out/save.sqlite" -seed 42
GOOS=windows GOARCH=amd64 go build -mod=readonly -o "$out/sloptown.exe" ./cmd/sloptown
GOOS=darwin GOARCH=arm64 go build -mod=readonly -o "$out/sloptown-mac" ./cmd/sloptown
GOOS=js GOARCH=wasm go test -exec="$(go env GOROOT)/lib/wasm/go_js_wasm_exec" \
  -count=1 -run '^TestFixtureV1$' ./internal/simulation
```

The checks ran on 25 November 2026 with Go 1.27.4 on Linux x86-64 after caching the Windows and macOS dependencies of the existing SQLite driver.

| Target | What passed | Remaining device checks |
| --- | --- | --- |
| Linux x86-64 | All five tests passed. The runner saved and reopened seed 42 twice with the reference hash. | Packaged app launch and saves from a normal user account |
| Windows x86-64 | Cross-compiled the complete runner and verified its PE executable format. | Launch, save, close and reopen on Windows |
| macOS Apple silicon | Cross-compiled the complete runner and verified its arm64 Mach-O executable format. | Launch, save, close and reopen on macOS |
| WebAssembly | The simulation reproduced the reference hash under Node 22.23.2. | Browser client, touch input, offline startup and persistent saves on target devices |

## Go package boundaries

| Package | Responsibility | Production dependencies |
| --- | --- | --- |
| `cmd/sloptown` | Parse flags, save and reopen the fixture, compare values, print the hash or error. | Simulation and storage |
| `internal/simulation` | Generate the seeded fixture and encode its canonical state for hashing. | Go standard library |
| `internal/storage` | Open SQLite files, run migrations, and commit or roll back fixture writes. | `database/sql` and `modernc.org/sqlite` |

Run `go test -count=1 ./internal/simulation` to check the seed-42 reference hash and the production dependency graph. The Phase 0 boundary test rejects SQL and packages outside the standard library or simulation tree, including transitive imports from adapters. See the [repository instructions](../../AGENTS.md) for ownership and contribution guidelines.

On 25 November 2026, the offline build and all five tests passed. Two successive runs against one save produced the original seed-42 hash, `60261db115f851c52ad8e47ec6c6b787992eab8518bf7656591bd22b923e96b5`. In a separate check, injecting a `database/sql` import into the simulation with a temporary Go source overlay caused the dependency test to fail as expected.

## Persistence checkpoints

The runner saves a seeded fixture, reopens the SQLite file, and verifies the value before reporting its hash.

Schema version 1 uses SQLite's `PRAGMA user_version`. The runner upgrades an unversioned save in the same transaction as the fixture insert. It keeps existing fixture rows and adds a single `fixture_metadata` row identifying the smoke world (`smoke`) and its timeline (`main`). Later runs skip the migration.

With dependencies cached, run `GOPROXY=off go test -count=1 -v ./...` to reproduce the persistence checks. The tests run the upgrade twice and force an insert to fail after changing an existing fixture. They reopen the save to verify its original hash, check that the new row and schema changes rolled back, and retry the save.
