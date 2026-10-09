package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransitionToCampingMap_CreatesLayeredMap verifies camping transition creates LayeredMap
func TestTransitionToCampingMap_CreatesLayeredMap(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()

	// Store pre-camping location
	preCampingLocation := gs.MapState.PlayerLocation.Location

	// Transition to camping map
	err := gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	// Verify LayeredMap was created for CombatMapType floor 0
	campingLayeredMap := gs.MapState.LayeredMaps.GetLayeredMap(references.CombatMapType, 0)
	require.NotNil(t, campingLayeredMap, "Combat LayeredMap should be created for camping")

	// Verify dimensions are correct (11x11 for camping map)
	assert.Equal(t, references.XCombatMapTiles, campingLayeredMap.XMaxTilesPerMap, "X dimension should be 11")
	assert.Equal(t, references.YCombatMapTiles, campingLayeredMap.YMaxTilesPerMap, "Y dimension should be 11")

	// Verify location changed to CampingLocation
	assert.Equal(t, references.CampingLocation, gs.MapState.PlayerLocation.Location,
		"Player location should be CampingLocation")

	// Verify pre-camping location was stored
	assert.Equal(t, preCampingLocation, gs.PreCampingLocation,
		"Pre-camping location should be stored")
}

// TestTransitionToCampingMap_TilesAccessible verifies tiles can be accessed via LayeredMap
func TestTransitionToCampingMap_TilesAccessible(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()

	// Load camping map reference for validation
	campingMapRef, err := references.LoadCampingMap(gs.GameConfig)
	require.NoError(t, err, "Camping map should load successfully")

	// Transition to camping map
	err = gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	campingLayeredMap := gs.MapState.LayeredMaps.GetLayeredMap(references.CombatMapType, 0)
	require.NotNil(t, campingLayeredMap, "Combat LayeredMap should exist")

	// Verify we can access tiles at known positions
	campfirePos := campingMapRef.GetCampfirePosition()
	tile := campingLayeredMap.GetTopTile(&campfirePos)
	require.NotNil(t, tile, "Should be able to get tile at campfire position")

	// Verify tile data matches what CombatMapReference has
	expectedIndex := campingMapRef.GetTileNumber(campfirePos)
	assert.Equal(t, expectedIndex, tile.Index,
		"LayeredMap tile should match CombatMapReference")
}

// TestTransitionFromCampingMap_RestoresLocation verifies returning from camping restores location
func TestTransitionFromCampingMap_RestoresLocation(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()

	preCampingLocation := gs.MapState.PlayerLocation.Location

	// Transition to camping
	err := gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	assert.Equal(t, references.CampingLocation, gs.MapState.PlayerLocation.Location,
		"Should be in camping location")

	// Transition back
	gs.TransitionFromCampingMap()

	// Verify location restored
	assert.Equal(t, preCampingLocation, gs.MapState.PlayerLocation.Location,
		"Location should be restored")

	// Verify no longer in camping
	assert.False(t, gs.IsInCamping(), "IsInCamping() should return false after leaving camping")
}

// TestCampingMap_VisibilityRecalculation verifies visibility system works with camping maps
func TestCampingMap_VisibilityRecalculation(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(87, 108). // Use realistic world coordinates
		WithSystemCallbacks().
		Build()

	// Transition to camping
	err := gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	// Verify player position was set to camping map center (CRITICAL!)
	assert.Equal(t, references.Coordinate(5), gs.MapState.PlayerLocation.Position.X,
		"Player X should be centered in camping map")
	assert.Equal(t, references.Coordinate(5), gs.MapState.PlayerLocation.Position.Y,
		"Player Y should be centered in camping map")

	// Get the combat LayeredMap
	campingLayeredMap := gs.MapState.LayeredMaps.GetLayeredMap(references.CombatMapType, 0)
	require.NotNil(t, campingLayeredMap, "Combat LayeredMap should exist")

	// Use the ACTUAL player position from GameState (this is what the game does)
	playerPos := gs.MapState.PlayerLocation.Position

	// Test visibility recalculation (this is where the panic occurs in the game)
	// This would panic if player position wasn't reset to camping map coordinates
	require.NotPanics(t, func() {
		campingLayeredMap.RecalculateVisibleTilesWithCustomLights(
			playerPos,
			&gs.MapState.Lighting,
			nil,                       // No moongates in camping
			nil,                       // No quest items in camping
			nil,                       // No NPCs in camping
			map_state.PlayerLightNone, // Camping is always fully lit
		)
	}, "Visibility recalculation should not panic for camping maps")

	// Verify we can check visibility after recalculation
	isVisible := campingLayeredMap.IsPositionVisible(&playerPos, gs.DateTime, false)
	assert.True(t, isVisible, "Player position should be visible")
}

// TestCampingMap_IsOutOfBounds verifies bounds checking works for camping maps
func TestCampingMap_IsOutOfBounds(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()

	// Transition to camping
	err := gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	// Verify we're in camping mode
	assert.True(t, gs.IsInCamping(), "Should be in camping mode")

	// Test valid positions (camping map is 11x11, so valid range is 0-10)
	validPositions := []references.Position{
		{X: 0, Y: 0},   // Corner
		{X: 10, Y: 10}, // Opposite corner
		{X: 5, Y: 5},   // Center (campfire)
		{X: 0, Y: 10},  // Edge
		{X: 10, Y: 0},  // Edge
	}

	for _, pos := range validPositions {
		assert.False(t, gs.IsOutOfBounds(pos),
			"Position (%d,%d) should be in bounds for camping map", pos.X, pos.Y)
	}

	// Test out-of-bounds positions
	outOfBoundsPositions := []references.Position{
		{X: -1, Y: 5},  // Negative X
		{X: 5, Y: -1},  // Negative Y
		{X: 11, Y: 5},  // X too large
		{X: 5, Y: 11},  // Y too large
		{X: 11, Y: 11}, // Both too large
		{X: -1, Y: -1}, // Both negative
	}

	for _, pos := range outOfBoundsPositions {
		assert.True(t, gs.IsOutOfBounds(pos),
			"Position (%d,%d) should be out of bounds for camping map", pos.X, pos.Y)
	}
}

// TestCampingMap_GetCurrentSmallLocationReference verifies nil handling for combat maps
// This test catches the panic that occurs in refreshStaticMapTiles() when rendering
// out-of-bounds positions on camping maps. The rendering code tries to get an "outer tile"
// for positions beyond the 11×11 camping map, but GetCurrentSmallLocationReference()
// returns nil for CampingLocation (0xFE) since it's not a real SmallLocation.
func TestCampingMap_GetCurrentSmallLocationReference(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()

	// Before camping: GetCurrentSmallLocationReference should work
	britainRef := gs.GetCurrentSmallLocationReference()
	require.NotNil(t, britainRef, "Britain should have a SmallLocationReference")

	// Transition to camping
	err := gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	// After camping: GetCurrentSmallLocationReference returns nil
	// This is expected because CampingLocation is a CombatMapType, not a SmallMapType
	campingRef := gs.GetCurrentSmallLocationReference()
	assert.Nil(t, campingRef,
		"CampingLocation should return nil from GetCurrentSmallLocationReference (it's a CombatMapType)")

	// Verify location type is correct
	assert.Equal(t, references.CombatMapType, gs.MapState.PlayerLocation.Location.GetMapType(),
		"CampingLocation should be CombatMapType")
}

// TestCampingMap_GetOuterTileForCurrentLocation verifies outer tile handling for different map types
func TestCampingMap_GetOuterTileForCurrentLocation(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()

	// Before camping: Should return town-specific outer tile
	outerTileBefore := gs.GetOuterTileForCurrentLocation()
	assert.NotEqual(t, indexes.NoSprites, outerTileBefore,
		"Britain should have a non-void outer tile")

	// Transition to camping
	err := gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	// After camping: Should return void/NoSprites for combat maps
	outerTileAfter := gs.GetOuterTileForCurrentLocation()
	assert.Equal(t, indexes.NoSprites, outerTileAfter,
		"CampingLocation should return NoSprites outer tile (combat maps have no borders)")

	// Verify location type is correct
	assert.Equal(t, references.CombatMapType, gs.MapState.PlayerLocation.Location.GetMapType(),
		"CampingLocation should be CombatMapType")
}

// TestCampingMap_GetEffectiveTileForOutOfBounds verifies tile rendering for out-of-bounds positions
// This is the comprehensive integration test that catches the refreshStaticMapTiles() panic.
// When rendering the 19×13 viewport centered on the 11×11 camping map, positions beyond
// the camping map bounds (like -4,-1 to 14,12) will be considered out-of-bounds.
// The rendering code needs to handle this gracefully rather than panicking.
func TestCampingMap_GetEffectiveTileForOutOfBounds(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(87, 108). // Realistic world coordinates
		WithSystemCallbacks().
		Build()

	// Transition to camping
	err := gs.TransitionToCampingMap()
	require.NoError(t, err, "Camping transition should succeed")

	// Player should now be at camping map center (5, 5)
	assert.Equal(t, references.Coordinate(5), gs.MapState.PlayerLocation.Position.X)
	assert.Equal(t, references.Coordinate(5), gs.MapState.PlayerLocation.Position.Y)

	// Test out-of-bounds positions that would be rendered in the viewport
	// The viewport is 19×13 tiles centered on player (5,5)
	// This means rendering positions from (-4, -1) to (14, 12) approximately
	// Positions beyond the 11×11 camping map bounds will be out-of-bounds
	outOfBoundsPositions := []references.Position{
		{X: -1, Y: 5},  // Left edge of viewport, beyond camping map
		{X: 11, Y: 5},  // Right edge of viewport, beyond camping map
		{X: 5, Y: -1},  // Top edge of viewport, beyond camping map
		{X: 5, Y: 11},  // Bottom edge of viewport, beyond camping map
		{X: -4, Y: -1}, // Viewport corner, way out of bounds
		{X: 14, Y: 12}, // Viewport corner, way out of bounds
	}

	for _, pos := range outOfBoundsPositions {
		// Verify position is indeed out of bounds
		require.True(t, gs.IsOutOfBounds(pos),
			"Position (%d,%d) should be out of bounds", pos.X, pos.Y)

		// Call GetEffectiveTile for out-of-bounds position
		// This should NOT panic - it needs to return a valid tile (probably a void/black tile)
		require.NotPanics(t, func() {
			tile := gs.GetEffectiveTile(&pos)
			// Should return some kind of tile, not panic
			assert.NotEqual(t, indexes.SpriteIndex(0), tile,
				"Out-of-bounds position (%d,%d) should return non-zero tile", pos.X, pos.Y)
		}, "GetEffectiveTile should not panic for out-of-bounds positions on camping maps")
	}

	// Also test in-bounds positions still work correctly
	inBoundsPositions := []references.Position{
		{X: 0, Y: 0},   // Corner of camping map
		{X: 5, Y: 5},   // Center (player/campfire)
		{X: 10, Y: 10}, // Opposite corner
	}

	for _, pos := range inBoundsPositions {
		require.False(t, gs.IsOutOfBounds(pos),
			"Position (%d,%d) should be in bounds", pos.X, pos.Y)

		require.NotPanics(t, func() {
			tile := gs.GetEffectiveTile(&pos)
			assert.NotEqual(t, indexes.SpriteIndex(0), tile,
				"In-bounds position (%d,%d) should return valid tile", pos.X, pos.Y)
		}, "GetEffectiveTile should work for in-bounds positions")
	}
}
