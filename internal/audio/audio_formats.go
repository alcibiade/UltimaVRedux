package audio

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// AudioFormat represents supported audio file formats
type AudioFormat int

const (
	AudioFormatUnknown AudioFormat = iota
	AudioFormatWAV
	AudioFormatMP3
	AudioFormatOGG
)

// String returns human-readable format name
func (af AudioFormat) String() string {
	switch af {
	case AudioFormatWAV:
		return "WAV"
	case AudioFormatMP3:
		return "MP3"
	case AudioFormatOGG:
		return "OGG"
	default:
		return "Unknown"
	}
}

// DetectAudioFormat determines the audio format from file extension
func DetectAudioFormat(filename string) AudioFormat {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".wav":
		return AudioFormatWAV
	case ".mp3":
		return AudioFormatMP3
	case ".ogg":
		return AudioFormatOGG
	default:
		return AudioFormatUnknown
	}
}

// DecodeAudioStream decodes audio data based on detected format
func DecodeAudioStream(audioData []byte, filename string, sampleRate int) (io.ReadSeeker, error) {
	format := DetectAudioFormat(filename)
	reader := bytes.NewReader(audioData)

	switch format {
	case AudioFormatWAV:
		stream, err := wav.DecodeWithSampleRate(sampleRate, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode WAV file %s: %w", filename, err)
		}

		return stream, nil

	case AudioFormatMP3:
		stream, err := mp3.DecodeWithSampleRate(sampleRate, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode MP3 file %s: %w", filename, err)
		}

		return stream, nil

	case AudioFormatOGG:
		stream, err := vorbis.DecodeWithSampleRate(sampleRate, reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OGG file %s: %w", filename, err)
		}

		return stream, nil

	default:
		return nil, fmt.Errorf("unsupported audio format for file %s (detected: %s)", filename, format.String())
	}
}
