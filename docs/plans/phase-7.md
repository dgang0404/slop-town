# Phase 7 — emergent narrative and deterministic dialogue

Section 17 of the master design calls for a consistent narrator, adjustable narration length, and a chronicle of what the player saw, heard, and did. Sections 38 and 47 ground that writing in the events and observations available to the character. Phase 7 is the roadmap home for this work. The first presentation checkpoint runs in the Phase 5 bakery. [Master design, revision 7](../design/Slop_Town_Master_Design_Revised_v7.docx)

## Checkpoint 1 — readable terminal turns

Open a session with the full scene so the player can get their bearings. After each command, give the action’s outcome, then describe what changed for the player. **Bold the new or changed detail** where the terminal supports it. Unchanged surroundings can drop out of the prose. Keep `look` available for the full description and grid at any time.

Suppose the player has already seen the crate. Opening it reveals what is inside.

> You open the crate. Inside lies **a wooden spoon**.

On the following quiet turn, “You wait.” is enough. If the porter approaches and speaks during that wait, give the introduction its own sentence. If the porter slips out of view, “You lose sight of **the porter**” says what the player knows.

The terminal should compare the previous and current player observations and include any interaction the player witnessed between them. Identify people and objects by their stable references so two wooden spoons with the same name remain distinct. During a named walk, combine uneventful steps into a short passage. An interruption ends the passage immediately and returns the prompt. Keep the grid available as a view of the current scene, with a fresh copy in the transcript when the player asks for it.

### Acceptance walkthrough

| In the bakery | What the reader should get |
| --- | --- |
| Open the crate, then wait nearby. | The opening and newly visible spoon appear once. A quiet wait gets a brief acknowledgement. |
| Walk toward a visible destination and receive an introduction en route. | The passage stops at the encounter. The speaker and what the player heard are clear before the next choice. |
| Lose sight of a resident, then use `look`. | The turn describes the loss of sight. The full scene shows what is visible now. The resident’s unseen destination stays private. |
| Browse actions, cancel, retry an acknowledged command, or reopen a save. | Browsing and cancelling spend no turns. A retry is acknowledged once. Reopening establishes the current scene as the starting point for later changes. |
| Read the same walkthrough in a terminal with bold disabled or redirect it to a file. | The sentences still convey every change. Plain output contains no terminal styling codes. Both runs produce the same simulation replay hash. |

### Checkpoint 1 results

On 26 November 2026, a seed-42 bakery session opened the door, walked east twice, opened the crate, waited, and used `look`. The spoon appeared in the opening sentence. The quiet wait took one line. The grid appeared twice, on arrival and on request. Reopening the save showed the spoon in the full scene. The following wait had no reason to mention it again. [Try the demo](../../README.md#look-inside-the-bakery)

The same commands ran with terminal bold, with `NO_COLOR=1`, with `TERM=dumb`, and with output redirected. All four transcripts had identical wording. Replay from sequence zero reached tick 5, sequence 10, with SHA-256 `2740f2ebd8f991dd568b3398c274d18da59cbfa525efa5c0e15070a152afca7e` in each save. Down and Enter also selected and opened the door in an interactive Linux terminal.

A three-step walk to the crate mentioned the approach once and arrived on the same tick as the three directional moves. A worker’s introduction on the second step of a separate walk paused the passage at tick 4. A player who lost sight of the baker learned only that the baker had passed out of view. A test with two spoons distinguished them by their visible locations. The offline build, test suite, both fixture smoke runs, and three race-detector runs passed. [Transcript tests](../../cmd/sloptown/narration_test.go) · [Interrupted walk and replay test](../../cmd/sloptown/walk_test.go)

## Checkpoint 2 — adjustable narration

The bakery starts with brief turn descriptions. Choose `narration story` at the prompt for fuller passages, or launch the demo with `go run ./cmd/sloptown bakery -play -narration story`. The arrow menu offers the other style. The choice applies to this session, including when playing from a SQLite save. [Play the bakery](../../README.md#look-inside-the-bakery)

| Seed-42 command | Brief | Story |
| --- | --- | --- |
| Step east from the bakery doorway | You enter **the bakery**. | You enter **the bakery**. The closed crate is **beside you now**. |
| Open the crate | You open the crate. Inside lies **a wooden spoon**. | You open the crate and find **a wooden spoon** inside. |
| Take the spoon | You **take** the wooden spoon. | You **pick up** the wooden spoon you saw in the crate. |
| Wait when nothing else changes | You wait. | You wait. |

The story templates use the player’s current and previous observations. They can recall where the player saw an object, or describe two residents disappearing together from the same part of the street. A spoken introduction still interrupts a walk. Asking about the setting or changing it during a numbered choice preserves that choice.

### Checkpoint 2 results

A seed-42 visit opened the door, stepped east twice, opened the crate, took the spoon, put it back, waited, and used `look`. Brief, story and a mid-visit switch produced identical receipts for all seven actions. Retrying each visit from the beginning acknowledged those seven actions. Replay from sequence zero reached tick 7, sequence 14, with SHA-256 `600f08f071d98edc8191e6aaacd48ede82254884a5c491c723b6490efe985d17`.

In an interactive Linux terminal, selecting `narration story` with the arrows and Enter left the opening save at tick zero. The walkthrough then showed the new spoon in bold and reached the same replay hash as both plain-text runs. The full offline build and test suite, both seed-42 smoke runs, and three race-detector runs passed. [Walkthrough and interruption tests](../../cmd/sloptown/narration_test.go)

## Checkpoint 3 — the player journal

`journal` opens a first-person account of the selected bakery timeline. Each committed player turn can contribute an action, a change of place or feeling, or a description of a person or object the player could see. The description preserves where the observer saw the object at the time. The journal also records personally witnessed interactions and introductions addressed to the player, including on another actor’s turn. A run of waits can share a sentence. A named walk records its approach once and keeps its arrival or interruption. [Read your journal in the demo](../../README.md#read-your-journal)

The simulation assembles the account from the player’s accepted receipts, addressed introductions and firsthand notebook sightings. The terminal writes the prose. SQLite reads one committed snapshot when the journal opens. The same projection serves in-memory play and existing bakery saves. Looking, browsing, reading, cancelling and rejected commands add no entries.

### Checkpoint 3 results

A seed-42 save created by the checkpoint 2 executable at `3ac9947` yielded the original seven-action visit. The journal remembered seeing the wooden spoon in the crate, picking it up and putting it back. Reading twice and retrying the visit preserved the entries. The old and new executables replayed to tick 7, sequence 14, with SHA-256 `600f08f071d98edc8191e6aaacd48ede82254884a5c491c723b6490efe985d17`. A failed receipt write added no entry. Opening a separate timeline in the same file produced an empty journal.

In the practice bakery, the player who waited at the shopfront remembered both workers’ greetings. The player who walked down the street missed those greetings and the baker’s break. A separate visit verified that hearing the porter introduce themselves twice produced two memories, even though the player learned to recognize the porter only once. Up and Enter opened the journal in a Linux terminal without advancing the save. The offline build, test suite, both seed-42 smoke runs and three race-detector runs passed. [Journal walkthroughs](../../cmd/sloptown/journal_test.go)

## Checkpoint 4 — chapters and plain-text export

The bakery clock counts turns. This prototype uses significant firsthand events for chapter boundaries. The first recorded visit opens the book. The first observed turn inside the bakery and a first personal introduction can each start a chapter. A player can introduce themselves or hear the other person speak first. A later greeting between the same two people stays in the account within the current chapter. When several introductions happen on the same turn, they share a heading. A meeting on the first turn inside also gets a single heading.

The journal on screen, the in-game `export journal` command and the standalone `sloptown journal -db PATH` command use the same plain-text rendering. The in-game command creates a new file for each copy. The standalone command reads the selected timeline from an existing save and writes the story to standard output. Both use the player’s journal projection. [Play and export a visit](../../README.md#read-your-journal)

### Checkpoint 4 results

Four waits at the seed-42 shopfront produced “The visit begins” followed by “Meeting the baker and the porter.” The player who went down the street missed the greetings and kept one chapter. On another visit, the player introduced themselves to the porter twice. The journal recorded both greetings under one heading. Returning through the bakery doorway also preserved the earlier chapter breaks. [Chapter and export tests](../../cmd/sloptown/journal_export_test.go)

A save created with the previous executable at `42c51f3` produced byte-identical text through shell export, Up and Enter in a Linux terminal, and a second export after retrying the original visit. Both executables replayed to tick 4, sequence 16, with SHA-256 `dad6bae76d36397648b43387efa41ff743efd54a3b42cfcc05eaacc6c223fe97`. A save deliberately named `slop-town-journal.txt` kept its replay hash while two exports took the next available names. The full offline build and test suite, both seed-42 smoke runs and three race-detector runs passed.

The CLI also cross-built for Windows amd64 and macOS arm64. Interactive export was exercised on Linux.

## Checkpoint 5 — conversations in the practice bakery

Use `talk to NAME` within reach of a worker. The menu offers up to eleven distinct lines drawn from the speaker’s observations, firsthand memories and reports they heard from this listener. The player sees the words they will say. After each exchange, nearby residents take their usual turns. If the listener is still within reach, the menu refreshes. [Try the seed-42 conversation](../../README.md#talk-to-a-worker)

Players and residents can submit the same speech action with a listener, topic, intent and tone. A question about a spoon in someone else’s pocket gets an answer from the listener’s own view. A worker seeking a seat can say they need one. A report preserves the person who told it and which event they witnessed. Both participants can read their exchange in their journal.

### Checkpoint 5 results

Three waits brought the seed-42 baker to the shopfront. The player asked what the baker had seen and learned that the porter had introduced themselves to the baker. A new line let the player confirm what they had heard. The journal quoted the baker. A separate simulation test had the baker volunteer the news to the player. The porter was nearby and received no journal entry for that exchange. A scene stocked with extra visible objects offered eleven distinct lines. Hiding the selected object before submission invalidated its question without spending a turn. [Simulation checks](../../internal/simulation/dialogue_test.go)

An interrupted SQLite receipt write preserved the pre-conversation checkpoint. Reopening the save restored the follow-up question. Retrying both lines by number preserved the journal, and substituting a different number produced a request conflict. [Saved conversation walkthrough](../../cmd/sloptown/dialogue_test.go)

A save made by checkpoint 4 at `4a1d626` replayed to the same tick-three hash in both executables. Continuing that save with typed choices and making the same choices in a fresh Linux terminal using Down and Enter both reached tick 5, sequence 20, with SHA-256 `5e58b9675cdff165c41eef113171ed6e4e21c8317483782b28280899294ffc66`. The full offline build and tests, both smoke runs and three race-detector runs passed.

## Checkpoint 6 — conversations between residents

In new practice bakeries (content version 9), a worker who recognizes a coworker can ask how they are doing when both have reached their work station and have a free turn. The answer reflects how the other worker feels at that moment. Walking, taking a needed break, making an introduction and speaking with the player take priority. The same pair waits at least eight turns before another routine check-in, enough time for a trip from the shopfront to the prep area and back. The exchange takes place within the existing turn. [Try the missed meeting](../../README.md#catch-up-on-a-conversation-you-missed)

An off-screen meeting records who spoke, the question's intent and tone, and the answer as a fact. Each participant remembers their side. On a later visit, either can describe the conversation in their own words. The player's journal credits the person who told them. Older saves keep the routines for their original content version. [Master design, section 14.9](../design/Slop_Town_Master_Design_Revised_v7.docx)

### Checkpoint 6 results

On seed 42, the player walked around a corner inside the bakery on turn four. Both workers were out of sight. At the shopfront, the baker checked in on the porter, who felt fine. The two workers’ notebooks and journals agreed on who had asked and answered. The player’s journal first mentioned the meeting on turn six, after the porter described it. A follow-up on turn seven let the player confirm who had spoken. [Simulation checks](../../internal/simulation/check_in_test.go)

Interrupting the SQLite write for turn four rolled back the meeting and both memories. After reopening, retrying the visit recorded each exchange once. Replay from sequences 0, 12 and 15 reached tick 7, sequence 25, with SHA-256 `650517bfbf9fe414a7d2f57a937b520d75ef56fc99d0f7e3d0bf8eda05606a08`. A version-eight save created by `3d5aadc` retained its journal text and replay hash through reopening and retry. The offline build and test suite, both smoke runs and three race-detector runs passed. The terminal also cross-built for Windows amd64 and macOS arm64. [Restart and replay walkthrough](../../internal/storage/check_in_test.go)

## Checkpoint 7 — news from an eyewitness

In new practice bakeries (content version 10), familiar workers can use a lull to describe an act they personally saw. A report can cover someone sitting down, opening a door or container, or closing one. The worker chooses a sighting they have yet to share with that coworker. Work, rest, introductions and the eight-turn interval between routine conversations still govern when they can speak. When neither worker has news to share, they can check in as before. [Try the seed-42 visit](../../README.md#find-out-who-saw-you)

“What have you seen around here?” draws on the listener's firsthand memories. “What have you heard from other people?” lets them repeat an eyewitness's account and name its source. A player can likewise pass on a report heard directly from a witness. This checkpoint covers one retelling. The recipient can confirm who told the story. The journal quotes the person who spoke to its owner.

### Checkpoint 7 results

The seed-42 porter saw the player sit down on turn one, then told the baker at the shopfront on turn four while the player was out of sight. The player first learned about that exchange by asking the baker on turn six. Their notebook links the baker's reply to the porter's report and then to the original sighting. In a separate test scene, the porter saw a miller open a crate that later closed. The baker could report who saw it open without claiming to have seen inside. [Simulation tests](../../internal/simulation/report_test.go)

Interrupting the SQLite write at either conversation rolled back that turn. Retrying after reopening preserved one copy of each account. Replay from sequences 0, 12 and 19 reached tick 7, sequence 25, with SHA-256 `1e33ef756f4c08e5d5030fb8171d7ae1eecc493f0d4d07bfb0d8f8d8cfe693bc`. A save made by checkpoint 6 at `a169e8f` kept its original journal and replay hash. [Restart and replay test](../../internal/storage/report_test.go)

## Checkpoint 8 — ask the named eyewitness

After hearing a secondhand account, the player can take it to the person named as its witness. Their question identifies the teller and the particular act. The witness answers from the notebook entry for that event. The journal keeps the teller's account alongside the later answer from the witness. Once the player has heard directly from the witness about that report, the menu can use its slot for another topic. [Try the witness question](../../README.md#find-out-who-saw-you)

This also works when the player heard the account in a version-ten save created at checkpoint 7. A remembered sighting remains available through its original event reference. A dialogue menu offers at most eleven lines. Previously recorded choices still replay if a new eyewitness question now occupies a menu slot.

### Checkpoint 8 results

In a save written by `a236e25`, the player had heard and confirmed the baker's secondhand account on turn seven. Reopening the save and retrying those seven commands preserved the journal and replay hash. In a Linux terminal, Down and Enter selected “The baker said you saw me sit down. Is that right?” The porter replied, “I saw you sit down.” Replay from sequences 0 and 25 reached tick 8, sequence 27, with SHA-256 `cead21908106706c6e4c0d95f3e0486e077d18110cd040ca94dde50b14f0ae75`.

In the miller test scene, the porter remembered the miller opening the crate even though it had since closed. Choosing the wrong person, guessing the original event number, or resubmitting the question after hearing the direct answer spent no turn. Interrupting the SQLite write left the earlier account intact and the question available on retry. The journal retained one direct answer. [Simulation checks](../../internal/simulation/report_test.go) · [SQLite walkthrough](../../internal/storage/report_test.go)

## Checkpoint 9 — a calendar for the journal

The bakery calendar gives the saved simulation clock a date. The prototype starts on day 1 of spring, year 1, uses 24 ticks per day and seven days per season, and cycles through spring, summer, autumn and winter. The turn that reaches tick 24 belongs to day 2. Resident activity within the same turn shares its date. These lengths set the pace for the demo. World-specific calendars, holidays and seasonal effects belong to later world simulation work. [Master design, sections 8 and 17](../design/Slop_Town_Master_Design_Revised_v7.docx)

In play, choose `calendar` with the arrows and Enter or type the command. It reads the current date even if another session has advanced the town. The journal dates each personal memory from its original event. A new day opens a chapter on the character's first entry that day. A meeting or arrival on that turn shares the heading. A walk that crosses into another day picks up in the next chapter. Days the character missed have no pages. [Read your journal](../../README.md#read-your-journal)

### Checkpoint 9 results

A seed-42 save created by `8faff65` reached the bakery doorway on day 7 of spring. Interrupting the SQLite receipt write on the next step kept the journal byte-for-byte intact and the clock at tick 167. Retrying put the character inside on day 1 of summer, with one heading for the date and the arrival. The old and new executables agreed on the final replay hash. Replay from sequences 0 and 334 reached tick 168, sequence 336, with SHA-256 `bf88d4af2e710fbcb49ff49ee131e975641659f58e831cbb0b31717579f5824c`.

Up and Enter showed the summer date in a Linux terminal. The in-game export matched the standalone journal byte for byte. Separate tests covered a named walk across midnight, days spent by another actor and the return to spring in year two. The offline build, full test suite, both seed-42 smoke runs and three race-detector runs passed. The CLI also cross-built for Windows amd64 and macOS arm64. [Calendar tests](../../internal/simulation/calendar_test.go) · [Journal walkthroughs](../../cmd/sloptown/calendar_test.go)

## Checkpoint 10 — illustrated journal export

`export illustrated journal` makes a single HTML book from the same dated chapters and first-person paragraphs as the text edition. The first three illustrations are ink sketches of the bakery door, the crate and the wooden spoon. They were generated for the book and ship in the executable. Exporting works offline, including from an in-memory visit. [Try both editions](../../README.md#export-an-illustrated-journal)

Each sketch belongs to a particular object. The journal selects it from the character’s first saved sighting of that object and places it with that passage. A report can mention a spoon before the player sees one. In that case the book quotes the speaker and waits for a firsthand sighting to draw the spoon. Text and alternative descriptions are readable in a narrow browser window or in print. [Master design, sections 7.8 and 17](../design/Slop_Town_Master_Design_Revised_v7.docx)

### Checkpoint 10 results

A seed-42 save written by `bd1fc70` yielded two sketches before the crate opened. After the player found the spoon, picked it up, put it back and closed the crate, the book showed the spoon once at its discovery. Retrying the saved visit and exporting through Up and Enter in a Linux terminal produced the same HTML as the standalone command. Both executables produced identical plain-text journals and replayed to tick 8, sequence 16, with SHA-256 `f2b87770a60aa9a7c5d76794dbd68e2aab0efaffdf8f4f332d9027efc25aa861`.

An empty timeline exported a book with no pictures. A save deliberately named `slop-town-journal.html` kept its contents while successive books received new filenames. The three embedded images decoded to the bundled artwork. The export was checked at desktop and phone widths in Chrome on Linux. [Export walkthrough and privacy tests](../../cmd/sloptown/journal_illustrated_test.go)

The offline build, full test suite, both seed-42 smoke runs and three race-detector runs passed. The CLI also cross-built for Windows amd64 and macOS arm64.

## Checkpoint 11 — remembering who told a story

Section 38 calls for story threads that can be rebuilt from the evidence available to a character. The first thread follows an account of someone sitting down, opening something or closing it. `journal accounts` collects what the character heard, who told them, and whether the named witness has spoken to them directly. It is available by typing or with the arrows and Enter. [Try the account in the bakery](../../README.md#recall-who-told-you)

Each telling is dated on the day the character heard it. The simulation follows the reference the teller shared and groups accounts by the witness and the original event. Hearing the same story from the same teller twice keeps the date of the first telling. Two eyewitnesses to the same act remain separate, as do two similar acts witnessed by one person. A witness who hears their own story repeated can recognize it. Reading uses the saved notebooks and spends no turn.

### Checkpoint 11 results

In a seed-42 practice save created by `c085632`, the workers discussed the player’s rest out of earshot. The player’s account list stayed empty until they asked the baker for news. The baker named the porter as the witness. Interrupting the write for the porter's reply left the baker as the only source at tick 7. After retry, opening the accounts with Up and Enter showed both sources once. Replay from sequences 0 and 25 reached tick 8, sequence 27, with SHA-256 `cead21908106706c6e4c0d95f3e0486e077d18110cd040ca94dde50b14f0ae75`. The previous executable replayed the same save to the same hash. [Restart and rollback walkthrough](../../internal/storage/report_test.go)

A separate test followed two witnesses with the same visible name and two sightings of the same action. It also checked the date when a later teller described a sighting the player had already heard about from the witness. [Account grouping tests](../../internal/simulation/accounts_test.go)

The full offline build and tests, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64. Interactive input was exercised on Linux.

## Checkpoint 12 — tell the reader what matters

Use a chapter heading to establish the place once. Under “Inside the bakery,” the spoon walkthrough opens with “I opened the crate. I saw the wooden spoon in the crate.” Skip uneventful compass steps in the journal. When something happens away from the last narrated place, give that incident its setting. Keep concrete discoveries, fatigue, named destinations, interruptions and the words the character heard.

Apply the same editing rule during play. A new location, discovery or encounter can carry the turn on its own. A quiet step or wait still gets a short acknowledgement so a player using the arrows or a screen reader knows it succeeded. The opening scene and `look` give the full description and grid. Scripted positional actions also use concise narration by default, with `-full` available for per-step snapshots.

The acceptance walkthrough reopens two visits at the same corner. One player asks the baker and the porter about a sighting and opens the crate. The other stays around the corner. The test compares their accounts, journal text, illustrations and replays. [Phase 7 bakery acceptance results](phase-7-results.md)

## The full-town display

Use text and the local top-down grid as the default navigation view. The grid follows the player and updates the visible cells, people and objects after each committed action. If the baker steps into a doorway beside the player, the grid can show both occupants at that location. Names and actions remain available through text. Closed containers and rooms outside the character’s view conceal their contents.

Offer one optional picture-book scene for atmosphere. For example, one picture can show the player’s view of the baker beside the open crate. Build it from the same visible people, objects and surroundings used for the grid. Reuse the picture while that view still matches. If a new scene is still being drawn, keep play responsive and show the result only if it still matches the player’s view. With illustrations off, text and grid support the whole visit. The journal’s object sketches remain attached to the passages where they were discovered.

The bakery currently has a text grid and illustrated HTML export. The graphical grid and optional live scene belong to later client work. [Master design, sections 7.8 and 47](../design/Slop_Town_Master_Design_Revised_v7.docx)

## Checkpoint 13 — facts behind the narration

The live bakery scene identifies the street, doorway and interior as distinct places. It also identifies sitting down, tired legs, relief and lingering relief when the character can feel them. The terminal uses these facts to describe a change. The journal uses them to give an incident its setting and to omit posture that the action already explains. A character who rests and feels relieved remembers both the rest and the relief.

The journal recovers those same facts from the character’s own version 1 receipts. Other saved observations still supply their original descriptions. The next extension is to give a newly seen object and its visible location the same structured treatment, including the difference between seeing a spoon in a crate and hearing someone mention it.

### Checkpoint 13 results

On the seed-42 visit where the player asked both workers and found the spoon, the executable from `ff1308b` and the revised build produced identical brief and story transcripts. Both journal exports matched byte for byte across builds. Replaying the old save from sequences 0 and 15 reached tick 9, sequence 31 and the hash recorded in the [bakery acceptance results](phase-7-results.md). A separate test varied the supplied wording for arriving, sitting down and feeling relieved. The book still opened with one bakery heading and “I rested. I felt relieved.”

The offline build, full test suite, two seed-42 smoke runs and three race-detector runs passed. The terminal also cross-built for Windows amd64 and macOS arm64. [Journal tests](../../cmd/sloptown/journal_test.go) · [Receipt compatibility test](../../internal/simulation/observation_test.go)

### When will the game call a narration API?

Phase 7 supplies the offline narrator. Phase 10 adds optional AI adapters, with a local model first and opt-in cloud providers after that. For a crate opening, either renderer should receive one scene packet containing the accepted action, the crate, and the spoon the character just saw. The model can phrase the discovery. The simulation decides what the crate contains and whether opening it revealed anything.

Before showing generated prose, check its claims against the packet and the character’s knowledge. If the provider is unavailable or adds an unsupported detail, use the offline text. Record accepted output against the triggering event and timeline so reopening or replaying a visit can reuse the wording. [Master design, sections 3, 17, 38 and 50](../design/Slop_Town_Master_Design_Revised_v7.docx)

## Checkpoint 14 — where I first saw it

Opening the crate can now give the journal a firsthand discovery with the object’s reference, its visible label, and where the character saw it. The bakery distinguishes an object at the character’s feet, in a direction, in a named container, or in their own inventory. A version 1 receipt can name a container without identifying which of two identically named containers it was. The journal keeps exactly that much detail. A spoken mention stays with its speaker. A sketch appears at the first sighting of the illustrated object.

The terminal calls the simulation’s journal API locally. With a save file, the storage layer reads the character’s receipts and events from SQLite. An in-memory session supplies the same facts directly. Both journal editions fill in the same bundled English phrases. For instance, “I saw the {object} in the {container}” can describe a spoon in a crate or a key in a box. The seven first-person phrases for places and sensations also live in that file. Rebuilding the game bundles any wording edits. [English journal phrases](../../cmd/sloptown/journal.en.json)

### Checkpoint 14 results

A seed-42 save made by `4c18125` reopened with the spoon still hidden. After opening the crate, taking the spoon outside and dropping it, the journal still placed the discovery inside the crate. The old and new builds produced byte-identical plain and illustrated books before and after the discovery. Replay from sequences 0 and 8 reached tick 9, sequence 18, with SHA-256 `d776e5add24b8b06b5bbca7384064bef640be6d271df2b50692fcfff4c20d646`. Brief and story walkthroughs also matched across builds when run from an empty directory without a save file.

A separate test gave two wooden spoons the same name and placed one on the street and one inside the crate. Each kept its own first sighting. In an illustrated journal test, hearing about a spoon produced dialogue, and seeing it later produced the discovery paragraph and its sketch. [Discovery tests](../../internal/simulation/discovery_test.go) · [Illustrated journal tests](../../cmd/sloptown/journal_illustrated_test.go)

The full offline build and tests, two seed-42 smoke runs and three race-detector runs passed. The terminal also cross-built for Windows amd64 and macOS arm64.

## Checkpoint 15 — look up a first sighting

`journal discoveries` lists the character’s recorded first sightings in order. Each date appears once and the setting changes when the sights do. Use the arrow menu to open the full list, or type a name such as `journal discoveries wooden spoon` to narrow it. A whole word or phrase matches the name the character observed. Two objects called “wooden spoon” can each have an entry. A mention during a conversation stays in the ordinary journal until the character has a sighting of their own to look up.

The terminal reads the existing journal for the current character and timeline. A search spends no turn and preserves a pending numbered choice and the object referred to by `it`. The list uses the same offline wording as the full journal. [Try the lookup](../../README.md#read-your-journal)

### Checkpoint 15 results

A seed-42 save created by `49c2f53` had reached tick 23 with the crate closed. A search for “spoon” found no entry. The player opened the crate on day 2, took the spoon outside and dropped it. On reopening, the street view showed the spoon at the player’s feet. Choosing the index with Up and Enter in a Linux terminal recalled the bakery and the crate on day 2.

Reading the index preserved the save. The old and new builds replayed from sequences 0 and 46 to tick 28, sequence 56, with SHA-256 `0b03fe7af6828a754b3a3013d057c25bd94cc1172b77d1a3dde5db92b58469c3`. The text and illustrated editions matched across builds. Other checks covered an in-memory visit, an empty timeline, a secondhand mention, and searching for two identically named spoons while choosing a destination box. [Discovery lookup tests](../../cmd/sloptown/journal_discoveries_test.go) · [Numbered choice walkthrough](../../cmd/sloptown/targets_test.go)

The offline build, full test suite, both smoke runs and three race-detector runs passed. The terminal cross-built for Windows amd64 and macOS arm64.

## Checkpoint 16 — where I last saw it

`journal last seen spoon` recalls the most recent saved turn on which this character could see a matching object. Typing `journal last seen` or selecting it with the arrows and Enter lists one memory per person or object, grouped by the original observation date and setting. Two spoons with the same name keep separate memories. Reading takes no turn and preserves a pending choice.

The journal takes the name and location from the character’s own saved observation. Carrying an object is a sighting. After a lid closes or someone leaves the character’s view, the earlier sighting keeps its date. Someone else’s account can be read under `journal accounts`. Browsing, `look` and reopening a save do not create a saved turn. Both sighting indexes work with existing saves and in-memory visits. [Try both lookups](../../README.md#read-your-journal)

### Checkpoint 16 results

A seed-42 save from `e7348f3` recorded the spoon in the bakery crate on day 1. On day 2 the player carried it to the street. A failed SQLite write while dropping it left the journal saying “I was carrying the wooden spoon.” Retrying the drop updated the last sighting to “I was on the street. I last saw the wooden spoon right beside me.” The first sighting still named the crate.

In a second visit, the baker took the spoon behind a closed door. The player recalled the crate and the baker recalled carrying the spoon. The player saw the new location after returning and taking a turn with the spoon visible again. [Observer and same-name tests](../../internal/simulation/discovery_test.go) · [Lookup and restart tests](../../cmd/sloptown/journal_discoveries_test.go)

Up and Enter opened the index in a Linux terminal. Both executables replayed the saved visit from sequences 0 and 46 to tick 27, sequence 54, with SHA-256 `aa20f9649978f08649874146dd4fa32e6bb58d1f7d8e140899394ac3b5c7d9df`. The plain and illustrated journal exports matched across builds. The full offline suite, both smoke runs and three race-detector runs passed. The terminal cross-built for Windows amd64 and macOS arm64.

## Checkpoint 17 — describe a walk’s arrival once

In a fresh seed-42 bakery, step west and choose `walk to door`. Brief and story mode both say “You reach the bakery door.” The journal records “I reached the bakery door.” On a longer walk the player can read about a departure, a change along the way, and an arrival on different turns. A greeting that interrupts a walk still gives the player the next choice.

The arrival comes from the selected destination in the walker’s saved observation. A second object with the same name has its own reference. Story mode skips the extra “beside you now” for the destination and still describes other things the player notices. Both journal editions use the same bundled first-person phrase. [Walk receipt tests](../../internal/simulation/approach_test.go) · [Terminal walkthroughs](../../cmd/sloptown/narration_test.go)

### Checkpoint 17 results

A save from `1b4666c` recorded the short walk to the bakery door. Reopening it with the new build gave the shorter arrival in the plain and illustrated journals. After a longer walk to the crate, `open it` revealed the spoon. Both builds replayed the visit from sequences 0 and 4 to tick 27, sequence 54, with SHA-256 `577bb5784076a6702bf54f7ea87e84fd12522521657c61efdce7cc1bf0abb107`.

A walk that ended on the first turn of day two opened the new chapter with “I reached the crate.” A walk that still had a step left at daybreak recorded the continuation and then the arrival. [Calendar tests](../../cmd/sloptown/calendar_test.go)

In a Linux terminal, Down and Enter selected the west step, the return walk and the journal. The arrival highlighted **bakery door** and the save ended at tick 2. The full offline test suite, both seed-42 smoke runs and three race-detector runs passed. The CLI also cross-built for Windows amd64 and macOS arm64.

## Checkpoint 18 — open questions for eyewitnesses

The master design describes leads that players can discover through conversations and revisit in the journal. The first kind of open question comes from hearing someone name a witness. In the seed-42 practice bakery, `journal leads` can show “Did the porter see me sit down, as the baker said?” Choose it at the main prompt or among the conversation choices with the arrows and Enter. The dialogue keeps its original choices while you read. [Try it in the bakery](../../README.md#find-out-who-saw-you)

Each question names the person who first told the reader and carries the date of that telling. The simulation groups reports by witness and sighting. Confirming the source with the baker leaves the question open. Hearing from the porter about that sighting settles it, even if someone later repeats the story. The dated tellings remain in `journal accounts`. [Account projection and tests](../../internal/simulation/accounts_test.go) · [Journal wording](../../cmd/sloptown/journal.en.json)

### Checkpoint 18 results

In a save created by `4cf825f`, the player had spoken with the baker on day one. Reopening the save and reading the lead during the conversation with the porter kept the clock at tick 7. In a Linux terminal, Down and Enter selected the original witness question. The porter's reply closed the lead on tick 8. The previous and current executables produced identical plain and illustrated journals and replayed from sequences 0 and 25 to sequence 27 with SHA-256 `cead21908106706c6e4c0d95f3e0486e077d18110cd040ca94dde50b14f0ae75`.

An interrupted SQLite write while hearing the baker's story left the leads empty. A failed write during the porter's reply kept the question open until retry. Separate checks covered two witnesses sharing a name, two sightings by the same witness and a witness hearing their own account repeated. [Rollback and restart walkthrough](../../internal/storage/report_test.go) · [Journal tests](../../cmd/sloptown/journal_test.go)

The full offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 19 — witnesses to item transfers

In a new practice bakery, a character can remember seeing someone take an item, drop it, or put it in a particular container. The simulation checks the witness's view just before a take and just after a placement, before the next character acts. The person and the item must both be visible. Putting something away also requires the witness to see the destination. The notebook records the item's reference and the destination of a put, so the account survives another move or a closed lid.

The same memory feeds the witness's journal, a direct answer, a coworker's retelling and the player's follow-up question. After the player hears from the witness, a later retelling leaves that dialogue slot available for another sighting. [Playable spoon walkthrough](../../README.md#follow-the-spoon-through-town-gossip)

### Checkpoint 19 results

On seed 42 the baker saw the player put the wooden spoon in the crate on tick 6 and told the porter during a lull on tick 15. The player heard the porter's account on tick 16. Asking the baker directly closed the lead on tick 17. Down and Enter selected both questions in a Linux terminal. Interrupting the SQLite write for the put rolled back the placement and the witnesses' entries together. After retry, replay from sequences 0 and 19 reached sequence 60 with SHA-256 `d720a538f3a7f1e9a4a543006a407ce4ab1b9794531d17ec1a26095ffe069882`. [Restart and replay test](../../internal/storage/report_test.go)

In a separate scene the player saw the baker take, put back, take and drop the same spoon. A second wooden spoon stayed in the baker's pocket. Two corner views tested partial sightings. In one the player could see only the baker, in the other only the open crate. Neither produced a witness account of the put. [Visibility and identity tests](../../internal/simulation/item_witness_test.go)

New practice timelines use content version 11. A version 10 save from `273bb17` continued under its original rules. Both executables produced identical transcripts and journal exports and replayed it from sequences 0 and 19 to tick 7, sequence 25, with SHA-256 `b0acff82692ad0758d15bf3c870fcabc694b10ed456ebc6d171a0cd616b7cac3`. The full offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.
