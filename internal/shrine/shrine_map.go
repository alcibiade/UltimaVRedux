package shrine

import (
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

const (
	MiniMapShrine MiniMapType = iota // Shrine meditation scene (MISCMAPS.DAT index 1)
	MiniMapCodex                     // Codex reading scene (MISCMAPS.DAT index 2)
	// Future: MiniMapFinalScene, etc. for indices 0 and 3
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

// ShrineMap represents the shrine/codex interior layout
// Used for both shrine meditation and codex reading scenes
type ShrineMap struct {
	// Center position (altar for shrine, codex stand for codex)
	CenterPosition references.Position

	// Entrance position (where player enters)
	EntrancePosition references.Position

	// Map tile data (if loaded from MISCMAPS.DAT)
	MapData *MiniMapData
}

// NewShrineMapWithData creates a new shrine map with loaded data
func NewShrineMapWithData(mapData *MiniMapData) *ShrineMap {
	shrineMap := &ShrineMap{
		CenterPosition:   references.Position{X: CenterPositionX, Y: CenterPositionY},
		EntrancePosition: references.Position{X: EntrancePositionX, Y: EntrancePositionY},
		MapData:          mapData,
	}

	return shrineMap
}

// GetTile returns the tile index at the given position
// Returns 0 (grass/default) if position is invalid or no map data loaded
func (sm *ShrineMap) GetTile(x, y int) byte {
	if !sm.IsValidPosition(x, y) {
		return 0
	}

	if sm.MapData == nil {
		// No map data loaded - return default tiles
		return sm.getDefaultTile(x, y)
	}

	return sm.MapData.Tiles[x][y]
}

// getDefaultTile returns a default tile for positions when no map data is loaded
// This creates a simple layout for testing
func (sm *ShrineMap) getDefaultTile(x, y int) byte {
	// Center (altar/codex)
	if x == CenterPositionX && y == CenterPositionY {
		return 0x19 // Altar tile (placeholder - will use correct index)
	}

	// Entrance path (center column from bottom to center)
	if x == EntrancePositionX && y > CenterPositionY {
		return 0x02 // Stone floor
	}

	// Simple floor around center
	dx := abs(x - CenterPositionX)
	dy := abs(y - CenterPositionY)

	if dx <= 2 && dy <= 2 {
		return 0x02 // Stone floor
	}

	// Walls/edges
	if x == 0 || x == MiniMapSize-1 || y == 0 || y == MiniMapSize-1 {
		return 0x07 // Wall tile
	}

	return 0x02 // Default floor
}

// IsValidPosition checks if the given position is within the mini-map bounds
func (sm *ShrineMap) IsValidPosition(x, y int) bool {
	return x >= 0 && x < MiniMapSize && y >= 0 && y < MiniMapSize
}

// GetCenterPosition returns the center position (altar for shrine, codex stand for codex)
func (sm *ShrineMap) GetCenterPosition() references.Position {
	return sm.CenterPosition
}

// GetEntrancePosition returns the entrance position
func (sm *ShrineMap) GetEntrancePosition() references.Position {
	return sm.EntrancePosition
}

// HasMapData returns true if map data was loaded from MISCMAPS.DAT
func (sm *ShrineMap) HasMapData() bool {
	return sm.MapData != nil
}

// IsCodexMap returns true if this is the Codex map (not a shrine)
func (sm *ShrineMap) IsCodexMap() bool {
	if sm.MapData == nil {
		return false
	}

	return sm.MapData.MapType == MiniMapCodex
}

// GetMapType returns the type of mini-map (shrine, codex, etc.)
func (sm *ShrineMap) GetMapType() MiniMapType {
	if sm.MapData == nil {
		return MiniMapShrine // Default to shrine
	}

	return sm.MapData.MapType
}

// miniMapIndices maps MiniMapType to MISCMAPS.DAT indices
var miniMapIndices = map[MiniMapType]int{
	MiniMapShrine: 1, // Shrine map at index 1 (offset 176)
	MiniMapCodex:  2, // Codex map at index 2 (offset 352)
	// Indices 0 and 3 are Blackthorn's prison and the endgame rescue map;
	// neither is a ShrineMap gameplay type.
}

// LoadMiniMapFromMiscmaps loads a mini-map (shrine, codex, etc.) from MISCMAPS.DAT
func LoadMiniMapFromMiscmaps(gameConfig *config.UltimaVConfiguration, mapType MiniMapType) (*MiniMapData, error) {
	mapIndex, ok := miniMapIndices[mapType]
	if !ok {
		return nil, fmt.Errorf("unknown mini-map type: %d", mapType)
	}
	decoded, err := minimap.LoadMiscMap(gameConfig.SavedConfigData.DataFilePath, mapIndex)
	if err != nil {
		return nil, fmt.Errorf("loading shrine mini-map: %w", err)
	}

	mapData := &MiniMapData{
		MapType: mapType,
		Tiles:   decoded.Tiles,
	}
	logging.Debug(logging.CategorySystem, "Loaded mini-map type %d from MISCMAPS.DAT index %d", mapType, mapIndex)

	return mapData, nil
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
