# Use Go for the simulation

Accepted 25 November 2026. The first executable is a terminal runner written in Go. SQLite stores local state. The simulation must run offline after dependencies are restored, independently of a renderer or language model.

Go is already available on the development laptop. A terminal runner lets contributors test deterministic state and persistence before building a graphical interface. This choice defers Godot and removes the .NET SDK and Godot C# build from Phase 0. A later graphical client will need an explicit command and observation interface to the simulation.
