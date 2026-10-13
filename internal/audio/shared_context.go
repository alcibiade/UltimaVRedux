package audio

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

//nolint:gochecknoglobals // Ebitengine has one process context; serialize default creation by these adapters.
var contextMutex sync.Mutex

// getSharedAudioContext borrows a host-created context, including its sample
// rate. Hosts create their context before constructing audio managers; otherwise
// the first adapter creates the process default. Managers never own this device.
func getSharedAudioContext() *audio.Context {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	if current := audio.CurrentContext(); current != nil {
		return current
	}

	const defaultSampleRate = 44100

	return audio.NewContext(defaultSampleRate)
}
