// Package shrine provides the shrine meditation and codex reading mini-map scenes.
package shrine

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/minimap"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// MiniMapSize is the size of mini-maps (11x11 like original Ultima V)
// Used for shrine, codex, and other cutscene maps from MISCMAPS.DAT
const MiniMapSize = 11

// MiniMapType identifies the type of mini-map being loaded
type MiniMapType int

// Mini-map scenes loaded from MISCMAPS.DAT; each value selects one scene.
const (
	MiniMapShrine MiniMapType = iota // Shrine meditation scene (MISCMAPS.DAT index 1)
	MiniMapCodex                     // Codex reading scene (MISCMAPS.DAT index 2)
	// Future: MiniMapFinalScene, etc. for indices 0 and 3
)

// MISCMAPS.DAT indices of the shrine and codex scenes.
const (
	shrineMiscMapIndex = 1
	codexMiscMapIndex  = 2
)

// Mini-map position constants (shared by shrine and codex)
const (
	CenterPositionX   = 5  // Center X of 11x11 map (altar/codex location)
	CenterPositionY   = 5  // Center Y of 11x11 map (camera center)
	EntrancePositionX = 5  // Entrance X (center of bottom edge)
	EntrancePositionY = 10 // Entrance Y (bottom of map)
)

// MiniMapData holds the raw tile data loaded from MISCMAPS.DAT
type MiniMapData struct {
	// Raw tile indices [x][y]
	Tiles [MiniMapSize][MiniMapSize]byte

	// Type of mini-map (shrine, codex, etc.)
	MapType MiniMapType
}

// Map represents the shrine/codex interior layout
// Used for both shrine meditation and codex reading scenes
type Map struct {
	// Center position (altar for shrine, codex stand for codex)
	CenterPosition references.Position

	// Entrance position (where player enters)
	EntrancePosition references.Position

	// Map tile data (if loaded from MISCMAPS.DAT)
	MapData *MiniMapData
}

// NewShrineMapWithData creates a new shrine map with loaded data
func NewShrineMapWithData(mapData *MiniMapData) *Map {
	shrineMap := &Map{
		CenterPosition:   references.Position{X: CenterPositionX, Y: CenterPositionY},
		EntrancePosition: references.Position{X: EntrancePositionX, Y: EntrancePositionY},
		MapData:          mapData,
	}

	return shrineMap
}

// GetTile returns the tile index at the given position
// Returns 0 (grass/default) if position is invalid or no map data loaded
func (sm *Map) GetTile(tileX, tileY int) byte {
	if !sm.IsValidPosition(tileX, tileY) {
		return 0
	}

	if sm.MapData == nil {
		// No map data loaded - return default tiles
		return sm.getDefaultTile(tileX, tileY)
	}

	return sm.MapData.Tiles[tileX][tileY]
}

// Placeholder default tile indices used when no MISCMAPS.DAT data is loaded.
const (
	defaultAltarTile      = 0x19
	defaultStoneFloorTile = 0x02
	defaultWallTile       = 0x07
	defaultFloorRadius    = 2
)

// getDefaultTile returns a default tile for positions when no map data is loaded
// This creates a simple layout for testing
func (sm *Map) getDefaultTile(tileX, tileY int) byte {
	// Center (altar/codex)
	if tileX == CenterPositionX && tileY == CenterPositionY {
		return defaultAltarTile
	}

	// Entrance path (center column from bottom to center)
	if tileX == EntrancePositionX && tileY > CenterPositionY {
		return defaultStoneFloorTile
	}

	// Simple floor around center
	dx := abs(tileX - CenterPositionX)
	dy := abs(tileY - CenterPositionY)

	if dx <= defaultFloorRadius && dy <= defaultFloorRadius {
		return defaultStoneFloorTile
	}

	// Walls/edges
	if tileX == 0 || tileX == MiniMapSize-1 || tileY == 0 || tileY == MiniMapSize-1 {
		return defaultWallTile
	}

	return defaultStoneFloorTile
}

// IsValidPosition checks if the given position is within the mini-map bounds
func (sm *Map) IsValidPosition(x, y int) bool {
	return x >= 0 && x < MiniMapSize && y >= 0 && y < MiniMapSize
}

// miniMapIndex maps MiniMapType to its MISCMAPS.DAT index.
// Indices 0 and 3 are Blackthorn's prison and the endgame rescue map;
// neither is a shrine Map gameplay type.
func miniMapIndex(mapType MiniMapType) (int, bool) {
	switch mapType {
	case MiniMapShrine:
		return shrineMiscMapIndex, true // offset 176
	case MiniMapCodex:
		return codexMiscMapIndex, true // offset 352
	default:
		return 0, false
	}
}

// errUnknownMiniMapType reports a MiniMapType with no MISCMAPS.DAT scene.
var errUnknownMiniMapType = errors.New("unknown mini-map type")

// LoadMiniMapFromMiscmaps loads a mini-map (shrine, codex, etc.) from MISCMAPS.DAT
func LoadMiniMapFromMiscmaps(gameConfig *config.UltimaVConfiguration, mapType MiniMapType) (*MiniMapData, error) {
	mapIndex, ok := miniMapIndex(mapType)
	if !ok {
		return nil, fmt.Errorf("%w: %d", errUnknownMiniMapType, mapType)
	}
	decoded, err := minimap.LoadMiscMap(gameConfig.SavedConfigData.DataFilePath, mapIndex)
	if err != nil {
		return nil, fmt.Errorf("loading shrine mini-map: %w", err)
	}

	mapData := &MiniMapData{
		MapType: mapType,
		Tiles:   decoded.Tiles,
	}
	logging.Debugf(logging.CategorySystem, "Loaded mini-map type %d from MISCMAPS.DAT index %d", mapType, mapIndex)

	return mapData, nil
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
