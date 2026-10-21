package map_state

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// LayeredMaps holds the layered map of every floor of the overworld, underworld and current small location.
type LayeredMaps struct {
	layeredMaps      map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap
	smallMapLocation references.Location
	doorOccupancy    DoorOccupancyProvider
}

// DoorOccupancyProvider reads authoritative actors, never cached map-unit sprites.
type DoorOccupancyProvider func(references.Location, references.Position, references.FloorNumber) bool

// SetDoorOccupancyProvider installs the runtime-only small-map terrain resolver.
// New floors inherit it; already loaded floors are rebound as well.
func (l *LayeredMaps) SetDoorOccupancyProvider(provider DoorOccupancyProvider) {
	if provider == nil {
		panic("small-map door occupancy provider is required")
	}
	l.doorOccupancy = provider
	for floor, layer := range l.layeredMaps[references.SmallMapType] {
		l.bindDoorOccupancy(layer, l.smallMapLocation, floor)
	}
}

func (l *LayeredMaps) bindDoorOccupancy(layer *LayeredMap, location references.Location, floor references.FloorNumber) {
	// Bare source maps and non-small maps have no actor authority.
	if l.doorOccupancy != nil {
		layer.doorOccupied = func(position references.Position) bool {
			return l.doorOccupancy(location, position, floor)
		}
	}
}

// LoadedMap identifies one currently materialized map floor.
type LoadedMap struct {
	MapType  references.GeneralMapType
	Location references.Location
	Floor    references.FloorNumber
	Map      *LayeredMap
}

// Loaded returns all materialized floors. Consumers must sort if stable order is required.
func (l *LayeredMaps) Loaded() []LoadedMap {
	loaded := make([]LoadedMap, 0)
	for mapType, floors := range l.layeredMaps {
		location := references.BritanniaUnderworld
		if mapType == references.SmallMapType {
			location = l.smallMapLocation
		}
		for floor, layeredMap := range floors {
			loaded = append(loaded, LoadedMap{MapType: mapType, Location: location, Floor: floor, Map: layeredMap})
		}
	}

	return loaded
}

// DiscardDepartedSmallMap ends a visit only on a different physical destination.
// Virtual combat/prison/camping scenes suspend the retained source location.
func (l *LayeredMaps) DiscardDepartedSmallMap(destination references.Location) {
	mapType := destination.GetMapType()
	if mapType == references.CombatMapType || mapType == references.CampingMapType ||
		(mapType == references.SmallMapType && destination == l.smallMapLocation) {
		return
	}
	delete(l.layeredMaps, references.SmallMapType)
	l.smallMapLocation = references.BritanniaUnderworld
}

// NewLayeredMaps builds the two large maps (overworld and underworld) from their references.
func NewLayeredMaps(
	tileRefs *references.Tiles,
	overworldRef *references.LargeMapReference,
	underworldRef *references.LargeMapReference,
	xTilesInMap int,
	yTilesInMap int,
) *LayeredMaps {
	maps := &LayeredMaps{
		layeredMaps:      make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
		smallMapLocation: 0,
		doorOccupancy:    nil,
	}
	maps.layeredMaps[references.LargeMapType] = make(map[references.FloorNumber]*LayeredMap)
	for _, floor := range []references.FloorNumber{0, -1} {
		maps.layeredMaps[references.LargeMapType][floor] = newLayeredMap(
			references.XLargeMapTiles, references.YLargeMapTiles, tileRefs, xTilesInMap, yTilesInMap, true,
		)
	}

	maps.SetInitialLargeMap(0, overworldRef)
	maps.SetInitialLargeMap(-1, underworldRef)

	return maps
}

// GetLayeredMap returns the layered map for a map type and floor, or nil when it is not loaded.
func (l *LayeredMaps) GetLayeredMap(
	generalMapType references.GeneralMapType,
	floorNumber references.FloorNumber,
) *LayeredMap {
	return l.layeredMaps[generalMapType][floorNumber]
}

// SetInitialLargeMap fills the base layer of a large map floor from its reference data.
func (l *LayeredMaps) SetInitialLargeMap(nFloor references.FloorNumber, lrg *references.LargeMapReference) {
	largeMap := l.layeredMaps[references.LargeMapType][nFloor]
	for tileX := references.Coordinate(0); tileX < references.XLargeMapTiles; tileX++ {
		for tileY := references.Coordinate(0); tileY < references.YLargeMapTiles; tileY++ {
			largeMap.SetTileByLayer(
				MapLayer, &references.Position{X: tileX, Y: tileY}, lrg.GetSpriteIndex(tileX, tileY),
			)
		}
	}
}

// ResetAndCreateSmallMap replaces the loaded small map with every floor of slr.
func (l *LayeredMaps) ResetAndCreateSmallMap(
	slr *references.SmallLocationReference,
	tileRefs *references.Tiles,
	xTilesInMap int,
	yTilesInMap int,
) {
	l.smallMapLocation = slr.Location
	l.layeredMaps[references.SmallMapType] = make(map[references.FloorNumber]*LayeredMap)
	for _, floor := range slr.ListOfFloors {
		l.ReloadSmallMapFloor(slr, tileRefs, floor, xTilesInMap, yTilesInMap)
	}
}

// ReloadSmallMapFloor rebuilds one source floor while retaining other floors.
func (l *LayeredMaps) ReloadSmallMapFloor(
	slr *references.SmallLocationReference,
	tileRefs *references.Tiles,
	floor references.FloorNumber,
	xTilesInMap int,
	yTilesInMap int,
) {
	l.smallMapLocation = slr.Location
	if l.layeredMaps[references.SmallMapType] == nil {
		l.layeredMaps[references.SmallMapType] = make(map[references.FloorNumber]*LayeredMap)
	}
	theFloor := newLayeredMap(
		references.XSmallMapTiles,
		references.YSmallMapTiles,
		tileRefs,
		xTilesInMap,
		yTilesInMap,
		false,
	)
	l.layeredMaps[references.SmallMapType][floor] = theFloor
	l.bindDoorOccupancy(theFloor, slr.Location, floor)
	boundaries := make([]bool, int(references.XSmallMapTiles*references.YSmallMapTiles))
	for tileX := references.Coordinate(0); tileX < references.XSmallMapTiles; tileX++ {
		for tileY := references.Coordinate(0); tileY < references.YSmallMapTiles; tileY++ {
			pos := references.Position{X: tileX, Y: tileY}
			// Use immutable source doors even when runtime layers later open or replace them.
			sprite := slr.GetTileNumber(pos, floor)
			theFloor.SetTileByLayer(MapLayer, &pos, sprite)
			boundaries[int(tileY*references.XSmallMapTiles+tileX)] = isExposureBoundary(sprite)
		}
	}
	theFloor.exposure = ClassifyMapExposure(int(references.XSmallMapTiles), int(references.YSmallMapTiles),
		boundaries, smallMapOutdoorSeeds(slr.Location, floor))
	theFloor.daylightOpenings = collectDaylightOpenings(slr, floor, theFloor.exposure)
}

func smallMapOutdoorSeeds(location references.Location, floor references.FloorNumber) []image.Point {
	switch {
	case floor == 0 && (location == references.Yew || location == references.Minoc):
		// Their ground-floor exteriors are enclosed by source doors and cannot
		// otherwise reach the map border used by structural exposure flood fill.
		//nolint:mnd // Verified source-map coordinate inside each outdoor component.
		return []image.Point{{X: 15, Y: 15}}
	case location == references.LordBritishsCastle && floor == 1:
		// Explicit Redux structural policy: the three enclosed upper wall walks are
		// outdoor. Immutable source components contain 57, 51, and 51 cells respectively.
		// This does not infer roofs from floor height or seed the enclosed central hall.
		//nolint:mnd // Verified source-map coordinates for the three explicitly outdoor wall walks.
		return []image.Point{{X: 4, Y: 12}, {X: 26, Y: 12}, {X: 15, Y: 4}}
	default:
		return nil
	}
}

// ResetAndCreateDungeonMap creates the retained top-down compatibility
// projection for semantic dungeon cells. Gameplay must never read this
// projection as dungeon authority.
func (l *LayeredMaps) ResetAndCreateDungeonMap(
	cells references.DungeonCells,
	tileRefs *references.Tiles,
	xTilesInMap int,
	yTilesInMap int,
) {
	l.layeredMaps[references.DungeonMapType] = make(map[references.FloorNumber]*LayeredMap)
	for floor := references.FloorNumber(0); floor < references.DungeonFloorCount; floor++ {
		l.layeredMaps[references.DungeonMapType][floor] = newLayeredMap(
			references.XDungeonMapTiles,
			references.YDungeonMapTiles,
			tileRefs,
			xTilesInMap,
			yTilesInMap,
			true,
		)
		l.RebuildDungeonFloor(cells, floor)
	}
}

// RebuildDungeonFloor replaces one derived projection floor from semantic
// authority. Rebuilding 64 cells keeps synchronization simple and explicit.
func (l *LayeredMaps) RebuildDungeonFloor(cells references.DungeonCells, floor references.FloorNumber) {
	theFloor := l.GetLayeredMap(references.DungeonMapType, floor)
	if theFloor == nil {
		panic("dungeon projection floor is not materialized")
	}
	for y := references.Coordinate(0); y < references.YDungeonMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XDungeonMapTiles; x++ {
			position := references.Position{X: x, Y: y}
			cell, ok := cells.Cell(floor, position)
			if !ok {
				panic("semantic dungeon cell is out of bounds")
			}
			theFloor.SetTileByLayer(MapLayer, &position, dungeonProjectionSprite(cell))
		}
	}
}

func dungeonProjectionSprite(cell references.DungeonCell) indexes.SpriteIndex {
	const (
		dungeonPoisonFieldPayload   = 1
		dungeonFireFieldPayload     = 2
		dungeonElectricFieldPayload = 3
	)
	if cell.Kind() == references.DungeonEnergyField {
		switch cell.Payload() & 0x07 {
		case dungeonPoisonFieldPayload:
			return indexes.PoisonField
		case dungeonFireFieldPayload:
			return indexes.FireField
		case dungeonElectricFieldPayload:
			return indexes.ElectricField
		default:
			return indexes.MagicField
		}
	}

	projections := [...]indexes.SpriteIndex{
		references.DungeonPassage:      indexes.BrickFloor,
		references.DungeonLadderUp:     indexes.LadderUp,
		references.DungeonLadderDown:   indexes.LadderDown,
		references.DungeonLadderBoth:   indexes.KlimbLadderUp,
		references.DungeonClosedChest:  indexes.Chest,
		references.DungeonFountain:     indexes.Fountain1,
		references.DungeonPitOrTrap:    indexes.BrickFloorHole,
		references.DungeonOpenChest:    indexes.Chest,
		references.DungeonEnergyField:  indexes.MagicField,
		references.DungeonReserved:     indexes.ScaryBlackThing1,
		references.DungeonClearedRoom:  indexes.BrickFloor,
		references.DungeonWall:         indexes.StoneBrickWall,
		references.DungeonSpecialWall:  indexes.LargeRockWall,
		references.DungeonHiddenDoor:   indexes.StoneBrickWallSecret,
		references.DungeonRevealedDoor: indexes.BrickWallArchway,
		references.DungeonLiveRoom:     indexes.BrickFloor,
	}

	return projections[cell.Kind()]
}

// ResetAndCreateCombatMap creates a LayeredMap for a combat map (camping, combat encounters)
// Combat maps are always 11x11 tiles, single floor (floor 0), non-wrapping
func (l *LayeredMaps) ResetAndCreateCombatMap(
	combatMap *references.CombatMapReference,
	tileRefs *references.Tiles,
	xTilesInMap int,
	yTilesInMap int,
) {
	// Hard failure if combatMap is nil - this is embedded game data that should always work
	if combatMap == nil {
		panic("ResetAndCreateCombatMap: combatMap is nil - system integrity compromised")
	}

	// Initialize CombatMapType with floor 0 only
	l.layeredMaps[references.CombatMapType] = make(map[references.FloorNumber]*LayeredMap)

	// Create LayeredMap for floor 0 (combat maps are single-floor)
	// Combat maps: 11x11, non-wrapping
	l.layeredMaps[references.CombatMapType][0] = newLayeredMap(
		references.XCombatMapTiles,
		references.YCombatMapTiles,
		tileRefs,
		xTilesInMap,
		yTilesInMap,
		false, // bWrappingMap = false for combat maps
	)

	theFloor := l.layeredMaps[references.CombatMapType][0]

	// Populate the MapLayer with tiles from CombatMapReference
	for x := references.Coordinate(0); x < references.XCombatMapTiles; x++ {
		for y := references.Coordinate(0); y < references.YCombatMapTiles; y++ {
			pos := references.Position{X: x, Y: y}
			tileIndex := combatMap.GetTileNumber(pos)
			theFloor.SetTileByLayer(MapLayer, &pos, tileIndex)
		}
	}
}

// ResetAndCreateMiniMapFromData creates a LayeredMap for an 11x11 mini-map (shrine, codex, etc.)
// Uses raw tile data directly to avoid package dependencies
// tileData is [x][y] indexed, where (0,0) is top-left
func (l *LayeredMaps) ResetAndCreateMiniMapFromData(
	tileData *[references.XCombatMapTiles][references.YCombatMapTiles]byte,
	tileRefs *references.Tiles,
	xTilesInMap int,
	yTilesInMap int,
) {
	if tileData == nil {
		panic("ResetAndCreateMiniMapFromData: tileData is nil - system integrity compromised")
	}

	// Initialize CombatMapType with floor 0 only (reuse combat map type for 11x11 mini-maps)
	l.layeredMaps[references.CombatMapType] = make(map[references.FloorNumber]*LayeredMap)

	// Create LayeredMap for floor 0 (mini-maps are single-floor)
	l.layeredMaps[references.CombatMapType][0] = newLayeredMap(
		references.XCombatMapTiles,
		references.YCombatMapTiles,
		tileRefs,
		xTilesInMap,
		yTilesInMap,
		false, // bWrappingMap = false for mini-maps
	)

	theFloor := l.layeredMaps[references.CombatMapType][0]

	// Populate the MapLayer with tiles from raw data
	for x := references.Coordinate(0); x < references.XCombatMapTiles; x++ {
		for y := references.Coordinate(0); y < references.YCombatMapTiles; y++ {
			pos := references.Position{X: x, Y: y}
			tileIndex := indexes.SpriteIndex(tileData[x][y])
			theFloor.SetTileByLayer(MapLayer, &pos, tileIndex)
		}
	}
}

// GetTileRefByPosition returns the tile at pos in one layer of the given map and floor.
func (l *LayeredMaps) GetTileRefByPosition(
	mapType references.GeneralMapType,
	mapLayer LayerType,
	pos *references.Position,
	nFloor references.FloorNumber,
) *references.Tile {
	// Use the LayeredMap's GetTileByLayer method which handles coordinate transformation properly
	return l.layeredMaps[mapType][nFloor].GetTileByLayer(mapLayer, pos)
}

// GetTileTopMapOnlyTileByPosition returns the top terrain tile at pos, ignoring the unit layer.
func (l *LayeredMaps) GetTileTopMapOnlyTileByPosition(
	mapType references.GeneralMapType,
	pos *references.Position,
	nFloor references.FloorNumber,
) *references.Tile {
	return l.GetLayeredMap(mapType, nFloor).GetTileTopMapOnlyTile(pos)
}

// GetSmallMapFloorKlimbOffset returns the floor change from climbing the stairs at position:
// 0 when there are none, -1 when they lead down, +1 when they lead up.
func (l *LayeredMaps) GetSmallMapFloorKlimbOffset(position references.Position, nFloor references.FloorNumber) int {
	tile := l.GetTileTopMapOnlyTileByPosition(references.SmallMapType, &position, nFloor)
	if !tile.Index.IsStairs() {
		return 0
	}

	if l.hasLowerFloor(nFloor) {
		lowerTile := l.GetTileTopMapOnlyTileByPosition(references.SmallMapType, &position, nFloor-1)
		if lowerTile.Index.IsStairs() {
			return -1
		}
	}

	return 1
}

func (l *LayeredMaps) hasLowerFloor(currentFloor references.FloorNumber) bool {
	_, exists := l.layeredMaps[references.SmallMapType][currentFloor-1]

	return exists
}
