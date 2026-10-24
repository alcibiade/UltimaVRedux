# Creatures and encounters

The original 48-entry creature index includes humanoid archetypes, named characters,
restrained shapes, fields, and an environmental Whirlpool—not just hostile monsters.
These are semantic descriptions of inspected material labelled 1.16, without an
immutable corpus fingerprint or verified retail-build identity. R(a,b) records
requested bounds. Original probabilities below are conditional on uniform inclusive
draws, not claims about the unavailable generator or C argument evaluation order.

### ENEMY-0001

**Complete creature catalog and ability meanings**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Unverified

STR/DEX/INT are base attributes; Arm is armor, Dmg fixed monster damage rating,
HP initial/maximum HP, Num number-appearing bound (not world population cap), Tr
treasure rating. Mode is a hexadecimal ability word. Range, missile type, and
friend candidate are separate verified fields. Table membership never guarantees
spawn eligibility. The party's persistent records are not these humanoid rows.

| Index | Semantic identity | STR | DEX | INT | Arm | Dmg | HP | Num | Tr | Mode | Range | Missile | Friend |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| 0 | Mage | 10 | 15 | 20 | 0 | 15 | 10 | 3 | 20 | 8000 | 7 | Blue | Skeleton |
| 1 | Bard | 15 | 20 | 10 | 4 | 12 | 15 | 9 | 10 | 4000 | 3 | Arrow | Bard |
| 2 | Fighter | 20 | 15 | 10 | 8 | 15 | 20 | 6 | 15 | 0000 | 1 | Arrow | Bard |
| 3 | Avatar archetype | 25 | 25 | 25 | 7 | 30 | 20 | 1 | 25 | 0000 | 1 | Arrow | Avatar |
| 4 | Villager | 12 | 12 | 12 | 0 | 6 | 8 | 1 | 10 | 0000 | 1 | Arrow | Villager |
| 5 | Merchant | 12 | 12 | 18 | 0 | 6 | 8 | 1 | 10 | 0000 | 1 | Arrow | Villager |
| 6 | Jester | 12 | 18 | 12 | 0 | 6 | 8 | 1 | 10 | 0000 | 1 | Arrow | Villager |
| 7 | Banjo-player shape; displayed Bard | 12 | 16 | 14 | 0 | 6 | 8 | 1 | 10 | 0000 | 1 | Arrow | Villager |
| 8 | In Stocks; no singular name | 12 | 12 | 12 | 0 | 0 | 5 | 1 | 0 | 0000 | 1 | Arrow | Villager |
| 9 | In Manacles; no singular name | 12 | 12 | 12 | 0 | 0 | 5 | 1 | 0 | 0000 | 1 | Arrow | Villager |
| 10 | Child | 8 | 8 | 8 | 0 | 0 | 5 | 1 | 0 | 0000 | 1 | Arrow | Child |
| 11 | Beggar | 8 | 8 | 8 | 0 | 0 | 5 | 1 | 0 | 0000 | 1 | Arrow | Villager |
| 12 | Guard | 22 | 30 | 10 | 6 | 30 | 99 | 8 | 5 | 0000 | 15 | Axe | Guard |
| 13 | Wanderer | 30 | 30 | 30 | 30 | 99 | 99 | 1 | 0 | B008 | 9 | Blue | Wanderer |
| 14 | Blackthorn | 30 | 30 | 30 | 30 | 30 | 99 | 1 | 0 | BC48 | 9 | Red | Blackthorn |
| 15 | Lord British | 30 | 30 | 30 | 30 | 99 | 99 | 1 | 0 | B008 | 9 | Blue | Lord British |
| 16 | Sea Horse | 17 | 20 | 20 | 2 | 10 | 30 | 3 | 0 | C001 | 5 | Blue | Squid |
| 17 | Squid | 24 | 20 | 8 | 0 | 20 | 50 | 2 | 0 | 4005 | 7 | Blue | Sea Horse |
| 18 | Sea Serpent | 17 | 17 | 8 | 2 | 30 | 70 | 1 | 0 | 4001 | 9 | Red | Squid |
| 19 | Shark | 20 | 17 | 5 | 0 | 8 | 22 | 10 | 0 | 0001 | 1 | Arrow | Shark |
| 20 | Giant Rat | 5 | 20 | 5 | 0 | 6 | 10 | 10 | 5 | 0300 | 1 | Arrow | Skeleton |
| 21 | Bat | 5 | 30 | 5 | 0 | 6 | 5 | 16 | 0 | 0001 | 1 | Arrow | Bat |
| 22 | Giant Spider | 10 | 10 | 5 | 0 | 8 | 10 | 4 | 5 | 0204 | 1 | Arrow | Giant Rat |
| 23 | Ghost | 1 | 20 | 10 | 0 | 12 | 20 | 6 | 0 | 0821 | 1 | Arrow | Skeleton |
| 24 | Slime | 6 | 6 | 2 | 0 | 4 | 10 | 16 | 0 | 0011 | 1 | Arrow | Slime |
| 25 | Gremlin | 10 | 21 | 10 | 2 | 4 | 10 | 13 | 12 | 0002 | 1 | Arrow | Mimic |
| 26 | Mimic | 20 | 30 | 12 | 3 | 15 | 30 | 1 | 20 | 0280 | 2 | Green | Ettin |
| 27 | Reaper | 20 | 25 | 12 | 4 | 20 | 40 | 3 | 25 | 8080 | 9 | Blue | Bat |
| 28 | Gazer | 8 | 10 | 25 | 0 | 10 | 20 | 4 | 0 | 8040 | 5 | Violet | Bat |
| 29 | Crawler | 17 | 15 | 12 | 0 | 15 | 35 | 4 | 0 | 0000 | 1 | Arrow | Slime |
| 30 | Gargoyle | 20 | 10 | 5 | 15 | 20 | 40 | 1 | 0 | 0090 | 9 | Rock | Gargoyle |
| 31 | Insect Swarm | 1 | 30 | 1 | 0 | 4 | 5 | 10 | 0 | 0001 | 1 | Arrow | Slime |
| 32 | Orc | 15 | 13 | 10 | 2 | 12 | 10 | 10 | 11 | 0080 | 1 | Arrow | Troll |
| 33 | Skeleton | 10 | 20 | 5 | 0 | 12 | 20 | 8 | 13 | 0020 | 1 | Arrow | Mage |
| 34 | Python | 5 | 18 | 8 | 1 | 8 | 10 | 4 | 0 | 0004 | 3 | Green | Giant Spider |
| 35 | Ettin | 20 | 15 | 12 | 3 | 15 | 30 | 6 | 17 | 4080 | 5 | Rock | Headless |
| 36 | Headless | 19 | 12 | 8 | 2 | 12 | 20 | 8 | 12 | 0080 | 1 | Arrow | Ettin |
| 37 | Wisp | 8 | 30 | 20 | 0 | 20 | 40 | 4 | 0 | 2041 | 1 | Arrow | Ghost |
| 38 | Daemon | 25 | 25 | 25 | 5 | 20 | 75 | 4 | 0 | 8461 | 9 | Red | Dragon |
| 39 | Dragon | 30 | 25 | 25 | 10 | 30 | 99 | 2 | 30 | 0400 | 9 | Red | Dragon |
| 40 | Sand Trap | 25 | 25 | 5 | 10 | 30 | 80 | 1 | 25 | 0000 | 1 | Arrow | Sand Trap |
| 41 | Troll | 18 | 17 | 9 | 4 | 15 | 15 | 4 | 15 | 4000 | 5 | Axe | Giant Rat |
| 42 | Force Field; no singular name | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0000 | 1 | Arrow | Force Field |
| 43 | Whirlpool; no singular name | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0000 | 1 | Arrow | Whirlpool |
| 44 | Mongbat | 10 | 30 | 15 | 4 | 20 | 20 | 16 | 5 | 0000 | 1 | Arrow | Mongbat |
| 45 | Corpser | 17 | 10 | 8 | 0 | 15 | 40 | 4 | 0 | 0000 | 1 | Arrow | Corpser |
| 46 | Rot Worm | 5 | 17 | 6 | 0 | 6 | 5 | 10 | 0 | 0300 | 1 | Arrow | Giant Rat |
| 47 | Shadowlord | 25 | 30 | 30 | 10 | 30 | 99 | 1 | 0 | BB61 | 9 | Red | Daemon |

Missile encoding is Arrow 0, Cannonball 1, Axe 2, Red 3, Blue 4, Green 5,
Violet 6, Rock 7. Range one is melee-only: an Arrow table entry at range one
does not establish that the creature fires arrows.

| Hex bit | Semantic ability |
|---:|---|
| 8000 | Magic-use gates for ranged attacks |
| 4000 | Normal ranged-attack label |
| 2000 | Teleport |
| 1000 | Disappear on death |
| 0800 | Tactical invisibility |
| 0400 | Gate Daemons |
| 0200 | Poison |
| 0100 | Plague label |
| 0080 | Bludgeon, selecting STR for hit checks |
| 0040 | Possession/charm |
| 0020 | Undead |
| 0010 | Divide after surviving damage |
| 0008 | Immortal, zero damage |
| 0004 | Ranged poison |
| 0002 | Food theft |
| 0001 | No corpse |

The poison consumer tests either 0200 or 0004; the no-corpse consumer tests either
1000 or 0001 and separately distinguishes disappearance presentation. No independent
plague or extra normal-range-bit consumer was established. Those labels must not
be promoted to another disease or attack. Null singular names for 8, 9, 42, 43
remain absent; descriptive labels above are not invented original display strings.
Redux convenience flags are derived behavior, not extra original columns. The
complete table is original evidence; this rule does not certify every Redux row.

### ENEMY-0002

**Creature encoding, terrain partition, and applicability**

**Kind:** rule
**Uses:** ENEMY-0001
**Original:** Established
**Redux:** Matches

A four-frame creature family normally starts at 0x40+4×index; clear the low two
bits to recover its base. This encoding is contextual: map-plane terrain and
combat object descriptors can use the same bytes differently. World creature
recognition excludes sub-0x80 shapes except pirates, the artifact family 0xB4–B7,
and field family 0xE8–EB. A Crawler-shaped artifact is not a roaming Crawler enemy.

World and tactical terrain admission share these shape classes, with occupancy
checked separately. This partition includes all 48 even when a row never moves:

| Class | Indexes | Terrain |
|---|---|---|
| On foot | 0–12, 14, 20, 22, 24–27, 29–37, 41 | Foot-passability bitmap; chairs only for Player or humanoids 0–3. |
| Water | 16–19 | Terrain below 4 or River family 0x60–6F. |
| Flying | 13, 15, 21, 28, 38, 39, 44 | Water or foot admission, retaining shape-specific chair restriction. |
| Ether | 23, 47 | Anything outside water predicate, including walls; occupancy still blocks. |
| Desert | 40 | Exact tile 7 only. |
| Rejected | 42 | No terrain. |
| Deep water | 43 | Exact tile 1 only. |
| Grass | 45 | Exact tile 5 only. |
| Swamp | 46 | Exact tile 4 only. |

Redux shares the original-byte predicate between enemy references and tactical
terrain, independent of expanded sprite indexes. World reads map-only terrain
including overrides, then occupancy and destination slowdown. Tactical callers
add their own off-map/Nothing/placement rules. Pirate vehicles use Deep Water/Sea,
not shoals/rivers. Daemon gates test the created Daemon, not a default Mage.

Applicability details prevent table labels from becoming universal behavior:

- 0–3 are humanoid archetypes; Mage has magic-use suppression/Turning gates.
  4–7, 10–11 are authored humanoids; Banjo Player also has a separate camp path.
  8–9 are restrained shapes, not pirates or ordinary generated enemies.
- 12 Guards have a town-group exception. 13/15 are immortal teleporting/disappearing
  humanoids; camping Wanderer presentation is not a hostile-spawn rule. 14 Blackthorn
  also possesses, conceals, and gates; authored admission is separate.
- 16–19 are water/no-corpse creatures; Squid poisons, Sea Horse has magic-use gates,
  and Sea Serpent additionally has outdoor fireballs independent of tactical damage.
- 20/46 share poison/plague labels but differ in Rat foot versus Rot Worm swamp
  terrain; no plague effect is invented. 21/44 fly and bypass certain world slowdowns.
  22/34 use compound poison. 23 is ether/undead/invisible/no-corpse.
- 24/30 divide; Gargoyle bludgeons and becomes Small Rocks on death. 25 steals food.
  26/27 are stationary tactical bludgeoners; Mimic always attempts its in-range
  missile branch and poisons, Reaper has magic-use gates. Crawler 29 is not stationary.
- 28 flies, possesses, may sleep victims, and produces an Insect on death. 31 is
  on-foot despite artwork and leaves no corpse. 32/36 use STR-based melee;
  33 is undead foot melee. 35/41 have physical ranged attacks; only Ettin bludgeons.
- 37 Wisp is foot terrain, teleporting/possessing/no-corpse, not water-flying.
  38/39 fly and gate; Daemon additionally possesses, uses magic, is undead/no-corpse,
  and retires after successful possession. Dragon has outdoor fireballs.
- 40 has exact-desert spawn/terrain gates. 42 is a non-actor descriptor; actual
  fields use field mechanics. 43 is a deep-water environmental contact/transport
  entity, not a zero-stat tactical enemy.
- 45 Corpser is grass-only: exposure, capture after damaging surviving hits, and
  underground release differ from magical invisibility. 47 Shadowlord is ether,
  undead, poison, possession, invisibility, teleport, disappearance, and magic-use;
  it has **no gating bit** and does not retire on possession. Quest identity is separate.
- World pirate ships are outside the index; all facings enter Bard combat. Redux
  distinguishes their sprites despite a convenience reference reusing index 8.

**Implementation:** `internal/references/original_creature_terrain.go`, `internal/game_state/combat_terrain.go`
**Tests:** `internal/references/original_creature_terrain_test.go`, `internal/ai/npc_ai_controller_large_map_terrain_test.go`

### ENEMY-0003

**Initialization, groups, friends, and authored town admission**

**Kind:** rule
**Uses:** ENEMY-0001
**Original:** Established
**Redux:** Matches

Create monsters at full catalog HP. DEX is stored as an unsigned byte after
base DEX+R(0,7)−4; if that stored value exceeds 30, restore base DEX. Negative
arithmetic therefore wraps then resets; this is not symmetric clamping. Initial
countdown is 36−DEX. PCs initialize countdown from their own DEX.

Ungrouped non-Guard town encounters have one creature. Otherwise start with Num.
Bounds 1, 8, and 16 are fixed; other bounds request R(1,bound), and the opening-
campaign easy period requests R(1,result) again. A count exceeding available
ordinary source range is capped at 26. First placement is always primary. For
later zero-based positions i with i<floor(count/4)+1, R(0,8)=0 chooses its friend
candidate (conditionally 1/9, not 1/8). Other positions remain primary.

Draw order is primary creation DEX first, then each eligible friend draw immediately
followed by that entry's DEX. Do not pre-roll the complete friend list. Redux's
natural outdoor entry preserves this interleaving and original positions, including
coincident placements; its general debug composer additionally validates its own
requested positions/capacity. Scoped natural tests cover both worlds and pirate/Bard
entry, fixed bounds, and opening-campaign reduction.

Town type comes from the four-frame shape, independent of location/dialogue.
Attack excludes artifact/Crawler-shaped quest objects and environmental field/
whirlpool families. Authored grouping marks Yew slots 15/17, Minoc 1, Windemere
3–9, and Stonegate 5–8. Guards retain their eight-member exception. Hostile contact
still requires an applicable schedule action; shape eligibility alone is not hostility.
Iolo's Hut Rat slots 1–3 at (23,8),(13,5),(3,6), and Skara Brae Bat slots 8–12
at (15,15),(14,14),(16,14),(16,16),(14,16) use hostile action 6 without grouping,
so they enter as single matching creatures, not a location/species allowlist.

Shadowlord shape selects its special battlefield and reclaims the Sceptre.
Stonegate slots 1–3 retain individual quest death identities: ordinary combat
entry/return does not set the shard-death bits. Redux retains ordinary-authored-NPC
provenance, not automatic/Yell-summoned state, and rejects queued sources already
quest-dead. Fixed dungeon descriptors specify entities directly, without this
friend substitution. Generated dungeon draw order remains DNG-0010.

**Implementation:** `internal/game_state/combat_encounter_composer.go`, `internal/game_state/combat_monster.go`
**Tests:** `internal/game_state/combat_natural_rng_test.go`, `internal/game_state/combat_small_map_authored_test.go`, `internal/game_state/combat_small_map_shadowlord_test.go`

### ENEMY-0004

**World spawn candidates and frequency**

**Kind:** rule
**Uses:** ENEMY-0001
**Original:** Established
**Redux:** Intentional deviation

Select R(0,255) and subtract these ordered weights; each pool totals 256:

| Pool | Ordered semantic weights |
|---|---|
| Britannia land | Orc 60; Python 50; Rat 40; Spider 30; Insects 20; Skeleton 15; Headless 15; Troll 10; Ettin 10; Wisp 3; Dragon 2; Daemon 1 |
| Underworld land | Bat 64; Rat 56; Spider 56; Mongbat 32; Corpser 32; Daemon 8; Dragon 8 |
| Britannia water | Shark 72; Squid 72; Sea Serpent 40; Sea Horse 38; Pirate ship 34 |
| Underworld water | Squid 128; Sea Serpent 128 |

Water first requires R(0,64)<16 (conditionally 16/65). Britannia Deep Water then
has R(0,7)=7 Whirlpool selection. Spawn-water includes terrain 0–3, rivers 0x60–6F,
waterfalls 0xD4–D7, and corner waters 0xE4–E7, broader than movement-water.
Exact Desert 7 selects Sand Trap on one of four outcomes, otherwise none.
Underworld Swamp directly selects Rot Worm. Mountains/Peaks select none. After
these branches only terrain below 0x10 or Beach 0x30–33 enters a land pool;
0x50–53 is parapet, not beach. Roads/entrances have no generic walkable fallback.

Generation attempts require threshold > R(1,30), independent of candidate gates.
Read underlying map terrain at the party, not a vehicle/object overlay:

| Party terrain | Day threshold | Night threshold |
|---|---:|---:|
| Roads 0x20–26 | 0 | 3 |
| Swamp 4 or terrain 0x09–0F | 2 | 5 |
| Other | 1 | 4 |

Underworld always uses 3. Conditional uniform attempt probability is
max(threshold−1,0)/30, not threshold/30. The inspected hour test's evening branch
compares against 0x20 (32), unreachable for hours 0–23, so its night bonus is only
00:00–04:59. This is a source observation, not independently verified shipping
behavior. Level, party size, karma, month, and distance from Iolo's Hut do not
enter this threshold. The easy-period rule reduces eligible tactical group counts,
not world frequency. A night road bonus cannot make a road a legal spawn candidate.

Original creation samples the loaded 32×32 region and requires **both** wrapped
axis distances to exceed six. Redux preserves default thresholds/minimum separation
but deliberately samples corner portions of a perimeter outside its larger 19×13
viewport so new actors remain unseen. It does not scale odds with the viewport.
Workbench roll-maximum tuning changes nonzero rates; default remains 30. Live
map-only overrides prevent an abandoned vehicle altering either probability or draw.

**Implementation:** `internal/ai/npc_ai_controller_large_map.go`, `internal/references/monster_terrain.go`
**Tests:** `internal/ai/npc_ai_controller_large_map_generation_test.go`, `internal/references/monster_spawn_candidates_test.go`

### ENEMY-0005

**World pursuit, slowdown, and ambiguity**

**Kind:** rule
**Uses:** ENEMY-0002
**Original:** Conflicting
**Redux:** Intentional deviation

Original world pursuit has a randomized axis-first/other-axis/random fallback
structure, but recovered playerward/repulsion conditions use assignments where
comparisons would be expected. Another branch is labeled Mongbat while testing
shape 0xFC (Shadowlord), not Mongbat 0xF0 or Bat 0x94. These conflicts cannot
establish reliable retail Bat/Mongbat repulsion.

Destination slowdown allows movement on one of two outcomes for Swamp, Brush,
Desert, Thick Brush, Wasteland, and one of three for 0x09–0F. Dragon, Bat, Daemon,
and Mongbat bypass it. Contact is checked before movement, so becoming adjacent
does not immediately start a second combat check.

Redux deliberately uses ordinary pursuit for Bat/Mongbat, retaining flight and
slowdown exemption rather than inventing behavior from conflicting source.
Ordinary Redux movement chooses random motion on one in four phases, then waits
on one in four remaining directed opportunities, otherwise uses A* within a
30-tile square around the party, crossing the world's wrapping edge when needed. Thus Redux
probabilities are 25% wander, 18.75% wait, 56.25% directed pursuit. Destination
slowdown still applies to the chosen step. The wait offsets A* routing around
obstacles. Deterministic synthetic comparison requires at least 10% greater mean
time to contact and no increase within 16 phases versus an independent intended-
cardinal oracle across open, slow-terrain, and obstacle cases. That oracle does
not certify execution of the ambiguous assignments or a retail binary.

**Implementation:** `internal/ai/npc_ai_controller_large_map.go`
**Tests:** `internal/ai/npc_ai_controller_large_map_movement_test.go`, `internal/ai/npc_ai_controller_large_map_pressure_test.go`

### ENEMY-0006

**Outdoor Sea Serpent and Dragon fireballs**

**Kind:** rule
**Uses:** ENV-0001
**Original:** Established
**Redux:** Matches

Use wrapped 256-tile axis distances. Cardinal adjacency is handled first as contact
without a projectile roll. Otherwise Sea Serpent and Dragon request R(0,7)=0 when
both axis distances are at most three: this is square range, including (1,1),
(2,1), and (3,3), without same-axis, vehicle, target-terrain, or world requirements.
Underworld water can therefore fire too. Carpet reduces actor cadence, not immunity.

A carpet/skiff rider on base terrain below Swamp receives direct contact damage
instead of tactical entry—Deep Water, Sea, Shoals, not every water-like tile.
Diagonal adjacency can still fire. Rough-seas damage on entering Deep Water is
a separate movement hazard.

World phase gates Time Stop, alternating Quickness, then alternating horse/carpet
calls, including Pass. Generation precedes descending actors 31→1. Each actor
tests attack before movement. Any attack suppresses subsequent movement but not
later attack checks; obstruction still counts as an attack. Moving adjacent adds
no same-phase second check. A negative test on the unsigned ship-plans value is
not an effective additional gate.

Only unblocked hits damage: each living active member takes its own R(1,8), or a
frigate takes R(1,30) hull and sinks on equality. Sinking prefers carried skiff,
then carpet, otherwise drowning. Tactical damage ratings do not apply. Launch,
fireball travel, and impact feedback are separate from pirate-only cannon text.

Redux interleaves descending checks/movement and commits a hit before the next
actor's draw. Detached results enter the shared FIFO world-projectile workflow;
animation callbacks never roll/apply damage, and cleanup cannot replay a hit.
Pirate warning/rotation retains its separate ship policy. The match is these
mechanisms, not original PRNG/cosmetic-draw sequence parity.

**Implementation:** `internal/game_state/outdoor_fireball.go`, `internal/ai/npc_ai_controller_large_map.go`

### ENEMY-0007

**Outdoor projectile obstruction uses the source display buffer**

**Kind:** rule
**Uses:** ENEMY-0006
**Original:** Established
**Redux:** Matches

Trace native pixels in the original player-centered 11×11 view with 16-pixel
tiles, sampling every 13 path entries. Convert samples to cells; exempt shooter
cell and skip final displayed sample at the terminator. This is not ordinary tile
LOS and can skip a cell crossed by the geometric line.

Collision tests the missile-pass bit of the prepared display buffer, not raw
terrain. Unseen cells remain 0xFF and permit missiles. Displayed object cells are
cleared to zero and also permit missiles. Low water passes. Source lighting uses
squared-distance thresholds 2 for night/Underworld, 10 torch, 18 magic light,
50 day, plus the six-step dawn/dusk table and fixed glowing-terrain pools.

Redux reconstructs collision-relevant source-buffer state from authoritative
world data and reuses the tactical pixel sampler. Renderer visibility caches,
debug X-ray, and cosmetic tints do not decide damage. Display scale and smooth
presentation cannot substitute a different obstruction test.

**Implementation:** `internal/game_state/outdoor_fireball_trace.go`, `internal/game_state/combat_missile.go`

### ENEMY-0008

**Tactical contact, interrupted world phases, and dungeon distinction**

**Kind:** rule
**Uses:** DNG-0010
**Original:** Established
**Redux:** Matches

Original descending world processing enters combat synchronously, restores the
world object list, removes only the contacted source (or converts a victorious
pirate to salvage), and resumes the next lower slot without repeating generation
or cadence. Living retreat does not cancel the caller; remaining slow-movement
actor phases precede its final time charge.

Redux retains exact descending cursor, world, source identity, and encounter ID.
Tactical contact suspends before lower actors roll/hit. After origin restoration
and exit-workflow release, drain lower attacks once, including victory, evacuation,
and living physical retreat. Another contact can suspend the same phase again
before a later fireball. Earlier attack still suppresses movement; generation,
mount/Quickness gates, and ordinary player work are not repeated. Explicit player
Attack does not create an actor-phase continuation.

Actual death, terminal handoff, raw teardown, invalid/replaced source, or shutdown
discards retained work. Cleanup does not replay damage or finish a cancelled
command tail. Tactical/pending-handoff saves are unsupported because this cursor
is transient. Ship repair resumes its current cycle only after lower actors and
owed presentation finish. Living retreat completes owed slow-step/ordinary minutes
once rather than treating the outcome label alone as death.

Whirlpool hit/vehicle transport originally precedes the next actor. Terrain/light/
wind reload does not itself recreate the world monster list. Redux on-foot hits
commit synchronously and capture presentation results. Vehicle contact commits
hit/transport at fade-out, then resumes lower slots after fade-in. Because Redux
has separate world controllers, continuation reads the destination controller,
not a stale old-world slice. Later attacks use resolved position, vehicle, and HP;
cancellation executes no continuation or repeated damage.

First-person dungeon wanderers use a different single-object system: Rat, Bat,
Spider, Ghost, Slime, Gremlin, Gazer, Reaper with initial animation bytes
0x60, 0xA0, 0x00, 0x90, 0x80, 0x60, 0x00, 0x00 respectively. DNG-0010 owns
placement/ceiling/movement and generated combat. Fixed room descriptors are not
that pool; generated dungeon counts can consume a draw even when subsequently
replaced by fixed bounds 8 or 16.

**Implementation:** `internal/game_state/outdoor_actor_continuation.go`, `internal/game_state/combat_dungeon_wanderer.go`
**Tests:** `internal/game_state/outdoor_actor_continuation_test.go`, `cmd/ultimav/gamescene_outdoor_retreat_continuation_test.go`
