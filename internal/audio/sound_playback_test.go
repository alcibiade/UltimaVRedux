package audio

import (
	"io"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatHitRemainsAudibleAlongsideSwing(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundCombatSwing))
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundCombatHit))
	active := activeSoundsByEffect(manager.soundEffectPlayer)
	require.Contains(t, active, game_state.SoundCombatSwing)
	require.Contains(t, active, game_state.SoundCombatHit)

	// Compare the selected source audio rather than just cue gains: the raw hit
	// and swing recordings have very different levels.
	levels := make(map[game_state.SoundEffect]float64)
	for _, effect := range []game_state.SoundEffect{game_state.SoundCombatSwing, game_state.SoundCombatHit} {
		cue := productionLibrary().Cues["sfx/"+catalogEffectNames()[effect]]
		recording := productionLibrary().Recordings[cue.RecordingID]
		clip, loadErr := LoadRecording("assets/" + *recording.File)
		require.NoError(t, loadErr)
		level, measureErr := clip.SuggestCalibration(recording.PlaybackWindow, "impact", nil)
		require.NoError(t, measureErr)
		levels[effect] = level.MeasuredDB + 20*math.Log10(
			cue.VolumeModifier*productionLibrary().recordingCalibration(cue.RecordingID))
	}
	require.Greater(t, levels[game_state.SoundCombatHit], levels[game_state.SoundCombatSwing]+3,
		"a damage impact should stand out over its overlapping opening swing")
}

func TestPlaybackRequestModeOverridesMetadataAndAppliesVolumeOnce(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.ensureAssetsLoaded())

	seenModes := make([]game_state.SoundPlaybackMode, 0, 2)
	for _, effect := range []game_state.SoundEffect{game_state.SoundCannonFire, game_state.SoundHeal} {
		asset := manager.soundEffectPlayer.soundAssets[effect][0]
		require.NotNil(t, asset)
		originalFactory := asset.StreamFactory
		asset.StreamFactory = func(mode game_state.SoundPlaybackMode) (io.ReadSeeker, error) {
			seenModes = append(seenModes, mode)

			return originalFactory(mode)
		}
	}

	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundCannonFire, Mode: game_state.SoundPlaybackOneShot, Volume: 0.5,
	}))
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundHeal, Mode: game_state.SoundPlaybackLoop, Volume: 0.25,
	}))

	assert.Equal(t, []game_state.SoundPlaybackMode{
		game_state.SoundPlaybackOneShot,
		game_state.SoundPlaybackLoop,
	}, seenModes)
	active := activeSoundsByEffect(manager.soundEffectPlayer)
	require.Len(t, active, 2)
	assert.Equal(t, game_state.SoundPlaybackOneShot, active[game_state.SoundCannonFire].Mode)
	cannon := soundEffectMetadataMap[game_state.SoundCannonFire]
	assert.InDelta(t, 0.43*0.5*cannon.VolumeModifier*productionLibrary().recordingCalibration(cannon.RecordingID),
		active[game_state.SoundCannonFire].TargetVolume, 0.0001)
	assert.Equal(t, game_state.SoundPlaybackLoop, active[game_state.SoundHeal].Mode)
	heal := soundEffectMetadataMap[game_state.SoundHeal]
	assert.InDelta(t, 0.43*0.25*heal.VolumeModifier*productionLibrary().recordingCalibration(heal.RecordingID),
		active[game_state.SoundHeal].TargetVolume, 0.0001)
}

func TestFadeOutStartsAtCurrentPlayerVolume(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	clock := int64(100)
	manager.soundEffectPlayer.nowMillis = func() int64 { return clock }

	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1.0, FadeInMs: 1000,
	}))
	clock = 600
	manager.Update()
	active := activeSoundsByEffect(manager.soundEffectPlayer)
	halfFadeInVolume := active[game_state.SoundEarthquake].Volume
	require.Greater(t, halfFadeInVolume, 0.0)

	manager.StopSound(game_state.SoundStopRequest{Effect: game_state.SoundEarthquake, FadeOutMs: 1000})
	active = activeSoundsByEffect(manager.soundEffectPlayer)
	assert.Equal(t, halfFadeInVolume, active[game_state.SoundEarthquake].FadeOutVolume)
	assert.Zero(t, active[game_state.SoundEarthquake].FadeInDuration)

	clock = 1100
	manager.Update()
	active = activeSoundsByEffect(manager.soundEffectPlayer)
	assert.InDelta(t, halfFadeInVolume*0.5, active[game_state.SoundEarthquake].Volume, 0.0001)

	clock = 1600
	manager.Update()
	assert.Zero(t, manager.soundEffectPlayer.GetActiveSoundCount())
}

func TestLoopRemainsActiveUntilStoppedWhileOneShotCleansUp(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.ensureAssetsLoaded())

	for _, mode := range []game_state.SoundPlaybackMode{
		game_state.SoundPlaybackOneShot,
		game_state.SoundPlaybackLoop,
	} {
		require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
			Effect: game_state.SoundEarthquake, Mode: mode, Volume: 1.0, FadeInMs: 0,
		}))
	}
	require.Equal(t, 2, manager.soundEffectPlayer.GetActiveSoundCount())
	manager.soundEffectPlayer.mutex.Lock()
	for _, sound := range manager.soundEffectPlayer.activeSounds {
		if sound.Mode == game_state.SoundPlaybackOneShot {
			sound.Player.Pause() // Simulate the finite stream reaching completion.
		}
	}
	manager.soundEffectPlayer.mutex.Unlock()
	manager.Update()
	require.Equal(t, 1, manager.soundEffectPlayer.GetActiveSoundCount())

	active := activeSoundsByEffect(manager.soundEffectPlayer)
	require.Len(t, active, 1)
	assert.Equal(t, game_state.SoundPlaybackLoop, active[game_state.SoundEarthquake].Mode)
	manager.StopSound(game_state.SoundStopRequest{Effect: game_state.SoundEarthquake, FadeOutMs: 0})
	assert.Zero(t, manager.soundEffectPlayer.GetActiveSoundCount())
}

func TestConcurrentFirstUseLoadsAssetsUnderExclusiveLock(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })

	const callers = 8
	errors := make(chan error, callers)
	var waitGroup sync.WaitGroup
	for range callers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			errors <- manager.ensureAssetsLoaded()
		}()
	}
	waitGroup.Wait()
	close(errors)
	for loadErr := range errors {
		require.NoError(t, loadErr)
	}
	assert.True(t, manager.assetsLoaded)
}

func TestMissingSoundAssetPolicy(t *testing.T) {
	configuration, err := NewAudioConfiguration()
	require.NoError(t, err)
	player, err := NewSoundEffectPlayerWithContext(configuration, getSharedAudioContext())
	require.NoError(t, err)

	for effect, metadata := range soundEffectMetadataMap {
		if isPlaceholderSound(effect) {
			assert.NotPanics(t, func() { player.loadConfiguredSoundAsset(effect, metadata) })
			assert.NotContains(t, player.soundAssets, effect)
		}
	}
	require.Error(t, player.loadSoundAsset(game_state.SoundEffect(-2), "assets/sounds/missing_test_asset.wav"))
}

func TestAmbientVolumeUsesAmbientCategory(t *testing.T) {
	manager := &AudioManager{config: &AudioConfiguration{
		SoundEffectVolume: 0.9,
		AmbientVolume:     0.2,
	}}
	ambient := &AmbientSoundManager{audioManager: manager}

	assert.InDelta(t, 0.1, ambient.effectiveVolume(0.5), 0.0001)
	assert.NotEqual(t, manager.config.GetEffectiveVolume(0.5), ambient.effectiveVolume(0.5))
}

func TestNonLoopingAmbientSoundHonorsAmbientMuteOnStart(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.SetAmbientVolume(0))
	definition := AmbientSoundDefinition{
		SoundType:   AmbientWater,
		FileName:    "assets/sounds/magic/healing_01.wav",
		MaxDistance: 10,
		BaseVolume:  1,
		LoopSound:   false,
		FadeInTime:  0,
		FadeOutTime: 0,
	}

	manager.ambientSounds.createAmbientSound(definition, references.Position{}, 0, AmbientWater)
	instance := manager.ambientSounds.activeSounds[AmbientWater]
	require.NotNil(t, instance)
	require.NotNil(t, instance.Player)
	assert.Zero(t, instance.Player.Volume())
}

func activeSoundsByEffect(player *SoundEffectPlayer) map[game_state.SoundEffect]*ActiveSound {
	player.mutex.RLock()
	defer player.mutex.RUnlock()
	active := make(map[game_state.SoundEffect]*ActiveSound, len(player.activeSounds))
	for _, sound := range player.activeSounds {
		copy := *sound
		active[sound.Effect] = &copy
	}

	return active
}
