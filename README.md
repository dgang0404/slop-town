# Slop Town

Slop Town is a design for a text-first simulation game set in a persistent, procedurally generated town. Residents follow their own routines, form relationships, and remember what happens. A walk across town takes time, and someone the player meets along the way can interrupt the trip.

For tested devices, hardware targets and offline play, see the [player system requirements](docs/system-requirements.md).

## Design documents

- [Master Design and Architecture, revision 7](docs/design/Slop_Town_Master_Design_Revised_v7.docx) describes the town, its residents, player commands, and how the simulation resolves actions.
- [Phase 0 Development Foundation](docs/design/Slop_Town_Phase_0_Development_Foundation.docx) outlines the first development milestone, including the headless runner, persistence, and tests.

See the [Phase 0 plan](docs/plans/phase-0.md) for the Go foundation milestones.

The [Phase 0 acceptance report](docs/plans/phase-0-results.md) collects the foundation test evidence. The [Phase 1 plan](docs/plans/phase-1.md) covers time, events, saves and replay. Its [acceptance report](docs/plans/phase-1-results.md) gives the reproducible restart and rollback checks and the CI status.

Phase 2 follows a wooden spoon from a bakery crate to the street. The runner can reopen the town and replay that journey. See the [Phase 2 plan](docs/plans/phase-2.md) and [acceptance results](docs/plans/phase-2-results.md).

Phase 3 adds a playable text-command session, contextual actions and numbered choices for ambiguous names. Its acceptance test compares a typed walkthrough with explicit commands across restarts. [Phase 3 plan](docs/plans/phase-3.md) · [Acceptance results](docs/plans/phase-3-results.md)

In Phase 4 the baker and porter follow work routines, choose where to rest and keep private notebooks. The acceptance walkthrough follows a break the porter witnesses while the player is down the street. [Phase 4 plan](docs/plans/phase-4.md) · [Acceptance results](docs/plans/phase-4-results.md)

In Phase 5, the player and residents build endurance, feel relief after a tiring walk, and recognize people who have introduced themselves. [Phase 5 plan](docs/plans/phase-5.md) · [Acceptance results](docs/plans/phase-5-results.md)

The first Phase 7 checkpoint is playable in the bakery. Each turn describes what your character notices, with new details in bold on supported terminals. [Phase 7 presentation plan](docs/plans/phase-7.md)

To validate a local content pack, run `go run ./cmd/packcheck PACK_DIRECTORY`. The [manifest guide](docs/content-packs.md) includes a bundled example.

## Run the Phase 0 fixture

With Go 1.25 or newer, run these commands from the repository root. The first build needs access to the module dependencies.

```sh
go mod download
go test ./...
go build -o sloptown ./cmd/sloptown
./sloptown -seed 42
```

The runner writes a fixture to `slop-town-smoke.sqlite`, closes the database, reopens it, and checks the saved value. Repeat the command to verify the same hash. Use `-db PATH` to choose another file. Once built, the runner works offline.

Use `./sloptown -trace -seed 42` to print a JSON record to standard error after each successful step or when a step fails. Standard output carries the usual seed and hash on success. Exit codes are 0 for success or help, 1 for a save or verification failure, and 2 for invalid arguments.

## Try a persistent Phase 1 step

After building the runner, run the following commands against the same save. Each invocation opens and closes SQLite. The first two request IDs advance the timeline to tick 2. The third invocation returns the original receipt for `r1` and leaves the clock at tick 2.

```sh
./sloptown step -seed 42 -request r1
./sloptown step -seed 42 -request r2
./sloptown step -seed 42 -request r1
```

This example assumes a fresh timeline. The step command uses actor `player` in world `smoke`, timeline `main`. Use `-db PATH` to select another save. The seed initializes a new timeline. Later invocations use its saved clock and streams. Changing `-stream` on a committed request ID returns `request_conflict` and exits 1.

## Replay a timeline from a checkpoint

With the runner built, try a fresh save on Linux. The first step ends at event sequence 2. Bookmark it, take a second step, then replay from the bookmark:

```sh
save_dir=$(mktemp -d)
./sloptown step -db "$save_dir/town.sqlite" -seed 42 -request r1
./sloptown checkpoint -db "$save_dir/town.sqlite"
./sloptown step -db "$save_dir/town.sqlite" -request r2
./sloptown replay -db "$save_dir/town.sqlite" -from 2
```

Use the sequence printed by `checkpoint` when trying an existing timeline. Replay opens the save read-only, checks each subsequent event, and compares the reconstructed state with the committed head. The example ends at tick 2 and sequence 4 with SHA-256 `d508c6d2e9416be9d60c32bbc51cb36d2c2be7e13c146057d74dad10c525554d`.

## Look inside the bakery

To play one line at a time, start a session from the repository root.

```sh
go run ./cmd/sloptown bakery -play -seed 42
```

Try these commands when the street appears.

```text
What can I do here?
open door
go east
e
open the crate
actions
pick up the spoon
put it in the crate
```

Ask `What can I do here?` or type `actions` for commands you can use from where you stand. On the street, the list offers to open the bakery door. Beside the open crate, it offers to take the wooden spoon. Asking takes no game time.

You can use a short name such as `door` or `spoon`. If several visible objects match, the game lists them by name and location. Enter a number to choose one or type `cancel`. Asking and cancelling take no game time. In `put it in the crate`, `it` refers to the spoon you just picked up. The terminal remembers the primary object of your last successful interaction for the rest of that session.

Opening the crate gives you “You open the crate. Inside lies **a wooden spoon**.” If you wait there and nothing changes, you get “You wait.” Type `look` whenever you want the full description and grid. To turn off bold, start the game with `NO_COLOR=1`. Redirected output is always plain text.

Type `help` for the command forms or `quit` to leave. Add `-db bakery.sqlite` to resume the same town on your next visit. Each visit opens with the full current scene.

From the repository root, run the street-to-crate walkthrough in a fresh, in-memory session.

```sh
go run ./cmd/sloptown bakery -seed 42 open "bakery door" move east move east open crate
```

This command-line walkthrough prints the full player view after each action. On the grid, `@` marks the player, `?` a square out of sight, and `#` a wall. Opening the lid changes the crate's square from `C` to `c*` as the spoon comes into view. Omit `-db` for a fresh in-memory session.

### Browse actions and walk to a destination

Start `go run ./cmd/sloptown bakery -practice -play -seed 42`. In an interactive terminal, press **Up** or **Down** to browse actions and **Enter** to choose. Escape restores what you were typing. You can use the arrows and Enter for numbered choices too. Typed commands and piped scripts still work.

Try `walk to the porter` from the starting street. When the crate is visible along a clear route, `walk to the crate` brings you within reach so you can open it. If two objects match, choose one by location. You can still choose a compass direction for an individual step.

**Time passes one step at a time.** A three-square walk costs three ticks. Choosing `rest` or `wait` costs one tick each. Looking, browsing, cancelling, or asking to approach something already within reach costs none. Residents act after each step. If someone introduces themselves to you on the way, the walk pauses and gives you the turn. You can choose another action or select the destination again to continue.

To see a moving destination interrupt a walk, start a fresh seed-42 practice session. Wait three turns, step west, step north, then enter `walk to the baker`. You take one step before the baker slips out of sight. [Checkpoint results](docs/plans/phase-5.md#checkpoint-4-demo-controls)

### Meet the baker

```sh
go run ./cmd/sloptown bakery -resident -play -seed 42
```

Stay outside and type `wait` three times. On the first turn the baker opens the door from inside. On the second the baker steps into the doorway. On the third you see “You see the baker here.” If you open the door yourself, the baker can cross a turn sooner. Looking around and asking for available actions take no game time.

Keep waiting on the same spot. On turn 9 both workers reach the doorway. On turn 10 the porter sits down there while the baker walks back into the bakery. On turn 12 you can see the baker sitting at the back. The next turn the baker returns to prep. You can also type `rest` to sit for a turn. Your next step brings you back to your feet.

To save this visit, add `-db resident.sqlite` when starting. Use a fresh save or a new `-timeline NAME` alongside an existing bakery. Each timeline remembers the content it started with, so you can omit `-resident` when resuming.

In a new resident timeline, each worker has a seeded preference for where to take a break. To inspect the saved preferences after playing, run the read-only developer commands from the repository root.

```sh
go run ./cmd/sloptown profile -db resident.sqlite -actor bakery/baker
go run ./cmd/sloptown profile -db resident.sqlite -actor bakery/porter
```

### Try the endurance walk

Start a fresh seed-42 bakery with both workers.

```sh
go run ./cmd/sloptown bakery -practice -play -seed 42
```

Walk the following route three times, sitting down at the end of each circuit. Notice where you are when your legs start feeling tired.

```text
west
south
east
north
south
north
rest
```

When you sit after a tiring walk, you feel relieved. The feeling lingers for two more turns. To see how it looks from the outside, start a fresh practice bakery and wait at the shopfront. On turn 10 you can see the porter’s shoulders relax, and on turn 12 you can see the same change in the baker. Walk two steps south at the start and you’ll miss the baker’s break.

To keep the visit, add `-db practice.sqlite` when creating the timeline and use the same path when returning.

### Introduce yourself

In a fresh seed-42 `-practice -play` session, try these commands.

```text
south
actions
introduce yourself to the porter
wait
```

The porter remembers meeting you. On the next turn the porter introduces themselves to the baker at the shopfront. The baker then recognizes the porter. To experience an introduction as the listener, start another fresh session and wait four turns at the shopfront. Your scene tells you who spoke to you and who you recognize.

### Inspect the residents' notebooks

In a new resident bakery, the porter starts on the street. The porter and the baker can each record seeing someone sit down, open a door or crate, or close one. To compare their accounts, start a fresh save and sit down outside, then walk inside and sit down again. The developer command `notebook` reads one character's sightings from the save. Reading costs no game time.

```sh
save_dir=$(mktemp -d)
save="$save_dir/notebooks.sqlite"
go run ./cmd/sloptown bakery -resident -seed 42 -db "$save" rest move east move east rest
go run ./cmd/sloptown notebook -db "$save" -actor bakery/porter
go run ./cmd/sloptown notebook -db "$save" -actor bakery/baker
```

Only the porter saw the player sit on the street at tick 1. Both workers saw the player sit in the prep area at tick 4. Each entry gives the sequence number of the event the character witnessed. [Phase 4 checkpoint results](docs/plans/phase-4.md#checkpoint-4-results)

### Save and replay the bakery

Use a fresh temporary directory for the reference walkthrough. Each command runs in a new process.

```sh
save_dir=$(mktemp -d)
save="$save_dir/bakery.sqlite"
go run ./cmd/sloptown bakery -db "$save" -seed 42 -batch pickup open "bakery door" move east move east open crate take "wooden spoon"
go run ./cmd/sloptown checkpoint -db "$save" -world bakery
go run ./cmd/sloptown bakery -db "$save" -batch return close crate move west move west close "bakery door" drop "wooden spoon"
go run ./cmd/sloptown bakery -db "$save"
go run ./cmd/sloptown replay -db "$save" -world bakery -from 10
```

The checkpoint records tick 5 at event sequence 10. Reopening after the return trip shows the spoon on the street. Replay verifies the next ten events and the final state at tick 10. Reuse a batch ID with the same ordered actions to retry after an interruption. Each action commits separately. A new invocation generates a fresh batch ID unless you supply one. Use `-timeline NAME` on all three subcommands for a separate history.

In `-play` mode, resume a saved batch by repeating its commands from the beginning with the same `-db` and `-batch`. When a command originally needed a numbered choice, enter the same number again. The terminal acknowledges each completed action. You can then continue with a new command.

## Run the Linux CI checks locally

From the repository root, restore the pinned Go modules once, then run the same verification script used for pushes and pull requests.

```sh
go mod download
sh scripts/check-phase-0.sh
```

The script disables Go downloads, runs vet and the full test suite, builds both commands, and validates the sample pack. It runs seed 42 twice against the same temporary save and checks both outputs against the reference hash. It removes the temporary binaries and save on exit.

In the test output, `TestPhase1Acceptance` shows the runner reopening a separate temporary timeline, retrying a request and recovering from a failed receipt write. It logs the replay hash before and after rollback.

To run the Phase 3 restart check on its own, use `go test -count=1 -v -run '^TestPhase3Acceptance$' ./internal/storage`.

## License

The Go source code is licensed under the [GNU General Public License, version 3 or later](LICENSE-CODE).

The Slop Town documentation is licensed under [Creative Commons Attribution-ShareAlike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/). You may share and adapt it, including commercially. When sharing, credit the Slop Town contributors, link to the license, and indicate any changes. Share adaptations under the same license or a compatible license as described in the [full terms](LICENSE).
