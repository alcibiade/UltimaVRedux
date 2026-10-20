package audio

import (
	"path/filepath"
	"strings"
)

// AudioFormat represents supported audio file formats.
//
//nolint:revive // Distinguish encoded audio formats from the game's other asset formats.
type AudioFormat int

// Supported encoded formats for authored game recording paths.
const (
	AudioFormatUnknown AudioFormat = iota
	AudioFormatWAV
	AudioFormatMP3
	AudioFormatOGG
)

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
