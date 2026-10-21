# Dungeon exploration and tactical rooms

Packed first-person cells and 11×11 tactical records are different formats.
`R(a,b)` denotes requested random bounds, without distribution or reversed-bound
guarantees. Inspected source paths do not certify authored binaries, retail builds,
undefined memory access, or every exceptional return.

### DNG-0001

**Dungeon addressing and identity**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Unverified

Eight blocks contain eight floors of 8×8 byte cells. For zero-based block/floor/X/Y:
`offset = 512*block + 64*floor + 8*Y + X`. Entry reads 512 bytes; displayed floor
is internal floor plus one. Block order/map IDs are Deceit 33, Despise 34, Destard
35, Wrong 36, Covetous 37, Shame 38, Hythloth 39, Doom 40. Addressed extent is
4,096 bytes, not a new measurement of DUNGEON.DAT length or trailing data.

### DNG-0002

**Packed families and mutation boundaries**

**Kind:** rule
**Uses:** DNG-0001
**Original:** Established
**Redux:** Unverified

| High nibble, hex | Meaning |
| --- | --- |
| 0 | Passage |
| 1 / 2 / 3 | Up / down / bidirectional ladder |
| 4 / 7 | Closed / open chest |
| 5 | Fountain |
| 6 | Pit/trap family |
| 8 | Energy field |
| 9 | No established normal semantic identity |
| A / F | Cleared / live room threshold, low nibble room ID |
| B | Wall; low bits can select graffiti |
| C | Style-specific special wall, sometimes animation bits |
| D / E | Hidden door / revealed doorway |

Low bits are family-specific, not universal flags. Opening-above bit 08 applies
to ordinary lower families. Reveal hidden door to E0 plus opening bit; Open chest
to 70 plus opening; Get to passage/opening; disarm exact bomb 62 to passage;
crumble skeleton-style C wall to B0 plus opening; clear live room F to A with
the same room ID. Search exact hidden pit 61 reveals 60 and adds opening below
only above floor 7. Search discovery is not the falling procedure.

### DNG-0003

**Entry, exits, facing, and relative movement**

**Kind:** rule
**Uses:** DNG-0001, TIME-0002
**Original:** Established
**Redux:** Unverified

Entry matches world entrance indexes 32–39 and requires exact ordinary foot.
Doom additionally requires all three Shadowlord-state high bits; failure creates
a Shadowlord and invokes outdoor combat. Surface entry starts floor 0,(1,1),east,
arrival 5. Non-Doom entry from nonzero world level starts floor 7,(7,7),west,
arrival 4; Doom uses the floor-zero initialization. Boundary exits restore the
indexed entrance: upper boundary Britannia, lower boundary Underworld.

Up advances, Down backs up, Left/Right rotate, period turns around; Return also
turns around in the non-CHEAT branch. X/Y wrap modulo eight. Revealed doorways
refuse quarter-turns but allow turnaround. Every attempt records resulting facing,
including blocked/refused attempts. Advance/back-up completes even blocked;
rotation skips post-processing but retains its original precharge.

Families 0–9 and E permit advance/back-up; A/F permit only advance; B–D block both.
The wanderer's coordinate blocks separately. Exact electrical cell 83 receives
movement bounce/party damage before normal family collision; 8B does not enter
that special branch. Redux's accepted-key/deferred-token policy is TIME-0002;
it does not certify original all-sleeping or valueless-return paths.

**Implementation:** `internal/game_state/dungeon_movement.go`, `internal/game_state/dungeon_command_cycle.go`

### DNG-0004

**Current-cell chests, Search, and actor selection**

**Kind:** rule
**Uses:** PROV-0004, RNG-0001, LIGHT-0005
**Original:** Established
**Redux:** Intentional deviation

Open inspects the current closed chest, resolves an actor, springs a trap when
low-three trap bits are nonzero, then opens while preserving bit 08. Trap-table
multiplicities are Acid 3, Poison 2, Bomb 2, Gas 1 among eight entries: Acid uses
transformed d30 on the actor, Poison targets the actor, Bomb damages party, Gas
attempts poison across six slots with active/dead gates. These are not verified odds.
Original active designation is trusted; only fallback checks Good/Poisoned.

Jimmy resolves actor before chest/key checks; its formula and safe 40/48 behavior
are in PROV-0004, using **internal floor**. Search/Look resolve actor, require
light, then request Ahead/Here/Left/Right/Pass and wrap coordinates. Search rerolls
inspection each time, recognizes exactly 40 as safe (not 48), invents severity
after failure, recognizes pit 61/bomb 62 but not 69/6A, reveals hidden doors without
DEX, and crumbles only skeleton-style special walls. Look disguises exact 61 as
passage, not 69. Fountain effects are WELL-0003.

**REDUX DEVIATION: DNG-0004:** Redux remembers first truthful trapped/safe or explicit
uncertain assessment for the party, avoiding repeated/fabricated reports. Repeat
Search rolls nothing; safety uses actual trap bits, including 48. Known-safe Jimmy
refuses harmlessly, blind attempts retain cost. Knowledge lasts the active visit,
survives floor/combat round trips and opening-bit-only changes, but removal,
replacement/rearming/fresh visits clear it. Open/An Sanct retire closed-chest
knowledge; successful Jimmy records safety on the still-closed chest.

**Implementation:** `internal/game_state/dungeon_interactions.go`, `internal/game_state/dungeon_chest_assessment.go`
**Tests:** `internal/game_state/dungeon_chest_assessment_test.go`

### DNG-0005

**Dungeon chest contents and draw order**

**Kind:** rule
**Uses:** DNG-0002, INV-0004
**Original:** Incomplete
**Redux:** Unverified

Get requires an open chest and removes it before contents. Visit categories in
table order; test R(1,4*(floor+1)) at least the threshold, and finish successful
quantity/type draws before testing the next category.

| Category | Threshold | Additional request |
| --- | ---: | --- |
| Food | 2 | R(1,31) quantity |
| Gold | 4 | R(1,8*floor) quantity |
| Keys | 5 | R(1,3) |
| Gems | 10 | R(1,3) |
| Torches | 20 | R(1,3) |
| Potion | 25 | R(0,7) type |
| Scroll | 25 | R(0,7) type |

Floor-zero gold requests R(1,0); the external generator's result is unresolved.
Do not silently replace it with a chosen amount or claim uniform independent draws.

### DNG-0006

**End-player field and trap order**

**Kind:** rule
**Uses:** DNG-0002, ENV-0001, PROV-0001, TIME-0002
**Original:** Established
**Redux:** Unverified

Sleeper wake tests R(0,63)<4 precede optional wanderer movement/contact. Fixed-room
processing **or** ordinary-cell hazard dispatch follows, then player update.
They are alternative branches, not unconditional room-then-hazard execution.

| Cell, hex | Effect | Mutation |
| --- | --- | --- |
| 80/88 | Each active slot R(1,30)>=DEX can become Sleeping | Clear to passage/opening |
| 81/89 | Same comparison can become Poisoned | Persist |
| 82/8A | Party damage | Persist |
| 62/6A | Clear trap, then party damage | Clear |
| 61/69 | Pit procedure | Source/destination mutation |
| 83/8B | No end-player damage case | 83 handled in movement |

Sleep/poison draws occur in ascending active slots before the successful branch's
Dead test, and replace the prior non-dead status. The following player update can
immediately apply poison. Redux has its own token-owned final-effects path; full
exceptional-room/transition equivalence remains unverified.

### DNG-0007

**Pit fall accesses destination before damage**

**Kind:** rule
**Uses:** DNG-0002, ENV-0001
**Original:** Established
**Redux:** Intentional deviation

Set downward arrival; clear source low-three bits; increment floor; read destination
at unchanged X/Y; add opening bit only below byte 90 hex; redraw; damage party;
repeat if another exact hidden-pit variant remains. From floor 7 this reads outside
the eight-floor array before a later outdoor-state assignment. It is not a ninth
floor or proven safe exit. **REDUX DEVIATION: DNG-0007:** Redux reveals/damages once
at the bottom and stays within valid floors, retaining its workflow command token.

**Implementation:** `internal/game_state/dungeon_hidden_pit.go`

### DNG-0008

**Packed-cell, vertical, and temporal spells**

**Kind:** rule
**Uses:** DNG-0002, INV-0006, TIME-0002, MOON-0005
**Original:** Established
**Redux:** Unverified

Dungeon handlers use facing without another direction prompt. An Sanct searches
current then ahead for a closed chest and opens it preserving 08. Field creation
accepts only 00/08 ahead: Fire→82/8A, Poison→81/89, Sleep→80/88, Protection→83/8B.
An Grav searches current then ahead for field family, replacing with passage/opening.
Non-CHEAT prepared resources commit before unsuitable-target rejection (INV-0006).

Uus Por/Des Por retain X/Y, require high-nibble-zero interior destination, and
exit at upper/lower boundaries without that interior test; both reject Doom.
Vas Rel Por is dungeon-legal including Doom, chooses 1–8, rejects carried destination,
and rejects the entire ship-shape high-nibble family including skiffs.
Quickness installs 30; prepared Time Stop installs 10 unless absorbed by a Shadowlord
object. Current command actor/time choice precedes casting; normal completion can
decrement the new duration. The Time Stop scroll instead consumes before rejecting
Doom/Stonegate and otherwise installs 20.

Redux completes transition spells' command ownership before changing maps,
suppressing stale post-transition dungeon work; rejection/cancellation follows its
validated-attempt policy. This is not blanket original resource/return equivalence.

**Implementation:** `internal/game_state/dungeon_spells.go`

### DNG-0009

**First-person presentation and dungeon color boundary**

**Kind:** rule
**Uses:** LIGHT-0005, PROV-0003
**Original:** Incomplete
**Redux:** Intentional deviation

Center presentation samples depths 0–3. Families below A continue; structural
families stop except a revealed doorway at depth zero. Deceit/Wrong/Covetous use
wall/hex-floor and skeleton special walls; Shame/Hythloth use rock/grass and
collapsed-passage style; remaining dungeons use rock/grass and stalactites.
Opening 08 draws above; hidden pit 61 lacks a visible floor pit. Original stalactite
drawing mutates animation bits and rendering consumes RNG.

Dungeon tactical setup calls color operation 0 when required, shared by fixed,
generated, and camping maps. Exact driver tile lists, palette/bitplane mapping,
and authored pixels were not reverified: the call does not prove their transformation.

**REDUX DEVIATION: DNG-0009:** Redux derives cosmetic animation deterministically
without changing packed cells/game RNG; optional subdued unlit geometry differs
from original black darkness. It selects tactical colors from displayed dungeon
context, with finale color handling separate. Exact driver parity remains unverified.

Redux field-removal feedback consumes a successful Sceptre or An Grav result
after the packed cell has changed. A 600 ms pixel fizzle stays anchored to that
cell and retains only the removed field, using a presentation-owned seed. Normal
depth, occlusion, and black/subdued-light rules still apply. At most four remnants
coexist; changing the affected cell, floor, dungeon visit, or combat context retires
the old remnant. Movement and unrelated cell changes do not relocate or restart it.
The fizzle adds no command wait or second completion. Each admitted success requests
`SoundDispel` once; its intentionally empty recording slot is shared with combat
An Grav. This is a success cue, not failure audio or a claim of audible playback.

Actual fire-field, electrical-contact, and dungeon poison-tick HP loss has separate
200 ms translucent light-red, light-blue, and light-green viewport feedback.
Party losses aggregate per resolved source; zero loss, sleep contact, and poison
onset alone do not produce an injury pulse. Repeated live requests coalesce by
kind, limiting this family to three active/pending requests in the serialized,
nonblocking feedback lane. Injury remains visible over an already-black viewport
without revealing terrain or changing illumination. It retains visit/floor
ownership and clears on expiry or invalidation. Audio is requested once per
resolved injury source even when the visual coalesces; electrical and poison
recordings are currently empty. These are Redux presentation contracts, not a
new original-renderer equivalence claim. Hazard resolution remains
[DNG-0006](#dng-0006); body-only Time Stop timing is
[SPELL-0004](MAGIC_SPELLS.md#spell-0004).

**Implementation:** `cmd/ultimav/dungeon_renderer.go`, `cmd/ultimav/dungeon_field_dispel.go`, `cmd/ultimav/gamescene_dungeon_injury.go`, `internal/game_state/dungeon_field_dispel_owner.go`, `internal/game_state/dungeon_injury_feedback.go`, `internal/game_state/dungeon_injury_owner.go`

### DNG-0010

**Wanderer placement, movement, and generated combat**

**Kind:** rule
**Uses:** DNG-0002, CAMP-0002
**Original:** Established
**Redux:** Unverified

Current creature is object slot 1; slot 2 supplies previous coordinates. Absence
uses the creature-number sentinel, not simply tile zero. Ordered pool:
Rat, Bat, Spider, Ghost, Slime, Gremlin, Gazer, Reaper; initial animation bytes
60,A0,00,90,80,60,00,00 hex respectively. Try up to eight random cells with repeats:
families below 6 or family 7; both X and Y must differ from party coordinates.
This is whole-row/column rejection, not LOS. Spider/Slime ceiling uses R(0,99)>48.

Reapers do not move. Others try eight wrapped cardinal candidates, rejecting 6,8,
and A upward; 9 is movement-admissible despite generation rejection. Returning to
previous coordinates requires R(0,7)=1. Successful movement copies only X/Y into
slot 2. Direct Attack checks wrapped ahead without a light gate and skips normal
end-player completion.

Generated combat makes 16 placement-swap draws, draws group count, then replaces
it with the bound when that bound is 8 or 16 (the draw still occurred). Trigger
coordinates are disabled. A generated center chest becomes tactical loot worth
`3*floor+7`, restoring the generated floor below it. Redux retains that metadata
through wanderer/camping handoff and creates the chest before monsters; all RNG
and malformed-object paths are not certified by this bounded comparison.

**Implementation:** `internal/game_state/dungeon_wanderer.go`, `internal/game_state/combat_dungeon_wanderer.go`

### DNG-0011

**Fixed-room addressing and envelope**

**Kind:** rule
**Uses:** CBT-0001, DNG-0001
**Original:** Established
**Redux:** Unverified

Room selection uses low-nibble room ID, no floor component. Compressed dungeon
index omits Despise: subtract 33 from map ID, decrement when result is at least
one. Record index=`compressed*16+room`, byte offset=`record*352`. Seven sets imply
112 addressed records/39,424 bytes; this is not measured DUNGEON.CBT length.
Unexpected Despise thresholds are not explicitly rejected and alias Deceit's set.

Each record has eleven 32-byte rows: terrain uses columns 0–10. Party X bytes use
columns 11–16 and Y 17–22 of row 3/2/4/1 for north/east/south/west entry. Vertical
arrival converts to facing first. Sixteen descriptors occupy AB–BA hex, X CB–DA,
Y EB–FA. BRIT.CBT shares the size, not these metadata consumers. A tactical grate
supports downward departure; the first departing PC fixes the common exit direction.

### DNG-0012

**Descriptor initialization and immediate loot**

**Kind:** rule
**Uses:** DNG-0011, INV-0004
**Original:** Established
**Redux:** Intentional deviation

Zero descriptors are absent. Values below NPC base 40 hex, artifact family B4–B7,
and field family E8–EB create objects; other descriptors decode creature index
`(descriptor-64)/4`. Live-room setup initializes players, makes four variable-
creature draws, then completes each of sixteen slots ascending, including value
draws, before the next. Cleared-room setup skips that descriptor branch and its
four draws; player initialization may already have consumed ring RNG.

| Descriptor, hex | Value generation |
| --- | --- |
| 01 chest | 3*floor+7, initially untrapped |
| 02 gold | R(1,10*(floor+1)) |
| 03 potion / 04 scroll | R(0,7) type |
| 05 weapon / 06 shield | R(0,3) offset from Long Sword / R(0,2) from Small Shield |
| 07 Keys / 08 Gems / 0D Torches / 0F Food | R(0,7)+1 |
| 09 helm / 0A ring / 0B armor / 0C amulet | R(0,3)/R(0,2)/R(0,5)/R(0,2) from category base |
| 0E box | R(0,0)+1; call still occurs |
| B4–B7 artifacts | Descriptor identity, no loot-generation draw |

Get grants inventory immediately and removes the object, not at victory. That
does not prove retention through every death/recovery path. Re-entered live rooms
rebuild descriptors; immortal-room exceptions are below.
**REDUX DEVIATION: DNG-0012:** Redux rejects invalid coordinates/exhausted object
capacity before loot RNG, avoiding unsafe original allocation writes.

**Implementation:** `internal/game_state/combat_dungeon_room.go`

### DNG-0013

**Concealed occupants and eight trigger slots**

**Kind:** rule
**Uses:** DNG-0011
**Original:** Established
**Redux:** Unverified

Occupants exist before display. Hidden/empty or arch underlay suppresses drawing,
not their authority; terrain replacement can expose them without a new population.
For trigger slot i=0–7, relative offsets are:

| Field | Offset, hex |
| --- | --- |
| Replacement terrain | 00B+i |
| Trigger X / Y | 10B+i / 113+i |
| Destination 1 X / Y | 12B+i / 133+i |
| Destination 2 X / Y | 14B+i / 153+i |

Dungeon-room/combat scenario gates apply. On exact match, set both trigger
coordinates FF, then independently write each destination if both coordinates
are below 11. Continue later slots: duplicate triggers run in order, (0,0) is
valid, and invalid destinations still consume the matched trigger. Reload restores
metadata. Successful movement, small-rock Klimb, Push target, and an unblocked
ordinary projectile endpoint invoke it. Push tests before pushability/occupancy;
miss endpoints count; reach-weapon paths bypass that launch call. Traversal cells
are not tested there. Generated maps disable triggers. Exact Covetous room-0
concealed chamber/trigger payload remains an authored-data verification gap.

### DNG-0014

**Room flags versus visit-local mutations**

**Kind:** rule
**Uses:** DNG-0011, DNG-0012
**Original:** Established
**Redux:** Intentional deviation

The 14-byte room bitset has 112 compressed-dungeon/room flags at SAVED.GAM 033A
hex. On fresh entry, load pristine 512-byte dungeon data and convert flagged live
thresholds to cleared. Wrong rooms 1,6,11,12 and Covetous rooms 0,11 are excluded
from persistent clearing, though their loaded thresholds can clear for the visit.
The active dungeon buffer is separately saved at 03B4 hex. Ordinary opened chests,
doors, fields, and pits do not persist across re-entry through the room bitset.

**REDUX DEVIATION: DNG-0014:** optional cross-visit hidden-discovery persistence
replays supported discoveries, extending original visit lifetime. Native hydration
restores active cells, facing, last direction, current wanderer and previous X/Y
without replaying entry. This is distinct from fresh-visit reconstruction.

**Implementation:** `internal/game_state/dungeon_discovery.go`, `internal/game_state/combat_dungeon_room.go`

### DNG-0015

**Mirror finale and authored-data limits**

**Kind:** rule
**Uses:** DNG-0011
**Original:** Incomplete
**Redux:** Intentional deviation

The generic mirror check requires an active non-object combat entry on row 2
with a matching mirror-family display object directly above on row 1; it does
not hard-code X=5. Successful room return with mirror exit invokes the ending.
Its rescue map is a separate 176-byte MISCMAPS.DAT record at offset 528, populated
procedurally. The second box question follows only an initial negative response;
final affirmation **and actual box possession** are required for success. Failure
enters a terminal movement/wandering tableau without success artwork/final screen.

Exact Doom thresholds, room-15 party/mirror coordinates, rescue-map mirror terrain,
Covetous concealed payload, driver color exchange, and narrative bytes require
separate binary evidence. Loader arithmetic/control flow does not certify the old
authored tables. These are explicit unresolved dispositions, not invented replacements.

**REDUX DEVIATION: DNG-0015:** Redux freezes the failed tableau. A failure without
the box, whatever the final answer, waits 5,000 deterministic ms and offers the
matching dedicated pre-Doom checkpoint or main menu; failing while holding the box
offers menu only.
The canonical checkpoint is outside eight player slots and needs the exact opaque
Doom-run association. This provides bounded recovery from the terminal original path.
