package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

func buryMoonstoneForTest(system *MoonstoneSystem, index int) {
	moonstone := system.GetMoonstoneByIndex(index)
	system.BuryMoonstone(index, moonstone.X, moonstone.Y, moonstone.Map, moonstone.Level)
}

// TestMoongateActiveAt_AnyMoonstone tests that ANY moonstone at a position creates an active moongate
func TestMoongateActiveAt_AnyMoonstone(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100). // Not at any moonstone
		WithSystemCallbacks().
		Build()
	// Set time to night when moongates are visible
	gs.DateTime.Hour = 21
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()
	buryMoonstoneForTest(gs.moonstoneSystem, 0)
	buryMoonstoneForTest(gs.moonstoneSystem, 1)

	// Set moon phase to 3 (different from moonstone 0)
	gs.DateTime.Day = 10 // Adjust day to get a different phase

	// Test moonstone 0 at position (224, 133) - phase 0
	pos0 := references.Position{X: 224, Y: 133}
	gs.observeMoonstone(0, gs.localMoonstoneSite(pos0), journal.MoonstoneSourceGate)
	assert.True(t, gs.IsMoongateActiveAt(pos0),
		"Moonstone 0 should create active moongate even when current phase is not 0")

	// Test moonstone 1 at position (96, 102) - phase 1
	pos1 := references.Position{X: 96, Y: 102}
	assert.True(t, gs.IsMoongateActiveAt(pos1),
		"Moonstone 1 should create active moongate even when current phase is not 1")

	// Test position with no moonstone
	posNone := references.Position{X: 100, Y: 100}
	assert.False(t, gs.IsMoongateActiveAt(posNone),
		"Position without moonstone should not have active moongate")
}

// TestMoongateTravel_SourceToDestination tests travel from any source moongate to phase destination
func TestMoongateTravel_SourceToDestination(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(96, 102). // Start at moonstone 1 position (phase 1)
		WithSystemCallbacks().
		Build()
	// Set time to night
	gs.DateTime.Hour = 21
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()
	buryMoonstoneForTest(gs.moonstoneSystem, 1)

	// Set moon phase to 0 (destination will be moonstone 0)
	gs.DateTime.Day = 1 // Adjust to get phase 0

	// Verify player is at moonstone 1 location
	assert.Equal(t, references.Coordinate(96), gs.MapState.PlayerLocation.Position.X)
	assert.Equal(t, references.Coordinate(102), gs.MapState.PlayerLocation.Position.Y)

	// Verify there's an active moongate at player position (moonstone 1)
	assert.True(t, gs.IsPlayerOnActiveMoongate(),
		"Player should be on active moongate at moonstone 1 position")

	// Clear callbacks
	mockCallbacks.Reset()

	// Execute moongate travel
	result := gs.PrepareMoongateTravel()
	assert.True(t, result.Success, "Moongate travel should succeed")

	// Verify travel occurred
	assert.True(t, len(mockCallbacks.SoundEffectsPlayed) > 0, "Should play sound effect")
	assert.Contains(t, mockCallbacks.SoundEffectsPlayed, SoundMoongateTravel)

	// Should have traveled FROM moonstone 1 TO moonstone 0 (current phase)
	// Note: Actual position change would be in MapState.PlayerLocation.Position
	// but our test doesn't fully simulate the teleportation mechanics
	assert.True(t, len(mockCallbacks.Messages) > 0, "Should have travel message")
	assert.Empty(t, mockCallbacks.TimeAdvanced(), "moongate activation has no intrinsic minute")
	assert.Equal(t, MoongateStageHidden, gs.MoongateStage())
}

// TestLookAtMoongate_ShowsCorrectPhase tests that looking at moongate shows the moonstone's phase, not current phase
func TestLookAtMoongate_ShowsCorrectPhase(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100). // Not at any moonstone
		WithSystemCallbacks().
		Build()
	// Set time to night
	gs.DateTime.Hour = 21
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()
	buryMoonstoneForTest(gs.moonstoneSystem, 0)
	buryMoonstoneForTest(gs.moonstoneSystem, 4)

	// Set current moon phase to 5 (different from moonstone we'll look at)
	gs.DateTime.Day = 15 // Adjust to get a different phase

	// Look at moonstone 0 (New Moon phase)
	pos0 := references.Position{X: 224, Y: 133}
	gs.observeMoonstone(0, gs.localMoonstoneSite(pos0), journal.MoonstoneSourceGate)
	mockCallbacks.Reset()
	result := gs.LookAtMoongate(pos0)

	assert.True(t, result, "Should successfully look at moongate")
	assert.Contains(t, mockCallbacks.Messages, "Moongate of New Moon",
		"Should show moonstone 0's phase name (New Moon), not current phase")

	// Look at moonstone 4 (Full Moon phase)
	pos4 := references.Position{X: 166, Y: 19}
	gs.observeMoonstone(4, gs.localMoonstoneSite(pos4), journal.MoonstoneSourceGate)
	mockCallbacks.Reset()
	result = gs.LookAtMoongate(pos4)

	assert.True(t, result, "Should successfully look at moongate")
	assert.Contains(t, mockCallbacks.Messages, "Moongate of Full Moon",
		"Should show moonstone 4's phase name (Full Moon), not current phase")
}

// TestMoongateInventory_NoActiveGate tests that moonstones in inventory don't create active moongates
func TestMoongateInventory_NoActiveGate(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(224, 133). // At moonstone 0 position
		WithSystemCallbacks().
		Build()
	// Set time to night
	gs.DateTime.Hour = 21
	gs.DateTime.Minute = 0
	gs.initializeMoongateStage()

	buryMoonstoneForTest(gs.moonstoneSystem, 0)
	// Verify moonstone 0 creates an active moongate while buried.
	pos0 := references.Position{X: 224, Y: 133}
	assert.True(t, gs.IsMoongateActiveAt(pos0),
		"Buried moonstone should create active moongate")

	// Move moonstone 0 to inventory
	gs.moonstoneSystem.Moonstones[0].Map = 0xFF // Inventory
	gs.moonstoneSystem.Moonstones[0].State = MoonstoneStateInInventory

	// Verify moonstone 0 no longer creates active moongate
	assert.False(t, gs.IsMoongateActiveAt(pos0),
		"Inventory moonstone should NOT create active moongate at its old position")
}
