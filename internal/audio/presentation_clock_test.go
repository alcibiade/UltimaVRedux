package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestRuntimeLanesShareMonotonicPresentationTime(t *testing.T) {
	now := time.Unix(100, 0)
	config, err := NewAudioConfiguration()
	require.NoError(t, err)
	manager, err := NewAudioManagerWithRuntime(config, getSharedAudioContext(), func() time.Time { return now })
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1, FadeInMs: 1000,
	}))
	require.NoError(t, manager.PlayMusicForLocation(references.Britain))
	manager.ambientSounds.AddCustomAmbientSound(indexes.FireLogs, AmbientSoundDefinition{
		SoundType: AmbientFire, FileName: "assets/sounds/magic/healing_01.wav", BaseVolume: 1, MaxDistance: 8,
		LoopSound: true, FadeInTime: time.Second, FadeOutTime: time.Second,
	})
	manager.UpdateAmbientSounds(references.Position{}, []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}}})
	sound := activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	ambient := manager.ambientSounds.activeSounds[AmbientFire]
	require.Zero(t, sound.StartTime)
	now = now.Add(time.Second / 2)
	manager.Update()
	for range 2 {
		sound = activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
		require.InDelta(t, 0.5, sound.FadeGain, 1e-9)
		require.InDelta(t, 0.5, ambient.FadeGain, 1e-9)
		require.InDelta(t, 0.25, manager.musicManager.policy.envelope, 1e-9)
		now = now.Add(-time.Hour)
		manager.Update()
	}
	now = time.Unix(100, 0).Add(time.Hour)
	manager.Update()
	sound = activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	require.Equal(t, 1.0, sound.FadeGain)
	require.Equal(t, 1.0, ambient.FadeGain)
	require.Equal(t, 1.0, manager.musicManager.policy.envelope)
}

func TestSFXReleaseAtPresentationZeroRetiresAfterLongSuspension(t *testing.T) {
	now := time.Time{}
	config, err := NewAudioConfiguration()
	require.NoError(t, err)
	manager, err := NewAudioManagerWithRuntime(config, getSharedAudioContext(), func() time.Time { return now })
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1}))
	voice := activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	manager.StopSound(game_state.SoundStopRequest{Effect: game_state.SoundEarthquake, FadeOutMs: 1000})
	voice = activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	require.Zero(t, voice.FadeOutStart)
	now = now.Add(time.Second / 2)
	manager.Update()
	voice = activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	require.InDelta(t, 0.5, voice.FadeGain, 1e-9)
	now = now.Add(24 * time.Hour)
	manager.Update()
	require.Empty(t, manager.soundEffectPlayer.activeSounds)
	require.True(t, voice.Player.retired)
}
