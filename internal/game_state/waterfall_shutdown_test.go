package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestWaterfallAdmissionRejectsShutdownWithoutPublishingOwner(t *testing.T) {
	position := references.Position{X: 50, Y: 50}
	state, callbacks := newWaterfallContinuationFixture(t, position)
	setWaterfallContinuationChain(state, references.MainFloor, position, 2)
	state.EventQueueManager.Shutdown()
	date, calls, hp := state.DateTime, state.GetRNG().Calls(), state.PartyState.GetAvatar().GetCurrentHp()
	for range 2 {
		assert.False(t, state.ProcessWaterfallHazard(&position))
		require.NotPanics(t, func() { assert.False(t, state.ResumeWaterfallContinuation()) })
		state.EventQueueManager.Shutdown()
	}
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, state.IsWaterfallHazardActive())
	assert.False(t, callbacks.InputBlocked)
	assert.Equal(t, date, state.DateTime)
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Equal(t, hp, state.PartyState.GetAvatar().GetCurrentHp())
	assert.Equal(t, position, state.MapState.PlayerLocation.Position)
	assert.NoError(t, state.ValidateSaveSnapshotEligibility())
}

func TestWaterfallCreationHandlesReentrantClockShutdown(t *testing.T) {
	position := references.Position{X: 50, Y: 50}
	state, callbacks := newWaterfallContinuationFixture(t, position)
	setWaterfallContinuationChain(state, references.MainFloor, position, 2)
	var manager *event_queue.EventQueueManager
	shutdownOnClockRead := false
	manager = event_queue.NewEventQueueManager(func() int64 {
		if shutdownOnClockRead {
			shutdownOnClockRead = false
			manager.Shutdown()
		}

		return state.ElapsedMs
	}, &eventQueueSystemCallbacks{gameState: state})
	manager.SetShutdownAction(func() { state.cancelWaterfallCascade(state.activeWaterfall) })
	state.EventQueueManager = manager
	shutdownOnClockRead = true
	assert.False(t, state.ProcessWaterfallHazard(&position), "a Start rejected during construction cannot claim ownership")
	assert.True(t, manager.IsShutdown())
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, manager.HasWorkflow(waterfallWorkflowName))
	require.NotPanics(t, func() { assert.False(t, state.ResumeWaterfallContinuation()) })
	assert.False(t, callbacks.InputBlocked)
	assert.Zero(t, state.GetRNG().Calls())
	assert.Equal(t, position, state.MapState.PlayerLocation.Position)
	assert.NoError(t, state.ValidateSaveSnapshotEligibility())
}
