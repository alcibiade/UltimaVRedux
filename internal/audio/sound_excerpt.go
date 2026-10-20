package audio

import (
	"github.com/bradhannah/berad-sdk/modules/berad-audio/authoring"
)

// PlaybackWindow is Redux's persisted source-relative authoring window in
// milliseconds (EndMs exclusive). The module projection converts to Duration.
type PlaybackWindow = authoring.PlaybackWindow
