# Phase 7 bakery acceptance

Two seed-42 visits began with the same four commands. The player rested, walked east twice and stepped south around the corner. The porter saw the player sit down and told the baker while the player was away. Both saves reopened at tick 4, sequence 15.

| After reopening | What reached the player | Final replay SHA-256 |
| --- | --- | --- |
| Return, ask the baker for news, check the source with the porter and open the crate | The accounts named both speakers. Both journal editions recorded their words and the spoon discovery. The illustrated edition pictured the spoon at that passage. | `953fb4adf607fc70a5d954fff855f483dbcb7f3d28a6797dfe102e981633cbb7` |
| Stay around the corner and wait five turns | The accounts stayed empty. The journal remembered waiting. The workers’ exchange and the hidden spoon stayed private in both editions. | `79e052d0bd534d966c00d7d8b0f9d29e81225693db367499284cfda3db0ed72c` |

Both visits reached tick 9, sequence 31. Replaying each visit from the beginning and from the corner reproduced its final hash after opening the accounts and exporting the two editions. Reissuing the first four commands after restart acknowledged all four. The earlier interrupted-witness check also covers rollback and retry. [Acceptance walkthrough](../../internal/storage/phase7_acceptance_test.go) · [Interrupted witness test](../../internal/storage/report_test.go)

The original spoon-demo save also produced the revised opening under “Inside the bakery” on export. Its first paragraph reads “I opened the crate. I saw the wooden spoon in the crate.” The three object sketches still accompany their discoveries. Up and Enter opened the same chapter in a Linux terminal.

The executable from `7cb27dc` and the revised build reproduced both visits’ hashes from sequences 0 and 15. The offline build, full test suite, both seed-42 smoke runs and three race-detector runs passed. The CLI also cross-built for Windows amd64 and macOS arm64.

To repeat the acceptance walkthrough from the repository root, run

```sh
go test -mod=readonly -count=1 -v -run '^TestPhase7Acceptance$' ./internal/storage
```
