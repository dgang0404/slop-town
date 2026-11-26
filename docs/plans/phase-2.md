# Phase 2 proposal — a street, a bakery and a closed crate

**Planning handoff, 25 November 2026.** A player starts outside a bakery. They open the door, walk inside, find a closed crate, take a wooden spoon from it and carry it outside. After quitting and reopening the town, the spoon is exactly where they left it. This is a proposed headless fixture for procedural space, movement, containers and visibility. The fixture size and action durations are test values. [Master design, sections 6, 7, 47 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 1 acceptance](phase-1-results.md)

## First checkpoint — generate a place

Use one floor with five columns and four rows. Coordinates start at the northwest corner. East increases `x`; south increases `y`. Columns 0–1 are the street, column 2 is the bakery wall, and columns 3–4 are the bakery interior. One wall cell contains a closed door. Opening it makes that cell passable and transparent.

Generate the doorway row `d = 1 + (draw & 1)` from the first draw on the existing `geography` stream. Put the player at `(1,d)`, the door at `(2,d)`, and a closed, fixed crate at `(3,d-1)` with a portable spoon inside. Generation consumes one draw and starts the town at tick 0. Record the seed, generator version and content version so reopening an existing timeline uses the saved layout and stream position.

Seed 42 places the door in row 1. Seed 43 places it in row 2. The map below is the developer's expected **initial** layout for seed 42. The crate is hidden from the player outside.

```text
       x  0 1 2 3 4
y 0       . . # C .
  1       . P D . .
  2       . . # . .
  3       . . # . .

. floor   # wall   P player   D closed door   C closed crate
```

Give spaces and objects stable, world-scoped IDs at generation. A cell is addressed by floor and coordinates. The spoon keeps its ID when carried or dropped. Its placement has exactly one parent at a time: a cell, an actor inventory or a container. Moving its container or carrier changes its effective position. Validate references and reject duplicate membership, cycles and out-of-bounds positions.

**First implementation deliverable:** an in-memory generator in `internal/simulation` and tests for both reference seeds. Generate each twice, compare the ordered cells, IDs, placements and stream counters, and verify that draws on an unrelated stream cannot change the layout.

## Walkthrough to verify

| Tick after action | Seed-42 action | Expected result |
| --- | --- | --- |
| 0 | Observe from the street | The closed door is visible. The crate and spoon are absent from the description, grid and usable target references. Observation costs no time. |
| 1–3 | Open the door, then move east twice | The player crosses `(2,1)` and reaches `(3,1)`. The crate is immediately visible to the north. Its contents are concealed. |
| 4–5 | Open the crate, then take the spoon | The spoon becomes a visible target when the lid opens. Taking it removes it from the crate and puts the same object in the player's inventory. Save a checkpoint here. |
| 6–9 | Close the crate, move west twice, then close the door | The player is back at `(1,1)`, carrying the spoon. The closed door blocks the interior from current observation. |
| 10 | Drop the spoon on the street | The spoon is on `(1,1)`. The crate is empty and closed. Reopen the save and replay from tick 5. Both paths must yield the same placements, door and lid states, actor observation, event order and canonical state hash as the uninterrupted run. |

## Implementation order and tests

| Checkpoint | Deliverable | Pass condition |
| --- | --- | --- |
| 1. Seeded layout and IDs | Generate the fixture above in memory. | Both reference layouts match, all references resolve, and the spoon has one placement. |
| 2. Movement and reach | Route typed spatial proposals through the coordinator. A cardinal move visits one passable cell. Opening and closing require the actor to be on or cardinally adjacent to the target. | Each successful action in the walkthrough costs one tick. Walls, closed doors and map boundaries reject movement. Closing an occupied doorway fails. Rejected requests leave state and time intact; an identical committed retry returns its receipt. |
| 3. Containers and inventory | Support opening, closing, taking, putting into a reachable open crate and dropping onto the current cell. | Verify the walkthrough and a separate take-and-put-back round trip. Attempts to take a fixed crate, access a closed crate or transfer a distant item preserve all placements. |
| 4. Actor observations | Use one actor-scoped projection for terminal text, the text grid and target selection. In this lit fixture, trace sight across cells. Walls and closed doors block sight beyond them; a closed lid conceals its contents. Block diagonal sight through touching opaque corners. | Ordinary visible objects appear automatically. Hidden and nonexistent targets produce the same player-facing failure. After returning outside, previously seen interior objects are absent from the current view. Inspect internal positions and event IDs only through developer diagnostics. |
| 5. Persistence, replay and handoff | Stage spatial mutations alongside time, stream positions, events and receipts in one SQLite transaction. Version the spatial state and its canonical encoding, with stable ordering by entity ID and cell coordinates. | Run the walkthrough across CLI process restarts in Linux CI. Inject a failure during the item transfer, verify the original placement survives, then retry exactly once. Verify separate timelines can diverge from the same initial layout. Keep the Phase 0 fixture and Phase 1 saves, replay contracts and reference hashes passing. Record actual results and the CI run. |

Dependencies follow table order. Checkpoints 2 and 3 can use developer state inspection until the actor projection lands. The CLI supplies explicit actions in this phase. Free-text parsing belongs to Phase 3, and resident routines and conversations have later milestones. Review the fixture and its acceptance conditions before implementing checkpoint 1.
