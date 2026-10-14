package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoundBuilderDispatchesOneCompletePlaybackRequest(t *testing.T) {
	requests := make([]SoundPlaybackRequest, 0, 1)
	callbacks := newSoundBuilderTestCallbacks(
		func(request SoundPlaybackRequest) { requests = append(requests, request) },
		func(SoundStopRequest) {},
	)

	callbacks.Play(SoundEarthquake).Loop().FadeIn(200).Volume(0.5).Start()

	require.Equal(t, []SoundPlaybackRequest{{
		Effect: SoundEarthquake, Mode: SoundPlaybackLoop, Volume: 0.5, FadeInMs: 200,
	}}, requests)
}

func TestSoundBuilderDefaultsToOneShotAndClampsVolume(t *testing.T) {
	requests := make([]SoundPlaybackRequest, 0, 3)
	callbacks := newSoundBuilderTestCallbacks(
		func(request SoundPlaybackRequest) { requests = append(requests, request) },
		func(SoundStopRequest) {},
	)

	callbacks.PlaySoundEffect(SoundUnlock)
	callbacks.Play(SoundEarthquake).Loop().OneShot().Volume(-0.5).Start()
	callbacks.Play(SoundEarthquake).Volume(1.5).Start()

	require.Len(t, requests, 3)
	assert.Equal(t, SoundPlaybackRequest{
		Effect: SoundUnlock, Mode: SoundPlaybackOneShot, Volume: 1.0,
	}, requests[0])
	assert.Equal(t, SoundPlaybackOneShot, requests[1].Mode)
	assert.Zero(t, requests[1].Volume)
	assert.Equal(t, 1.0, requests[2].Volume)
}

func TestStopBuilderDispatchesOneCompleteStopRequest(t *testing.T) {
	requests := make([]SoundStopRequest, 0, 2)
	callbacks := newSoundBuilderTestCallbacks(
		func(SoundPlaybackRequest) {},
		func(request SoundStopRequest) { requests = append(requests, request) },
	)

	callbacks.Stop(SoundWind).Now()
	callbacks.Stop(SoundEarthquake).FadeOut(300).Now()

	assert.Equal(t, []SoundStopRequest{
		{Effect: SoundWind},
		{Effect: SoundEarthquake, FadeOutMs: 300},
	}, requests)
}

func TestShardRitualEarthquakeSoundIDIsAppended(t *testing.T) {
	if got, want := SoundShardRitualEarthquake, SoundTavernBardMusic+1; got != want {
		t.Fatalf("SoundShardRitualEarthquake ID = %d, want appended ID %d", got, want)
	}
}

func newSoundBuilderTestCallbacks(
	requestPlayback func(SoundPlaybackRequest),
	requestStop func(SoundStopRequest),
) AudioCallbacks {
	return NewAudioCallbacks(requestPlayback, requestStop, func() {}, func() {}, func() {},
		func(request SoundPlaybackRequest) SoundVoice {
			requestPlayback(request)
			return testSoundVoice(func(fade int64) { requestStop(SoundStopRequest{Effect: request.Effect, FadeOutMs: fade}) })
		})
}

type testSoundVoice func(int64)

func (voice testSoundVoice) Stop(fade int64) { voice(fade) }
