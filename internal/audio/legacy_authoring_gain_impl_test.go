package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync/atomic"
)

var errGainPlayerRetired = errors.New("audio player has been retired")

// gainStream is the standalone TUI's calibrated PCM adapter. Gameplay playback
// uses Berad Audio; the authoring transport remains local pending its extraction.
type gainStream struct {
	source          io.ReadSeeker
	gain            atomic.Uint64
	buffer, pending []byte
	pendingErr      error
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
