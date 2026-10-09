package audio

import (
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestMetalFootstepUsesOneFiniteFootfallFromFourStepRecording(t *testing.T) {
	// Keep the original multi-event recording as regression input, independent
	// of which replacement a designer eventually assigns to the metal cue.
	const sampleRate = 44100
	path := "assets/sounds/footsteps/stone_steps_01.ogg"
	original, err := soundFiles.ReadFile(path)
	require.NoError(t, err)
	decoded, err := DecodeAudioStream(original, path, sampleRate)
	require.NoError(t, err)
	source, err := io.ReadAll(decoded)
	require.NoError(t, err)
	asset, err := excerptSoundAsset(game_state.SoundWalkMetal1, decoded, sampleRate, PlaybackWindow{StartMs: 40, EndMs: 260})
	require.NoError(t, err)
	pcm, err := io.ReadAll(asset.StreamFactory(game_state.SoundPlaybackOneShot))
	require.NoError(t, err)
	require.Equal(t, int64(220), asset.Duration)
	start := 40 * asset.SampleRate / 1000 * 4
	end := 260 * asset.SampleRate / 1000 * 4
	require.Equal(t, source[start:end], pcm, "play exactly one original footfall, not a sequence")
	require.Equal(t, 4, audibleBursts(source, asset.SampleRate), "source recording contains four separate footsteps")
	require.Equal(t, 1, audibleBursts(pcm, asset.SampleRate), "playback must contain one footstep")
	loop := asset.StreamFactory(game_state.SoundPlaybackLoop)
	twoLoops := make([]byte, len(pcm)*2)
	_, err = io.ReadFull(loop, twoLoops)
	require.NoError(t, err)
	require.Equal(t, pcm, twoLoops[:len(pcm)])
	require.Equal(t, pcm, twoLoops[len(pcm):], "explicit loop mode still loops the selected excerpt")
}

func TestRepeatedMetalMovementCreatesOneVoiceWithoutSuppressingOtherSurfaces(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	require.NoError(t, manager.ensureAssetsLoaded())
	// Prevent wall-clock cleanup from making assertions timing-dependent.
	for _, asset := range manager.soundEffectPlayer.soundAssets[game_state.SoundWalkMetal1] {
		asset.Duration = 60_000
	}
	for range 12 {
		require.NoError(t, manager.PlayWalkingSound(&references.Tile{Index: indexes.MetalFloor}))
		require.Equal(t, 1, manager.soundEffectPlayer.GetActiveSoundCount())
	}
	require.Equal(t, uint64(12), manager.soundEffectPlayer.nextSoundSequence, "one request per step, not a throttle")
	for range 2 {
		require.NoError(t, manager.PlayWalkingSound(&references.Tile{Index: indexes.BrickFloor}))
	}
	require.Equal(t, 3, manager.soundEffectPlayer.GetActiveSoundCount())
	require.NoError(t, manager.PlayWalkingSound(&references.Tile{Index: indexes.MetalFloor}))
	require.Equal(t, 3, manager.soundEffectPlayer.GetActiveSoundCount(), "replace metal only; retain other surfaces")
}

// Count substantial PCM bursts separated by at least 100ms below -30dBFS.
// This measures the actual recording, rather than echoing its excerpt metadata.
func audibleBursts(pcm []byte, sampleRate int) int {
	windowBytes := sampleRate / 100 * 4
	silentWindows, bursts := 10, 0
	for offset := 0; offset+windowBytes <= len(pcm); offset += windowBytes {
		peak := int32(0)
		for i := offset; i < offset+windowBytes; i += 2 {
			sample := int32(int16(binary.LittleEndian.Uint16(pcm[i : i+2])))
			if sample < 0 {
				sample = -sample
			}
			peak = max(peak, sample)
		}
		if peak > 1036 {
			if silentWindows >= 10 {
				bursts++
			}
			silentWindows = 0
		} else {
			silentWindows++
		}
	}
	return bursts
}
