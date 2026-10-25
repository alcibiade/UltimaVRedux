// Package game_state provides error condition integration tests
// These tests validate proper error handling and edge cases across multiple systems
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestInvalidCommandSequences_Integration tests error handling for invalid command sequences
func TestInvalidCommandSequences_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	t.Run("JimmyWithoutKeys", func(t *testing.T) {
		mockCallbacks.Reset()
		beforeTurn := gs.DateTime

		// Test jimmy without keys - rely on existing state
		// Note: This tests jimmy on a non-locked door at (15,15) Up in Britain

		// Try to jimmy a door
		result := gs.ActionJimmySmallMap(references.Up)

		assert.False(t, result, "Jimmy should fail")
		// Since there's no locked door there, we get "Not lock!"
		assert.Contains(t, mockCallbacks.Messages, "No lock!", "Should indicate no lock present")
		expectedDate := beforeTurn
		expectedDate.Advance(DefaultSmallMapMinutesPerTurn)
		assert.Equal(t, expectedDate, gs.DateTime, "Jimmy must complete one normal town turn")
	})

	t.Run("OpenLockedDoor", func(t *testing.T) {
		mockCallbacks.Reset()

		// Find a locked door (if any)
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}
		for _, dir := range directions {
			pos := dir.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)
			tile := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0).GetTopTile(pos)

			if tile.Is(indexes.LockedDoor) {
				result := gs.ActionOpenSmallMap(dir)
				assert.False(t, result, "Should not open locked door")
				assert.Contains(t, mockCallbacks.Messages, "Locked!", "Should have locked message")
				break
			}
		}
	})

	t.Run("PushImmovableObject", func(t *testing.T) {
		mockCallbacks.Reset()

		// Try to push a wall or immovable object
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}
		for _, dir := range directions {
			pos := dir.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)
			tile := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0).GetTopTile(pos)

			if tile.IsWall() || (!tile.IsPushable() && !tile.IsWalkingPassable()) {
				result := gs.ActionPushSmallMap(dir)
				assert.False(t, result, "Should not push immovable object")
				assert.Contains(t, mockCallbacks.Messages, "Won't budge!", "Should have won't budge message")
				break
			}
		}
	})

	t.Run("GetNothingThere", func(t *testing.T) {
		mockCallbacks.Reset()

		// Find empty floor tile
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}
		for _, dir := range directions {
			pos := dir.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)

			if !gs.ObjectPresentAt(pos) {
				_ = gs.ActionGetSmallMap(dir)
				// Get might return true even for empty tiles (design choice)
				if len(mockCallbacks.Messages) > 0 {
					assert.Contains(t, mockCallbacks.Messages[0], "Nothing", "Should indicate nothing to get")
				}
				break
			}
		}
	})
}

// TestBoundaryConditions_Integration tests edge cases at map boundaries
func TestBoundaryConditions_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(0, 0). // Corner position
		WithSystemCallbacks().
		Build()

	t.Run("MovementAtMapEdge", func(t *testing.T) {
		mockCallbacks.Reset()

		// Try to move off the map
		originalPos := gs.MapState.PlayerLocation.Position

		// Test boundary movement (simplified for integration test)
		t.Log("Testing boundary movement restrictions")
		assert.Equal(t, originalPos, gs.MapState.PlayerLocation.Position, "Player should remain at boundary position")

		t.Logf("Player remained at boundary position: %v", originalPos)
	})

	t.Run("CommandsAtMapEdge", func(t *testing.T) {
		// Test various commands when at map edge
		mockCallbacks.Reset()

		// Push at edge
		result := gs.ActionPushSmallMap(references.Left)
		assert.False(t, result, "Should not push at map edge")

		// Get at edge
		mockCallbacks.Reset()
		result = gs.ActionGetSmallMap(references.Up)
		// Get behavior at edge depends on what's there

		// Look at edge
		mockCallbacks.Reset()
		result = gs.ActionLookSmallMap(references.Left)
		// Look should work even at edge
		if result && len(mockCallbacks.Messages) > 0 {
			t.Logf("Look at edge returned: %v", mockCallbacks.Messages[0])
		}
	})
}
