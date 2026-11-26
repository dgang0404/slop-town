# Phase 1 — authoritative time, events and replay

**Approved by the owner, 25 November 2026.** The first milestone is a timeline the player can quit and resume without changing what happens next. Demonstrate it through the headless runner with a small synthetic action. Phase 2 introduces generated space and objects. Conversation choices, residents, rendering and AI have their own later milestones. [Master roadmap, section 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 0 acceptance](phase-0-results.md)

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

## Checkpoint 1 results

The in-memory timeline starts at tick zero. A positive advance adds that many ticks. Zero, negative and overflowing advances fail without changing the timeline. Each named stream has its own unsigned 64-bit draw counter. For example, extra weather draws leave the next movement draw unchanged.

Version 1 snapshots use UTF-8 JSON with `version`, signed 64-bit `seed` and `tick`, and a `draws` object containing the count for each used stream. An unused stream starts at index zero. To reproduce a draw, concatenate the seed's 64-bit two's-complement representation and the zero-based draw index (both eight bytes, big-endian), followed by the UTF-8 name. Read the first eight SHA-256 digest bytes as an unsigned big-endian integer. Store the new count after a successful draw. This makes resuming a stream independent of the number of past draws. Future changes to the draw algorithm require a new snapshot version.

On 25 November 2026, `sh scripts/check-phase-0.sh` passed locally with Go downloads disabled, including the existing seed-42 SQLite smoke check. All five new timeline tests also passed as WebAssembly under Node. The resume test takes a snapshot at tick 1 and compares subsequent draws and serialized state at tick 2 with an uninterrupted run. This checkpoint covers clock and stream state in memory. Checkpoint 2 will resolve a synthetic action and record its events.

## Checkpoint 2 results

The [in-memory coordinator](../../internal/simulation/coordinator.go) owns one world and timeline. A caller supplies the actor separately and submits a version 1 `fixture_step` proposal with a request ID and a named stream. The coordinator stages a one-tick advance and one draw. On success, it publishes the new state, two events and the actor's request receipt together.

For two successive steps, tick 1 has sequences 1 and 2, and tick 2 has sequences 3 and 4. Each `tick_advanced` event identifies the causing request and actor. Its `stream_drawn` child records the parent event ID, stream and sampled value. Event IDs combine the escaped world and timeline IDs with the sequence. The caller receives an actor-scoped observation. The developer history exposes the ordered events.

Within that actor and timeline, an identical retry returns the stored result. Changing the stream under an already committed request ID yields `request_conflict`. Invalid proposals and failed draws leave the clock, streams, events and receipts untouched. These receipts currently last for the lifetime of the coordinator. Checkpoint 3 will persist them with the state and events in one SQLite transaction.

On 25 November 2026, `sh scripts/check-phase-0.sh` passed offline and both SQLite smoke runs produced the reference hash. The four coordinator tests also passed five times under the Go race detector and once under WebAssembly in Node. They covered deterministic histories, equal-tick ordering, causal links, actor and timeline isolation, rejected commands, retries after a later commit, and 12 concurrent submissions of one request. Reproduce the focused Linux checks with:

```sh
go test -race -count=5 -run 'Test(Command|ScopedCommand|ConcurrentCommand)' ./internal/simulation
```

## Checkpoint 3 results

Schema version 2 preserves the Phase 0 seed/value rows and adds three tables. `timeline_state` stores each scope's versioned clock snapshot, stream counters and last event sequence. `timeline_event` stores the ordered event records. `request_receipt` keys the original proposal and result by world, timeline, actor and request ID.

The SQLite adapter acquires a write transaction before it reads the timeline or checks for a receipt. It uses the same resolution function as the in-memory coordinator. For a fresh accepted request, it writes the clock, events and receipt together and returns the result after commit. A competing writer reads the committed receipt when it obtains the lock. The runner can exercise this path across process restarts through the [step command](../../README.md#try-a-persistent-phase-1-step).

On 25 November 2026 the offline verification suite passed, including the original seed-42 hash. The storage tests upgraded a version 1 fixture and injected a receipt failure after writing state and events. A fresh connection saw exactly the previous clock, stream positions, events and receipts. Retrying produced the same records as a clean two-step run. In a separate Linux CLI check, 12 processes submitted one request to a new save. A second unique request, a late retry and a conflicting retry left the save at tick 2 with two draws, four events and two receipts.

The next checkpoint will replay recorded events from a versioned checkpoint and compare the resulting history and state with uninterrupted execution.

## Checkpoint 4 results

Schema 3 adds immutable bookmarks keyed by world, timeline and committed event sequence. The `checkpoint` command records the existing timeline's state at a command boundary. Repeating it at the same position verifies the existing bookmark. For older saves the migration adds the checkpoint table alongside the existing state, events and receipts.

The fixture's version 1 state digest is SHA-256 over compact Go JSON in this field order: `version`, `world_id`, `timeline_id`, `sequence`, `engine_version`, `content_version`, `clock`. The embedded clock uses the checkpoint 1 encoding with sorted stream names. The format, fixture engine and fixture content versions are each 1. Event history is checked separately by comparing each recorded event with the event regenerated from the saved request.

Replay starts at an explicitly saved bookmark. It reads the selected timeline through one read-only SQLite transaction and resolves each subsequent request in memory. It checks event order, metadata, causal links, payloads and recorded results, then compares the reconstructed state digest with the current head. The bookmark is the starting point for verification; events before it are outside the checked interval.

On 25 November 2026, the Linux CLI replayed the two seed-42 weather steps across process restarts. Starting at sequence 2 yielded tick 2 and sequence 4 with SHA-256 `d508c6d2e9416be9d60c32bbc51cb36d2c2be7e13c146057d74dad10c525554d`, also reproduced independently from the specified JSON. The storage tests checked an empty suffix, two independent timelines, missing and reordered events, an altered draw, a missing receipt, version mismatches and a changed head. [Reproduce the CLI run](../../README.md#replay-a-timeline-from-a-checkpoint).

Checkpoint 5 will exercise the full restart, replay, retry and rollback demonstration in hosted CI and collect Phase 1 handoff evidence.

## Rules to settle in implementation

- Carry the [existing simulation contracts](../decisions/0002-simulation-contracts.md) into executable types. Keep SQL handles, wall-clock time and logging run IDs outside authoritative state. A failed storage attempt must never publish staged events.
- Specify the canonical encoding before freezing new reference hashes. Include the logical identity, tick, ordered authoritative state and stream positions. Order keyed collections explicitly so machine and iteration order cannot change the digest.
- Record enough information in each committed transition to reconstruct its state and stream positions during replay. Replay uses recorded inputs and versioned rules and must have no external side effects.
- Keep world and timeline identity distinct from seed and request ID. Scope retries to actor and timeline. Validate that a checkpoint belongs to the selected timeline before applying any records. Player-facing branching and its interface remain later work.

**Exit criterion.** A clean checkout passes the Phase 0 suite and the restart, replay, retry and rollback scenarios above after dependency restore. The handoff identifies the exact fixture, commands and CI run. Advancing to the next phase requires a separate review.
