package camping

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestSleepingAccelerationIsOneWayAndEndsAtFinalCycle(t *testing.T) {
	scene := NewCampingScene(1, -1, references.PlayerLocation{})
	require.False(t, scene.RequestAcceleration(), "setup is not sleep")
	scene.StartCamping()
	scene.BeginSleeping()
	require.True(t, scene.RequestAcceleration())
	require.True(t, scene.RequestAcceleration(), "subsequent requests are idempotent")
	require.True(t, scene.IsAccelerated())
	require.Zero(t, scene.CurrentCycle, "intent does not advance gameplay")
	for range CyclesPerHour {
		require.True(t, scene.AdvanceExactCycle())
	}
	require.False(t, scene.IsSleeping())
	require.False(t, scene.IsAccelerated())
	require.False(t, scene.RequestAcceleration(), "results cannot accelerate")
	scene.StartCamping()
	scene.BeginSleeping()
	require.False(t, scene.IsAccelerated(), "a new camp begins normally")
	require.True(t, scene.RequestAcceleration())
	scene.InterruptCamping()
	require.False(t, scene.IsSleeping())
	require.False(t, scene.IsAccelerated())
}
