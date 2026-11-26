# Run the simulation on the player's device

Phase 0 architecture decision, 25 November 2026. The desktop game will bundle the Go simulation with its client and store the town locally. The player should be able to open the installed app, load a town and play offline. The terminal runner exercises the first save format. Gameplay, a graphical client and release packages belong to later phases.

## Who owns a change

Consider a player typing “open the door.” The client sends an intention referring to a door in the player's observation. The simulation checks reach, current state and the actor's capabilities. Only the engine responsible for the door can decide its new state. Once the change commits, the client receives what the character can perceive. A resident trying the same door goes through the same rules.

| Component | Owns | May depend on |
| --- | --- | --- |
| Client or optional AI adapter | Input, proposals, presentation of authorized observations | Command and observation contracts |
| Simulation coordinator | World and timeline identity, command validation, atomic resolution | Engine registry, simulation clock, random streams and a persistence interface |
| Domain engine | Its declared state, action rules and versioned event payloads | Declared engine inputs and services supplied by the coordinator |
| Storage adapter | Schema migrations and atomic reads and writes | Platform storage and the simulation's persistence contract |
| Observation projection | Facts and references visible to one actor | Committed events, perception and that actor's knowledge |
| Developer observatory | Diagnostics and inspection | Read-only access to committed state and event history |

The app connects these components in process. The UI and AI receive no database handle or mutable world state. An accepted AI proposal produces a recorded event so replay can use the recorded outcome offline. Player views, narration and suggestions consume the same filtered observations. Preserve the distinction between current perceptions, remembered facts and secondhand claims.

## Version 1 message shapes

These field definitions specify the Phase 1 implementation target. IDs are opaque strings. A world ID identifies a generated town. A timeline ID selects one history within it. Neither identity is derived from the fixture hash. Contract versions evolve independently of the SQLite schema version.

| Record | Required fields | Rules |
| --- | --- | --- |
| Action proposal | Contract version, request ID, world ID, timeline ID, action kind, arguments | The coordinator binds the actor from the caller's context. Validate the action kind and its argument schema before resolution. References supplied by a player must be valid for that player's observation. |
| Action result | Contract version, request ID, status | An accepted result carries an actor-scoped observation after commit. A rejected result carries a stable reason code and a player-safe explanation. Acceptance means the attempt resolved, including ordinary in-world failure. |
| Internal event | Contract version, world ID, timeline ID, event ID, timeline sequence, tick, owning engine, event kind, payload version, payload | Persist a causing request ID for command-driven events and parent event IDs for causal links. Sequence gives a total order within the timeline, including events at the same tick. |
| Actor observation | Contract version, observer ID, world ID, timeline ID, visible facts and usable references | The projection filters facts and action targets before delivery. It keeps hidden scores, event payloads and private NPC knowledge inside the simulation. |

Treat a repeated request ID with identical input in the same actor and timeline context as a retry. Return the recorded result of a committed request without consuming time or randomness again. Reject reuse with different input as `request_conflict`. Reject unknown contract versions as `unsupported_version` and malformed proposals as `invalid_request`. An unavailable target and a hidden target produce the same player-facing response.

For example, a player may still have a door selected when a new observation makes its reference unusable. The client's proposal could be encoded as

```json
{"version":1,"request_id":"r17","world_id":"w1","timeline_id":"t1","action":"open","args":{"target":"door-2"}}
```

The response gives the player enough information to choose a new action without revealing who moved or what changed beyond their perception.

```json
{"version":1,"request_id":"r17","status":"rejected","code":"target_unavailable","message":"You can't reach that target."}
```

## Resolution and persistence

1. At startup, each engine declares its stable ID, version, owned state and dependencies. The registry rejects duplicate owners, missing dependencies and cycles before advancing time.
2. The scheduler supplies an integer simulation tick and runs due work in dependency order. Break ties between ready engines by stable engine ID. Resolve causal parents before their children. Wall-clock timing and map iteration order must not determine outcomes.
3. Each engine draws from a named deterministic stream supplied by the simulation. Stage all resulting state changes, clock advances, stream positions and events together. Other engines request changes to the owning engine through its declared contract.
4. Ask the storage adapter to commit the staged changes, ordered events and request receipt atomically. Publish the new state and result only after success. On storage failure, discard the staged changes, retain the last committed clock and RNG state, and allow the caller to retry. Keep storage error details in developer diagnostics.
5. Restore a checkpoint with its world, timeline, clock, random streams, engine and content versions, and last event sequence. Reject incompatible versions before replay. A future branch reads its historical prefix and writes subsequent events under its own timeline ID.

Phase 0 demonstrates atomic writes, rollback and deterministic fixture hashes. The engine registry, actor projections, request receipts and event replay are contracts for later implementation.

## Device support

Desktop packages should include the runtime, interface and SQLite support each target needs. Keep saves in a writable per-user data directory when packaging the finished game. The developer runner currently writes in the working directory or the path passed with `-db`.

A browser client could extend the same rules to phones, tablets and Chromebooks through WebAssembly. It would need touch and keyboard input, durable local saves, and offline loading after the first visit. Test a full save, close and reopen on each supported browser before promising support. Native mobile packaging remains an option. The client technology and release targets remain open. [Go WebAssembly documentation](https://go.dev/wiki/WebAssembly)
