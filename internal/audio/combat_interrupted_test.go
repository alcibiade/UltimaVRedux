package audio

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestCombatInterruptedCueResolvesToIntentionalSilentSlot(t *testing.T) {
	library := productionLibrary()
	metadata, known := GetSoundEffectMetadata(game_state.SoundCombatInterrupted)
	require.True(t, known)
	require.Equal(t, "combat_interrupted_01", metadata.RecordingID)
	cue := library.Cues["sfx/SoundCombatInterrupted"]
	require.Equal(t, []string{metadata.RecordingID}, cue.Recordings())
	require.Equal(t, cue.Name, metadata.Name)
	require.Positive(t, metadata.VolumeModifier)
	require.Contains(t, cue.HowToHear, "Intentionally empty recording slot")
	recording := library.Recordings[metadata.RecordingID]
	require.Nil(t, recording.File)
	require.Empty(t, library.recordingPath(metadata.RecordingID))
	require.True(t, isPlaceholderSound(game_state.SoundCombatInterrupted))
	require.Equal(t, "status", library.Taxonomy.recordingFamily(metadata.RecordingID))
	profile := library.Taxonomy.Profiles[library.Taxonomy.cueProfile("sfx/SoundCombatInterrupted")]
	require.Equal(t, CueClassification{
		TargetAcousticFamily: "status", Phase: "resolution", Outcome: "failure",
		MixRole: "foreground_feedback", Emitters: []string{"creature"}, Contexts: []string{audioContextCombat},
	}, profile)
	manager, backend, _ := gameAudioLibraryFixture(t, defaultRuntimeLibrary())
	require.NoError(t, manager.PlaySoundEffect(game_state.SoundCombatInterrupted))
	require.Empty(t, manager.sfx.Snapshot())
	require.Empty(t, backend.players)
}
