package audio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

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
