package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func calibrationTone(t *testing.T, amplitudes ...float64) *RecordingAudio {
	t.Helper()
	pcm := make([]byte, len(amplitudes)*EditorSampleRate*decodedPCMBytesPerFrame)
	for second, amplitude := range amplitudes {
		for frame := range EditorSampleRate {
			value := int16(amplitude * 32767 * math.Sin(2*math.Pi*1000*float64(frame)/EditorSampleRate))
			offset := (second*EditorSampleRate + frame) * decodedPCMBytesPerFrame
			binary.LittleEndian.PutUint16(pcm[offset:], uint16(value))
			binary.LittleEndian.PutUint16(pcm[offset+2:], uint16(value))
		}
	}
	peaks, loudness, err := recordingAnalysis(context.Background(), bytes.NewReader(pcm))
	require.NoError(t, err)

	return &RecordingAudio{DurationMs: int64(len(amplitudes) * 1000), Peaks: peaks, loudness: loudness}
}

func TestCalibrationMeasuresSelectedRegionAndPreservesRelativeLevels(t *testing.T) {
	clip := calibrationTone(t, 0.1, 0.2, 0)
	quiet, err := clip.SuggestCalibration(&PlaybackWindow{StartMs: 0, EndMs: 1000}, "footstep", nil)
	require.NoError(t, err)
	loud, err := clip.SuggestCalibration(&PlaybackWindow{StartMs: 1000, EndMs: 2000}, "footstep", nil)
	require.NoError(t, err)
	require.InDelta(t, -23.0103, quiet.MeasuredDB, 0.03)
	require.InDelta(t, 6.0206, loud.MeasuredDB-quiet.MeasuredDB, 0.02)
	require.InDelta(t, -6.0206, loud.GainDB-quiet.GainDB, 0.02)
	require.InDelta(t, quiet.TargetDB, quiet.MeasuredDB+quiet.GainDB, 0.01)
	_, err = clip.SuggestCalibration(&PlaybackWindow{StartMs: 2000, EndMs: 3000}, "footstep", nil)
	require.ErrorContains(t, err, "silent")
}

func TestCalibrationGatedLoudnessAndSilence(t *testing.T) {
	clip := calibrationTone(t, 0, 0.1, 0.1, 0.1, 0)
	measured, err := clip.SuggestCalibration(&PlaybackWindow{StartMs: 1100, EndMs: 2900}, "score", nil)
	require.NoError(t, err)
	require.Contains(t, measured.Method, "LUFS")
	// A dual-mono 1 kHz sine at -20 dBFS peak measures about -20 LUFS
	// after K weighting, channel summation, and the BS.1770 offset.
	require.InDelta(t, -20, measured.MeasuredDB, 0.15)
	padded, err := clip.SuggestCalibration(nil, "score", nil)
	require.NoError(t, err)
	require.InDelta(t, measured.MeasuredDB, padded.MeasuredDB, 0.6)
	short, err := clip.SuggestCalibration(&PlaybackWindow{StartMs: 1000, EndMs: 1100}, "score", nil)
	require.NoError(t, err)
	require.Contains(t, short.Method, "RMS", "short clips cannot use a 400 ms integrated block")
}

func TestCalibrationPeakAndBoostLimits(t *testing.T) {
	clip := calibrationTone(t, 0.01)
	value, err := clip.SuggestCalibration(nil, "footstep", nil)
	require.NoError(t, err)
	require.Equal(t, 12.0, value.GainDB)
	require.True(t, value.Limited)
	// A rare transient must still constrain gain even when absent from P95.
	clip.loudness[50].peak = 0.9
	value, err = clip.SuggestCalibration(nil, "footstep", nil)
	require.NoError(t, err)
	require.InDelta(t, -3, value.PeakDB+value.GainDB, 0.0001)
	require.True(t, value.Limited)
	badTarget := math.NaN()
	_, err = clip.SuggestCalibration(nil, "footstep", &badTarget)
	require.Error(t, err)
}

func TestCalibrationMetadataValidationAndAutoRoundTrip(t *testing.T) {
	editor, _ := editorFixture(t)
	recording := editor.Recording("healing_01")
	recording.CalibrationPolicy = CalibrationAuto
	require.Error(t, editor.SetRecording("healing_01", recording), "mode requires an authored gain")
	gain, target := -4.0, -24.0
	recording.CalibrationGainDB, recording.CalibrationTargetDB = &gain, &target
	recording.CalibrationPolicy = "typo"
	require.Error(t, editor.SetRecording("healing_01", recording))
	recording.CalibrationPolicy = CalibrationAuto
	target = 0
	require.Error(t, editor.SetRecording("healing_01", recording))
	target = -24
	require.NoError(t, editor.SetRecording("healing_01", recording))
	require.NoError(t, editor.Save())
	reopened, err := OpenLibraryEditor(editor.Path())
	require.NoError(t, err)
	require.Equal(t, recording, reopened.Recording("healing_01"))
	require.Equal(t, CalibrationAuto, reopened.Recording("healing_01").CalibrationMode())
}

func TestManualCalibrationSurvivesSaveImportAndDuplicate(t *testing.T) {
	editor, wav := editorFixture(t)
	recording := editor.Recording("healing_01")
	gain := 2.5
	recording.CalibrationGainDB = &gain // Existing numeric values are implicitly manual.
	require.NoError(t, editor.SetRecording("healing_01", recording))
	require.Equal(t, CalibrationManual, editor.Recording("healing_01").CalibrationMode())
	require.NoError(t, editor.Save())
	reopened, err := OpenLibraryEditor(editor.Path())
	require.NoError(t, err)
	require.Equal(t, CalibrationManual, reopened.Recording("healing_01").CalibrationMode())
	path := filepath.Join(t.TempDir(), "replacement.wav")
	require.NoError(t, os.WriteFile(path, wav, 0o644))
	require.NoError(t, reopened.ImportRecording("healing_01", path))
	require.Equal(t, gain, *reopened.Recording("healing_01").CalibrationGainDB)
	require.NoError(t, reopened.DuplicateClip("healing_01", "new_clip", PlaybackWindow{StartMs: 20, EndMs: 60}, ""))
	require.Equal(t, gain, *reopened.Recording("new_clip").CalibrationGainDB)
	require.Equal(t, CalibrationManual, reopened.Recording("new_clip").CalibrationMode())
}

func TestGainStreamBoostPreservesSeekAndOddReads(t *testing.T) {
	pcm := make([]byte, 8)
	for index, sample := range []int16{1000, -2000, 10000, 20000} {
		binary.LittleEndian.PutUint16(pcm[index*2:], uint16(sample))
	}
	stream := new(gainStream)
	stream.source = bytes.NewReader(pcm)
	stream.gain.Store(math.Float64bits(2))
	var result []byte
	for {
		buffer := make([]byte, 3)
		count, err := stream.Read(buffer)
		result = append(result, buffer[:count]...)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}
	for index, sample := range []int16{2000, -4000, 20000, 32767} {
		require.Equal(t, sample, int16(binary.LittleEndian.Uint16(result[index*2:])))
	}
	_, err := stream.Seek(0, io.SeekStart)
	require.NoError(t, err)
	reread, err := io.ReadAll(stream)
	require.NoError(t, err)
	require.Equal(t, result, reread)
}
