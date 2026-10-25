package audio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcousticFamiliesRemainIndependentOfGameplayContextAndPlaybackCategory(t *testing.T) {
	library := productionLibrary()
	for _, test := range []struct {
		cue, family, category, role string
	}{
		{"ambient/AmbientFire", "fire", catalogAmbience, "source_loop"},
		{"sfx/SoundFireCrackle", "fire", catalogSFX, "source_loop"},
		{"sfx/SoundFireBurn", "fire", catalogSFX, "foreground_feedback"},
		{"sfx/SoundEarthquake", "impact", catalogSFX, "foreground_feedback"},
		{"sfx/SoundShardRitualEarthquake", "impact", catalogSFX, "source_loop"},
		{"sfx/SoundApparitionAppears", "magic", catalogSFX, "foreground_feedback"},
		{"sfx/SoundLevelUp", "magic", catalogSFX, "stinger"},
		{"ambient/AmbientMagical", "magic", catalogAmbience, "source_loop"},
	} {
		t.Run(test.cue, func(t *testing.T) {
			profile := library.Taxonomy.Profiles[cueProfile(library.Taxonomy, test.cue)]
			require.Equal(t, test.family, profile.TargetAcousticFamily)
			require.Equal(t, test.category, cueCategory(test.cue))
			require.Equal(t, test.role, profile.MixRole)
		})
	}
}

func TestPreparedRecordingSlotsHaveConsistentIntendedFamilies(t *testing.T) {
	library := productionLibrary()
	for cueID, cue := range library.Cues {
		profile := library.Taxonomy.Profiles[cueProfile(library.Taxonomy, cueID)]
		for _, recordingID := range cue.Recordings() {
			if library.Recordings[recordingID].File == nil {
				require.Equal(t, profile.TargetAcousticFamily, recordingFamily(library.Taxonomy, recordingID),
					"empty slot %s should describe the intended sound for %s", recordingID, cueID)
			}
		}
	}
}
