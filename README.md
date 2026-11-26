# Slop Town

Slop Town is a design for a text-first simulation game set in a persistent, procedurally generated town. Residents follow their own routines, form relationships, and remember what happens. A walk across town takes time, and someone the player meets along the way can interrupt the trip.

For tested devices, hardware targets and offline play, see the [player system requirements](docs/system-requirements.md).

## Design documents

- [Master Design and Architecture, revision 7](docs/design/Slop_Town_Master_Design_Revised_v7.docx) describes the town, its residents, player commands, and how the simulation resolves actions.
- [Phase 0 Development Foundation](docs/design/Slop_Town_Phase_0_Development_Foundation.docx) outlines the first development milestone, including the headless runner, persistence, and tests.

See the [Phase 0 plan](docs/plans/phase-0.md) for the Go foundation milestones.

The [Phase 0 acceptance report](docs/plans/phase-0-results.md) collects the foundation test evidence. The [Phase 1 plan](docs/plans/phase-1.md) covers time, events, saves and replay. Its [acceptance report](docs/plans/phase-1-results.md) gives the reproducible restart and rollback checks and the CI status.

The Phase 2 tests walk a character into an in-memory bakery, open a crate, and carry a wooden spoon back outside. Run `go test -v -count=1 -run '^TestBakery' ./internal/simulation` to exercise both seeded layouts and the inventory rules. See the [Phase 2 plan](docs/plans/phase-2.md) for the walkthrough and checkpoint results.

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

## Run the Linux CI checks locally

From the repository root, restore the pinned Go modules once, then run the same verification script used for pushes and pull requests.

```sh
go mod download
sh scripts/check-phase-0.sh
```

The script disables Go downloads, runs vet and the full test suite, builds both commands, and validates the sample pack. It runs seed 42 twice against the same temporary save and checks both outputs against the reference hash. It removes the temporary binaries and save on exit.

In the test output, `TestPhase1Acceptance` shows the runner reopening a separate temporary timeline, retrying a request and recovering from a failed receipt write. It logs the replay hash before and after rollback.

## License

The Go source code is licensed under the [GNU General Public License, version 3 or later](LICENSE-CODE).

The Slop Town documentation is licensed under [Creative Commons Attribution-ShareAlike 4.0 International](https://creativecommons.org/licenses/by-sa/4.0/). You may share and adapt it, including commercially. When sharing, credit the Slop Town contributors, link to the license, and indicate any changes. Share adaptations under the same license or a compatible license as described in the [full terms](LICENSE).
