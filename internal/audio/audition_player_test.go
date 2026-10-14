package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditionDeviceStub struct {
	stream   *auditionStream
	playing  bool
	stopped  int
	buffered int
	volume   float64
	err      error
}

func (d *auditionDeviceStub) Play()                    { d.playing = true }
func (d *auditionDeviceStub) Pause()                   { d.playing = false }
func (d *auditionDeviceStub) PauseAndStopReading()     { d.playing = false; d.stopped++ }
func (d *auditionDeviceStub) IsPlaying() bool          { return d.playing }
func (d *auditionDeviceStub) BufferedSize() int        { return d.buffered }
func (d *auditionDeviceStub) SetVolume(volume float64) { d.volume = volume }
func (d *auditionDeviceStub) Err() error               { return d.err }
func (d *auditionDeviceStub) Seek(offset int64, whence int) (int64, error) {
	d.buffered = 0
	return d.stream.Seek(offset, whence)
}

func TestStandaloneAuditionTracksConsumedPCMAndSeek(t *testing.T) {
	const bytesPerSecond = EditorSampleRate * decodedPCMBytesPerFrame
	stream := newAuditionStream(bytes.NewReader(make([]byte, bytesPerSecond*3)))
	device := &auditionDeviceStub{stream: stream, buffered: bytesPerSecond / 2}
	player := &auditionPlayer{device: device, stream: stream}
	_, err := io.ReadFull(stream, make([]byte, bytesPerSecond))
	require.NoError(t, err)
	assert.Equal(t, int64(500), player.PositionMs(), "read-ahead bytes are not played bytes")
	require.NoError(t, player.SeekMs(1500))
	assert.Equal(t, int64(1500), player.PositionMs())
	assert.Error(t, player.SeekMs(-1))
	player.Play()
	assert.True(t, player.IsPlaying())
	player.Pause()
	assert.False(t, player.IsPlaying())
}

func TestStandaloneAuditionUsesGameplayPCMCalibration(t *testing.T) {
	samples := make([]byte, 4)
	binary.LittleEndian.PutUint16(samples, 20000)
	binary.LittleEndian.PutUint16(samples[2:], 10000)
	stream := newAuditionStream(bytes.NewReader(samples))
	device := &auditionDeviceStub{stream: stream}
	player := &auditionPlayer{device: device, stream: stream}
	player.SetVolume(2)
	output := make([]byte, len(samples))
	_, err := io.ReadFull(stream, output)
	require.NoError(t, err)
	assert.Equal(t, uint16(32767), binary.LittleEndian.Uint16(output))
	assert.Equal(t, uint16(20000), binary.LittleEndian.Uint16(output[2:]))
	assert.Equal(t, 1.0, device.volume)
	player.SetVolume(0.25)
	assert.Equal(t, 0.25, device.volume)
}

func TestAuditionRetirementStopsReadsAndPreservesErrors(t *testing.T) {
	closeErr := errors.New("decoder failed")
	device := &auditionDeviceStub{playing: true, err: closeErr}
	audition := &Audition{player: &auditionPlayer{device: device, stream: newAuditionStream(bytes.NewReader(nil))}}
	require.ErrorIs(t, audition.Close(), closeErr)
	assert.Equal(t, 1, device.stopped)
	assert.False(t, audition.Active())
	require.NoError(t, audition.Close())
	assert.Equal(t, 1, device.stopped)
}
