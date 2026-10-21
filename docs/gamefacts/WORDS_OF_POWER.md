# Words of Power and learned clues

Canonical words, terrain activation, learned clues, and surveyed entrances are
separate. Original dialogue payload associations and arbitrary decorated-text
matching are not certified from dispatcher code alone.

### WORD-0001

**Canonical words and entrance coordinates**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Matches

| Index | Dungeon | Word | X,Y decimal | X,Y hex |
| ---: | --- | --- | --- | --- |
| 0 | Deceit | FALLAX | 240,73 | F0,49 |
| 1 | Despise | VILIS | 91,67 | 5B,43 |
| 2 | Destard | INOPIA | 72,168 | 48,A8 |
| 3 | Wrong | MALUM | 126,20 | 7E,14 |
| 4 | Covetous | AVIDUS | 156,27 | 9C,1B |
| 5 | Shame | INFAMA | 58,102 | 3A,66 |
| 6 | Hythloth | IGNAVUS | 239,240 | EF,F0 |
| 7 | Doom | VERAMOCOR | 128,128 | 80,80 |

Entrance indexes are 32–39, not keep indexes 24–31. Redux uses this mapping.
The original outdoor Yell handler has no plane or level gate, so it runs in
Britannia and in the Underworld alike. Each of the seven non-Doom dungeons has
an Underworld portal at the same coordinate as its surface entrance, and both
share that dungeon's single open/sealed state. Its word therefore opens or seals
the dungeon from either side. Doom's entrance exists only in the Underworld.
Redux accepts the word on every plane where that dungeon has an entrance.

**Implementation:** `internal/game_state/dungeon_words_of_power.go`

### WORD-0002

**Shared canonical-token matching and its overlap limit**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Intentional deviation

Original matching searches the complete expected canonical word/name/mantra
inside the submitted response, case-insensitively. Partial canonical words do
not suffice. Meditation's expected virtue tokens are deliberately four letters,
not full-name aliases. After a partial mismatch the assembly matcher skips by
matched length plus one, potentially missing overlapping candidate starts.
Therefore not every arbitrary response containing the term must succeed, and no
modern whitespace-normalization contract is established.

**REDUX DEVIATION: WORD-0002:** Redux normalized substring helpers accept complete
tokens without reproducing skipped overlaps. Predictable matching is deliberately
retained rather than rejecting valid containing text because of a partial prefix.
This is not exact all-input original equivalence.

**Implementation:** `internal/game_state/shrine_data.go`

### WORD-0003

**Outdoor Yell dispatch and candidate priority**

**Kind:** rule
**Uses:** WORD-0001, WORD-0002, SHRINE-0005, SHIP-0001
**Original:** Established
**Redux:** Unverified

Scan words ascending. For the first recognized word, announce/earthquake first,
then choose the first accepted adjacent tile west,south,east,north: indexed open
entrance, any sealed entrance, or any ruined shrine. No candidate gives failure.
A ruin invokes restoration and returns. Otherwise matching indexed coordinates
toggle the dungeon-state high bit and XOR open/sealed terrain, invalidating sight.
Even a coordinate mismatch marks the attempt effective, suppressing final failure
without mutation; an earlier wrong candidate blocks a later correct direction.

Ordinary input capacity is 30. Maps 1–32 dispatch town names; map 0 outdoor words;
other contexts do not activate outdoor words. Ship form below map ID 128 toggles
sails before text. Nonempty outdoor Yell retains consuming completion; ship/empty/
some town failures have valueless integer returns. No learned-word prerequisite
exists.

Redux retains blind correct Yell and cardinal priority, and accepts a dungeon's
word on each plane holding its entrance ([WORD-0001](#word-0001)). Normalized
matching, safe shrine dispatch, modern failure feedback and Escape differ.
Feedback/cancellation are existing presentation choices; undefined-return
equivalence remains unverified.

Successful Keep summoning is the explicit no-normal-cycle result in
[SHADOW-0004](SHADOWLORDS.md#shadow-0004). Its input workflow does not enqueue
deferred ordinary completion after appearance; failures/cancellation retain
their separately established Redux costs.

For the frigate toggle, Redux stops automatic sailing before showing `FURL!`,
preserves facing, and leaves subsequent movement to fresh manual input. Hoisting
alone does not restart automatic sailing; a fresh direction does. This does not
resolve the original sail-toggle completion-cost uncertainty in
[SHIP-0001](SHIP_MECHANICS.md#ship-0001).

**Implementation:** `internal/game_state/dungeon_words_of_power.go`, `internal/game_state/action_yell.go`

### WORD-0004

**Redux word discovery and entrance observation**

**Kind:** rule
**Uses:** WORD-0001, ASTRO-0006, VIEW-0006
**Original:** Incomplete
**Redux:** Unverified

Original NPC clue payloads were not independently decoded in this review. These
are current Redux discovery producers, not new claims about original TLK records:

| Producer / response boundary | Player-visible discovery |
| --- | --- |
| Malifora WORD | FALLAX / Deceit |
| Annon question 05, y | VILIS / Despise |
| Goeth question 01, drat | Destard and spoken AIPONI |
| Felespar question 05, dawn | MALUM / Wrong |
| Fiona question 08, word | AVIDUS / Covetous |
| Sindar WORD | INFAMA only |
| Hassad question 04, y | IGNAVUS / Hythloth |
| Acknowledged Codex word page | VERAMOCOR, without an unspoken Doom label |

First discovery creates one dated, source-qualified knowledge/Chronicle entry;
repetition preserves metadata and emits no duplicate notice. Learning neither
opens a dungeon nor changes shrine progress nor adds a Yell prerequisite. Talk
patches commit when displayed output reaches the scalar discovery action.
Goeth's accepted answer establishes Destard; AIPONI remains a Spoken clue until
successful INOPIA use confirms it. Sindar's separate naming response does not
retroactively name his word note. Hidden persistence keys do not authorize names
in display or search.

Look at an unobscured canonical open/sealed exterior records name, world, and
observed seal state without teaching its word. Wrong artwork/world/occupants do
not disclose by proximity. Surface/Underworld memories are independent. Automatic
surveying is the existing ASTRO-0006 extension: Look samples before its minute,
Enter retains source and commits after transition, known-identity Yell samples
after toggle. Pre-transition cancellation grants nothing; afterward keeps committed
observation. Doom Enter/blind Yell retain disclosure exclusions. Unmeasured sites
show Not measured, with no internal coordinates in readers; revisits preserve
genuine old surveys. Tool acquisition, clock/read/load changes never backfill.

**Implementation:** `internal/game_state/player_knowledge.go`, `internal/game_state/journal_dungeons.go`
**Tests:** `internal/game_state/word_of_power_knowledge_test.go`

### WORD-0005

**Redux Codex acknowledgement and complete-record persistence**

**Kind:** rule
**Uses:** WORD-0004, SHRINE-0003
**Original:** Incomplete
**Redux:** Unverified

Current Redux reveals the eighth word only after message page 3 has been displayed
and acknowledged, before page 4, with Codex source provenance. Quest completion
or workflow construction alone is insufficient. Cancellation before this boundary
grants nothing; repeated visits are idempotent and may fill missing historic
knowledge on a real revisit.

After final page 4 acknowledgement, enrich the same Chronicle event as complete
Codex revelation, retaining initial date/source/identity and all four clue paragraphs
with responsive wrapping. Page-turn announcement is not a clue. No unspoken Doom
name, entrance association, or coordinate is added; search uses visible clue text,
not hidden IDs. Word-only knowledge remains if interrupted before completion.
First completion marks the existing event updated; repeats are silent. Native v24
persists `codexRevelationComplete` independently of quest/word flags; older native/
legacy imports start incomplete. Browsing never completes it. Original narrative
payload and exclusive revelation claims remain unverified here.

**Implementation:** `internal/game_state/codex_revelation.go`, `cmd/ultimav/gamescene_shrine.go`
