# Phase 7 — emergent narrative and deterministic dialogue

Section 17 of the master design calls for a consistent narrator, adjustable narration length, and a chronicle of what the player saw, heard, and did. Sections 38 and 47 ground that writing in the events and observations available to the character. Phase 7 is the roadmap home for this work. The first presentation checkpoint runs in the Phase 5 bakery. [Master design, revision 7](../design/Slop_Town_Master_Design_Revised_v7.docx)

## Planned mechanic — group conversation and attention

The current bakery resolves dialogue between a speaker and one listener. The planned group mechanic lets three or more people share a conversation. At a party of 25, several circles can carry on at once. A toast might draw most of the room into one conversation for a moment. People can join, drift away or start a side conversation while others carry on.

**Speaking in a group.** A speaker can address the circle or a particular person in it. For example, the baker can ask the porter when a delivery is due, and a customer who heard the question can answer first. The porter can respond, let the answer stand or pick it up later. Resolve one intelligible contribution at a time within each conversation. If two people start together, someone can yield or their words can overlap and be missed. Separate circles can speak during the same town turn. A player or resident with enough listening skill and rest, standing where both groups are audible, can participate in both while their involvement is light. The porter might answer “Two trays” to the baker, catch a question from the next table and reply “At noon” during a gap. Both conversations stay open to the porter. If the questions overlap, the porter may have to ask someone to repeat theirs.

**Paying attention.** First establish which voices reach each character through distance, doors and competing noise. Then resolve what that listener follows. A rested bystander can keep up with a few nearby conversations and favor one when voices overlap. A casual participant can still catch parts of a nearby exchange, especially while someone else has the floor. Speaking, preparing a reply, fatigue and the character’s perceptual abilities affect how much they can follow. An emotionally demanding exchange can absorb someone completely. For them the other circles blend into a loud murmur, even while a calmer person beside them can make out words. Attention can widen again as the exchange eases. Exact capacities and thresholds remain open for playtesting.

**Emotional weight through the day.** Each remark carries weight for the person saying it and for each person who understands it. “At noon” might take little thought for the porter giving a delivery time, yet land heavily for a baker worried about missing that delivery. A routine answer leaves attention free for the other circle. A difficult admission can occupy the speaker as fully as hearing difficult news can occupy a listener. Emotional inertia carries that feeling into later exchanges, even after the subject changes. A fresh remark can reinforce it, ease it or let it gradually fade. Keep these judgments specific to the character, their relationships and recent experience.

**Playing and remembering.** Let the player listen to a recognizable circle, shift attention, join in, address someone by name or step away, using typed commands or the arrow menu. Previewing choices takes no time. Choosing to listen follows the next exchange as town time passes. Shifting focus cannot recover a missed line. Keep the existing limit of up to 11 suggested lines in total across the conversations the player is following. Identify the intended person or group on each reply. A player might hear “after closing” from beside the door without knowing who spoke or what will happen. Their journal can keep that fragment and its uncertainty. A murmur supplies atmosphere alone. Later questions, retellings and optional narration use only what that individual understood. Residents follow the same rules and choose their own focus.

**Acceptance scene for implementation.** Run the same seeded party from the perspectives of a rested bystander, a tired participant and someone absorbed in a difficult exchange. Check that a third person can answer a group question, another circle can continue during that turn, and listeners leave with different accounts. Give a rested, skilled character between the two circles a routine question from each and verify they can answer both. Repeat as a resident. Give one answer personal stakes, then check that its lingering effect can make the character miss a later remark from the other circle. Move beyond earshot and verify that skill alone cannot recover the words. Switch attention midway and verify that earlier missed words stay missing. Reopen the save and replay it offline. Each participant should remember the same words, fragments and speaker uncertainties they experienced before reopening.

The first slice, content version 32, lets someone who can see both people remember a check-in they did not join. Someone out of sight remembers nothing. Asked about recent conversations, that bystander retells the exchange and names a participant who is listening. That participant can ask whether they heard it, and the bystander confirms the same account. The participant can then pass that retelling to the other person in the check-in. That person can ask them to confirm the passed account. Fresh practice towns still use the earlier conversation rules until this checkpoint is part of a new town. [Bystander check](../../internal/simulation/group_test.go)

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

Interrupting the SQLite write for turn four rolled back the meeting and both memories. After reopening, retrying the visit recorded each exchange once. Replay from sequences 0, 12 and 15 reached tick 7, sequence 25, with SHA-256 `d095d85c12cfdbea83832f8cf4af89ba23d02f27e0b4f649ccc9f0b4e883f0a5`. A version-eight save created by `3d5aadc` retained its journal text and replay hash through reopening and retry. The offline build and test suite, both smoke runs and three race-detector runs passed. The terminal also cross-built for Windows amd64 and macOS arm64. [Restart and replay walkthrough](../../internal/storage/check_in_test.go)

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

`export illustrated journal` makes a single HTML book from the same dated chapters and first-person paragraphs as the text edition. This checkpoint originally shipped three ink sketches of the bakery door, crate and wooden spoon. The [grid illustration decision](#the-full-town-display) replaced them with observed maps. Exporting works offline, including from an in-memory visit. [Try both editions](../../README.md#export-an-illustrated-journal)

Each original sketch belonged to a particular object. The journal selected it from the character’s first saved sighting of that object and placed it with that passage. A report could mention a spoon before the player saw one. In that case the book quoted the speaker and waited for a firsthand sighting to draw the spoon. [Master design, sections 7.8 and 17](../design/Slop_Town_Master_Design_Revised_v7.docx)

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

**Camera decision, 27 November 2026.** All new scene and journal illustrations use the grid, seen from directly above or from a Diablo-style isometric angle. This supersedes the separate picture-book camera in sections 7.8 and 47 of the master design and the earlier bakery portrait experiment. The terminal and the HTML book use the top-down projection today. [Master design](../design/Slop_Town_Master_Design_Revised_v7.docx)

For the Phase 9 client, draw visible ground, structures, furnishings, people and interaction overlays against the same floor and cell coordinates. Keep multiple occupants selectable on a shared square. At an isometric angle, draw occupants in depth order and keep the selected square readable behind tall objects. Text identifies the same visible targets. Reuse tile and object art across locations. Measure memory and frame time on a representative town before setting an asset budget.

The illustrated journal now takes one map from the character’s saved observation on a turn with a new sighting. Two tools discovered together share a map. An unheard event or a secondhand account supplies no visual evidence. Earlier saves already contain these observed grids. [HTML export and privacy checks](../../cmd/sloptown/journal_illustrated_test.go)

On a six-turn seed-42 visit saved by `f940d29`, the player entered the bakery, opened the crate, took the spoon and closed the lid. The previous executable exported a 121,880-byte HTML journal. The top-down edition exported the same visit in 3,368 bytes, with the visible spoon still in the crate on its discovery map. The plain-text editions matched byte for byte.

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

## Checkpoint 20 — stealth and perception checks

In new practice bakeries, a character can discreetly pick up a visible item within reach. A player can type `discreetly take NAME` or choose it with the arrows and Enter. The pickup takes one turn. Ordinary pickups still produce an eyewitness account for everyone who can see the person and the item. [Playable discreet pickup](../../README.md#try-a-discreet-pickup)

Just before a discreet pickup, each observer who can see the acting character makes one hidden check. The simulation weighs the actor’s stealth, the observer’s perception and their distance in tiles. Each character’s initial aptitudes come from the town seed, and each observer has their own saved sequence of draws. Players and residents use the same rules.

A witness who identifies both the person and the item remembers an ordinary pickup. A partial sighting records an anonymous hand movement and names the container only when it was visible. A missed movement leaves no account to pass along. Checkpoint 21 adds deliberate listening for footsteps and doors. Checkpoint 23 adds the sound of a dropped spoon. Checkpoint 24 adds daylight to discreet pickups. Checkpoint 25 adds stealth practice. Checkpoint 26 lets a character hide behind the crate. [Master design, sections 7.6, 12.2, 12.5 and 46–47](../design/Slop_Town_Master_Design_Revised_v7.docx)

### Acceptance walkthrough

| Situation | Expected result |
| --- | --- |
| Look at a spoon in an open crate, then close the lid. | The spoon in plain sight appears automatically. The closed lid blocks visual identification regardless of perception. |
| Make one discreet pickup with two differently placed observers. Repeat with a resident taking the spoon. | The transfer costs its normal action time. A seeded fixture demonstrates different accounts of the same act under the same rules for players and residents. |
| Ask each witness what happened, then read the journal and any relayed accounts. | Only a witness who identified both person and item can attribute the pickup. Partial observations retain their uncertainty. The player learns another character's reaction through observable behavior or conversation. Rolls, odds and missed checks stay hidden. |
| Browse actions, use `look`, reopen the save and retry the original command. Interrupt a write and replay from a checkpoint. | Resolve each eligible observer once per committed action in stable order. Reads and retries cannot reroll it. The item and witness memories commit or roll back together. Earlier content versions reproduce their original sightings. |

### Checkpoint 20 results

On seed 4 the player discreetly took the wooden spoon on tick 6. Asked immediately, the porter identified the player and the spoon. The baker saw only a hand move near the crate. In a second run the player waited until the workers had spoken, then heard the baker’s account from the porter. Asking the baker directly confirmed the hand movement and closed the journal lead. In a Linux terminal, Down and Enter selected the pickup and both eyewitness questions. [Rollback, dialogue and replay test](../../internal/storage/discreet_take_test.go)

The same rule handled a resident taking an item. With the baker picking up a spoon from the floor, the player saw a hand move and could name neither the person nor the object. A witness on the other side of a closed door missed the movement entirely. At an open doorway, seeing just the actor allowed an anonymous sighting, while seeing just the item produced no eyewitness account. [Visibility tests](../../internal/simulation/discreet_take_test.go)

An interrupted SQLite write rolled back the pickup, notebooks and perception draws together. After retry, replay from sequences 0 and 19 reached tick 17, sequence 60, with SHA-256 `ea026b91111c8258a910b8ecce73ac15097ea4a2fbf044292f797857629b1797`. New practice timelines use content version 12. On a version 11 save from `7744c0a`, both executables produced the same ordinary pickup transcript and replayed to tick 7 with SHA-256 `25bca1e85ac15f2d14a71dc8541238efe1a74ff6c253fa777d360486c368ac6c`.

The offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 21 — listen from the street

In a new practice bakery, a player or resident can spend one turn listening. A step makes a footfall at the square where the walker lands. A moving door creaks at its hinge. The simulation checks how far each sound can travel at the moment it occurs. Nearby footsteps can carry around an open doorway. A closed door blocks footsteps passing through it, and its own creak can reach either side. [Playable walkthrough](../../README.md#listen-from-the-street)

For this fixture, footsteps carry three floor squares and a creak carries four. Characters in the upper third of the existing perception aptitude range hear one square farther. Each sound takes the shortest connected route through adjacent floor squares. The listener remembers the distinct sounds in the order they heard them. Hearing alone supplies no name or selectable target. The journal keeps sounds and any other discoveries made during that turn. Checkpoint 22 brings these sounds into conversation. Checkpoint 23 introduces the first sound from handling an object.

### Checkpoint 21 results

At the seed-42 shopfront, Down and Enter selected `listen`. The player heard a door creak followed by footsteps, then read both in the journal. On seed 4, the player walked two steps south, waited three turns, and heard footsteps while both workers were out of sight. A second listen on the quiet seventh turn added no sound to the journal. In a separate scene, the baker stepped behind a closed door just before the porter opened it. The listener heard only the creak. The porter could also listen around a corner and hear an unseen walker. [Simulation checks](../../internal/simulation/listen_test.go)

A failed SQLite receipt write rolled back the listening turn and the workers’ movements. Reopening and retrying reproduced the uninterrupted journal. Replay from sequences 0 and 19 reached tick 7, sequence 25, with SHA-256 `c332aa4106c6c91e41e1a675c316d9b03f75c4068ae33e010462955ff93b3342`. New practice timelines use content version 13. A version 12 save from `4e2a404` replayed its discreet pickup and eyewitness account to the same hash in both builds. [Rollback and replay check](../../internal/storage/listen_test.go)

The full offline suite and both seed-42 smoke runs passed. The terminal, simulation and storage packages passed three race-detector runs. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 22 — talk about what you heard

In a new practice bakery, a character who heard both footsteps and a door creak can tell someone about either one. The conversation draws from recent memories and offers each line once. The original account records the sound and who heard it. Hearing footsteps supplies no identity for the walker. [Playable conversation](../../README.md#tell-someone-what-you-heard)

“What sounds have you heard lately?” draws on the respondent’s latest firsthand sound. “What have you heard from other people?” can name the person who supplied an account. The dialogue menu still offers up to eleven lines. A listener can confirm a retelling with its original source, and the journal keeps the date and attribution of each telling. Players and residents use the same listening and speech actions. The workers’ daily schedule still focuses on moving between stations, resting and talking.

### Checkpoint 22 results

On seed 42 the player heard a door creak and footsteps on the first turn. Using Down and Enter, the player told the porter only about footsteps. When asked for secondhand news, the porter replied, “You told me you heard footsteps.” The journal showed the porter repeating the player’s own account. Separate checks rejected a character borrowing someone else’s sound and a speaker swapping the creak’s source for the footsteps. [Source and dialogue checks](../../internal/simulation/sound_account_test.go)

In a SQLite visit the porter took the listening turn. The baker asked the porter what they heard and later told the player about the creak. The player’s lead read “Did the porter hear a door creak, as the baker said?” The porter confirmed that specific sound and the lead closed. Interrupting the save while the baker was telling the player left the town and both characters’ memories at the previous turn. After retry, replay from sequences 0 and 15 reached tick 6, sequence 23, with SHA-256 `9c7953fbc92ac477e0905b04e62b7244a55aa7d02133af6edfce730518610663`. [Restart and replay check](../../internal/storage/sound_account_test.go)

New practice timelines use content version 14. A version 13 save from `87f610e` kept its original listening and dialogue behavior. Both builds replayed it from sequences 0 and 4 to tick 3, sequence 12, with SHA-256 `01142b74d8fce6611514cb86d66be56db78ec75d790349ac0701c90d4f34676b`.

The offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 23 — hear a spoon hit the floor

In a new practice bakery, dropping the wooden spoon makes a clatter. Anyone within earshot can hear it while going about their work, including the player when a resident drops the spoon. The sound travels up to three floor squares along an open route, or four for a character in the upper third of the seeded perception range. The simulation uses the listeners’ positions and the state of the door at the instant the spoon hits the floor. [Playable walkthrough](../../README.md#drop-the-spoon-around-the-corner)

The floor sound belongs to the item. Picking up the spoon or putting it into the crate is quiet. A character who only heard the drop can remember “I heard a clatter” in their journal and say it in conversation. Identifying who dropped which object still requires seeing the transfer.

### Checkpoint 23 results

On seed 42 the player dropped the spoon outside on tick 9, walked back to the porter, and asked what they had heard. The porter answered, “I heard a clatter.” Down and Enter selected the drop, the walk, the conversation and the question in a Linux terminal. The account survived reopening the save. An interrupted SQLite write rolled back the drop and every listener’s memory together. After retry, replay from sequences 0 and 27 reached tick 11, sequence 44, with SHA-256 `db62a367b969ca5bfc477b0bb438428bc4258174ffe52bed3b086f7adbaa7f3a`. [Restart and replay test](../../internal/storage/drop_sound_test.go)

In a separate scene the baker dropped the spoon around the corner. With the door open, the player and porter heard it. A more distant customer missed it. With the door closed, only the baker heard the impact, even though the porter opened the door later that turn. A second wooden spoon with the same name and no assigned sound fell quietly. [Hearing and identity tests](../../internal/simulation/drop_sound_test.go)

New practice timelines use content version 15. On a version 14 save from `50bfef7`, the spoon still fell quietly and the journal kept the earlier creak and footsteps. Both executables produced the same transcript and replayed from sequences 0 and 29 to tick 10, sequence 39, with SHA-256 `bbfa7ceee8201fc89f51c06e43fd94f761cb86e5344b7977ead3e883a00c40cf`.

The offline suite and both seed-42 smoke runs passed. The terminal, simulation and storage packages each passed three race-detector runs. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 24 — a discreet pickup in dim light

Daylight enters the practice bakery through the street door. Shutting it leaves enough light to see a worker or a spoon in an open crate. A quick hand movement becomes harder to identify. The simulation samples the light where the hand reaches the item, before it moves into the actor’s inventory and before a worker can reopen the door. Players and residents use the same rule. Each eligible witness still uses one draw from their own perception sequence.

Inside the bakery, the terminal describes the change in light once and `look` gives the current scene. The journal remembers the dim room. A character standing outside a closed door gets only the street view. [Playable comparison](../../README.md#try-the-same-pickup-in-different-light)

### Checkpoint 24 results

In two seed-2 visits, the player and porter stood in the same places for the discreet pickup on tick 8. With the door open, the porter identified the player and the spoon. With it shut, the porter saw a hand move near the crate. Both accounts were still in the player’s journal after reopening. A separate scene put the baker in the player’s role. The player missed the baker’s discreet pickup in dim light, even though the porter opened the door later that turn. The same player saw an ordinary pickup in the dim room. [Simulation checks](../../internal/simulation/discreet_take_test.go)

Interrupting the SQLite receipt write preserved the closed door, the spoon in the crate and the witnesses’ perception draws. After retry, replay from sequences 0 and 24 reached tick 9, sequence 30 in both visits. The daylight hash was `3ce6a849ad8f2ac2a982f5076af634a0127bfaaae6ebffe1ebdd0a52d35f23be`. The dim-room hash was `914301916d5beb9b7e6cfd006f4950ff419482f6e544867dfb1ff2fe8ebba660`. [Restart, dialogue and replay checks](../../internal/storage/discreet_take_test.go)

New practice bakeries use content version 16. On a version 15 save from `841b9ae` with the door already shut, both executables produced the same transcript and journal. The porter identified the player and the spoon. Replay from sequences 0 and 24 reached tick 9, sequence 30, with SHA-256 `9188eeaca1283fdbbae5010625bffdb930c097d790dd7edac9ee86579fe9413c`.

In a Linux terminal, Down and Enter closed the door, selected the discreet pickup, asked the porter what they saw and read the account in the journal. The full offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 25 — learning from discreet pickups

Two kinds of reach offer practice in the bakery. One takes an item from an open container, the other from the floor. An attempt qualifies when another character within one square of the actor can see both the person and the item as the reach begins. The actor can learn even when the witness identifies the pickup. Once the actor has practiced both reaches, later discreet pickups get a small, lasting advantage in the existing perception check. Each actor can earn this improvement once. [Playable comparison](../../README.md#practice-a-discreet-pickup)

The developer history marks the pickup that first counted for each kind of reach. Players learn what a witness noticed by asking them. Players and residents use the same rule. In a seed-5 scene the player identified the baker taking a spoon while the baker earned the second kind of practice. With the baker already practiced, the player saw only a hand movement under the same conditions and perception draw. [Resident and eligibility tests](../../internal/simulation/stealth_practice_test.go)

### Checkpoint 25 results

In two seed-51 visits the player made three discreet pickups, ending at the crate on tick 10. After a reach into the crate and a reach to the floor, the porter remembered a hand moving near the crate. After two earlier reaches into the crate, the porter identified the player and the wooden spoon. The player asked on tick 11. Both accounts survived reopening and retrying the visit. [SQLite walkthrough](../../internal/storage/stealth_practice_test.go)

Interrupting the second qualifying pickup left the spoon on the floor and the earlier practice, perception draws and notebooks intact. After retry, replay from sequences 0, 28 and 30 reached sequence 44 with SHA-256 `454a018d6ada3d3069deeff3159a7f398c12882a35ea84abceb50a85e8affa6c`. The crate-only visit replayed from sequences 0, 25 and 27 to sequence 41 with SHA-256 `48d1f928bca04229b964c1594597cce7acfa288c79c362031943eacb8dee1fff`.

New practice bakeries use content version 17. On a version 16 save from `e9cd104`, both executables produced the same transcript and journal. The porter identified the player on the final reach into the crate even after the earlier floor pickup. Both replayed from sequences 0 and 28 to tick 11, sequence 44, with SHA-256 `27fa296ea10c4d887d044be1184c412d4fa1fb9f1584acb16c1d9472ed1856f9`.

Down and Enter completed the seed-51 floor-and-crate visit in a Linux terminal, including the conversation and journal lookup. The offline suite, both seed-42 smoke runs and three race-detector runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 26 — hide behind the crate

A character beside the bakery crate can spend one turn crouching behind it. The crate blocks sightlines that cross its square at crouching height, in both directions. Someone looking from the side can still see the crouching character. The player’s text, grid, available actions and choice of conversation partners use that view. Sound still travels around the crate. Waiting, listening and handling an item can all be done while crouching. Moving, resting or choosing `come out` ends the posture. [Playable walkthrough](../../README.md#hide-behind-the-crate)

A witness who could see the actor and the crate as the actor ducked can recall the hiding place in conversation. That memory records what happened on that turn. It gives the witness no view of what the actor does behind the crate afterward.

### Checkpoint 26 results

On seed 2 the player crouched on the east side of the crate on tick 9. The doorway and both workers dropped out of the player’s view. After reopening the save, `look` showed the crate and `come out` was available in the action menu. The player heard footsteps on tick 10, stood up on tick 11 and heard the baker recall the hiding place on tick 12. Down and Enter selected the full walkthrough and the question in a Linux terminal. [Restart and journal test](../../internal/storage/cover_test.go)

In a separate scene the baker crouched and dropped the wooden spoon. The player at the doorway heard a clatter. A customer on the baker’s side of the crate saw the drop as well. Both had seen the baker duck. After the baker stood up, the player could see the baker again. [Sightline and hearing tests](../../internal/simulation/cover_test.go)

Interrupting the SQLite receipt write rolled back the hiding turn and the witnesses’ memories. After retry, replay from sequences 0, 26 and 30 reached tick 12, sequence 45, with SHA-256 `655b64ef3ad3786d7cf81ce00225a615bd7710af06b558b7fd4f370b9b45977d`.

New practice timelines use content version 18. A version 17 save from `d104063` kept its previous action menu and stealth practice. Both executables produced the same transcript and journal and replayed the seed-51 visit from sequences 0 and 28 to tick 11, sequence 44, with SHA-256 `454a018d6ada3d3069deeff3159a7f398c12882a35ea84abceb50a85e8affa6c`.

The offline suite and both seed-42 smoke runs passed. The terminal, simulation and storage packages each passed three race-detector runs. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 27 — a knock at the bakery door

A character within reach of a visible door can spend a turn knocking on it. The sound starts at the door, so listeners on either side can hear a knock even when it is shut. Distance, obstacles and the listener’s perception determine how far it carries. To identify the knocker, a witness must see both the person and the door at the moment of contact. Residents continue their routines and can later tell the player what they saw or heard. [Playable walkthrough](../../README.md#knock-on-the-bakery-door)

### Checkpoint 27 results

On seed 43, the porter saw the player knock from the street. The baker heard it indoors, then opened the door on the way to work. After reopening the save, the porter replied, “I saw you knock on the bakery door.” The baker replied, “I heard a knock.” Both accounts appeared in the player’s journal. Down and Enter completed the visit in a Linux terminal. [Restart and dialogue test](../../internal/storage/knock_test.go)

When the baker knocked from inside, the player outside heard the front door. A knock on a second door farther inside reached the player only when the front door was open. With a clear view, the journal named the baker and the door once. [Perception tests](../../internal/simulation/knock_test.go)

Interrupting the receipt write rolled back the turn and its memories. After retry, replay from sequences 0 and 6 reached tick 3, sequence 14, with SHA-256 `ccb381f0dbcf0eb5c3354adee06110eaac7b6656dd89a60970a0a3ee68d6b7fb`.

New practice timelines use content version 19. A version 18 save from `94beab3` produced the same seed-2 hiding transcript and journal in both executables. Replay from sequences 0 and 26 reached tick 12, sequence 45, with SHA-256 `655b64ef3ad3786d7cf81ce00225a615bd7710af06b558b7fd4f370b9b45977d`.

The offline suite and both seed-42 smoke runs passed. The terminal, simulation and storage packages passed three race-detector runs each. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 28 — examine a visible person or object

The master design’s examination command takes one turn in the practice bakery. A character can examine any person or object in their current view. An open container reveals its directly visible contents. Posture, lids and doors can be seen at a distance. A small distinguishing mark requires daylight and a place within arm’s reach. The bundled English descriptions are saved with the individual people and objects when the town is created. [Play the seed-42 visit](../../README.md#examine-the-bakery-crate)

### Checkpoint 28 results

On seed 42, the player examined the closed bakery door and saw the worn brass around the latch. The baker opened it later that turn. Examining the crate revealed a scrape on the closed lid. Opening the crate and examining the spoon revealed a notch in its handle. After taking the spoon, the player found the crate empty. The journal kept both examinations in order after reopening the save. Down and Enter completed the nine-turn visit in a Linux terminal. [SQLite walkthrough](../../internal/storage/examine_test.go)

The porter’s frayed sleeve became visible on the second examination, after they moved closer. Replaying the first request returned the original, more distant view. When the baker examined an open crate in a separate scene, they could see a flour tin and a wooden spoon. A key inside the closed tin and a pencil carried by someone else stayed private. Closing the street door hid the fine scrape on the lid in the dim room. [Visibility and timing tests](../../internal/simulation/examine_test.go)

A failed receipt write rolled back the examination and that turn’s worker activity. After retry, replay from sequences 0 and 19 reached tick 9, sequence 31, with SHA-256 `40555b8b5b065d2a04882a46bfa257fc2115557df88a83e07f565503863af859`.

New practice towns use content version 20. A version 19 save from `250e6c6` kept its earlier actions. Both executables produced identical seed-43 knock transcripts and journals. Replay from sequences 0 and 6 reached tick 3, sequence 14, with SHA-256 `ccb381f0dbcf0eb5c3354adee06110eaac7b6656dd89a60970a0a3ee68d6b7fb`.

The offline suite and both seed-42 smoke runs passed. The terminal, simulation and storage packages each passed three race-detector runs. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 29 — see across a doorway and remember people

A character looking through the bakery door can distinguish someone outside on the street, inside the bakery or standing in the doorway. Nearby objects still use bearings when those help with navigation. If two targets match the same name across the doorway, the choice includes their place and bearing. An item in an open crate keeps its container as well as the place where the character saw it. Doors, lids and the observer’s line of sight determine what can appear. [Doorway tests](../../internal/simulation/landmark_test.go)

`journal people` lists people in the order the character first exchanged introductions with them. An introduction in either direction opens an entry. Each entry can recall a distinguishing detail actually noticed during examination, something the person said directly to the reader, and the last recorded sighting. Seeing someone from a distance leaves them out of the book until an introduction. Type `contacts` or use the arrows to choose **journal people**. [Try both visits](../../README.md#look-through-the-doorway-and-remember-people)

### Checkpoint 29 results

On seed 42 the player looked out from inside the bakery and saw the porter outside. In a second visit, the player noticed the porter’s frayed sleeve and introduced themselves. After the player walked down the street, the last sighting still placed the porter in the doorway three turns later. The book survived reopening. Looking up a person kept the replay head unchanged, and two people called “porter” kept separate descriptions and quotes. Down and Enter opened the same book in a Linux terminal. [Contact book tests](../../cmd/sloptown/journal_people_test.go)

Fresh practice towns use content version 23 to record visible landmarks. A version 22 save retained its original harvest transcript, journal and turn-15 replay hash from Phase 6. The book can also read an introduction and a close examination that were already saved under the previous rules. [Previous harvest replay](phase-6.md#checkpoint-2--harvest-a-sprig)

## Checkpoint 30 — send and read a letter

After an introduction, either character can ask for the other’s address or offer their own in conversation. Each decision passes one address to one listener. The people journal records who shared it and the town date. The practice town has three numbered postal boxes. [Play the letter walkthrough](../../README.md#write-to-the-porter)

A character can post from anywhere to an address they have learned, including when the recipient is out of sight. Writing and posting a one-line note together cost one turn. Delivery takes three more town turns. The recipient sees an unopened envelope and spends a turn to read it. The sender keeps the outgoing note in their journal, and the recipient keeps the text after opening. The arrow menu offers a ready-written note. Typed notes preserve spelling and punctuation and can contain up to 500 printable characters.

During a free turn, a resident can open an envelope. On a later free turn, a resident with the sender’s address can post one acknowledgment. Work, movement, rest and introductions keep their existing priority. Replying to the acknowledgment is a fresh player decision. [Address and privacy tests](../../internal/simulation/mail_test.go)

### Checkpoint 30 results

On seed 42 the player and porter shared their addresses separately. The player walked out of sight and posted on turn 8. The letter reached the porter on turn 11. They opened it on turn 12 and posted a reply on turn 15. The player received the sealed reply on turn 18 and read it on turn 19. In a separate visit the porter learned no return address and sent no reply.

Interrupting SQLite while sharing an address or while posting, delivering or opening an envelope preserved the previous town and its history. Retrying the complete batch produced one original and one reply. Replays from sequences 0, 28, 38 and 69 reached turn 19, sequence 73, with SHA-256 `50b446b575ece142d24c304c9c048ac12b789ff549805914f402cd5d82998364`. [SQLite restart test](../../internal/storage/mail_test.go)

Fresh practice towns use content version 24. A version 23 visit from `77771ec` retained its original conversation choices, people journal and turn-4 replay hash `dcb474a8391d8aea98a84385f0a481261585bbf2d5a6cfc03b2abf59839bba59` in both executables.

Down and Enter completed the full correspondence visit in a Linux terminal. The offline test suite and both seed-42 smoke runs passed. The CLI cross-built for Windows amd64 and macOS arm64.

## Checkpoint 31 — place a phone call

After an introduction, either character can offer their own number or ask for the other person’s. The listener’s people journal records the number and the date it was shared. A character who knows a number can dial it or choose the person by name at a working phone. In the practice town, extension 101 reaches the player at the street phone. Extensions 201 and 202 reach the baker and porter at the bakery phone. [Play both calls](../../README.md#call-the-porter)

A call spends one turn. A recipient can answer if they can see and reach their working handset and are free to speak. Resting, seeking a break or hiding can leave a call unanswered. On an answered call, the caller identifies themselves and asks how the other person is doing. The reply uses the same check-in as an in-person conversation. Both participants remember the words. On a missed call, the caller hears the line ring out. The contact book continues to show the person’s last observed location.

### Checkpoint 31 results

On seed 42 the player learned the porter’s extension on turn 4 and got no answer on turn 5. The player then walked out of sight of the shop. On turn 8 the porter answered from inside the bakery. Both calls were available through the arrow controls. In a separate visit the porter used the bakery phone to call the player at the street phone. A disconnected handset, a stale contact selection and a number the caller hadn’t learned were also tested. [Simulation checks](../../internal/simulation/phone_test.go)

Interrupting the SQLite receipt write during the number exchange, the missed call or the answered call preserved the preceding town and its history. Retrying the visit left one call of each kind. Replay from sequences 0, 15, 19 and 26 reached turn 8, sequence 28, with SHA-256 `08fce79c5b37c1f6705eb158e8cba5d47fd8330416c12df8fe1e7a1df3a5a649`. [SQLite restart test](../../internal/storage/phone_test.go)

Fresh practice towns use content version 25. A version 24 save from `1a040cf` kept its letter walkthrough, journal and conversation choices. Both executables replayed the visit to the checkpoint 30 hash on turn 19.

## Checkpoint 32 — ask the baker about the workroom order

In fresh workrooms (content version 27), a character within speaking distance can ask the baker what the workroom needs. The baker names the tool chosen for that town and the intended drawer. A delivery outside the baker’s view leaves the answer unchanged until the baker collects the tool. The baker can thank the speaker for a placement personally witnessed, matching the person, tool and drawer against the baker’s notebook. The reply makes no claim about who delivered an item found during rounds.

On seed 42 the player opened the bakery door and asked on turn 2, before seeing the dough scraper. Reopening the SQLite journal preserved the conversation and replay reached the same turn. Simulation checks cover an unseen delivery, collection, another person delivering the tool, a firsthand delivery and the previous workroom version. [Playable conversation](../../README.md#fill-the-workroom-supply-order)
