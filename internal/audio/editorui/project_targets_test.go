package editorui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

func TestEditorUsesConsumerTargetsAndVocabulary(t *testing.T) {
	document := audio.LibraryDocument{SchemaVersion: 3,
		Recordings: map[string]audio.Recording{"signal": {Provenance: audio.RecordingProvenance{
			OriginalFilename: "unknown", SourceType: "unknown", SourceURL: "unknown", SourceCreator: "unknown", License: "unknown", ImportedFrom: "unknown", Processing: []audio.ProcessingStep{}}}},
		Cues: map[string]audio.Cue{"panel/accept": {Name: "Accept", RecordingID: "signal", VolumeModifier: 1, HowToHear: "Confirm a panel"}},
		Taxonomy: audio.Taxonomy{RecordingFamilies: map[string][]string{"signal": {"signal"}}, CueProfiles: map[string][]string{"panel": {"panel/accept"}},
			Profiles: map[string]audio.CueClassification{"panel": {TargetAcousticFamily: "signal", Phase: "accept", Outcome: "confirm", MixRole: "interface", Emitters: []string{"panel"}, Contexts: []string{"menu"}}}},
	}
	data, err := json.Marshal(document)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "library.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	targets := audio.AuthoringTargets{Tiles: []audio.TileTarget{{ID: "plate", Name: "Metal plate", Frames: []int{1}}}, Music: map[string]string{"hangar": "Hangar"}, Families: []string{"signal"}, CueCategories: map[string]string{"panel/accept": "music"}}
	editor, err := audio.OpenLibraryProject(path, audio.EditorOptions{Targets: targets})
	require.NoError(t, err)
	targets.Music["hangar"] = "mutated caller label"
	model := newModel(editor)
	t.Cleanup(model.close)
	require.Equal(t, "Hangar", model.targets.Music["hangar"])
	require.Len(t, model.routing.tiles, 1)
	require.Equal(t, "plate", model.routing.tiles[0].ID)
	require.Equal(t, []string{"signal"}, editor.AcousticFamilies())
	require.Equal(t, previewMusic, model.categoryForPreview("signal", "panel/accept"))
	require.Contains(t, model.View(), "signal")
}
