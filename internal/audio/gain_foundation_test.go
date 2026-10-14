package audio

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestIntermediateGainsStayContinuous(t *testing.T) {
	config := &AudioConfiguration{SoundEffectVolume: 0.8, AmbientVolume: 0.5}
	require.InDelta(t, 0.8*0.37, config.GetEffectiveVolume(0.37), 1e-9)
	require.InDelta(t, 0.5*0.37, config.GetEffectiveAmbientVolume(0.37), 1e-9)
	require.Zero(t, config.GetEffectiveVolume(0))
	require.InDelta(t, 0.8, config.GetEffectiveVolume(1), 1e-9)
	require.InDelta(t, 1.6, config.GetEffectiveVolume(2), 1e-9, "preserve calibration boost above unity")
	require.Equal(t, ConvertSliderToVolume(10), ConvertSliderToVolume(11), "invalid high slider is bounded")
}

func TestActiveSFXReactsToCategoryChangesDuringFade(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	clock := int64(100)
	modifier := soundEffectMetadataMap()[game_state.SoundEarthquake].VolumeModifier
	modifier *= productionLibrary().recordingCalibration(soundEffectMetadataMap()[game_state.SoundEarthquake].RecordingID)
	manager.soundEffectPlayer.nowMillis = func() int64 { return clock }
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 0.5, FadeInMs: 1000,
	}))
	clock = 600
	manager.Update()
	voice := activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	require.InDelta(t, 0.43*0.5*modifier*0.5, voice.Volume, 1e-9)
	require.NoError(t, manager.SetSoundEffectVolume(0.2))
	voice = activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	require.InDelta(t, 0.2*0.5*modifier*0.5, voice.Volume, 1e-9)
	manager.StopSound(game_state.SoundStopRequest{Effect: game_state.SoundEarthquake, FadeOutMs: 1000})
	clock = 1100
	manager.Update()
	voice = activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	require.InDelta(t, 0.2*0.5*modifier*0.25, voice.Volume, 1e-9)
	require.NoError(t, manager.SetSoundEffectVolume(0.8))
	voice = activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundEarthquake]
	require.InDelta(t, 0.8*0.5*modifier*0.25, voice.Volume, 1e-9)
}

func TestOwnedSFXLoopSurvivesOneShotPressure(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1,
	}))
	for range 12 {
		require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
			Effect: game_state.SoundHeal, Mode: game_state.SoundPlaybackOneShot, Volume: 1,
		}))
	}
	active := activeSoundsByEffect(manager.soundEffectPlayer)
	require.Equal(t, game_state.SoundPlaybackLoop, active[game_state.SoundEarthquake].Mode)
	require.LessOrEqual(t, manager.soundEffectPlayer.GetActiveSoundCount(), 8)
}

func TestAllOwnedLoopsRejectNewLoopWithoutEvictingOne(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	request := game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1,
	}
	for range 8 {
		require.NoError(t, manager.PlaySound(request))
	}
	require.ErrorIs(t, manager.PlaySound(request), errSoundVoiceLimit)
	require.Len(t, manager.soundEffectPlayer.activeSounds, 8)
	for _, sound := range manager.soundEffectPlayer.activeSounds {
		require.Equal(t, game_state.SoundPlaybackLoop, sound.Mode)
	}
}

func TestInvalidSFXRequestDoesNotCreateVoice(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	for _, volume := range []float64{-1, 2, math.NaN(), math.Inf(1)} {
		require.Error(t, manager.PlaySound(game_state.SoundPlaybackRequest{
			Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackOneShot, Volume: volume,
		}))
	}
	require.Zero(t, manager.soundEffectPlayer.GetActiveSoundCount())
}

func TestMusicCategoryRefreshKeepsContextGain(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlayBackgroundMusic("town"))
	require.NoError(t, manager.backgroundMusic.SetVolume(0.4))
	require.NoError(t, manager.SetMusicVolume(0.2))
	town := musicTrackMetadataMap()["town"]
	require.InDelta(t, 0.2*0.4*town.VolumeModifier*productionLibrary().recordingCalibration(town.RecordingID),
		manager.backgroundMusic.player.Volume(), 1e-9)
	require.NoError(t, manager.PlayBackgroundMusic("combat"))
	combat := musicTrackMetadataMap()["combat"]
	require.InDelta(t, 0.2*0.4*combat.VolumeModifier*productionLibrary().recordingCalibration(combat.RecordingID),
		manager.backgroundMusic.player.Volume(), 1e-9)
	require.Error(t, manager.backgroundMusic.SetVolume(math.NaN()))
}

func TestAmbientReentryAndLiveCategoryGain(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	now := time.Unix(10, 0)
	ambient.now = func() time.Time { return now }
	definition := AmbientSoundDefinition{
		SoundType: AmbientFire, FileName: "assets/sounds/magic/healing_01.wav",
		MaxDistance: 10, BaseVolume: 0.8, LoopSound: true,
		FadeInTime: time.Second, FadeOutTime: time.Second,
	}
	ambient.AddCustomAmbientSound(indexes.FireLogs, definition)
	position := references.Position{}
	tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}, Position: position}}
	ambient.UpdatePlayerPosition(position, tiles)
	now = now.Add(500 * time.Millisecond)
	ambient.UpdatePlayerPosition(position, tiles)
	instance := ambient.activeSounds[AmbientFire]
	require.InDelta(t, 0.5, instance.FadeGain, 1e-9)
	ambient.UpdatePlayerPosition(position, nil)
	now = now.Add(250 * time.Millisecond)
	ambient.UpdatePlayerPosition(position, nil)
	require.InDelta(t, 0.375, instance.FadeGain, 1e-9)
	ambient.UpdatePlayerPosition(position, tiles)
	require.Same(t, instance, ambient.activeSounds[AmbientFire], "re-entry retains the loop")
	require.InDelta(t, 0.375, instance.FadeGain, 1e-9, "re-entry must not jump")
	require.NoError(t, manager.SetAmbientVolume(0.25))
	require.InDelta(t, 0.25*0.8*0.375, instance.Player.Volume(), 1e-9)
	now = now.Add(time.Second)
	ambient.UpdatePlayerPosition(position, tiles)
	require.InDelta(t, 0.25*0.8, instance.Player.Volume(), 1e-9)
}

func TestAmbientSourceStopsAfterMovingOutOfRange(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	now := time.Unix(10, 0)
	ambient.now = func() time.Time { return now }
	definition := AmbientSoundDefinition{
		SoundType: AmbientFire, FileName: "assets/sounds/magic/healing_01.wav",
		MaxDistance: 8, BaseVolume: 1, LoopSound: true,
		FadeOutTime: time.Second,
	}
	ambient.AddCustomAmbientSound(indexes.FireLogs, definition)
	source := references.Position{X: 10, Y: 10}
	tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}, Position: source}}
	ambient.UpdatePlayerPosition(source, tiles)
	require.Contains(t, ambient.activeSounds, AmbientFire)
	farAway := references.Position{X: 30, Y: 30}
	ambient.UpdatePlayerPosition(farAway, tiles)
	require.Equal(t, AmbientFadeOut, ambient.activeSounds[AmbientFire].FadeState)
	now = now.Add(time.Second)
	ambient.UpdatePlayerPosition(farAway, tiles)
	require.NotContains(t, ambient.activeSounds, AmbientFire)
}

func TestAmbientSuppressionAndDisableCloseExistingVoices(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	ambient := manager.ambientSounds
	definition := AmbientSoundDefinition{
		SoundType: AmbientWater, FileName: "assets/sounds/magic/healing_01.wav",
		MaxDistance: 10, BaseVolume: 1, LoopSound: true,
	}
	ambient.createAmbientSound(definition, references.Position{}, 0, AmbientWater)
	require.Len(t, ambient.activeSounds, 1)
	manager.SetAmbientSuppressed(true)
	require.Empty(t, ambient.activeSounds)
	manager.SetEnabled(false)
	manager.SetEnabled(true)
	require.False(t, ambient.IsEnabled(), "global enable must respect workflow suppression")
	manager.SetAmbientSuppressed(false)
	require.True(t, ambient.IsEnabled())
	ambient.createAmbientSound(definition, references.Position{}, 0, AmbientWater)
	manager.SetEnabled(false)
	require.Empty(t, ambient.activeSounds)
}

func TestAmbientSourceKeepsAcceptedPositiveCalibrationGain(t *testing.T) {
	var source AmbientSoundSource
	source.SetBaseVolume(2)
	require.Equal(t, 2.0, source.GetBaseVolume(), "do not clamp an intermediate calibrated gain to one")
	source.SetBaseVolume(math.NaN())
	require.Zero(t, source.GetBaseVolume())
}
