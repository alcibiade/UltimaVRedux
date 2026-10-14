package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPlaybackSnapshotReportsActualGainsWithoutServicing(t *testing.T) {
	data, files := neutralProject(t, 0.5)
	library, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media"})
	require.NoError(t, err)
	now, calls := time.Unix(100, 0), 0
	backend := new(projectBackend)
	runtime, err := NewRuntime(library, backend, func() time.Time { calls++; return now }, 0.2)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtime.Close()) })
	_, err = runtime.Play(CueRequest{CueID: "panel/accept", Volume: 0.5, Loop: true, FadeInMs: 1000})
	require.NoError(t, err)
	now = now.Add(500 * time.Millisecond)
	runtime.Update()
	beforeCalls := calls
	snapshot := runtime.Snapshot()
	require.Equal(t, beforeCalls, calls, "diagnostics cannot advance the presentation clock")
	require.Len(t, snapshot.Voices, 1)
	voice := snapshot.Voices[0]
	require.Equal(t, "panel/accept", voice.CueID)
	require.Equal(t, "pulse", voice.RecordingID)
	require.Equal(t, "media/clips/pulse.wav", voice.Asset)
	require.Equal(t, "playing", voice.State)
	require.InDelta(t, 0.5, voice.Gain.Cue, 1e-9)
	require.InDelta(t, 0.5, voice.Gain.Request, 1e-9)
	require.InDelta(t, 0.2, voice.Gain.Category, 1e-9)
	require.InDelta(t, 0.5, voice.Gain.Envelope, 1e-9)
	require.InDelta(t, 0.025, voice.Gain.Effective, 1e-9)
	require.Contains(t, snapshot.String(), "panel/accept -> pulse")
	snapshot.Voices[0].RecordingID = "changed by reader"
	require.Equal(t, "pulse", runtime.Snapshot().Voices[0].RecordingID)
	require.NoError(t, runtime.Close())
	closed := runtime.Snapshot()
	require.True(t, closed.Closed)
	require.Empty(t, closed.Voices)
}
