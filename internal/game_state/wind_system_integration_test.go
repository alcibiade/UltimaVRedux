package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestWindSystemIntegrationBasic(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(12345).
		Build()

	// Test that wind system initializes with calm winds
	currentWind := gameState.GetCurrentWind()
	assert.Equal(t, environment.Calm, currentWind, "Should start with calm winds")
	assert.True(t, gameState.IsWindCalm(), "Should report wind as calm")

	// Test setting wind direction
	gameState.SetWindDirection(environment.North, true)
	assert.Equal(t, environment.North, gameState.GetCurrentWind(), "Should set wind to north")
	assert.False(t, gameState.IsWindCalm(), "Should no longer be calm")

	// Note: Wind UI messages are disabled by design - wind changes are silent
}

func TestWindSystemIntegrationTurnAdvancement(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(11111).
		Build()

	// Process many turns to potentially trigger wind change
	for i := 0; i < 200; i++ {
		gameState.ProcessWindOnTurnAdvancement()
	}

	// Wind may or may not have changed (1/64 chance per turn)
	// But the system should still be working
	currentWind := gameState.GetCurrentWind()

	// Verify wind is still a valid direction
	assert.True(t, currentWind >= environment.Calm && currentWind <= environment.West,
		"Wind should be a valid direction")

	// Note: Wind changes are silent - no UI messages expected for natural wind changes
}

func TestWindSystemIntegrationPirateShipSpeed(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(22222).
		Build()

	// Test wind effects on pirate ship movement in different directions
	// Based on authentic Ultima V algorithm analysis
	testCases := []struct {
		windDir       environment.WindDirection
		shipDir       references.Direction
		expectedSpeed int
		description   string
	}{
		{environment.North, references.Up, 2, "North wind, north ship = Y-axis mismatch"},
		{environment.North, references.Down, 1, "North wind, south ship = Y-axis match"},
		{environment.North, references.Left, 3, "North wind, west ship = both axes mismatch"},
		{environment.North, references.Right, 3, "North wind, east ship = both axes mismatch"},
		{environment.Calm, references.Up, 4, "Calm wind = all ships stalled"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			gameState.SetWindDirection(tc.windDir, false)
			speed := gameState.GetPirateShipSpeedMultiplier(tc.shipDir)
			assert.Equal(t, tc.expectedSpeed, speed, tc.description)
		})
	}
}

func TestWindSystemIntegrationDirectionMapping(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(33333).
		Build()

	// Test mapping between references.Direction and environment.WindDirection
	testCases := []struct {
		refDir  references.Direction
		windDir environment.WindDirection
		success bool
	}{
		{references.Up, environment.North, true},
		{references.Down, environment.South, true},
		{references.Left, environment.West, true},
		{references.Right, environment.East, true},
		{references.NoneDirection, environment.Calm, false},
	}

	for _, tc := range testCases {
		result := gameState.SetWindByDirection(tc.refDir, false)
		assert.Equal(t, tc.success, result)

		if tc.success {
			assert.Equal(t, tc.windDir, gameState.GetCurrentWind())
		}
	}
}

func TestWindSystemIntegrationLazyCaching(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithRandomSeed(12345).
		WithSystemCallbacks().
		Build()

	// Wind system is now initialized during GameState creation
	assert.NotNil(t, gameState.windSystem, "Wind system should be initialized during GameState creation")

	// Wind system should work with initialized system
	wind := gameState.GetCurrentWind()
	assert.NotNil(t, gameState.windSystem, "Wind system should remain initialized after use")
	assert.Equal(t, environment.Calm, wind, "Should start with calm winds")

	// Subsequent calls should reuse
	firstInstance := gameState.windSystem
	gameState.ProcessWindOnTurnAdvancement()
	assert.Equal(t, firstInstance, gameState.windSystem, "Should reuse existing wind system")
}

func TestWindSystemIntegrationDeterministicBehavior(t *testing.T) {
	// Test that wind changes are deterministic with fixed seeds
	gameState1, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithRandomSeed(55555).
		WithSystemCallbacks().
		Build()

	gameState2, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithRandomSeed(55555).
		WithSystemCallbacks().
		Build()

	// Process same number of wind updates
	for i := 0; i < 100; i++ {
		gameState1.ProcessWindOnTurnAdvancement()
		gameState2.ProcessWindOnTurnAdvancement()
	}

	// Should have identical wind states
	assert.Equal(t, gameState1.GetCurrentWind(), gameState2.GetCurrentWind(),
		"Deterministic seeds should produce identical wind states")
}
