# Shrines and Codex

Karma rewards/offerings retain their existing canonical rules. Meditation,
restoration, answer completion, and knowledge disclosure have separate boundaries.

### SHRINE-0001

**Indexed shrine identity**

**Kind:** rule
**Uses:** WORD-0001, MOON-0005, KARMA-0009
**Original:** Established
**Redux:** Unverified

| Index | Virtue | Mantra | World X,Y |
| ---: | --- | --- | --- |
| 0 | Honesty | Ahm | 233,66 |
| 1 | Compassion | Mu | 128,92 |
| 2 | Valour | Ra | 36,229 |
| 3 | Justice | Beh | 73,11 |
| 4 | Sacrifice | Cah | 205,45 |
| 5 | Honor | Summ | 81,207 |
| 6 | Spirituality | Om | 0,0 sentinel |
| 7 | Humility | Lum | 231,216 |

Restoration word is the same index in WORD-0001. Meditation chooses coordinate
match then defaults to Spirituality; the sentinel is not a normal walkable
location or proof of its access route. Reward stats/karma belong to KARMA-0009.

### SHRINE-0002

**Meditation prompts and quest state**

**Kind:** rule
**Uses:** SHRINE-0001, WORD-0002, KARMA-0009, KARMA-0010
**Original:** Established
**Redux:** Intentional deviation

Original virtue keys, by index, are **hone, comp, valo, just, sacr, hono, spir,
humi**, not full names. Read up to 12 bytes for virtue and for each of three
mantras. Empty submission exits immediately. A nonempty mismatch records failure
but does not skip later mantra prompts; reject after all answers if any mismatch.
Full mantras use the shared matcher qualifications in WORD-0002.

If completion bit is clear, set active and assign/reassign quest. If complete
but inactive, offer (KARMA-0010). If both set, clear active and reward (KARMA-0009).
Redux preserves these main branches and four-letter virtue matching, but rejects
wrong nonempty virtue before mantra entry. **REDUX DEVIATION: SHRINE-0002:** immediate
rejection avoids asking for mantras after the visit has already failed.
Escape immediately cancels in Redux; original line editing clears
text then requires Return. Wrong nonempty mantras still collect three attempts.
Unaffordable offerings repeat within the same visit without deduction.

**Implementation:** `internal/game_state/shrine_data.go`, `internal/game_state/shrine_service.go`, `cmd/ultimav/gamescene_shrine.go`

### SHRINE-0003

**Codex answer selection and revelation**

**Kind:** rule
**Uses:** SHRINE-0002
**Original:** Established
**Redux:** Unverified

Choose lowest active quest bit, with no progression when none exists. Set its
completion bit **before** answer display; do not clear active. Already-completed
active bits remain selectable until shrine return clears them. Completion byte
FF invokes final revelation without a separate one-time flag. Memorial enumeration
uses permanent-loss companion markers, not resurrection.

Message dispatch/offsets are established, including Compassion at MISCMSG.DAT
1273; actual prose/newlines, opposed-vice wording, final clues, and exclusive
narrative word revelation were not independently decoded here. Redux keeps its
existing answer/revelation content; its Compassion sentence is formatted for
responsive wrapping rather than preserving embedded line breaks. Word knowledge
and acknowledged full-revelation persistence are separately specified in WORD-0004.

### SHRINE-0004

**Visit time and guardian checkpoint**

**Kind:** rule
**Uses:** TIME-0001, TIME-0002, SHRINE-0002
**Original:** Established
**Redux:** Unverified

After restoring world state, shrine/Codex scene requests 16 minutes; completed
outdoor Enter requests another two. Eighteen is the ordinary unmodified total,
not universal under Quickness/Time Stop. The consumed outdoor turn checks map 0,
level 0,(233,235): any low-byte active quest permits passage; none pushes south
one tile. This is an else-if after bridge/swamp/lava handling, not a universal
ungated coordinate trigger. The branch has no sound call.
Redux commits its guardian check synchronously on consumed surface turns, including
nonmovement actions. Complete malformed-state and all terrain-precedence parity
remain unverified; atlas discovery is separately VIEW-0006.

### SHRINE-0005

**Ruined state and restoration input**

**Kind:** rule
**Uses:** SHRINE-0001, WORD-0002, KARMA-0013
**Original:** Established
**Redux:** Intentional deviation

Closure above 127 replaces shrine terrain with an impassable ruin during world
load; Enter does not meditate there. Correct Blackthorn disclosure owns the ruin
consequence (KARMA-0013). Word recognition chooses adjacent candidates west/south/
east/north. Restoration requests full canonical virtue, then three full mantras,
up to 15 bytes each; unlike meditation, empty answers do not exit early. Require
all matches and indexed shrine coordinate. Success clears only bit 80 hex,
restores terrain and invalidates visibility; failure emits spacing, not meditation
rejection prose. Gameplay editor Escape clears text; Return submits, Backspace/
Left erases. Creation editor behavior is not inferred from this routine.

**REDUX DEVIATION: SHRINE-0005:** Redux retains safe dispatch, explicit unfocused
feedback, longer modern input capacity, early stop and immediate Escape cancellation,
honoring its visible Cancel control without later prompts or restoration. Original
meditation/restoration contains no Dead-member revival.
