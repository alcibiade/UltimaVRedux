# Ultima V Search Locations

This document describes the 113 fixed Search records, their
class-aware Redux decoding, and Search persistence. The
[record table](#record-table) lists every record; the immutable source-order
arrays in `internal/references/search_locations.go` remain authoritative. The
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
| `0x04` | Scroll | Scroll subtype `0..7`; `0xff` is H.M.S. Cape Plans | One scroll or the Plans quest item |
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
| 12 | Scroll class, payload `0xff`; map `0x15`, floor 0, `(15,2)` | H.M.S. Cape Plans |
| 13 | Key class, payload `0x09`; map `0x12`, floor -1, `(8,6)` | Available only with no ordinary Keys and no object already at the coordinate |
| 14 | Key class, payload `0x85`; map `0x05`, floor 0, `(2,2)` | Five Skull Keys, available when the Britannian day differs from the saved last-found day |
| 15 | Weapon class, payload `0x27`; map 0, floor 0, `(64,80)` | One Glass Sword, available only with no Glass Swords and no object already at the coordinate |

Records 13 and 15 do not set ordinary one-time found bits. Their inventory and
coordinate conditions control later availability. Record 14 stores the day on
successful placement and can regenerate on a different day.

## Record Table

Generated from `GetAllSearchLocations` and `DecodeSearchLocationItem`. Location
names are the `references.Location` identifiers with the raw map number in
parentheses; floor `-1` is the Underworld on map 0 and a basement elsewhere.
Class/payload are the raw source bytes; equipment, potion and scroll results are
one item, while provisions show their pickup quantity.

| # | Location (map) | Floor | X,Y | Class/payload | Decoded result |
|---:|---|---:|---|---|---|
| 0 | BritanniaUnderworld (0) | -1 | 233,233 | `0b/0f` | Mystic Armour |
| 1 | BritanniaUnderworld (0) | -1 | 233,233 | `05/29` | Mystic Sword |
| 2 | BritanniaUnderworld (0) | -1 | 233,233 | `0b/0f` | Mystic Armour |
| 3 | BritanniaUnderworld (0) | -1 | 233,233 | `05/29` | Mystic Sword |
| 4 | BritanniaUnderworld (0) | -1 | 233,233 | `0b/0f` | Mystic Armour |
| 5 | BritanniaUnderworld (0) | -1 | 233,233 | `05/29` | Mystic Sword |
| 6 | BritanniaUnderworld (0) | -1 | 233,233 | `0b/0f` | Mystic Armour |
| 7 | BritanniaUnderworld (0) | -1 | 233,233 | `05/29` | Mystic Sword |
| 8 | BritanniaUnderworld (0) | -1 | 233,233 | `0b/0f` | Mystic Armour |
| 9 | BritanniaUnderworld (0) | -1 | 233,233 | `05/29` | Mystic Sword |
| 10 | BritanniaUnderworld (0) | -1 | 233,233 | `0b/0f` | Mystic Armour |
| 11 | BritanniaUnderworld (0) | -1 | 233,233 | `05/29` | Mystic Sword |
| 12 | EastBritanny (21) | 0 | 15,2 | `04/ff` | H.M.S. Cape Plans |
| 13 | PalaceOfBlackthorn (18) | -1 | 8,6 | `07/09` | Keys ×9 |
| 14 | Minoc (5) | 0 | 2,2 | `07/85` | Skull Keys ×5 |
| 15 | BritanniaUnderworld (0) | 0 | 64,80 | `05/27` | Glass Sword |
| 16 | PalaceOfBlackthorn (18) | 1 | 7,6 | `05/23` | Sword of Chaos |
| 17 | PalaceOfBlackthorn (18) | 1 | 23,6 | `05/28` | Jeweled Sword |
| 18 | Moonglow (1) | 0 | 8,5 | `08/01` | Gems ×1 |
| 19 | Moonglow (1) | 0 | 25,6 | `0b/0a` | Leather Armour |
| 20 | Moonglow (1) | 0 | 25,8 | `0b/0a` | Leather Armour |
| 21 | Moonglow (1) | 0 | 23,5 | `03/05` | Purple Potion |
| 22 | Moonglow (1) | 0 | 13,13 | `03/06` | Black Potion |
| 23 | Moonglow (1) | 0 | 14,13 | `03/07` | White Potion |
| 24 | Moonglow (1) | 0 | 16,13 | `04/05` | Kal Xen Corp Scroll |
| 25 | Moonglow (1) | 0 | 17,13 | `04/07` | An Tym Scroll |
| 26 | Moonglow (1) | 1 | 24,19 | `0f/0a` | Food ×10 |
| 27 | Moonglow (1) | 0 | 27,3 | `0d/03` | Torches ×3 |
| 28 | Moonglow (1) | 0 | 27,29 | `0a/2a` | Ring of Invisibility |
| 29 | Britain (2) | 0 | 2,1 | `0f/05` | Food ×5 |
| 30 | Britain (2) | 0 | 6,26 | `05/14` | Main Gauche |
| 31 | Britain (2) | 1 | 24,6 | `0a/2b` | Ring of Protection |
| 32 | Jhelom (3) | 0 | 21,16 | `04/01` | Rel Hur Scroll |
| 33 | Jhelom (3) | 0 | 20,10 | `0f/05` | Food ×5 |
| 34 | Jhelom (3) | 0 | 29,1 | `0f/0a` | Food ×10 |
| 35 | Jhelom (3) | 0 | 30,23 | `05/26` | Magic Axe |
| 36 | Jhelom (3) | 0 | 1,29 | `0d/04` | Torches ×4 |
| 37 | Yew (4) | 0 | 29,11 | `05/12` | Club |
| 38 | Yew (4) | 0 | 22,26 | `03/03` | Green Potion |
| 39 | Yew (4) | 0 | 1,26 | `04/04` | In Quas Wis Scroll |
| 40 | Yew (4) | 0 | 13,2 | `1f/00` | Moldy corpse |
| 41 | Yew (4) | 0 | 14,2 | `1f/00` | Moldy corpse |
| 42 | Yew (4) | 0 | 14,4 | `1e/00` | Rotting body |
| 43 | Yew (4) | 0 | 16,3 | `1e/00` | Rotting body |
| 44 | Yew (4) | 0 | 18,2 | `1e/00` | Rotting body |
| 45 | Yew (4) | 0 | 16,3 | `05/15` | Spear |
| 46 | Yew (4) | -1 | 27,22 | `05/25` | Silver Sword |
| 47 | Yew (4) | -1 | 20,22 | `07/05` | Keys ×5 |
| 48 | Minoc (5) | 0 | 27,8 | `02/63` | Gold ×99 |
| 49 | Minoc (5) | 1 | 13,11 | `04/01` | Rel Hur Scroll |
| 50 | Minoc (5) | 1 | 12,11 | `04/01` | Rel Hur Scroll |
| 51 | Minoc (5) | 1 | 8,21 | `03/01` | Yellow Potion |
| 52 | Minoc (5) | 1 | 8,23 | `03/02` | Red Potion |
| 53 | Minoc (5) | 1 | 7,23 | `04/06` | In Mani Corp Scroll |
| 54 | Trinsic (6) | 1 | 24,6 | `08/04` | Gems ×4 |
| 55 | SkaraBrae (7) | 0 | 5,2 | `1e/00` | Rotting body |
| 56 | SkaraBrae (7) | 0 | 6,7 | `03/01` | Yellow Potion |
| 57 | NewMagincia (8) | 1 | 21,18 | `03/03` | Green Potion |
| 58 | NewMagincia (8) | 1 | 25,21 | `07/07` | Keys ×7 |
| 59 | Fogsbane (9) | 0 | 10,12 | `0d/09` | Torches ×9 |
| 60 | Greyhaven (11) | 0 | 21,15 | `03/00` | Blue Potion |
| 61 | Greyhaven (11) | 0 | 14,9 | `08/05` | Gems ×5 |
| 62 | Greyhaven (11) | 0 | 16,12 | `02/32` | Gold ×50 |
| 63 | IolosHut (13) | 0 | 24,2 | `03/07` | White Potion |
| 64 | SuteksHut (14) | 0 | 16,12 | `04/05` | Kal Xen Corp Scroll |
| 65 | SuteksHut (14) | 0 | 14,16 | `0c/2d` | Amulet of Turning |
| 66 | SuteksHut (14) | 0 | 17,12 | `04/07` | An Tym Scroll |
| 67 | SuteksHut (14) | 0 | 14,12 | `03/04` | Orange Potion |
| 68 | LordBritishsCastle (17) | 0 | 20,7 | `0b/0a` | Leather Armour |
| 69 | LordBritishsCastle (17) | 0 | 21,7 | `0b/0b` | Ring Mail |
| 70 | LordBritishsCastle (17) | 0 | 22,7 | `0b/09` | Cloth Armour |
| 71 | LordBritishsCastle (17) | 0 | 23,7 | `0b/0c` | Scale Mail |
| 72 | LordBritishsCastle (17) | 1 | 21,13 | `05/1e` | Long Sword |
| 73 | LordBritishsCastle (17) | -1 | 7,18 | `07/07` | Keys ×7 |
| 74 | LordBritishsCastle (17) | -1 | 20,23 | `0a/2c` | Ring of Regeneration |
| 75 | PalaceOfBlackthorn (18) | 1 | 17,18 | `03/03` | Green Potion |
| 76 | PalaceOfBlackthorn (18) | 2 | 13,6 | `04/05` | Kal Xen Corp Scroll |
| 77 | PalaceOfBlackthorn (18) | 2 | 14,6 | `04/05` | Kal Xen Corp Scroll |
| 78 | PalaceOfBlackthorn (18) | 2 | 16,6 | `04/02` | In Sanct Scroll |
| 79 | PalaceOfBlackthorn (18) | 2 | 17,6 | `04/07` | An Tym Scroll |
| 80 | PalaceOfBlackthorn (18) | 2 | 19,7 | `0a/2b` | Ring of Protection |
| 81 | WestBritanny (19) | 0 | 3,2 | `1e/00` | Rotting body |
| 82 | WestBritanny (19) | 0 | 5,7 | `1e/00` | Rotting body |
| 83 | WestBritanny (19) | 0 | 7,7 | `1e/00` | Rotting body |
| 84 | WestBritanny (19) | 0 | 7,2 | `1e/00` | Rotting body |
| 85 | WestBritanny (19) | 0 | 5,7 | `04/06` | In Mani Corp Scroll |
| 86 | WestBritanny (19) | 0 | 3,2 | `0a/2c` | Ring of Regeneration |
| 87 | NorthBritanny (20) | 0 | 18,25 | `08/03` | Gems ×3 |
| 88 | EastBritanny (21) | 0 | 13,2 | `04/01` | Rel Hur Scroll |
| 89 | EastBritanny (21) | 0 | 14,2 | `04/01` | Rel Hur Scroll |
| 90 | EastBritanny (21) | 0 | 16,2 | `04/01` | Rel Hur Scroll |
| 91 | Paws (22) | 0 | 13,13 | `0a/2a` | Ring of Invisibility |
| 92 | Paws (22) | 0 | 3,12 | `07/05` | Keys ×5 |
| 93 | Cove (23) | 0 | 15,1 | `0c/2f` | Ankh |
| 94 | BuccaneersDen (24) | 0 | 19,22 | `07/01` | Keys ×1 |
| 95 | BuccaneersDen (24) | 0 | 19,22 | `03/03` | Green Potion |
| 96 | Ararat (25) | 0 | 25,16 | `04/01` | Rel Hur Scroll |
| 97 | Bordermarch (26) | 0 | 11,4 | `0c/2e` | Spiked Collar |
| 98 | Farthing (27) | 0 | 11,17 | `03/05` | Purple Potion |
| 99 | Farthing (27) | 0 | 10,17 | `03/04` | Orange Potion |
| 100 | Windemere (28) | 0 | 19,22 | `04/05` | Kal Xen Corp Scroll |
| 101 | Lycaeum (30) | 2 | 23,9 | `0a/2a` | Ring of Invisibility |
| 102 | Lycaeum (30) | 2 | 23,7 | `0c/2e` | Spiked Collar |
| 103 | Lycaeum (30) | 2 | 20,7 | `03/07` | White Potion |
| 104 | Lycaeum (30) | 1 | 22,19 | `05/12` | Club |
| 105 | Lycaeum (30) | 1 | 22,17 | `0c/2e` | Spiked Collar |
| 106 | EmpathAbbey (31) | 0 | 6,3 | `03/01` | Yellow Potion |
| 107 | EmpathAbbey (31) | 0 | 19,7 | `0f/14` | Food ×20 |
| 108 | SerpentsHold (32) | 1 | 8,21 | `03/07` | White Potion |
| 109 | Moonglow (1) | 1 | 19,24 | `0f/10` | Food ×16 |
| 110 | Moonglow (1) | 1 | 20,24 | `0f/0d` | Food ×13 |
| 111 | LordBritishsCastle (17) | 2 | 12,12 | `03/06` | Black Potion |
| 112 | LordBritishsCastle (17) | 2 | 12,12 | `04/07` | An Tym Scroll |

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

Identified item lists and Get receipts use the full catalog labels from
[INV-0003](../gamefacts/INVENTORY_SYSTEM.md#inv-0003), including potion colours and
scroll incantations with their family suffixes. H.M.S. Cape Plans punctuation is
editorial; its class/payload and stable quest ID are unchanged. Authored Search
discovery sentences remain separate and may deliberately identify only a potion,
weapon or strange rock. Naming normalization does not reveal hidden identities
or powers before the existing discovery flow does.

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

`search_locations_decode_test.go` pins the raw source tables and checks that
every record decodes to a catalog item or world object. Regenerate the record
table from the same functions if the source arrays change.
