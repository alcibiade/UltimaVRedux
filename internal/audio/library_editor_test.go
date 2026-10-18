package audio

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func editorFixture(t *testing.T) (*LibraryEditor, []byte) {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "assets", "sounds"), 0o755))
	library := freshTestLibrary(t)
	for id, cue := range library.Cues {
		library.Cues[id] = cue.WithRecordings([]string{cue.PrimaryRecording()})
	}
	for id, recording := range library.Recordings {
		recording.File, recording.CalibrationGainDB, recording.PlaybackWindow = nil, nil, nil
		recording.CalibrationPolicy, recording.CalibrationTargetDB = "", nil
		library.Recordings[id] = recording
	}
	file := "sounds/tone.wav"
	recording := library.Recordings["healing_01"]
	recording.File = &file
	library.Recordings["healing_01"] = recording
	wav := editorTestWAV()
	require.NoError(t, os.WriteFile(filepath.Join(root, "assets", file), wav, 0o644))
	manifest, err := json.MarshalIndent(library, "", "  ")
	require.NoError(t, err)
	path := filepath.Join(root, "assets", "library.json")
	require.NoError(t, os.WriteFile(path, manifest, 0o644))
	editor, err := OpenLibraryEditor(path)
	require.NoError(t, err)
	return editor, wav
}

func editorTestWAV() []byte {
	// 100 ms stereo: silence for 20 ms, a pulse in the right channel until
	// 60 ms, then silence. No cancellation from averaging opposite channels.
	pcm := make([]byte, EditorSampleRate/10*decodedPCMBytesPerFrame)
	for frame := EditorSampleRate / 50; frame < EditorSampleRate*3/50; frame++ {
		binary.LittleEndian.PutUint16(pcm[frame*4+2:], 16000)
	}
	var wav bytes.Buffer
	wav.WriteString("RIFF")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(36+len(pcm)))
	wav.WriteString("WAVEfmt ")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(16))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(1))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(2))
	_ = binary.Write(&wav, binary.LittleEndian, uint32(EditorSampleRate))
	_ = binary.Write(&wav, binary.LittleEndian, uint32(EditorSampleRate*4))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(4))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(16))
	wav.WriteString("data")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(len(pcm)))
	wav.Write(pcm)
	return wav.Bytes()
}

func TestEditorRoundTripUndoAndFocusedSave(t *testing.T) {
	e, _ := editorFixture(t)
	original := bytes.Clone(e.saved)
	require.NoError(t, e.Save())
	after, err := os.ReadFile(e.Path())
	require.NoError(t, err)
	require.Equal(t, original, after, "opening and saving must not reformat the manifest")

	cue := e.Cue("sfx/SoundHeal")
	cue.VolumeModifier = 0.25
	require.NoError(t, e.SetCue("sfx/SoundHeal", cue))
	require.True(t, e.Dirty())
	require.True(t, e.Undo())
	require.False(t, e.Dirty())
	require.True(t, e.Redo())
	require.NoError(t, e.Save())
	require.False(t, e.Dirty())
	reopened, err := OpenLibraryEditor(e.Path())
	require.NoError(t, err)
	require.Equal(t, 0.25, reopened.Cue("sfx/SoundHeal").VolumeModifier)
	beforeFields, afterFields := objectSpans(original), objectSpans(e.saved)
	require.Equal(t, beforeFields["recordings"].data, afterFields["recordings"].data)
	beforeCues, afterCues := objectSpans(beforeFields["cues"].data), objectSpans(afterFields["cues"].data)
	for id, span := range beforeCues {
		if id != "sfx/SoundHeal" {
			require.Equal(t, span.data, afterCues[id].data, id)
		}
	}
	require.True(t, e.Undo())
	require.True(t, e.Dirty(), "undo after save is an unsaved authored change")
}

func TestRecordingLicenseCanBeEditedWithoutDecodingAMissingAudioFile(t *testing.T) {
	editor, _ := editorFixture(t)
	recording := editor.Recording("healing_01")
	recording.PlaybackWindow = &PlaybackWindow{StartMs: 0, EndMs: 80}
	require.NoError(t, editor.SetRecording("healing_01", recording))
	path, err := editor.RecordingPath("healing_01")
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))
	require.NoError(t, editor.SetRecordingLicense("healing_01", "  CC0-1.0  "))
	expected := recording
	expected.Provenance.License = "CC0-1.0"
	require.Equal(t, expected, editor.Recording("healing_01"), "only the license changes")
	require.Error(t, editor.SetRecordingLicense("healing_01", "\t  "))
	require.Equal(t, expected, editor.Recording("healing_01"), "invalid edits are atomic")
	require.Error(t, editor.SetRecordingLicense("missing_recording", "CC0-1.0"))
	require.True(t, editor.Undo())
	require.Equal(t, recording, editor.Recording("healing_01"))
}

func TestEditorClassificationSaveUndoAndClipInheritance(t *testing.T) {
	e, _ := editorFixture(t)
	before := bytes.Clone(e.saved)
	require.Equal(t, "magic", e.RecordingFamily("healing_01"))
	require.Equal(t, "interaction_action", e.CueProfile("sfx/SoundPushObject"))
	require.NoError(t, e.SetRecordingFamily("healing_01", "status"))
	require.True(t, e.RecordingChanged("healing_01"))
	require.NoError(t, e.SetCueProfile("sfx/SoundPushObject", "environment_harm"))
	require.True(t, e.CueChanged("sfx/SoundPushObject"))
	require.Error(t, e.SetCueProfile("sfx/SoundPushObject", "not_a_profile"))
	require.True(t, e.Undo())
	require.Equal(t, "interaction_action", e.CueProfile("sfx/SoundPushObject"))
	require.True(t, e.Redo())
	require.NoError(t, e.Save())
	reopened, err := OpenLibraryEditor(e.Path())
	require.NoError(t, err)
	require.Equal(t, "status", reopened.RecordingFamily("healing_01"))
	require.Equal(t, "environment_harm", reopened.CueProfile("sfx/SoundPushObject"))
	originalRoots, savedRoots := objectSpans(before), objectSpans(e.saved)
	require.Equal(t, originalRoots["recordings"].data, savedRoots["recordings"].data)
	require.Equal(t, originalRoots["cues"].data, savedRoots["cues"].data)
	require.NoError(t, reopened.DuplicateClip("healing_01", "healing_03", PlaybackWindow{StartMs: 0, EndMs: 50}, ""))
	require.Equal(t, "status", reopened.RecordingFamily("healing_03"))
}

func TestEditorRejectsInvalidAndExternalEdits(t *testing.T) {
	e, _ := editorFixture(t)
	recording := e.Recording("healing_01")
	recording.PlaybackWindow = &PlaybackWindow{StartMs: 10, EndMs: 1000}
	require.Error(t, e.SetRecording("healing_01", recording))
	require.False(t, e.Dirty())
	recording.PlaybackWindow = nil
	nan := math.NaN()
	recording.CalibrationGainDB = &nan
	require.Error(t, e.SetRecording("healing_01", recording))
	require.False(t, e.Dirty())
	cue := e.Cue("sfx/SoundHeal")
	cue.RecordingID = "missing"
	require.Error(t, e.SetCue("sfx/SoundHeal", cue))
	external := append(bytes.Clone(e.saved), '\n')
	require.NoError(t, os.WriteFile(e.Path(), external, 0o644))
	require.ErrorContains(t, e.Save(), "changed on disk")
	actual, err := os.ReadFile(e.Path())
	require.NoError(t, err)
	require.Equal(t, external, actual)
}

func TestEditorImportAndNewSlotPreserveBytes(t *testing.T) {
	e, wav := editorFixture(t)
	require.NoError(t, e.AddRecording("new_door"))
	require.Error(t, e.AddRecording("new_door"))
	source := filepath.Join(t.TempDir(), "original.wav")
	require.NoError(t, os.WriteFile(source, wav, 0o644))
	require.NoError(t, e.ImportRecording("new_door", source))
	path, err := e.RecordingPath("new_door")
	require.NoError(t, err)
	imported, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, wav, imported)
	require.Equal(t, "original.wav", e.Recording("new_door").Provenance.OriginalFilename)
	require.Equal(t, "unknown", e.Recording("new_door").Provenance.License)
	require.NoError(t, e.Save())
	reopened, err := OpenLibraryEditor(e.Path())
	require.NoError(t, err)
	require.Equal(t, e.Recording("new_door"), reopened.Recording("new_door"))
}

func TestRecordingWaveformAndWindowPlayback(t *testing.T) {
	e, _ := editorFixture(t)
	path, err := e.RecordingPath("healing_01")
	require.NoError(t, err)
	clip, err := LoadRecording(path)
	require.NoError(t, err)
	require.Equal(t, int64(100), clip.DurationMs())
	window, ok := clip.SuggestedWindow()
	require.True(t, ok)
	require.Equal(t, &PlaybackWindow{StartMs: 10, EndMs: 70}, window)
	stream, err := clip.Stream(window, false)
	require.NoError(t, err)
	pcm, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Len(t, pcm, 60*EditorSampleRate*4/1000)
	loop, err := clip.Stream(window, true)
	require.NoError(t, err)
	twice := make([]byte, len(pcm)*2)
	_, err = io.ReadFull(loop, twice)
	require.NoError(t, err)
	require.Equal(t, append(bytes.Clone(pcm), pcm...), twice)
	_, err = stream.Seek(-4, io.SeekStart)
	require.Error(t, err)
	_, err = stream.Seek(0, io.SeekStart)
	require.NoError(t, err)
	replayed, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Equal(t, pcm, replayed)

	recording := e.Recording("healing_01")
	recording.PlaybackWindow = window
	require.NoError(t, e.SetRecording("healing_01", recording))
	for _, cueID := range []string{"music/track/town", "ambient/AmbientForest"} {
		cue := e.Cue(cueID)
		cue.RecordingID = "healing_01"
		require.NoError(t, e.SetCue(cueID, cue))
	}
	require.NoError(t, e.Save(), "music and ambience accept the same validated recording window")
}

type lengthReader struct{ *bytes.Reader }

func (r lengthReader) Length() int64 { return r.Size() }

func TestRecordingWindowBeyondOneMinute(t *testing.T) {
	source := lengthReader{bytes.NewReader(make([]byte, 70*1000*4))}
	window := &PlaybackWindow{StartMs: 65000, EndMs: 66000}
	section, err := boundedRecording(source, 1000, window)
	require.NoError(t, err)
	pcm, err := io.ReadAll(section)
	require.NoError(t, err)
	require.Len(t, pcm, 4000)
}
