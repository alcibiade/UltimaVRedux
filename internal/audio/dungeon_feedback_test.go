package audio

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestDungeonFeedbackCuesResolveToIntentionalSilentSlots(t *testing.T) {
	library := productionLibrary()
	for _, test := range []struct {
		effect                  game_state.SoundEffect
		symbol, name, recording string
		classification          CueClassification
	}{
		{
			effect: game_state.SoundDispel, symbol: "SoundDispel", name: "Dispel Field", recording: "field_dispel_01",
			classification: CueClassification{
				TargetAcousticFamily: "magic", Phase: "resolution", Outcome: "success",
				MixRole: "foreground_feedback", Emitters: []string{"player"},
				Contexts: []string{"combat", "exploration", "interaction", "spellcasting"},
			},
		},
		{
			effect: game_state.SoundElectricalDamage, symbol: "SoundElectricalDamage",
			name: "Electrical Injury", recording: "electrical_injury_01",
			classification: CueClassification{
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
			require.Empty(t, library.recordingPath(test.recording))
			require.True(t, isPlaceholderSound(test.effect))
			require.Equal(t, test.classification.TargetAcousticFamily, library.Taxonomy.recordingFamily(test.recording))
			profile := library.Taxonomy.Profiles[library.Taxonomy.cueProfile(cueID)]
			require.Equal(t, test.classification, profile)
			// No device is needed: intentional silence resolves before backend
			// preparation, voice creation or variant/instance advancement.
			player := &SoundEffectPlayer{assetsLoaded: true, library: defaultRuntimeLibrary()}
			request := game_state.SoundPlaybackRequest{
				Effect: test.effect, Mode: game_state.SoundPlaybackOneShot, Volume: 1,
			}
			asset, skip, err := player.assetForRequest(request)
			require.NoError(t, err)
			require.True(t, skip)
			require.Nil(t, asset)
			for range 3 {
				require.NoError(t, player.Play(request))
			}
			require.Empty(t, player.activeSounds)
			require.Empty(t, player.soundAssets)
			require.Empty(t, player.nextVariant)
			require.Zero(t, player.nextSoundSequence)
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
	require.Equal(t, CalibrationManual, recording.CalibrationMode())
	require.NotEqual(t, "unknown", recording.Provenance.License)
	require.False(t, isPlaceholderSound(game_state.SoundAbsorb))
	clip, err := LoadRecording(library.recordingPath(metadata.RecordingID))
	require.NoError(t, err)
	_, err = clip.SuggestCalibration(recording.PlaybackWindow, "magic", nil)
	require.NoError(t, err, "Absorb retains a decodable, non-silent recording")
}

func TestDungeonFeedbackSlotsExposeSearchAndAuditionGuidance(t *testing.T) {
	editor, _ := editorFixture(t)
	for _, test := range []struct {
		cue, recording string
		terms          []string
	}{
		{"sfx/SoundDispel", "field_dispel_01", []string{"sceptre", "an grav", "field removal", "fizzle", "dungeon"}},
		{"sfx/SoundElectricalDamage", "electrical_injury_01", []string{"electrical field", "shock", "injury", "dungeon"}},
	} {
		for _, term := range test.terms {
			require.Contains(t, strings.ToLower(editor.SearchText(test.cue, true)), term)
			require.Contains(t, strings.ToLower(editor.SearchText(test.recording, false)), term,
				"recording search inherits its cue's intended trigger")
		}
		require.Equal(t, RecordingNoFile, editor.FileAvailability(test.recording).State)
	}
}
