# Tactical combat

This specification covers the 11×11 battlefield, 32-slot combat scheduling,
physical actions, conditions, hazards, loot, and return to exploration. Species
data belong to [Enemies](ENEMIES.md); prepared magic to [Magic spells](MAGIC_SPELLS.md).
Original means inspected DOS-era material labelled 1.16, not an immutable source
release or identified retail executable. R(a,b) denotes a random request, not
proof of external-generator uniformity. D30 reuses
[RNG-0001](ENVIRONMENTAL_HAZARDS.md#rng-0001). Separate calls establish distinct
draw sites, not statistical independence or a general C argument-evaluation order.

### COMBAT-0001

**Battlefield, actor placement, and authored concealment**

**Kind:** rule
**Uses:** ENEMY-0003
**Original:** Established
**Redux:** Matches

Tactical maps are 11×11 and non-wrapping. Surface/town/ship encounters select a
BRIT.CBT scenario; fixed dungeon rooms use DUNGEON.CBT. Dungeon wanderers and
camping synthesize compatible maps. Scenario entry direction supplies party
coordinates. Up to six party members use low slots, ordinary monsters begin at
slot 6, and combatants and objects use fixed 32-entry tables. Monster initialization
and interleaved group draws are ENEMY-0003.

The full authored combat terrain bypasses exploration darkness/LOS, but objects
on terrain 0xFF (Nothing) or 0x87 (Arch) are suppressed in display. Terrain remains
visible. This differs from magical invisibility: concealed creatures retain their
queue/object entries and can act or receive effects. Triggers edit terrain, not
spawn replacements. Replacing a suppressing tile reveals the same occupant next
update; restoring it conceals again without a discovery flag.

Redux uses live map-only terrain, including overrides, for this concealment.
Passive identity, HUD, and targeting assistance respect it; authoritative turns
and manually selected attack/spell effects remain available. Its shared-slot
capacity includes fields, loot, corpses, and combatants. Removing loot frees its
object slot, not a retained dead combat source slot. Summoning capacity depends
on available source/object slots, not historical manager length.

**Implementation:** `internal/game_state/combat_terrain.go`, `internal/game_state/combat_object.go`, `internal/game_state/combat_collection.go`
**Tests:** `internal/game_state/combat_terrain_test.go`, `internal/game_state/combat_object_test.go`

### COMBAT-0002

**Countdown scheduler and exceptional player tails**

**Kind:** rule
**Uses:** TIME-0003
**Original:** Incomplete
**Redux:** Intentional deviation

Scan slots 0–31 repeatedly, skipping empty, dead, object-only, stock-bound, and
persistently dead PC entries. Decrement each eligible countdown; at zero reset it
to 36−stored DEX, advance the persistent actor count, dispatch by current loyalty,
apply the standing hazard, and recount remaining sides. High DEX therefore acts
more often; this is not round-robin initiative. The actor count and minute charge
before every tenth dispatch are canonical in TIME-0003. SAVED.GAM offset 0x02DC
persists that cadence across encounters; entry does not reset it.

The ordinary player-controlled action tail restores ring invisibility/regeneration,
tests every field for expiration, and decrements shared duration before the
scheduler's standing-hazard check. Controlled monsters receive that tail;
monster-loyal actors do not. Numeric active-character selection changes selection
without completing an action. The active-player skip path explicitly completes it.

The recovered sleeping, swallowed, and initial Chaos branches can reach a final
key comparison without initializing its local key value. Their field-expiry and
duration tail is therefore undefined, not guaranteed original behavior.
Redux deliberately resolves this deterministically using loyalty **at dispatch**:
a party-loyal exceptional dispatch receives its standing effect once and duration
decrement, but no invented field-expiry rolls. An already monster-loyal dispatch
receives no player-side duration tail. Initial Chaos interception occurs before
input. Immediate and presentation-stepped scheduling share that policy, preserving
events, forecast, standing effects, draw count, and next RNG state.

**Implementation:** `internal/game_state/combat_turn_queue.go`, `internal/game_state/combat_turn_scheduler.go`, `internal/game_state/combat_status.go`
**Tests:** `internal/game_state/combat_turn_scheduler_test.go`, `internal/game_state/combat_turn_queue_test.go`

### COMBAT-0003

**Commands, movement, triggers, and individual escape**

**Kind:** rule
**Uses:** ENEMY-0002, DNG-0011
**Original:** Established
**Redux:** Intentional deviation

| Command | Original tactical behavior |
|---|---|
| Attack | Separate attack-capable helm, main-hand, off-hand aims, or bare-hand fallback. |
| Cast | PC-only, after remembered adjacent interference and combat gates. |
| Get, Jimmy, Open, Ready, Search, Use | PC-only common-command forms. |
| Klimb | Ladder/grate exit or movement onto adjacent small rock. |
| Push | Test dungeon trigger first, then common push. |
| Yell | Common Yell behavior. |
| Z-stats | Inspect, then complete the PC action. |
| Space | Pass. |
| Arrows | Cardinal step; off-map step attempts fleeing. |
| Escape | Finish only when battle-exit gates allow. |
| 0–6 | Clear/select active member without action consumption. |

Board, Enter, Fire, Hole up, Ignite, Look, Mix, New Order, Quit, Talk, View, X-it,
and unknown keys reprompt without ending the original action. Sleeping/swallowed
PCs do not receive input. Redux deliberately keeps Z-stats read-only and free in
all contexts, preserving current actor, queue, duration, and action for inspection.

Movement checks: off-map coordinates enter flee handling; in-map terrain must
admit the moving shape and not be Nothing. Then scan all 32 object/combat entries
for occupancy. Corpses, underground entries, and fire/poison/sleep fields do not
block; protection fields, living actors (including invisible/sleeping ones), and
loose objects do. Terrain admission never overrides occupancy. An occupant already
on Nothing may leave for a legal square but cannot enter another Nothing cell.
Arch concealment itself adds no movement ban.

The first departing PC records an exit direction. Fixed dungeon-room departures
must subsequently use that edge or ladder direction. PCs aboard ship cannot flee
individually. Successful movement in dungeon combat tests the destination trigger;
small-rock Klimb, aimed Push, and an unblocked physical or launch-based spell
endpoint also test it. Traversed cells and blocked endpoints do not. A trigger is
consumed for this loaded encounter and can replace two independently addressed
terrain destinations.

Redux additionally offers a post-victory group exit after a route is established,
avoiding repetitive individual departures; original physical edges/ladders remove
one PC at a time. Collision and established-route requirements remain authoritative.

**Implementation:** `internal/game_state/combat_terrain.go`, `internal/game_state/combat_map_trigger.go`
**Tests:** `internal/game_state/combat_map_trigger_test.go`, `internal/game_state/combat_flee_victory_test.go`

### COMBAT-0004

**Physical attacks, scattering, and controlled-monster uncertainty**

**Kind:** rule
**Uses:** EQUIP-0012, EQUIP-0013, EQUIP-0015, EQUIP-0016, EQUIP-0017
**Original:** Incomplete
**Redux:** Intentional deviation

Equipment rules own source order, per-item range, hit threshold, damage, ammunition,
and special weapons. Physical bludgeoning uses STR; other physical attacks use
DEX. Monster damage starts with its fixed rating, not a PC weapon damage draw.
Use encounter DEX and effective sleep/Mimic/Time Stop overrides. A successful hit
still needs projectile arrival and positive post-modifier damage.

An ordinary miss scatters to an adjacent in-bounds cell around the aim and can hit
another occupant; terrain may stop it. Morning Star/Halberd bypass obstruction
only on successful primary hits. The original does not recheck range after
scatter. Redux deliberately limits physical scatter to the source's normal
floor-Euclidean maximum range, making displayed and damaging endpoints obey the
same range contract. This covers PCs, controlled and hostile monsters, reach
weapons, and forced misses; accepted in-range scatter and primary hits retain
their normal rules.

The original controlled-monster player-attack path passes sentinel 255 into a
weapon-indexed path. It satisfies a literal magical-weapon comparison but also
indexes a range table outside its valid domain and fails to establish a monster
attribute route. The comparison alone cannot prove reliable magical damage or
retail controlled-hit behavior. Bare-hand sentinel and rare equipment mapping
uncertainties remain in EQUIP-0012/EQUIP-0017.

Redux explicitly classifies controlled melee and physical missiles like hostile
physical attacks, including undead halving, immortality, poison fallback, and
surviving-hit flee reevaluation. Magic-capable species still participate in
missile suppression/Turning checks; control does not itself upgrade physical
damage to magic. This is an approved coherent combat policy under unresolved
original behavior, not a blanket parity claim.

**Implementation:** `internal/game_state/combat_formulas.go`, `internal/game_state/combat_controlled_monster.go`, `internal/game_state/combat_loyalty_attack.go`, `internal/game_state/combat_ranged_attack.go`
**Tests:** `internal/game_state/combat_controlled_monster_test.go`, `internal/game_state/combat_miss_range_test.go`

### COMBAT-0005

**Loyalty, sleep, captivity, and attack-side special effects**

**Kind:** rule
**Uses:** RNG-0001, ENEMY-0001
**Original:** Incomplete
**Redux:** Intentional deviation

Dispatch and enemy eligibility use current loyalty rather than original side.
A party-loyal swallowed PC attempts D30 < DEX to escape but loses that turn either
way. A sleeping party-loyal PC wakes when R(0,255)<16 and still loses its turn;
monster-loyal sleepers use R(0,16)=16 instead. Monster-loyal underground rejection
precedes both wake/release RNG. Poison has no automatic tactical-turn HP tick;
exploration poisoning is a separate update.

A Chaos wielder is charmed and routed toward the party instead of receiving input.
Gazer special hits sleep a target rather than damage it unless already asleep.
Compound poison-ability hits test R(0,3)≠0 and replace ordinary damage with
toxification: Good PCs become Poisoned without a fallback draw; other eligible
targets take R(0,20). Food-stealing success uses the same three-of-four selector
outcomes, removes up to five Food, and replaces ordinary damage. These selector
counts correct misleading one-in-four comments; original probabilities remain
conditional on the generator.

Redux recognizes the full name Saduj for intrinsic hostility instead of an
original character-position heuristic, preventing unrelated names from matching.
Source and clone Saduj remain hostile independently of possession. Exact
`CombatTurnIdentity` owns sleep, underground, and possession; a clone does not
inherit live state by consulting its source flag. HP and roster status retain
their shared gameplay authority. Controlled monsters can attack hostile Saduj
entries by melee or missile. Unreadying an Invisibility Ring clears only the
acting PC/clone and refunds the item without an extra draw.

**REDUX DEVIATION: bounded enemy possession.** Original enemy possession has no
natural countdown. Redux restores agency by releasing a surviving affected exact
PC/clone no later than the end of its twelfth own completed scheduled turn.
Turns lost to sleep, captivity, Time Stop and Quickness still count. Resolution
follows the action and standing hazards, before selection/end-condition checks;
other actors, forecasts, rendering and presentation delays never advance it.

| Completed turns | Base percent | INT 10 | INT 20 | INT 30 |
|---|---:|---:|---:|---:|
| 1–3 | 0 | 0% | 0% | 0% |
| 4–6 | 3 | 2.25% | 3% | 3.75% |
| 7–9 | 8 | 6% | 8% | 10% |
| 10 | 20 | 15% | 20% | 25% |
| 11 | 35 | 26.25% | 35% | 43.75% |
| 12 | guaranteed | 100% | 100% | 100% |

These are independent per-check chances, not cumulative probabilities. For
turns 4–11, clamp current shared character INT to 1–30 and compute
`thresholdBP = floor(basePercent * 100 * (20 + INT) / 40)`; exactly one gameplay
RNG draw in 0–9999 recovers when below that threshold. Turns 1–3 and 12 use no
resistance draw. These new draws intentionally change gameplay RNG progression.

Each successful enemy acquisition records a unique episode, the controller's
stable encounter combatant ID, and progress under the complete turn identity
(including clone Slot). Failed/redundant acquisition never resets it. Dispatch
captures the episode; duplicate/stale completion cannot tick a replacement.
A possessed clone copies controller/progress by value but receives a fresh
episode. Monster clones receive fresh controller IDs and inherit no victims.

Defeating, actually escaping off-map, or successfully replacing the responsible
controller with a rat releases all its current victims in stable exact-actor
order without RNG. Beginning flight, teleportation, invisibility, allegiance
changes, administrative removal and a Daemon's successful-possession retirement
do not release them. Rat replacement grants no kill rewards. This replaces the
source-unaware disappearance/Shadowlord-escape recovery; unrelated controllers'
victims remain possessed. Controller names and locations are not disclosed.

Natural/controller release clears only possession, preserving HP, equipment,
sleep, poison, captivity and intrinsic Saduj hostility. It grants no extra turn,
resources or time. Chaos Sword control and charmed monsters remain separate
untimed mechanics. An Xen Ex retains its immediate intelligence-save/resource
semantics: failure preserves progress; success clears only the exact episode
and retains the spell's existing roster-status behavior. Emergency pass-out
recovery remains separate under COMBAT-0011. Death (including shared source/clone
death), departure, replacement, clone deletion and teardown discard episode
metadata. Combat is excluded from native snapshots; no save schema is added.

Surviving episodes emit resistance messages at turns 3, 7 and 10; recovery
replaces a same-turn milestone. Typed resistance/release events feed the report,
with distinct clone names and no duplicate generic allegiance announcement.
Condition text shows `Possessed - 8/12 turns` only for genuine timed episodes;
Help explains own-turn counting, intelligence, the cap and rescue alternatives.
Feedback and episode-guarded onset effects own no gameplay work. Directed-attack
awakening remains the separate COMBAT-0006 policy.

**Implementation:** `internal/game_state/combat_status.go`, `internal/game_state/combat_melee_engagement.go`, `internal/game_state/combat_rings.go`, `internal/game_state/combat_possession.go`
**Tests:** `internal/game_state/combat_status_test.go`, `internal/game_state/combat_saduj_allegiance_test.go`, `internal/game_state/combat_loyalty_attack_test.go`, `internal/game_state/combat_possession_test.go`

### COMBAT-0006

**Redux directed-attack awakening**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Intentional deviation

Original nonlethal attacks, including zero damage, preserve PC sleep. Redux wakes
a living sleeping PC at authoritative resolution of an attack directed at that
exact battlefield entry, for intuitive threat response. Miss, graze, absorbed
armor damage, resisted offensive magic, and blocked committed projectiles qualify.
An actually struck PC also qualifies, including collateral/friendly fire.
Melee, physical missiles, magic, controlled and possessed attackers share the rule.

The effect occurs at resolution, never launch or presentation, emits one wake
event, and clears that copy's tactical sleep and shared roster Sleep while retaining
other roster statuses. Other copies keep their tactical conditions. Lethal outcomes
remain death. Initial Gazer sleep and newly delivered sleep-field effects survive;
a later directed attack may wake the sleeper. Protection/cloning, nearby cone
victims, standing fields, environmental hazards, and exploration hazards do not
qualify without a directed attack.

No extra RNG, bonus turn, or queue reorder occurs. A woken PC acts at its next due
turn. Guarded camping, spontaneous wake checks, and combat-exit sleep persistence
retain their own rules; an unguarded ambush uses this same attack policy.

**Implementation:** `internal/game_state/combat_status.go`, `internal/game_state/combat_field_effects.go`
**Tests:** `internal/game_state/combat_attack_wake_test.go`

### COMBAT-0007

**Field creation, expiration, and standing-hazard order**

**Kind:** rule
**Uses:** RNG-0001
**Original:** Incomplete
**Redux:** Unverified

Field projectiles have range 15 and create at a reached endpoint; fields can
overlap actors or fields. Protection blocks movement; other fields do not.
All permit missile travel. Fixed dungeon field descriptors instantiate the same
four types. An Grav deletes the first selected field, not every overlapping one.

Ordinary player-side action completion separately tests each field with R(0,255)<16
for expiration. The undefined tails in COMBAT-0002 cannot promise that check.
After every dispatched action, terrain has hazard priority; otherwise the first
matching harmful field in ascending object order applies:

| Hazard | Standing effect |
|---|---|
| Lava, fireplace, fire field | R(0,10) damage, no armor roll. |
| Swamp, poison field | Original base-tile gate admits tiles below the monster-family boundary; toxify then poisons a Good PC or uses R(0,20) fallback damage. |
| Sleep field | Sleep without an INT save. |
| Protection field | No standing damage. |

The poison gate is **not** an unconditional “all monsters ignore standing poison”
rule: humanoid-shaped non-PC entries and changed appearances require attention.
Redux's non-PC standing-poison helper excludes species indexes 16 and above,
while permitting humanoid archetypes below 16. Purple changes the acting entry's
base appearance to Rat, suppressing standing poison and its repeat-contact damage.
Redux preserves that protection per tactical entry, including clones; it takes
effect before the same action's standing check. It neither cures Poisoned status
nor replaces character identity/stats/equipment. Black potion replaces the base
appearance and ends Purple protection; Sanct Lor changes rendered appearance only
and retains that protection. Direct field contact differs from standing contact:
direct fire uses rating 21 then armor, direct poison uses toxify, and direct sleep
has no save. Terrain precedence, exact identities, and unattributed death handling
remain core state operations, not presentation decisions.

**Implementation:** `internal/game_state/combat_field_effects.go`
**Tests:** `internal/game_state/combat_field_test.go`, `internal/game_state/findings_closeout_test.go`

### COMBAT-0008

**Monster decision order, target selection, and low-HP fleeing**

**Kind:** rule
**Uses:** RNG-0001, ENEMY-0001, ENEMY-0002
**Original:** Incomplete
**Redux:** Known discrepancy

The original monster-loyal action order is:

1. Time Stop suppresses; Quickness requires R(0,1)≠0; underground actors do nothing.
2. Sleep checks wake then stops the turn.
3. A fleeing non-PC may gain 1 HP when R(0,3)=3, reevaluates condition, and seeks.
   Even if fleeing clears, it does not restart special/attack processing this turn.
4. Try possession, invisibility, then gating.
5. Try an in-range melee/missile attack; otherwise take a greedy movement step.

Nearest-target search scans 31→0 and retains only strictly closer targets, so
equal distances favor the earlier reverse-scan entry. Ignore underground and
normally invisible enemies; Shadowlords and Doom can see invisible targets.
Distance is floor(sqrt(dx²+dy²)). Movement is greedy, not pathfinding: a preferred
axis step then up to four random cardinal attempts. Reapers/Mimics do not move.
Teleport-capable actors, unless suppressed by Crown or Negate Magic, try one
random coordinate when boxed or when R(0,3)≠3.

Crown and Negate Magic suppress the three specials. Possession samples one raw
slot 0..31; an eligible visible, awake, above-ground, uncharmed PC consumes the
special action whether its INT save succeeds or fails. Successful Daemon
possession removes the Daemon; Shadowlord possession does not. Invisibility toggles
on R(0,255)<32. Gating is tested afterward with that same threshold and requires a
legal Daemon-shaped destination plus free slots. Mimics always attempt their
in-range missile branch; other monsters require R(0,255)<128.

Redux successful acquisition and controller-removal resistance semantics are
defined by COMBAT-0005. Daemon retirement is explicitly not controller defeat;
its victims keep their bounded episodes even after it disappears.

Fleeing uses q=floor(maximum HP/4): below q always flee; below 2q request R(0,256)
and flee when greater than 251; otherwise clear fleeing. The original comment's
1/64 and the requested upper endpoint conflict. Inclusive uniform 0..256 would
give 5/257, but the external function is unavailable. Redux concretely draws
0..255 and tests >251 (4/256). This difference is retained as a discrepancy/evidence
limit, not relabeled an intentional fidelity adjustment.

**Implementation:** `internal/game_state/combat_monster_condition.go`, `internal/game_state/combat_daemon_ai.go`
**Tests:** `internal/game_state/combat_monster_condition_test.go`, `internal/game_state/combat_daemon_ai_test.go`

### COMBAT-0009

**Division, death attribution, and treasure**

**Kind:** rule
**Uses:** RNG-0001, PROGRESS-0001
**Original:** Established
**Redux:** Intentional deviation

An attributed monster kill awards floor(monster maximum HP/4)+1 XP to the killing
PC, not divided among the party. Field/unattributed paths may kill without XP.
On ordinary corpse-eligible terrain, request D30: a result ≤ treasure rating
creates a chest; only then request a separate D30 and mark trapped if it is
strictly < rating. A failed drop leaves a corpse and consumes no trap draw.

Gazer death instead leaves a corpse and attempts an Insect at the same square;
Gargoyle death changes terrain to Small Rocks and leaves no object. These occur
before ordinary water/Arch suppression. No-corpse, disappearance, low-index water,
and Arch have separate deletion behavior before ordinary treasure. Capacity
constraints do not authorize dropping required draw sites.

A surviving Slime/Gargoyle tries up to eight random adjacent positions for division.
The first legal child is a fresh hostile entry with the parent's surviving HP,
not its charm/status flags; an allied parent stays allied. Original descendants
can divide repeatedly. Redux defaults to one successful split per encounter-created
monster, marking its child used as well, to prevent exponential growth. The
AllowRepeatedMonsterDivision gameplay setting restores repeated division.
Another marked Redux choice treats a fully deflected ordinary PC weapon/bare-hand
attack as unsuccessful and skips post-hit division; original surviving zero-damage
applications can reach division. This scope matters when comparing callers rather
than the shared division helper alone.

Redux preserves the separate drop/trap calls, special-death ordering, shared
capacity, and fresh hostile child semantics. These are scoped comparisons, not
proof of binary PRNG sequences or every malformed capacity edge.

**Implementation:** `internal/game_state/combat_formulas.go`, `internal/game_state/combat_special_death.go`, `internal/game_state/combat_monster_division.go`
**Tests:** `internal/game_state/combat_treasure_test.go`, `internal/game_state/combat_special_death_test.go`, `internal/game_state/combat_monster_division_test.go`

### COMBAT-0010

**Ground display, collection, and chest knowledge**

**Kind:** rule
**Uses:** INV-0004
**Original:** Established
**Redux:** Intentional deviation

Persistent original ground display has stable slot precedence, not elapsed-time
cycling or universal chest priority. In the descending scan, ordinary objects
overwrite earlier ordinary objects, so the lowest ordinary slot wins. Fields and
dead-PC/dead-monster markers write only an unclaimed cell: with no ordinary object,
the highest such slot wins regardless of marker type. Terrain concealment still
applies. Temporary chest-opening reveal shows each admitted item and description,
then returns to stable ground display.

Redux preserves that ordering. Its field overlay cannot cover a winning item/body
or temporary reveal; display queries do not alter slots, collisions, inventory, or
RNG. Get skips noncollectible markers and selects chests/items by ascending original
slot; an unopened selected chest requires Open. Collection never removes the party
body or changes the dead character. Each chest-generated item requires a free
shared object slot.

Redux deliberately stores an honest party-shared assessment per encounter-local
chest instead of repeated fallible inspection reports. Known-safe Jimmy refuses
harmlessly; blind safe attempts and failed armed attempts retain indistinguishable
one-Key cost. Successful Jimmy/An Sanct disarms and certifies that exact object.
Overlapping chests have separate knowledge; nested loot and reused slots begin
uninspected. Dungeon tactical chests use this object model, not first-person
dungeon DEX/depth inspection. This avoids repeated guesswork without merging
distinct objects or allowing presentation to own trap state.

**Implementation:** `internal/game_state/combat_loot_presentation.go`, `internal/game_state/combat_collection.go`, `internal/game_state/combat_chest.go`
**Tests:** `internal/game_state/combat_ground_display_test.go`, `internal/game_state/combat_chest_assessment_test.go`

### COMBAT-0011

**Victory, defeat, restoration, and outdoor continuation**

**Kind:** rule
**Uses:** ENEMY-0008
**Original:** Established
**Redux:** Matches

Count living entries by current loyalty. No hostile entries latches victory,
but tactical movement/looting continue. Escape normally requires victory, with
a separate no-PC end condition; fixed dungeon-room Escape is rejected while PCs
remain. Losing all party-loyal entries attempts possession recovery, otherwise
loses the battle. A surviving allied creature does not fabricate victory from
the no-PC departure condition.

The independent no-friendly/no-opponent emergency recovery still frees one
possessed actor and makes it pass out, with its established Chaos Sword handling.
It also clears that exact episode's metadata. It is not invoked as a second
recovery after controller defeat/escape: COMBAT-0005 owns those source-specific
releases, which neither sleep the victims nor remove their equipment.

Original return restores source map, coordinates, and objects before exploration
tests all-dead recovery. Pirate salvage requires retained victory. Redux requests
actual party-wipe recovery exactly once after combat clears and origin restoration
completes; later fatal world effects retain their own recovery owner.

A world actor-phase encounter suspends its descending cursor. A living return,
including physical retreat reported as CombatDefeat, resumes lower actors after
restoration/workflow release, then owed slow-movement or repair work. Death,
terminal handoff, cancellation, or invalid origin discards it. The defeat label
alone is insufficient to discard a living continuation. ENEMY-0008 owns the
exact-once, consecutive-encounter, and unsupported-pending-save contract.

**Implementation:** `internal/game_state/game_state_combat.go`, `internal/game_state/outdoor_actor_continuation.go`
**Tests:** `internal/game_state/combat_flee_victory_test.go`, `internal/game_state/outdoor_actor_continuation_test.go`

### COMBAT-0012

**Redux attack assistance and presentation contracts**

**Kind:** rule
**Uses:** COMBAT-0004
**Original:** Established
**Redux:** Intentional deviation

Tab automates the existing per-item manual attack sequence as one action per
accepted press, not continuous AI. Choose the closest visible eligible hostile
reachable by at least one available equipped source from the current square;
break ties by lowest HP, then monster index. Only when no attack is available,
choose a one-step melee approach among complete reachable routes using distance,
HP, route length, and index. This is not optimal firing-position search.

Sources retain helm/main/off-hand order and independent ranges; melee zero
normalizes to one in existing integer-distance geometry. Bare hands applies only
when no offensive equipment is readied, not when a bow is empty. Availability
uses live ammunition, readied identity, and remembered melee interference.
Shot checks use sampled projectile collision, final-sample exception, and successful
Morning Star/Halberd bypass; they do not substitute walking passability or add
intermediate-actor interception.

The auto sequence retains the exact chosen monster, not just a reusable slot or
cell, and revalidates every source after impact. Unavailable/blocked/out-of-range
sources are skipped without expenditure; death/ineligibility stops the sequence,
never retargeting a replacement. Exact PC/clone identity supplies origin. Existing
confirm/commit/impact routes execute; scene cleanup and returning projectiles
finish before the next source or action completion.

Preview is detached and pure: no RNG, state mutation, target memory, cursor
advance, time/action cost, or output. Before aiming markers consider any available
source; while aiming they represent only the current source. Hidden actors gain
no names/markers; Doom retains its existing visibility semantics. A clear shot is
eligibility, not a guaranteed hit or scatter prediction.

Original remembered adjacent interference rejects Bow, Crossbow, Magic Bow, Oil,
Sling, and Cast before aiming/payment and reports the interrupter. Redux preserves
that boundary and emits one structured interruption per rejected attempt; pure
queries emit none. Manual entry, confirmation rechecks, and Tab share it. Battle
Report freezes identity/disclosure at event admission instead of renaming history
after slot reuse. An additional interrupted-action cue is an intentional
presentation extension; its recording slot remains silent until assigned.

Original missile presentation advances ordinary shots 13 native pixels per 40 ms,
rocks 8 outside towns, and town shots 6. Redux uses the corresponding 0.325, 0.2,
and 0.15 pixels/ms rates without display-scale changes, but deliberately smooths
Euclidean travel with a 100 ms minimum so short flights remain perceptible. This
does not change resolved endpoints, damage, or gameplay RNG.

Committed possession, teleport, gating, and Corpser events drive feedback without
rerolls or duplicate creation. Gate presentation intentionally uses 125 ms arrival,
600 ms bright-leading-silhouette materialization, and 125 ms settling for readability;
hitches retain all phases. Ally/Enemy text and a one-source-pixel cyan/red outline
outside the black HP frame show current side; HP size/color/opacity still express
health. Exact clone identity, visibility, possession, and recorded summon/charm
provenance come from the authoritative projection, never presentation writes.

**Implementation:** `internal/game_state/combat_attack_preview.go`, `internal/game_state/combat_auto_action.go`, `cmd/ultimav/gamescene_projectile.go`
**Tests:** `internal/game_state/combat_attack_preview_test.go`, `internal/game_state/combat_auto_action_test.go`, `internal/game_state/combat_interruption_test.go`
