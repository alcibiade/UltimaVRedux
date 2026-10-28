# Review: crafting, land claims and housing (UltimaVRedux commit `4344c1b`)

**Date:** 2026-10-28
**Commit:** `4344c1b42f4354f55d12a3bcc69191bc95fed84a` on `feat/crafting-housing`
("Add optional crafting, land claims and house building on the overworld")
**Scope:** 49 files, +2559/-37. All paths below are relative to `UltimaVRedux/`.
**Labels:** [C] confirmed by reading the code and running it, [H] hypothesis not yet exercised.

## 1. Verdict

The feature is well isolated, off by default, faithful to the fork's own invariants
(gameplay in `GameState`, presentation in `GameScene`, typed tunables in
`references`, strict versioned save), and it ships with gamefacts, an ADR, a frozen
V25 schema and a reasonable unit-test base. Build, vet, the touched test packages,
the `cmd/ultimav` suite, `make gamefacts-check` and `make docs-check` all pass.

Nothing found is a crash or a save-corruption risk. The one finding I would fix
before merging is that **Craft (Alt+C) ignores the surface and vehicle gates** that
Gather and Build enforce, so the documented rule "overworld surface, on foot" is
false for one of the three commands. The remaining findings are time-cost drift
between code and gamefacts, chord-routing inconsistencies, a few design gaps
(moongate squares, item stacks, Atlas pinning), shallow save validation, and
missing tests around the menus and the save validator.

## 2. What was verified

| Check | Result |
| --- | --- |
| `go build ./...` | pass |
| `go vet` on `internal/game_state`, `internal/references`, `internal/save_snapshot`, `cmd/ultimav` | pass |
| `go test` on `internal/{game_state,references,save_snapshot,ai,party_state,config,player_save}` | pass (exit 0) |
| `go test ./cmd/ultimav/` | pass (exit 0) |
| `make gamefacts-check` | PASS, 40 topics, 273 references |
| `make docs-check` | pass on re-run; the first run exited 2 with no output and I could not reproduce it |
| `go vet -tags craftingmanual ./cmd/ultimav/` (untracked capture rig) | pass |
| Pinned linter (`bin/tools/golangci-lint` 2.12.2) | **not verifiable here**: it cannot load export data from the installed Go 1.27.3 toolchain ("export data version 5 is greater than maximum supported version 4") and reports five typecheck errors on standard-library imports. No code finding; re-run `make lint` on a machine where the pinned linter matches the toolchain. |

Not done: a manual play-through, and the full `make validate` (its `check` target runs
the entire test tree; only the touched packages were run).

## 3. Findings, most severe first

### F1 [C] Medium: Craft works in the Underworld and aboard vehicles

`ApplyCraft` (`internal/game_state/action_craft.go:296`) checks only
`IsCraftingEnabled`. Gather and Build both refuse with "Not here!" unless
`isOnOverworldSurface()` holds and the party is on foot
(`action_gather.go:339`, `action_build.go:153`). The chord router
`largeMapCraftingChord` (`cmd/ultimav/gamescene_crafting.go:27`) runs for every
large map, and the Underworld is a large map with floor -1.

Repro: crafting on, party in the Underworld (or on a horse or ship on the
overworld) with 3 Wood and 2 Stone, press Alt+C, choose Axe. The axe is crafted.
CRAFT-0001 states the three commands "work only on the overworld surface (floor 0),
on foot, never in ... the underworld ... or aboard a vehicle"; CRAFT-0003 inherits
that.

Fix: add the same guard to `ApplyCraft` (return `ErrCraftingNotOnSurface`, which
already exists in `crafting.go:24`) and, better, gate once in
`largeMapCraftingChord` so no menu opens off the surface. Add a test mirroring
`TestGatherRefusedOffTheSurface` for Craft, with the party in
`references.BritanniaUnderworld` at floor -1 and aboard a vehicle.

### F2 [C] Low: Craft and Stake cost 32 and 12 minutes, docs say 30 and 10

Gather and Build use `advanceCraftingLabor` (`crafting.go:406`), which nets exactly
`GatherMinutes`/`BuildMinutes` both when routed (the ordinary 2-minute cycle is
owned by `CompleteSurfaceCommandCycle`) and when called directly (tests).

`ApplyCraft` (`action_craft.go:318`) and `StakeLandClaim` (`action_build.go:124`)
instead call `AdvanceGameplayMinutesAndProcessProvisions` for the full 30 or 10
minutes. Both run from a menu or workflow callback on a later frame, so no routing
is active; afterwards `craftingCompletion` completes the command with
`surfaceCommandCompleted`, whose `ConsumesNormalCycle` is true
(`cmd/ultimav/gamescene_surface_completion.go:27`), and
`CompleteSurfaceCommandCycle` adds `DefaultLargeMapMinutesPerTurn` (2). Net cost is
32 and 12 minutes against the 30 and 10 in CRAFT-0003 and CRAFT-0004.

Note that `advanceCraftingLabor` is not the fix for these two: outside routing its
`AdvanceOrdinarySurfaceCommandMinutes` call is not gated, so it would add 2 more.
Either advance `minutes - DefaultLargeMapMinutesPerTurn` in the menu paths, or have
`craftingCompletion` clear `ConsumesNormalCycle` for Completed when the action has
already paid its time, or change the two gamefacts to 32 and 12. Add a minutes
assertion through the scene path (the existing `sumMinutes` assertion in
`crafting_test.go:89` covers only the direct Gather call).

### F3 [C] Low: a refused Craft still spends the ordinary turn

`openCraftMenu` maps an `ApplyCraft` error to `finish(surfaceCommandRejected)`
(`gamescene_crafting.go:117`), which consumes the normal 2-minute cycle. CRAFT-0003
says an unaffordable recipe "spends nothing". The disabled-chord path explicitly
sets `ConsumesNormalCycle = false` (`gamescene_crafting.go:35`); do the same for
the unaffordable case, or state in CRAFT-0003 that materials are kept but the turn
passes, matching Push and other original rejections.

### F4 [C] Low: Alt chords are not routed consistently across maps

- `smallMapInputHandlerWithGame` (`cmd/ultimav/gamescene_input_smallmap.go:13`)
  has no modifier check, so in a town Alt+G performs Get, Alt+C Cast and Alt+B
  Board. The help entries say "Available on the surface overworld." and the large
  map answers "What?" when the option is off. The underworld reaches the crafting
  code (see F1).
- `docs/systems/INPUT_HANDLING.md` documents the "exact chord" policy (Ctrl+N and
  Ctrl+Q require no other modifier and are release-gated). `largeMapCraftingChord`
  only checks that Alt is held, so Ctrl+Alt+G also triggers. The document was not
  updated for the three new chords.

Suggest one shared guard that answers "What?" (option off) or "Not here!" (not on
the overworld surface) before any map-specific handler, using the same exact-chord
helpers as Ctrl+N, and an INPUT_HANDLING.md paragraph for Alt+G/C/B.

### F5 [H] Low: moongate squares can be claimed and built on

`ProposeLandClaim` (`action_build.go:19`) excludes `WorldLocations` entries and
settlement sprites (`isSettlementSprite`, `action_build.go:70`). Moongates are not
locations, not map units and not tiles: `IsMoongateActiveAt`
(`moongate_system_integration.go:29`) answers from references, and
`largeMapPushOccupied` cannot see them. By day a wall or fireplace can be placed on a
moongate square inside a claim. At night the gate is unreachable, and I have not
checked what moonstone arrival does when the destination tile is impassable.

Suggest treating the eight moongate positions like settlements in
`ProposeLandClaim`, or at least refusing them in `buildTarget`, and adding a test.

### F6 [C] Low: Gather ignores item stacks

`ActionGatherLargeMap` checks `ObjectPresentAt` only (`action_gather.go:354`, NPC
and monster units), while `buildTarget` uses `largeMapPushOccupied`
(`action_push.go:224`, item stacks plus units). Felling a tree under dropped loot
turns the terrain beneath the stack to grass. CRAFT-0002 says "An object on the
square ... refuses". Use `largeMapPushOccupied` in Gather for symmetry.

### F7 [C] Low: Atlas pinning message and name repertoire

- `pinLandClaimOnAtlas` (`crafting.go:415`) returns false both when the Atlas is
  full and when a note already exists at the claim's centre; the player always sees
  "Your Atlas has no room to mark it." Either distinguish the cases or stay silent
  for the already-marked one.
- `NormalizeLandClaimName` accepts any valid UTF-8 of 1 to 24 runes and the save
  schema allows the same, but `ValidateAtlasNoteText` requires printable ASCII
  (' ' to '~'). The in-game input box enforces `^[ -~]{0,24}$`, so today this only
  matters for non-UI callers (tests, future workbench), where a non-ASCII name
  silently fails to pin. Align the three rules (simplest: make
  `NormalizeLandClaimName` reject anything outside ' '..'~').

### F8 [C] Low: save validation is shallow and has no negative tests

`validateCrafting` (`internal/save_snapshot/validation_v26.go`) checks ranges,
uniqueness and the catalog, which is the right baseline. It does not check that
claims do not overlap, that placements and chests lie inside a claim, that a home
chest has a matching storage-chest placement, or that each placement has a
`mapMutations` entry with the piece's sprite. Loading such a save is still safe
(nil-safe maps, demolition restores `under`), so this is robustness, not a bug.

There is no test that feeds invalid crafting data to the validator: the only caller
in tests is the schema-format hook in `journal_v16_test.go:48`. Add a table test
with one case per rejection path (cap overflow, duplicate claim ID, bad name, size
not 16, unknown piece, `under` out of range, duplicate placement or chest
position). Minor: `craftingMapSize` duplicates `references.XLargeMapTiles`.

### F9 [C] Low: unused results and dead code

- `BuildPieceOptions().Affordable` and `CraftRecipeOptions().Owned` are computed but
  the menus ignore them (`gamescene_crafting.go:112`, `:167`); rows show the cost
  only. AGENTS.md asks to "explain availability in text as well as colour". Marking
  unaffordable rows, or sorting them last, would use the data already there.
- `references.BuildPieceBySprite` (`crafting_data.go:207`) and
  `config.SetCraftingEnabled` (`config.go:828`) have no callers; the Workbench is
  not wired (ADR-014 says so). Keep the setter if the workbench follows soon,
  otherwise drop both.

### F10 [C] Info: behaviour once the option is switched off after use

Built tiles stay in the world as plain map mutations: walls still block, a placed
door opens through the ordinary Open, and a storage chest answers the generic
"It's open!" from `actionOpenLargeMap` (`action_open.go:133`) without giving access
to its contents. Materials and tools stay in the inventory and Ztats. Claims give
no protection. This is coherent, but CRAFTING.md's "the game behaves as the
original" is not quite true once something was built; one sentence there would
prevent bug reports.

### F11 [C] Info: permanent, world-wide deforestation

Axe harvesting depletes trees and forests anywhere on the overworld, not only
inside claims, with no regrowth, and the result is persisted as map mutations. That
also shifts monster terrain gates (`PickMonsterByTerrain`) and movement costs over
cleared areas. This is a design choice CRAFT-0002 states, so it is not a defect; if
unintended, limit depletion to claims or add regrowth on a calendar basis.

## 4. Architecture and conventions

- **Boundaries.** All rules live in `internal/game_state` and `internal/references`
  with no Ebitengine imports; `cmd/ultimav/gamescene_crafting.go` only routes keys,
  menus and the naming workflow. The `GameScene`/`GameState` relationship and the
  `SystemCallbacks` channel are respected.
- **Time and randomness.** Yields use `RandomIntInRange` and `OneInXOdds`; time
  goes through the surface-cycle helpers. The only drift is F2/F3.
- **Monster generation.** `SetSpawnBlocked` on the large-map controller is optional
  by design (nil keeps the original RNG draws, covered by
  `TestClaimBlockedLargeMapMonsterGenerationConsumesNoRandomness`). This is a
  deliberate exception to the "no nil checks for mandatory wiring" rule and
  ADR-014 records it; acceptable.
- **Camping.** The ambush roll is skipped only for a camp pitched inside an own
  claim and never for forced debug ambushes; the branch order in
  `processSingleCampingCycle` keeps the wanderer-test path first. Fine.
- **Config.** `Gameplay.Crafting.Enabled` is typed-validated, defaulted to false,
  loaded, saved and staged by the Configure dialog, which persists the whole
  `GameplayConfiguration` struct, so the new field needs no per-field apply. The
  Debug Workbench status payload omits it (documented).
- **Inventory.** `Materials` and `Tools` reuse the generic capped-stack type
  (large 9999 / small 99); `CanAfford`, `SpendMaterials` and `AddMaterials` are
  small and correct. The home-chest provision slots index by `references.Provision`
  (Torches 0 ... Gold 5), and `HomeChestSummary`'s name table matches that order.
- **Save format.** Version 26 adds a required `crafting` member; V25 is frozen in
  `migration_v26.go` and both V24 and V25 promote with it empty. The s26 schema
  (`$defs/snapshot` requires `crafting`, coordinates 0..255, `pieceId` 0..22,
  `under` 0..255, six provisions) agrees with the Go validator. Placed tiles ride
  on the existing `mapMutations`, and the capture orders placements and chests
  deterministically. Round trip is covered by `TestCraftingStateSurvivesSaveAndReload`.
- **Map layers.** Build and Demolish write the base layer and clear the override
  layer; an opened placed door lives in the override layer (`OpenDoorAt`) and the
  base layer keeps `RegularDoor`, so `IsBuildableGround` refuses building over an
  open door and Demolish restores the terrain correctly.
- **Ztats.** Material and tool rows use IDs 1000+; the dossier widget addresses
  rows by index, not ID, so no collision with original item IDs.

## 5. Test coverage

Present and useful: gather with and without tools, bare hands never depleting,
off-surface refusal (Gather only), craft spending, claim pricing/overlap/
settlement/water rules, wrap-around containment, build gating, build-over-floor and
demolish refund, wall passability, save round trip with a chest, gold deposit and
cap-limited withdrawal, camp-site and spawn guards, Alt+G chord routing, help
availability, and the no-RNG spawn guard.

Gaps worth closing:

1. Craft off the surface and aboard a vehicle (F1).
2. Scene-level tests for the Craft and Build menus, the claim-naming workflow
   (submit, empty name, Esc), `BuildDirectionInput` Esc cancellation, and the
   home-chest Open interception (`largeMapHomeChestOpen`), including the "free
   cancel" versus "used" completion states and the minutes they cost (F2, F3).
3. `TransferHomeChest` for materials, tools, food and supplies, the deposit cap,
   and "Everything" (only gold is tested).
4. Negative cases for `validateCrafting` (F8) and a V25 fixture promoted with an
   absent `crafting` member asserted at the `game_state` level (the historical
   fixture test only strips the member for comparison).
5. The camping ambush skip through `processSingleCampingCycle`, not only via
   `isClaimedCampSite`.
6. A moongate-square claim or build attempt (F5).

## 6. Documentation

CRAFTING.md is clear and matches the code except for: CRAFT-0001/0003 versus F1,
the 30/10-minute figures versus F2, "spends nothing" versus F3, and the
"behaves as the original" sentence versus F10. ADR-014 is accurate; its status is
"Proposed" on a merged fork branch, so set it to Accepted when the branch lands.
SAVE_SNAPSHOT_SCHEMA.md, the gamefacts README and catalog, and docs/README.md are
updated and pass the checks. `docs/systems/INPUT_HANDLING.md` should gain the three
Alt chords (F4).

## 7. Uncommitted work in the tree

- `cmd/ultimav/gamescene_crafting.go` (modified): moves the claim price from the
  button label into the menu description and renames the button "Stake a claim
  here". Harmless; commit it with the feature or as a follow-up.
- `cmd/ultimav/crafting_manual_capture_test.go` (untracked, build tag
  `craftingmanual`): a screenshot rig that drives a real scene through production
  input and writes PNGs to `CRAFTING_MANUAL_DIR`. It vets cleanly under its tag,
  is excluded from normal builds and from the linter (`tests: false`), and uses
  wall-clock sleeps per frame, which is fine for a manual-only tool. Consider a
  one-line mention in `docs/TESTING.md` so the tag is discoverable.

## 8. Suggested order of work

1. Gate Craft on the overworld surface and on foot; add the test (F1).
2. Decide the time model for menu commands and align code or gamefacts (F2, F3).
3. Route Alt chords uniformly and document them in INPUT_HANDLING.md (F4).
4. Exclude moongate squares from claims (F5) and item stacks from Gather (F6).
5. Tidy the Atlas message and name repertoire (F7), the validator tests (F8) and
   the unused fields (F9).
6. Re-run `make lint` and `make validate` on a machine where the pinned linter can
   read the toolchain, then commit the two uncommitted files.
