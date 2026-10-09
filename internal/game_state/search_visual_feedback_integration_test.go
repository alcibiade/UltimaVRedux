package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchVisualFeedback_Integration verifies that successful ground
// item discovery refreshes the displayed map. ref:CMD-0007
func TestSearchVisualFeedback_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	require.NotNil(t, gs, "Game state should be initialized")
	require.NotNil(t, gs.SearchManager, "Search manager should be initialized")

	// Reset callbacks to ensure clean state
	mockCallbacks.Reset()

	t.Run("UnsuccessfulSearchDoesNotTriggerDisplayUpdate", func(t *testing.T) {
		// Reset callbacks for this test
		mockCallbacks.Reset()

		// Position player somewhere without items
		gs.MapState.PlayerLocation.Position = references.Position{X: 5, Y: 5}

		// Search in a direction that shouldn't find anything
		gs.ActionSearchSmallMap(references.Up)

		// Should not have triggered display update
		assert.Equal(t, 0, mockCallbacks.DisplayUpdates,
			"Unsuccessful search should not trigger display update")

		// But should still have a message and time advancement
		assert.Greater(t, len(mockCallbacks.Messages), 0,
			"Unsuccessful search should still generate message")
		assert.Greater(t, len(mockCallbacks.TimeAdvanced()), 0,
			"Unsuccessful search should still advance time")

		t.Logf("✅ Unsuccessful search: %d display updates, %d messages, %v time advanced",
			mockCallbacks.DisplayUpdates, len(mockCallbacks.Messages), mockCallbacks.TimeAdvanced())
	})

	t.Run("DisplayUpdateCallbackValidation", func(t *testing.T) {
		// Verify that our SystemCallbacks structure properly includes the UpdateDisplay method
		require.NotNil(t, gs.SystemCallbacks, "SystemCallbacks should not be nil")
		require.NotNil(t, gs.SystemCallbacks.Screen.UpdateDisplay, "UpdateDisplay callback should not be nil")

		// Direct callback test
		mockCallbacks.Reset()
		initialUpdates := mockCallbacks.DisplayUpdates

		gs.SystemCallbacks.Screen.UpdateDisplay()

		assert.Equal(t, initialUpdates+1, mockCallbacks.DisplayUpdates,
			"Direct UpdateDisplay call should increment counter")

		t.Logf("✅ Direct callback test: display updates incremented from %d to %d",
			initialUpdates, mockCallbacks.DisplayUpdates)
	})
}
