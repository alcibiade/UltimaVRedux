package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

type projectBackend struct{ players []*conformanceDevice }

func (*projectBackend) SampleRate() int { return EditorSampleRate }
func (backend *projectBackend) NewPlayer(source io.ReadSeeker) (PlayerTransport, error) {
	player := &conformanceDevice{source: source, barrierEntered: make(chan struct{})}
	backend.players = append(backend.players, player)
	return player, nil
}

// No Redux IDs, routing targets, recordings or original installation enter this fixture.
func neutralProject(t *testing.T, gain float64) ([]byte, fstest.MapFS) {
	t.Helper()
	file := "clips/pulse.wav"
	provenance := RecordingProvenance{OriginalFilename: "pulse.wav", SourceType: "generated", SourceURL: "unknown",
		SourceCreator: "project test generator", License: "MIT", ImportedFrom: "generated in memory", Processing: []ProcessingStep{}}
	document := LibraryDocument{SchemaVersion: 3,
		Recordings: map[string]Recording{"pulse": {File: &file, Provenance: provenance}},
		Cues:       map[string]Cue{"panel/accept": {Name: "Accept", RecordingID: "pulse", VolumeModifier: gain, HowToHear: "Accept a panel."}},
		Taxonomy: Taxonomy{RecordingFamilies: map[string][]string{"signal": {"pulse"}},
			CueProfiles: map[string][]string{"panel": {"panel/accept"}}, Profiles: map[string]CueClassification{
				"panel": {TargetAcousticFamily: "signal", Phase: "select", Outcome: "accepted", MixRole: "interface", Emitters: []string{"panel"}, Contexts: []string{"menu"}},
			}},
	}
	data, err := json.MarshalIndent(document, "", "  ")
	require.NoError(t, err)
	return data, fstest.MapFS{"media/" + file: &fstest.MapFile{Data: editorTestWAV(), Mode: 0o644}}
}

func TestIndependentLibrariesFreezeAssetsMetadataAndSelection(t *testing.T) {
	data, files := neutralProject(t, 0.5)
	first, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media"})
	require.NoError(t, err)
	for offset := 44; offset+1 < len(files["media/clips/pulse.wav"].Data); offset += 2 {
		if binary.LittleEndian.Uint16(files["media/clips/pulse.wav"].Data[offset:]) != 0 {
			binary.LittleEndian.PutUint16(files["media/clips/pulse.wav"].Data[offset:], 4000)
		}
	}
	secondData, _ := neutralProject(t, 0.25)
	second, err := NewLibrary(secondData, files, LibraryOptions{AssetRoot: "media"})
	require.NoError(t, err)
	view := first.Document()
	cue := view.Cues["panel/accept"]
	cue.VolumeModifier = 0
	view.Cues["panel/accept"] = cue
	clear(files["media/clips/pulse.wav"].Data)
	clear(data)
	backends := []*projectBackend{{}, {}}
	runtimes := make([]*Runtime, 2)
	var decoded [][]byte
	for index, library := range []*Library{first, second} {
		runtimes[index], err = NewRuntime(library, backends[index], func() time.Time { return time.Time{} }, 1)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, runtimes[index].Close()) })
		voice, err := runtimes[index].Play(CueRequest{CueID: "panel/accept", Volume: 1})
		require.NoError(t, err)
		require.NotEqual(t, VoiceHandle{}, voice)
		pcm, err := backends[index].players[0].read(EditorSampleRate * decodedPCMBytesPerFrame)
		require.ErrorIs(t, err, io.EOF)
		require.NotEqual(t, make([]byte, len(pcm)), pcm, "caller filesystem mutation cannot rewrite a frozen library")
		decoded = append(decoded, pcm)
	}
	require.NotEqual(t, decoded[0], decoded[1], "same-named paths in independent projects must resolve to their own bytes")
	require.InDelta(t, 0.5, backends[0].players[0].volume, 1e-9)
	require.InDelta(t, 0.25, backends[1].players[0].volume, 1e-9)
	require.NoError(t, runtimes[0].Close())
	_, err = runtimes[1].Play(CueRequest{CueID: "panel/accept", Loop: true, Volume: 1})
	require.NoError(t, err, "closing one project cannot close its sibling")
}

func TestNeutralProjectNeverConsultsProductionLibrary(t *testing.T) {
	const marker = "U5_AUDIO_NEUTRAL_CHILD"
	if os.Getenv(marker) == "" {
		command := exec.Command(os.Args[0], "-test.run=^TestNeutralProjectNeverConsultsProductionLibrary$", "-test.count=1")
		command.Env = append(os.Environ(), marker+"=1")
		output, err := command.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return
	}
	productionLibrary = func() *audioLibrary { panic("neutral project opened Redux metadata") }
	defaultRuntimeLibrary = func() *Library { panic("neutral project opened production assets") }
	data, files := neutralProject(t, 1)
	library, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media"})
	require.NoError(t, err)
	backend := new(projectBackend)
	runtime, err := NewRuntime(library, backend, func() time.Time { return time.Time{} }, 1)
	require.NoError(t, err)
	owner := runtime.NewVoiceOwner()
	_, err = owner.PlayCue(CueRequest{CueID: "panel/accept", Loop: true, Volume: 0.5})
	require.NoError(t, err)
	runtime.Update()
	_, err = runtime.Play(CueRequest{CueID: "not/known", Volume: 1})
	require.Error(t, err)
	require.NoError(t, owner.Close())
	require.NoError(t, runtime.Close())
}

func TestNeutralAuthoringUsesExplicitRootAndSharedRegionTransformation(t *testing.T) {
	data, files := neutralProject(t, 0.5)
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "media", "clips"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "media", "clips", "pulse.wav"), files["media/clips/pulse.wav"].Data, 0o644))
	manifest := filepath.Join(root, "library.json")
	require.NoError(t, os.WriteFile(manifest, data, 0o644))
	_, err := OpenLibraryEditor(manifest)
	require.Error(t, err, "Redux completeness is not weakened by the generic entry point")
	editor, err := OpenLibraryProject(manifest, EditorOptions{AssetRoot: "media", ImportDirectory: "imported"})
	require.NoError(t, err)
	require.NoError(t, editor.Save())
	unchanged, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.Equal(t, data, unchanged)
	path, err := editor.RecordingPath("pulse")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "media", "clips", "pulse.wav"), path)
	clip, err := LoadRecording(path)
	require.NoError(t, err)
	recording := editor.Recording("pulse")
	calibration := -6.0
	recording.CalibrationGainDB = &calibration
	require.NoError(t, editor.SetRecording("pulse", recording))
	draft, err := editor.BeginRecordingRegion("pulse", clip)
	require.NoError(t, err)
	window := &PlaybackWindow{StartMs: 20, EndMs: 60}
	preview, err := draft.Preview(window)
	require.NoError(t, err)
	require.NoError(t, editor.ApplyRecordingRegion(context.Background(), draft, window))
	require.True(t, editor.Undo())
	require.True(t, editor.Redo())
	require.NoError(t, editor.Save())
	published, err := os.ReadFile(manifest)
	require.NoError(t, err)
	library, err := NewLibrary(published, os.DirFS(root), LibraryOptions{AssetRoot: "media"})
	require.NoError(t, err)
	backend := new(projectBackend)
	runtime, err := NewRuntime(library, backend, func() time.Time { return time.Time{} }, 1)
	require.NoError(t, err)
	defer runtime.Close()
	_, err = runtime.Play(CueRequest{CueID: "panel/accept", Volume: 1})
	require.NoError(t, err)
	actual, err := backend.players[0].read(EditorSampleRate * decodedPCMBytesPerFrame)
	require.ErrorIs(t, err, io.EOF)
	stream, err := clip.Stream(preview.Window, false)
	require.NoError(t, err)
	want, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Equal(t, want, actual)
	require.InDelta(t, 0.5*preview.Gain, backend.players[0].volume, 1e-9)
	routing := editor.Routing()
	selected := "panel/accept"
	routing.Music = map[string]*string{"hangar": &selected}
	require.NoError(t, editor.SetRouting(routing))
	require.NoError(t, editor.Save(), "empty neutral routing groups can be authored without corrupting JSON")
	reopened, err := OpenLibraryProject(manifest, EditorOptions{AssetRoot: "media"})
	require.NoError(t, err)
	require.Equal(t, selected, *reopened.Routing().Music["hangar"])
	// Import outside the asset root, then preserve its receipt across Undo.
	external := filepath.Join(t.TempDir(), "replacement.wav")
	require.NoError(t, os.WriteFile(external, editorTestWAV(), 0o644))
	require.NoError(t, editor.ImportRecording("pulse", external))
	require.True(t, editor.Undo())
	receipts := editor.RetainedImports()
	require.Len(t, receipts, 1)
	require.False(t, receipts[0].Referenced)
	require.Contains(t, receipts[0].Path, filepath.Join("media", "imported"))
}

func TestLibraryPolicyAndAssetPathsAreProjectSpecific(t *testing.T) {
	data, files := neutralProject(t, 1)
	_, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media", Validate: ReduxLibraryValidator})
	require.Error(t, err)
	for _, root := range []string{"../media", "/media", "media\\clips"} {
		_, err := NewLibrary(data, files, LibraryOptions{AssetRoot: root})
		require.Error(t, err)
	}
	for _, path := range []string{"../outside.wav", "/absolute.wav", "clips\\pulse.wav"} {
		invalid := bytes.ReplaceAll(data, []byte("clips/pulse.wav"), []byte(path))
		_, err := NewLibrary(invalid, files, LibraryOptions{AssetRoot: "media"})
		require.Error(t, err)
	}
	called := false
	library, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media", Validate: func(document LibraryDocument) error {
		called = true
		delete(document.Cues, "panel/accept") // Validators cannot mutate the parsed snapshot.
		return nil
	}})
	require.NoError(t, err)
	require.True(t, called)
	require.Contains(t, library.Document().Cues, "panel/accept")
}
