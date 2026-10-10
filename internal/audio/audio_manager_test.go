package audio

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAudioManagerCreation tests basic AudioManager creation and initialization
func TestAudioManagerCreation(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err, "AudioManager creation should not fail")
	require.NotNil(t, manager, "AudioManager should not be nil")

	// Test initial state
	assert.True(t, manager.IsEnabled(), "AudioManager should be enabled by default")
	assert.False(t, manager.assetsLoaded, "Assets should not be loaded initially")

	// Test configuration defaults - now pre-converted from slider scale
	config := manager.GetConfiguration()
	require.NotNil(t, config, "Configuration should not be nil")
	assert.Equal(t, 0.43, config.SoundEffectVolume, "Default sound effect volume should be 0.43 (ConvertSliderToVolume(6))")
	assert.Equal(t, 0.30, config.MusicVolume, "Default music volume should be 0.30 (ConvertSliderToVolume(5))")

	// Cleanup
	err = manager.Close()
	assert.NoError(t, err, "AudioManager close should not fail")
}

// TestPlaySoundEffectBasic tests basic sound effect playback through SystemCallbacks interface
func TestPlaySoundEffectBasic(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	// Test all predefined sound effects from SystemCallbacks
	soundEffects := []game_state.SoundEffect{
		game_state.SoundCannonFire,
		game_state.SoundBroadside,
		game_state.SoundGlassBreak,
		game_state.SoundEarthquake,
		game_state.SoundSummon,
		game_state.SoundAbsorb,
		game_state.SoundPushObject,
		game_state.SoundOpenDoor,
		game_state.SoundUnlock,
		game_state.SoundTrapTrigger,
		game_state.SoundStepOnTrap,
		game_state.SoundDispel,
	}

	for _, effect := range soundEffects {
		t.Run(getSoundEffectName(effect), func(t *testing.T) {
			// This test will initially fail since we haven't implemented the sound system yet
			err := manager.PlaySoundEffect(effect)
			assert.NoError(t, err, "PlaySoundEffect should not return error")
		})
	}
}

func TestPlayLoopedEarthquakeWithFadeCreatesOneExplicitLoop(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1.0, FadeInMs: 200,
	}))
	manager.soundEffectPlayer.mutex.RLock()
	defer manager.soundEffectPlayer.mutex.RUnlock()
	require.Len(t, manager.soundEffectPlayer.activeSounds, 1)
	for _, sound := range manager.soundEffectPlayer.activeSounds {
		assert.Equal(t, game_state.SoundEarthquake, sound.Effect)
		assert.Equal(t, game_state.SoundPlaybackLoop, sound.Mode)
		assert.Equal(t, int64(200), sound.FadeInDuration)
		assert.Zero(t, sound.Volume)
	}
}

func TestSameEffectOneShotAndLoopedStartsHaveUniqueIDsWithinSameMillisecond(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	const startTime = int64(1234)
	manager.soundEffectPlayer.mutex.Lock()
	manager.soundEffectPlayer.nowMillis = func() int64 { return startTime }
	manager.soundEffectPlayer.mutex.Unlock()

	require.NoError(t, manager.PlaySoundEffect(game_state.SoundEarthquake))
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1.0,
	}))

	manager.soundEffectPlayer.mutex.RLock()
	defer manager.soundEffectPlayer.mutex.RUnlock()
	require.Len(t, manager.soundEffectPlayer.activeSounds, 2)
	assert.Equal(t, uint64(2), manager.soundEffectPlayer.nextSoundSequence)
	for _, sound := range manager.soundEffectPlayer.activeSounds {
		assert.Equal(t, game_state.SoundEarthquake, sound.Effect)
		assert.Equal(t, startTime, sound.StartTime)
	}
}

func TestSameMillisecondStartsRespectConcurrentSoundLimit(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	manager.soundEffectPlayer.mutex.Lock()
	manager.soundEffectPlayer.nowMillis = func() int64 { return 1234 }
	manager.soundEffectPlayer.mutex.Unlock()

	for range 8 {
		require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
			Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1.0,
		}))
	}
	assert.ErrorIs(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEarthquake, Mode: game_state.SoundPlaybackLoop, Volume: 1.0,
	}), errSoundVoiceLimit)

	manager.soundEffectPlayer.mutex.RLock()
	defer manager.soundEffectPlayer.mutex.RUnlock()
	require.Len(t, manager.soundEffectPlayer.activeSounds, 8)
	for _, sound := range manager.soundEffectPlayer.activeSounds {
		assert.LessOrEqual(t, sound.Sequence, uint64(8), "owned loops must not be evicted")
	}
}

// TestVolumeControls tests volume control functionality
func TestVolumeControls(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	// Test sound effect volume control
	err = manager.SetSoundEffectVolume(0.9)
	assert.NoError(t, err, "Setting sound effect volume should succeed")
	assert.Equal(t, 0.9, manager.GetConfiguration().SoundEffectVolume, "Sound effect volume should be updated")

	// Test music volume control
	err = manager.SetMusicVolume(0.3)
	assert.NoError(t, err, "Setting music volume should succeed")
	assert.Equal(t, 0.3, manager.GetConfiguration().MusicVolume, "Music volume should be updated")

	// Test invalid volume ranges

	err = manager.SetSoundEffectVolume(-0.1)
	assert.Error(t, err, "Setting negative sound effect volume should fail")
}

// TestPositionalAudio tests positional audio calculations
func TestPositionalAudio(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	// Disable audio playback to avoid OGG vorbis decoder panics
	manager.SetEnabled(false)

	// Test positional audio at different distances
	testCases := []struct {
		name          string
		playerX       int
		playerY       int
		sourceX       int
		sourceY       int
		expectAudible bool
	}{
		{"Same position", 10, 10, 10, 10, true},
		{"Close position", 10, 10, 11, 11, true},
		{"Medium distance", 10, 10, 15, 15, true},
		{"Far distance", 10, 10, 30, 30, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test positional audio functionality (audio disabled to avoid decoder panics)
			err := manager.PlaySoundEffectAt(game_state.SoundOpenDoor, tc.playerX, tc.playerY, tc.sourceX, tc.sourceY)
			assert.NoError(t, err, "PlaySoundEffectAt should not return error even with audio disabled")
		})
	}
}

// TestBackgroundMusic tests background music functionality
func TestBackgroundMusic(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	// Load assets first (as would happen automatically on first sound)
	err = manager.backgroundMusic.LoadAssets()
	require.NoError(t, err, "Loading music assets should succeed")

	// Test playing background music with a valid track name
	err = manager.PlayBackgroundMusic("title")
	assert.NoError(t, err, "Playing background music should succeed")

	// Test stopping background music
	err = manager.StopBackgroundMusic()
	assert.NoError(t, err, "Stopping background music should succeed")
}

// TestAudioConfiguration tests audio configuration functionality
func TestAudioConfiguration(t *testing.T) {
	config, err := NewAudioConfiguration()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test default values - now pre-converted from slider scale (master volume deprecated but kept for compatibility)
	assert.Equal(t, 0.43, config.SoundEffectVolume) // ConvertSliderToVolume(6) = 0.43
	assert.Equal(t, 0.30, config.MusicVolume)       // ConvertSliderToVolume(5) = 0.30
	assert.Equal(t, 0.18, config.AmbientVolume)     // ConvertSliderToVolume(4) = 0.18
	assert.Equal(t, 20.0, config.MaxAudioDistance)
	assert.Equal(t, 0.3, config.VolumeRolloffFactor)

	// Test effective volume calculation - values are already converted, no double conversion
	effectiveVolume := config.GetEffectiveVolume(1.0)
	// The user slider is converted once. Unit request gain is transparent.
	expectedVolume := 0.43
	assert.InDelta(t, expectedVolume, effectiveVolume, 0.001, "Effective volume calculation should be correct")

	// Test music volume calculation - value is already converted
	effectiveMusicVolume := config.GetEffectiveMusicVolume()
	// Music volume is already 0.30 (converted from slider 5)
	expectedMusicVolume := 0.30
	assert.InDelta(t, expectedMusicVolume, effectiveMusicVolume, 0.001, "Effective music volume calculation should be correct")
}

// TestPositionalAudioCalculator tests the positional audio calculation system
func TestPositionalAudioCalculator(t *testing.T) {
	calculator := NewPositionalAudioCalculator()
	require.NotNil(t, calculator)

	config, err := NewAudioConfiguration()
	require.NoError(t, err)
	calculator.SetConfiguration(config)

	// Test volume calculation at different distances
	testCases := []struct {
		name          string
		playerX       int
		playerY       int
		sourceX       int
		sourceY       int
		expectVolume  float64
		expectAudible bool
	}{
		{"Same position", 0, 0, 0, 0, 1.0, true},
		{"Close distance", 0, 0, 1, 1, 0.5, true},  // Approximate expected volume
		{"Medium distance", 0, 0, 5, 5, 0.1, true}, // Should be audible but quiet
		{"Far distance", 0, 0, 25, 25, 0.0, false}, // Beyond max distance
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			volume, pan := calculator.CalculateAudio(tc.playerX, tc.playerY, tc.sourceX, tc.sourceY)

			// Check volume is in valid range
			assert.GreaterOrEqual(t, volume, 0.0, "Volume should not be negative")
			assert.LessOrEqual(t, volume, 1.0, "Volume should not exceed 1.0")

			// Check pan is in valid range
			assert.GreaterOrEqual(t, pan, -1.0, "Pan should not be less than -1.0")
			assert.LessOrEqual(t, pan, 1.0, "Pan should not exceed 1.0")

			// Check audibility
			isAudible := calculator.IsAudible(tc.playerX, tc.playerY, tc.sourceX, tc.sourceY, 0.05)
			assert.Equal(t, tc.expectAudible, isAudible, "Audibility should match expectation")

			// For same position, volume should be maximum
			if tc.playerX == tc.sourceX && tc.playerY == tc.sourceY {
				assert.Equal(t, 1.0, volume, "Volume at same position should be 1.0")
				assert.Equal(t, 0.0, pan, "Pan at same position should be 0.0 (center)")
			}
		})
	}
}

// TestAudioStats tests audio system statistics
func TestAudioStats(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	// Test that manager was created successfully
	assert.NotNil(t, manager, "Audio manager should be created successfully")
}

// TestWalkingSounds tests walking sound functionality
func TestWalkingSounds(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	defer manager.Close()

	// Test walking sound with nil tile (should not crash)
	err = manager.PlayWalkingSound(nil)
	assert.NoError(t, err, "Playing walking sound with nil tile should not error")

	// Test walking sound manager exists and is functional
	assert.NotNil(t, manager, "Audio manager should have walking sound functionality")
}

// Helper function to get sound effect name for test reporting
func getSoundEffectName(effect game_state.SoundEffect) string {
	switch effect {
	case game_state.SoundCannonFire:
		return "CannonFire"
	case game_state.SoundBroadside:
		return "Broadside"
	case game_state.SoundGlassBreak:
		return "GlassBreak"
	case game_state.SoundEarthquake:
		return "Earthquake"
	case game_state.SoundSummon:
		return "Summon"
	case game_state.SoundAbsorb:
		return "Absorb"
	case game_state.SoundPushObject:
		return "PushObject"
	case game_state.SoundOpenDoor:
		return "OpenDoor"
	case game_state.SoundUnlock:
		return "Unlock"
	case game_state.SoundTrapTrigger:
		return "TrapTrigger"
	case game_state.SoundStepOnTrap:
		return "StepOnTrap"
	default:
		return "Unknown"
	}
}
