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

## Persistence checkpoints

The runner saves a seeded fixture, reopens the SQLite file, and verifies the value before reporting its hash.

Schema version 1 uses SQLite's `PRAGMA user_version`. The runner upgrades an unversioned save in the same transaction as the fixture insert. It keeps existing fixture rows and adds a single `fixture_metadata` row identifying the smoke world (`smoke`) and its timeline (`main`). Later runs skip the migration.

With dependencies cached, run `GOPROXY=off go test -count=1 -v ./...` to reproduce the persistence checks. The tests run the upgrade twice and force an insert to fail after changing an existing fixture. They reopen the save to verify its original hash, check that the new row and schema changes rolled back, and retry the save.
