# Shadowlords

Assignments, automatic town influence, physical actors, last Yell summon, and
persistent destruction are separate state. Random requests below do not certify
the original generator or equivalence of seeded patterns across implementations.

### SHADOW-0001

**Identity and midnight assignment**

**Kind:** rule
**Uses:** TIME-0001
**Original:** Established
**Redux:** Matches

Indexes: 0 Faulinei opposes Truth/Falsehood shard; 1 Astaroth opposes Love/Hatred;
2 Nosfentor opposes Courage/Cowardice. Generator assignments start zero. Daily
reassignment processes state below 128; destruction stores FF hex. At 23→0,
visit identities in order and retry R(1,8) until candidate differs from current
map and **every currently stored assignment, including its own previous town**.
Earlier identities now have new towns; later ones retain old values during testing.
Redux preserves this inspected rejection/order contract, not the original RNG stream.

**Implementation:** `internal/game_state/shadowlord_schedule.go`

### SHADOW-0002

**Automatic presence and source-floor vegetation**

**Kind:** rule
**Uses:** SHADOW-0001
**Original:** Established
**Redux:** Intentional deviation

Clear automatic presence, skip selection at player Y=4, otherwise choose first
assignment matching the town. Apply terrain effects, then create a physical lord
if none is loaded, X=15,floor 0,action 6 in every schedule slot. Spawn Y by town:
Moonglow 4, Britain 9, Jhelom 15, Yew 8, Minoc 17, Trinsic 10, Skara Brae 11,
New Magincia 10. Dead-NPC removal follows initialization; announcement/influence
requires a living party result.

Vegetation seeds original RNG from day-of-month, scans Y-major then X-major,
draws only for fruit trees/crops, and converts tree→stump/crop→plowed when R(0,7)
is nonzero; afterward it reseeds from wall time. Floor loading first reads fresh
1,024-byte terrain, then effects and optional NPC placement. Stairs/ladders retain
presence; midnight changes assignments/day, not loaded presence/terrain. Same-day
source regeneration repeats its pattern; later day can differ. Re-entry without
presence restores healthy source terrain.

**REDUX DEVIATION: SHADOW-0002:** Redux isolates the day-seeded effect stream so
shared gameplay RNG is unchanged. It rebuilds source floor before one pass, avoiding
compounded destruction; floor transitions retain controller/influence/physical
identity without replaying entry announcements. No-presence, jail-bypass, and
ritual-only summons do not acquire vegetation effects. Same-day reproducibility
does not prove byte-identical patterns with the unrecovered original generator.

**Implementation:** `internal/game_state/shadowlord_spawn.go`
**Tests:** `internal/game_state/shadowlord_floor_effects_test.go`

### SHADOW-0003

**NPC influence and separate theft paths**

**Kind:** rule
**Uses:** SHADOW-0002, ENV-0001
**Original:** Established
**Redux:** Intentional deviation

Faulinei enables conversation theft and separate shop gold theft. Astaroth makes
affected NPCs attack with talk FE hex; its response damages party then flees.
Nosfentor makes them flee with talk FD. Affected filtering examines nonzero
schedule times and ordinary-NPC shape, then consumes R(0,1), nonzero disqualifying.
The inspected original mistakenly reads shape entry 4 for every candidate, and
still draws for already-disqualified candidates.

Conversation theft reseeds from wall time, retries Keys/Gems/Torches until a
nonempty category loses one; otherwise removes highest occupied equipment, then
potion, then scroll index; otherwise loses R(1,15) gold floored zero. Shop theft
separately takes R(1,64) for automatic identity zero. A misleading original comment
does not change the executable Faulinei selection.

**REDUX DEVIATION: SHADOW-0003:** Redux tests each candidate's own shape rather
than the erroneous shared index and keeps gameplay randomness controlled. Its
conversation and shop authorities remain distinct. Full malformed schedule and
missing-argument original outcomes are not invented.

**Implementation:** `internal/game_state/shadowlord_spawn.go`

### SHADOW-0004

**Yell summon and pursuit**

**Kind:** rule
**Uses:** SHADOW-0001, WORD-0002
**Original:** Established
**Redux:** Unverified

Only Lycaeum/Empath Abbey/Serpent's Hold maps 30–32 qualify. Match the complete
canonical name inside response; require Y>=2, identity state not FF, and no loaded
Shadowlord-tile object. Success stores last summon, creates it at party X,Y−2 on
current floor with action 6, and reports no ordinary completion. It does not
install automatic presence or terrain/personality effects. Several failure returns
do not establish a reliable original time result.

Action 6 attracts only below Manhattan distance four and otherwise does nothing;
action 7 attracts without that gate but has random variation. Adjacency precedes
movement; ordinary pursuit picks the first legal cardinal distance-reducing step.
Blocked fallback contains a source-array hazard, so intended greedy behavior is
not a complete runtime specification. Redux separates summoned/automatic actors
and validates placement; unusual collision/failed insertion equivalence is unverified.

**Implementation:** `internal/game_state/shadowlord_spawn.go`

### SHADOW-0005

**Shard ritual and Stonegate boundaries**

**Kind:** rule
**Uses:** SHADOW-0001, ENV-0002, INV-0006
**Original:** Established
**Redux:** Unverified

| Identity/shard | Map | Player X,Y | Floor | Flame | Stonegate persistence bit |
| --- | --- | --- | ---: | --- | --- |
| Faulinei/Falsehood | Lycaeum 30 | 15,9 | 2 | Truth | 02 hex |
| Astaroth/Hatred | Empath Abbey 31 | 15,3 | 1 | Love | 04 |
| Nosfentor/Cowardice | Serpent's Hold 32 | 15,16 | FF/-1 | Courage | 08 |

Present selected shard/sound; check exact site before consumption; present flame
and three earthquakes; require matching last summon and physical lord exactly
one tile north. Success removes NPC, performs seven impacts, stores destroyed FF,
clears shard, sets persistence bit, and presents victory. Late identity failure
retains shard without another failure message. Ordinary combat death does not
perform this persistent destruction.

Stonegate announces living identities in reverse order but that branch does not
create actors or apply personality effects; authored population is unverified.
Held Sceptre removes NPC 9 on initialization; physical-Shadowlord combat reclaims
it. Prepared Cast absorption precedes resources. Only the independently checked
Time Stop scroll rejects exact Stonegate/Doom after consumption; do not generalize
to all magical items or uninspected combat-origin routes. Trapdoor wipe is ENV-0002.
Redux implements these separate state authorities; a blanket all-path Match is
not established by this conversion.
