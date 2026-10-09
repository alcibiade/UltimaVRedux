package map_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResetAndCreateCombatMap_BasicCreation verifies that a combat map LayeredMap can be created from CombatMapReference
func TestResetAndCreateCombatMap_BasicCreation(t *testing.T) {
	// Load real game data
	cfg := loadTestConfig(t)
	gameRefs := loadTestGameReferences(t, cfg)

	// Load the camping combat map (BRIT.CBT map 0)
	combatMap, err := references.LoadCampingMap(cfg)
	require.NoError(t, err, "Combat map should load successfully")
	require.NotNil(t, combatMap, "Combat map should be loaded")

	// Create LayeredMaps container
	layeredMaps := &LayeredMaps{
		layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
	}

	// Create combat map LayeredMap
	layeredMaps.ResetAndCreateCombatMap(combatMap, gameRefs.TileReferences, 19, 13)

	// Verify LayeredMap was created
	combatLayeredMap := layeredMaps.GetLayeredMap(references.CombatMapType, 0)
	require.NotNil(t, combatLayeredMap, "Combat LayeredMap should be created")

	// Verify dimensions are correct (11x11)
	assert.Equal(t, references.XCombatMapTiles, combatLayeredMap.XMaxTilesPerMap, "X dimension should be 11")
	assert.Equal(t, references.YCombatMapTiles, combatLayeredMap.YMaxTilesPerMap, "Y dimension should be 11")
}

// TestResetAndCreateCombatMap_TileDataCorrect verifies tiles are correctly transferred from CombatMapReference
func TestResetAndCreateCombatMap_TileDataCorrect(t *testing.T) {
	// Load real game data
	cfg := loadTestConfig(t)
	gameRefs := loadTestGameReferences(t, cfg)

	// Load the camping combat map
	combatMap, err := references.LoadCampingMap(cfg)
	require.NoError(t, err, "Combat map should load successfully")
	require.NotNil(t, combatMap, "Combat map should be loaded")

	// Create LayeredMaps container
	layeredMaps := &LayeredMaps{
		layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
	}

	// Create combat map LayeredMap
	layeredMaps.ResetAndCreateCombatMap(combatMap, gameRefs.TileReferences, 19, 13)

	combatLayeredMap := layeredMaps.GetLayeredMap(references.CombatMapType, 0)
	require.NotNil(t, combatLayeredMap, "Combat LayeredMap should be created")

	// Verify tile data at known positions
	// Check campfire position (should be at center of map: 5,5)
	campfirePos := combatMap.GetCampfirePosition()
	tile := combatLayeredMap.GetTopTile(&campfirePos)
	require.NotNil(t, tile, "Campfire tile should exist")

	// Verify the tile matches what CombatMapReference has
	expectedTileIndex := combatMap.GetTileNumber(campfirePos)
	assert.Equal(t, expectedTileIndex, tile.Index, "Campfire tile should match CombatMapReference")

	// Check corner positions
	positions := []references.Position{
		{X: 0, Y: 0},   // Top-left corner
		{X: 10, Y: 0},  // Top-right corner
		{X: 0, Y: 10},  // Bottom-left corner
		{X: 10, Y: 10}, // Bottom-right corner
	}

	for _, pos := range positions {
		tile := combatLayeredMap.GetTopTile(&pos)
		require.NotNil(t, tile, "Tile at (%d,%d) should exist", pos.X, pos.Y)

		expectedIndex := combatMap.GetTileNumber(pos)
		assert.Equal(t, expectedIndex, tile.Index,
			"Tile at (%d,%d) should match CombatMapReference", pos.X, pos.Y)
	}
}

// TestResetAndCreateCombatMap_NonWrapping verifies combat maps don't wrap at edges
func TestResetAndCreateCombatMap_NonWrapping(t *testing.T) {
	// Load real game data
	cfg := loadTestConfig(t)
	gameRefs := loadTestGameReferences(t, cfg)

	// Load the camping combat map
	combatMap, err := references.LoadCampingMap(cfg)
	require.NoError(t, err, "Combat map should load successfully")
	require.NotNil(t, combatMap, "Combat map should be loaded")

	// Create LayeredMaps container
	layeredMaps := &LayeredMaps{
		layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
	}

	// Create combat map LayeredMap
	layeredMaps.ResetAndCreateCombatMap(combatMap, gameRefs.TileReferences, 19, 13)

	combatLayeredMap := layeredMaps.GetLayeredMap(references.CombatMapType, 0)
	require.NotNil(t, combatLayeredMap, "Combat LayeredMap should be created")

	// Verify non-wrapping behavior
	// Accessing out of bounds should return nil or edge tile, not wrap around
	outOfBoundsPos := references.Position{X: 11, Y: 11}
	tile := combatLayeredMap.GetTopTile(&outOfBoundsPos)
	// For non-wrapping maps, out of bounds typically returns nil or stays at edge
	// This verifies it doesn't wrap back to (0,0)
	if tile != nil {
		// If tile is returned, it should be the edge tile, not wrapped position
		edgePos := references.Position{X: 10, Y: 10}
		edgeTile := combatLayeredMap.GetTopTile(&edgePos)
		assert.Equal(t, edgeTile.Index, tile.Index,
			"Out of bounds access should return edge tile or nil, not wrapped tile")
	}
}

// TestResetAndCreateCombatMap_Floor0Only verifies combat maps only support floor 0
func TestResetAndCreateCombatMap_Floor0Only(t *testing.T) {
	// Load real game data
	cfg := loadTestConfig(t)
	gameRefs := loadTestGameReferences(t, cfg)

	// Load the camping combat map
	combatMap, err := references.LoadCampingMap(cfg)
	require.NoError(t, err, "Combat map should load successfully")
	require.NotNil(t, combatMap, "Combat map should be loaded")

	// Create LayeredMaps container
	layeredMaps := &LayeredMaps{
		layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
	}

	// Create combat map LayeredMap
	layeredMaps.ResetAndCreateCombatMap(combatMap, gameRefs.TileReferences, 19, 13)

	// Verify floor 0 exists
	floor0 := layeredMaps.GetLayeredMap(references.CombatMapType, 0)
	assert.NotNil(t, floor0, "Floor 0 should exist for combat maps")

	// Verify other floors don't exist
	floor1 := layeredMaps.GetLayeredMap(references.CombatMapType, 1)
	assert.Nil(t, floor1, "Floor 1 should not exist for combat maps")

	floorNeg1 := layeredMaps.GetLayeredMap(references.CombatMapType, -1)
	assert.Nil(t, floorNeg1, "Floor -1 should not exist for combat maps")
}

// TestResetAndCreateCombatMap_NilCombatMapPanics verifies hard failure when nil combatMap is passed
func TestResetAndCreateCombatMap_NilCombatMapPanics(t *testing.T) {
	// Load real game data for TileReferences
	cfg := loadTestConfig(t)
	gameRefs := loadTestGameReferences(t, cfg)

	// Create LayeredMaps container
	layeredMaps := &LayeredMaps{
		layeredMaps: make(map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap),
	}

	// Verify that passing nil combatMap causes a panic (hard failure)
	assert.Panics(t, func() {
		layeredMaps.ResetAndCreateCombatMap(nil, gameRefs.TileReferences, 19, 13)
	}, "ResetAndCreateCombatMap should panic when combatMap is nil")
}

// Helper functions for loading test data

func loadTestConfig(t *testing.T) *config.UltimaVConfiguration {
	t.Helper()

	return helpers.LoadTestConfiguration(t)
}

func loadTestGameReferences(t *testing.T, cfg *config.UltimaVConfiguration) *references.GameReferences {
	t.Helper()

	return helpers.LoadTestGameReferences(t, cfg)
}
