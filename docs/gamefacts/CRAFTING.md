# Crafting and deeds

The original game has no gathering, crafting, building or land ownership. Every rule in
this topic describes an optional Redux addition, off by default and enabled with the
Configure option **Crafting and Deeds** (`Gameplay.Crafting.Enabled`). While the option
is off, the crafting chords answer "What?", a deed's hut is plain scenery that Enter
answers with "What?", and every original rule applies. Deeds, huts, materials and tools
already owned are kept, so switching the option back on restores them. Tunable numbers
live in one table, `internal/references/crafting_data.go`; this topic quotes the
defaults. The design and delivery order are recorded in ADR-014; this topic covers what
is implemented now.

### CRAFT-0001

**Optional crafting chords and stock**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Intentional deviation

The original binds no Alt chords and keeps no material, tool or ownership state. Redux
adds three chords: Alt+G Gather, Alt+C Craft and Alt+B Build. One guard runs ahead of
every map's command routing, ahead of intoxicated movement too, and owns any G, C or B
accepted with Alt held, so a chord never becomes Get, Cast or Board:

```
on accepted key G, C or B with Alt held:
    if Ctrl, Shift or Meta is also held, or the option is off: print "What?"
    else if the command does not apply here:                   print "Not here!"
    else run the command
    a refusal opens nothing and spends no time
```

A command applies only with the party on foot. Gather applies on the party's own deed
map ([CRAFT-0004](#craft-0004)); Craft and Build apply on the overworld surface (floor 0)
and on deed maps. Towns, dungeons, the Underworld, combat and vehicles answer
"Not here!". In this version Gather answers "Nothing to gather yet!", Craft
"Nothing to craft yet!", and Build on a deed map "Nothing to build yet!", each without
spending time; Build on the overworld stakes deeds ([CRAFT-0002](#craft-0002)).

The party carries seven materials (Wood, Stone, Ore, Forage, Planks, Cut stone,
Ingots; large stacks capped at 9999) and two tools (Axe, Pick; small stacks capped at
99). Ztats lists carried ones under Material and Tool. Native save version 26 stores
the stock and every deed in its `crafting` member; saves from versions 24 and 25 load
with none.

Rationale: an opt-in sandbox for players who want to own and shape land, kept apart from
every original rule it does not touch.

**Implementation:** `internal/references/crafting_data.go`, `internal/game_state/crafting.go`, `cmd/ultimav/gamescene_crafting.go`
**Tests:** `cmd/ultimav/gamescene_crafting_test.go`, `internal/game_state/crafting_test.go`, `internal/save_snapshot/validation_v26_test.go`

### CRAFT-0002

**Staking a deed**

**Kind:** rule
**Uses:** CRAFT-0001
**Original:** Established
**Redux:** Intentional deviation

Alt+B on the overworld surface examines the square the party stands on. On one of the
party's own huts it prints "<name> is yours. Enter to go in." Otherwise the square must
pass every check, in this order; the first failure is printed and nothing is spent:

| Check | Refusal |
| --- | --- |
| The square is not already a deed | "This land is already yours!" |
| No orthogonal neighbour (wrapping at the world's edges) is a deed; diagonal neighbours are allowed | "Too close to your other land!" |
| The square is not a world location (towne, keep, dungeon, hut and so on) | "Someone else lives here!" |
| The square is neither one of the eight charted moongate sites nor a square where a buried moonstone can open a gate | "The moongates belong to no one!" |
| The terrain is open ground (table below); water, swamp, mountains, roads, bridges, every settlement sprite and every structure are not | "No one can build here!" |
| The party carries at least 200 gold | "A deed costs 200 gold!" |

| Overworld terrain | Deed terrain |
| --- | --- |
| Grass, brush (Forest1), plowed field, crops, grassy shore | Grassland |
| Thick brush, trees, tropical forest (Forest2-4) | Forest (woodland) |
| Foothills | Hills |
| Desert, including dunes | Desert |

A suitable square opens a Build menu offering "Stake a deed here (200 gold)". Esc closes
it for free. Choosing it asks "Name your land:"; Esc, or a name that is not 1 to 24
printable characters once surrounding spaces are trimmed, stakes nothing and is free.
A good name pays 200 gold, records the deed, prints "<name> is yours!" and:

- turns the square into a hut (sprite 16), an ordinary overworld map mutation that the
  native save keeps like any other changed tile;
- pins an Atlas note with the deed's name on the square, without a sextant fix. An
  existing note there is kept; a full Atlas prints "Your Atlas has no room to mark it."

There is no limit on the number of deeds, a deed has no material cost, and deeds are
permanent. A completed stake spends exactly the ordinary 2-minute surface turn
([TIME-0002](TIME_COSTS.md#time-0002)) and nothing more; staking consumes no game RNG.

**Implementation:** `internal/game_state/crafting_deeds.go`, `cmd/ultimav/gamescene_crafting.go`
**Tests:** `internal/game_state/crafting_test.go`, `cmd/ultimav/gamescene_crafting_test.go`

### CRAFT-0003

**Generated deed maps**

**Kind:** rule
**Uses:** CRAFT-0002
**Original:** Established
**Redux:** Intentional deviation

Each deed has a 32x32 map with one floor, the size of an original towne or hut. It is
generated from the deed's terrain, a 32-bit seed and a generator version. The seed is
FNV-1a over the deed's ID and hut square, so it uses no game RNG. Version 1 lays out a
clearing:

- a solid decorative ring three squares deep at every edge, with a ragged fringe on the
  fourth (40% of those squares);
- an open interior of the terrain's ground with scattered accents, and a clear 9x9 yard
  around 15,15;
- an entry lane at column 15 from the south edge (row 31) up to the yard, through the
  entry square 15,30;
- resource nodes in loose clusters, at least five squares from every edge, never on or
  beside the lane, never in the yard, and never touching another node even diagonally,
  so no node can cut off part of the interior.

| Terrain | Ground and ring | Wood | Stone | Ore | Forage | Reagent patches |
| --- | --- | --- | --- | --- | --- | --- |
| Grassland | grass; trees | 7 trees, 2 stumps | 5 rock piles | 1 outcrop | 7 fruit trees | 5: ginseng, garlic |
| Forest | grass with brush; tropical forest | 14 trees, 5 stumps | 3 rock piles | none | 5 fruit trees | 4: spider silk, blood moss, ginseng |
| Hills | grass with foothills; mountains | 4 trees | 9 rock piles | 7 outcrops | 3 fruit trees | 3: sulphurous ash, blood moss |
| Desert | desert; mountains | 4 stumps | 7 rock piles | 4 outcrops | 6 cacti | 2: sulphurous ash |

Trees are the "trees" tile (Forest3), stumps "a hollow stump", rock piles "a pile of
rocks", outcrops the small mountains tile, fruit trees "a fruit tree", and reagent
patches the crops tile; each patch holds one of the listed reagents. Every interior
square is open ground that later steps can build on, a resource node, or the lane.

In this version nodes are scenery: Look describes their original tiles and nothing can
be gathered yet. The save stores each deed's terrain, seed, generator version and only
the squares that differ from the generated map, never the 1024 tiles. A generator
change gets a new version, and existing deeds keep regenerating with theirs.

**Implementation:** `internal/references/deed_map.go`, `internal/references/crafting_data.go`
**Tests:** `internal/references/deed_map_test.go`

### CRAFT-0004

**Entering and leaving deed maps**

**Kind:** rule
**Uses:** CRAFT-0002, CRAFT-0003
**Original:** Established
**Redux:** Intentional deviation

With the option on, Enter on one of the party's huts prints "Enter <name>" and runs the
ordinary location entry: the fade, "Enter your land" and the deed's name, and arrival
at 15,30 on floor 0, at the same time cost as entering a towne. Walking off any edge of
the deed map asks "Dost thou wish to leave?" as on every small map; leaving shows
"Leaving your land" and the name and returns the party to the hut square, at the cost
of leaving a towne.

Every deed map shares one Redux location value, 128 (0x80), clear of the original
locations 0-41 and the virtual ones 249-255; the deed whose map is loaded is game
state, and only one deed map is loaded at a time. Its map frame shows the deed's name,
the save list shows "Your land", the overworld theme plays, the surround beyond the edge
matches the terrain, the sky strip shows as outdoors, and there are no NPCs. With
Flexible sextant use ([ASTRO-0006](ASTRONOMY.md#astro-0006)) the sextant fix on a deed
map is the hut square, as a settlement's fix is its exterior. A deed is not an Atlas
landmark; its own pin ([CRAFT-0002](#craft-0002)) marks it.

A save made on a deed map records the deed; loading it regenerates that deed's map
before the party is placed. Deed maps never join the original location tables, so
deeds from one game cannot appear after loading another.

**Implementation:** `internal/game_state/crafting.go`, `internal/game_state/crafting_save.go`, `internal/game_state/action_enter.go`
**Tests:** `internal/game_state/crafting_test.go`
