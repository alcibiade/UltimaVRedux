package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync/atomic"
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

var errGainPlayerRetired = errors.New("audio player has been retired")

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

type gainStream struct {
	source     io.ReadSeeker
	gain       atomic.Uint64
	buffer     []byte
	pending    []byte
	pendingErr error
}

func (stream *gainStream) Read(buffer []byte) (int, error) {
	written := copy(buffer, stream.pending)
	stream.pending = stream.pending[written:]
	if len(stream.pending) == 0 && stream.pendingErr != nil {
		err := stream.pendingErr
		stream.pendingErr = nil

		return written, err
	}
	if written == len(buffer) {
		return written, nil
	}
	const sampleBytes = 2
	wanted := (len(buffer) - written + sampleBytes - 1) / sampleBytes * sampleBytes
	if cap(stream.buffer) < wanted {
		stream.buffer = make([]byte, wanted)
	}
	pcm := stream.buffer[:wanted]
	count, err := readCalibratedPCM(stream.source, pcm)
	if count%sampleBytes != 0 {
		return written, fmt.Errorf("%w: incomplete calibrated PCM sample", ErrAudioLibrary)
	}
	gain := math.Float64frombits(stream.gain.Load())
	const maxPCMSample = 32767
	for offset := 0; gain != 1 && offset+1 < count; offset += sampleBytes {
		//nolint:gosec // Bit-preserving signed PCM decoding.
		sample := int16(binary.LittleEndian.Uint16(pcm[offset:]))
		value := math.Max(-signedPCMScale, math.Min(maxPCMSample, math.Round(float64(sample)*gain)))
		//nolint:gosec // Clamped signed PCM converted to its unsigned bit representation.
		binary.LittleEndian.PutUint16(pcm[offset:], uint16(int16(value)))
	}
	copied := copy(buffer[written:], pcm[:count])
	stream.pending = pcm[copied:count]
	if len(stream.pending) > 0 {
		stream.pendingErr = err
		err = nil
	}

	return written + copied, err
}

//nolint:wrapcheck // Seeker adapter preserves source errors and byte offsets.
func (stream *gainStream) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekCurrent {
		offset -= int64(len(stream.pending))
	}
	position, err := stream.source.Seek(offset, whence)
	if err == nil {
		stream.pending = nil
		stream.pendingErr = nil
	}

	return position, err
}
