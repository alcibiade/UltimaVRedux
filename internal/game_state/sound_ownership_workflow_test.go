package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEarthquakeWorkflowReleasesItsOwnVoiceOnce(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().WithRandomSeed(42).Build()
	base := state.SystemCallbacks.Audio
	var started, stopped []int
	legacyStops := 0
	state.SystemCallbacks.Audio = NewAudioCallbacks(base.RequestPlayback,
		func(SoundStopRequest) { legacyStops++ }, base.UpdateLocationMusic,
		base.PauseBackgroundMusic, base.ResumeBackgroundMusic,
		func(request SoundPlaybackRequest) SoundVoice {
			require.Equal(t, SoundEarthquake, request.Effect)
			id := len(started) + 1
			started = append(started, id)
			return testSoundVoice(func(int64) { stopped = append(stopped, id) })
		})
	beforeRNG, beforeDate := state.GetRNG().Calls(), state.DateTime
	state.startHarpsichordEarthquakeWorkflow()
	state.EventQueueManager.Update()
	require.Equal(t, []int{1}, started)
	for range 20 {
		state.ElapsedMs += 250
		state.EventQueueManager.Update()
		if !state.EventQueueManager.HasWorkflow(harpsichordEarthquakeWorkflowName) {
			break
		}
	}
	require.Equal(t, []int{1}, stopped)
	state.EventQueueManager.Shutdown()
	require.Equal(t, []int{1}, stopped, "shutdown does not finalize a completed presentation again")
	require.Zero(t, legacyStops)
	require.Equal(t, beforeRNG, state.GetRNG().Calls())
	require.Equal(t, beforeDate, state.DateTime)
}

func TestCancelledBeforeStartCreatesAndStopsNoAudioVoice(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.startHarpsichordEarthquakeWorkflow()
	state.EventQueueManager.Shutdown()
	require.Empty(t, callbacks.PlaybackRequests)
	require.Empty(t, callbacks.StopRequests)
}
