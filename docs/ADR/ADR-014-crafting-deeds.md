# ADR-014: Optional crafting on deeded land

**Status**: Proposed (fork branch `feat/craft-v2`)

This number was also used on the abandoned branch `feat/crafting-housing` (overworld-scale
crafting, plus a homestead plan there numbered ADR-015). Neither document was ever merged.
This ADR replaces both. The old branch stays on `origin` as a reference.

## Context

The original game has no gathering, crafting, building or land ownership. A first attempt
placed walls and furniture directly on overworld tiles. That is the wrong scale: one
overworld square stands for a whole area, so a house read as a giant's floor plan. That
attempt also made only two tools, which mattered for nothing outside crafting, and it
felled forests across the whole world permanently.

The original game already shows the right pattern: a dwelling is one overworld square (a
hut), and Enter leads to a 32x32 town-scale map. Iolo's hut is the reference.

## Decision

Add an opt-in crafting system, off by default, behind the gameplay option
`Gameplay.Crafting.Enabled`. While the option is off, every original rule applies.

### Deeds

- On a suitable overworld square, the party pays a **flat gold price** and a deed is
  created on the spot. There is **no limit** on the number of deeds.
- A suitable square:
  - is on the overworld surface, with the party on foot;
  - is buildable ground (not water, mountains or swamp);
  - is not a world location, a settlement sprite or a moongate square;
  - is neither a deed square nor orthogonally adjacent to one.
- The square becomes a hut marker, stored as an ordinary overworld map mutation. Enter on
  it leads to the deed map. Walking off any edge of the deed map returns to the hut square.
- A deed has a player-chosen name and is pinned on the Atlas.

### Deed maps

- A deed map is **32x32**, the size of original towns and huts, with one floor.
- It is **generated from the terrain of the deeded square** (grassland, forest, hills,
  desert, and so on). The terrain sets the scenery and the resource mix: forest gives
  mostly wood, hills give stone and ore.
- Generation is deterministic from a seed derived from the deed. The save stores the
  seed, the terrain kind, and the differences from the generated map (built pieces and
  resource state). It does not store all 1024 tiles, so a save with many deeds stays
  small. Changes to the generator are versioned, so existing deeds keep their layout.
- **Location identity.** All deeds share one synthetic `references.Location` value for
  deed maps. The active deed ID is part of the game state. Only one deed map is loaded at
  a time, so the number of deeds is not limited by the `Location` byte. Code that assumes
  a location comes from DATA.OVL gets an explicit branch for deed maps. This affects the
  location name, entering and exiting text, exit position, audio routing, the sextant,
  the NPC tables, the legacy SAVED.GAM export, and the loops over original towns.
- **Raiders.** Monsters suited to the terrain occasionally appear on a deed map, so walls
  and doors are useful. Default until tuned: raids happen only while the party is on the
  map, and raiders never destroy built pieces.

### Gathering

- Gathering (Alt+G) works **only on deed maps**. The overworld is never changed.
- Resource nodes are tiles on the deed map: trees and brush give Wood, rocks give Stone,
  ore outcrops give Ore. A node is used up after a number of harvests.
- **Respawn follows Trammel.** Each Trammel phase change (16 per 28-day month, about every
  1.75 days; `internal/datetime/ultima_date.go`) restores a share of the used-up nodes on
  every deed. The phase change is the trigger, not elapsed time, so restoring catches up
  in one step when the party returns.
- Tools: an Axe for wood and a Pick for stone and ore. Bare hands can gather a little wood
  and stone, so a new player can make the first tools.

### Materials, stations and recipes

- **Raw materials:** Wood, Stone, Ore. **Refined materials:** Planks, Cut stone, Ingots.
- **Stations** are pieces built on a deed map. A recipe requires its station within one
  square of the party.
  - Workbench: Cut stone, furniture and building pieces, wooden weapons and bows.
  - Sawmill: Planks.
  - Forge: Ingots, metal weapons, armour, shields.
  - Alchemy table: potions.
- Tools (Axe, Pick) need no station and can be crafted anywhere, so play can start.
- **Outputs:**
  - Gathering tools.
  - Building pieces and furniture. They are crafted into the inventory first, then placed
    with Build; demolishing a piece returns it to the inventory.
  - Original game items: weapons, armour and shields, provisions (torches, food, keys),
    and reagents and potions. Reagents are grown or harvested on deed maps. Crafted items
    are the original items, with no new item types, so they fit the existing equipment,
    shops and saves.
- All costs, yields, times, prices and recipes live in one typed table in
  `internal/references`.

### Commands

- Alt+G Gather, Alt+C Craft and Alt+B Build. Build stakes a deed on the overworld; on a
  deed map, it places crafted pieces and demolishes them.
- Chords are matched exactly (Alt alone), with one shared guard ahead of every map
  handler. The guard answers "What?" when the option is off and "Not here!" where a
  command does not apply. A chord never falls through to Get, Cast or Board.
- Each command charges exactly the time stated in its Game Fact, including the ordinary
  turn.

### Carried over from the first attempt

Kept: the option gate, the single data table, capped inventory stacks for materials and
tools, rows in Ztats, the storage chest (no karma cost, now on deed maps), Game Facts
documentation (topic CRAFTING, ids CRAFT-*), and a new versioned section in the native
save. The first attempt's review findings are designed out: Craft's missing surface
check, time drift on menu commands, inconsistent chord routing, moongate claims and
permanent overworld deforestation.

## Delivery order

1. Deeds, map generation, Enter and Exit, the deed `Location` branch, save support.
2. Gathering, resource nodes, Trammel respawn, tools.
3. Stations, refined materials, crafted pieces, Build and Demolish, the storage chest.
4. Original items: weapons, armour, shields, provisions.
5. Reagents and potions.
6. Raiders.

Each step ships with its Game Facts, tests, and a manual check in the running game.

## Consequences

- Building happens at the scale of every other building in the game, and the overworld is
  only changed by hut markers.
- Deed maps are a new kind of location: generated, saved, and outside the original
  location table. This is the main technical risk, and tests guard it.
- The native save version is bumped from 25. A diverging save format makes later
  upstream save bumps harder to merge.
- Crafted original items affect the game economy. Prices and costs are tuned so that
  crafting does not replace shops or quest rewards outright.
