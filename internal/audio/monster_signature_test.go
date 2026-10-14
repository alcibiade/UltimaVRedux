package audio

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestMonsterSignatureCuesResolveToSilentCombatSlots(t *testing.T) {
	library := productionLibrary()
	// Exercise slot resolution without decoding recordings or opening an audio device.
	player := &SoundEffectPlayer{assetsLoaded: true, library: defaultRuntimeLibrary()}
	for _, test := range []struct {
		effect                 game_state.SoundEffect
		cue, recording, family string
		phase, outcome         string
	}{
		{game_state.SoundMonsterGate, "sfx/SoundMonsterGate", "monster_gate_01", "magic", "onset", "success"},
		{game_state.SoundMonsterPossession, "sfx/SoundMonsterPossession", "monster_possession_01", "magic", "impact", "harm"},
		{game_state.SoundMonsterTeleport, "sfx/SoundMonsterTeleport", "monster_teleport_01", "magic", "action", "success"},
		{game_state.SoundCorpserCapture, "sfx/SoundCorpserCapture", "corpser_capture_01", "creature", "impact", "harm"},
		{game_state.SoundCorpserRelease, "sfx/SoundCorpserRelease", "corpser_release_01", "creature", "resolution", "success"},
	} {
		t.Run(test.cue, func(t *testing.T) {
			metadata, known := GetSoundEffectMetadata(test.effect)
			require.True(t, known)
			require.Equal(t, test.recording, metadata.RecordingID)
			cue, exists := library.Cues[test.cue]
			require.True(t, exists)
			require.Equal(t, []string{test.recording}, cue.Recordings())
			require.Equal(t, cue.Name, metadata.Name)
			require.Positive(t, metadata.VolumeModifier, "silence comes from the empty slot, not a muted cue")
			recording, exists := library.Recordings[test.recording]
			require.True(t, exists)
			require.Nil(t, recording.File)
			require.Empty(t, library.recordingPath(test.recording))
			require.True(t, isPlaceholderSound(test.effect))

			require.Equal(t, test.family, library.Taxonomy.recordingFamily(test.recording))
			profile, exists := library.Taxonomy.Profiles[library.Taxonomy.cueProfile(test.cue)]
			require.True(t, exists)
			require.Equal(t, CueClassification{
				TargetAcousticFamily: test.family,
				Phase:                test.phase,
				Outcome:              test.outcome,
				MixRole:              "foreground_feedback",
				Emitters:             []string{"creature"},
				Contexts:             []string{audioContextCombat},
			}, profile)

			asset, skip, err := player.assetForRequest(game_state.SoundPlaybackRequest{
				Effect: test.effect, Mode: game_state.SoundPlaybackOneShot, Volume: 1,
			})
			require.NoError(t, err)
			require.True(t, skip, "a known empty slot must resolve to intentional silence")
			require.Nil(t, asset)
		})
	}
	_, skip, err := player.assetForRequest(game_state.SoundPlaybackRequest{
		Effect: game_state.SoundEffect(-1), Mode: game_state.SoundPlaybackOneShot, Volume: 1,
	})
	require.ErrorIs(t, err, errSoundEffectNotFound, "unknown IDs must not resolve as silent slots")
	require.False(t, skip)
}
