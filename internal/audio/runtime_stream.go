package audio

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
)

type streamFailure struct{ err error }

// ownedRuntimeStream preserves a decoder's close ownership through loop/window
// adapters, which expose seeking but do not forward io.Closer.
type ownedRuntimeStream struct {
	io.ReadSeeker
	owner io.ReadSeeker
}

func (stream *ownedRuntimeStream) Close() error { return closeRuntimeSource(stream.owner) }

func closeRuntimeSource(source io.ReadSeeker) error {
	if closer, ok := source.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("close audio decoder: %w", err)
		}
	}

	return nil
}

// A broken decoder must not spin forever inside io.ReadFull. Allow bounded
// temporary empty reads, using the same threshold as standard library readers.
type progressingReader struct {
	io.Reader
	emptyReads int
}

// Unlike io.ReadFull, retain an error delivered with a completely filled buffer.
// Otherwise a decoder that reports its fault only once could look like normal EOF.
func readCalibratedPCM(source io.Reader, buffer []byte) (int, error) {
	reader := progressingReader{Reader: source, emptyReads: 0}
	var count int
	for count < len(buffer) {
		n, err := reader.Read(buffer[count:])
		if n < 0 || n > len(buffer)-count {
			return count, fmt.Errorf("%w: invalid decoded byte count", ErrAudioLibrary)
		}
		count += n
		if err != nil {
			return count, err
		}
	}

	return count, nil
}

//nolint:wrapcheck // Reader adapter preserves the original decoder error.
func (reader *progressingReader) Read(buffer []byte) (int, error) {
	count, err := reader.Reader.Read(buffer)
	if count == 0 && err == nil && len(buffer) != 0 {
		reader.emptyReads++
		const maximumEmptyReads = 100
		if reader.emptyReads >= maximumEmptyReads {
			return 0, io.ErrNoProgress
		}
	} else {
		reader.emptyReads = 0
	}

	return count, err
}

// runtimeStream contains decoder/PCM read failures before they reach the
// Ebitengine context-wide error hook. The backend sees finite EOF; the owning
// service sees the original failure and terminally retires the voice.
// Backend read/seek serialization is retained; failure inspection is concurrent.
type runtimeStream struct {
	source  io.ReadSeeker
	failure atomic.Pointer[streamFailure]
}

func (stream *runtimeStream) Failure() error {
	if failure := stream.failure.Load(); failure != nil {
		return failure.err
	}

	return nil
}

//nolint:wrapcheck // EOF and original source errors are part of the reader contract.
func (stream *runtimeStream) Read(buffer []byte) (int, error) {
	if stream.Failure() != nil {
		return 0, io.EOF
	}
	count, err := stream.source.Read(buffer)
	if errors.Is(err, io.EOF) {
		return count, io.EOF
	}
	if err != nil {
		stream.failure.CompareAndSwap(nil, &streamFailure{err: fmt.Errorf("runtime audio read: %w", err)})

		return count, io.EOF
	}

	return count, err
}

func (stream *runtimeStream) Seek(offset int64, whence int) (int64, error) {
	if err := stream.Failure(); err != nil {
		return 0, err
	}
	position, err := stream.source.Seek(offset, whence)
	if err != nil {
		failure := fmt.Errorf("runtime audio seek: %w", err)
		stream.failure.CompareAndSwap(nil, &streamFailure{err: failure})

		return position, failure
	}

	return position, nil
}
