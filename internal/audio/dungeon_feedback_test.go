package audio

import (
	"context"
	"testing"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
	"github.com/bradhannah/berad-sdk/modules/berad-audio/backend/ebitengine"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestDungeonFeedbackCuesResolveToIntentionalSilentSlots(t *testing.T) {
	library := productionLibrary()
	for _, test := range []struct {
		effect                  game_state.SoundEffect
		symbol, name, recording string
		classification          authoring.CueClassification
	}{
		{
			effect: game_state.SoundDispel, symbol: "SoundDispel", name: "Dispel Field", recording: "field_dispel_01",
			classification: authoring.CueClassification{
				TargetAcousticFamily: "magic", Phase: "resolution", Outcome: "success",
				MixRole: "foreground_feedback", Emitters: []string{"player"},
				Contexts: []string{"combat", "exploration", "interaction", "spellcasting"},
			},
		},
		{
			effect: game_state.SoundElectricalDamage, symbol: "SoundElectricalDamage",
			name: "Electrical Injury", recording: "electrical_injury_01",
			classification: authoring.CueClassification{
				TargetAcousticFamily: "impact", Phase: "impact", Outcome: "harm",
				MixRole: "foreground_feedback", Emitters: []string{"environment"}, Contexts: []string{"exploration"},
			},
		},
	} {
		t.Run(test.symbol, func(t *testing.T) {
			require.Equal(t, test.symbol, catalogEffectNames()[test.effect])
			metadata, known := GetSoundEffectMetadata(test.effect)
			require.True(t, known)
			require.Equal(t, test.name, metadata.Name)
			require.Equal(t, test.recording, metadata.RecordingID)
			require.Positive(t, metadata.VolumeModifier, "silence comes from the empty slot, not muted gain")
			cueID := "sfx/" + test.symbol
			cue := library.Cues[cueID]
			require.Equal(t, []string{test.recording}, cue.Recordings())
			require.Contains(t, cue.HowToHear, "Intentionally empty recording slot")
			recording := library.Recordings[test.recording]
			require.Nil(t, recording.File)
			require.Nil(t, recording.PlaybackWindow)
			require.Nil(t, recording.CalibrationGainDB)
			require.Nil(t, recording.CalibrationTargetDB)
			require.Empty(t, recording.CalibrationPolicy)
			require.Equal(t, unknownRecordingProvenance(), recording.Provenance)
			require.Empty(t, assetRecordingPath(library, test.recording))
			require.True(t, isPlaceholderSound(test.effect))
			require.Equal(t, test.classification.TargetAcousticFamily, recordingFamily(library.Taxonomy, test.recording))
			profile := library.Taxonomy.Profiles[cueProfile(library.Taxonomy, cueID)]
			require.Equal(t, test.classification, profile)
			// No device is needed: intentional silence resolves before backend
			// preparation, voice creation or variant/instance advancement.
			manager, backend, _ := gameAudioLibraryFixture(t, defaultRuntimeLibrary())
			for range 3 {
				require.NoError(t, manager.PlaySoundEffect(test.effect))
			}
			require.Empty(t, manager.sfx.Snapshot())
			require.Empty(t, backend.players)
		})
	}
}

func TestDispelSlotPreservesAbsorbHealingRecording(t *testing.T) {
	library := productionLibrary()
	metadata, known := GetSoundEffectMetadata(game_state.SoundAbsorb)
	require.True(t, known)
	require.Equal(t, "healing_02", metadata.RecordingID)
	require.Equal(t, []string{"healing_02"}, library.Cues["sfx/SoundAbsorb"].Recordings())
	require.NotEqual(t, library.Cues["sfx/SoundDispel"].RecordingID, metadata.RecordingID)
	recording := library.Recordings[metadata.RecordingID]
	require.NotNil(t, recording.File)
	require.NotNil(t, recording.CalibrationGainDB)
	require.Equal(t, authoring.CalibrationManual, recording.CalibrationPolicy)
	require.NotEqual(t, "unknown", recording.Provenance.License)
	require.False(t, isPlaceholderSound(game_state.SoundAbsorb))
	clip, err := authoring.LoadRecordingContext(context.Background(), assetRecordingPath(library, metadata.RecordingID), ebitengine.Decode)
	require.NoError(t, err)
	_, err = clip.SuggestCalibration(recording.PlaybackWindow, authoring.DefaultCalibrationOptions())
	require.NoError(t, err, "Absorb retains a decodable, non-silent recording")
}
