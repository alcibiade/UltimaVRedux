# Britannia combat records

`BRIT.CBT` supplies fixed outdoor, town, camping, Shadowlord, and ship encounter
layouts. It shares a record envelope with `DUNGEON.CBT`, but not all metadata
semantics. Values below describe the DOS format; exact physical size and artwork
for every distributed file are not established by the record consumers alone.

### CBT-0001

**Record boundaries and addressing**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Matches

Each addressed record occupies 352 bytes (`0x160`): eleven rows with a 32-byte
stride. The battlefield occupies only columns 0-10 of those rows, yielding an
11 x 11 tile plane. All indexes are zero-based:

```text
record_start = record_index * 352
cell_offset = record_start + row * 32 + column
```

Selectors address through index 15, requiring a span of at least 5,632 bytes
(`0x1600`). That requirement is not proof that every physical file has exactly
16 records and no trailing data. Redux bounds record reads against actual file
length and decodes the same tile plane.

**Implementation:** `internal/references/combat_map.go`, `internal/references/combat_map_reference.go`
**Tests:** `internal/references/combat_map_reference_test.go`

### CBT-0002

**Britannia formation metadata**

**Kind:** rule
**Uses:** CBT-0001
**Original:** Established
**Redux:** Matches

| Meaning | Row and columns | Relative byte offsets | Count |
| --- | --- | --- | ---: |
| Terrain | Rows 0-10, columns 0-10 | row*32+column | 121 |
| Party X | Row 3, columns 11-16 | 006B-0070 | 6 |
| Party Y | Row 3, columns 17-22 | 0071-0076 | 6 |
| Encounter X | Row 6, columns 11-26 | 00CB-00DA | 16 |
| Encounter Y | Row 7, columns 11-26 | 00EB-00FA | 16 |

Coordinates are bytes, not tiles. Six party positions are copied even where the
destination arrays have more capacity. Sixteen encounter positions are formation
capacity, not a guaranteed monster count; population is determined separately.
Other Britannia metadata bytes remain uninterpreted here. Zero does not prove
that an unused byte is a disabled dungeon trigger.

**Implementation:** `internal/references/combat_map_reference.go`
**Tests:** `internal/references/combat_map_reference_test.go`

### CBT-0003

**Original layout selection**

**Kind:** rule
**Uses:** CBT-0001
**Original:** Incomplete
**Redux:** Unverified

| Index | Selection |
| ---: | --- |
| 0 | Camping |
| 1 | Swamp |
| 2 | Grass and ordinary outdoor fallback |
| 3 | Brush or thick brush |
| 4 | Desert or either waste half |
| 5 | Forest or tropical vegetation |
| 6 | Hills, mountains, peaks, or split-hill halves |
| 7 | Bridge families |
| 8 | Floor and ordinary small-map fallback |
| 9 | No selector established in the reviewed normal selection path |
| 10 | Shadowlord encounter |
| 11 | Ship versus sea creature |
| 12 | Land party versus pirate ship |
| 13 | Ship versus shore creature |
| 14 | Ship versus pirate ship |
| 15 | Land party versus water creature |

Creature base families disregard their two low frame bits. The initial water
classification accepts terrain below swamp, or river-family terrain that is not
the excluded bridge pair. Sea-creature families force water classification.
Shadowlord selection takes precedence; ship/shore combinations precede ordinary
terrain selection. Index 9 must not acquire an original basement/dungeon meaning
merely because a remake gives it a convenient label. The full original caller
set and every authored record remain outside this verified selection scope.

**Implementation:** `internal/references/combat_map.go`, `internal/references/combat_map_reference.go`
**Tests:** `internal/references/combat_map_reference_test.go`

### CBT-0004

**Tactical bounds and dungeon-only metadata**

**Kind:** rule
**Uses:** CBT-0001, CBT-0002
**Original:** Established
**Redux:** Matches

Combat coordinates are 0-10. An off-screen destination may be accepted by movement
legality so a later escape path can process it; this does not enlarge the file's
tile plane.

For a room loaded from `DUNGEON.CBT`, entry facing selects party rows 1-4;
row 5 holds up to sixteen entity descriptors and rows 6-7 their coordinates.
Eight terrain-replacement triggers use row-0 columns 11-18 and rows 8-10. A
trigger can replace two in-bounds cells and disables itself after activation.
These meanings do not apply to ordinary Britannia records solely because the
buffer sizes match. Generated dungeon encounters synthesize their own terrain
and metadata rather than selecting another Britannia record.

Redux separates dungeon entity/trigger handling from the Britannia plane and
generates the corresponding dungeon encounter records. Exact original artwork,
unused metadata, and all malformed records are not implied by that comparison.

**Implementation:** `internal/references/combat_map_reference.go`, `internal/references/combat_map_dungeon_generated.go`
**Tests:** `internal/references/combat_map_reference_test.go`, `internal/references/combat_map_dungeon_generated_test.go`
