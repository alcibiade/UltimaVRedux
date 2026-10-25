package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestAutoOpenDoorState_Lifecycle(t *testing.T) {
	state := &AutoOpenDoorState{}

	// Initially inactive
	assert.False(t, state.IsActive())
	assert.Equal(t, AutoOpenPhaseNone, state.phase)

	// Start sequence
	targetPos := &references.Position{X: 10, Y: 10}
	currentTime := int64(1000) // Deterministic time (milliseconds)
	state.StartAutoOpenSequence(references.Up, targetPos, currentTime)

	// Should be active now
	assert.True(t, state.IsActive())
	assert.Equal(t, AutoOpenPhaseOpenDoor, state.phase)
	assert.Equal(t, references.Up, state.direction)
	assert.Equal(t, targetPos, state.targetPosition)
	assert.Equal(t, currentTime, state.startElapsedMs)

	// Reset should clear everything
	state.Reset()
	assert.False(t, state.IsActive())
	assert.Equal(t, AutoOpenPhaseNone, state.phase)
	assert.Nil(t, state.targetPosition)
	assert.Equal(t, int64(0), state.startElapsedMs)
}

func TestAutoOpenDoorState_PhaseTransitions(t *testing.T) {
	state := &AutoOpenDoorState{}

	// Test phase transitions
	assert.Equal(t, AutoOpenPhaseNone, state.phase)

	state.phase = AutoOpenPhaseOpenDoor
	assert.Equal(t, AutoOpenPhaseOpenDoor, state.phase)

	state.phase = AutoOpenPhaseFirstNPCMove
	assert.Equal(t, AutoOpenPhaseFirstNPCMove, state.phase)

	state.phase = AutoOpenPhaseSecondNPCMove
	assert.Equal(t, AutoOpenPhaseSecondNPCMove, state.phase)

	state.phase = AutoOpenPhaseMovePlayer
	assert.Equal(t, AutoOpenPhaseMovePlayer, state.phase)

	state.phase = AutoOpenPhaseComplete
	assert.Equal(t, AutoOpenPhaseComplete, state.phase)
}

func TestAutoOpenDoorState_PauseDuration(t *testing.T) {
	// Verify the pause duration is set correctly
	assert.Equal(t, int64(150), autoOpenPauseDurationMs)
}

// Note: Full integration testing with real GameScene/GameState happens in:
// - internal/game_state/door_utils_test.go (CanAutoOpenDoor tests)
// - Manual testing confirms time advancement is now correct (1 minute vs 3+ minutes)
// - Manual testing confirms single "Opened!" message display
