# ADR-014: Optional crafting and housing

**Status**: Proposed (fork branch `feat/crafting-housing`)

## Context

Players asked for a sandbox after or alongside the quest: gather wood and stone, make
tools, claim land and build a house. The original game has none of this. Redux's
rules are faithful by default, every deviation is documented in Game Facts, and the
native save format is strict and versioned.

## Decision

Add the system as an opt-in gameplay option, `Gameplay.Crafting.Enabled`, off by
default. Rules live in `internal/game_state` (`action_gather.go`, `action_craft.go`,
`action_build.go`, `crafting.go`); tunables and catalogs are typed Go in
`internal/references/crafting_data.go`; the scene only routes keys and menus
(`cmd/ultimav/gamescene_crafting.go`). The behaviour is specified in
[CRAFTING](../gamefacts/CRAFTING.md) (CRAFT-0001 to CRAFT-0007).

- **Scope:** overworld surface only. Overworld base-layer tiles are already saved as
  map mutations against BRIT.DAT, so placed and felled tiles need no new map
  persistence. Towns are rebuilt from source on entry and are excluded.
- **Keys:** Alt+G, Alt+C and Alt+B. Every letter is bound by the original; Alt
  chords are unused. With the option off they answer "What?".
- **Pieces are original tiles.** Walls, doors and furniture reuse existing sprites,
  so passability, light and rendering need no new data.
- **Save:** snapshot version 26 adds a required `crafting` member (materials, tools,
  claims, placements with the tile each covered, storage chest contents). Versions 24
  and 25 promote with it empty.
- **Claims** gate building and grant two protections, both only while the option is
  on: no natural monster generation in or into a claim, and no camping ambush roll.

## Consequences

- One more save version and frozen V25 schema to maintain.
- The large-map AI controller takes an optional spawn predicate; nil keeps the
  original generation exactly, including its RNG draws.
- Opened overworld doors stay open, a pre-existing limitation that now affects
  player houses.
- The Debug Workbench does not yet expose the option.

## Alternatives Rejected

- **Extending `Provision` or `QuestItem` for materials:** their IDs are tied to the
  SAVED.GAM layout and fixed-size snapshot arrays.
- **Building in towns:** needs a per-town persistent overlay and interacts with NPC
  schedules; out of scope for a first version.
- **Repurposing D and W:** they keep the original "What?" feedback.
