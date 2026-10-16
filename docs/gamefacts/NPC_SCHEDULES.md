# NPC schedules, movement, and contact

These rules describe the inspected DOS source model. Its version label does not
identify an immutable retail build. Authored actors require separately verified
NPC data; source algorithms alone do not establish every town's population.
Redux comparisons describe the current application; configured-data fixtures
establish actor-specific behavior only for the installation they actually test.

### NPC-0001

**Three destinations and four time markers**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Matches

A schedule occupies 16 bytes: three action bytes, three X bytes, three Y bytes,
three floor bytes, then four hour markers. The fourth marker selects destination
1, not a fourth destination. For an hour byte, selection is:

```text
best = earliest index minimizing ((hour - marker[index]) modulo 256)
slot = [0, 1, 2, 1][best]
```

Unsigned subtraction and first-index tie retention matter. With markers
`6,12,18,0`, hours 0–5 select slot 1, 6–11 slot 0, 12–17 slot 1, and
18–23 slot 2. All-zero markers select slot 0. This is byte arithmetic, not
subtraction modulo 24, even though ordinary hour inputs describe a day.

Redux selects the minimum unsigned-byte difference and retains the first tie.
For markers `6,6,18,0` at hour 7, differences are `1,1,245,7`, selecting slot 0.
Regression cases include duplicate, all-zero and unsorted markers. This establishes
algorithm behavior without claiming every exceptional marker occurs in installed data.

**Implementation:** `internal/references/npc_schedule.go`
**Tests:** `internal/references/npc_schedule_test.go`

### NPC-0002

**Placement, arrival, travel, and recovery**

**Kind:** rule
**Uses:** NPC-0001
**Original:** Established
**Redux:** Intentional deviation

Initialization considers authored slots 1–31 with nonzero shape. Each actor is
placed at its selected destination, made inactive, and records that arrived
slot. Later schedule changes distinguish same-floor travel, travel to a ladder,
arrival from another floor, and off-floor relocation. Desired destination and
last arrival are different facts; wandering away from an arrived work anchor
does not itself mean the actor never arrived.

Floor travel is classified relative to the floor observed by the party:

| Current actor floor | Selected destination floor | Schedule transition |
| --- | --- | --- |
| Observed | Observed | Walk to the scheduled destination. |
| Observed | Hidden | Walk to a suitable departure ladder/stair, then relocate to the hidden destination and record arrival. |
| Hidden | Observed | Find a suitable receiving-floor entry using the destination as search anchor, establish an entry-to-destination route, place at the entry, then walk on later updates. |
| Hidden | Hidden | Relocate directly to the destination and record arrival, even when both endpoints are on the same hidden floor. |

Arrival from above uses the receiving floor's upward connection; arrival from
below uses its downward connection, with stairs following the established
directional rules. Merely entering the observed floor does not change the
last-arrived slot: the actor must reach the selected destination (an entry that
is itself the destination already satisfies that condition). Settled hidden
actors do not wander. These schedule transitions are separate from changing the
**player's** floor, which resets NPC positions to their current scheduled
destinations. That same-location reset retains loaded dialogue mutations and AI
overrides; overrides retain their precedence over ordinary scheduled travel.

Original movement retains an active travel operation, uses shared path-search
budget and pause counters, and can attempt a gated random recovery after blocked
movement or failed search. Its planner tests terrain differently from actual
steps (NPC-0004).

**REDUX DEVIATION:** Redux uses weighted A* and separate desired/arrived state,
reclassifies against the newest schedule, and discards obsolete cached goals.
Every cached step is rechecked for adjacency, terrain, and occupancy; rejection
clears the remaining path. Arrival and AI overrides also clear cached paths.
Occupied goals are excluded, as are chairs when routing to another floor. This
avoids repeatedly planning unusable routes rather than preserving original
path activity, limited nearby-object planning, and pause counters.

Redux searches connections on the observed floor's authoritative persistent
terrain, including overrides, independently of visual object sprites. Entry and
route occupancy use actual same-floor actors and the party. It checks candidates
in stable nearest-anchor order and tries alternatives when a nearer connection
is occupied or unreachable. The full 32×32 map is included, including the origin
and final row/column. Missing connections or blocked entry routes leave the actor
on its prior floor, discard obsolete paths, and retry on the next update without
recovery RNG or destination teleportation. Each update has a finite candidate
scan and bounded map search; this does not reproduce the original shared search
budget. Direct hidden relocation clears old routes. Incoming entry retains its
route and prior arrival until physical destination arrival; native persistence
retains those distinct facts. A matching last-arrived slot counts as settled only
on that slot's destination floor. If an incoming walk from A to B is interrupted
by selecting A again before reaching B, the retained arrival at A does not cancel
the required departure from B's floor. The actor must still reach a usable
departure connection before relocating back to A. This floor check does not
require a same-floor arrived wanderer to return to its anchor.

After unsuccessful same-floor travel on the active floor, Redux tries one
original-style gated cardinal recovery step, retaining scheduled-travel terrain
permissions. It does not cross occupants or unrelated beds. Settled fixed actors
consume no recovery RNG; failed ladder travel does not use this recovery. Redux
retains usable cached routes and retries failed planning on later updates without
the original pause counters. A seeded Serpent's Hold 05:00 regression covers eight
ground-floor guards reaching their shift destinations without a reload.

The Sindar regression resolves a Trinsic actor slot through its installed town
conversation identity (dialogue 32), rather than treating the dialogue number as
an actor index. It checks the configured schedule, destination terrain, and
physical travel during ordinary Pass from 23:50 through 07:30 without a floor
change or sleep; a separate sequence compares the deliberate player-floor reset.
Each Pass logs time, actor identity, position/floor, selected and arrived slots,
remaining route, and the transition class inferred from the pre-update state.
It also checks same-floor occupancy throughout the observed walk. The party
stays at ground-floor coordinate (15,15); this is a controlled schedule trace,
not a rendered line-of-sight test at the healer entrance.
The fixture expects a fixed upper-floor destination at midnight, a different
upper-floor wandering destination at 01:00, the first upper-floor destination at
05:00, and a ground-floor bed destination at 06:00. The configured installation's
first upper-floor anchor is a fireplace at (25,7), not ordinary passable terrain.
An exact schedule destination has the NPC-0004 terrain exception; hidden-floor
relocation also does not require a walkable route on the observed floor. The
fixture logs all installed anchor tiles and checks the actual morning receiving
route, including cardinal steps, map bounds, intermediate terrain and its final
destination. These are schedule-selection times, not guaranteed arrival times.
A different installation must be reviewed
rather than silently accepted as the same actor-data proof. This controlled
ground-floor observation does not establish the reporter's unspecified position,
hours, or whether their playthrough included a floor change or sleeping.
Actor-data acceptance requires a passing configured-data run; an unavailable-data
skip does not verify Sindar's installed itinerary or route.

**Implementation:** `internal/ai/npc_ai_controller_small_map.go`, `internal/ai/npc_schedule_paths.go`, `internal/map_units/npc_friendly.go`, `internal/references/small_location_reference.go`
**Tests:** `internal/ai/npc_schedule_floors_test.go`, `internal/ai/npc_ai_controller_small_map_recovery_test.go`, `internal/ai/npc_ai_controller_small_map_recovery_original_test.go`, `internal/references/small_location_ladders_test.go`, `internal/game_state/npc_schedule_state_test.go`, `internal/game_state/npc_sindar_integration_test.go`

### NPC-0003

**Settled actions and source-undefined pursuit fallback**

**Kind:** rule
**Uses:** RNG-0001, NPC-0001
**Original:** Incomplete
**Redux:** Intentional deviation

Distances below are Manhattan tile distances. The action belongs to the selected
schedule slot; being combat-capable does not make a fixed or talking actor hostile.

| Action | Settled behavior |
| ---: | --- |
| 0 | Remain fixed. |
| 1 | Attempt gated wandering, destination at most 3 tiles from schedule anchor. |
| 2 | Attempt gated wandering without the anchor-radius limit. |
| 3 | Flee only while actor–party distance is below 4. |
| 4 | Approach/talk when party–schedule-anchor distance is below 4; otherwise action-1 wandering. |
| 5 | Approach/talk without that range gate, with movement variation. |
| 6 | Approach/attack only while actor–party distance is below 4. |
| 7 | Approach/attack without that range gate, with movement variation. |

Wandering first requests a value in `0..255` and proceeds only if bit `0x08`
is set. It then requests a draw with bound 64 and uses its low two bits to
choose East, North, West, South for values 0,1,2,3 respectively. Blocked
destinations do not cause retry. These
bounds and masks do not prove the external generator's distribution.

Approach first checks existing adjacency, then evaluates legal cardinal steps
in East, North, West, South order. It takes the first strictly distance-reducing step.
Flee considers distance-increasing steps and uses a binary draw to replace an
already selected candidate. Actions 5 and 7 request `0..63`; below 16 enables
variation among legal alternatives, excluding the original preferred direction;
the first alternative needs no replacement draw, later alternatives do.

The original fallback beyond those four candidates reads outside its four-entry
distance array. Outcomes on a blocked/no-improvement path are therefore not a
determinate greedy algorithm. **REDUX DEVIATION: NPC-0003:** Redux safely stays or
selects a checked alternative rather than reproducing an out-of-bounds fallback.
This keeps movement deterministic and within legal terrain/occupancy constraints.
Redux supports action 5's ungated approach, variation and adjacent Talk dispatch.
Action 3 still attempts escape when adjacent; adjacency dispatch applies only to
actions above 3. The native save override enumeration remains frozen: action 5 is
supported through authored schedules, not newly admitted as a saved AI override.

**Implementation:** `internal/ai/npc_ai_controller_small_map.go`, `internal/references/npc_ai.go`, `internal/save_snapshot/validation.go`
**Tests:** `internal/ai/npc_ai_controller_small_map_schedule_test.go`

### NPC-0004

**Town terrain, destination exceptions, and occupancy**

**Kind:** rule
**Uses:** NPC-0002
**Original:** Established
**Redux:** Intentional deviation

Ordinary NPCs have their own 256-tile permission table, independent of player
walking and outdoor monsters. Terrain is the underlying town tile, not an
object sprite. Actual movement rejects coordinates outside the 32×32 map and
separately rejects same-floor object occupancy.

The ordinary predicate permits exactly these decimal raw tile IDs before
context exceptions: `0`, `4–11`, `14–15`, `29–38`, `44–45`, `64`, `68–69`,
`71–73`, `106–107`, `135`, `144–147`, `170`, `184`, `186`, `196–201`.
All other raw IDs are rejected by that predicate. This semantic set preserves
the complete terrain contract without reproducing its encoded bitmap.

| Terrain/context (decimal tile IDs) | Actual-step rule |
| --- | --- |
| Fireplace 188, lava 143, grate 134, trap/hole 140 | Blocked. |
| Location markers 16–25, 27, 57, 62; water edges 52–55 | Blocked even where player walking is possible. |
| Moongate 220, fake desert 221, ship sign 249, nothing 255 | Blocked. |
| Chairs 144–147 | Allowed only during same-floor scheduled movement, not inactive wandering, attraction, or ladder travel. |
| Beaches 48–51 | Actual-step exception permits them. |
| Unlocked doors 184, 186 | Permitted, including wandering. |
| Locked/magical doors; beds 171–172 | Normally blocked. |
| Swamp, plowed fields, crops, stairs, ladders | Permitted; this is not a blanket hazard filter. |
| Exact selected destination X/Y/floor | Bypasses ordinary terrain check when supplied by travel or wandering; attraction supplies no destination. Earlier chair and occupancy checks still win. |
| Shadowlord | Bypasses terrain but not bounds or occupancy. |

Original path construction uses the ordinary terrain predicate, without the
beach/Shadowlord actual-step exceptions; it admits chairs independently of travel
activity. Original actual steps still enforce the stricter contextual rules.
Redux keeps terrain separate from authoritative occupancy and reads persistent
terrain plus overrides; raw base tile zero remains meaningful. Its planner's
occupied-goal and ladder-chair changes are the deliberate routing policy in
NPC-0002, not a broader walking permission. Occupied-door rendering does not
grant NPCs additional traversal rights.

**Implementation:** `internal/references/tile.go`, `internal/ai/npc_ai_controller_small_map.go`
**Tests:** `internal/references/tile_npc_passability_test.go`, `internal/ai/npc_ai_controller_small_map_terrain_test.go`

### NPC-0005

**Contact precedes movement and the last eligible slot wins**

**Kind:** rule
**Uses:** NPC-0003, CRIME-0002
**Original:** Established
**Redux:** Matches

At cardinal distance one, actions 4/5 queue Talk only with nonzero Talk state;
actions 6/7 queue attack. This happens before movement, so moving adjacent waits
until the next NPC phase. Ascending actor traversal retains one pending command;
the last eligible source slot wins. Astaroth influence takes precedence over
ordinary shape dispatch; otherwise Guards arrest, and other eligible combat
shapes fight. Combat composition is distinct from schedule intent.

Redux live, visible, same-floor actors use that contact path. Its automatic
Astaroth response/damage/flee workflow does not start combat or charge another
exploration turn. The reviewed hostile-contact scope does not certify action 5
dispatch, whose missing settled handler is recorded in NPC-0003.

**Implementation:** `internal/ai/npc_ai_controller_small_map.go`, `cmd/ultimav/gamescene_talk_workflow.go`
**Tests:** `internal/ai/npc_ai_controller_small_map_contact_test.go`, `cmd/ultimav/gamescene_shadowlord_talk_test.go`

### NPC-0006

**Loose horses and hitching**

**Kind:** rule
**Uses:** RNG-0001
**Original:** Established
**Redux:** Intentional deviation

Process loose current-floor horses in object-slot order. A `0..1` draw gates the
update: nonzero stays, zero continues. Then probe south, east, north, west terrain
for Hitching Post `0xA2` or Rail `0x43`. A hitched horse stays. Otherwise a `0..1`
draw selects horizontal when nonzero, vertical when zero; another selects
negative for zero, positive for one. Attempt exactly that one tile. It must be in bounds,
horse-passable, and free of same-floor objects and party. East/west movement
changes horizontal facing; north/south preserves it. There is no schedule-home
radius and no retry after failure.

**REDUX DEVIATION:** An original off-map hitch probe aliases terrain `(31,31)`.
Redux ignores off-map probes so unrelated southeast terrain cannot hitch an
edge horse. Gate and direction ordering, floor occupancy, and facing are retained.

**Implementation:** `internal/ai/npc_ai_controller_small_map.go`
**Tests:** `internal/ai/npc_ai_controller_small_map_horse_test.go`

### NPC-0007

**Authored Windemere Devils and limits of actor-specific evidence**

**Kind:** rule
**Uses:** CRIME-0002, CRIME-0004
**Original:** Incomplete
**Redux:** Unverified

The baseline local-data audit reports two Windemere `KEEP.NPC` actors with shape
`0xD8`, Talk zero, and all three destinations fixed: slot 10 at `(13,20,0)` and
slot 11 at `(17,20,0)`. They remain stationary and give no response until guard
activation makes them hostile; this does not turn their actor type into Guard.
Redux retains that distinction. The binary rows were not independently reopened
for this conversion, so their provenance remains narrower than the source rules.

No source-backed Stillwelt schedule override is established. Nor does a review
of general scheduling prove every authored actor or every installed NPC edition.

**Implementation:** `internal/references/npc_schedule.go`, `internal/game_state/guard_activation.go`
