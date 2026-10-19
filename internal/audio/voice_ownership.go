package audio

import (
	"math"
	"time"

	berad "github.com/bradhannah/berad-sdk/modules/berad-audio"
)

// GameVoice exposes only cleanup to gameplay; transport state remains presentation-owned.
type GameVoice struct {
	manager *GameAudioManager
	voice   berad.Voice
}

// Stop releases only this instance; zero/stale/closed voices remain safe.
func (voice GameVoice) Stop(milliseconds int64) {
	if voice.manager != nil {
		voice.manager.report(voice.voice.Stop(gameFade(milliseconds)))
	}
}

func gameFade(milliseconds int64) time.Duration {
	const maximum = math.MaxInt64 / int64(time.Millisecond)

	return time.Duration(max(0, min(milliseconds, maximum))) * time.Millisecond
}
