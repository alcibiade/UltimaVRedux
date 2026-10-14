package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/stretchr/testify/require"
)

// This controlled device performs real adapter reads and synchronizes in-flight
// reads with retirement. It verifies our adapter contract, not native hardware,
// Ebitengine game-loop initialization, mixer buffering latency or listening.
type conformanceDevice struct {
	source         io.ReadSeeker
	readMutex      sync.Mutex
	playing        atomic.Bool
	failure        atomic.Pointer[streamFailure]
	position       int64
	volume         float64
	barrierEntered chan struct{}
	barriers       atomic.Int32
}

func (device *conformanceDevice) Play()                    { device.playing.Store(true) }
func (device *conformanceDevice) Pause()                   { device.playing.Store(false) }
func (device *conformanceDevice) IsPlaying() bool          { return device.playing.Load() }
func (device *conformanceDevice) SetVolume(volume float64) { device.volume = volume }
func (device *conformanceDevice) BufferedSize() int        { return 0 }
func (device *conformanceDevice) Err() error {
	if failure := device.failure.Load(); failure != nil {
		return failure.err
	}
	return nil
}
func (device *conformanceDevice) PauseAndStopReading() {
	device.Pause()
	if device.barriers.Add(1) == 1 {
		close(device.barrierEntered)
	}
	device.readMutex.Lock()
	defer device.readMutex.Unlock()
}
func (device *conformanceDevice) Seek(offset int64, whence int) (int64, error) {
	device.readMutex.Lock()
	defer device.readMutex.Unlock()
	position, err := device.source.Seek(offset, whence)
	if err == nil {
		device.position = position
	}
	return position, err
}
func (device *conformanceDevice) SetPosition(position time.Duration) error {
	offset := int64(position/time.Millisecond) * EditorSampleRate / millisecondsPerSecond * decodedPCMBytesPerFrame
	_, err := device.Seek(offset, io.SeekStart)
	return err
}
func (device *conformanceDevice) Position() time.Duration {
	device.readMutex.Lock()
	defer device.readMutex.Unlock()
	return time.Duration(device.position * int64(time.Second) / (EditorSampleRate * decodedPCMBytesPerFrame))
}
func (device *conformanceDevice) read(count int) ([]byte, error) {
	device.readMutex.Lock()
	defer device.readMutex.Unlock()
	if !device.IsPlaying() {
		return nil, nil
	}
	buffer := make([]byte, count)
	n, err := device.source.Read(buffer)
	device.position += int64(n)
	if err != nil {
		device.Pause()
		if !errors.Is(err, io.EOF) {
			device.failure.CompareAndSwap(nil, &streamFailure{err: err})
		}
	}
	return buffer[:n], err
}

type conformanceVoice struct {
	device      *conformanceDevice
	play, pause func()
	playing     func() bool
	gain        func(float64)
	seek        func(int64) error
	position    func() int64
	failure     func() error
	close       func() error
}

func makeConformanceVoice(t *testing.T, backend string, source io.ReadSeeker) conformanceVoice {
	t.Helper()
	device := &conformanceDevice{barrierEntered: make(chan struct{})}
	if backend == "ebitengine" {
		voice, err := newGainPlayerWithDevice(source, func(stream io.ReadSeeker) (gainDevicePlayer, error) {
			device.source = stream
			return device, nil
		})
		require.NoError(t, err)
		return conformanceVoice{device: device, play: voice.Play, pause: voice.Pause, playing: voice.IsPlaying, gain: voice.SetVolume,
			seek:     func(ms int64) error { return voice.SetPosition(time.Duration(ms) * time.Millisecond) },
			position: func() int64 { return voice.Position().Milliseconds() }, failure: voice.Failure, close: voice.Close}
	}
	voice, err := newAuditionPlayerWithDevice(source, func(stream io.Reader) auditionDevicePlayer {
		var ok bool
		device.source, ok = stream.(io.ReadSeeker)
		require.True(t, ok)
		return device
	})
	require.NoError(t, err)
	return conformanceVoice{device: device, play: voice.Play, pause: voice.Pause, playing: voice.IsPlaying, gain: voice.SetVolume,
		seek: voice.SeekMs, position: voice.PositionMs, failure: voice.Failure, close: voice.Close}
}

func TestBackendConformanceGainPauseSeekEOFAndRetirement(t *testing.T) {
	for _, backend := range []string{"ebitengine", "oto"} {
		t.Run(backend, func(t *testing.T) {
			pcm := make([]byte, EditorSampleRate*decodedPCMBytesPerFrame)
			for offset := 0; offset < len(pcm); offset += 4 {
				binary.LittleEndian.PutUint16(pcm[offset:], 20000)
				binary.LittleEndian.PutUint16(pcm[offset+2:], 45536) // -20000 in signed 16-bit PCM.
			}
			section, err := boundedRecording(benchmarkPCM{bytes.NewReader(pcm)}, EditorSampleRate, &PlaybackWindow{StartMs: 200, EndMs: 700})
			require.NoError(t, err)
			voice := makeConformanceVoice(t, backend, section)
			voice.gain(2)
			voice.play()
			output, err := voice.device.read(4)
			require.NoError(t, err)
			require.Equal(t, []byte{255, 127, 0, 128}, output, "identical saturating authored amplification")
			require.Equal(t, 1.0, voice.device.volume)
			voice.gain(0.25)
			output, err = voice.device.read(4)
			require.NoError(t, err)
			require.Equal(t, pcm[:4], output, "attenuation belongs to the device mixer, not a second PCM multiplier")
			require.Equal(t, 0.25, voice.device.volume)
			voice.pause()
			before := voice.position()
			output, err = voice.device.read(4)
			require.NoError(t, err)
			require.Empty(t, output)
			require.Equal(t, before, voice.position())
			require.NoError(t, voice.seek(250))
			require.EqualValues(t, 250, voice.position())
			require.False(t, voice.playing(), "seeking a paused voice does not resume it")
			voice.play()
			_, err = voice.device.read(EditorSampleRate * decodedPCMBytesPerFrame)
			require.ErrorIs(t, err, io.EOF)
			require.False(t, voice.playing())
			require.NoError(t, voice.failure(), "natural EOF is not decoder failure")
			require.EqualValues(t, 500, voice.position())
			require.NoError(t, voice.seek(0))
			voice.play()
			require.True(t, voice.playing(), "healthy retained EOF can be rewound")
			require.NoError(t, voice.close())
			require.ErrorIs(t, voice.close(), errGainPlayerRetired)
			require.ErrorIs(t, voice.seek(0), errGainPlayerRetired)
			require.Panics(t, voice.play)
			require.False(t, voice.playing())
			require.EqualValues(t, 1, voice.device.barriers.Load())
		})
	}
}

func TestBackendConformanceLoopedSelectionAndBufferedGainBoundary(t *testing.T) {
	for _, backend := range []string{"ebitengine", "oto"} {
		t.Run(backend, func(t *testing.T) {
			pcm := []byte{1, 0, 2, 0, 3, 0, 4, 0}
			voice := makeConformanceVoice(t, backend, eaudio.NewInfiniteLoop(bytes.NewReader(pcm), int64(len(pcm))))
			voice.play()
			output, err := voice.device.read(3)
			require.NoError(t, err)
			require.Equal(t, pcm[:3], output)
			voice.gain(2)
			remainder, err := voice.device.read(13)
			require.NoError(t, err)
			output = append(output, remainder...)
			require.Equal(t, []byte{1, 0, 2, 0, 6, 0, 8, 0, 2, 0, 4, 0, 6, 0, 8, 0}, output,
				"pending PCM keeps its original complete sample; fresh samples use the new gain")
			require.True(t, voice.playing())
			require.NoError(t, voice.close())
		})
	}
}

func TestBackendConformancePreservesDecoderAndSeekErrors(t *testing.T) {
	for _, backend := range []string{"ebitengine", "oto"} {
		t.Run(backend, func(t *testing.T) {
			broken := errors.New("controlled decoder failure")
			source := &faultPCM{Reader: bytes.NewReader([]byte{1, 2, 3, 4}), failure: broken}
			voice := makeConformanceVoice(t, backend, source)
			voice.play()
			_, readErr := voice.device.read(8)
			if backend == "ebitengine" {
				require.ErrorIs(t, readErr, io.EOF)
				require.NoError(t, voice.device.Err(), "runtime adapter contains the context-wide error path")
			} else {
				require.ErrorIs(t, readErr, broken, "Oto exposes per-player reader failure")
			}
			require.ErrorIs(t, voice.failure(), broken)
			closeErr := voice.close()
			if backend == "oto" {
				require.ErrorIs(t, closeErr, broken)
			} else {
				require.NoError(t, closeErr)
			}
			require.Equal(t, 1, source.closes)

			seeker := &faultPCM{Reader: bytes.NewReader(make([]byte, 256))}
			voice = makeConformanceVoice(t, backend, seeker)
			seeker.seekFailure = broken
			require.ErrorIs(t, voice.seek(1), broken)
			require.NoError(t, voice.close())
			require.Equal(t, 1, seeker.closes)
		})
	}
}

type inFlightPCM struct {
	*bytes.Reader
	entered, release         chan struct{}
	inRead, closedDuringRead atomic.Bool
	closes                   atomic.Int32
}

func (source *inFlightPCM) Read(buffer []byte) (int, error) {
	source.inRead.Store(true)
	close(source.entered)
	<-source.release
	n, err := source.Reader.Read(buffer)
	source.inRead.Store(false)
	return n, err
}
func (source *inFlightPCM) Close() error {
	source.closedDuringRead.Store(source.inRead.Load())
	source.closes.Add(1)
	return nil
}

func TestBackendConformanceRetirementWaitsForInFlightRead(t *testing.T) {
	for _, backend := range []string{"ebitengine", "oto"} {
		t.Run(backend, func(t *testing.T) {
			source := &inFlightPCM{Reader: bytes.NewReader([]byte{1, 2, 3, 4}), entered: make(chan struct{}), release: make(chan struct{})}
			var release sync.Once
			releaseRead := func() { release.Do(func() { close(source.release) }) }
			t.Cleanup(releaseRead)
			voice := makeConformanceVoice(t, backend, source)
			voice.play()
			readDone := make(chan struct{})
			go func() { _, _ = voice.device.read(4); close(readDone) }()
			awaitConformanceSignal(t, source.entered)
			closeDone := make(chan error, 1)
			go func() { closeDone <- voice.close() }()
			awaitConformanceSignal(t, voice.device.barrierEntered)
			require.Zero(t, source.closes.Load(), "source must stay open behind the read barrier")
			select {
			case <-closeDone:
				t.Fatal("retirement returned with a read still in flight")
			default:
			}
			releaseRead()
			awaitConformanceSignal(t, readDone)
			select {
			case err := <-closeDone:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("retirement did not finish after the read returned")
			}
			require.False(t, source.closedDuringRead.Load())
			require.EqualValues(t, 1, source.closes.Load())
			require.ErrorIs(t, voice.close(), errGainPlayerRetired)
			require.EqualValues(t, 1, source.closes.Load())
		})
	}
}

func awaitConformanceSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("controlled audio operation did not reach its synchronization boundary")
	}
}
