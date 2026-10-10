package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestMoongateEntry_Integration tests moongate entry when player steps on active moongate
func TestMoongateEntry_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(224, 133). // Initial location of moonstone 0
		WithSystemCallbacks().
		Build()
	// Set time to night (9 PM) when moongates are visible
	gs.DateTime.Hour = 21
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()
	buryMoonstoneForTest(gs.moonstoneSystem, 0)

	// Set moon phase to 0 so moonstone 0 is the active destination
	gs.DateTime.Day = 1 // Adjust day to get phase 0

	// Verify player starts at moonstone 0 location (224, 133)
	assert.Equal(t, references.Coordinate(224), gs.MapState.PlayerLocation.Position.X)
	assert.Equal(t, references.Coordinate(133), gs.MapState.PlayerLocation.Position.Y)

	// Verify moongates are visible at night
	assert.True(t, gs.AreMoongatesVisible(), "Moongates should be visible at 9 PM")

	// Verify player is on an active moongate
	assert.True(t, gs.IsPlayerOnActiveMoongate(), "Player should be on active moongate at moonstone 0 location")

	// Clear previous callbacks
	mockCallbacks.Reset()

	// Trigger moongate preparation (replaces CheckMoongateEntry which is now in UI layer)
	result := gs.PrepareMoongateTravel()

	// Verify moongate travel was triggered (Success or FailedButActivated)
	assert.True(t, result.Success || result.FailedButActivated, "Moongate should activate")

	// Should have sound effect
	assert.True(t, len(mockCallbacks.SoundEffectsPlayed) > 0, "Should play sound effect for moongate travel")
	assert.Contains(t, mockCallbacks.SoundEffectsPlayed, SoundMoongateTravel, "Should play moongate travel sound for moongate")

	assert.Empty(t, mockCallbacks.TimeAdvanced(), "activation has no intrinsic time cost")
	assert.Equal(t, MoongateStageHidden, gs.MoongateStage())
}

func TestApplyMoongateDepartureEffectsPreservesSourceGate(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	gs.moongateStage = 7

	gs.ApplyMoongateDepartureEffects()

	assert.Equal(t, MoongateStage(7), gs.MoongateStage())
}

// TestMoongateEntry_DuringDay_DoesNotWork tests that moongates are inactive during day
func TestMoongateEntry_DuringDay_DoesNotWork(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(224, 133). // Initial location of moonstone 0
		WithSystemCallbacks().
		Build()
	// Set time to day (12 PM) when moongates are invisible
	gs.DateTime.Hour = 12
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()

	// Verify moongates are invisible during day
	visibility := gs.GetMoongateVisibilityLevel()
	assert.Equal(t, float32(0.0), visibility, "Moongates should be invisible during day")

	// Verify player is NOT on an active moongate during day (moongates completely inactive)
	assert.False(t, gs.IsPlayerOnActiveMoongate(),
		"Player should NOT be on active moongate during day (moongates completely inactive)")

	// Clear previous callbacks
	mockCallbacks.Reset()

	// Trigger moongate preparation
	result := gs.PrepareMoongateTravel()

	// Should fail since moongate is not active during day
	assert.False(t, result.Success, "Moongate travel should not succeed during day")
	assert.False(t, result.FailedButActivated, "Moongate should not activate during day")

	// Verify moongate travel does NOT work during day (matching original Ultima V behavior)
	assert.Equal(t, 0, len(mockCallbacks.SoundEffectsPlayed), "Should not play sound effects during day")
	assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "Should not advance time during day")
}

// TestMoongateEntry_NotOnMoongate_NoEntry tests that nothing happens when not on moongate
func TestMoongateEntry_NotOnMoongate_NoEntry(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100). // Not on a moonstone location
		WithSystemCallbacks().
		Build()
	// Set time to night when moongates are visible
	gs.DateTime.Hour = 21
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()

	// Verify player is not on an active moongate
	assert.False(t, gs.IsPlayerOnActiveMoongate(), "Player should not be on active moongate at (100, 100)")

	// Clear previous callbacks
	mockCallbacks.Reset()

	// Trigger moongate preparation
	result := gs.PrepareMoongateTravel()

	// Should fail since player is not on a moongate
	assert.False(t, result.Success, "Moongate travel should fail when not on moongate")
	assert.False(t, result.FailedButActivated, "Moongate should not activate when player not on it")

	// Verify no moongate travel was triggered
	assert.Equal(t, 0, len(mockCallbacks.SoundEffectsPlayed), "Should not play sound effects when not on moongate")
	assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "Should not advance time when not on moongate")
	assert.Equal(t, 0, len(mockCallbacks.Messages), "Should not show travel messages when not on moongate")
}
