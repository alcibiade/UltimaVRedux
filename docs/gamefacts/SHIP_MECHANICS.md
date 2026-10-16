# Frigates, sailing, and cannon fire

Rules describe inspected DOS paths, not every retail build. `R(a,b)` means a
random request with those bounds; distribution and undefined return values remain
unresolved. Boarding/Exit belong to [vehicles](VEHICLES.md).

### SHIP-0001

**Sails and automatic waits**

**Kind:** rule
**Uses:** TIME-0001, TIME-0002
**Original:** Established
**Redux:** Unverified

Yell switches raised/furled frigate state, preserving facing. Its value-returning
routine omits a value on this branch, so original completion cost is not established.
Raised-sail direction input enables automatic sailing; changing facing does not
move on that invocation and also has an undefined return-value boundary.

For fixed non-calm wind:

```text
n = 1 + (wind_x != movement_x) + (wind_y != movement_y)
waits = n modulo 3
```

Each ordinary wait requests two minutes and one actor phase. With initial wait
count zero, no temporal modifiers, and successful ordinary completion, zero/one/
two waits produce totals 2/4/6 minutes. Calm prevents movement but continues waits
and actor work until interrupted. This is not a universal manual or startup cost.
Redux retains this wait cadence and definitely charges normal outdoor completion
for production sail toggling; undefined original returns cannot prove that cost.

Redux furling stops automatic sailing before its feedback, preserving facing.
Each automatic wait/movement boundary revalidates outdoor exploration and a
raised-sail frigate; invalidation clears the sailing state without another wait,
movement, actor phase, or random draw. Command presses between cadence phases
also stop sailing. Space stops with sails raised and no extra sailing phase;
other commands retain their own ordinary completion costs. Finalization clears
the sailing camera projection and releases the workflow name. A held movement
key across interruption must be released before it can resume movement.

**REDUX DEVIATION: SHIP-0001:** The inspected original tactical-entry/normal-return
path retains the party's sail state and stored sailing direction. Redux stops
automatic sailing and furls the party frigate at every validated tactical entry,
including stationary raised sails and player-initiated encounters. Normal return
leaves sails down and movement under manual control, giving the player a decision
point after combat. Facing is preserved. Ranged-hit interruption instead preserves
sails, as described in [SHIP-0005](#ship-0005).

**Implementation:** `internal/game_state/sailing.go`, `internal/game_state/action_yell.go`
**Tests:** `internal/game_state/sailing_state_test.go`, `internal/game_state/sailing_interruption_test.go`, `cmd/ultimav/gamescene_sailing_interruption_test.go`

### SHIP-0002

**HMS Cape plans affect waits, not every movement**

**Kind:** rule
**Uses:** SHIP-0001, INV-0008
**Original:** Established
**Redux:** Intentional deviation

Original Use requires frigate form and sets the plans high bit, with no off toggle.
Active plans make automatic waits one minute and invoke actors on alternating
waits. They do not halve rowing, other vehicles, or every ship action. Pickup
already writes all plans bits, so Use is not established as the sole activation
route. A separate negative comparison against an unsigned plans byte cannot
establish an operative additional universal speed bonus.

**REDUX DEVIATION: SHIP-0002:** Redux exposes a persisted reversible global
preference usable anywhere after obtaining the plans. This makes enhanced rigging
an explicit player option; its wait timing remains separate from ordinary movement.

**Implementation:** `internal/game_state/action_use.go`, `internal/game_state/sailing.go`

### SHIP-0003

**Hull damage, sinking, collisions, and whirlpools**

**Kind:** rule
**Uses:** ENV-0001
**Original:** Established
**Redux:** Unverified

Frigate damage requests R(1,30): subtract only when hull exceeds damage; equality
or excess sinks it. Prefer a carried skiff preserving facing, otherwise consume
one carpet and request R(0,1) for facing; without either, repeatedly damage the
party until no survivors are reported. Sinking changes shape without decrementing
the old carried-skiff byte, unlike voluntary Exit. Non-frigate hits damage the party.
Repair's cap does not prove every loaded hull assignment is bounded to 99.

Blocked automatic movement at a pier furls sails and stops without hull damage.
Other blockers damage and stop automatic sailing, but do **not** explicitly furl
sails. Already performed waits remain spent; failed movement need not receive
ordinary completion. Whirlpool contact damages; for non-ordinary-foot form it
also removes the contacted whirlpool and transitions to Underworld (34,18).

Redux has authoritative hull/fallback and contact paths. Pier collisions furl;
other blockers stop automatic sailing with sails still raised, applying hull
damage separately. Exact byte-transfer equivalence is not implied.
Its destroyed frigate is not kept. Rough seas on deep water damage skiff/carpet
riders through party damage, separately from hull damage and additional turn costs.

Redux cancels automatic sailing immediately before a sinking frigate is replaced
by a skiff/carpet or drowning begins. The original leaves sailing cleanup to a
later outer-command boundary; Redux does not retain that stale interval. Fallback
movement requires fresh input after the owed attack presentation finishes. Cannon
hits on occupied skiffs damage their riders, not a separate skiff hull; a surviving
skiff is not evidence that the hit was ignored.

Rough-seas requests retain one damage pass per qualifying movement, even when a
programmatic caller submits more while presentation is active. The fixed-name
workflow serializes those passes and preempts normal-priority travel. A request
admitted while the previous owner is finishing input-release cleanup is retained;
normal completion starts its next owner only after releasing the fixed name.
Workflows that need to await the hazard compose its sequence inline, with owner cleanup,
instead of waiting on an independently started child of equal/lower priority.
If a rough-seas pass wipes the party, its batch is closed and remaining old-batch
requests are canceled **before** requesting death recovery. Resurrection cannot
reactivate that batch or damage the recovered party at its new location. A fresh
qualifying request admitted after recovery belongs to a new batch, even while the
old workflow is still releasing ownership; old cleanup does not decrement it.
All surviving passes preceding the terminal hit still resolve exactly once.
Duplicate live workflow names remain programming errors. Current Redux rough-seas
admission remains successful-movement-only; the inspected original also checks
deep water on completed stationary outdoor turns such as Pass. This lifecycle fix
does not establish cadence equivalence.

**Implementation:** `internal/game_state/ship_collision.go`
**Tests:** `internal/game_state/source_hit_test.go`, `internal/game_state/skiff_deep_water_ownership_test.go`, `cmd/ultimav/gamescene_input_largemap_sailing_test.go`

### SHIP-0004

**Repair and retained encounter continuation**

**Kind:** rule
**Uses:** TIME-0006, SHIP-0003
**Original:** Established
**Redux:** Matches

Repair requires frigate form and rejects raised sails. Each of five iterations
performs display/update work, an outdoor actor phase, then checks that the party
still occupies a furled frigate before requesting five minutes. After all five,
add R(1,3), cap hull at 99, and repeat the repair draw while hull is below 10.
At least one draw occurs even at 99. Twenty-seven elapsed minutes is only the
uninterrupted unmodified case including ordinary outdoor completion.

Redux's repair workflow preserves the reviewed phase/check/time order. Tactical
contact suspends the descending actor scan; normal return resumes lower slots
before the anchored-frigate check, without new generation/cadence draws. Living
physical retreat still owes the tail; actual death, invalid origin, cancellation,
or terminal handoff discards it. Match is scoped to that continuation and repair
contract, not every original exceptional return.

**Implementation:** `internal/game_state/action_hole_up_repair.go`, `internal/game_state/outdoor_actor_continuation.go`
**Tests:** `internal/game_state/action_hole_up_repair_test.go`

### SHIP-0005

**Player broadside and pirate contact precedence**

**Kind:** rule
**Uses:** TIME-0004, SHIP-0003
**Original:** Established
**Redux:** Intentional deviation

Player Fire requires frigate shape 20–27 hex and a perpendicular direction. Scan
distances 1–3 for the first eligible monster, excluding whirlpools. Test projectile
clearance only to that target and stop even if blocked. A clear hit subtracts
R(1,20) from an unsigned byte; removal occurs only if the stored result exceeds
127. Exact zero survives this test. There is no ammunition counter.

Pirates resolve wrapped cardinal adjacency as contact before cannon logic. For
normal non-overlapping positions, later collinear cannon opportunities are at
distance 2 or 3. A misaligned original pirate rotates and fires in one invocation.
Slow steps can supply multiple actor phases, not guaranteed shots.

**REDUX DEVIATION: SHIP-0005:** Redux removes cannon targets at zero HP to avoid
the original zero-survivor defect. A misaligned pirate uses one warning/rotation
phase and may fire on a later eligible phase; already aligned pirates fire in
descending source-slot order. The warning is an existing deliberate policy.

**REDUX DEVIATION: SHIP-0005:** Original outdoor ranged hits do not cancel automatic
sailing or furl sails. A successful pirate cannonball or outdoor dragon/sea-serpent
projectile in Redux stops active automatic sailing, preserving a surviving
frigate's facing and sail state. Warnings, rotation, blocked shots, and missed
firing opportunities do not cause this interruption. A concise nonmodal message
is emitted only when sailing actually stops; interruption costs no additional
turn or acknowledgement. Fire and manual furl remain ordinary commands. Releasing
movement and pressing a fresh direction can resume with sails still raised, so
the decision point imposes no required rowing/hoisting escape penalty.

Stopping sailing does not abort the already-started descending outdoor actor
phase: remaining owed hits still resolve exactly once, in order, including a
later sinking or fatal hit. Their retained projectile presentation drains without
starting another sailing wait, movement, or actor phase. Terminal outcomes take
precedence over resumption.

**Implementation:** `internal/game_state/action_fire.go`, `internal/game_state/pirate.go`
**Tests:** `internal/game_state/pirate_test.go`, `internal/game_state/sailing_interruption_test.go`

### SHIP-0006

**Wind selection and water-spawn predicates**

**Kind:** rule
**Uses:** SHIP-0001
**Original:** Established
**Redux:** Unverified

Wind updates first require R(0,63)=0, then select R(0,4). A calm candidate is
redrawn while an additional R(0,255) is below 192. Water selection first requires
R(0,64)<16; surface deep water then tests R(0,7)=7 for a whirlpool. These predicates
alone do not prove statistical probabilities or every region/daylight spawn gate.

Redux uses ordinary large-map generation rather than a separate sailing-only
whirlpool retry. Multiple whirlpools may exist; contact precedes movement, so a
newly adjacent one waits for a later phase. Debug spawns do not prove production
odds. Pirate facings remain world vehicle identities, using boat terrain and
separate occupancy, not the creature-table row reused by original encoding.
