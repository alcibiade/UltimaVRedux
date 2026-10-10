package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCanAutoOpenDoor(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		Build()

	if gameState == nil {
		t.Skip("Skipping integration test - real game data not available")
		return
	}

	// Test position - get a nearby position for testing
	testPos := &references.Position{X: 16, Y: 15}

	// Test that regular doors can be auto-opened
	t.Run("RegularDoor", func(t *testing.T) {
		// Manually set a regular door at the test position
		smallMap := gameState.MapState.LayeredMaps.GetLayeredMap(
			references.SmallMapType, gameState.MapState.PlayerLocation.Floor)
		smallMap.SetTileByLayer(0, testPos, indexes.RegularDoor)

		assert.True(t, gameState.CanAutoOpenDoor(testPos), "Regular door should be auto-openable")
	})

	t.Run("RegularDoorView", func(t *testing.T) {
		// Manually set a regular door with view at the test position
		smallMap := gameState.MapState.LayeredMaps.GetLayeredMap(
			references.SmallMapType, gameState.MapState.PlayerLocation.Floor)
		smallMap.SetTileByLayer(0, testPos, indexes.RegularDoorView)

		assert.True(t, gameState.CanAutoOpenDoor(testPos), "Regular door with view should be auto-openable")
	})

	// Test that locked doors cannot be auto-opened
	t.Run("LockedDoor", func(t *testing.T) {
		smallMap := gameState.MapState.LayeredMaps.GetLayeredMap(
			references.SmallMapType, gameState.MapState.PlayerLocation.Floor)
		smallMap.SetTileByLayer(0, testPos, indexes.LockedDoor)

		assert.False(t, gameState.CanAutoOpenDoor(testPos), "Locked door should not be auto-openable")
	})

	t.Run("MagicLockDoor", func(t *testing.T) {
		smallMap := gameState.MapState.LayeredMaps.GetLayeredMap(
			references.SmallMapType, gameState.MapState.PlayerLocation.Floor)
		smallMap.SetTileByLayer(0, testPos, indexes.MagicLockDoor)

		assert.False(t, gameState.CanAutoOpenDoor(testPos), "Magic lock door should not be auto-openable")
	})

	// Test that non-doors cannot be auto-opened
	t.Run("NonDoorTile", func(t *testing.T) {
		smallMap := gameState.MapState.LayeredMaps.GetLayeredMap(
			references.SmallMapType, gameState.MapState.PlayerLocation.Floor)
		smallMap.SetTileByLayer(0, testPos, indexes.BrickFloor)

		assert.False(t, gameState.CanAutoOpenDoor(testPos), "Floor tile should not be auto-openable")
	})
}

func TestCanAutoOpenDoor_LargeMap(t *testing.T) {
	// Test that auto-open doesn't work on large maps
	gameState, _ := NewIntegrationTestBuilder(t).
		Build()

	if gameState == nil {
		t.Skip("Skipping integration test - real game data not available")
		return
	}

	// Force large map location
	gameState.MapState.PlayerLocation.Location = references.BritanniaUnderworld

	testPos := &references.Position{X: 100, Y: 100}

	// Even if there was a door on the large map, it shouldn't auto-open
	assert.False(t, gameState.CanAutoOpenDoor(testPos), "Auto-open should not work on large maps")
}
