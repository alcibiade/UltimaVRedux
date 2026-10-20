package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDetectAudioFormat tests the audio format detection
func TestDetectAudioFormat(t *testing.T) {
	testCases := []struct {
		filename string
		expected AudioFormat
	}{
		{"test.wav", AudioFormatWAV},
		{"test.WAV", AudioFormatWAV},
		{"test.mp3", AudioFormatMP3},
		{"test.MP3", AudioFormatMP3},
		{"test.ogg", AudioFormatOGG},
		{"test.OGG", AudioFormatOGG},
		{"test.flac", AudioFormatUnknown},
		{"test", AudioFormatUnknown},
		{"", AudioFormatUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			format := DetectAudioFormat(tc.filename)
			assert.Equal(t, tc.expected, format, "Format detection should work for %s", tc.filename)
		})
	}
}
