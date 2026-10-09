package audio

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEditorCanRepairMissingFilesWithoutWeakeningProductionValidation(t *testing.T) {
	editor, wav := editorFixture(t)
	missing, err := editor.RecordingPath("healing_01")
	require.NoError(t, err)
	require.NoError(t, os.Remove(missing))
	reopened, err := OpenLibraryEditor(editor.Path())
	require.NoError(t, err, "missing files must be repairable in the editor")
	require.Equal(t, RecordingMissing, reopened.FileAvailability("healing_01").State)
	require.Error(t, reopened.Save(), "even a no-op Save must reject broken file references")
	_, err = parseAudioLibrary(editor.saved, editor.files)
	require.Error(t, err, "embedded production loading remains strict")
	cue := reopened.Cue("sfx/SoundHeal")
	cue.VolumeModifier = 0.25
	require.NoError(t, reopened.SetCue("sfx/SoundHeal", cue), "unrelated repairs can be staged before every file is fixed")
	path := filepath.Join(t.TempDir(), "repair.wav")
	require.NoError(t, os.WriteFile(path, wav, 0o644))
	require.NoError(t, reopened.ImportRecording("healing_01", path))
	require.Equal(t, RecordingReady, reopened.InspectRecording(context.Background(), "healing_01").State)
	require.NoError(t, reopened.Save())
	require.False(t, reopened.Dirty())
}

func TestRecordingAvailabilitySeparatesPresenceDecodeAndBounds(t *testing.T) {
	editor, wav := editorFixture(t)
	require.Equal(t, RecordingNoFile, editor.FileAvailability("healing_02").State)
	require.Equal(t, RecordingChecking, editor.FileAvailability("healing_01").State)
	status := editor.InspectRecording(context.Background(), "healing_01")
	require.True(t, status.Exists)
	require.Equal(t, RecordingReady, status.State)
	path, err := editor.RecordingPath("healing_01")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("not a WAV"), 0o644))
	status = editor.InspectRecording(context.Background(), "healing_01")
	require.True(t, status.Exists)
	require.Equal(t, RecordingInvalid, status.State)
	require.Error(t, editor.Save(), "an existing but undecodable file is not valid")
	require.NoError(t, os.WriteFile(path, wav, 0o644))
	broken := editor.library.Recordings["healing_01"]
	broken.PlaybackWindow = &PlaybackWindow{StartMs: 0, EndMs: 1000}
	editor.library = cloneAudioValue(editor.library)
	editor.library.Recordings["healing_01"] = broken
	status = editor.InspectRecording(context.Background(), "healing_01")
	require.True(t, status.Exists)
	require.Equal(t, RecordingInvalid, status.State)
	require.Contains(t, status.Reason, "duration")
}

func TestEditorEntryChangeMarkersFollowUndoAndSave(t *testing.T) {
	editor, _ := editorFixture(t)
	require.False(t, editor.RecordingChanged("healing_01"))
	require.False(t, editor.CueChanged("sfx/SoundHeal"))
	recording := editor.Recording("healing_01")
	recording.Note = "Listening note"
	require.NoError(t, editor.SetRecording("healing_01", recording))
	require.True(t, editor.RecordingChanged("healing_01"))
	require.False(t, editor.CueChanged("sfx/SoundHeal"))
	require.True(t, editor.Undo())
	require.False(t, editor.RecordingChanged("healing_01"))
	require.True(t, editor.Redo())
	require.NoError(t, editor.Save())
	require.False(t, editor.RecordingChanged("healing_01"))
}
