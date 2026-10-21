// Integration tests for Search command workflows with real game data
// These tests validate the search command flow including stub behavior,
// light requirements for dungeons, and SystemCallbacks integration.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestSearchCommand_SmallMap_Integration tests search in small maps
func TestSearchCommand_SmallMap_Integration(t *testing.T) {
	// Test search command in Lord British's Castle
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()

	t.Run("BasicSearchBehavior", func(t *testing.T) {
		mockCallbacks.Reset()

		// Test search in a direction
		result := gs.ActionSearchSmallMap(references.Up)

		// Current implementation always returns false (not found)
		assert.False(t, result, "Search currently returns not found (stub implementation)")

		// Validate SystemCallbacks integration
		assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
		assert.Equal(t, "Thou dost find nothing of note.", mockCallbacks.Messages[0])
		assert.Equal(t, mockCallbacks.Messages, mockCallbacks.SearchFeedbackMessages)

		// Validate time advancement
		assert.Equal(t, 1, len(mockCallbacks.TimeAdvanced()), "Should advance time")
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance 1 minute")
	})

	t.Run("SearchMultipleDirections", func(t *testing.T) {
		mockCallbacks.Reset()

		// Search in all directions
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gs.ActionSearchSmallMap(direction)
			assert.False(t, result, "Search should return not found for direction %v", direction)
		}

		// Validate all searches were processed
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each search")
		assert.Equal(t, 4, len(mockCallbacks.TimeAdvanced()), "Should advance time for each search")

		// All messages should be consistent
		for i, message := range mockCallbacks.Messages {
			assert.Equal(t, "Thou dost find nothing of note.", message, "Message %d should be consistent", i)
		}
	})
}

// TestSearchCommand_LargeMap_Integration tests search on large maps
func TestSearchCommand_LargeMap_Integration(t *testing.T) {
	// Test search command on large map
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	mockCallbacks.Reset()

	// Test search on large map
	result := gs.ActionSearchLargeMap(references.Up)

	// Current implementation always returns false (not found)
	assert.False(t, result, "Large map search currently returns not found (stub implementation)")

	// Validate SystemCallbacks integration
	assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
	assert.Equal(t, "Thou dost find nothing of note.", mockCallbacks.Messages[0])
	assert.Equal(t, mockCallbacks.Messages, mockCallbacks.SearchFeedbackMessages)

	// Validate time advancement
	assert.Equal(t, 1, len(mockCallbacks.TimeAdvanced()), "Should advance time")
	assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance 1 minute")
}

// TestSearchCommand_CombatMap_Integration tests search in combat (should be blocked)
func TestSearchCommand_CombatMap_Integration(t *testing.T) {
	// Test search during combat
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	mockCallbacks.Reset()

	// Test search during combat
	result := gs.ActionSearchCombatMap(references.Up)

	// Should be blocked during combat
	assert.False(t, result, "Search should not be allowed during combat")

	// Validate error message
	assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have error message")
	assert.Equal(t, "Not now!", mockCallbacks.Messages[0], "Should show 'Not now!' during combat")

	// Should not advance time during invalid action
	assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "Should not advance time when blocked")
}

// TestSearchCommand_DungeonMap_Integration tests real semantic dungeon search timing and light requirements.
func TestSearchCommand_DungeonMap_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Deceit).
		WithFloor(0).
		WithPlayerAt(1, 1).
		WithSystemCallbacks().
		Build()

	t.Run("SearchWithoutLight", func(t *testing.T) {
		mockCallbacks.Reset()
		gs.MapState.Lighting.RestoreTorchMinutes(0)
		gs.MapState.Lighting.SetMagicLight(false)
		var feedback string
		gs.SystemCallbacks.UI.ShowActionFeedback = func(message string, _ bool) { feedback = message }
		token, start := gs.BeginDungeonCommandCycle()
		result := gs.ActionSearchDungeonMap(references.Up)
		gs.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

		assert.False(t, result)
		assert.Equal(t, 1, start.ElapsedMinutes)
		assert.Equal(t, []string{"You find darkness."}, mockCallbacks.Messages)
		assert.Equal(t, "You find darkness.", feedback)
		assert.True(t, gs.IsAtDungeonCommandBoundary())
	})

	t.Run("SearchWithLight", func(t *testing.T) {
		mockCallbacks.Reset()
		gs.MapState.Lighting.LightTorch()
		token, start := gs.BeginDungeonCommandCycle()
		result := gs.ActionSearchDungeonMap(references.Up)
		gs.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

		assert.False(t, result)
		assert.Equal(t, 1, start.ElapsedMinutes)
		assert.Contains(t, mockCallbacks.Messages, "Search-Thou dost find nothing of note.")
		assert.True(t, gs.IsAtDungeonCommandBoundary())
	})
}

// TestSearchCommand_SystemCallbacks_Integration tests comprehensive callback behavior
func TestSearchCommand_SystemCallbacks_Integration(t *testing.T) {
	// Test SystemCallbacks consistency across different map types
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	testCases := []struct {
		name         string
		searchFunc   func(references.Direction) bool
		expectMsg    string
		expectTime   int
		expectResult bool
	}{
		{
			name:         "SmallMapSearch",
			searchFunc:   gs.ActionSearchSmallMap,
			expectMsg:    "Nothing there!",
			expectTime:   1,
			expectResult: false,
		},
		{
			name:         "LargeMapSearch",
			searchFunc:   gs.ActionSearchLargeMap,
			expectMsg:    "Nothing there!",
			expectTime:   1,
			expectResult: false,
		},
		{
			name:         "CombatMapSearch",
			searchFunc:   gs.ActionSearchCombatMap,
			expectMsg:    "Not now!",
			expectTime:   0,
			expectResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCallbacks.Reset()

			result := tc.searchFunc(references.Up)

			// Validate consistent SystemCallbacks behavior
			assert.Equal(t, tc.expectResult, result, "Result should match expected")
			assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
			assert.Equal(t, tc.expectMsg, mockCallbacks.Messages[0], "Should have correct message")
			assert.Equal(t, tc.expectTime, len(mockCallbacks.TimeAdvanced()), "Should have correct time advancement")

			if tc.expectTime > 0 {
				assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Should advance 1 minute when applicable")
			}
		})
	}
}

// TestSearchCommand_Regression_Integration tests for future implementation
func TestSearchCommand_Regression_Integration(t *testing.T) {
	// This test validates the current stub behavior and will catch regressions
	// when the full search implementation is added
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	t.Run("CurrentImplementationBaseline", func(t *testing.T) {
		mockCallbacks.Reset()

		// Document current behavior
		result := gs.ActionSearchSmallMap(references.Up)

		// This test will need updating when search is fully implemented
		assert.False(t, result, "Current implementation always returns false")
		assert.Equal(t, "Nothing there!", mockCallbacks.Messages[0], "Current implementation message")
		assert.Equal(t, 1, mockCallbacks.TimeAdvanced()[0], "Current implementation time cost")

		t.Logf("ℹ️ Search command tested with stub implementation - ready for full implementation")
		t.Logf("ℹ️ Future implementation should handle: stone caches, reagent caches, hidden objects, secret doors")
	})
}
