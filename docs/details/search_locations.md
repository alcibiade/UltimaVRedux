# Ultima V Search Locations

This document describes the 113 fixed Search records, their
class-aware Redux decoding, and Search persistence. The Debug Workbench Search
Location Atlas (`codex.search-location.catalog`) is the exhaustive per-record
view; it is generated from the same immutable source-order arrays used by
gameplay and avoids maintaining a second hand-transcribed table here. The
canonical behavioral reference is [CMD-0007](../gamefacts/COMMANDS_REFERENCE.md#cmd-0007).

## Source Records

Each record combines six fields stored in parallel arrays:

| Field | Meaning |
|---|---|
| Object class | World-object class; this is not a Redux inventory item ID |
| Payload | Class-dependent item identity, quantity, or world state |
| Map | Exact raw map/location ID |
| Floor | Exact signed floor; byte `0xff` means floor `-1`, not a wildcard |
| X | X coordinate |
| Y | Y coordinate |

`internal/references/search_locations.go` preserves all 113 records in exact
source order. `DecodeSearchLocationItem` interprets the payload only after
examining the object class.

## Object-Class Decoding

| Class byte | Class | Payload meaning | Redux result |
|---:|---|---|---|
| `0x02` | Gold | Pickup quantity | Gold provision |
| `0x03` | Potion | Potion subtype `0..7` | One potion of that subtype |
| `0x04` | Scroll | Scroll subtype `0..7`; `0xff` is HMS Cape Plans | One scroll or the Plans quest item |
| `0x05` | Weapon | Equipment ID | One exact weapon |
| `0x06` | Shield | Equipment ID | One exact shield |
| `0x07` | Key | Quantity; high bit selects Skull Keys | Keys or low-seven-bit Skull Key quantity |
| `0x08` | Gem | Pickup quantity | Gems |
| `0x09` | Helm | Equipment ID | One exact helm |
| `0x0a` | Ring | Equipment ID | One exact ring |
| `0x0b` | Armour | Equipment ID | One exact armour item |
| `0x0c` | Amulet | Equipment ID | One exact amulet |
| `0x0d` | Torch | Pickup quantity | Torches |
| `0x0f` | Food | Pickup quantity | Food |
| `0x19` | Moonstone | World-object state | Moonstone world object, not inventory quantity |
| `0x1e` | Dead PC | World-object state | Persistent non-carriable rotting body |
| `0x1f` | Dead monster | World-object state | Persistent searchable moldy corpse |

Equipment payloads are identities, never quantities. Potion and scroll payloads
are subtypes and produce one item. This distinction is essential: for example,
records 0-11 alternate Mystic Armour (class `0x0b`, payload `0x0f`) and Mystic Sword
(class `0x05`, payload `0x29`) at `(233,233)`, raw map 0, floor -1. They are not
armour quantities or 41-torch caches.

## Special Records

| Index | Raw source record | Behavior |
|---:|---|---|
| 12 | Scroll class, payload `0xff`; map `0x15`, floor 0, `(15,2)` | HMS Cape Plans |
| 13 | Key class, payload `0x09`; map `0x12`, floor -1, `(8,6)` | Available only with no ordinary Keys and no object already at the coordinate |
| 14 | Key class, payload `0x85`; map `0x05`, floor 0, `(2,2)` | Five Skull Keys, available when the Britannian day differs from the saved last-found day |
| 15 | Weapon class, payload `0x27`; map 0, floor 0, `(64,80)` | One Glass Sword, available only with no Glass Swords and no object already at the coordinate |

Records 13 and 15 do not set ordinary one-time found bits. Their inventory and
coordinate conditions control later availability. Record 14 stores the day on
successful placement and can regenerate on a different day.

## Search Command Flow

Surface Search asks for direction and then uses the selected active character
when that character is Good or Poisoned. With no active selection, or when the
selected character becomes ineligible and is cleared, the existing workflow
chooses among eligible active party members. The resolved actor owns chest trap
detection and moldy-corpse plague effects. Programmatic calls use the same active
character preference, otherwise the first eligible active party member.

For a valid adjacent target, handlers run in this order:

1. Existing surface chest trap inspection.
2. Existing searchable Dead Monster corpse.
3. Secret door.
4. Buried moonstone.
5. Midnight reagent cache.
6. First eligible fixed Search record in source order.
7. Searchable-terrain or existing-item fallback.

The original terrain-specific prelude is folded into the successful response.
Gameplay constructs the complete message once, sends popup feedback, appends the
same text to the message log, and advances the returned Search time exactly once.
No stale response may commit inventory, found state, or another time cost.

Ordinary discoveries create their exact decoded world item first and set the
found bit only after placement succeeds. Items are then collected through the
normal Get path. If multiple source records share a coordinate, repeated Search
discovers them in source order and the item stack preserves the resulting Get
order.

## Corpse Semantics

Dead PC and Dead Monster records create passable, non-carriable current-map
objects rather than inventory entries.

- Dead PC remains a rotting body and cannot be collected with Get.
- Searching a Dead Monster removes the corpse and runs the original corpse-loot
  roll. It may produce Plague or transform into a normal carriable item stack.
- Corpse mutation is authoritative before popup or sprite presentation.
- Leaving and returning to a map reconstructs active corpses in discovery order
  from Search persistence.

Redux deliberately retains a failed Search placement for a later attempt rather
than reproducing original eviction of another dynamic object. The source
record is not marked found until its representation exists.

## Persistence

Native snapshots store one 24-byte `Progression.SearchFound` value. The first
113 bits preserve the source record's one-time found flags:

```text
byte = index / 8
mask = 1 << (index & 7)
found = SearchFound[byte] & mask != 0
```

Bits beginning at 113 encode active surface-corpse discovery order as packed
four-bit values. Each nonzero value is one plus the corpse record's ordinal in
source order; zero terminates the list. There are 19 available slots in the
remaining 79 bits, sufficient for the ten source corpse records. There is no
separate native corpse snapshot field. Hydration restores the 24 bytes and then
reconstructs only corpses belonging to the current map and floor.

The same snapshot progression section separately stores `SkullKeysFoundDay` and
the three daily reagent-cache markers. Legacy `SAVED.GAM` imports the original
24-byte Search-found region at offset `0x02b6` ([SAVE-0003](../gamefacts/SAVED_GAM_STRUCTURE.md#save-0003)); Redux-owned tail encoding is a
native snapshot concern and must not be inferred as original DOS save semantics.

## Verification

Focused coverage includes:

- `internal/references/search_locations_decode_test.go`
- `internal/game_state/search_source_fidelity_test.go`
- `internal/game_state/surface_corpse_persistence_test.go`
- `internal/game_state/search_save_data_integration_test.go`

The Workbench `codex.search-location.catalog` query exposes raw class/payload,
decoded item identity, pickup quantity semantics, exact map/floor/coordinate,
and repeat requirements for every record. It is immutable reference data and
does not expose or mutate live found state.
