# Crafting, land claims and housing

The original game has no gathering, crafting, building or land ownership. Every rule
in this topic describes an optional Redux addition, off by default and enabled with
the Configure option **Crafting and Housing** (`Gameplay.Crafting.Enabled`). While
the option is off, the added chords answer "What?", claims grant no protection, and
the game behaves as the original. Tunable numbers live in one table,
`internal/references/crafting_data.go`; this topic quotes the defaults.

### CRAFT-0001

**Optional crafting system and its state**

**Kind:** rule
**Uses:** none
**Original:** Established
**Redux:** Intentional deviation

The original binds no Alt chords and keeps no material, tool or ownership state.
Redux adds three surface-overworld commands: Alt+G Gather, Alt+C Craft and Alt+B
Build. They work only on the overworld surface (floor 0), on foot, never in towns,
dungeons, the underworld, combat or aboard a vehicle.

The party carries two materials (Wood, Stone; large stacks capped at 9999) and two
tools (Axe, Pick; small stacks capped at 99). Ztats lists them under Material and
Tool. Native save version 26 stores materials, tools, land claims, placed pieces
and storage chest contents in its `crafting` member; saves from versions 24 and 25
load with all of them empty. Placed and felled tiles are ordinary overworld map
mutations.

Rationale: a house-building sandbox for players who have finished or want to
extend the quest, kept separate from every original rule it does not touch.

**Implementation:** `internal/references/crafting_data.go`, `internal/game_state/crafting.go`, `cmd/ultimav/gamescene_crafting.go`
**Tests:** `internal/game_state/crafting_test.go`

### CRAFT-0002

**Gathering wood and stone**

**Kind:** rule
**Uses:** CRAFT-0001
**Original:** Established
**Redux:** Intentional deviation

Gather takes a direction and examines the adjacent overworld terrain. An object on
the square, or terrain that yields nothing, refuses with "Nothing to gather!".

| Terrain | Tool | Tool yield | Tile after a tool harvest | Bare hands |
| --- | --- | --- | --- | --- |
| Tree, dead tree | Axe | 2..4 Wood | Grass | 1 Wood, 1 in 3 |
| Tropical forest (Forest4) | Axe | 2..4 Wood | Forest | 1 Wood, 1 in 3 |
| Forest (Forest3) | Axe | 2..4 Wood | Thick brush | 1 Wood, 1 in 3 |
| Thick brush (Forest2) | Axe | 2..4 Wood | Brush | 1 Wood, 1 in 3 |
| Brush (Forest1) | Axe | 2..4 Wood | Grass | 1 Wood, 1 in 3 |
| Hills | Pick | 1..2 Stone | unchanged | 1 Stone, 1 in 3 |
| Mountains | Pick | 2..3 Stone | unchanged | refused: "You need a Pick!" |

Bounds are inclusive and use the game RNG. Bare hands never change terrain, so a
new player can always collect enough to craft the first tools. Every attempt,
successful or not, spends one ordinary surface turn plus labour up to 10 minutes.

**Implementation:** `internal/game_state/action_gather.go`

### CRAFT-0003

**Crafting tools**

**Kind:** rule
**Uses:** CRAFT-0001
**Original:** Established
**Redux:** Intentional deviation

Craft opens a recipe menu. An Axe costs 3 Wood and 2 Stone; a Pick costs 4 Wood and
3 Stone. Each takes 30 minutes. A recipe the party cannot afford is refused with the
missing cost and spends nothing. Cancelling the menu is free.

**Implementation:** `internal/game_state/action_craft.go`

### CRAFT-0004

**Staking a land claim**

**Kind:** rule
**Uses:** CRAFT-0001
**Original:** Established
**Redux:** Intentional deviation

From the Build menu, outside any claim, the party may stake a 16x16 claim centred
on its square (the party's square is at offset 8,8 from the north-west corner).
Claims wrap with the overworld. A claim is refused when the party already holds 4,
when it would overlap another claim, when any square holds a settlement, castle,
shrine, dungeon or other entrance, or when more than half of it is water. It costs
300 gold plus 10 Wood and 5 Stone of stakes, takes a name of 1 to 24 printable
characters, and spends 10 minutes. The claim's name is pinned on the Atlas at its
centre without needing a sextant fix; a full Atlas leaves it unpinned. Claims are
permanent.

**Implementation:** `internal/game_state/action_build.go`

### CRAFT-0005

**Building and demolishing**

**Kind:** rule
**Uses:** CRAFT-0004
**Original:** Established
**Redux:** Intentional deviation

Build places one piece on an adjacent square inside one of the party's claims. The
square must hold no object or item, and its terrain must be grass, brush, desert,
path, plowed field or a placed floor. Pieces are original tiles: wood and stone
floors, stone and rough walls, windows, fences, doors, tables, chairs, beds, a
dresser, bookcase, end table, barrel, fireplace and storage chest. Passability,
light and sight follow each tile's original properties, so walls block movement.
Each placement spends its material cost and 5 minutes.

Building over a placed floor keeps the terrain under the floor and refunds half the
floor's cost. Demolish works only on a piece the party placed: it restores that
terrain, refunds half the piece's cost (rounded down) and spends 5 minutes. A
storage chest must be empty first. An opened placed door stays open, as every
overworld door does.

**Implementation:** `internal/game_state/action_build.go`

### CRAFT-0006

**Safety inside a claim**

**Kind:** rule
**Uses:** CRAFT-0004
**Original:** Established
**Redux:** Intentional deviation

The original generates overworld monsters anywhere the terrain allows and can ambush
any camp. While the option is on, no natural monster generation is attempted while
the party stands inside one of its claims, no monster is generated onto a claimed
square, and a camp pitched inside a claim makes no ambush roll. Monsters already
present still move and fight normally. Forced debug ambushes are unaffected.

**Implementation:** `internal/ai/npc_ai_controller_large_map.go`, `internal/game_state/action_hole_up_camping.go`

### CRAFT-0007

**Storage chests**

**Kind:** rule
**Uses:** CRAFT-0005
**Original:** Established
**Redux:** Intentional deviation

Open on a storage chest the party built shows its contents and lets the party store
or take, per category or everything at once: materials, tools, gold, food, and
torches, gems and keys. Moves stop at the receiving stack's cap. A storage chest
holds no random loot, and opening it never costs karma: it is the party's own
property. Contents are saved with the claim.

**Implementation:** `internal/game_state/crafting.go`, `cmd/ultimav/gamescene_crafting.go`
