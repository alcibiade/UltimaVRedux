# Environmental hazards

`R(a,b)` denotes the inspected random request, not proof of its distribution.
Ordinary context completion is canonical in TIME-0002; successful movement is
not the only way to trigger a standing hazard.

### ENV-0001

**Ordered party damage**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Unverified

Visit slots 0–5 ascending. For each active non-Dead member, request separate
R(1,8) damage; Sleeping and Poisoned members participate. There is no armor/DEX
save in this helper. Zero-or-lower HP clamps to zero, sets Dead, and clears that
member's active designation. Separate calls do not establish statistical independence.
Redux's hazard consumers use living active-member sweeps, but caller-specific
presentation and complete exceptional-state equivalence are separate questions.

### RNG-0001

**Transformed d30 and probability limits**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Unverified

The original helper takes R(0,60), integer-divides by two, and maps zero to one.
If and only if the external input is uniform over all 61 endpoints, output 1 has
four preimages, outputs 2–29 have two each, and 30 has one. The external generator
and exact sequence are not recovered, so these are conditional probabilities.
Direct R(1,30) callers are not this helper. Future combat and observation rules
reuse this definition rather than deriving incompatible d30 conventions.

### ENV-0002

**Town trapdoors, Stonegate, and swamp/burning gates**

**Kind:** rule
**Uses:** ENV-0001, TIME-0002
**Original:** Established
**Redux:** Intentional deviation

After ordinary town minute charge: sleeper wake checks use R(0,15)=15, then
standing terrain, then player update. An unprotected trapdoor damages, descends
one floor, reloads with zero added time, and repeats on another trapdoor. Riding
carpet form protects; owning a carpet does not. No explicit minimum-floor check
was found. Grate 86 hex is manual downward Klimb, distinct from automatic trapdoor
8C; grate Klimb does not itself call trap damage and rejects horse riding.

At Stonegate, the unprotected trapdoor instead damages once, changes all 1,024
loaded terrain bytes to lava, clears loaded objects, and sets every active member
Dead/zero HP without descending. The claimed eight-cell ring and absent grates
require authored KEEP.DAT verification; loader offset 1800 hex alone proves neither.

Town swamp requires exact ordinary-foot form; for non-Dead/non-Poisoned members,
R(0,29)>DEX poisons, including sleepers. Outdoor swamp instead requests R(1,30).
Invisible-foot fails the exact form gate. Fireplace/lava has no general riding
immunity. Existing poison is a separate player-update consequence (PROV-0001).

Redux validates and commits Stonegate damage/flood/object removal/wipe synchronously,
preserving inactive roster members and rejecting stale review without RNG.
**REDUX DEVIATION: ENV-0002:** its owned cinematic presents an already committed
outcome, rather than interleaving blocking original effects and mutation. Cleanup
cannot undo consequences or add another minute; one owner hands off recovery.
The flood clears superseded door/doorway overrides on that floor and retires its
tracked door timer without restoring the closed tile. Unrelated overrides and
other-floor timers remain. **REDUX DEVIATION: ENV-0002 / CMD-0006:** an original
pending closure can write a door after the flood; Redux prevents that resurrection
so the lava reveal contains no superseded room doors.

**Implementation:** `internal/game_state/stonegate_trap_service.go`, `internal/game_state/environmental_integration.go`, `internal/game_state/action_fall_through.go`, `internal/game_state/stonegate_doors_test.go`, `cmd/ultimav/gamescene_stonegate_doors_render_test.go`

### ENV-0003

**Waterfalls and mountain climbing**

**Kind:** rule
**Uses:** RNG-0001, ENV-0001, TIME-0002
**Original:** Established
**Redux:** Unverified

Waterfalls move south twice, then each non-dead member takes one HP when transformed
d30 is at least DEX. Restore party presentation; resulting (54,138) transitions
to the Underworld. There is no intrinsic minute charge. Under the conditional
uniform-input assumption in RNG-0001, failure is certain at DEX 1 and
`(61 - 2*DEX)/61` at DEX 2–30; this is not a recovered RNG guarantee.

The original has two distinct waterfall admission points. Ordinary consumed
outdoor completion requests its modified two minutes, processes terrain and
earthquake consequences, updates the party, processes rough seas, then calls the
fall when the **standing tile** is a waterfall, followed by the actor phase.
Separately, the outer pre-input path can call a fall when the cell immediately
south in its visible map buffer is a waterfall; that route skips normal command
completion. Neither forced southward step adds positive time, a player update,
or an actor phase. Thus the actor phase owed by an initiating consumed command
occurs after its first complete fall and before subsequent automatic falls;
those subsequent falls do not repeat the command or actor tail.

Mountain Klimb checks grapple possession, exact walking form, direction, then
Mountains destination (not Peaks). Each non-dead member failing R(1,30)>DEX takes
R(1,5) damage; sleepers participate. Move afterward even when members fell, without
consuming the grapple. Redux has separate waterfall/Klimb authorities; exact
random-stream, abnormal party, and all presentation comparisons remain unverified.

Redux checks the tile immediately **south** of the authoritative large-map
position at consumed surface completion, after the initiating command's calendar
charge and player update, before its owed actor phase. This includes Pass while
already at a trigger. Blocked outdoor movement and explicitly free commands do
not create a checkpoint or charge. Movement's existing terrain costs remain with
its completion owner; the waterfall does not replay ordinary turns.

One retained logical owner controls the cascade, with at most one live waterfall
workflow. Each descent makes two wrapped southward steps with the short inter-step
delay, then applies the existing ordered Dexterity checks and one-HP failure
damage, and performs the canonical surface
entrance transition when appropriate. The initiating command's actor phase runs
**once after the first completed descent**, before another automatic fall.
Automatic sailing stops when the hazard takes ownership. Later falls add no new
input requirement, calendar charge, food/status sweep, or actor phase.
There is no extra 2.5-second post-step presentation wait. Damage feedback, required
transitions and owed actor work finish before input returns; the nonblocking falls
notice may remain. Held movement must release before a fresh press resumes travel.

If that phase yields to a projectile, tactical contact or whirlpool, the current
waterfall workflow finalizes and releases input while the logical cascade retains
its visited states and completed actor-tail flag. Existing outdoor actor cursors
preserve source order through multiple contacts without rerolling generation or
cadence. After those owners finish, the pre-input continuation boundary reacquires
the waterfall workflow and checks the **current map, floor and landing**. A
validated owned whirlpool return can update that landing; unrelated relocation
cannot. Tactical input remains usable during the handoff. No ordinary surface
command can slip between contact completion and automatic cascade resumption.

**Current Redux comparison limits:** Redux retains its existing wrapped raw-map
south-tile predicate for consumed admission and landing rechecks. This differs
from the original standing-tile consumed checkpoint and does not establish
equivalence with the original visible
buffer's pre-input south-cell predicate. Visibility/buffer correspondence remains
unverified. Rough-sea admission remains with its existing successful-movement
caller; an already queued rough-sea workflow completes before the waterfall
workflow starts. The waterfall does not request a second rough-sea hit or expand
that hazard's cadence.

The logical cascade records visited `(location, floor, position)` trigger states.
If a wrapped chain returns to one, it stops at the already committed landing and
emits one nonmodal notice rather than holding input forever. Actual party death,
a terminal handoff, invalidated ownership (including unrelated relocation or
combat replacement), or queue shutdown stops remaining fall work. A living retreat
reported as `CombatDefeat` preserves the owned outdoor actor continuation and
resumes the cascade after that phase drains; the combat result alone does not
cancel it.
Committed steps/damage are retained; teardown does not execute an abandoned actor
tail. Each workflow instance has one finalizer for owned fade/flash cleanup and
input release. Resumption waits for that instance to finalize and never starts
from its cleanup. Repeated reviews add no workflow, damage, or cost. Native capture
rejects the retained cascade even between workflow instances; shutdown clears it
even when only an outdoor actor continuation remains.

**Implementation:** `internal/game_state/waterfall_hazard.go`, `internal/game_state/waterfall_continuation.go`, `internal/game_state/outdoor_actor_continuation.go`, `internal/game_state/turn.go`, `cmd/ultimav/gamescene_input.go`, `internal/game_state/action_klimb.go`
**Tests:** `internal/game_state/waterfall_continuation_test.go`, `internal/game_state/waterfall_actor_continuation_test.go`, `cmd/ultimav/gamescene_waterfall_continuation_test.go`

### ENV-0004

**Bridge toll and outdoor hazard order**

**Kind:** rule
**Uses:** ENV-0001, PROV-0001, TIME-0002
**Original:** Established
**Redux:** Intentional deviation

Bridge trolls first request R(0,7), proceed only on zero, then require exact foot
form. Test non-dead, awake members in order; first R(1,30)>DEX invokes toll/combat
and stops later tests. Toll is `99 - 3*Strength` of the first Good/Poisoned member.
Accepting subtracts first; negative gold restores the subtraction and starts
combat. Refusal starts combat without payment.

Outdoor completion orders time, bridge/swamp/lava or special passage, earthquake,
player update, deep-water rough-sea damage for skiff/carpet riders, waterfall,
then actor phase. Earthquake requires nonzero level and R(0,255)=69 hex; original
presentation consumes additional random calls before party damage. Poison applies
at every completed player update, not just movement.

**REDUX DEVIATION: ENV-0004:** Redux explains an unaffordable accepted toll and
waits for acknowledgement before combat, keeping the transcript readable without
changing payment/time ownership. Its random earthquake presentation is nonblocking
and deterministic. Exact original probabilities are not inferred from Redux RNG.

**Implementation:** `internal/game_state/environmental_integration.go`

### ENV-0005

**Dungeon pit consequences are cell-owned**

**Kind:** rule
**Uses:** DNG-0007, ENV-0001
**Original:** Established
**Redux:** Intentional deviation

Use DNG-0007 for mutation/destination/damage order; do not move damage before
destination access. **REDUX DEVIATION: ENV-0005:** a bottom-floor Redux hidden pit
reveals and damages once on that floor, avoiding the original out-of-range floor-8
access. Undefined original storage access is not a verified safe dungeon exit.

**Implementation:** `internal/game_state/dungeon_hidden_pit.go`
**Tests:** `internal/game_state/dungeon_hidden_pit_test.go`
