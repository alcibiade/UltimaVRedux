package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestProductionDungeonAmbienceMatchesCaveAndBrickLocations(t *testing.T) {
	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	for _, test := range []struct {
		cue       string
		locations []references.Location
	}{
		{"ambient/dungeon_cave", []references.Location{
			references.Despise, references.Destard, references.Doom, references.Shame, references.Hythloth,
		}},
		{"ambient/dungeon_brick", []references.Location{references.Deceit, references.Wrong, references.Covetous}},
	} {
		for _, location := range test.locations {
			source, exists := manager.DungeonAmbientSource(location, references.Position{X: 2, Y: 3})
			require.True(t, exists, location.String())
			require.Equal(t, AmbientSoundType(test.cue), source.SoundType)
			cue := productionLibrary().Cues[test.cue]
			require.Equal(t, productionLibrary().recordingPath(cue.RecordingID), source.SoundFile,
				"routing follows the assigned recording, including an empty slot")
			require.Zero(t, source.MaxDistance)
			require.Positive(t, source.FadeInTime)
			require.Positive(t, source.FadeOutTime)
		}
	}
	for _, location := range []references.Location{references.BritanniaUnderworld, references.Britain, references.CombatLocation} {
		_, exists := manager.DungeonAmbientSource(location, references.Position{})
		require.False(t, exists, "outside contexts never inherit a dungeon bed")
	}
}

func TestDungeonBedEmptySlotAndAssignedLifecycle(t *testing.T) {
	for _, location := range []references.Location{references.Despise, references.Deceit} {
		t.Run(location.String(), func(t *testing.T) {
			manager, err := NewAudioManager()
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, manager.Close()) })
			ambient := manager.ambientSounds
			now := time.Unix(10, 0)
			ambient.now = func() time.Time { return now }
			require.NoError(t, manager.SetAmbientVolume(1))
			source, exists := manager.DungeonAmbientSource(location, references.Position{})
			require.True(t, exists)
			definition := ambient.soundTypeDefinitions[source.SoundType]
			definition.FileName = ""
			ambient.soundTypeDefinitions[source.SoundType] = definition
			source, exists = manager.DungeonAmbientSource(location, references.Position{})
			require.True(t, exists)
			manager.UpdateAmbientSoundsWithDynamicSources(source.Position, nil, []AmbientSoundSource{source})
			require.Empty(t, ambient.activeSounds, "empty slots must not create voices")
			// Supply an existing fixture only to this manager. The production
			// library stays unchanged when exercising an assigned recording.
			definition.FileName = "assets/sounds/magic/healing_01.wav"
			ambient.soundTypeDefinitions[source.SoundType] = definition
			source, exists = manager.DungeonAmbientSource(location, references.Position{})
			require.True(t, exists)
			update := func() {
				manager.UpdateAmbientSoundsWithDynamicSources(source.Position, nil, []AmbientSoundSource{source})
			}
			update()
			key := AmbientSoundType("background/" + string(source.SoundType))
			voice := ambient.activeSounds[key]
			require.NotNil(t, voice, "assigned recording must create a voice")
			require.Zero(t, voice.Player.Volume())
			now = now.Add(source.FadeInTime)
			// Unchanged listener (turn), movement and tactical-map coordinates must
			// update the same cue-keyed player rather than restarting the bed.
			for _, position := range []references.Position{{}, {X: 7, Y: 7}, {X: 10, Y: 10}} {
				source, exists = manager.DungeonAmbientSource(location, position)
				require.True(t, exists)
				update()
				require.Len(t, ambient.activeSounds, 1)
				require.Same(t, voice, ambient.activeSounds[key])
			}
			require.InDelta(t, source.GetBaseVolume(), voice.Player.Volume(), 1e-9)
			require.NoError(t, manager.SetSoundEffectVolume(0))
			require.Positive(t, voice.Player.Volume(), "bed belongs to Ambience, not Sound Effects")
			require.NoError(t, manager.SetAmbientVolume(0))
			require.Zero(t, voice.Player.Volume())
			update()
			require.Same(t, voice, ambient.activeSounds[key], "mute preserves loop ownership")
			require.NoError(t, manager.SetAmbientVolume(0.5))
			require.InDelta(t, source.GetBaseVolume()*0.5, voice.Player.Volume(), 1e-9)
			manager.UpdateAmbientSoundsWithDynamicSources(source.Position, nil, nil)
			require.Equal(t, AmbientFadeOut, voice.FadeState)
			now = now.Add(source.FadeOutTime / 2)
			manager.UpdateAmbientSoundsWithDynamicSources(source.Position, nil, nil)
			require.InDelta(t, source.GetBaseVolume()*0.25, voice.Player.Volume(), 1e-9)
			now = now.Add(source.FadeOutTime)
			manager.UpdateAmbientSoundsWithDynamicSources(source.Position, nil, nil)
			require.Empty(t, ambient.activeSounds, "leaving retires the bed after its fade")
			update()
			manager.SetEnabled(false)
			update()
			require.Empty(t, ambient.activeSounds, "global disable closes and suppresses the bed")
			manager.SetEnabled(true)
			update()
			require.Len(t, ambient.activeSounds, 1)
		})
	}
}

func TestReachRecordingDecodesNonSilentProductionAudio(t *testing.T) {
	recording := productionLibrary().Recordings["combat_swing_01"]
	require.NotNil(t, recording.File, "reach attacks keep their existing audible swing")
	clip, err := LoadRecording("assets/" + *recording.File)
	require.NoError(t, err)
	_, err = clip.SuggestCalibration(recording.PlaybackWindow, "impact", nil)
	require.NoError(t, err, "the actual playback region must contain audible PCM")
	metadata, ok := GetSoundEffectMetadata(game_state.SoundCombatSwing)
	require.True(t, ok)
	require.Equal(t, "combat_swing_01", metadata.RecordingID)
}
