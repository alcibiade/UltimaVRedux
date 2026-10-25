// Integration tests for Look command workflows with real game data
// These tests validate the complete look command flow including tile description lookup,
// special case handling (clocks), and SystemCallbacks integration.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestLookCommand_SmallMap_Integration tests the look command in small maps with real game data
func TestLookCommand_SmallMap_Integration(t *testing.T) {
	// Test look command with real game data in Lord British's Castle
	gameState, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle). // Rich environment for looking
		WithPlayerAt(12, 12).                        // Center of castle
		WithSystemCallbacks().
		Build()

	t.Run("LookAtBasicTile", func(t *testing.T) {
		mockCallbacks.Reset()

		// Look in a direction (up)
		result := gameState.ActionLookSmallMap(references.Up)

		// Validate command succeeded
		assert.True(t, result, "Look command should always succeed")

		// Validate SystemCallbacks integration
		assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
		assert.Contains(t, mockCallbacks.Messages[0], "Thou dost see", "Message should contain look description")
		assert.NotEmpty(t, mockCallbacks.Messages[0], "Look description should not be empty")

		// Validate time advancement
		assert.Equal(t, 1, len(mockCallbacks.TimeAdvanced()), "Should advance time once")
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance 1 minute")
	})

	t.Run("LookAtMultipleTiles", func(t *testing.T) {
		mockCallbacks.Reset()

		// Look in all four directions to test various tiles
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gameState.ActionLookSmallMap(direction)
			assert.True(t, result, "Look command should succeed in all directions")
		}

		// Validate multiple look operations
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each look")
		assert.Equal(t, 4, len(mockCallbacks.TimeAdvanced()), "Should advance time for each look")

		// Each look should have content
		for i, message := range mockCallbacks.Messages {
			assert.Contains(t, message, "Thou dost see", "Message %d should contain look description", i)
		}
	})
}

// TestLookCommand_LargeMap_Integration tests the look command in large maps
func TestLookCommand_LargeMap_Integration(t *testing.T) {
	// Test look command on large map (Britannia)
	gameState, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Large map environment
		WithPlayerAt(100, 100).                       // Open area on large map
		WithSystemCallbacks().
		Build()

	mockCallbacks.Reset()

	// Look in a direction
	result := gameState.ActionLookLargeMap(references.Up)

	// Validate command succeeded
	assert.True(t, result, "Look command should always succeed on large map")

	// Validate SystemCallbacks integration
	assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
	assert.Contains(t, mockCallbacks.Messages[0], "Thou dost see", "Message should contain look description")

	// Validate time advancement
	assert.Equal(t, 1, len(mockCallbacks.TimeAdvanced()), "Should advance time once")
	assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance 1 minute")
}

// TestLookCommand_ErrorHandling_Integration tests edge cases and bounds
func TestLookCommand_ErrorHandling_Integration(t *testing.T) {
	// Test look command edge cases
	gameState, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(1, 1). // Near edge to test boundary conditions
		WithSystemCallbacks().
		Build()

	t.Run("LookAtEdgePositions", func(t *testing.T) {
		mockCallbacks.Reset()

		// Look in all directions from edge position
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gameState.ActionLookSmallMap(direction)
			// Look should handle edge cases gracefully
			assert.True(t, result, "Look should handle edge positions gracefully for direction %v", direction)
		}

		// Should have responses for all directions
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should handle all edge directions")
	})
}
