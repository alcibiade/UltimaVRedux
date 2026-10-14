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

func TestManagerServiceAdvancesAmbientWithoutNewSourceSnapshots(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	now := time.Unix(100, 0)
	ambient := manager.ambientSounds
	ambient.now = func() time.Time { return now }
	ambient.AddCustomAmbientSound(indexes.FireLogs, AmbientSoundDefinition{
		SoundType: AmbientFire, FileName: "assets/sounds/magic/healing_01.wav", BaseVolume: 1,
		MaxDistance: 8, LoopSound: true, FadeInTime: time.Second, FadeOutTime: time.Second,
	})
	position := references.Position{}
	manager.UpdateAmbientSounds(position, []TileWithPosition{{Tile: &references.Tile{Index: indexes.FireLogs}}})
	voice := ambient.activeSounds[AmbientFire]
	require.NotNil(t, voice)
	now = now.Add(time.Second / 2)
	manager.Update()
	require.InDelta(t, 0.5, voice.FadeGain, 1e-9)
	manager.UpdateAmbientSounds(position, nil) // Explicit departure, then no more discovery calls.
	now = now.Add(time.Second)
	manager.Update()
	require.Empty(t, ambient.activeSounds)
}

func TestFailedSoundPreparationDoesNotEvictOrReplaceExistingVoice(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "budget", true: "same effect"}[replacement], func(t *testing.T) {
			manager, err := NewAudioManager()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, manager.Close()) })
			player := manager.soundEffectPlayer
			effect, count := game_state.SoundHeal, 8
			if replacement {
				effect, count = game_state.SoundWalkMetal1, 1
			}
			for range count {
				require.NoError(t, manager.PlaySoundEffect(effect))
			}
			before := make(map[string]*ActiveSound)
			for id, voice := range player.activeSounds {
				before[id] = voice
			}
			sequence, variant := player.nextSoundSequence, player.nextVariant[effect]
			for _, asset := range player.soundAssets[effect] {
				asset.StreamFactory = func(game_state.SoundPlaybackMode) (io.ReadSeeker, error) { return nil, errSoundStreamUnavailable }
			}
			require.Error(t, manager.PlaySoundEffect(effect))
			require.Equal(t, before, player.activeSounds)
			require.Equal(t, sequence, player.nextSoundSequence)
			require.Equal(t, variant, player.nextVariant[effect])
		})
	}
}
