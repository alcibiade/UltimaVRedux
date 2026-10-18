package audio

import (
	"context"
	"io"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRecordingWaveformViewCannotChangeAuthoringSnapshot(t *testing.T) {
	editor, _ := editorFixture(t)
	path, err := editor.RecordingPath("healing_01")
	require.NoError(t, err)
	clip, err := LoadRecording(path)
	require.NoError(t, err)
	require.Equal(t, path, clip.Path())
	require.Equal(t, int64(100), clip.DurationMs())
	window, suggested := clip.SuggestedWindow()
	require.True(t, suggested)
	calibration, err := clip.SuggestCalibration(window, "spell", nil)
	require.NoError(t, err)
	fingerprint := clip.Fingerprint()
	stream, err := clip.Stream(window, false)
	require.NoError(t, err)
	pcm, err := io.ReadAll(stream)
	require.NoError(t, err)

	peaks := clip.Peaks()
	require.NotEmpty(t, peaks)
	peak := clip.PeakBetween(0, clip.DurationMs())
	require.Positive(t, peak)
	clear(peaks)
	require.NotEqual(t, peaks, clip.Peaks())
	require.Equal(t, peak, clip.PeakBetween(0, clip.DurationMs()))
	after, stillSuggested := clip.SuggestedWindow()
	require.True(t, stillSuggested, "editing a waveform view must not change source analysis")
	require.Equal(t, window, after)
	afterCalibration, err := clip.SuggestCalibration(window, "spell", nil)
	require.NoError(t, err)
	require.Equal(t, calibration, afterCalibration)
	require.Equal(t, fingerprint, clip.Fingerprint())
	require.NoError(t, clip.VerifySource(context.Background()))
	stream, err = clip.Stream(window, false)
	require.NoError(t, err)
	afterPCM, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Equal(t, pcm, afterPCM)
}

func TestRecordingPeakRangesUseSourceBuckets(t *testing.T) {
	editor, _ := editorFixture(t)
	path, err := editor.RecordingPath("healing_01")
	require.NoError(t, err)
	clip, err := LoadRecording(path)
	require.NoError(t, err)
	for _, bounds := range [][2]int64{{-20, -1}, {0, 19}, {60, 100}, {101, math.MaxInt64}, {60, 20}} {
		require.Zero(t, clip.PeakBetween(bounds[0], bounds[1]), "bounds %v", bounds)
	}
	for _, bounds := range [][2]int64{{0, 20}, {20, 20}, {20, 60}, {-1, math.MaxInt64}} {
		require.InDelta(t, 16000.0/32768, clip.PeakBetween(bounds[0], bounds[1]), 1e-9, "bounds %v", bounds)
	}
}

func TestRecordingInspectionCancellationIsNotInvalidMedia(t *testing.T) {
	editor, _ := editorFixture(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, expire := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer expire()
	for name, ctx := range map[string]context.Context{"canceled": canceled, "deadline": expired} {
		t.Run(name, func(t *testing.T) {
			status := editor.InspectRecording(ctx, "healing_01")
			require.Equal(t, RecordingChecking, status.State)
			require.True(t, status.Exists)
			_, err := LoadRecordingContext(ctx, "not-opened.wav")
			require.ErrorIs(t, err, ctx.Err())
		})
	}
	require.Equal(t, RecordingReady, editor.InspectRecording(context.Background(), "healing_01").State)
}
