package audio

import (
	"fmt"
	"io"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

const (
	decodedPCMBytesPerFrame = 4
	millisecondsPerSecond   = 1000
)

// PlaybackWindow is Redux's persisted source-relative authoring window in
// milliseconds (EndMs exclusive). The module projection converts to Duration.
type PlaybackWindow struct {
	StartMs int64 `json:"startMs"`
	EndMs   int64 `json:"endMs"`
}

// soundPlaybackWindow is catalog metadata, independent of live playback.
func soundPlaybackWindow(effect game_state.SoundEffect) *PlaybackWindow {
	metadata, exists := GetSoundEffectMetadata(effect)
	if !exists {
		return nil
	}
	window := productionLibrary().Recordings[metadata.RecordingID].PlaybackWindow
	if window == nil {
		return nil
	}
	result := *window

	return &result
}

func decodedStreamLength(stream io.ReadSeeker) (int64, error) {
	length, ok := stream.(interface{ Length() int64 })
	if !ok || length.Length() <= 0 || length.Length()%decodedPCMBytesPerFrame != 0 {
		return 0, fmt.Errorf("%w: recording requires a positive whole-frame decoded PCM length", ErrAudioLibrary)
	}

	return length.Length(), nil
}
