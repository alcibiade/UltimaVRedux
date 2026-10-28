# ADR-015: One-square land claims with a town-scale homestead map

**Status**: Proposed (implementation plan, fork branch `feat/crafting-housing`). Amends
[ADR-014](ADR-014-crafting-and-housing.md).

## Context

In ADR-014, a land claim is a 16x16 rectangle of overworld tiles, centred on the
party. Build pieces are written straight into the overworld `MapLayer`, and those
tiles persist as BRIT.DAT map mutations.

Every build piece is a town-scale sprite: beds, tables, chairs, walls, doors. Placed
on the world map, each one covers a whole overworld square, the size of a forest or
mountain tile. A house built that way reads as a giant's floor plan, not as a home.

The original game shows dwellings as one overworld square (a hut or a town icon) that
the party enters to reach a 32x32 town-scale map. Iolo's hut is the reference. Claims
should work the same way.

## Decision

A claim is a **single overworld square**, marked by a hut sprite. The party stakes it,
then uses **Enter** to reach a **32x32 town-scale map** where all building happens.
Walking off any edge leaves to the claim's overworld square.

Decisions taken with the user:

| Topic | Decision |
|---|---|
| Size of the claim map | 32x32, the same as original towns and huts (`XSmallMapTiles`/`YSmallMapTiles`). |
| Starting terrain | Generated from the claimed square's terrain: grassland, woodland or desert. |
| Gathering | Overworld only. The claim map is for building and storage. |
| Save format | Snapshot **v26 is redefined in place**. v26 exists only on this unmerged branch, so saves made with 16x16 claims stop loading and get no migration. |

Line references below were taken at commit `10ee782` plus the uncommitted review
fixes, so they may drift.

### 1. Claim model

Files: `internal/game_state/crafting.go`, `internal/references/crafting_data.go`

- **`LandClaim` becomes `{ID, Name, Position, Terrain}`.**
  - `Position` is the single overworld square.
  - `Terrain` is the generator kind: `Grassland`, `Woodland` or `Desert`.
  - `Origin`, `Center()`, `Contains()` and the wrapped-offset helper are removed.
  - `LandClaimAt(pos)` becomes an exact square match.
- **Each claim owns its map**, a `*references.SmallLocationReference` with one floor
  (floor 0). It is the single source of truth for the claim's tiles.
- **Each claim gets a synthetic `references.Location`.**
  - Add `FirstLandClaimLocation Location = 0x80` and `IsLandClaimLocation()` in
    `internal/references/locations.go`. Claim `i` uses `0x80 + i`; `MaxLandClaims`
    stays at 4.
  - The range 0x80-0x83 is clear of the original values (0x00-0x28) and the virtual
    ones (0xF9-0xFF). `Location.GetMapType()` already returns `SmallMapType` by
    default, so these values route as small maps with no change.
  - Update the comment on `Location` to mention the claim range.
- **Constants:**
  - Remove `LandClaimSize` and `MaxLandClaimWaterTiles`.
  - Keep `LandClaimGoldCost`, `LandClaimStakeCost`, `LandClaimStakeMinutes`,
    `MaxLandClaims` and `MaxLandClaimNameLength`.
  - Add `LandClaimMapEntry = {15, 30}`, matching the fixed small-map entry point in
    `internal/game_state/action_enter.go:15-17`.

### 2. Staking a claim

File: `internal/game_state/action_build.go`

`ProposeLandClaim` checks only the square the party stands on. That square must be:

- on the overworld surface (`isOnOverworldSurface`), with no vehicle (`IsCraftingSite`);
- buildable ground (`references.IsBuildableGround`). That rules out water, mountains
  and swamp, so the water-count rule is no longer needed;
- not a world location (`WorldLocations.GetLocationByPosition`) and not a settlement
  sprite (`isSettlementSprite`);
- not a moongate (`isMoongateSquare`);
- not an existing claim, and not orthogonally adjacent to one, so huts never touch.

The claim limit and the gold and material costs are unchanged.

`StakeLandClaim`:

1. Spends the cost and advances `LandClaimStakeMinutes`, as today.
2. Classifies the square's terrain:
   - `Forest1` gives Woodland;
   - desert tiles give Desert;
   - everything else gives Grassland.
3. Generates the claim map (§3) and registers it under the claim's location.
4. Writes `indexes.Hut` (sprite 16) into the overworld `MapLayer` at the square and
   clears `MapOverrideLayer`, reusing the write in `ActionBuildLargeMap`.
   - The marker persists through the existing `MapMutations` diff against BRIT.DAT,
     the same way felled forests do.
   - The hut is passable, so the party stays on it and can press E straight away.
5. Keeps the Atlas pin (`pinLandClaimOnAtlas`) and the naming workflow
   (`cmd/ultimav/gamescene_crafting.go:266`, `startLandClaim`).

`isSettlementSprite` already includes `Hut`. Staking a claim next to another claim is
refused by the adjacency rule, and a claim's own square by "already claimed". The
player-facing messages in `LandClaimRejection` change to match: drop "too wet", add
"too close to your other claim".

### 3. The generated claim map

Package: `internal/references`

- **New constructor** `NewGeneratedSmallLocationReference(loc, name, tiles *[32][32]byte, outer indexes.SpriteIndex)`.
  - `NewSingleSmallMapReference` reads `dataOvl.LocationNames[location]`, which would
    panic for a new value, so the new constructor must not touch DATA.OVL or the NPC
    files.
  - It fills `rawData[0]` (via `addBlankFloor`); `ListOfFloors` is `{0}` and
    `HasBasement` is false.
  - The NPC list is empty. `NPCReferences.getNPCIndexesByLocation` uses
    `(loc-1)*npcsPerTown` and would read out of range, so it must never be called for
    a claim.
  - The outer tile is Grass for Grassland and Woodland, Desert1 for Desert. See
    `GetOuterTile`; the renderer and the outer-terrain dither read it.
  - The entering text is "Entering <name>" and the exiting text "Leaving <name>",
    replacing the "NOT IMPLEMENTED" fallback in `getEnteringText`/`GetExitingText`.
- **New mutator** `SetTileNumber(floor, pos, sprite)` on `SmallLocationReference`, used
  by build and demolish.
- **New file `references/land_claim_map.go`** with
  `GenerateLandClaimMap(terrain, seed) *[32][32]byte`.
  - It is deterministic. The seed is derived from the claim ID and overworld position.
    The style follows `BuildGeneratedDungeonCombatMap` (`combat_map_dungeon_generated.go`).
  - **Interior** (about 24x24, centred): only tiles that `IsBuildableGround` accepts,
    so the whole interior can be built on.
    - Grassland: grass with scattered `Forest1` and plowed or path accents.
    - Woodland: denser `Forest1`.
    - Desert: desert tiles.
  - **Border ring** (about 4 tiles): decorative tiles that cannot be built on (trees
    and dense forest, rocks, dunes) so the plot reads as an enclosed clearing.
    Gathering is overworld-only, so the ring is never cleared; it frames the
    buildable area.
  - **Entry path**: a clear path from `LandClaimMapEntry {15,30}` to the south edge, so
    the party always enters and leaves on walkable ground.
- **Registry**: `LocationReferences` gains `generated map[Location]*SmallLocationReference`
  with `RegisterGenerated(slr)` and `ClearGenerated()`. `GetLocationReference`
  (`location_references.go:40`) checks `generated` first.
  - `GameReferences` lives longer than one game, so `restoreCrafting` and new-game
    setup must call `ClearGenerated()` before registering. Claims from an earlier game
    must never leak into the next load.

### 4. Entering and leaving

File: `internal/game_state/action_enter.go`

- **Enter.** In `ResolveEnterLargeMap` (`:48`), check `LandClaimAt(partyPosition)`
  before `WorldLocations.GetLocationByPosition` (`:97`).
  - On a match, call `StartEnterLocationWorkflow(claimSLR)`. That reuses the fade,
    `TransitionToLocation(loc, 0, {15,30}, TransitionFullLocation, true)` and the
    entering popup.
  - With `saveLastLargeMap = true`, `TransitionToLocation` records
    `LastLargeMapPosition`/`LastLargeMapFloor` (`map_transitions.go:85-88`) as usual.
  - `enterLocationDescription` (`:180`) says "Enter <claim name>" instead of "Enter hut".
- **Leave.** Walking off the edge already raises the "Dost thou wish to leave?"
  prompt for any small map (`cmd/ultimav/gamescene_movement.go:138-166`, using
  `IsOutOfBounds` and `slr.GetMaxX/Y`).
  - `ThenExitLocationTransition` (`:363`) takes the exit position from
    `WorldLocations.GetWorldLocation(currentLocation)` (`:382`), which panics for a
    location it does not know (`world_locations.go:79`). Branch first: if
    `IsLandClaimLocation(currentLocation)`, exit to the claim's `Position` on floor 0.
- **Other paths that assume an original location.** Audit each one; most only need a
  fallback for claim locations:
  - the location name in the status line and title (it must use the claim's name, not
    `dataOvl.LocationNames`);
  - audio routing (`internal/audio/routing.go:37`): use the outdoor or town default;
  - `sextantSettlementExterior` (`sextant.go:85`): report the claim's square;
  - loops over `Moonglow..SerpentsHold` (`atlas.go:83`, `party_state.go:98`,
    `debugging/initial_catalog.go:915`): check that claim values fall outside them;
  - daylight and exposure (`layered_maps.go:174` `smallMapOutdoorSeeds`,
    `daylight.go:175`): an open map flood-fills as outdoor, so add a seed entry only if
    one is needed;
  - `GetMapMasterFromLocation`/`GetSmallMapFile` (`small_map.go:40-80`) must never be
    reached for a claim. Small-map rebuilds (`rebuildSmallMapTerrain` →
    `ResetAndCreateSmallMap` → `ReloadSmallMapFloor`) read only
    `slr.GetTileNumber`, so this should already hold; add a test to confirm it.
  - NPC flag validation in `save_snapshot/validation.go:785`.

### 5. Building at town scale

Files: `action_build.go`, `crafting.go`, `cmd/ultimav/gamescene_crafting.go`, the input handlers

- **Actions.** `ActionBuildLargeMap` and `ActionDemolishLargeMap` are replaced by
  `ActionBuildClaimMap(pieceID, direction)` and `ActionDemolishClaimMap(direction)`.
  - They are allowed only when the current location is one of the party's claims.
  - The target is the adjacent square, which must be inside 0..31, not occupied by the
    party or an item stack, and on buildable ground or an earlier placement.
  - They write the claim reference (`SetTileNumber`) and the live `LayeredMap`
    (`SetTileByLayer(MapLayer)` plus `UnSetTileByLayer(MapOverrideLayer)`). Both are
    needed because small maps are rebuilt from the reference on every entry, floor
    change and combat return (`small_map.go:39`).
  - These ADR-014 rules are kept: the first `under` is preserved when replacing a
    placement, demolish refunds half, demolish refuses a non-empty storage chest, and
    each action takes 5 minutes (`advanceCraftingLabor`).
- **State.** `placements` and `homeChests` are keyed by `(claimID, Position)`, with
  positions in claim-map coordinates.
- **Input routing.**
  - `craftingChordOffLargeMap` (`cmd/ultimav/gamescene_input.go:424`) lets Alt+B and
    Alt+C through when the current location is a claim map. Alt+G there answers
    "Not here!", and other small maps keep today's refusal.
  - The small-map input handler gains the `BuildDirectionInput` state and reads
    `pendingBuildPieceID`, mirroring `gamescene_input_largemap.go:334` and
    `largeMapCraftingDirection`. Factor out a shared helper rather than duplicating it.
  - The Escape cancel in `gamescene_surface_direction_cancel.go:15-17` must also cover
    the small-map state.
  - The home chest Open intercept (`gamescene_crafting.go:307`, today hooked at
    `gamescene_input_largemap.go:338`) moves to the small-map Open command. It still
    costs no karma, even though Open on a town chest normally does.
- **Menus.**
  - On the overworld, Alt+B offers only "Stake a claim here" when the square qualifies.
    On the party's own hut square it says "You stand at <name>. Enter to build."
  - Inside a claim map, Alt+B offers the four piece categories and Demolish.
  - Alt+C works on the overworld and inside claim maps. `IsCraftingSite` is split into
    the overworld check (Gather, Stake) and a check for crafting anywhere on the
    overworld or in a claim (Craft).
  - Update the command help in `cmd/ultimav/gamescene_command_help.go:72-86`.

### 6. Monsters and camping

- **Overworld spawns.** `isClaimSpawnBlocked` (`crafting.go:251`) blocks only the hut
  square. Remove the rule that stops all generation while the party stands in a claim
  (`internal/ai/npc_ai_controller_large_map.go:471`); one square no longer needs it.
  The hook installation (`legacy_save_game.go:241`, `save_snapshot.go:1116`) stays.
- **Inside the claim** there are no NPCs, because the NPC list is empty. No monsters
  are generated on small maps.
- **Camping (CRAFT-0006).** Check how Hole Up behaves on small maps
  (`action_hole_up_camping.go:613`).
  - If camping is allowed there, keep an ambush-free rest inside the party's own claim
    map.
  - If not, drop the clause, record that in `docs/gamefacts/CRAFTING.md`, and remove
    `isClaimedCampSite` (`crafting.go:261`).

### 7. Save snapshot v26, redefined

Package: `internal/save_snapshot`

- **`snapshot.go`:**
  - `LandClaimSnapshot` becomes `{id, name, x, y, terrain, tiles}`. `x`/`y` is the
    overworld square, and `tiles` holds 1024 sprite indexes in row-major order
    (`y*32 + x`).
  - The tiles are stored rather than regenerated from the seed, so a later change to
    the generator can never alter an existing claim.
  - `PlacementSnapshot` and `HomeChestSnapshot` gain `claimId`, and their `x`/`y`
    become claim-map coordinates (0..31).
  - Update the comment at `snapshot.go:84` that explains why built tiles are not
    stored there.
- **Restore order.** A save made inside a claim stores a claim `Location`. The load
  path requires `GetLocationReference(loc) != nil` (`save_snapshot.go:313-332`) and
  then rebuilds the small map. So `restoreCrafting` (called at `:932`) must
  `ClearGenerated()` and register every claim reference **before** that check. Either
  move the call earlier or split out a `registerLandClaims` pre-pass.
- **Map mutations.** `captureMapMutations` (`:816`) still diffs the overworld, which
  carries the hut marker. For the active small map it diffs against the claim
  reference, which already holds the builds, so it captures nothing new. The two
  stores never disagree.
- **`validation_v26.go`:**
  - claim IDs are unique and below `MaxLandClaims`, so each claim location is
    distinct;
  - positions lie on the world map and no two claims are on the same or orthogonally
    adjacent squares;
  - `terrain` is a known value, `tiles` has exactly 1024 entries, and each entry is
    below 256;
  - every placement and chest names an existing `claimId` and lies within 0..31;
  - the existing pairing rules stay: each chest sits on a chest placement, each chest
    placement has its chest, and `under` is below 256;
  - remove the rectangle overlap check and the width/height checks.
- **Legacy SAVED.GAM** (`legacy_save_game.go:103` takes the location byte as is). If a
  path writes or converts a native save to the legacy format, it must refuse while
  inside a claim, or write the claim's overworld square instead. Byte 0x80 means
  nothing to the original.
- **Docs:**
  - `docs/details/SAVE_SNAPSHOT_SCHEMA.md`, section "Snapshot 26 Crafting";
  - `docs/schemas/native-save-e2-s26.schema.json`: the `landClaim` definition (replace
    the `width`/`height` fixed at 16 with `terrain` and a `tiles` array with
    `minItems`/`maxItems` 1024), and `claimId` on placements and chests.

### 8. Documentation and requirements

- **`docs/gamefacts/CRAFTING.md`:** rewrite CRAFT-0004 (claims: one square, adjacency,
  hut marker, entering), CRAFT-0005 (building inside the claim map), CRAFT-0006
  (safety, per §6) and CRAFT-0007 (home chest inside the claim). Mark the change as a
  `REDUX DEVIATION`, as the existing rules are.
- **[ADR-014](ADR-014-crafting-and-housing.md):** change its Scope bullet to point here.
  Its claim is that towns are rebuilt from source and so excluded; claim maps meet
  that by making the generated reference itself the source.
- **`docs/CRAFTING_AND_HOUSING.md`:** rewrite "Claiming land" and "Building".
  Regenerate the screenshots with `TestCraftingManualCapture`
  (`cmd/ultimav/crafting_manual_capture_test.go:213`), adding the hut marker on the
  overworld, the fresh claim interior for each terrain, and a finished house.
- **`review-crafting-housing-4344c1b.md`:** add a note that its findings on 16x16
  claims and overworld building are superseded by this ADR.
- **Parent knowledge base (UltimaV-Extended):** record a decision amending D-012 in
  `docs/decisions.md`, update `docs/STATUS.md`, and commit at the verified checkpoint
  (D-008).

## Critical files

| Area | Files |
|---|---|
| Rules | `internal/game_state/crafting.go`, `action_build.go`, `action_enter.go`, `action_craft.go`, `action_hole_up_camping.go` |
| References | `internal/references/crafting_data.go`, `locations.go`, `location_references.go`, `small_location_reference.go`, new `land_claim_map.go` |
| AI | `internal/ai/npc_ai_controller_large_map.go` |
| Scene and input | `cmd/ultimav/gamescene_crafting.go`, `gamescene_input.go`, `gamescene_input_largemap.go`, the small-map input handler, `gamescene_surface_direction_cancel.go`, `gamescene_command_help.go` |
| Save | `internal/save_snapshot/snapshot.go`, `validation_v26.go`, `save_snapshot.go` (restore order) |
| Tests | `internal/game_state/crafting_test.go`, `crafting_review_test.go`, `internal/save_snapshot/validation_v26_test.go`, `cmd/ultimav/gamescene_crafting_test.go`, `crafting_manual_capture_test.go`, `internal/ai/npc_ai_controller_large_map_generation_test.go` |
| Docs | the files listed in §7 and §8 |

## Verification

1. **`go test ./...`.** Rewrite the existing crafting tests for the new model, and add
   these:
   - staking checks only the party's square, and refuses non-buildable ground, a
     settlement, a moongate and a square next to a claim;
   - staking writes the hut, and the hut survives a save round-trip through
     `MapMutations`;
   - the generator is deterministic per seed, every interior tile is buildable, the
     path from the entry to the south edge is walkable, and the outer tile matches the
     terrain;
   - Enter on the hut goes into the claim map at `{15,30}`, and walking off the edge
     returns to the hut square on the overworld;
   - a build and a demolish survive leaving and re-entering, a save made outside the
     claim and a save made inside it;
   - a home chest round-trips, and demolishing a non-empty chest is refused;
   - `ClearGenerated` stops claims from one game appearing after loading another;
   - v26 validation rejects bad `tiles` lengths and values, an unknown `claimId`,
     out-of-range claim-map coordinates and adjacent claims;
   - inside a claim map, Alt+G is refused while Alt+B and Alt+C work; inside a town,
     all three are refused;
   - no monster spawns on the hut square.
2. **Checks** from `AGENTS.md` and `CONTRIBUTING.md`: format, vet, lint and the
   documentation checker (`scripts/check-documentation`).
3. **Manual run** with `Gameplay.Crafting.Enabled`:
   - stake claims on grassland, in a forest and in desert;
   - enter, build a small house with walls, a door, a bed and a storage chest, and
     fill the chest;
   - leave, save, reload, re-enter and confirm everything persisted;
   - repeat with a save made inside the claim.
4. **Manual screenshots:** regenerate them and review the pictures.

## Consequences

- Houses are built at the same scale as every other building in the game, and a claim
  costs one overworld square instead of 256 squares of the world map.
- Claims add a new kind of location: generated, saved and outside the original
  location table. Code that assumes a location comes from DATA.OVL needs a claim
  branch (§4). This is the main risk, and tests should guard it.
- Saves grow by about 1 KB of tiles per claim, at most 4 claims.
- Saves made on this branch before the change will not load (v26 redefined).
