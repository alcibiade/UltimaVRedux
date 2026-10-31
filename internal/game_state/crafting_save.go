package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func (g *GameState) captureCrafting(snapshot *save_snapshot.SaveSnapshot) {
	inventory := &g.PartyState.Inventory
	crafting := save_snapshot.CraftingSnapshot{Deeds: make([]save_snapshot.DeedSnapshot, 0, len(g.crafting.deeds))}
	for _, material := range references.MaterialCatalog() {
		crafting.Materials[material] = inventory.Materials.Get(material)
	}
	for _, tool := range references.ToolCatalog() {
		crafting.Tools[tool] = inventory.Tools.Get(tool)
	}
	for _, deed := range g.crafting.deeds {
		saved := save_snapshot.DeedSnapshot{
			ID: deed.ID, Name: deed.Name, X: int(deed.Square.X), Y: int(deed.Square.Y), Terrain: deed.Terrain.Key(),
			GeneratorVersion: deed.GeneratorVersion, Seed: deed.Seed,
			TileOverrides:       make([]save_snapshot.DeedTileOverrideSnapshot, 0, len(deed.overrides)),
			NodeHarvests:        make([]save_snapshot.DeedNodeHarvestSnapshot, 0, len(deed.harvests)),
			TrammelPhaseChanges: deed.trammelPhaseChanges,
		}
		for _, position := range deed.TileOverridePositions() {
			saved.TileOverrides = append(saved.TileOverrides, save_snapshot.DeedTileOverrideSnapshot{
				X: int(position.X), Y: int(position.Y), Sprite: int(deed.overrides[position]),
			})
		}
		for _, position := range deed.HarvestedNodePositions() {
			saved.NodeHarvests = append(saved.NodeHarvests, save_snapshot.DeedNodeHarvestSnapshot{
				X: int(position.X), Y: int(position.Y), Harvests: deed.harvests[position],
			})
		}
		crafting.Deeds = append(crafting.Deeds, saved)
	}
	if g.MapState.PlayerLocation.Location.IsDeedLocation() {
		if g.crafting.active == nil {
			panic("capturing a deed-map save with no active deed")
		}
		id := g.crafting.active.ID
		crafting.ActiveDeedID = &id
	}
	snapshot.Crafting = crafting
}

// restoreCrafting rebuilds stock and deeds from a validated snapshot, and loads the active deed
// map. It must run before the current small map is rebuilt, because a save made on a deed map
// resolves its location through the active deed (ref:CRAFT-0004).
func (g *GameState) restoreCrafting(saved save_snapshot.CraftingSnapshot) {
	inventory := &g.PartyState.Inventory
	for _, material := range references.MaterialCatalog() {
		inventory.Materials.Set(material, saved.Materials[material])
	}
	for _, tool := range references.ToolCatalog() {
		inventory.Tools.Set(tool, saved.Tools[tool])
	}
	g.crafting = restoreDeedState(saved)
}

// restoreDeedState rebuilds the deeds of a validated snapshot and loads its active deed map.
func restoreDeedState(saved save_snapshot.CraftingSnapshot) craftingState {
	var state craftingState
	for _, savedDeed := range saved.Deeds {
		terrain, ok := references.DeedTerrainFromKey(savedDeed.Terrain)
		if !ok {
			panic(fmt.Sprintf("validated deed %d has unknown terrain %q", savedDeed.ID, savedDeed.Terrain))
		}
		deed := &Deed{
			ID: savedDeed.ID, Name: savedDeed.Name, Terrain: terrain,
			Square:           references.Position{X: references.Coordinate(savedDeed.X), Y: references.Coordinate(savedDeed.Y)},
			GeneratorVersion: savedDeed.GeneratorVersion, Seed: savedDeed.Seed,
			overrides:           make(map[references.Position]indexes.SpriteIndex, len(savedDeed.TileOverrides)),
			trammelPhaseChanges: savedDeed.TrammelPhaseChanges,
		}
		if len(savedDeed.NodeHarvests) > 0 {
			deed.harvests = make(map[references.Position]int, len(savedDeed.NodeHarvests))
		}
		for _, harvest := range savedDeed.NodeHarvests {
			position := references.Position{X: references.Coordinate(harvest.X), Y: references.Coordinate(harvest.Y)}
			deed.harvests[position] = harvest.Harvests
		}
		for _, override := range savedDeed.TileOverrides {
			position := references.Position{X: references.Coordinate(override.X), Y: references.Coordinate(override.Y)}
			deed.overrides[position] = indexes.SpriteIndex(override.Sprite)
		}
		state.deeds = append(state.deeds, deed)
	}
	if saved.ActiveDeedID != nil {
		deed := state.deedByID(*saved.ActiveDeedID)
		if deed == nil {
			panic("validated active deed is missing")
		}
		state.activate(deed)
	}

	return state
}

// deedLocationReferenceFromSnapshot rebuilds the active deed map of a snapshot without a live
// game, for compatibility checks that must read its source tiles. It is nil off a deed map.
func deedLocationReferenceFromSnapshot(snapshot save_snapshot.SaveSnapshot) *references.SmallLocationReference {
	state := restoreDeedState(snapshot.Crafting)

	return state.activeMap
}
