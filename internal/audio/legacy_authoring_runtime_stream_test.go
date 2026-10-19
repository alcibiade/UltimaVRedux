package audio

import (
	"fmt"
	"io"
)

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
