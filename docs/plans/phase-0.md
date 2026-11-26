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

The first implementation checkpoint covers a seeded fixture persisted across database connections. Remaining workstreams and final Phase 0 acceptance require separate verification.
