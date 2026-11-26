# Phase 5 — hidden progression, emotion and relationships

The player and residents develop through what they do and experience. The simulation owns changes to capability, temporary emotion and directional relationships. Player views can show effort, expression and behavior. The developer history records the causes. This phase builds on the saved routines, notebooks and actor-specific observations from Phase 4. [Master design, sections 9–10, 13, 46 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx) · [Phase 4 acceptance](phase-4-results.md)

| Checkpoint | Behavior to verify |
| --- | --- |
| 1. Endurance through practice | A demanding walk followed by recovery contributes to a small lasting improvement. Player and resident movement use the same rule. |
| 2. Temporary emotion | A personally experienced event changes an actor's temporary emotional state. Its effects subside with time while the actor's baseline personality persists. |
| 3. Directional relationships | A concrete interaction can change how one participant regards the other. Test both directions and a resident who missed it. |
| 4. Acceptance | Follow the consequences through ordinary play, an interrupted save, restart and replay. Compare character views, developer history and local and hosted results. |

## Checkpoint 1 results

Checkpoint 1 introduced practice bakeries with both Phase 4 workers in content version 6. The seed-42 player walked west, south, east, north, south and north, then sat down. The first two walks each took the player to four distinct destinations and left them tired. On the third walk the player felt tired on step six. On the first walk that happened on step five. The baker and porter also earned practice during their ordinary work routines. [Run the walkthrough](../../README.md#try-the-endurance-walk) · [Simulation checks](../../internal/simulation/endurance_test.go)

For this prototype a recovery counts after at least five steps and four distinct destinations since the last rest. The simulation marks the qualifying rest in the developer event history. After two qualifying recoveries, the first step from sitting costs no fatigue. This bakery supports one endurance improvement. Pacing back and forth between two cells and repeatedly sitting down both earned zero additional practice. Existing saves retain their original rules.

Failing the SQLite receipt write on the second recovery left the actor state and history intact. Retrying awarded the recovery once. Replay from sequence 0 and from sequence 46, immediately before that rest, reached tick 14, sequence 49, with SHA-256 `c972f354662c8587789538378f74ad13052dbb799c741fd833667c7d362a05f7`. [Restart and retry test](../../internal/storage/endurance_test.go)

Repeat the focused checks with

```sh
go test -count=1 -v -run 'WalkingPractice|EnduranceAcrossRestart' ./internal/simulation ./internal/storage
```

On 26 November 2026 the full offline Linux check passed with Go 1.27.4, including both seed-42 smoke runs. The terminal, simulation and storage packages passed three race-detector runs. Hosted Linux verification passed at `b233190`. [Hosted run](https://github.com/dgang0404/slop-town/actions/runs/45853342478) [Offline verification script](../../scripts/check-phase-0.sh)

## Checkpoint 2 results

A fresh `-practice` bakery now uses content version 7. Sitting down after reaching the existing fatigue threshold brings the acting character relief. The developer history marks the rest that caused it. The character feels relieved on that turn and feels a little relief on the following two turns. Someone who can see the character on the first turn can notice relaxed shoulders. Looking, retrying a command and reopening the save spend no time. Sitting again while rested spends a turn and lets the feeling fade. The baker and porter keep their original break preferences throughout. [Simulation checks](../../internal/simulation/emotion_test.go)

On seed 42, the porter rested in the doorway on tick 10 and the baker rested at the back on tick 12. A player waiting at the shopfront could see both expressions. A player who walked two steps south missed the baker’s expression. The workers experienced the same relief in both walks. The player could also feel relief after pacing until tired, even though that route earned no endurance practice. [Terminal walkthrough](../../README.md#try-the-endurance-walk)

A failed SQLite receipt write on tick 12 rolled back the baker’s reaction and the fading of the porter’s earlier feeling. Retrying recorded one trigger per break. Both feelings had passed by tick 15. Replay from sequences 0, 35 and 42 reproduced sequence 51 with SHA-256 `4987e8bb6acaef449feeeb317673a2d3aab4f887b4b2c978db8978fc382d6112`. A version 6 save from the previous executable reopened at its published tick-14 digest. Both executables continued that save to the same tick-18 digest. [SQLite restart and rollback test](../../internal/storage/emotion_test.go)

The full offline Linux check and three race-detector runs passed. Hosted verification of checkpoint 2 is pending its push. Repeat the focused checks with

```sh
go test -count=1 -v -run Relief ./internal/simulation ./internal/storage
```
