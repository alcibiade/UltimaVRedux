package audio

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestFireBurnUsesBoundedExcerptWithoutTrimmingAmbientRecording(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.ensureAssetsLoaded())
	library := productionLibrary()
	burn := library.Cues["sfx/SoundFireBurn"].RecordingID
	ambient := library.Cues[string(AmbientFire)].RecordingID
	require.NotEqual(t, ambient, burn, "damage and sustained fire need independent excerpt metadata")
	require.Equal(t, ambient, library.Cues["sfx/SoundFireCrackle"].RecordingID)
	require.Equal(t, library.Recordings[ambient].File, library.Recordings[burn].File, "reuse existing bytes")
	require.Nil(t, library.Recordings[ambient].PlaybackWindow)
	asset := manager.soundEffectPlayer.soundAssets[game_state.SoundFireBurn][0]
	require.Equal(t, int64(1000), asset.Duration)
	pcm, err := io.ReadAll(asset.StreamFactory(game_state.SoundPlaybackOneShot))
	require.NoError(t, err)
	require.Len(t, pcm, asset.SampleRate*decodedPCMBytesPerFrame)
	full, err := manager.ambientSounds.loadAmbientAudioFile(library.recordingPath(ambient))
	require.NoError(t, err)
	fullLength, err := decodedStreamLength(full)
	require.NoError(t, err)
	require.Greater(t, fullLength, int64(33*asset.SampleRate*decodedPCMBytesPerFrame))
	prefix := make([]byte, len(pcm))
	_, err = io.ReadFull(full, prefix)
	require.NoError(t, err)
	require.Equal(t, prefix, pcm, "damage plays only the selected first second of the existing source")
	require.NotEqual(t, make([]byte, len(pcm)), pcm, "the selected excerpt must contain audio")
	require.Equal(t, int64(33378), manager.soundEffectPlayer.soundAssets[game_state.SoundFireCrackle][0].Duration)
	definition, found := manager.GetAmbientSoundDefinition(AmbientFire)
	require.True(t, found)
	require.True(t, definition.LoopSound)
}

func TestFireBurnScheduledReleaseHonorsRequestedFadeIn(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	clock := int64(1000)
	manager.soundEffectPlayer.nowMillis = func() int64 { return clock }
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundFireBurn, Mode: game_state.SoundPlaybackOneShot, Volume: 1, FadeInMs: 1600,
	}))
	clock += 400
	manager.Update()
	require.InDelta(t, 0.25, activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundFireBurn].FadeGain, 1e-9)
	clock += 400
	manager.Update()
	require.InDelta(t, 0.5, activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundFireBurn].FadeGain, 1e-9)
	clock += 100
	manager.Update()
	require.InDelta(t, 0.25, activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundFireBurn].FadeGain, 1e-9)
	clock += 100
	manager.Update()
	require.Zero(t, manager.soundEffectPlayer.GetActiveSoundCount())
}

func TestFireBurnRetriggersOneVoiceAndFadesWithoutStoppingAmbientFire(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	clock := int64(1000)
	manager.soundEffectPlayer.nowMillis = func() int64 { return clock }
	ambientNow := time.Unix(10, 0)
	manager.ambientSounds.now = func() time.Time { return ambientNow }
	position := references.Position{X: 10, Y: 10}
	tiles := []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}, Position: position}}
	manager.ambientSounds.UpdatePlayerPosition(position, tiles)
	ambientNow = ambientNow.Add(time.Second)
	manager.ambientSounds.UpdatePlayerPosition(position, tiles)
	ambient := manager.ambientSounds.activeSounds[AmbientFire]
	require.NotNil(t, ambient)
	require.Positive(t, ambient.Player.Volume())
	require.NoError(t, manager.PlaySound(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundFireCrackle, Mode: game_state.SoundPlaybackLoop, Volume: 1,
	}))
	campfire := activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundFireCrackle]
	for range 12 {
		require.NoError(t, manager.PlaySoundEffect(game_state.SoundFireBurn))
		require.Equal(t, 2, manager.soundEffectPlayer.GetActiveSoundCount(), "one burn plus the owned campfire loop")
	}
	burn := activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundFireBurn]
	require.Equal(t, game_state.SoundPlaybackOneShot, burn.Mode)
	clock += 800
	manager.Update()
	require.InDelta(t, 1, activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundFireBurn].FadeGain, 1e-9)
	clock += 100
	manager.Update()
	require.InDelta(t, 0.5, activeSoundsByEffect(manager.soundEffectPlayer)[game_state.SoundFireBurn].FadeGain, 1e-9)

	// A fresh hazard replaces even a fading voice, receiving its own finite tail.
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundFireBurn))
	require.False(t, burn.Player.IsPlaying())
	clock += 999
	manager.Update()
	require.Contains(t, activeSoundsByEffect(manager.soundEffectPlayer), game_state.SoundFireBurn)
	clock++
	manager.Update()
	active := activeSoundsByEffect(manager.soundEffectPlayer)
	require.NotContains(t, active, game_state.SoundFireBurn)
	require.Same(t, campfire.Player, active[game_state.SoundFireCrackle].Player)
	require.True(t, campfire.Player.IsPlaying())
	require.Same(t, ambient, manager.ambientSounds.activeSounds[AmbientFire])
	require.True(t, ambient.Player.IsPlaying())

	// Normal proximity cleanup is independent of the short damage cue.
	far := references.Position{X: 18, Y: 10}
	manager.ambientSounds.UpdatePlayerPosition(far, tiles)
	ambientNow = ambientNow.Add(800 * time.Millisecond)
	manager.ambientSounds.UpdatePlayerPosition(far, tiles)
	require.NotContains(t, manager.ambientSounds.activeSounds, AmbientFire)
}
