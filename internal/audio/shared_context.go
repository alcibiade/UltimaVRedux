package audio

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

var (
	sharedContext *audio.Context
	contextMutex  sync.Mutex
)

// getSharedAudioContext returns a shared audio context to avoid multiple context creation
func getSharedAudioContext() *audio.Context {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	if sharedContext == nil {
		sharedContext = audio.NewContext(44100)
	}

	return sharedContext
}
