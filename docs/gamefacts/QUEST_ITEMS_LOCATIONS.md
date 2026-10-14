# Quest and special items

Original quantities, singleton bytes, operational bits, and discovery flags are
not one contiguous fourteen-item collection. Generator starts and direct placements
must remain distinct from uninspected authored map/NPC/TLK acquisition.

### QUEST-0001

**Storage and source-defined starting state**

**Kind:** rule
**Uses:** SAVE-0003, INV-0008
**Original:** Established
**Redux:** Unverified

Amulet, Crown, Sceptre, three named shards, Spyglass, Watch, Badge, Box, and Sextant
use singleton-style fields; ordinary grants store FF hex when held. Carpets and
Grapples are quantities. Plans mix presence and operational high-bit state. The
generator starts Watch held and these other listed items zero, not verified shipped
INIT.GAM bytes. Four shard bytes are allocated despite three named identities.
Redux IDs 0–13 order Amulet, Crown, Sceptre, three shards, Carpet, Grapple, Spyglass,
HMS Cape, Watch, Badge, Box, Sextant; that model is not the original save structure.

### QUEST-0002

**Fixed placements and acquisition limits**

**Kind:** rule
**Uses:** QUEST-0001, SHADOW-0001
**Original:** Incomplete
**Redux:** Unverified

On nonzero world level, original fixed objects are written at encoded level FF:

| Item | X,Y | Slot/object number, hex | Presence |
| --- | --- | --- | --- |
| Amulet | 105,225 | 1C/F3 | Not owned |
| Falsehood shard | 192,80 | 1D/F0 | Not owned and identity 0 state below 128 |
| Hatred shard | 130,65 | 1E/F1 | Not owned and identity 1 state below 128 |
| Cowardice shard | 176,184 | 1F/F2 | Not owned and identity 2 state below 128 |

Pickup identifies Amulet by B7 hex and shards by B4 plus low-two object-number bits.
Other evidence is narrower: held Sceptre removes Stonegate NPC 9; Castle Britannia
carpet pickup removes NPC 22; Box pickup sets castle NPC-31 persistence. Crown
pickup records/removes its source NPC but does not identify its exact placement.
Talk grants quantities of Grapple/Carpet capped 99 or held Sextant/Spyglass/Badge;
special plans pickup writes FF. Granting conversations and exact remaining sites
require authored data.

Redux places Crown at Palace (15,13),floor 3 and Sceptre at Stonegate (15,15),floor
0. Its Amulet/shard coordinates match the table at Underworld floor -1. Do not
infer original Crown/Sceptre coordinates from that implementation table.

**Implementation:** `internal/game_state/quest_item_placement.go`

### QUEST-0003

**Artifact competition and Sceptre safety**

**Kind:** rule
**Uses:** QUEST-0001, SHADOW-0005, VEH-0004, SHIP-0002, ASTRO-0003, TIME-0005
**Original:** Established
**Redux:** Intentional deviation

Original Amulet/Crown/Badge share the active-effect slot with duration spells.
Reusing the active artifact clears slot/duration; another replaces both with
duration 255. No replaced effect resumes. Sceptre outside dungeon IDs scans the
centered 3×3 in X-major order, changing barrier tiles 70–7F hex to grass. If no
replacement occurs, it unconditionally calls field removal; that routine's
noncombat branch can index unrelated/out-of-range dungeon storage with surface
coordinates. Not every such coordinate necessarily lies outside storage.

**REDUX DEVIATION: QUEST-0003:** Redux keeps Badge/Amulet independent for concurrent
artifact use; Palace guards use Badge state. Crown competes with Protection,
Quickness, Confusion, Negate Magic, and Time Stop; recast refreshes, removal/expiry
never resumes a replaced effect. Independent native artifact fields remain; this
does not implement any missing Amulet effect. Redux suppresses unsafe surface
Sceptre fallback and reports no effect; dungeon/combat removal stays context-owned.
Shard consumption is SHADOW-0005; Carpet, plans, instruments, Watch use their owning
rules. Box has no operative Use effect in the inspected original handler.

Owned Crown, Amulet, and Black Badge are actionable through combat Use: confirmed
wear/remove retains the artifact and completes exactly one current actor action.
Selecting the Crown preserves its duration-effect replacement rules and its
protection against enemy magical abilities, including acquisition of possession.
This is prevention, not a cure of an already active possession episode. Inventory,
context, and exact encounter/actor ownership are revalidated before activation;
stale or unavailable selections are free under CONSUME-0001.

**Implementation:** `internal/game_state/duration_effect.go`, `internal/game_state/action_use.go`, `cmd/ultimav/gamescene_use.go`
**Tests:** `internal/game_state/combat_artifact_use_test.go`, `cmd/ultimav/gamescene_combat_artifact_use_test.go`
