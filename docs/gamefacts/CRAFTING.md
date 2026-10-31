# Crafting and deeds

The original game has no gathering, crafting, building or land ownership. Every rule in
this topic describes an optional Redux addition, off by default and enabled with the
Configure option **Crafting and Deeds** (`Gameplay.Crafting.Enabled`). While the option
is off, the crafting chords answer "What?", a deed's hut is plain scenery that Enter
answers with "What?", and every original rule applies. Deeds, huts, materials and tools
already owned are kept, so switching the option back on restores them. Tunable numbers
live in `internal/references` (`crafting_data.go`, `crafting_recipes.go` and
`crafting_pieces.go`); this topic quotes the defaults. The design and delivery order are recorded in ADR-014; this topic covers what
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
"Not here!". Gather harvests resource nodes ([CRAFT-0005](#craft-0005)), Craft makes
tools, materials, pieces and provisions ([CRAFT-0007](#craft-0007)), mundane arms and
armour ([CRAFT-0013](#craft-0013)), and potions, scrolls and refined reagents
([CRAFT-0015](#craft-0015) to [CRAFT-0017](#craft-0017)), and Build stakes deeds on
the overworld ([CRAFT-0002](#craft-0002)) and places or demolishes pieces on deed maps
([CRAFT-0010](#craft-0010)).

The party carries seven materials (Wood, Stone, Ore, Forage, Planks, Cut stone,
Ingots; large stacks capped at 9999), two tools (Axe, Pick; small stacks capped at
99) and 26 build pieces ([CRAFT-0009](#craft-0009); small stacks capped at 99). Ztats
lists carried ones under Material, Tool and Piece. Native save version 26 stores the
stock and every deed in its `crafting` member; saves from versions 24 and 25 load
with none. With the option off, built pieces stay on deed maps as ordinary tiles,
their storage chests keep their contents, and the original Open and Hole Up rules
apply to them.

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
| The party carries at least 200 gold | "A deed costs 200 gold; thou hast <gold carried>." |

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

A forest deed has no ore outcrop, so it yields no Ore. Its own land can neither build a
Forge (3 Ore) nor smelt Ingots (2 Ore each, [CRAFT-0008](#craft-0008)), so it cannot reach
forged tools, arms, armour or keys on its own: the party needs a hills, grassland or desert
deed as well, or Ore carried in from one.

Nodes are gathered with Alt+G ([CRAFT-0005](#craft-0005)) and restored by Trammel
([CRAFT-0006](#craft-0006)); Look names them by what they are rather than by their
original tile ("an ore vein", not "mountains"; "a patch of Garlic", not "bountiful
crops"). An ore outcrop keeps the small mountains tile of version 1, the same tile as
part of the hills and desert ring; outcrops always stand inside the clearing, and Look
tells them apart. The save stores each deed's terrain, seed, generator version and only
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
map is the hut square, as a settlement's fix is its exterior. The Atlas opens on a deed
map as in a settlement and reads "Within your land". A deed is not an Atlas
landmark; its own pin ([CRAFT-0002](#craft-0002)) marks it.

A save made on a deed map records the deed; loading it regenerates that deed's map
before the party is placed. Deed maps never join the original location tables, so
deeds from one game cannot appear after loading another.

**Implementation:** `internal/game_state/crafting.go`, `internal/game_state/crafting_save.go`, `internal/game_state/action_enter.go`
**Tests:** `internal/game_state/crafting_test.go`

### CRAFT-0005

**Gathering on deed maps**

**Kind:** rule
**Uses:** CRAFT-0001, CRAFT-0003
**Original:** Established
**Redux:** Intentional deviation

Alt+G on the party's own deed map prints "Gather-" and asks for a direction; Esc cancels
for free. Anywhere else it answers "Not here!" ([CRAFT-0001](#craft-0001)). The command
acts on the adjacent square in that direction:

```
first, bring every deed up to date with Trammel (CRAFT-0006)
if the square holds a creature or an item stack, is no generated node,
   or shows anything other than its node or depleted tile:  "Nothing to gather!"
else if the node is used up:                                 "Nothing left to gather!"
else if the stack it fills is full (Wood 9999, Garlic 99):   "Thou canst carry no more <name>!"
else if the node needs a tool the party lacks:
    if bare hands cannot work it (ore):                      "Thou dost need a Pick!"
    else roll one in 3: success gathers 1, failure prints "Thy bare hands win no <material>."
else gather a random yield in the node's range
a gathered yield adds to the stack (saturating at its cap), prints
"Gathered <n> <name>.", and counts one harvest; the harvest that reaches the node's
capacity changes its tile to the depleted tile and prints "It is used up!"
```

| Node (tile) | Yields | Tool | Bare hands | Yield | Capacity | Minutes | Depleted tile | Look (fresh / used up) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Tree (trees) | Wood | Axe | 1 in 3: 1 | 2-4 | 4 | 10 | brush | a stand of timber / a felled tree |
| Stump (a hollow stump) | Wood | Axe | 1 in 3: 1 | 2-4 | 4 | 10 | terrain ground | a dry stump / a grubbed-out stump |
| Rock pile (a pile of rocks) | Stone | Pick | 1 in 3: 1 | 2-3 | 4 | 10 | terrain ground | a pile of rocks / a quarried rock pile |
| Ore outcrop (small mountains) | Ore | Pick | never | 1-2 | 3 | 15 | foothills | an ore vein / a worked-out ore vein |
| Fruit tree (a fruit tree) | Forage | none | - | 1-3 | 3 | 5 | brush | a fruit tree |
| Cactus (a cactus) | Forage | none | - | 1-3 | 3 | 5 | desert ground | a fruiting cactus / a cactus |
| Reagent patch (crops) | its reagent | none | - | 1-2 | 2 | 10 | a plowed patch | a patch of <reagent> |

Reagents go into the original reagent inventory (cap 99) and are the ones listed for each
terrain in [CRAFT-0003](#craft-0003). Tools are not used up. A gathered square shows its
depleted tile as an ordinary deed-map override, and Look on it adds "(depleted)", for
example "Thou dost see a felled tree (depleted)". A node whose square shows something else
(a later building step) is not described as a node and cannot be gathered.

Every attempt, successful or not, costs exactly the node's minutes, the ordinary
one-minute deed-map turn included; a refusal costs nothing and opens nothing. Game RNG:
a harvest with a tool or by hand draws once for its yield; a bare-handed try at a node that
needs a tool draws once for success and nothing for its yield of 1.

Rationale: the first tools need Wood and Stone, so bare hands win a little of both; ore,
the start of metalwork, needs the Pick.

**Implementation:** `internal/references/crafting_data.go`, `internal/game_state/crafting_gather.go`, `internal/game_state/action_look.go`, `cmd/ultimav/gamescene_crafting.go`
**Tests:** `internal/game_state/crafting_gather_test.go`, `cmd/ultimav/gamescene_crafting_gather_test.go`

### CRAFT-0006

**Trammel restores used-up nodes**

**Kind:** rule
**Uses:** CRAFT-0005, MOON-0002
**Original:** Established
**Redux:** Intentional deviation

Trammel's phase is indexed by day ([MOON-0002](MOONSTONE_LOCATIONS.md#moon-0002)) and
changes on 16 of the 28 days of each month. At each change, on every deed, whether its
map is loaded or not:

```
every node that has given harvests but is not used up becomes fresh again
n = the used-up nodes still showing their depleted tile
restore ceil(25% of n) of them, in row-major order (top row first, then left to right):
    the harvest count clears and the generated tile returns
on the loaded deed map, skip a node under the party, a creature or an item stack
    and take the next node in order instead
```

Each deed keeps a marker: the number of Trammel phase changes since the epoch (day 1 of
month 1 of year 1) that its nodes have been brought up to. A new deed starts at the day it
is staked. A deed is brought up to date at every change of day and before every gathering
attempt; when several phase changes have passed (a long absence, a rest, a time skip), the
rule runs once for each change, in order, so 8 used-up nodes become 6, then 4, 3, 2, 1, 0.
A date moved backwards only resets the marker. A deed map suspended under combat or camping
waits until the next change of day or gathering attempt, so no node reappears under a party
that has not yet returned. Respawn consumes no game RNG and spends no time. Native save
version 26 keeps each deed's harvest counts and marker.

Rationale: a stripped deed recovers in about a week of game time, tied to a moon the party
can see, and the catch-up never depends on whether the deed was visited.

**Implementation:** `internal/datetime/ultima_date.go`, `internal/game_state/crafting_respawn.go`, `internal/game_state/turn.go`
**Tests:** `internal/game_state/crafting_gather_test.go`, `internal/datetime/trammel_phase_changes_test.go`, `internal/save_snapshot/validation_v26_test.go`

### CRAFT-0007

**Crafting**

**Kind:** rule
**Uses:** CRAFT-0001, CRAFT-0005
**Original:** Established
**Redux:** Intentional deviation

Alt+C on the overworld surface or a deed map, on foot, prints "Craft-" and opens a menu
of categories: Tools, Refined materials, Stations, Walls and doors, Floors, Tables and
chairs, Furnishings and Provisions, then the arms, armour and alchemy categories of
[CRAFT-0019](#craft-0019). Each category row says how many of its recipes can be made
now ("Tools (1 ready)", or "(none ready)"). Choosing one lists its recipes; each row names
the recipe and its inputs (reagents by short name, [CRAFT-0019](#craft-0019)), and a row
the party cannot make now ends with
"(needs <station>)" when the station is not within reach ([CRAFT-0008](#craft-0008)) or
"(lacking)" when inputs are missing. The description lists the materials carried (for
alchemy, the Planks and reagents carried). Esc
on the categories closes the menu for free; Esc on a category's recipes returns to the
categories.

| Recipe | Inputs | Output | Minutes | Station |
| --- | --- | --- | --- | --- |
| Axe | 2 Wood, 1 Stone | 1 Axe | 30 | none |
| Pick | 2 Wood, 2 Stone | 1 Pick | 30 | none |

Refined materials, stations and food are in [CRAFT-0008](#craft-0008), building pieces in
[CRAFT-0009](#craft-0009), arms and armour in [CRAFT-0013](#craft-0013), torches and keys in
[CRAFT-0014](#craft-0014), and alchemy in [CRAFT-0015](#craft-0015) to
[CRAFT-0017](#craft-0017).

Choosing a row checks, in order: the command still applies ("Not here!"), the station is
within reach ("Thou dost need a Sawmill nearby!", "Thou dost need an Alchemy table nearby!"), the party carries every input
("Thou dost need 2 Wood, 2 Stone."), and every output fits its stack ("Thou canst carry
no more Pick!"; food stops at 9999, pieces at 99). A refusal spends nothing and no time.
A craft spends the inputs, adds the outputs, prints "Crafted 1 Axe." and costs exactly
the recipe's minutes, the ordinary turn included (two minutes on the overworld, one on a
deed map). Crafting consumes no game RNG.

Rationale: tools need no station so play can start anywhere; everything else is made at
stations built on deed maps.

**Implementation:** `internal/references/crafting_recipes.go`, `internal/game_state/crafting_craft.go`, `cmd/ultimav/gamescene_crafting.go`
**Tests:** `internal/game_state/crafting_craft_test.go`, `cmd/ultimav/gamescene_crafting_gather_test.go`

### CRAFT-0008

**Stations, refined materials and cooking**

**Kind:** rule
**Uses:** CRAFT-0007, CRAFT-0010
**Original:** Established
**Redux:** Intentional deviation

A station is a build piece ([CRAFT-0009](#craft-0009)) placed on a deed map
([CRAFT-0010](#craft-0010)). A recipe that needs a station can be made only while a placed
piece serving as that station stands within one square of the party, diagonals and the
party's own square included (Chebyshev distance 1), on the deed map the party stands on.
The overworld has no stations. Each station reuses an original tile and has its own Look
text:

| Station | Tile | Look |
| --- | --- | --- |
| Workbench | fire logs (163) | a sturdy workbench with a vise |
| Sawmill | stocks (132) | a saw frame for cutting planks |
| Forge | anvil (88) | a forge and anvil |
| Alchemy table | water jug table (169) | an alchemist's table with flask and retort |
| Fireplace | fireplace (188) | a hearth fit for cooking |

A placed Fireplace keeps the original fireplace's properties: it can be walked on and burns
whoever does.

| Recipe | Inputs | Output | Minutes | Station |
| --- | --- | --- | --- | --- |
| Planks | 2 Wood | 4 Planks | 20 | Sawmill |
| Cut stone | 3 Stone | 2 Cut stone | 20 | Workbench |
| Ingots | 2 Ore, 1 Wood | 1 Ingots | 30 | Forge |
| Workbench | 6 Wood, 2 Stone | 1 Workbench | 60 | none |
| Sawmill | 8 Wood, 4 Stone | 1 Sawmill | 60 | none |
| Forge | 8 Cut stone, 3 Ore | 1 Forge | 90 | Workbench |
| Alchemy table | 6 Planks, 1 Ingots | 1 Alchemy table | 60 | Workbench |
| Fireplace | 6 Cut stone | 1 Fireplace | 60 | Workbench |
| Cook food | 2 Forage, 1 Wood | 3 Food | 20 | Fireplace |

Cooked food is the party's original food stock (cap 9999), eaten at meals as bought
food is. Cooking is in the Provisions category with torches and keys
([CRAFT-0014](#craft-0014)). The Forge and Workbench also make arms and armour
([CRAFT-0013](#craft-0013)); the Alchemy table mixes potions, writes scrolls and refines
reagents ([CRAFT-0015](#craft-0015) to [CRAFT-0017](#craft-0017)).

A party with nothing can reach every station: bare hands win Wood and Stone
([CRAFT-0005](#craft-0005)), which make an Axe and a Pick; tools gather enough for the
Workbench and Sawmill, which need no station; the Workbench makes Cut stone, and with Ore
from the Pick, the Forge.

Rationale: stations give a deed a working purpose and pace the economy; the first two need
no station so the chain can start from bare hands.

**Implementation:** `internal/references/crafting_recipes.go`, `internal/references/crafting_pieces.go`, `internal/game_state/crafting_craft.go`, `internal/game_state/crafting_build.go`
**Tests:** `internal/game_state/crafting_build_test.go`

### CRAFT-0009

**Build pieces**

**Kind:** rule
**Uses:** CRAFT-0007
**Original:** Established
**Redux:** Intentional deviation

Build pieces are crafted into the party's piece stock (small stacks, cap 99, shown in Ztats
under Piece) and then placed with Build ([CRAFT-0010](#craft-0010)). Every piece is an
original map tile and keeps that tile's rules: walls, windows, fences, tables and stations
block movement; floors, chairs and beds can be walked on; an opened door lets the party
through. Unlike town furniture, no built piece can be pushed or pulled ("Won't budge!") or
destroyed by An Ylem (it has no effect), with the option on or off, so a chest, chair, table or station stays on the square it was built on.
Look describes a placed station, bed or storage chest by what it is ("thy own
bed", "thy storage chest", with "(empty)" when it holds nothing); every other piece keeps
its tile's original Look text. Building pieces are made at the Workbench:

| Piece | Category | Tile | Inputs | Made | Minutes |
| --- | --- | --- | --- | --- | --- |
| Stone wall | Walls and doors | stone brick wall (79) | 1 Cut stone | 1 | 10 |
| Window | Walls and doors | glass window (75) | 1 Cut stone, 1 Planks | 1 | 15 |
| Door | Walls and doors | door (184) | 3 Planks | 1 | 20 |
| Fence (east-west), Fence (north-south) | Walls and doors | fences (202, 203) | 1 Planks | 1 | 10 |
| Wood floor | Floors | wood floor (64) | 1 Planks | 2 | 10 |
| Stone floor | Floors | brick floor (68) | 1 Cut stone | 2 | 10 |
| Table (west end, middle, east end) | Tables and chairs | tables (148-150) | 2 Planks | 1 | 15 |
| Chair (forward, left, back, right) | Tables and chairs | chairs (144-147) | 1 Planks | 1 | 10 |
| Bed (head) and Bed (foot) | Furnishings | beds (171, 172) | 4 Planks | 1 of each | 30 |
| Dresser, Bookcase | Furnishings | 173, 90 | 3 Planks | 1 | 20 |
| End table | Furnishings | 165 | 1 Planks | 1 | 10 |
| Barrel | Furnishings | 166 | 2 Planks | 1 | 15 |
| Storage chest | Furnishings | footlocker (175) | 4 Planks | 1 | 20 |

The five stations of [CRAFT-0008](#craft-0008) are pieces of the Stations category. Piece
identities are save identities and are never reordered.

**Implementation:** `internal/references/crafting_pieces.go`, `internal/references/crafting_recipes.go`, `internal/game_state/crafting_build.go`, `internal/game_state/ztats_dossier.go`, `internal/game_state/action_look.go`, `internal/game_state/action_push.go`, `internal/game_state/action_cast_direction.go`
**Tests:** `internal/game_state/crafting_build_test.go`

### CRAFT-0010

**Building and demolishing on a deed map**

**Kind:** rule
**Uses:** CRAFT-0001, CRAFT-0003, CRAFT-0009, CMD-0006
**Original:** Established
**Redux:** Intentional deviation

Alt+B on the party's own deed map prints "Build-" and opens a menu: one row per category
of carried pieces ("Walls and doors (2 kinds)"), then Demolish. With no pieces it offers
only Demolish and says how to make pieces. A category lists its carried pieces with their
counts ("Stone wall (6 carried)"); choosing one prints "Place-" and asks for a direction.
Demolish prints "Demolish-" and asks for a direction. Esc on any menu or prompt is free.

Placing checks the adjacent square, in order; the first failure is printed and nothing is
spent:

| Check | Refusal |
| --- | --- |
| The command applies (option on, on foot, on the deed map) | "Not here!" |
| The square lies on the map | "Thou canst not build there!" |
| No party, creature or item stack stands there | "Something is in the way!" |
| The square is no resource node, fresh or used up, and lies inside the open interior (at least four squares from every edge, clear of the decorative ring and its fringe) | "Thou canst not build there!" |
| An earlier piece there is a floor and the new piece is not a floor | "Something is built there!" |
| With no piece there, the tile is buildable ground (grass, brush, foothills, desert, fields, paths) | "Thou canst not build there!" |
| The party carries the piece | "Thou hast no <piece>!" |

A placement spends one piece, writes its tile as a deed-map override, prints
"Built: <piece>." and records the piece and the ground it covered; a piece placed on a
floor also records the floor beneath. Demolish needs a piece the party built ("Thou didst
not build that!"), the same free square, an empty storage chest ("Empty the chest
first!") and room in the piece's stack ("Thou canst carry no more <piece>!"). It restores
the floor beneath, or else the ground, returns the piece to stock and prints
"Demolished: <piece>.". A built door opens and closes as a town door
([CMD-0006](COMMANDS_REFERENCE.md#cmd-0006)); demolishing it while open also drops its
closing timer, so no door reappears on the cleared square.

Placing and demolishing each cost exactly 5 minutes, the ordinary one-minute deed-map turn
included; a refusal or cancel costs nothing. Neither consumes game RNG. Native save
version 26 keeps each deed's placements and the piece stock; placements survive leaving
and re-entering the deed.

Rationale: building happens at town scale, one tile at a time like every original
building, and nothing built is ever lost.

**Implementation:** `internal/game_state/crafting_build.go`, `internal/game_state/crafting_save.go`, `cmd/ultimav/gamescene_crafting_build.go`
**Tests:** `internal/game_state/crafting_build_test.go`, `cmd/ultimav/gamescene_crafting_build_test.go`, `internal/save_snapshot/validation_v26_test.go`

### CRAFT-0011

**Storage chests**

**Kind:** rule
**Uses:** CRAFT-0010, KARMA-0005
**Original:** Established
**Redux:** Intentional deviation

Open on a storage chest the party built on its own deed map, with the option on, prints
"Storage chest" and shows the chest: what it holds, and Store and Take. Each lists an
everything row, then Materials, Tools, Pieces, Gold, and Food, torches, gems and keys (skull
keys included), then Arms and armour, Potions and Scrolls ([CRAFT-0018](#craft-0018)).
Choosing a row moves every unit of those goods; each stack stops at the receiving stack's
cap (materials, food and gold 9999; tools, pieces, torches, gems and keys 99) and the rest
stays where it was. The menu prints "Store materials: 20." or "Nothing to move." and
returns to the chest.

Store's everything row reads "Everything but food, torches, keys and tools": it leaves
Food, Torches, Keys, Skull keys and every tool with the party, as well as readied
equipment ([CRAFT-0018](#craft-0018)), so storing everything never leaves the party to
starve in the dark. Their own rows (Tools; Food, torches, gems and keys) still store them.
Gems and gold go in with everything else. Take's row reads "Everything" and takes every
unit the party can carry.

Opening it never changes karma, unlike a town chest
([KARMA-0005](KARMA.md#karma-0005)). Closing the chest after any move costs the ordinary
one-minute turn; closing it with nothing moved is free. Contents are saved per deed and
survive leaving the deed. A chest must be empty before it can be demolished
([CRAFT-0010](#craft-0010)). With the option off it is an ordinary footlocker and Open
answers "It's open!" as the original does; its contents wait.

Rationale: a home is somewhere to keep things; no karma is lost for taking one's own goods.

**Implementation:** `internal/game_state/crafting_chest.go`, `cmd/ultimav/gamescene_crafting_build.go`, `cmd/ultimav/gamescene_jimmy_integration.go`
**Tests:** `internal/game_state/crafting_build_test.go`, `cmd/ultimav/gamescene_crafting_build_test.go`

### CRAFT-0012

**Resting in the party's own bed**

**Kind:** rule
**Uses:** CAMP-0006, CAMP-0007, CRAFT-0010
**Original:** Established
**Redux:** Intentional deviation

In the original, Hole Up on a town bed ([CAMP-0006](CAMPING_MECHANICS.md#camp-0006)) lets
the party sleep for chosen hours with no healing, and only a paid inn stay
([CAMP-0007](CAMPING_MECHANICS.md#camp-0007)) restores it. With the option on, Hole Up
while standing on the head of a bed the party built on its own deed map sleeps the inn's
night instead, for free:

```
"Hole up-" (the original's town bed prompt, not "Hole up & camp...")
if any active member is poisoned:
    warn that poison not cured by dawn is fatal, as at an inn; Esc cancels for free
"Zzzzzzz...", fade out
Good members fall asleep; active duration magic ends
twelve five-minute steps, then nine-minute steps with ring checks, until the hour is 6
every member not dead: full HP; Avatar and mages MP = Intelligence, bards Int / 2;
    a member still poisoned dies ("Alas! <name> has died!"); sleepers wake
fade in, "Rested!", and the party steps out of bed to the east when it can
```

There is no charge, vacancy check or theft, no meal is eaten in the night (as at an
inn), and the ordinary one-minute turn completes the command. The foot of the bed, beds
elsewhere and the option off keep the original bed rules.

Rationale: Hole Up is the original bed command, so a home bed needs no new key; resting
like an inn without paying is what owning a house buys.

**Implementation:** `internal/game_state/crafting_bed.go`, `internal/game_state/inn_shop.go`, `cmd/ultimav/gamescene_crafting_build.go`, `cmd/ultimav/gamescene_input_smallmap.go`
**Tests:** `internal/game_state/crafting_build_test.go`, `cmd/ultimav/gamescene_crafting_build_test.go`

### CRAFT-0013

**Mundane arms and armour**

**Kind:** rule
**Uses:** CRAFT-0007, CRAFT-0008, EQUIP-0001
**Original:** Established
**Redux:** Intentional deviation

The original sells weapons, armour and shields in shops and makes none. With the option
on, the Forge and the Workbench make the **mundane** original equipment, as the original
items ([EQUIP-0001](EQUIPMENT_ORIGINAL_DATA.md#equip-0001)): the same identities, stats,
prices and save slots, added to the party's equipment inventory (small stacks, cap 99) and
readied, sold and stored like bought ones. Metal is forged from Ingots; wooden, leather and
cloth goods are made at the Workbench from Planks and Forage, with Ingots for metal parts.

Never craftable, and kept as shop and quest rewards: everything magic, enchanted, quest-bound
or precious: Magic Shield, Jewel Shield, Mystic Armour, Sword of Chaos, Magic Bow, Silver
Sword, Magic Axe, Glass Sword, Jeweled Sword, Mystic Sword, the three rings, the Amulet of
Turning, the Spiked Collar and the Ankh. Flaming Oil is mundane but needs lamp oil, which
deeds do not yield, so it is not craftable either.

| Category | Item | Inputs | Made | Minutes | Station | Shop base |
| --- | --- | --- | ---: | ---: | --- | ---: |
| Blades | Dagger | 1 Ingots | 1 | 30 | Forge | 1 |
| Blades | Main Gauche | 2 Ingots | 1 | 45 | Forge | 15 |
| Blades | Short Sword | 3 Ingots, 1 Planks | 1 | 60 | Forge | 40 |
| Blades | Long Sword | 4 Ingots, 1 Planks | 1 | 90 | Forge | 70 |
| Blades | Two-Handed Sword | 7 Ingots, 1 Planks | 1 | 150 | Forge | 200 |
| Axes and maces | Throwing Axe | 1 Ingots, 1 Planks | 1 | 30 | Forge | 3 |
| Axes and maces | Mace | 3 Ingots, 1 Planks | 1 | 60 | Forge | 50 |
| Axes and maces | Morning Star | 4 Ingots, 1 Planks | 1 | 75 | Forge | 60 |
| Axes and maces | Two-Handed Hammer | 5 Ingots, 2 Planks | 1 | 90 | Forge | 85 |
| Axes and maces | Two-Handed Axe | 6 Ingots, 2 Planks | 1 | 120 | Forge | 150 |
| Axes and maces | Halberd | 8 Ingots, 3 Planks | 1 | 150 | Forge | 250 |
| Bows and wooden arms | Club | 2 Planks | 1 | 15 | Workbench | 5 |
| Bows and wooden arms | Sling | 1 Planks, 1 Forage | 1 | 20 | Workbench | 10 |
| Bows and wooden arms | Spear | 2 Planks, 1 Ingots | 1 | 30 | Workbench | 7 |
| Bows and wooden arms | Bow | 6 Planks, 1 Forage | 1 | 90 | Workbench | 75 |
| Bows and wooden arms | Arrows | 1 Planks, 1 Ingots | 10 | 30 | Workbench | 10 |
| Bows and wooden arms | Crossbow | 4 Planks, 2 Ingots | 1 | 120 | Workbench | 150 |
| Bows and wooden arms | Quarrels | 1 Planks, 1 Ingots | 10 | 30 | Workbench | 15 |
| Body armour | Cloth Armour | 4 Forage | 1 | 45 | Workbench | 20 |
| Body armour | Leather Armour | 4 Planks, 2 Forage | 1 | 60 | Workbench | 50 |
| Body armour | Ring Mail | 5 Ingots | 1 | 90 | Forge | 100 |
| Body armour | Scale Mail | 6 Ingots | 1 | 120 | Forge | 150 |
| Body armour | Chain Mail | 8 Ingots | 1 | 180 | Forge | 300 |
| Body armour | Plate Mail | 12 Ingots | 1 | 240 | Forge | 700 |
| Helms and shields | Leather Helm | 2 Planks, 1 Forage | 1 | 30 | Workbench | 15 |
| Helms and shields | Chain Coif | 2 Ingots | 1 | 60 | Forge | 50 |
| Helms and shields | Iron Helm | 3 Ingots | 1 | 75 | Forge | 120 |
| Helms and shields | Spiked Helm | 4 Ingots | 1 | 90 | Forge | 150 |
| Helms and shields | Small Shield | 3 Planks, 1 Ingots | 1 | 45 | Workbench | 40 |
| Helms and shields | Large Shield | 5 Planks, 2 Ingots | 1 | 60 | Workbench | 70 |
| Helms and shields | Spiked Shield | 4 Ingots, 3 Planks | 1 | 90 | Forge | 120 |

Crafting follows [CRAFT-0007](#craft-0007): the station within one step, every input
carried, and room for the output ("Thou canst carry no more Arrows!" when 10 more would
pass 99). Equipment is only ever an output, never an input, so a readied item is never
spent.

Rationale: each Ingots costs about 50 minutes of mining and smelting, so a forged item
costs hours of labour that grow with its shop price (Plate Mail about 14 hours), while
cheap shop goods stay cheaper to buy. The Bow (6 Planks and 1 Forage, 90 minutes) and the
Crossbow (120 minutes) take more wood and time than their shop prices alone would suggest,
so making them to sell is no quicker gold than other work. A self-sufficient party can arm
itself, but magic gear still has to be found or bought.

**Implementation:** `internal/references/crafting_recipes_items.go`, `internal/game_state/crafting_craft.go`
**Tests:** `internal/references/crafting_recipes_items_test.go`, `internal/game_state/crafting_items_test.go`

### CRAFT-0014

**Torches and keys**

**Kind:** rule
**Uses:** CRAFT-0007, CRAFT-0008
**Original:** Established
**Redux:** Intentional deviation

The original buys torches and keys. With the option on, the Provisions category of the Craft
menu (formerly Food) holds cooking ([CRAFT-0008](#craft-0008)) and two more recipes, which
add to the party's original provision stacks:

| Recipe | Inputs | Output | Minutes | Station |
| --- | --- | --- | ---: | --- |
| Torches | 1 Wood, 1 Forage | 2 Torches | 10 | Workbench |
| Key | 1 Ingots | 1 Keys | 20 | Forge |

Torches and keys are small stacks (cap 99). Food still comes only from the Fireplace.

Rationale: light and keys are the two provisions a party burns through away from towns.

**Implementation:** `internal/references/crafting_recipes_items.go`
**Tests:** `internal/game_state/crafting_items_test.go`

### CRAFT-0015

**Potions**

**Kind:** rule
**Uses:** CRAFT-0007, CRAFT-0008
**Original:** Established
**Redux:** Intentional deviation

The original finds potions but never mixes them. With the option on, the Alchemy table
mixes each of the eight original potions from a fixed reagent recipe, one of each listed
reagent, into the party's potion inventory (small stacks, cap 99). Crafted potions are the
original potions and keep their original effects, including unreliable labels. A recipe
borrows the reagents of the matching spell where one exists.

| Potion | Inputs | Minutes | Based on |
| --- | --- | ---: | --- |
| Blue Potion (awaken) | Sulfurous Ash, Ginseng, Garlic | 30 | An Zu, with ash |
| Yellow Potion (heal) | Ginseng, Spider Silk | 30 | Mani |
| Red Potion (cure poison) | Ginseng, Garlic | 30 | An Nox |
| Green Potion (poison) | Blood Moss, Nightshade | 30 | - |
| Orange Potion (sleep) | Ginseng, Spider Silk, Nightshade | 30 | In Zu |
| Purple Potion (rat form) | Spider Silk, Blood Moss, Mandrake Root | 30 | - |
| Black Potion (invisibility) | Blood Moss, Nightshade, Mandrake Root | 30 | Sanct Lor |
| White Potion (x-ray) | Sulfurous Ash, Mandrake Root | 30 | Wis An Ylem |

Craft menu rows name reagents by short names, as [CRAFT-0019](#craft-0019) describes.

Rationale: potions are the alchemist's staple; spell reagents make the recipes guessable
for players who know the magic system.

**Implementation:** `internal/references/crafting_recipes_items.go`, `internal/references/item_potion.go`
**Tests:** `internal/references/crafting_recipes_items_test.go`, `internal/game_state/crafting_items_test.go`

### CRAFT-0016

**Scrolls**

**Kind:** rule
**Uses:** CRAFT-0007, CRAFT-0008, CRAFT-0015
**Original:** Established
**Redux:** Intentional deviation

The original finds scrolls but never writes them. With the option on, the Alchemy table
writes each of the eight original scrolls from one of each reagent of the matching spell
and one Planks as paper, into the party's scroll inventory (large stacks, cap 9999). Each
takes 60 minutes. Crafted scrolls are the original scrolls and read as found ones do.

| Scroll | Reagents (plus 1 Planks) |
| --- | --- |
| Vas Lor | Sulfurous Ash, Mandrake Root |
| Rel Hur | Sulfurous Ash, Blood Moss |
| In Sanct | Sulfurous Ash, Ginseng, Garlic |
| In An | Sulfurous Ash, Garlic, Mandrake Root |
| In Quas Wis | Nightshade, Mandrake Root |
| Kal Xen Corp | Garlic, Spider Silk, Blood Moss, Mandrake Root |
| In Mani Corp | Sulfurous Ash, Ginseng, Garlic, Spider Silk, Blood Moss, Mandrake Root |
| An Tym | Garlic, Blood Moss, Mandrake Root |

Rationale: a scroll is a spell kept on paper; it costs the spell's reagents and labour but
no mana or level, which the hour of writing and the rare Mandrake Root balance.

**Implementation:** `internal/references/crafting_recipes_items.go`, `internal/references/spell_reagents.go`
**Tests:** `internal/references/crafting_recipes_items_test.go`, `internal/game_state/crafting_items_test.go`

### CRAFT-0017

**Reagent refining**

**Kind:** rule
**Uses:** CRAFT-0003, CRAFT-0005, CRAFT-0015, CRAFT-0016
**Original:** Established
**Redux:** Intentional deviation

Deed maps grow only five of the eight reagents ([CRAFT-0003](#craft-0003)): Ginseng and
Garlic on grassland; Spider Silk, Blood Moss and Ginseng in woodland; Sulfurous Ash and
Blood Moss in the foothills; Sulfurous Ash in the desert. The Alchemy table refines the
other three from them at a poor rate:

| Recipe | Inputs | Output | Minutes |
| --- | --- | --- | ---: |
| Black Pearl | 3 Sulfurous Ash | 1 Black Pearl | 30 |
| Nightshade | 3 Blood Moss | 1 Nightshade | 30 |
| Mandrake Root | 3 Ginseng | 1 Mandrake Root | 30 |

Refined reagents go into the original reagent inventory (cap 99) and serve spells as bought
ones do. With deeds on the right terrains, every potion and scroll can be made from what
deeds grow alone; reagent shops remain the quicker source.

Rationale: the rare reagents stay rare and slow to make, but no recipe depends on a shop.

**Implementation:** `internal/references/crafting_recipes_items.go`, `internal/references/crafting_data.go`
**Tests:** `internal/references/crafting_recipes_items_test.go`, `internal/game_state/crafting_items_test.go`

### CRAFT-0018

**Storage chests keep arms, potions and scrolls**

**Kind:** rule
**Uses:** CRAFT-0011, CRAFT-0013
**Original:** Established
**Redux:** Intentional deviation

A storage chest ([CRAFT-0011](#craft-0011)) also keeps the party's equipment (every weapon,
armour, shield, ammunition, ring and amulet, magic or not), potions and scrolls. Store and
Take add three rows: Arms and armour, Potions and Scrolls; the everything rows include them.
Storing arms leaves every readied item with the party: only units beyond those readied by
the active party go into the chest. Each chest slot is capped like the party stack it comes
from (equipment and potions 99, scrolls 9999). Native save version 26 keeps them in each
chest's `equipment`, `potions` and `scrolls`.

Rationale: a home's chest should keep spare arms and the alchemist's output, and storing
must never strip a character's readied weapon.

**Implementation:** `internal/game_state/crafting_chest.go`, `internal/game_state/crafting_save.go`, `internal/save_snapshot/validation_v26.go`
**Tests:** `internal/game_state/crafting_items_test.go`, `internal/save_snapshot/validation_v26_test.go`

### CRAFT-0019

**Craft menu rows for arms and alchemy**

**Kind:** rule
**Uses:** CRAFT-0007
**Original:** Established
**Redux:** Intentional deviation

The Craft menu lists 16 categories: Tools, Refined materials, Stations, Walls and doors,
Floors, Tables and chairs, Furnishings, Provisions, Blades, Axes and maces, Bows and wooden
arms, Body armour, Helms and shields, Potions, Scrolls and Reagent refining. No category
lists more than nine recipes.

Recipe rows name reagents by their two-letter short names, counted only above one: SA
Sulfurous Ash, Gi Ginseng, Ga Garlic, SS Spider Silk, BM Blood Moss, BP Black Pearl, NS
Nightshade, MR Mandrake Root ("In Mani Corp: SA Gi Ga SS BM MR, 1 Planks",
"Black Pearl: 3 SA"). The description of a category with reagent inputs lists the carried
Planks and reagents by short name, then the key ("Key: SA Sulfurous Ash, ..."). A row longer than the
menu (52 characters) whose station is out of reach names only the recipe and the station
("In Mani Corp (needs Alchemy table)"); its inputs show at the station. Refusals and
"Crafted" messages use the full names ("Thou dost need an Alchemy table nearby!").

Rationale: rows must fit one line of the menu, which does not scroll, and still say what a
recipe takes.

**Implementation:** `cmd/ultimav/gamescene_crafting.go`, `internal/references/crafting_recipes.go`, `internal/references/item_reagent.go`
**Tests:** `cmd/ultimav/gamescene_crafting_gather_test.go`, `internal/references/crafting_recipes_items_test.go`

### CRAFT-0020

**Raids on deed maps**

**Kind:** rule
**Uses:** CRAFT-0003, CRAFT-0004, ENEMY-0004
**Original:** Established
**Redux:** Intentional deviation

With the option on, monsters suited to the land now and then raid the party's own deed map,
while the party is on it. Each deed-map actor turn (the turn of every ordinary command,
after the actors move) runs:

```
if the party is not on its deed map:            nothing (no count, no RNG)
if the option is off:                           any raiders leave; the count resets
if raiders are on the map, or combat is pending: nothing
count one turn; at 30 turns:
    the count restarts
    roll one in 10; on success, start a raid
```

A raid draws the game RNG in this order: the monster, from the deed terrain's weighted
table; the number of raiders, 1 to 3; then one draw per raider for its square among the
free spawn squares. All raiders of a raid are the same monster. Spawn squares lie in the
decorative ring or its fringe (at most three squares inside the map edge), are passable to
raiders ([CRAFT-0021](#craft-0021)), hold no resource node, placement, party, creature or
item stack, and are at least 8 squares (Manhattan) from the party. Nothing can be built
that close to the edge ([CRAFT-0010](#craft-0010)), so a raider never appears inside a
built room. In a hills or desert ring of mountains that leaves the entry lane and the gaps
in the fringe. A raid prints "Raiders approach!"; with no free square, nothing appears and
nothing is printed.

| Deed terrain | Raiders (weight) |
| --- | --- |
| Grassland | rats (40), fighters (30), orcs (30) |
| Forest | spiders (35), orcs (30), insect swarms (20), ettins (15) |
| Hills | orcs (45), headless (30), trolls (25) |
| Desert | snakes (45), insect swarms (35), orcs (20) |

Every monster is from the original overworld pool ([ENEMY-0004](ENEMIES.md#enemy-0004)) or a
town fighter; the rare and strong ones of that pool (wisps, dragons, daemons) never raid. At these odds a party that stays on its
deed sees a raid about every five hours of game time on average. The count is not saved and
starts again at zero each time the party enters a deed map, and nothing counts while the
party is elsewhere, resting or in combat. Off deed maps, and on deed maps with the option off,
the hook draws no RNG, so every original RNG sequence is unchanged.

Rationale: walls and doors are worth building only if something tries to get in.

**Implementation:** `internal/references/crafting_data.go`, `internal/game_state/crafting_raids.go`, `internal/game_state/turn.go`
**Tests:** `internal/game_state/crafting_raids_test.go`

### CRAFT-0021

**Raiders hunt the party and fight it on the deed**

**Kind:** rule
**Uses:** CRAFT-0010, CRAFT-0020, CRIME-0004, NPC-0004
**Original:** Established
**Redux:** Intentional deviation

Each actor phase, a raider on the party's floor takes one step toward the party:

```
if next to the party (orthogonally): wait; contact is checked before the actors move
else follow the shortest route to the party, around walls, closed doors and other units
else, with the way blocked only by units, follow the shortest route around walls and doors
else (the party walled in) step to the first free neighbour, east, north, west, south,
     that is closer; stop when none is
```

A raider may stand on any square town NPCs may walk on ([NPC-0004](NPC_SCHEDULES.md#npc-0004))
except a door: **raiders cannot open doors**, so a closed door is a wall to them. An open
door shows its floor and lets them in, so a party that leaves a door open while raiders
roam may be followed inside. Raiders walk over built floors and around everything else,
draw no RNG to move, and never change a tile: walls, doors and furniture are never damaged
or removed.

Before the actors move, a raider next to the party starts combat, as a town Shadowlord
does ([CRIME-0004](CRIME_PRISON.md#crime-0004)); Attack on an adjacent raider starts the same
battle and completes the turn. The whole raid fights together: every raider on the map joins
the battle and leaves the deed map, whatever the outcome, and the battle prints the
monster's plural name and "Attacked!". The battlefield follows the deed terrain: grassland
the glade, forest the treed map, hills the mountains, desert the desert. A raider standing on
a built piece (an open doorway or a built floor) fights on the brick floor of a town
interior. There are no karma, guard or crime consequences. Combat returns the party to its
square on the deed map with every placement intact, and the raid count resumes.

Leaving the deed map removes its raiders, and re-entering starts without any. Raiders are
transient and never saved: a save made during a raid loads the deed map without them.

Rationale: raiders exist to make building worthwhile, never to undo it.

**Implementation:** `internal/ai/npc_ai_deed_raiders.go`, `internal/game_state/crafting_raids.go`, `internal/game_state/combat_small_map_encounter.go`, `internal/game_state/action_attack.go`, `internal/game_state/save_snapshot.go`, `internal/map_units/npc_enemy.go`
**Tests:** `internal/game_state/crafting_raids_test.go`

### CRAFT-0022

**No moongates on deed maps**

**Kind:** rule
**Uses:** MOON-0003, CONSUME-0001, CRAFT-0004
**Original:** Established
**Redux:** Intentional deviation

The original prints nothing when the moongates rise at 20:00 or fade at 05:00; Redux
announces the shared stage change ([MOON-0003](MOONSTONE_LOCATIONS.md#moon-0003)) with
"Instantly, a shimmering blue door springs up!" and "The shimmering door fades from view."
wherever the party is. On a deed map those lines stay silent; the stage still changes, and
towns and the overworld keep both lines.

No gate can stand on a deed map. The original buries a moonstone only on map numbers below
0x21 ([CONSUME-0001](POTIONS_SCROLLS.md#consume-0001)), and the shared deed location is 128,
so Use on a moonstone there answers "Not here!" and the item list says "You cannot bury it
here." Without this, a stone buried on one deed would raise a gate on every deed map, since
they share one location value.

Rationale: an announcement for a door no one can see reads as a bug.

**Implementation:** `internal/game_state/moongate_events.go`, `internal/game_state/action_use.go`, `internal/game_state/action_use_projection.go`, `internal/game_state/journal_moonstones.go`
**Tests:** `internal/game_state/crafting_moongate_test.go`
