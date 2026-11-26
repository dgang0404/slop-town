# Working on Slop Town

Read the [master design](docs/design/Slop_Town_Master_Design_Revised_v7.docx) and [current Phase 2 plan](docs/plans/phase-2.md) before coding. The [Go decision](docs/decisions/0001-go-simulation.md) governs the implementation stack. Phase 2 covers the seeded bakery, movement, containers, actor observations and replay of spatial actions.

Keep simulation rules in `internal/simulation`. The simulation owns world state and must run offline. SQLite belongs in `internal/storage`. The terminal composes the two packages. Future UI and AI adapters submit proposals for the simulation to validate. Record accepted AI proposals as events.

Apply the same rules to players and NPCs. Allow missed encounters and independent relationships. Avoid forced plots, friendships and automatic progression feedback. Player-facing views consume only observations available to the character. Hidden mechanics, character statistics and world-wide diagnostics belong in developer tools.

Keep changes small and identify which package owns each behavior. Run `go test ./...` and the README smoke command when changing the fixture or persistence. Record checkpoint results and blockers in the phase plan. Discuss requirement changes or paid dependencies with the owner first.
