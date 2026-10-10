package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func (g *GameState) UpdateSmallMap(tileRefs *references.Tiles, locs *references.LocationReferences) {
	chests := g.basementChestLifetimes()
	slr := locs.GetLocationReference(g.MapState.PlayerLocation.Location)
	previous, wasSmallMap := g.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	preserveAbsent := wasSmallMap && previous.Location() == slr.Location
	g.resetAndCreateSmallMap(slr, tileRefs)
	g.CurrentNPCAIController = ai.NewNPCAIControllerSmallMap(slr, tileRefs, &g.MapState, &g.DateTime, g,
		basementChestPopulationChecker{state: g, preserveAbsent: preserveAbsent})
	g.CurrentNPCAIController.PopulateMapFirstLoad()
	g.restoreBasementChestLifetimes(chests)

	// Initialize lighthouse rotation system for current floor
	// Scans only current floor - will be rescanned on movement/floor changes
	g.UpdateLighthouseForSmallMap()
}

func (g *GameState) resetAndCreateSmallMap(slr *references.SmallLocationReference, tileRefs *references.Tiles) {
	g.rebuildSmallMapTerrain(slr, tileRefs, nil)
}

// reloadCurrentSmallMapTerrain reproduces load_level(FALSE): source terrain is
// rebuilt while the current NPC controller and its transient AI state survive.
func (g *GameState) reloadCurrentSmallMapTerrain() {
	slr := g.GameReferences.LocationReferences.GetLocationReference(g.MapState.PlayerLocation.Location)
	g.rebuildSmallMapTerrain(slr, g.GameReferences.TileReferences, &g.MapState.PlayerLocation.Floor)
	g.UpdateLighthouseForSmallMap()
}

func (g *GameState) rebuildSmallMapTerrain(slr *references.SmallLocationReference, tileRefs *references.Tiles, onlyFloor *references.FloorNumber) {
	g.MapState.LayeredMaps.SetDoorOccupancyProvider(func(location references.Location, position references.Position, floor references.FloorNumber) bool {
		return g.MapState.PlayerLocation.Location == location && g.isSmallMapDoorOccupied(position, floor)
	})
	g.MapState.CloseOpenDoor()
	// REDUX DEVIATION: terrain reconstruction reloads source locks on floor and
	// combat return. Retain latest door overrides for this visit to prevent soft locks.
	// The existing native map mutations continue the visit across save/load.
	// See ref:CMD-0006.
	type doorOverride struct {
		floor    references.FloorNumber
		position references.Position
		sprite   indexes.SpriteIndex
	}
	var doors []doorOverride
	for _, loaded := range g.MapState.LayeredMaps.Loaded() {
		if loaded.MapType != references.SmallMapType || loaded.Location != slr.Location ||
			(onlyFloor != nil && loaded.Floor != *onlyFloor) {
			continue
		}
		for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
			for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
				position := references.Position{X: x, Y: y}
				sprite := loaded.Map.SpriteIndexByLayer(map_state.MapOverrideLayer, position)
				switch sprite { //nolint:exhaustive // Only these closed-door overrides survive a visit rebuild.
				case indexes.RegularDoor, indexes.RegularDoorView, indexes.MagicLockDoor, indexes.MagicLockDoorWithView:
					doors = append(doors, doorOverride{floor: loaded.Floor, position: position, sprite: sprite})
				}
			}
		}
	}
	if onlyFloor == nil {
		g.MapState.LayeredMaps.ResetAndCreateSmallMap(slr, tileRefs,
			g.MapState.XTilesVisibleOnGameScreen, g.MapState.YTilesVisibleOnGameScreen)
	} else {
		g.MapState.LayeredMaps.ReloadSmallMapFloor(slr, tileRefs, *onlyFloor,
			g.MapState.XTilesVisibleOnGameScreen, g.MapState.YTilesVisibleOnGameScreen)
	}
	for _, door := range doors {
		g.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, door.floor).
			SetTileByLayer(map_state.MapOverrideLayer, &door.position, door.sprite)
	}
	g.applyLordBritishPassageState()
}
