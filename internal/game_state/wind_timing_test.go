package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestWindUpdateTiming_IndependentOfTurns verifies that wind updates happen independently
// of player turns, matching original Ultima V behavior where wind changes occur
// continuously in the main game loop rather than only when the player moves
func TestWindUpdateTiming_IndependentOfTurns(t *testing.T) {
	// Initialize game state using integration test builder for proper setup
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(12345). // Fixed seed for reproducible results
		Build()

	// Record initial wind state
	initialWind := gameState.GetCurrentWind()

	// Call ProcessWindOnTurnAdvancement multiple times (as would happen in updateFixed)
	// This simulates the frequent calls that happen in the main game loop
	changeCount := 0
	const testIterations = 200 // Enough iterations to likely trigger some changes

	for i := 0; i < testIterations; i++ {
		previousWind := gameState.GetCurrentWind()
		gameState.ProcessWindOnTurnAdvancement()
		currentWind := gameState.GetCurrentWind()

		if currentWind != previousWind {
			changeCount++
		}
	}

	// With 1/64 chance per call and 200 iterations, we should see some changes
	// but not too many (expected average would be around 3-4 changes)
	assert.Greater(t, changeCount, 0, "Wind should change at least once during frequent updates")
	assert.Less(t, changeCount, 20, "Wind shouldn't change too frequently (sanity check)")

	t.Logf("Wind changes: %d out of %d iterations", changeCount, testIterations)
	t.Logf("Initial wind: %s, Final wind: %s", initialWind.String(), gameState.GetCurrentWind().String())
}
