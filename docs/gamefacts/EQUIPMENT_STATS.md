# Equipment mechanics

This describes scoped DOS equipment behavior. Table values belong to
[the equipment catalog](EQUIPMENT_ORIGINAL_DATA.md#equip-0001); layout belongs to
[the save format](SAVED_GAM_STRUCTURE.md#save-0002). Unknown compiler/runtime paths
are not silently converted into intended behavior or a Redux design decision.

### EQUIP-0010

**Carried stock and Ready session**

**Kind:** rule
**Uses:** EQUIP-0001, SAVE-0002
**Original:** Established
**Redux:** Unverified

Carried stock is party-wide; each character owns its equipped slots. Ready lists
carried and already-equipped items. Selecting an equipped item removes it and
returns one copy only while carried stock is below 99. Lower-level removal by
destruction or disappearance does not automatically refund an item.

Ordinary selections keep the Ready session open; Escape closes it. A disappearing
ring also ends it. In the inspected tactical route, the session is one action,
not a separate action for each selection. Body armor cannot be equipped or removed
during unresolved combat; this restriction ends after victory and does not cover
all equipment categories.

### EQUIP-0011

**Slot admission and cumulative Strength**

**Kind:** rule
**Uses:** EQUIP-0001, EQUIP-0010
**Original:** Established
**Redux:** Intentional deviation

The original requires the sum of the Strength costs in all six occupied fields
plus the new item to be at most the character's Strength. Occupied slots must
first be cleared. A one-handed item chooses the first hand when both are free,
otherwise whichever hand is free. Two hands must be available for a two-handed
item, which is stored once in the first hand and reserves the other.

Arrows and Quarrels cannot be readied. Bow/Magic Bow require an Arrow; Crossbow
requires a Quarrel. No profession check appears in this admission path.

**REDUX DEVIATION: EQUIP-0011** — Redux permits confirmed atomic replacement for
usability. It computes the final loadout, subtracts displaced costs, adds the new
cost, and returns displaced items to carried stock. A two-handed replacement can
displace both hands. A full return stack rejects replacement instead of silently
losing equipment; changed equipment or Strength invalidates the reviewed plan.
Ordinary explicit unready retains the capped-refund contract.

**Implementation:** `internal/game_state/ready_equipment_preview.go`, `internal/game_state/action_ready.go`
**Tests:** `internal/game_state/ready_equipment_preview_test.go`, `internal/game_state/action_ready_integration_test.go`

### EQUIP-0012

**Attack order, range, and hit test**

**Kind:** rule
**Uses:** EQUIP-0001, EQUIP-0002
**Original:** Incomplete
**Redux:** Unverified

Attack-capable items are considered in helmet, first-hand, second-hand order,
with separate aiming. A two-handed item is attacked once. No offensive equipment
selects bare hands; the sentinel's complete accuracy path remains unresolved.
Adjacent handling has aiming distance one. Distance is floor-Euclidean, so diagonal
adjacency also has distance one.

For ordinary physical weapons, use Strength for the bludgeoning category and
Dexterity otherwise:

```text
threshold = trunc((30 + target_effective_dexterity - attacker_attribute) / 2)
hit if combat_roll() >= threshold
combat_roll(): halve the result of random_call(0, 60); replace zero with one
```

The underlying random generator's distribution was not established; this is not
a conventional uniform d30. Sleep, Mimic identity, and monster Time Stop conditions
can supply effective Dexterity 1 without making every attack automatically hit.
Glass, Chaos, and Jeweled Sword bypass this hit test when evaluated as those weapons.
Hit testing does not guarantee projectile arrival or final HP loss.

### EQUIP-0013

**Damage, magical classification, and special 99**

**Kind:** rule
**Uses:** EQUIP-0001, EQUIP-0012
**Original:** Established
**Redux:** Unverified

Bare-hand rating is 1. Ordinary ratings above 1, except 99, request a draw from
1 through the rating. Rating 1 stays 1. Glass Sword requests destruction and uses
99; Jeweled Sword replaces its reference rating with zero. Other special 99
ratings remain 99 in this stage.

Unless rating is 99, subtract a draw from 1 through nonzero target armor. There is
no added Strength damage bonus. Nonpositive results cause no HP loss; this does
not mean the intermediate damage function itself always returns a clamped value.
Monster armor is a monster statistic; character armor is a stored field whose
equipment synchronization remains unresolved.

For monsters, nonmagical damage is halved against undead after that calculation;
immortality reduces it to zero. A surviving damage value of 99 is also treated as
lethal, not merely 99 ordinary points. The physical special-weapon classification
begins at ID 23 hex and includes Silver Sword. There is no separate verified
Silver Sword undead instant-kill or multiplier branch in these paths.

### EQUIP-0014

**Effective player armor uncertainty**

**Kind:** rule
**Uses:** EQUIP-0001, SAVE-0002
**Original:** Incomplete
**Redux:** Unverified

A calculation attempts to sum equipped defense ratings and add 3 for Protection,
with a Doom/Mystic Armour exception. The damage path instead reads the stored
character armor byte. Equipment-change synchronization and empty-slot handling
are not established in a matching retail executable.

Do not assert either that equipment always recalculates effective retail armor or
that changing armor never matters. Redux derives defense from its equipped state;
that implementation choice does not prove either original-runtime claim. Any
comparison must retain this evidence gap rather than manufacture certainty.

### EQUIP-0015

**Ammunition and ordinary thrown items**

**Kind:** rule
**Uses:** EQUIP-0010, EQUIP-0012
**Original:** Established
**Redux:** Unverified

Bow/Magic Bow consume one Arrow and Crossbow one Quarrel when a valid combatant
occupies the initially aimed square, before hit testing and projectile travel.
An initially empty square does not consume ammunition through this path; a miss
or blockage after target selection does not refund it.

Using the last ammunition unreadies the exact firing weapon type across the
active party. Bow does not automatically unready Magic Bow or vice versa. This
return-to-stock path does not apply the usual 99-cap helper.

Dagger, Spear, and Throwing Axe use the same initial-target gate. At distance one
they consume no copy; beyond one they consume a carried spare, otherwise remove
the readied copy. This is range-based commitment, not random recovery.

### EQUIP-0016

**Oil, returning weapons, and interference**

**Kind:** rule
**Uses:** EQUIP-0012, EQUIP-0015
**Original:** Established
**Redux:** Unverified

Magic Axe is not consumed by the thrown-item path and draws a returning missile.
Flaming Oil commits a spare or the readied flask after non-aborted aiming but
before checking target occupancy; a successful landing attempts a fire field.
Sling, Morning Star, and Halberd have no ammunition/copy consumption in the
reviewed equipment-consumption path.

Bow, Magic Bow, Crossbow, Sling, and Oil may be stopped by adjacent-enemy
interference before aiming/consumption. Eligibility includes the remembered
opponent, enemy allegiance, active non-sleeping/non-underground state, no Time
Stop, and distance one. Full projectile obstruction is a separate rule.

### EQUIP-0017

**Chaos, Glass, Jeweled, and Mystic weapons**

**Kind:** rule
**Uses:** EQUIP-0012, EQUIP-0013
**Original:** Incomplete
**Redux:** Unverified

Chaos Sword's direct evaluation has automatic-hit/special-damage behavior, but
normal wielding can charm its owner at player-turn entry and hand control to
hostile movement. That charmed attack route substitutes an adjacent 2H Sword
attack. Later recovery can remove the sword without refund and put one affected
PC to sleep. Do not describe ordinary wielding as an unconditional 99-damage attack.

Glass Sword's damage path requests destruction without refund; this does not occur
merely on selecting Ready or Attack. Correct removal for every possible combat-slot
versus roster-index mapping remains unverified. Jeweled Sword's applied damage
rating is zero despite its reference table. Mystic Sword uses normal table/two-hand
rules; Mystic Armour's effective special protection remains subject to armor uncertainty.

### EQUIP-0018

**Ring disappearance, invisibility, and regeneration**

**Kind:** rule
**Uses:** EQUIP-0010, EQUIP-0012
**Original:** Established
**Redux:** Unverified

After successful Ready consumes an Invisibility or Regeneration Ring, a requested
0-15 draw equal to zero removes it without refund. Battle or camping initialization
has a separate non-dead-wearer loss check for result 11. These are nominal 1-in-16
conditions only under an appropriate uniform inclusive generator assumption.

Invisibility can change appearance immediately when readied in combat. Its
separate condition update excludes sleeping/object/dead combat entries; appearance
and condition processing must not be conflated.

A regeneration check scans all active party members. Each non-dead wearer obtaining
7 from a requested 0-7 draw gains one HP, capped at maximum; Sleep and Poison do
not exclude the wearer. Checks occur at player updates and relevant rest/combat
initialization/completion boundaries, not a guaranteed two HP per tactical turn.

### EQUIP-0019

**Turning Amulet and other accessories**

**Kind:** rule
**Uses:** EQUIP-0001, EQUIP-0012, EQUIP-0014
**Original:** Established
**Redux:** Unverified

When an eligible magic-using monster targets an amulet wearer, a requested 0-255
draw below 128 forces its missile path to miss. Melee ignores that flag. This is
not reflection or universal magic immunity: a diverted projectile may still hit
another occupied square. Exact probabilities inherit the RNG qualification.

Protection Ring and Spiked Collar have reference defense 2; Ankh has zero. No
extra Protection Ring +5 branch was established. Their effective armor contribution
is subject to the unresolved stored-field/calculation relationship.

### EQUIP-0020

**Arms trading formulas and stock mutations**

**Kind:** rule
**Uses:** EQUIP-0001, EQUIP-0010
**Original:** Established
**Redux:** Unverified

For normal Intelligence values and base price B:

```text
buy_price = B + trunc((100 - 3 * Intelligence) * B / 100)
sell_offer = trunc(B * 3 * Intelligence / 100) + 1
```

There is no rounding to multiples of ten. Carried quantity exactly 99 prevents
buying. An ammunition purchase sets that stock to 99; another purchase adds one
with a cap. Used ammunition and zero-base-price items are not accepted for sale.
An accepted sale credits capped gold and removes one carried copy. Stock lists,
confirmation flow, and economic side effects remain merchant-service concerns.
