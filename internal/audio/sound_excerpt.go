package audio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

const (
	decodedPCMBytesPerFrame = 4 // DecodeAudioStream returns stereo signed 16-bit PCM.
	millisecondsPerSecond   = 1000
)

// PlaybackWindow identifies a finite excerpt within the original recording.
// Bounds are milliseconds from the start, with EndMs exclusive.
type PlaybackWindow struct {
	StartMs int64 `json:"startMs"`
	EndMs   int64 `json:"endMs"`
}

func soundPlaybackWindow(effect game_state.SoundEffect) *PlaybackWindow {
	metadata, exists := GetSoundEffectMetadata(effect)
	if !exists {
		return nil
	}

	// A replacement single-footfall recording must not inherit the old source's trim.
	window := productionLibrary().Recordings[metadata.RecordingID].PlaybackWindow
	if window == nil {
		return nil
	}
	result := *window

	return &result
}

func excerptSoundAsset(
	effect game_state.SoundEffect, stream io.ReadSeeker, sampleRate int, window PlaybackWindow,
) (*SoundAsset, error) {
	excerpt, err := boundedRecording(stream, sampleRate, &window)
	if err != nil {
		return nil, err
	}
	// SFX stream factories are reused. Keep their bounded PCM independent so
	// simultaneous voices do not seek a shared decoder.
	pcm, err := io.ReadAll(excerpt)
	if err != nil {
		return nil, fmt.Errorf("read sound excerpt: %w", err)
	}

	return &SoundAsset{
		Effect:      effect,
		RecordingID: "",
		StreamFactory: func(mode game_state.SoundPlaybackMode) (io.ReadSeeker, error) {
			stream := bytes.NewReader(pcm)
			if mode == game_state.SoundPlaybackLoop {
				return audio.NewInfiniteLoop(stream, int64(len(pcm))), nil
			}

			return stream, nil
		},
		Duration:   int64(len(pcm)) * millisecondsPerSecond / (int64(sampleRate) * decodedPCMBytesPerFrame),
		SampleRate: sampleRate,
	}, nil
}

func decodedStreamLength(stream io.ReadSeeker) (int64, error) {
	length, ok := stream.(interface{ Length() int64 })
	if !ok || length.Length() <= 0 || length.Length()%decodedPCMBytesPerFrame != 0 {
		return 0, fmt.Errorf("%w: recording requires a positive whole-frame decoded PCM length", ErrAudioLibrary)
	}

	return length.Length(), nil
}
