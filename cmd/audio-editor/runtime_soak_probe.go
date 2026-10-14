package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

const (
	nativeSoakDuration      = time.Minute
	nativeSoakWarmup        = 5 * time.Second
	nativeSoakCadence       = 200 * time.Millisecond
	nativeSoakVoices        = 4
	nativeSoakHeapTolerance = 8 * 1024 * 1024
)

type runtimeSoakState struct {
	owner     *audio.VoiceOwner
	nextCycle time.Time
	cycles    int
	baseline  uint64
	warmed    bool
}

func (probe *runtimeProbeGame) updateSoak() error {
	now := time.Now()
	if probe.started.IsZero() {
		probe.started = now
	}
	probe.runtime.Update()
	snapshot := probe.runtime.Snapshot()
	for _, voice := range snapshot.Voices {
		if voice.Failure != "" {
			return fmt.Errorf("%w: native soak: %s", audio.ErrAudioLibrary, voice.Failure)
		}
		probe.advanced = probe.advanced || voice.PositionMs > 0
	}
	elapsed := now.Sub(probe.started)
	if elapsed >= nativeSoakDuration {
		return probe.finishSoak()
	}
	if !probe.soak.warmed && elapsed >= nativeSoakWarmup {
		runtime.GC()
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		probe.soak.baseline, probe.soak.warmed = memory.HeapAlloc, true
	}
	if now.Before(probe.soak.nextCycle) {
		return nil
	}
	if probe.soak.owner != nil {
		if err := probe.soak.owner.Close(); err != nil {
			return fmt.Errorf("retire native soak cycle: %w", err)
		}
	}
	probe.soak.owner = probe.runtime.NewVoiceOwner()
	for range nativeSoakVoices {
		request := audio.CueRequest{CueID: probe.cue, Loop: true, Volume: 1, FadeInMs: 0, ReplacePrevious: false}
		_, err := probe.soak.owner.PlayCue(request)
		if err != nil {
			return fmt.Errorf("start native soak cycle: %w", err)
		}
	}
	probe.soak.cycles++
	probe.soak.nextCycle = now.Add(nativeSoakCadence)

	return nil
}

func (probe *runtimeProbeGame) finishSoak() error {
	if !probe.advanced || probe.soak.owner == nil {
		return fmt.Errorf("%w: native soak device did not advance", audio.ErrAudioLibrary)
	}
	if err := probe.soak.owner.Close(); err != nil {
		return fmt.Errorf("finish native soak: %w", err)
	}
	probe.runtime.Update()
	if len(probe.runtime.Snapshot().Voices) != 0 {
		return fmt.Errorf("%w: native soak retained active voices", audio.ErrAudioLibrary)
	}
	runtime.GC()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	if memory.HeapAlloc > probe.soak.baseline+nativeSoakHeapTolerance {
		return fmt.Errorf("%w: native soak retained heap grew beyond %d bytes",
			audio.ErrAudioLibrary, nativeSoakHeapTolerance)
	}
	logging.Info(logging.CategoryAudio, "Native soak: cycles=%d transports=%d baseline_heap=%d final_heap=%d active=0",
		probe.soak.cycles, probe.soak.cycles*nativeSoakVoices, probe.soak.baseline, memory.HeapAlloc)
	probe.complete = true

	return ebiten.Termination
}
