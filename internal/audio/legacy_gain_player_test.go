package audio

import (
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

// gainPlayer keeps the device slider in [0,1] while supporting positive authored
// calibration. Scaling occurs on decoded PCM, without rewriting source files.
type gainPlayer struct {
	device   gainDevicePlayer
	stream   *gainStream
	boundary *runtimeStream
	volume   float64
	retired  bool
}

// gainDevicePlayer is the narrow Ebitengine transport boundary. Production uses
// the borrowed context; conformance tests drive this same adapter with controlled
// device reads, independently of native device initialization/listening checks.
type gainDevicePlayer interface {
	Play()
	Pause()
	PauseAndStopReading()
	IsPlaying() bool
	SetVolume(volume float64)
	SetPosition(position time.Duration) error
	Position() time.Duration
}

func newGainPlayerWithBackend(backend PlaybackBackend, source io.ReadSeeker) (*gainPlayer, error) {
	return newGainPlayerWithDevice(source, backend.NewPlayer)
}

func newGainPlayerWithDevice(
	source io.ReadSeeker, create func(io.ReadSeeker) (gainDevicePlayer, error),
) (*gainPlayer, error) {
	if source == nil {
		return nil, errSoundStreamUnavailable
	}
	// Ebitengine defers reader initialization until playback. Validate the
	// required seeker before the owner admits this voice or evicts another one.
	if _, err := source.Seek(0, io.SeekCurrent); err != nil {
		return nil, errors.Join(fmt.Errorf("prepare calibrated source: %w", err), closeRuntimeSource(source))
	}
	stream := new(gainStream)
	stream.source = source
	stream.gain.Store(math.Float64bits(1))
	boundary := new(runtimeStream)
	boundary.source = stream
	player, err := create(boundary)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("create calibrated player: %w", err), closeRuntimeSource(source))
	}
	if err := boundary.Failure(); err != nil {
		player.PauseAndStopReading()

		return nil, errors.Join(err, closeRuntimeSource(source))
	}

	return &gainPlayer{device: player, stream: stream, boundary: boundary, volume: 1, retired: false}, nil
}

func (player *gainPlayer) Pause()                  { player.device.Pause() }
func (player *gainPlayer) PauseAndStopReading()    { player.device.PauseAndStopReading() }
func (player *gainPlayer) IsPlaying() bool         { return !player.retired && player.device.IsPlaying() }
func (player *gainPlayer) Position() time.Duration { return player.device.Position() }

// Close retains the owner's terminal lifecycle while using Ebitengine 2.10's
// supported read barrier. Owners discard the voice after this returns; the
// decoder never calls into an audio-manager mutex while the barrier waits.
func (player *gainPlayer) Close() error {
	if player.retired {
		return errGainPlayerRetired
	}
	player.PauseAndStopReading()
	player.retired = true

	return closeRuntimeSource(player.stream.source)
}

func (player *gainPlayer) Failure() error { return player.boundary.Failure() }

func (player *gainPlayer) Play() {
	if player.retired {
		panic("cannot play a retired audio voice")
	}
	player.device.Play()
}

func (player *gainPlayer) SetPosition(position time.Duration) error {
	if player.retired {
		return errGainPlayerRetired
	}
	if position < 0 {
		return fmt.Errorf("%w: negative runtime audio position", ErrAudioLibrary)
	}
	if err := player.device.SetPosition(position); err != nil {
		return fmt.Errorf("seek calibrated player: %w", err)
	}

	return nil
}

func (player *gainPlayer) Rewind() error { return player.SetPosition(0) }

func (player *gainPlayer) SetVolume(gain float64) {
	player.volume = outputGain(gain)
	// Retain the device's normal attenuation/fade path below unity. Only the
	// excess boost is applied to PCM; accepted auto gains retain peak headroom.
	player.stream.gain.Store(math.Float64bits(math.Max(1, player.volume)))
	player.device.SetVolume(math.Min(1, player.volume))
}

func (player *gainPlayer) Volume() float64 { return player.volume }
