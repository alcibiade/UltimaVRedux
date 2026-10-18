package audio

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNeutralAmbientCycleReturnsValidationError(t *testing.T) {
	data, files := neutralProject(t, 1)
	var document LibraryDocument
	require.NoError(t, json.Unmarshal(data, &document))
	document.Recordings["other"] = document.Recordings["pulse"]
	cue := document.Cues["panel/accept"]
	document.Cues["panel/accept"] = cue.WithRecordings([]string{"pulse", "other"})
	document.Taxonomy.RecordingFamilies["signal"] = []string{"pulse", "other"}
	document.Routing.Ambient = map[string]AmbientProfile{"panel/accept": {RadiusTiles: 1}}
	data, err := json.Marshal(document)
	require.NoError(t, err)
	_, err = NewLibrary(data, files, LibraryOptions{AssetRoot: "media"})
	require.ErrorIs(t, err, ErrAudioLibrary)
	require.ErrorContains(t, err, "ambient profile panel/accept requires one recording")
}

func TestNeutralColdMutedCueDoesNotPrepareEncodedAudio(t *testing.T) {
	for _, mute := range []string{"request", "category", "cue"} {
		t.Run(mute, func(t *testing.T) {
			cueGain, categoryGain, requestGain := 1.0, 1.0, 1.0
			switch mute {
			case "request":
				requestGain = 0
			case "category":
				categoryGain = 0
			case "cue":
				cueGain = 0
			}
			data, files := neutralProject(t, cueGain)
			files["media/clips/pulse.wav"].Data = []byte("not a WAV")
			library, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media"})
			require.NoError(t, err)
			backend := new(projectBackend)
			runtime, err := NewRuntime(library, backend, func() time.Time { return time.Time{} }, categoryGain)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, runtime.Close()) })
			voice, err := runtime.Play(CueRequest{CueID: "panel/accept", Volume: requestGain})
			require.NoError(t, err)
			require.Equal(t, VoiceHandle{}, voice)
			require.Empty(t, backend.players)
			require.Empty(t, runtime.manager.soundEffectPlayer.assetErrors)
			_, err = runtime.Play(CueRequest{CueID: "unknown", Volume: 0})
			require.Error(t, err, "mute must not hide invalid cue identities")
		})
	}
}

func TestNeutralReplacementRequiresExplicitOwner(t *testing.T) {
	data, files := neutralProject(t, 1)
	library, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media"})
	require.NoError(t, err)
	backend := new(projectBackend)
	runtime, err := NewRuntime(library, backend, func() time.Time { return time.Time{} }, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtime.Close()) })
	request := CueRequest{CueID: "panel/accept", Volume: 1, ReplacePrevious: true}
	_, err = runtime.Play(request)
	require.Error(t, err, "independent playback cannot promise owner-local replacement")
	require.Empty(t, backend.players)
	owner := runtime.NewVoiceOwner()
	_, err = owner.PlayCue(request)
	require.NoError(t, err)
	_, err = owner.PlayCue(request)
	require.NoError(t, err)
	require.Equal(t, 1, runtime.manager.soundEffectPlayer.GetActiveSoundCount())
	require.NoError(t, owner.Close())
}
