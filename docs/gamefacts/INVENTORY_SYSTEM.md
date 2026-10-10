# Inventory ownership and resource commitment

The original party inventory consists of several categories and per-character
equipment, not one universal bag. Exact saved offsets are canonical in the
[save-format tables](SAVED_GAM_STRUCTURE.md#save-0003). Numeric limits below belong
to specific normal mutation paths rather than all possible loaded byte values.

### INV-0001

**Ownership and indexed categories**

**Kind:** rule
**Uses:** SAVE-0002, SAVE-0003, EQUIP-0001, EQUIP-0010
**Original:** Established
**Redux:** Unverified

The party shares 48 carried-equipment quantities, 48 prepared-spell quantities,
eight scroll quantities, eight potion quantities, and eight reagent quantities.
Character slots separately hold equipped items. Quest objects/tools use their own
fields; discovery bits are not interchangeable with possession.

Prepared spells have six entries per circle:

```text
slot = (circle - 1) * 6 + within_circle_index
circle = floor(slot / 6) + 1
```

There is no separate original saved combat-spell stock, universal rune field,
moonstone possession mask, or four-item quest array implied by these categories.

### INV-0002

**Scroll, potion, and reagent order**

**Kind:** rule
**Uses:** SAVE-0003
**Original:** Established
**Redux:** Unverified

| Index | Scroll | Potion color / ordinary effect | Reagent |
| ---: | --- | --- | --- |
| 0 | Vas Lor / Light | Blue / awaken | Sulfur Ash |
| 1 | Rel Hur / Wind Change | Yellow / heal | Ginseng |
| 2 | In Sanct / Protection | Red / cure poison | Garlic |
| 3 | In An / Negate Magic | Green / poison | Spider Silk |
| 4 | In Quas Wis / View | Orange / sleep | Blood Moss |
| 5 | Kal Xen Corp / Summon Daemon | Purple / combat Rat form | Black Pearl |
| 6 | In Mani Corp / Resurrection | Black / combat invisibility | Nightshade |
| 7 | An Tym / Negate Time | White / surface X-ray view | Mandrake Root |

These category indexes do not guarantee effect eligibility or reliability. Potion
substitution and resource commitment are described below; effect-specific details
belong in their owning consumable/magic rules.

### INV-0003

**Runtime item-menu projection**

**Kind:** rule
**Uses:** INV-0001, INV-0002
**Original:** Established
**Redux:** Unverified

The usable-item list projects saved fields into 38 entries: scrolls 0-7, potions
8-15, carpets 16, Skull Keys 17, Amulet/Crown/Sceptre 18-20, moonstones 21-28,
three named shards 29-31, Spyglass 32, plans 33, Sextant 34, Watch 35, Badge 36,
and box 37. A special quantity value suppresses numeric display for unique items.
This list is not another independently saved inventory, and a display marker does
not establish every item's consumption or lifetime rules.

### INV-0004

**Mutation-specific capacities**

**Kind:** rule
**Uses:** INV-0001, EQUIP-0015, EQUIP-0020
**Original:** Established
**Redux:** Unverified

| Path | Normal behavior |
| --- | --- |
| Ordinary equipment purchase | Add one, cap 99 |
| Ammunition purchase | Set selected quantity to 99 |
| Explicit unready through Ready | Return one while below 99 |
| Last-ammunition weapon return | No ordinary cap helper at that return site |
| Equipment pickup | Ordinary copies capped at 99; ammunition adds five with cap |
| Scroll/potion pickup | Normal increment capped at 99 |
| Prepared-spell mixing | Add requested quantity, cap 99 |
| Reviewed reagent purchase/collection | Cap 99 |
| Reviewed food/gold additions | Cap 9999 |
| Reviewed key, Skull Key, gem, torch pickups | Cap 99 |

No universal writer-level repair of corrupt or excessive inventory is established.
Removing worn equipment through destruction can omit a refund even though the
Ready UI normally returns it.

### INV-0005

**Mixing preparation and loss**

**Kind:** rule
**Uses:** INV-0001, INV-0004
**Original:** Established
**Redux:** Unverified

Toggling an ingredient changes a selection mask, not stock. After nonzero
selection/quantity and availability checks, subtract the requested amount from
every selected reagent. An exact recipe match adds that many preparations, capped
at 99. A wrong mixture still consumes the reagents and invokes its failure effect.
Overflow production is not refunded. This path has no caster or caster-level
selection; cancellation before commitment consumes nothing.

### INV-0006

**Prepared casting resource order and build qualification**

**Kind:** rule
**Uses:** INV-0001
**Original:** Incomplete
**Redux:** Unverified

The ordinary resource-enforcing path selects caster/spell, checks context and
location absorption, and requires a prepared copy. It consumes the copy before
checking MP; sufficient MP is deducted before checking level. Spell-specific
targeting and effects occur later. Thus insufficient MP can lose a preparation,
and insufficient level can lose both preparation and MP.

An alternative reduced-cost/debug branch also exists in the reviewed material.
Its relationship to a particular shipped executable is not established. Do not
present the normal-path order as independently certified for every build, or
generalize downstream cancellation/refund rules without examining the effect.

### INV-0007

**Scroll and potion commitment**

**Kind:** rule
**Uses:** INV-0002
**Original:** Established
**Redux:** Unverified

A selected scroll is consumed before its direction, context, target, and effect
handling. A selected potion is consumed before noncombat recipient selection;
combat uses the current PC. Cancellation after selection does not refund it.

After a recipient exists, a potion requests a 0-15 draw: result 0 substitutes
Orange; result 1 requests a second 0-7 effect selection; otherwise the labeled
effect remains. Cancellation before this stage consumes the potion but not the
substitution draw. Generator distribution and each effect's detailed eligibility
are separate questions.

### INV-0008

**Moonstones, discovery, and unique state**

**Kind:** rule
**Uses:** SAVE-0003
**Original:** Established
**Redux:** Unverified

Each moonstone stores X, Y, map, and level. Map value 255 means carried. Search
exposes a loose object without setting possession; pickup marks it carried.
Successful burial writes current coordinates/map/level. The map byte is a map or
possession sentinel, not an X-state bitmask.

The 24-byte hidden-object array has 192 bits. For ordinary one-time entries:

```text
byte_index = floor(object_index / 8)
mask = 1 << (object_index modulo 8)
mark discovery when successfully found
```

Renewable castle Keys, daily Skull Keys, and Glass Swords have separate conditions;
discovery and possession remain distinct. Three shards are named, but four saved
bytes are allocated and generic pickup can address the fourth index. Its gameplay
meaning remains unknown. Plans pickup can already set all bits, so its high bit
does not uniquely prove a later rigging action.
