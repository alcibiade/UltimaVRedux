package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchReset_Integration tests the complete search reset functionality
func TestSearchReset_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Moonglow).
		WithSystemCallbacks().
		Build()

	require.NotNil(t, gs, "Game state should be initialized")
	require.NotNil(t, gs.SearchManager, "Search manager should be initialized")

	aslm := gs.SearchManager.GetAuthenticLocationManager()
	require.NotNil(t, aslm, "Authentic search location manager should be initialized")

	t.Run("ResetFunctionality", func(t *testing.T) {
		// Start with clean state
		aslm.ResetAllFoundStatus()
		initialFoundCount := aslm.GetFoundCount()
		assert.Equal(t, 0, initialFoundCount, "Should start with no found locations")

		// Mark some locations as found
		testIndices := []int{5, 15, 25, 35, 45}
		for _, idx := range testIndices {
			aslm.markAsFound(idx)
		}

		// Verify they are marked as found
		afterMarkingCount := aslm.GetFoundCount()
		assert.Equal(t, len(testIndices), afterMarkingCount,
			"Should have marked %d locations as found", len(testIndices))

		for _, idx := range testIndices {
			assert.True(t, aslm.isAlreadyFound(idx),
				"Index %d should be marked as found", idx)
		}

		// Now test the reset functionality
		aslm.ResetAllFoundStatus()

		// Verify all locations are reset
		afterResetCount := aslm.GetFoundCount()
		assert.Equal(t, 0, afterResetCount, "Should have no found locations after reset")

		for _, idx := range testIndices {
			assert.False(t, aslm.isAlreadyFound(idx),
				"Index %d should not be found after reset", idx)
		}

		// Verify the objFound array is completely cleared
		foundStatus := aslm.getFoundStatus()
		for i, byteVal := range foundStatus {
			assert.Equal(t, byte(0), byteVal,
				"Found status byte %d should be zero after reset", i)
		}
	})

	t.Run("ResetAfterRealSearch", func(t *testing.T) {
		// Reset to clean state
		mockCallbacks.Reset()
		aslm.ResetAllFoundStatus()

		// Try to search and find a real location
		moongowLocations := []references.SearchLocationReference{}
		for _, loc := range references.GetAllSearchLocations() {
			if references.Location(loc.MapNumber) == references.Moonglow {
				moongowLocations = append(moongowLocations, loc)
			}
		}

		require.Greater(t, len(moongowLocations), 0, "Should have Moonglow search locations")

		// Position player to search at a known location
		loc := moongowLocations[0]
		gs.MapState.PlayerLocation.Position = references.Position{
			X: references.Coordinate(loc.X),
			Y: references.Coordinate(loc.Y - 1), // Position south of target
		}

		// Perform search
		result := gs.ActionSearchSmallMap(references.Up)

		// Check if we found something
		foundCountAfterSearch := aslm.GetFoundCount()
		if result {
			// If search succeeded, should have found at least one location
			assert.Greater(t, foundCountAfterSearch, 0,
				"Should have found at least one location after successful search")

			// Verify that reset works even after real gameplay
			aslm.ResetAllFoundStatus()
			foundCountAfterReset := aslm.GetFoundCount()
			assert.Equal(t, 0, foundCountAfterReset,
				"Should have no found locations after reset, even after real search")
		}

		// Test multiple resets don't cause issues
		aslm.ResetAllFoundStatus()
		aslm.ResetAllFoundStatus()
		aslm.ResetAllFoundStatus()

		finalFoundCount := aslm.GetFoundCount()
		assert.Equal(t, 0, finalFoundCount,
			"Multiple resets should not cause issues")
	})

	t.Run("ResetStatistics", func(t *testing.T) {
		// Test that GetTotalSearchLocations returns correct count
		totalLocations := aslm.GetTotalSearchLocations()
		assert.Equal(t, 113, totalLocations,
			"Should have 113 total search locations in the system")

		// Test that found locations by map works correctly after reset
		aslm.ResetAllFoundStatus()
		foundByMap := aslm.GetFoundLocationsByMap()

		// After reset, no maps should have found locations
		totalFoundInAllMaps := 0
		for _, foundList := range foundByMap {
			totalFoundInAllMaps += len(foundList)
		}
		assert.Equal(t, 0, totalFoundInAllMaps,
			"No maps should have found locations after reset")
	})

	t.Run("ResetDebugOutput", func(t *testing.T) {
		// Test the debug print functionality after reset
		aslm.ResetAllFoundStatus()
		debugOutput := testDebugPrintFoundStatus(aslm)

		// Should contain expected elements
		assert.Contains(t, debugOutput, "=== SEARCH LOCATION DEBUG STATUS ===",
			"Debug output should contain header")
		assert.Contains(t, debugOutput, "Total found: 0/113 (0.0%)",
			"Debug output should show no found locations after reset")
		assert.Contains(t, debugOutput, "🔍",
			"Debug output should contain search symbols for unfound locations")
		assert.NotContains(t, debugOutput, "✅",
			"Debug output should not contain found symbols after reset")
	})

	t.Run("AuthenticSystemStandalone", func(t *testing.T) {
		// Test that the authentic system works standalone after legacy removal
		aslm.ResetAllFoundStatus()

		// Authentic system should be in clean state
		assert.Equal(t, 0, aslm.GetFoundCount(),
			"Authentic system should be reset")

		t.Log("Authentic search system reset completed successfully")
	})
}
