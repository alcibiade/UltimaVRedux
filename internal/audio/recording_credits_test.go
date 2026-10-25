package audio

import (
	"io/fs"
	"slices"
	"testing"

	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
	"github.com/stretchr/testify/require"
)

func TestRecordingCreditsAreStableDetachedMetadataWithoutCueDuplication(t *testing.T) {
	library := productionLibrary()
	credits := RecordingCredits()
	require.Equal(t, credits, RecordingCredits())
	wantCount := 0
	for _, recording := range library.Recordings {
		if recording.File != nil {
			wantCount++
		}
	}
	require.Len(t, credits, wantCount)
	music := slices.IndexFunc(credits, func(credit RecordingCredit) bool { return credit.Name == "Britannic Lands.mp3" })
	require.NotEqual(t, -1, music)
	require.True(t, credits[music].Music)
	credits[music].Provenance.Processing[0].PerformedBy = "mutated caller copy"
	require.Equal(t, "Brad Hannah", library.Recordings["britannic_lands"].Provenance.Processing[0].PerformedBy)
	require.Equal(t, "Brad Hannah", RecordingCredits()[music].Provenance.Processing[0].PerformedBy)
	// Metadata projection has no dependency on opening, decoding or inspecting
	// the referenced recording; fake paths and duplicate cues need no audio FS.
	file := "sounds/nonexistent.wav"
	fixture := &LibraryDocument{Recordings: map[string]authoring.Recording{
		"present":     {File: &file, Provenance: unknownRecordingProvenance()},
		"placeholder": {Provenance: unknownRecordingProvenance()},
	}, Cues: map[string]authoring.Cue{
		"one": {RecordingID: "present"}, "two": {RecordingID: "present"},
	}}
	projected := recordingCredits(fixture)
	require.Len(t, projected, 1)
	require.Equal(t, "nonexistent.wav", projected[0].Name)
}

func TestRecordingCreditsAccountForBundledLegacyAttributions(t *testing.T) {
	library := productionLibrary()
	registered := make(map[string]bool)
	for _, recording := range library.Recordings {
		if recording.File != nil {
			registered["assets/"+*recording.File] = true
		}
	}
	require.NoError(t, fs.WalkDir(soundFiles, "assets", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && DetectAudioFormat(path) != AudioFormatUnknown {
			require.True(t, registered[path], "bundled recording must retain its attribution in library.json: %s", path)
		}
		return nil
	}))
	require.Contains(t, library.Recordings["legacy_water_01"].Provenance.License, "danielpodlovics")
	require.Equal(t, "HealingSound.wav", library.Recordings["legacy_healing_01"].Provenance.OriginalFilename)
	require.Equal(t, "unknown", library.Recordings["legacy_healing_01"].Provenance.SourceCreator)
	for _, cue := range library.Cues {
		require.NotContains(t, []string{"legacy_water_01", "legacy_healing_01", "legacy_healing_02"}, cue.RecordingID)
	}
}
