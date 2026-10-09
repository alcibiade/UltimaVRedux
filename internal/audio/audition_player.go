package audio

import (
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Oto permits one device context per process. The standalone editor uses this
// context; the game continues using Ebitengine's separate process-owned context.
//
//nolint:gochecknoglobals // Native audio context creation must be process-wide and exactly once.
var editorDevice struct {
	once    sync.Once
	context *oto.Context
	err     error
}

func standaloneAudioContext() (*oto.Context, error) {
	editorDevice.once.Do(func() {
		const stereoChannels = 2
		var ready chan struct{}
		editorDevice.context, ready, editorDevice.err = oto.NewContext(&oto.NewContextOptions{
			SampleRate: EditorSampleRate, ChannelCount: stereoChannels, Format: oto.FormatSignedInt16LE,
			BufferSize: 0, ApplicationName: "Ultima V Redux Audio Editor",
		})
		if editorDevice.err != nil {
			return
		}
		const deviceReadyTimeout = 5 * time.Second
		timer := time.NewTimer(deviceReadyTimeout)
		defer timer.Stop()
		select {
		case <-ready:
			editorDevice.err = editorDevice.context.Err()
		case <-timer.C:
			editorDevice.err = fmt.Errorf("%w: audio device readiness timed out", ErrAudioLibrary)
		}
	})
	if editorDevice.err != nil {
		return nil, fmt.Errorf("open standalone audio device: %w", editorDevice.err)
	}

	return editorDevice.context, nil
}

// This is only the native editor voice boundary, not a second game audio API.
type auditionDevicePlayer interface {
	Play()
	Pause()
	PauseAndStopReading()
	IsPlaying() bool
	SetVolume(volume float64)
	BufferedSize() int
	Seek(offset int64, whence int) (int64, error)
	Err() error
}

type auditionPlayer struct {
	device auditionDevicePlayer
	stream *auditionStream
}

// auditionStream counts PCM bytes delivered to Oto, including complete loops.
// Oto serializes Read and Seek; the UI observes the position atomically.
type auditionStream struct {
	gain     *gainStream
	position atomic.Int64
}

func newAuditionStream(source io.ReadSeeker) *auditionStream {
	gain := &gainStream{source: source, gain: atomic.Uint64{}, buffer: nil, pending: nil}
	gain.gain.Store(math.Float64bits(1))

	return &auditionStream{gain: gain, position: atomic.Int64{}}
}

func (stream *auditionStream) Read(buffer []byte) (int, error) {
	count, err := stream.gain.Read(buffer)
	stream.position.Add(int64(count))

	return count, err
}

func (stream *auditionStream) Seek(offset int64, whence int) (int64, error) {
	position, err := stream.gain.Seek(offset, whence)
	if err == nil {
		stream.position.Store(position)
	}

	return position, err
}

func newAuditionPlayer(source io.ReadSeeker) (*auditionPlayer, error) {
	device, err := standaloneAudioContext()
	if err != nil {
		return nil, err
	}
	stream := newAuditionStream(source)

	return &auditionPlayer{device: device.NewPlayer(stream), stream: stream}, nil
}

func (player *auditionPlayer) Play()           { player.device.Play() }
func (player *auditionPlayer) Pause()          { player.device.Pause() }
func (player *auditionPlayer) IsPlaying() bool { return player.device.IsPlaying() }

func (player *auditionPlayer) SetVolume(gain float64) {
	gain = outputGain(gain)
	// Match gameplay calibration: only amplification is applied to PCM, with
	// attenuation applied by the mixer. The shared gainStream clips identically.
	player.stream.gain.gain.Store(math.Float64bits(math.Max(1, gain)))
	player.device.SetVolume(math.Min(1, gain))
}

func (player *auditionPlayer) PositionMs() int64 {
	played := max(0, player.stream.position.Load()-int64(player.device.BufferedSize()))

	return played * millisecondsPerSecond / (EditorSampleRate * decodedPCMBytesPerFrame)
}

func (player *auditionPlayer) SeekMs(position int64) error {
	if position < 0 || position > math.MaxInt64/(EditorSampleRate*decodedPCMBytesPerFrame) {
		return fmt.Errorf("%w: audition seek position out of range", ErrAudioLibrary)
	}
	bytePosition := position * EditorSampleRate / millisecondsPerSecond * decodedPCMBytesPerFrame
	if _, err := player.device.Seek(bytePosition, io.SeekStart); err != nil {
		return fmt.Errorf("seek standalone player: %w", err)
	}

	return nil
}

func (player *auditionPlayer) Close() error {
	// Pause alone allows read-ahead. Stop it before dropping the decoder and
	// voice; Oto.Close is a deprecated no-op, not a resource-lifecycle operation.
	player.device.PauseAndStopReading()
	if err := player.device.Err(); err != nil {
		return fmt.Errorf("stop standalone player: %w", err)
	}

	return nil
}
