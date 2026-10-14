package audio

import (
	"io"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type soakBackend struct{ active, peak, created, retired atomic.Int64 }

func (*soakBackend) SampleRate() int { return EditorSampleRate }
func (backend *soakBackend) NewPlayer(source io.ReadSeeker) (PlayerTransport, error) {
	count := backend.active.Add(1)
	backend.peak.Store(max(count, backend.peak.Load()))
	backend.created.Add(1)
	return &soakTransport{conformanceDevice: conformanceDevice{source: source, barrierEntered: make(chan struct{})}, backend: backend}, nil
}

type soakTransport struct {
	conformanceDevice
	backend *soakBackend
}

func (player *soakTransport) PauseAndStopReading() {
	player.conformanceDevice.PauseAndStopReading()
	if player.barriers.Load() == 1 {
		player.backend.active.Add(-1)
		player.backend.retired.Add(1)
	}
}

func TestRuntimeLongSessionLifecycleAndRetainedHeapPlateau(t *testing.T) {
	data, files := neutralProject(t, 1)
	library, err := NewLibrary(data, files, LibraryOptions{AssetRoot: "media"})
	require.NoError(t, err)
	backend := new(soakBackend)
	engine, err := NewRuntime(library, backend, func() time.Time { return time.Time{} }, 1)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engine.Close()) })
	cycle := func(count int) {
		for range count {
			owner := engine.NewVoiceOwner()
			for range 4 {
				_, err := owner.PlayCue(CueRequest{CueID: "panel/accept", Loop: true, Volume: 0.5})
				require.NoError(t, err)
			}
			for range 12 {
				_, err := engine.Play(CueRequest{CueID: "panel/accept", Volume: 0.5})
				require.NoError(t, err)
			}
			require.NoError(t, owner.Close())
			engine.manager.soundEffectPlayer.StopAll()
		}
	}
	cycle(50)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	cycle(1000)
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	require.Equal(t, backend.created.Load(), backend.retired.Load())
	require.Zero(t, backend.active.Load())
	require.LessOrEqual(t, backend.peak.Load(), int64(9), "eight admitted voices plus one prepared replacement")
	require.Empty(t, engine.Snapshot().Voices)
	require.Len(t, engine.manager.soundEffectPlayer.cueAssets, 1, "prepared factories are per cue, not per play")
	const heapTolerance = 2 * 1024 * 1024
	require.LessOrEqual(t, int64(after.HeapAlloc)-int64(before.HeapAlloc), int64(heapTolerance))
	t.Logf("synthetic lifecycle soak: created=%d retired=%d peak prepared+active=%d retained heap delta=%d bytes",
		backend.created.Load(), backend.retired.Load(), backend.peak.Load(), int64(after.HeapAlloc)-int64(before.HeapAlloc))
}
