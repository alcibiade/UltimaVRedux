package audio

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthoredProvenanceSurvivesCatalogAndRenamedPaths(t *testing.T) {
	catalog, err := BuildCatalog(false)
	require.NoError(t, err)
	for _, asset := range catalog.Assets {
		if !asset.Registered {
			continue
		}
		recording := productionLibrary().Recordings[asset.ID]
		require.Equal(t, recording.Provenance, asset.Provenance, asset.ID)
		require.NoError(t, validateRecordingProvenance(asset.ID, asset.Provenance))
		if recording.File == nil {
			require.Equal(t, unknownRecordingProvenance(), asset.Provenance)
		}
	}
	index := slices.IndexFunc(catalog.Assets, func(asset CatalogAsset) bool { return asset.ID == "britannic_lands" })
	require.NotEqual(t, -1, index)
	require.Equal(t, "assets/music/britannic_lands.mp3", catalog.Assets[index].Path)
	require.Equal(t, "Britannic Lands.mp3", catalog.Assets[index].Provenance.OriginalFilename)
	catalog.Assets[index].Provenance.Processing[0].Tool = "changed in detached catalog"
	require.Equal(t, "Audacity", productionLibrary().Recordings["britannic_lands"].Provenance.Processing[0].Tool)
}

func TestCreditMigrationPreservesKnownSourceLinks(t *testing.T) {
	// These are the nine source references from the retired Credit.txt. The
	// source is now attached to recording identity, not the vendor filename.
	groups := map[string][]string{
		"https://opengameart.org/content/grass-foot-step-sounds-yo-frankie": {"footstep_grass_left_01", "footstep_grass_right_01"},
		"https://freesound.org/people/Nagwense/sounds/407655/":              {"footstep_leaves_01", "footstep_leaves_02", "footstep_leaves_03"},
		"https://freesound.org/people/AlexMurphy53/sounds/328459/":          {"footstep_concrete_01", "footstep_concrete_02", "footstep_concrete_03", "footstep_concrete_04"},
		"https://freesound.org/people/Phil25/sounds/208103/":                {"stone_steps_01"},
		"https://freesound.org/people/EminYILDIRIM/sounds/536092/":          {"door_locked_01"},
		"https://freesound.org/people/BenjaminNelan/sounds/321086/":         {"door_open_01"},
		"https://opengameart.org/content/nature-sounds-pack":                {"stream_birds_01", "forest_nature_01"},
		"https://opengameart.org/content/wind1":                             {"wind_01"},
		"https://freesound.org/people/raremess/sounds/222558/":              {"cardboard_burning_01"},
	}
	for source, recordings := range groups {
		for _, id := range recordings {
			provenance := productionLibrary().Recordings[id].Provenance
			require.Equal(t, source, provenance.SourceURL, id)
			require.Equal(t, provenanceUnknown, provenance.License, "a source link cannot establish license terms")
		}
	}
}

func TestMusicRecordsUserConfirmedConversionWithoutClaimingMIDIAuthor(t *testing.T) {
	for _, track := range musicTrackIDs() {
		cue := productionLibrary().Cues["music/track/"+track]
		provenance := productionLibrary().Recordings[cue.RecordingID].Provenance
		require.Equal(t, "midi_conversion", provenance.SourceType)
		require.Equal(t, provenanceUnknown, provenance.SourceURL)
		require.Equal(t, provenanceUnknown, provenance.SourceCreator)
		require.Equal(t, provenanceUnknown, provenance.License)
		require.Len(t, provenance.Processing, 1)
		require.Equal(t, "Audacity", provenance.Processing[0].Tool)
		require.Equal(t, "Brad Hannah", provenance.Processing[0].PerformedBy)
		require.Contains(t, provenance.Processing[0].Description, "MIDI file found online")
	}
}

func TestProvenanceValidationRequiresExplicitUnknownRatherThanBlank(t *testing.T) {
	mutations := map[string]func(*RecordingProvenance){
		"missing object":     func(p *RecordingProvenance) { *p = RecordingProvenance{} },
		"blank original":     func(p *RecordingProvenance) { p.OriginalFilename = "" },
		"blank source":       func(p *RecordingProvenance) { p.SourceURL = " " },
		"blank license":      func(p *RecordingProvenance) { p.License = "" },
		"null processing":    func(p *RecordingProvenance) { p.Processing = nil },
		"partial processing": func(p *RecordingProvenance) { p.Processing = []ProcessingStep{{Tool: "Audacity"}} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			library := freshTestLibrary(t)
			recording := library.Recordings["magic_failure_fizzle_01"]
			mutate(&recording.Provenance)
			library.Recordings["magic_failure_fizzle_01"] = recording
			data, err := json.Marshal(library)
			require.NoError(t, err)
			_, err = parseAudioLibrary(data, soundFiles)
			require.ErrorIs(t, err, ErrAudioLibrary)
		})
	}
}

func TestGuideRendersProvenanceDirectlyFromJSON(t *testing.T) {
	catalog, err := BuildCatalog(false)
	require.NoError(t, err)
	guide := catalog.GenerationGuide()
	require.Contains(t, guide, "## Recording provenance")
	require.Contains(t, guide, "328459__alexmurphy53__walking-on-a-concrete-floor-01.ogg")
	require.Contains(t, guide, "https://freesound.org/people/AlexMurphy53/sounds/328459/")
	require.Contains(t, guide, "midi_conversion")
	require.Contains(t, guide, "Tool: Audacity; by: Brad Hannah.")
	require.Contains(t, guide, "unknown")
}
