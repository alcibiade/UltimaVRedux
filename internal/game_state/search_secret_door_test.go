package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

// TestSecretDoorDiscovery_SmallMapBehavior tests secret door behavior in small maps
func TestSecretDoorDiscovery_SmallMapBehavior(t *testing.T) {
	t.Run("SmallMap_SecretDoor_BecomesLockedDoor", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle). // Small map
			WithPlayerAt(10, 10).
			WithSystemCallbacks().
			Build()

		// Place secret door adjacent to player (north direction)
		secretDoorPos := &references.Position{X: 10, Y: 9}
		currentMap := gs.GetLayeredMapByCurrentLocation()
		currentMap.SetTileByLayer(map_state.MapLayer, secretDoorPos, indexes.StoneBrickWallSecret)

		// Search north to find the secret door
		result := gs.ActionSearchSmallMap(references.Up)

		// Verify success and transformation
		assert.True(t, result, "Search should succeed")
		transformedTile := currentMap.GetTopTile(secretDoorPos)
		assert.Equal(t, indexes.LockedDoor, int(transformedTile.Index), "Secret door should become locked door in small map")

		assert.Equal(t, "Thou dost find\na hidden door!", mockCallbacks.Messages[0], "Should display correct message")
		assert.Equal(t, SoundUnlock, mockCallbacks.SoundEffectsPlayed[0], "Should play unlock sound")
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance time by 1 minute")
	})
}

// TestSecretDoorDiscovery_LargeMapBehavior tests secret door behavior in large maps
func TestSecretDoorDiscovery_LargeMapBehavior(t *testing.T) {
	t.Run("LargeMap_SecretDoor_BecomesRegularDoor", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld). // Large map
			WithPlayerAt(100, 100).
			WithSystemCallbacks().
			Build()

		// Place secret door adjacent to player (east direction)
		secretDoorPos := &references.Position{X: 101, Y: 100}
		currentMap := gs.GetLayeredMapByCurrentLocation()
		currentMap.SetTileByLayer(map_state.MapLayer, secretDoorPos, indexes.StoneBrickWallSecret)

		// Search east to find the secret door
		result := gs.ActionSearchLargeMap(references.Right)

		// Verify success and transformation for large map
		assert.True(t, result, "Search should succeed")
		transformedTile := currentMap.GetTopTile(secretDoorPos)
		assert.Equal(t, indexes.RegularDoor, int(transformedTile.Index), "Secret door should become regular door in large map")

		assert.Equal(t, "Thou dost find\na hidden door!", mockCallbacks.Messages[0], "Should display correct message")
		assert.Equal(t, SoundUnlock, mockCallbacks.SoundEffectsPlayed[0], "Should play unlock sound")
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance time by 1 minute")
	})
}

// TestSecretDoorDiscovery_NonSecretWalls tests searching walls that are not secret doors
func TestSecretDoorDiscovery_NonSecretWalls(t *testing.T) {
	t.Run("RegularWall_NoSecretDoor_Found", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(10, 10).
			WithSystemCallbacks().
			Build()

		// Place regular wall (not secret door) adjacent to player
		wallPos := &references.Position{X: 10, Y: 9}
		currentMap := gs.GetLayeredMapByCurrentLocation()
		currentMap.SetTileByLayer(map_state.MapLayer, wallPos, indexes.StoneBrickWall) // Regular wall, not secret

		// Search north - should find nothing special
		result := gs.ActionSearchSmallMap(references.Up)

		// Verify no secret door was found
		assert.False(t, result, "Search should fail for regular wall")

		// Verify wall was not transformed
		unchangedTile := currentMap.GetTopTile(wallPos)
		assert.Equal(t, indexes.StoneBrickWall, int(unchangedTile.Index), "Regular wall should remain unchanged")

		assert.Equal(t, "In the wall, thou dost find nothing of note.", mockCallbacks.Messages[0], "Should show wall search message")

		// Verify no unlock sound was played
		assert.Len(t, mockCallbacks.SoundEffectsPlayed, 0, "Should not play sound for regular wall")

		// Verify time was still advanced (search takes time even if unsuccessful for secret doors)
		assert.Len(t, mockCallbacks.TimeAdvanced(), 1, "Should advance time for search attempt")
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance time by 1 minute")
	})
}

// TestSecretDoorDiscovery_AlreadyFound tests searching a position where secret door was already discovered
func TestSecretDoorDiscovery_AlreadyFound(t *testing.T) {
	t.Run("AlreadyTransformed_Door_NoChange", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(10, 10).
			WithSystemCallbacks().
			Build()

		// Place already-transformed door (locked door) adjacent to player
		doorPos := &references.Position{X: 10, Y: 9}
		currentMap := gs.GetLayeredMapByCurrentLocation()
		currentMap.SetTileByLayer(map_state.MapLayer, doorPos, indexes.LockedDoor) // Already a door

		// Search north - should not find secret door since it's already a door
		result := gs.ActionSearchSmallMap(references.Up)

		// Verify door remains unchanged
		unchangedTile := currentMap.GetTopTile(doorPos)
		assert.Equal(t, indexes.LockedDoor, int(unchangedTile.Index), "Already-transformed door should remain unchanged")

		// The behavior here depends on how the search system handles doors -
		// it might find items near the door or just say nothing found
		// This documents the expected behavior
		t.Logf("Search result for already-transformed door: result=%t, message='%s'", result, mockCallbacks.Messages[0])
	})
}

// TestSecretDoorDiscovery_EdgeCases tests edge cases and error conditions
func TestSecretDoorDiscovery_EdgeCases(t *testing.T) {
	t.Run("InvalidDirection_HandledGracefully", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(10, 10).
			WithSystemCallbacks().
			Build()

		// Test with invalid direction (this tests the general search system robustness)
		result := gs.ActionSearchSmallMap(references.Direction(-1)) // Invalid direction

		// Should handle gracefully without crashing
		assert.False(t, result, "Invalid direction should not succeed")

		// Should provide some kind of feedback
		assert.Len(t, mockCallbacks.Messages, 1, "Should provide feedback message")
		t.Logf("Invalid direction feedback: %s", mockCallbacks.Messages[0])
	})

	t.Run("OutOfBounds_Position_HandledGracefully", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(0, 0). // Corner position
			WithSystemCallbacks().
			Build()

		// Search in direction that would go out of bounds
		result := gs.ActionSearchSmallMap(references.Up) // Would go to Y=-1

		// Should handle gracefully
		assert.False(t, result, "Out of bounds search should not succeed")

		// Should provide feedback
		assert.Len(t, mockCallbacks.Messages, 1, "Should provide feedback message")
		t.Logf("Out of bounds feedback: %s", mockCallbacks.Messages[0])
	})
}
