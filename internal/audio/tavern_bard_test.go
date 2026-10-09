package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestTavernBardsHaveDedicatedSilentSlotAndCampingKeepsRecording(t *testing.T) {
	library := productionLibrary()
	recording := library.Recordings["tavern_lute_01"]
	require.Nil(t, recording.File)
	require.Equal(t, unknownRecordingProvenance(), recording.Provenance)
	require.Contains(t, recording.Note, "user will supply")
	for _, cue := range []string{string(AmbientBard), "sfx/SoundTavernBardMusic"} {
		require.Equal(t, "tavern_lute_01", library.Cues[cue].RecordingID)
	}
	require.Equal(t, string(AmbientBard), library.Routing.Emitters["bard"])
	require.Equal(t, "bard_lute_01", library.Cues["sfx/SoundBardLute"].RecordingID)
	require.NotNil(t, library.Recordings["bard_lute_01"].File)

	manager, err := NewAudioManager()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	policy, score, _ := newPolicyTest(t)
	manager.musicManager.policy = policy
	require.NoError(t, manager.PlayMusicForLocation(references.Britain))
	manager.Update()
	require.True(t, score.playing)
	definition := manager.AmbientEmitterDefinition("bard")
	require.Empty(t, definition.FileName)
	require.Equal(t, 10.0, definition.MaxDistance)
	require.Equal(t, 800*time.Millisecond, definition.FadeInTime)
	require.Equal(t, time.Second, definition.FadeOutTime)
	source := AmbientSoundSource{
		Emitter: "bard", SoundType: definition.SoundType, IsActive: true, IsDynamic: true,
		Position: references.Position{}, SoundFile: definition.FileName, MaxDistance: definition.MaxDistance,
		baseVolume: definition.BaseVolume, VolumeModifier: 1,
		FadeInTime: definition.FadeInTime, FadeOutTime: definition.FadeOutTime,
	}
	manager.UpdateAmbientSoundsWithDynamicSources(source.Position, nil, []AmbientSoundSource{source})
	require.Empty(t, manager.ambientSounds.activeSounds)
	require.False(t, manager.ambientSounds.hasBardMusic(), "silence must not suppress the score")
	manager.Update()
	require.True(t, score.playing, "an active but unrecorded tavern bard cannot claim local score ownership")
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundTavernBardMusic))
	require.NotContains(t, activeSoundsByEffect(manager.soundEffectPlayer), game_state.SoundTavernBardMusic)
	require.False(t, manager.soundEffectPlayer.hasInstrumentMusic())
	manager.Update()
	require.True(t, score.playing, "the silent direct tavern SFX cannot claim local score ownership either")
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundBardLute))
	require.Contains(t, activeSoundsByEffect(manager.soundEffectPlayer), game_state.SoundBardLute)
}
