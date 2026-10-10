package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"time"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
)

// gainPlayer keeps the device slider in [0,1] while supporting positive authored
// calibration. Scaling occurs on decoded PCM, without rewriting source files.
type gainPlayer struct {
	*eaudio.Player
	stream  *gainStream
	volume  float64
	retired bool
}

var errGainPlayerRetired = errors.New("audio player has been retired")

func newGainPlayer(context *eaudio.Context, source io.ReadSeeker) (*gainPlayer, error) {
	stream := new(gainStream)
	stream.source = source
	stream.gain.Store(math.Float64bits(1))
	player, err := context.NewPlayer(stream)
	if err != nil {
		return nil, fmt.Errorf("create calibrated player: %w", err)
	}

	return &gainPlayer{Player: player, stream: stream, volume: 1, retired: false}, nil
}

// Close retains the owner's terminal lifecycle while using Ebitengine 2.10's
// supported read barrier. Owners discard the voice after this returns; the
// decoder never calls into an audio-manager mutex while the barrier waits.
func (player *gainPlayer) Close() error {
	if player.retired {
		return errGainPlayerRetired
	}
	player.PauseAndStopReading()
	player.retired = true

	return nil
}

func (player *gainPlayer) Play() {
	if player.retired {
		panic("cannot play a retired audio voice")
	}
	player.Player.Play()
}

func (player *gainPlayer) SetPosition(position time.Duration) error {
	if player.retired {
		return errGainPlayerRetired
	}
	if err := player.Player.SetPosition(position); err != nil {
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
	player.Player.SetVolume(math.Min(1, player.volume))
}

func (player *gainPlayer) Volume() float64 { return player.volume }

type gainStream struct {
	source  io.ReadSeeker
	gain    atomic.Uint64
	buffer  []byte
	pending []byte
}

func (stream *gainStream) Read(buffer []byte) (int, error) {
	written := copy(buffer, stream.pending)
	stream.pending = stream.pending[written:]
	if written == len(buffer) {
		return written, nil
	}
	const sampleBytes = 2
	wanted := (len(buffer) - written + sampleBytes - 1) / sampleBytes * sampleBytes
	if cap(stream.buffer) < wanted {
		stream.buffer = make([]byte, wanted)
	}
	pcm := stream.buffer[:wanted]
	count, err := io.ReadFull(stream.source, pcm)
	if count%sampleBytes != 0 {
		return written, fmt.Errorf("%w: incomplete calibrated PCM sample", ErrAudioLibrary)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		err = io.EOF
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
	}

	return position, err
}
