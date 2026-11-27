# Phase 6 — economy, plants, animals and environment

The master design calls for plants that grow with town time, including when nobody is looking. This phase starts with watering an individual plant and observing it again later. The first two checkpoints cover growth and harvesting. Animal interactions and trading produce come later. [Master design, sections 29 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx)

## Town settings and population over time

The following controls are planned for **New Game**. Show the resulting starting resident count and town dimensions before the player confirms. Exact preset counts and hardware limits need testing.

| Setting | Choices | What changes |
| --- | --- | --- |
| Town size | Small, medium, large | The area available for streets, lots and interiors. |
| Starting population | Few, standard, many | How many people move in initially. Offer combinations the town can house and the device can simulate. |
| Plant density | Sparse, standard, lush | How densely suitable ground starts planted. Reserve paths, entrances and working space at every density. |
| Town theme | Wholesome Whimsy foundation, plus compatible optional packs | The foundation supplies everyday residents, food and shops. A foodie pack could favor growers, kitchens and restaurant work. Horror or other themed packs can give the same underlying town mechanics a different emphasis. |

Save these creation settings with the seed and pack versions. **During play**, allow changes to narration detail, illustration display, accessibility and content presentation. Changing how much the player sees must preserve the people, plants and history already in the save. Gardening, arrivals and departures can change the living town through play.

**Residents.** Include women, men and nonbinary people in the population, including trans residents. Let players choose their own character’s gender and pronouns. A trans woman can simply be the town’s baker. Her gender tells the simulation nothing about her baking skill, fertility or desire to have children.

**Long-running towns — exploratory.** A later life-cycle option could support households having children, residents aging, and people moving into or out of town. Let spare housing and sustainable population limits shape future arrivals and household growth. A full town need not evict someone when a child is born. Keep departed residents in family histories and journals. Before adopting this feature, simulate several generations in a small town and measure population, save growth and turn time. Check that children can grow up, that shops can change hands and that old letters still identify their writers.

For old age in the base game, explore voluntary retirement and continuing as another resident. The player’s journal and connections should survive that handover. Timing and succession remain open design questions.

## Base-game food and community realism mods

Wholesome Whimsy uses Cute/Kawaii rules by default. Healthy livestock lay special food eggs. A cow’s egg yields hamburgers, a chicken’s yields chicken nuggets, and a pig’s yields bacon. Farmers collect the eggs and sell them to the butcher, who prepares the food and fills restaurant orders. The animals stay alive and unharmed. These food eggs are distinct from eggs that can hatch.

Realistic character death, sexual intimacy between consenting adult characters, and traditional livestock slaughter and butchery belong exclusively to optional community mods. “Realism Mode” describes a modded world. The base-game plan uses the Cute/Kawaii supply chain. Retirement remains an option to explore for aging characters. This decision updates the earlier content and life-cycle proposals. [Content packs](../content-packs.md)

## Planned scenario — a restaurant supplied by the town

A restaurant can be common across generated towns where there are enough customers, workers and suppliers to sustain one. Give each restaurant a menu its cooks can make with the equipment on hand and ingredients they can obtain. A riverside town and a farming village may serve different lunches. Some towns can have a smaller menu or no restaurant at all. The frequency is open for playtesting.

Bread for lunch might come from the bakery, meat from the butcher and vegetables directly from a farmer. The farmer and butcher arrange a price and collection time for food eggs. The butcher then needs time to prepare the restaurant’s order. A grower might sell culinary herbs directly to the cook or through a local apothecary that stocks them. Which relationships exist depends on the town. A purchase changes the seller’s and buyer’s stock and money together. Cooking consumes ingredients, selling the meal brings in revenue, and paying the server comes out of the business’s funds. These trades use the town’s existing economy design. [Master design](../design/Slop_Town_Master_Design_Revised_v7.docx)

**Listening during service.** Listening is a valuable professional skill for a server. Between two tables, a practiced, rested server might catch “bread for two” from one group and a correction to the other group’s order. If the kitchen door shuts, the room grows louder or a difficult exchange demands their attention, they may catch only a fragment. They can return to the table and ask. The kitchen gets the order the server actually communicates. A player working the shift and a resident server use the same rules. [Learnable skills](phase-5.md#planned-mechanic--professions-and-learnable-skills) · [Conversation and attention](phase-7.md#planned-mechanic--group-conversation-and-attention)

**Future balance test.** Follow one lunch service from the morning deliveries through the server’s break. Check that the farmer has time to tend crops and meet the butcher, the cook can prepare the advertised lunch before customers arrive, and another worker can cover a break or let customers wait. Repeat with a delayed food-egg delivery, a vegetable out of season and a tired server in a noisy room. Observe what the cook can still offer, which orders need clarification, how long diners wait and whether the shift leaves time to recover. Trace a sold meal back to its ingredients and payments. Reopen the save mid-service and verify that the outstanding orders, remaining stock, balances and workers’ schedules carry through once. Use those results to tune travel, work and recovery times.

## Checkpoint 1 — water the bakery mint

New practice towns have a mint planter with a watering spout on the street. Any character who can see and reach it may spend one turn watering the dry soil. An already-formed pair of leaves unfolds four to six turns later. The interval depends on the town seed and the individual plant. This first fixture covers one watering and one change in growth stage. [Play the walkthrough](../../README.md#water-the-mint-outside-the-bakery)

The dry and damp planter uses `y` on the grid. New leaves use `Y`. Text and examination also describe the condition. A player standing around the corner can still read their last sighting. When they see the planter again, the journal records the new leaves on that visit.

## Checkpoint 1 results

On seed 42, the player watered the planter on turn one. The leaves unfolded on turn seven with the player inside the bakery, out of sight of the street. The last-sighting entry still described damp soil. On turn eight the player stepped back into view, and the journal recorded “I saw new leaves on the mint planter.” In a separate visit the player stayed outside and saw the change on turn seven.

Across three seeds, waiting an entire day beside a dry planter produced no leaves. The baker could water from beside the planter. Trying to water the damp soil again spent no time. The simulation recorded one growth event with a reference to the original watering. [Simulation tests](../../internal/simulation/plant_test.go)

Interrupting the SQLite receipt write on the watering turn and again on the growth turn preserved the plant, workers, clock and history. After retry, replay from sequences 0, 4, 23 and 26 reached tick 8, sequence 28, with SHA-256 `452ef173bcca2e2f17d1d2826cb3deebe31ee5a653c74c16aca8b67b0994bce1`. [Restart and replay test](../../internal/storage/plant_test.go)

Fresh practice saves use content version 21. A version 20 save from `6e79493` kept its bakery layout, action menu and journal. Both executables replayed the seed-42 spoon examination from sequences 0 and 19 to tick 9, sequence 31, with SHA-256 `40555b8b5b065d2a04882a46bfa257fc2115557df88a83e07f565503863af859`.

Down and Enter completed the whole visit in a Linux terminal. The offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 2 — harvest a sprig

Fresh practice towns use content version 22. When new leaves appear, a character beside the planter can spend one turn harvesting a mint sprig. It goes into that character’s hand and the planter shows clipped stems. This plant yields one cutting. A witness who sees the gardener and the planter can describe the harvest. [Try the harvest](../../README.md#harvest-a-sprig-and-carry-it-inside)

On seed 42 the player harvested on turn eight, examined the sprig, dropped it, picked it back up and stored it beside the wooden spoon. The baker could also harvest when within reach. A witness saw the clipped stems and the act of harvesting. Someone around the corner missed it. [Simulation checks](../../internal/simulation/harvest_test.go)

Interrupting the SQLite receipt write left the leafy plant intact. Retrying after restart created one sprig. Repeating the entire visit left that sprig in the crate. Replay from sequences 0, 28 and 30 reached turn 15, sequence 57, with SHA-256 `9882ff224e51a749deb959513aa2e6ea8a226c26e22c02a5906e05c677290568`. Both executables replayed a version 21 mint visit to its checkpoint 1 hash. [SQLite restart test](../../internal/storage/harvest_test.go)

The offline suite and both seed-42 smoke runs passed. Down and Enter completed the harvest, examination, drop and journal visit in a Linux terminal. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 3 — a second cutting

Fresh practice towns use content version 29. After the first harvest, clipped mint grows another set of leaves over town time without watering again. The regrowth takes four to six turns, seeded separately from the first growth. A second harvest creates `bakery/mint/sprig-2` even while someone still holds the first sprig. Only clipped stems remain afterward.

### Checkpoint 3 results

On seed 42 the player harvested on turn eight, waited through the regrowth interval, and harvested again while still carrying the first sprig. A third attempt reported clipped stems only. Version 22 saves keep the original one-cutting rule.

Interrupting the SQLite receipt write on the second harvest rolled back the clipping and the new sprig together. Retrying awarded one second cutting. [Simulation checks](../../internal/simulation/regrowth_test.go) · [Restart and replay](../../internal/storage/regrowth_test.go)

The offline suite and both seed-42 smoke runs passed. Down and Enter completed the visit in a Linux terminal. The CLI cross-built for Windows amd64 and macOS arm64.
