// Integration tests for Use command workflows with real game data
// These tests validate the use command flow including stub behavior for all map types,
// special item usage mechanics, context validation, and SystemCallbacks integration.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestUseCommand_SmallMap_Integration tests use command in small maps (towns, buildings)
func TestUseCommand_SmallMap_Integration(t *testing.T) {
	// Test use command in Lord British's Castle (small map environment)
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(12, 12).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	t.Run("BasicUseBehavior", func(t *testing.T) {
		mockCallbacks.Reset()

		// Test use in a direction - current implementation is stub
		result := gs.ActionUseSmallMap(references.Up)

		// Current implementation always returns false (special items not implemented)
		assert.False(t, result, "Use currently returns false (special items not implemented)")

		// Validate SystemCallbacks integration - now uses ShowActionFeedback (tracked in Messages)
		assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
		assert.Equal(t, "Nothing to use!", mockCallbacks.Messages[0], "Should show 'Nothing to use!' feedback")
	})

	t.Run("UseAllDirections", func(t *testing.T) {
		mockCallbacks.Reset()

		// Use in all directions to test consistency
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gs.ActionUseSmallMap(direction)
			assert.False(t, result, "Use should return false for direction %v", direction)
		}

		// Validate all uses were processed consistently
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each use")

		// All messages should be consistent
		for i, message := range mockCallbacks.Messages {
			assert.Equal(t, "Nothing to use!", message, "Message %d should be consistent", i)
		}
	})

	t.Run("UseAtDifferentPositions", func(t *testing.T) {
		// Test use behavior at various positions in the castle
		testPositions := []references.Position{
			{X: 5, Y: 5}, {X: 10, Y: 10}, {X: 15, Y: 15}, {X: 8, Y: 12},
		}

		originalPosition := gs.MapState.PlayerLocation.Position

		for _, pos := range testPositions {
			gs.MapState.PlayerLocation.Position = pos
			mockCallbacks.Reset()

			result := gs.ActionUseSmallMap(references.Up)

			// Current behavior should be consistent regardless of position
			assert.False(t, result, "Use should return false at position (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have message at position (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, "Nothing to use!", mockCallbacks.Messages[0], "Should have consistent message at position (%d,%d)", pos.X, pos.Y)
		}

		// Restore original position
		gs.MapState.PlayerLocation.Position = originalPosition
	})
}

// TestUseCommand_LargeMap_Integration tests use command on large maps (overworld)
func TestUseCommand_LargeMap_Integration(t *testing.T) {
	// Test use command on large map (Britannia Underworld)
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	mockCallbacks.Reset()

	// Test use on large map
	result := gs.ActionUseLargeMap(references.Up)

	// Current implementation returns false (special items not implemented)
	assert.False(t, result, "Large map use currently returns false (special items not implemented)")

	// Validate SystemCallbacks integration
	assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
	assert.Equal(t, "Nothing to use!", mockCallbacks.Messages[0], "Should show 'Nothing to use!' message for large map")

	t.Run("LargeMapAllDirections", func(t *testing.T) {
		mockCallbacks.Reset()

		// Test all directions on large map
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gs.ActionUseLargeMap(direction)
			assert.False(t, result, "Large map use should return false for direction %v", direction)
		}

		// Validate consistency
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each use attempt")

		for _, message := range mockCallbacks.Messages {
			assert.Equal(t, "Nothing to use!", message, "All large map use messages should be consistent")
		}
	})
}

// TestUseCommand_CombatMap_Integration tests use command in combat mode
func TestUseCommand_CombatMap_Integration(t *testing.T) {
	// Test use command in combat context
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle). // Use castle as proxy for combat testing
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	mockCallbacks.Reset()

	// Test combat map use
	result := gs.ActionUseCombatMap(references.Up)

	// Not allowed during combat
	assert.False(t, result, "Combat map use returns false (not allowed during combat)")

	// Validate SystemCallbacks integration
	assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
	assert.Equal(t, "Not now!", mockCallbacks.Messages[0], "Should show 'Not now!' message for combat map")

	t.Run("CombatUseDirections", func(t *testing.T) {
		mockCallbacks.Reset()

		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gs.ActionUseCombatMap(direction)
			assert.False(t, result, "Combat use should return false for direction %v", direction)
		}

		// Validate combat-specific messaging
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each combat use attempt")
		for _, message := range mockCallbacks.Messages {
			assert.Equal(t, "Not now!", message, "All combat use messages should indicate blocked")
		}
	})
}

// TestUseCommand_DungeonMap_Integration tests use command in dungeons
func TestUseCommand_DungeonMap_Integration(t *testing.T) {
	// Test use command in dungeon context
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle). // Use castle as proxy for dungeon testing
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	mockCallbacks.Reset()

	// Test dungeon map use
	result := gs.ActionUseDungeonMap(references.Up)

	// Dungeon special items not implemented yet
	assert.False(t, result, "Dungeon map use returns false (special items not implemented)")

	// Validate SystemCallbacks integration (tracked in Messages since alsoAddToOutput=true)
	assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
	assert.Equal(t, "Nothing to use!", mockCallbacks.Messages[0], "Should show 'Nothing to use!' feedback for dungeon map")

	t.Run("DungeonUseConsistency", func(t *testing.T) {
		mockCallbacks.Reset()

		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gs.ActionUseDungeonMap(direction)
			assert.False(t, result, "Dungeon use should return false for direction %v", direction)
		}

		// Validate dungeon-specific behavior (tracked in Messages since alsoAddToOutput=true)
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each dungeon use")

		for _, message := range mockCallbacks.Messages {
			assert.Equal(t, "Nothing to use!", message, "All dungeon use messages should be consistent")
		}
	})
}

// TestUseCommand_SystemCallbacks_Integration tests comprehensive callback behavior
func TestUseCommand_SystemCallbacks_Integration(t *testing.T) {
	// Test SystemCallbacks consistency across different map types
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	testCases := []struct {
		name           string
		useFunc        func(references.Direction) bool
		expectFeedback string
		expectResult   bool
	}{
		{
			name:           "SmallMapUse",
			useFunc:        gs.ActionUseSmallMap,
			expectFeedback: "Nothing to use!",
			expectResult:   false,
		},
		{
			name:           "LargeMapUse",
			useFunc:        gs.ActionUseLargeMap,
			expectFeedback: "Nothing to use!",
			expectResult:   false,
		},
		{
			name:           "CombatMapUse",
			useFunc:        gs.ActionUseCombatMap,
			expectFeedback: "Not now!",
			expectResult:   false,
		},
		{
			name:           "DungeonMapUse",
			useFunc:        gs.ActionUseDungeonMap,
			expectFeedback: "Nothing to use!",
			expectResult:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCallbacks.Reset()

			result := tc.useFunc(references.Up)

			// Validate consistent SystemCallbacks behavior (tracked in Messages since alsoAddToOutput=true)
			assert.Equal(t, tc.expectResult, result, "Result should match expected")
			assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
			assert.Equal(t, tc.expectFeedback, mockCallbacks.Messages[0], "Should have correct feedback")

			// No audio effects in current stub implementation
			assert.Equal(t, 0, len(mockCallbacks.SoundEffectsPlayed), "Stub implementation should not play sound effects")
		})
	}
}

// TestUseCommand_ErrorHandling_Integration tests edge cases and error handling
func TestUseCommand_ErrorHandling_Integration(t *testing.T) {
	// Test use command error handling
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(1, 1). // Near edge to test boundary conditions
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	t.Run("UseAtMapEdges", func(t *testing.T) {
		// Test use near map edges
		edgePositions := []references.Position{
			{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 30, Y: 30},
		}

		originalPosition := gs.MapState.PlayerLocation.Position

		for _, pos := range edgePositions {
			gs.MapState.PlayerLocation.Position = pos
			mockCallbacks.Reset()

			result := gs.ActionUseSmallMap(references.Up)

			// Should handle edge cases gracefully (tracked in Messages since alsoAddToOutput=true)
			assert.False(t, result, "Use should handle edge positions gracefully at (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have message at edge position (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, "Nothing to use!", mockCallbacks.Messages[0], "Should have consistent message at edge")
		}

		// Restore original position
		gs.MapState.PlayerLocation.Position = originalPosition
	})

	t.Run("UseConsistency", func(t *testing.T) {
		// Test that feedback is consistent across multiple uses
		mockCallbacks.Reset()

		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			gs.ActionUseSmallMap(direction)
		}

		// Feedback should be consistent (tracked in Messages since alsoAddToOutput=true)
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each use")
		for i, message := range mockCallbacks.Messages {
			assert.Equal(t, "Nothing to use!", message, "Message %d should be consistent", i)
		}
	})
}

// TestUseCommand_Regression_Integration tests for future implementation
func TestUseCommand_Regression_Integration(t *testing.T) {
	// This test validates the current stub behavior and will catch regressions
	// when the full use implementation is added
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	t.Run("CurrentImplementationBaseline", func(t *testing.T) {
		mockCallbacks.Reset()

		// Document current behavior across all map types
		useFunctions := map[string]func(references.Direction) bool{
			"SmallMap":   gs.ActionUseSmallMap,
			"LargeMap":   gs.ActionUseLargeMap,
			"CombatMap":  gs.ActionUseCombatMap,
			"DungeonMap": gs.ActionUseDungeonMap,
		}

		expectedFeedback := map[string]string{
			"SmallMap":   "Nothing to use!",
			"LargeMap":   "Nothing to use!",
			"CombatMap":  "Not now!",
			"DungeonMap": "Nothing to use!",
		}

		for mapType, useFunc := range useFunctions {
			mockCallbacks.Reset()
			result := useFunc(references.Up)

			// This test will need updating when use is fully implemented (tracked in Messages since alsoAddToOutput=true)
			assert.False(t, result, "Current %s use implementation always returns false", mapType)
			assert.Equal(t, expectedFeedback[mapType], mockCallbacks.Messages[0], "Current %s implementation feedback", mapType)

			t.Logf("ℹ️ %s use tested with stub implementation", mapType)
		}

		t.Logf("ℹ️ Use command tested with stub implementation - ready for full implementation")
		t.Logf("ℹ️ Future implementation should handle: Magic Carpet, Skull Key, Amulet/Crown, Sceptre, Spyglass usage mechanics")
	})
}
