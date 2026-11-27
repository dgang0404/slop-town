# Phase 6 — economy, plants, animals and environment

The master design calls for plants that grow with town time, including when nobody is looking. This phase starts with watering an individual plant and observing it again later. Harvesting, animal interactions and trading produce are later parts of the roadmap. [Master design, sections 29 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx)

## Checkpoint 1 — water the bakery mint

New practice towns have a mint planter with a watering spout on the street. Any character who can see and reach it may spend one turn watering the dry soil. An already-formed pair of leaves unfolds four to six turns later. The interval depends on the town seed and the individual plant. This first fixture covers one watering and one change in growth stage. [Play the walkthrough](../../README.md#water-the-mint-outside-the-bakery)

The dry and damp planter uses `y` on the grid. New leaves use `Y`. Text and examination also describe the condition. A player standing around the corner can still read their last sighting. When they see the planter again, the journal records the new leaves on that visit.

## Checkpoint 1 results

On seed 42, the player watered the planter on turn one. The leaves unfolded on turn seven with the player inside the bakery, out of sight of the street. The last-sighting entry still described damp soil. On turn eight the player stepped back into view, and the journal recorded “I saw new leaves on the mint planter.” In a separate visit the player stayed outside and saw the change on turn seven.

Across three seeds, waiting an entire day beside a dry planter produced no leaves. The baker could water from beside the planter. Trying to water the damp soil again spent no time. The simulation recorded one growth event with a reference to the original watering. [Simulation tests](../../internal/simulation/plant_test.go)

Interrupting the SQLite receipt write on the watering turn and again on the growth turn preserved the plant, workers, clock and history. After retry, replay from sequences 0, 4, 23 and 26 reached tick 8, sequence 28, with SHA-256 `452ef173bcca2e2f17d1d2826cb3deebe31ee5a653c74c16aca8b67b0994bce1`. [Restart and replay test](../../internal/storage/plant_test.go)

Fresh practice saves use content version 21. A version 20 save from `6e79493` kept its bakery layout, action menu and journal. Both executables replayed the seed-42 spoon examination from sequences 0 and 19 to tick 9, sequence 31, with SHA-256 `40555b8b5b065d2a04882a46bfa257fc2115557df88a83e07f565503863af859`.

Down and Enter completed the whole visit in a Linux terminal. The offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Next checkpoint — harvest a sprig

Let a character within reach harvest a sprig from the leafy plant. The harvested sprig should enter that character’s inventory as a physical item, and the planter should show what remains. Verify that retrying a harvest across a restart produces one sprig.
