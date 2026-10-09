package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestMoongateTeleportationMaterializesDestinationBeforeReveal(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(3, 4).
		WithSystemCallbacks().
		Build()
	gs.DateTime.Hour = 21
	gs.moongateStage = 7
	destination := MoongateDestination{
		X: 9, Y: 11, Map: uint8(references.Britain), Level: 0, Location: references.Britain,
	}
	var destinationStabilized bool
	gs.SystemCallbacks.Visual.ResetTeleportationSourceRenderer = func() {
		assert.Equal(t, MoongateStage(7), gs.MoongateStage(), "source gate remains steady through dissolve-out")
	}
	gs.SystemCallbacks.Visual.StabilizeTeleportationDestinationRenderer = func(got MoongateDestination) {
		assert.Equal(t, destination, got)
		assert.Equal(t, MoongateStageFull, gs.MoongateStage())
		destinationStabilized = true
	}
	gs.SystemCallbacks.Screen.UpdateDisplay = func() {
		require.True(t, destinationStabilized, "destination gate must be stabilized before display update")
		assert.Equal(t, MoongateStageFull, gs.MoongateStage())
	}

	gs.EventQueueManager.Workflow("moongate_destination_stage_test").
		Priority(event_queue.PriorityHigh).
		Sequential().
		First(func() {}).
		ThenSequence(gs.BuildMoongateTeleportationSequence(destination)).
		Finally(func() {}).
		Start()
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs += 1000
		gs.EventQueueManager.Update()
	}

	require.True(t, destinationStabilized)
	assert.Equal(t, MoongateStageFull, gs.MoongateStage())
	gs.UpdateMoongateTransition()
	assert.False(t, gs.IsMoongateTransitionActive(), "arrival must not start gate growth")
}
