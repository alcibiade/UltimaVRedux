package audio

import (
	"fmt"
	"io"
	"time"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
)

// PlayerTransport is the host's serialized audio transport contract. Retirement
// must wait for in-flight reads; creating a transport must not start playback.
type PlayerTransport = gainDevicePlayer

// PlaybackBackend is borrowed from the host. A runtime never closes its device.
type PlaybackBackend interface {
	SampleRate() int
	NewPlayer(source io.ReadSeeker) (PlayerTransport, error)
}

// EbitengineBackend adapts an explicitly supplied host context without creating
// another process device or selecting the application's configuration.
type EbitengineBackend struct{ Context *eaudio.Context }

// SampleRate reports the borrowed host rate; a missing context fails initialization.
func (backend EbitengineBackend) SampleRate() int {
	if backend.Context == nil {
		return 0
	}

	return backend.Context.SampleRate()
}

// NewPlayer creates a stopped transport on the borrowed host device.
//
//nolint:ireturn // Backend injection deliberately hides the concrete native player.
func (backend EbitengineBackend) NewPlayer(source io.ReadSeeker) (PlayerTransport, error) {
	if backend.Context == nil {
		return nil, fmt.Errorf("%w: host audio context is required", ErrAudioLibrary)
	}
	player, err := backend.Context.NewPlayer(source)
	if err != nil {
		return nil, fmt.Errorf("create host audio player: %w", err)
	}

	return player, nil
}

// CueRequest selects a neutral cue identity. ReplacePrevious is consumer policy,
// restricted to this lifetime; empty slots and inaudible finite cues remain silent.
type CueRequest struct {
	CueID           string
	Loop            bool
	Volume          float64
	FadeInMs        int64
	ReplacePrevious bool
}

// Runtime is the neutral cue-facing entry point to the same owned playback engine
// used by the Redux facade. It has no location, tile, game enum or save API.
type Runtime struct{ manager *AudioManager }

// NewRuntime injects one immutable library, host backend and presentation clock.
// Volume is caller-owned preference, not persisted or read from Redux settings.
func NewRuntime(library *Library, backend PlaybackBackend, now func() time.Time, volume float64) (*Runtime, error) {
	configuration, err := NewAudioConfiguration()
	if err != nil {
		return nil, err
	}
	if err := configuration.SetSoundEffectVolume(volume); err != nil {
		return nil, err
	}
	configuration.MusicVolume, configuration.AmbientVolume = 0, 0
	manager, err := newProjectAudioManager(configuration, library, backend, nil, now, false)
	if err != nil {
		return nil, err
	}

	return &Runtime{manager: manager}, nil
}

// Play starts an independently owned neutral cue instance.
func (runtime *Runtime) Play(request CueRequest) (VoiceHandle, error) {
	return runtime.manager.playCueVoice(request, runtime.manager.NewVoiceOwner())
}

// NewVoiceOwner groups several cue instances under one explicit lifetime.
func (runtime *Runtime) NewVoiceOwner() *VoiceOwner { return runtime.manager.NewVoiceOwner() }

// SetVolume applies a live category preference without changing authored gains.
func (runtime *Runtime) SetVolume(volume float64) error {
	return runtime.manager.SetSoundEffectVolume(volume)
}

// Update services fades and retirement using the injected presentation clock.
func (runtime *Runtime) Update() { runtime.manager.Update() }

// Close terminally releases all voices without closing the host backend.
func (runtime *Runtime) Close() error { return runtime.manager.Close() }
