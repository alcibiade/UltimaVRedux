# Vehicle movement and transfer

Terrain admission, object occupancy, and inventory transfer are distinct. These
rules scope original DOS behavior separately from modern validation and ownership.

### VEH-0001

**Terrain and directional shapes**

**Kind:** rule
**Uses:** TILE-0001
**Original:** Established
**Redux:** Unverified

| Form | Original terrain admission |
| --- | --- |
| Foot | Walking bitmap; set bits prohibit movement. |
| Horse | Walking admission except lava/swamp. |
| Carpet rider | Water/river or walking admission. |
| Skiff | Facing-sensitive coast/river masks plus admitted water. |
| Frigate | Terrain value no greater than Sea. |

Chair exceptions belong to ordinary-foot/NPC tile families, not horse/carpet
riders. Occupancy is checked separately by callers. Skiffs pass river bridge 6A
hex north/south and 6B east/west; this admits neither red bridges nor lowered
drawbridges, and frigates cannot use river bridges. Horse/carpet vertical movement
retains horizontal art; skiffs/frigates have four facings, with frigate sail state
encoded separately in its shape. Redux uses effective terrain and floor-aware
occupancy; complete bitmap/corrupt-object equivalence is not claimed here.

### VEH-0002

**Boarding transfers existing state**

**Kind:** rule
**Uses:** TIME-0002, VEH-0001
**Original:** Established
**Redux:** Unverified

Reject dungeon map IDs 33–40, find an object at party X/Y and level, validate its
vehicle gates, transfer state, then remove the boarded object. A linked horse
with nonzero talk refuses before the on-foot gate. Horse/carpet/skiff boarding
accepts ordinary or invisible-foot through the common foot predicate.

A frigate target must be furled. Boarding accepts foot, invisible-foot, carpet,
or skiff; copies hull and carried skiffs; returns a ridden carpet to inventory or
adds the ridden skiff to carried stock; installs frigate state and removes the
map object. No intrinsic time is added. Success and many refusals complete the
context-owned turn; no boardable object returns false.

Redux preserves current-position/floor lookup, authored horse refusal, and
carpet/skiff transfer. Its capacity and stale-state checks are additional
boundaries, not blanket proof of every original transfer edge.

**Implementation:** `internal/game_state/action_board.go`

### VEH-0003

**Exit tests nearby land without moving there**

**Kind:** rule
**Uses:** VEH-0001, VEH-0002
**Original:** Established
**Redux:** Unverified

The original opening dungeon exclusion is logically impossible. Raised-sail
frigates refuse. Horse exit has no terrain gate; carpet exit accepts adjacent
land or walkable current terrain; skiff exit requires adjacent land and rejects
river-bridge underlay. Furled frigates choose land, carried skiff, consumed carpet,
then refusal, in that order.

Land checks north/south/east/west, short-circuiting, using display terrain and
allowed object categories: carpet, player, furled frigate, horse, skiff. It is
an eligibility test, **not relocation**: party and abandoned vehicle stay at the
current coordinate. Voluntary frigate-to-skiff Exit records one fewer skiff on
the abandoned frigate and preserves hull.

Redux's inspected `exitVehicle` retains these main fallback/current-position
contracts with wrapped outdoor probes, bounded town probes, floor-aware effective
terrain, and object-capacity preflight. Its production completion is context-owned.
Malformed contexts and original first-slot object-lookup quirks remain unverified.

**Implementation:** `internal/game_state/action_exit.go`

### VEH-0004

**Carpet deployment versus collection policy**

**Kind:** rule
**Uses:** INV-0004, VEH-0002
**Original:** Established
**Redux:** Intentional deviation

Original inventory Use requires an owned/selectable carpet, map below 33, current
terrain not exactly Mountains, and exact ordinary-foot form. It requests R(0,1)
for facing, changes form, then consumes one carpet. There is no level/plane gate:
outdoor Underworld qualifies; invisible-foot fails this particular Use gate.
Original directional Get accepts any carpet object, increments stock with its
ordinary cap, and removes the object regardless of ownership classification.

Redux shares deployment eligibility between review and menu projection, commits
with occupancy/capacity/staleness checks, and retains one facing draw and normal
workflow time. **REDUX DEVIATION: VEH-0004:** Get collects Lord British's carpet
and party-deployed carpets but leaves other authored carpets as scenery. Explicit
ownership survives boarding, Exit, and native saves to preserve that distinction.

**Implementation:** `internal/game_state/action_use_commits.go`, `internal/game_state/action_get.go`
