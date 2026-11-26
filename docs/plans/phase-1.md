# Phase 1 proposal — authoritative time, events and replay

**Draft for owner approval, 25 November 2026.** The first milestone is a timeline the player can quit and resume without changing what happens next. Demonstrate it through the headless runner with a small synthetic action. Phase 2 introduces generated space and objects. Conversation choices, residents, rendering and AI have their own later milestones. [Master roadmap, section 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 0 acceptance](phase-0-results.md)

## End-to-end demonstration

Create a fixture timeline at tick 0 with a known seed. Submit two uniquely identified requests that each advance time by one tick and draw a value from a named random stream. Save after the first request, exit, reopen the timeline and submit the second. Compare that history with a second execution from identical initial conditions that stayed running. Both executions must end at tick 2 with the same event sequence, random stream state and canonical state hash.

Resubmit the second request. It must return the committed result and leave the timeline at tick 2. Inject a persistence failure during a fresh request and reopen the save. Neither the timeline nor the stream may have advanced. A retry must produce the same result as a run that succeeded on its first attempt.

## Implementation order and acceptance tests

| Checkpoint | Depends on | Deliverable and evidence |
| --- | --- | --- |
| 1. Clock and random streams | Owner approval and the Phase 0 contracts | In `internal/simulation`, represent the integer tick and versioned state of named deterministic streams. Test tick 0, successive advances, identical sequences from the same initial state, interleaved draws from a second stream, and continuation from a serialized state. Reject invalid advances without changing state. This first checkpoint can run entirely in memory. |
| 2. Command resolution and event order | 1 | Resolve the synthetic action through the coordinator. Bind an actor and a world/timeline context, validate contract versions, and stage changes with a request receipt. Assign stable event IDs and a total sequence order across equal ticks. Test deterministic ordering, valid causal parents, rejected proposals, identical retries and conflicting reuse of a request ID. |
| 3. Transactional SQLite saves | 1 and 2 | Add a versioned migration for timeline state, stream positions, ordered events and receipts. Preserve existing Phase 0 fixtures and their reference hash. Publish results after commit. Inject a failure between writes, close and reopen the file, and verify that state, events, receipt, tick and RNG all rolled back. Competing submissions of the same request must produce one committed result. |
| 4. Replay from a checkpoint | 2 and 3 | Store a checkpoint with its world and timeline IDs, sequence position, engine/content versions and deterministic state. Replay subsequent records in sequence. Compare event history and hashes with uninterrupted execution. Test an empty suffix, a missing or reordered event, incompatible versions and two independent timelines. Report the first failing sequence or version in developer diagnostics. |
| 5. Headless acceptance and handoff | 1–4 | Run the demonstration above across process restarts in Linux CI using a temporary database. Keep the Phase 0 command and reference fixture passing. Record repeatable commands, actual hashes, rollback evidence and review items for Phase 2. |

## Rules to settle in implementation

- Carry the [existing simulation contracts](../decisions/0002-simulation-contracts.md) into executable types. Keep SQL handles, wall-clock time and logging run IDs outside authoritative state. A failed storage attempt must never publish staged events.
- Specify the canonical encoding before freezing new reference hashes. Include the logical identity, tick, ordered authoritative state and stream positions. Order keyed collections explicitly so machine and iteration order cannot change the digest.
- Record enough information in each committed transition to reconstruct its state and stream positions during replay. Replay uses recorded inputs and versioned rules and must have no external side effects.
- Keep world and timeline identity distinct from seed and request ID. Scope retries to actor and timeline. Validate that a checkpoint belongs to the selected timeline before applying any records. Player-facing branching and its interface remain later work.

**Exit criterion.** A clean checkout passes the Phase 0 suite and the restart, replay, retry and rollback scenarios above after dependency restore. The handoff identifies the exact fixture, commands and CI run. Advancing to the next phase requires a separate review.
