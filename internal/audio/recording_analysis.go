package audio

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

// Editor analysis/audition uses fixed-rate stereo; runtime adapters decode at
// the host context's rate. Authored windows remain source-relative milliseconds.
const (
	EditorSampleRate   = 44100
	WaveformBucketMs   = 10
	maxEditorFileBytes = 128 << 20
	signedPCMScale     = 32768.0
)

// RecordingAudio is immutable decoded metadata plus encoded bytes. Each player
// gets an independent decoder. Peak buckets preserve transients in both channels.
type RecordingAudio struct {
	Path        string
	DurationMs  int64
	Peaks       []float64
	data        []byte
	loudness    []loudnessBucket
	fingerprint [sha256.Size]byte
	fileInfo    os.FileInfo
}

// LoadRecording can run outside the UI loop. It streams analysis rather than
// retaining the entire decoded PCM of a long music track.
func LoadRecording(path string) (*RecordingAudio, error) {
	return LoadRecordingContext(context.Background(), path)
}

// LoadRecordingContext cancels obsolete waveform requests during fast browsing.
func LoadRecordingContext(ctx context.Context, path string) (*RecordingAudio, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load recording canceled: %w", err)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve recording path: %w", err)
	}
	data, info, err := readRecordingSource(path)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load recording canceled: %w", err)
	}
	stream, err := DecodeAudioStream(data, path, EditorSampleRate)
	if err != nil {
		return nil, err
	}
	length, err := decodedStreamLength(stream)
	if err != nil {
		return nil, err
	}
	peaks, loudness, err := recordingAnalysis(ctx, stream)
	if err != nil {
		return nil, err
	}

	return &RecordingAudio{Path: path, data: data, Peaks: peaks, loudness: loudness,
		DurationMs:  length * millisecondsPerSecond / (EditorSampleRate * decodedPCMBytesPerFrame),
		fingerprint: sha256.Sum256(data), fileInfo: info}, nil
}

func readRecordingSource(path string) ([]byte, os.FileInfo, error) {
	initialInfo, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect recording: %w", err)
	}
	if !initialInfo.Mode().IsRegular() || initialInfo.Size() > maxEditorFileBytes {
		return nil, nil, fmt.Errorf("%w: select a regular recording no larger than 128 MiB", ErrAudioLibrary)
	}
	file, err := os.Open(path) //nolint:gosec // Explicit local recording chosen in the authoring tool.
	if err != nil {
		return nil, nil, fmt.Errorf("open recording: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logging.Warn(logging.CategoryAudio, "Close analyzed recording: %v", err)
		}
	}()
	data, err := io.ReadAll(io.LimitReader(file, maxEditorFileBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read recording: %w", err)
	}
	if len(data) > maxEditorFileBytes {
		return nil, nil, fmt.Errorf("%w: recording exceeds 128 MiB", ErrAudioLibrary)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("inspect recording source: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() != int64(len(data)) || !os.SameFile(initialInfo, info) ||
		!initialInfo.ModTime().Equal(info.ModTime()) {
		return nil, nil, fmt.Errorf("%w: recording changed during loading", ErrRecordingSourceChanged)
	}

	return data, info, nil
}

func recordingAnalysis(ctx context.Context, stream io.Reader) ([]float64, []loudnessBucket, error) {
	buffer := make([]byte, EditorSampleRate*WaveformBucketMs/millisecondsPerSecond*decodedPCMBytesPerFrame)
	var peaks []float64
	var loudness []loudnessBucket
	filters := kWeightingFilters()
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("analyze waveform: %w", err)
		}
		count, err := io.ReadFull(stream, buffer)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, nil, fmt.Errorf("analyze waveform: %w", err)
		}
		peak := 0.0
		bucket := loudnessBucket{frames: count / decodedPCMBytesPerFrame, energy: 0, weighted: 0, peak: 0}
		const stereoChannels = 2
		const sampleBytes = 2
		for offset := 0; offset+1 < count; offset += 2 {
			//nolint:gosec // PCM bit reinterpretation of an unsigned little-endian word as signed 16-bit audio.
			sample := int16(binary.LittleEndian.Uint16(buffer[offset:]))
			peak = math.Max(peak, math.Abs(float64(sample)/signedPCMScale))
			value := float64(sample) / signedPCMScale
			channel := (offset / sampleBytes) % stereoChannels
			weighted := filters[channel][1].sample(filters[channel][0].sample(value))
			bucket.energy += value * value / stereoChannels // Stereo mean energy for RMS.
			bucket.weighted += weighted * weighted          // BS.1770 sums channel energy.
		}
		if count > 0 {
			peaks = append(peaks, peak)
			bucket.peak = peak
			loudness = append(loudness, bucket)
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return peaks, loudness, nil
		}
	}
}

// Stream uses the exact bounded decoder used by production music/ambience/SFX.
func (clip *RecordingAudio) Stream(window *PlaybackWindow, loop bool) (io.ReadSeeker, error) {
	return decodeRecording(clip.data, clip.Path, EditorSampleRate, window, loop)
}

// SuggestedWindow proposes -40 dB boundaries with one bucket of edge padding.
// It never changes the document. All-silent files have no useful suggestion.
func (clip *RecordingAudio) SuggestedWindow() (*PlaybackWindow, bool) {
	const threshold = 0.01
	first, last := -1, -1
	for index, peak := range clip.Peaks {
		if peak >= threshold {
			if first == -1 {
				first = index
			}
			last = index
		}
	}
	if first < 0 {
		return nil, false
	}
	start := int64(max(0, first-1)) * WaveformBucketMs
	const endPaddingBuckets = 2 // End of the last audible bucket plus one padding bucket.
	end := min(clip.DurationMs, int64(last+endPaddingBuckets)*WaveformBucketMs)

	return &PlaybackWindow{StartMs: start, EndMs: end}, end > start
}
