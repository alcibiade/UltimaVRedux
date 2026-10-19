package audio

import (
	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// PlaybackWindow is Redux's persisted source-relative authoring window in
// milliseconds (EndMs exclusive). The module projection converts to Duration.
type PlaybackWindow = authoring.PlaybackWindow

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
