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

// TestWindUpdateTiming_FrequencyCheck verifies that wind changes occur at the expected
// frequency based on the original Ultima V 1/64 probability at ~18Hz tick rate
func TestWindUpdateTiming_FrequencyCheck(t *testing.T) {
	// Initialize game state using integration test builder
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(123456). // Different seed that should give more variation
		Build()

	// Simulate original game timing: 18 ticks per second for several seconds
	// Original: ~18 ticks/sec * 10 seconds = 180 wind checks
	// Expected: 180 / 64 ≈ 2.8 changes over 10 seconds
	const ticksPerSecond = 18
	const testDurationSeconds = 10
	const totalTicks = ticksPerSecond * testDurationSeconds
	changeCount := 0

	for i := 0; i < totalTicks; i++ {
		previousWind := gameState.GetCurrentWind()
		gameState.ProcessWindOnTurnAdvancement()
		currentWind := gameState.GetCurrentWind()

		if currentWind != previousWind {
			changeCount++
		}
	}

	// Expected: 180 ticks / 64 = ~2.8 changes (allow 1-6 changes for variance)
	expectedChanges := float64(totalTicks) / 64.0
	minChanges := 1
	maxChanges := 8 // Allow for randomness variance

	assert.GreaterOrEqual(t, changeCount, minChanges,
		"Wind change count too low (expected ~%.1f, got %d)", expectedChanges, changeCount)
	assert.LessOrEqual(t, changeCount, maxChanges,
		"Wind change count too high (expected ~%.1f, got %d)", expectedChanges, changeCount)

	actualRate := float64(changeCount) / float64(testDurationSeconds)
	expectedRate := float64(ticksPerSecond) / 64.0 // ~0.28 changes per second

	t.Logf("Wind change rate: %.2f changes/second (expected: %.2f)", actualRate, expectedRate)
	t.Logf("Changes: %d over %d seconds (%d ticks)", changeCount, testDurationSeconds, totalTicks)
	t.Logf("Average interval between changes: %.1f seconds", float64(testDurationSeconds)/float64(changeCount))
}
