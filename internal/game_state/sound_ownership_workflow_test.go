package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEarthquakePresentationsReleaseTheirOwnVoices(t *testing.T) {
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
	require.True(t, state.QueueDebugEarthquakeFeedback(10000, 1))
	state.EventQueueManager.UpdateFeedback()
	require.Equal(t, []int{1, 2}, started)
	for range 20 {
		state.ElapsedMs += 250
		state.EventQueueManager.Update()
		state.EventQueueManager.UpdateFeedback()
		if !state.EventQueueManager.HasWorkflow(harpsichordEarthquakeWorkflowName) {
			break
		}
	}
	require.Equal(t, []int{1}, stopped, "finishing the workflow must not stop the overlapping feedback voice")
	require.True(t, state.IsDebugEarthquakeFeedbackActive())
	state.EventQueueManager.Shutdown()
	state.EventQueueManager.Shutdown()
	require.Equal(t, []int{1, 2}, stopped, "shutdown finalizes each owned presentation exactly once")
	require.Zero(t, legacyStops)
	require.Equal(t, beforeRNG, state.GetRNG().Calls())
	require.Equal(t, beforeDate, state.DateTime)
}

func TestCancelledBeforeStartCreatesAndStopsNoAudioVoice(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.startHarpsichordEarthquakeWorkflow()
	state.QueueDebugEarthquakeFeedback(1000, 1)
	state.EventQueueManager.Shutdown()
	require.Empty(t, callbacks.PlaybackRequests)
	require.Empty(t, callbacks.StopRequests)
}
