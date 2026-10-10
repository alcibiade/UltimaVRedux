# Equipment reference data

These are the 48 DOS equipment identities, expressed as semantic tables rather
than program arrays. A rating is an input to a mechanic, not guaranteed damage,
protection, availability, or a quoted shop price. Effective player armor has an
explicit unresolved boundary in the equipment-mechanics reference.

### EQUIP-0001

**Equipment identity, slots, and reference ratings**

**Kind:** rule
**Uses:** SAVE-0002
**Original:** Established
**Redux:** Unverified

IDs 00-2F below are hexadecimal. Strength is the contribution to cumulative Ready
cost; Attack and Defense are reference ratings. Reach 0 selects adjacent handling,
not a zero-distance attack. A dash means the item is not an independently readied
attack. Base is the merchant formula input, not final price or stock availability.

| ID | Item | Slot | Strength | Attack | Defense | Reach | Base |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: |
| 00 | Leather Helm | Head | 0 | 0 | 1 | - | 15 |
| 01 | Chain Coif | Head | 1 | 0 | 2 | - | 50 |
| 02 | Iron Helm | Head | 2 | 0 | 3 | - | 120 |
| 03 | Spiked Helm | Head | 3 | 4 | 3 | 0 | 150 |
| 04 | Small Shield | One hand | 2 | 0 | 2 | - | 40 |
| 05 | Large Shield | One hand | 3 | 0 | 3 | - | 70 |
| 06 | Spiked Shield | One hand | 4 | 6 | 3 | 0 | 120 |
| 07 | Magic Shield | One hand | 0 | 0 | 5 | - | 2000 |
| 08 | Jewel Shield | One hand | 0 | 0 | 0 | - | 0 |
| 09 | Cloth Armour | Body | 0 | 0 | 1 | - | 20 |
| 0A | Leather Armour | Body | 2 | 0 | 2 | - | 50 |
| 0B | Ring Mail | Body | 4 | 0 | 3 | - | 100 |
| 0C | Scale Mail | Body | 6 | 0 | 4 | - | 150 |
| 0D | Chain Mail | Body | 10 | 0 | 5 | - | 300 |
| 0E | Plate Mail | Body | 12 | 0 | 7 | - | 700 |
| 0F | Mystic Armour | Body | 0 | 0 | 10 | - | 0 |
| 10 | Dagger | One hand | 1 | 6 | 0 | 3 | 1 |
| 11 | Sling | Both hands | 2 | 6 | 0 | 4 | 10 |
| 12 | Club | One hand | 3 | 8 | 0 | 0 | 5 |
| 13 | Flaming Oil | Both hands | 2 | 8 | 0 | 4 | 5 |
| 14 | Main Gauche | One hand | 3 | 8 | 1 | 0 | 15 |
| 15 | Spear | One hand | 4 | 10 | 0 | 5 | 7 |
| 16 | Throwing Axe | One hand | 6 | 10 | 0 | 4 | 3 |
| 17 | Short Sword | One hand | 5 | 12 | 0 | 0 | 40 |
| 18 | Mace | One hand | 7 | 15 | 0 | 0 | 50 |
| 19 | Morning Star | One hand | 8 | 15 | 0 | 2 | 60 |
| 1A | Bow | Both hands | 8 | 10 | 0 | 7 | 75 |
| 1B | Arrows | Cannot ready | 0 | 1 | 0 | - | 10 |
| 1C | Crossbow | Both hands | 6 | 12 | 0 | 8 | 150 |
| 1D | Quarrels | Cannot ready | 0 | 1 | 0 | - | 15 |
| 1E | Long Sword | One hand | 9 | 15 | 0 | 0 | 70 |
| 1F | 2H Hammer | Both hands | 16 | 20 | 0 | 0 | 85 |
| 20 | 2H Axe | Both hands | 15 | 20 | 0 | 0 | 150 |
| 21 | 2H Sword | Both hands | 13 | 20 | 0 | 0 | 200 |
| 22 | Halberd | Both hands | 18 | 30 | 0 | 2 | 250 |
| 23 | Sword of Chaos | Both hands | 0 | 99 | 0 | 0 | 0 |
| 24 | Magic Bow | Both hands | 0 | 15 | 0 | 15 | 800 |
| 25 | Silver Sword | One hand | 8 | 12 | 0 | 0 | 250 |
| 26 | Magic Axe | One hand | 0 | 20 | 0 | 15 | 1000 |
| 27 | Glass Sword | One hand | 5 | 99 | 0 | 0 | 0 |
| 28 | Jeweled Sword | One hand | 0 | 1 | 0 | 0 | 0 |
| 29 | Mystic Sword | Both hands | 0 | 30 | 1 | 0 | 0 |
| 2A | Ring of Invisibility | Ring | 0 | 0 | 0 | - | 450 |
| 2B | Ring of Protection | Ring | 0 | 0 | 2 | - | 500 |
| 2C | Ring of Regeneration | Ring | 0 | 0 | 0 | - | 200 |
| 2D | Amulet/Turning | Neck | 0 | 0 | 0 | - | 900 |
| 2E | Spiked Collar | Neck | 0 | 0 | 2 | - | 240 |
| 2F | Ankh | Neck | 0 | 0 | 0 | - | 0 |

Special behavior overrides some table values: Jeweled Sword's applied rating is
zero; value 99 has special death handling; Chaos Sword commonly changes control
flow before a normal weapon attack. The table alone does not settle those effects.
The complete row-by-row Redux transcription is not re-certified by this conversion.

### EQUIP-0002

**Usage and attack-category encodings**

**Kind:** rule
**Uses:** EQUIP-0001
**Original:** Established
**Redux:** Unverified

Compatibility usage values are head `80`, body `40`, one hand `20`, two hands
`30`, neck `04`, ring `02` (hexadecimal). A feet value `08` exists but has no
equipped item in this 48-entry table. Ammunition has usage zero. Empty character
slots are `FF`, not item zero. Neither usage nor Strength data establishes a
profession/class restriction.

Attack category 8 selects Strength for Spiked Helm, Spiked Shield, Club, Mace,
and 2H Hammer. Ordinary other offensive items use Dexterity. Categories 0-7
identify arrow, cannonball, axe, red, blue, green, violet, and rock presentation.
Range zero, rather than that category, selects adjacent handling. Several combat
tables continue with seven spell entries after equipment ID 2F; those entries
are not carried equipment.

### EQUIP-0003

**Display codes and price interpretation**

**Kind:** rule
**Uses:** EQUIP-0001
**Original:** Established
**Redux:** Unverified

Equipment display character codes are glyph indexes, not world tiles. The exact
artwork is external presentation data. Base price zero does not establish a quest
classification, acquisition route, free purchase, or merchant availability. Merchant
stock lists and pricing formulas are separate from the identity table.

The reference table is canonical for these values. Mechanics documents should
link to it rather than transcribe another set of arrays or rounded prices.
