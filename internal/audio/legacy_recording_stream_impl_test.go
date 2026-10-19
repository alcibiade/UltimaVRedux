package audio

import (
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

func decodeRecording(
	data []byte, path string, sampleRate int, window *PlaybackWindow, loop bool,
) (io.ReadSeeker, error) {
	stream, err := DecodeAudioStream(data, path, sampleRate)
	if err != nil {
		return nil, err
	}
	section, err := boundedRecording(stream, sampleRate, window)
	if err != nil {
		return nil, err
	}
	if loop {
		return audio.NewInfiniteLoop(section, section.Length()), nil
	}

	return section, nil
}

// recordingSection keeps all seek positions relative to the selected region.
// Music/ambience remain streaming rather than buffering a whole decoded track.
type recordingSection struct {
	source io.ReadSeeker
	start  int64
	size   int64
	pos    int64
}

func boundedRecording(source io.ReadSeeker, sampleRate int, window *PlaybackWindow) (*recordingSection, error) {
	length, err := decodedStreamLength(source)
	if err != nil {
		return nil, err
	}
	start, end := int64(0), length
	if window != nil {
		// Check before multiplying to avoid overflow from malformed timestamps.
		maxMs := length * millisecondsPerSecond / (int64(sampleRate) * decodedPCMBytesPerFrame)
		if window.StartMs < 0 || window.EndMs <= window.StartMs || window.EndMs > maxMs {
			return nil, fmt.Errorf("%w: excerpt exceeds recording duration", ErrAudioLibrary)
		}
		start = window.StartMs * int64(sampleRate) / millisecondsPerSecond * decodedPCMBytesPerFrame
		end = window.EndMs * int64(sampleRate) / millisecondsPerSecond * decodedPCMBytesPerFrame
	}
	section := &recordingSection{source: source, start: start, size: end - start, pos: 0}
	if _, err := section.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return section, nil
}

func (s *recordingSection) Length() int64 { return s.size }

//nolint:wrapcheck // An io.Reader adapter must preserve the underlying EOF and short-read behavior.
func (s *recordingSection) Read(buffer []byte) (int, error) {
	if s.pos >= s.size {
		return 0, io.EOF
	}
	if int64(len(buffer)) > s.size-s.pos {
		buffer = buffer[:s.size-s.pos]
	}
	n, err := s.source.Read(buffer)
	s.pos += int64(n)

	return n, err
}

func (s *recordingSection) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += s.pos
	case io.SeekEnd:
		offset += s.size
	default:
		return 0, fmt.Errorf("%w: invalid seek origin", ErrAudioLibrary)
	}
	if offset < 0 || offset > s.size {
		return 0, fmt.Errorf("%w: seek outside selected recording", ErrAudioLibrary)
	}
	if _, err := s.source.Seek(s.start+offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek recording: %w", err)
	}
	s.pos = offset

	return offset, nil
}
