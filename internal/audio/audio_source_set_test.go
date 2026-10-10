package audio

import (
	"bytes"
	"io"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/stretchr/testify/assert"
)

// Test creating an empty AudioSourceSet
func TestNewAudioSourceSet_Empty(t *testing.T) {
	audioContext := getSharedAudioContext()

	audioSet, err := NewAudioSourceSet(audioContext, []io.ReadSeeker{})
	assert.NoError(t, err)
	assert.Nil(t, audioSet) // Empty set should return nil

	audioSet, err = NewAudioSourceSet(audioContext, nil)
	assert.NoError(t, err)
	assert.Nil(t, audioSet) // Nil streams should return nil
}

// Test creating an AudioSourceSet with valid streams
func TestNewAudioSourceSet_ValidStreams(t *testing.T) {
	audioContext := getSharedAudioContext()

	// Create mock audio streams (simple WAV-like data)
	stream1 := bytes.NewReader(createMockWAVData(44100, 1.0)) // 1 second of audio
	stream2 := bytes.NewReader(createMockWAVData(44100, 0.5)) // 0.5 seconds of audio

	audioSet, err := NewAudioSourceSet(audioContext, []io.ReadSeeker{stream1, stream2})
	assert.NoError(t, err)
	assert.NotNil(t, audioSet)
	assert.Equal(t, 2, len(audioSet.clips))
	assert.False(t, audioSet.IsEmpty())

	// Clean up
	audioSet.Close()
}

// Test GetNext method with single clip
func TestAudioSourceSet_GetNext_SingleClip(t *testing.T) {
	audioContext := getSharedAudioContext()

	stream := bytes.NewReader(createMockWAVData(44100, 1.0))
	audioSet, err := NewAudioSourceSet(audioContext, []io.ReadSeeker{stream})
	assert.NoError(t, err)
	assert.NotNil(t, audioSet)
	defer audioSet.Close()

	// Should always return the same clip
	player1 := audioSet.GetNext()
	player2 := audioSet.GetNext()
	assert.NotNil(t, player1)
	assert.NotNil(t, player2)
	assert.Equal(t, player1, player2) // Same player instance
}

// Test GetNext method with multiple clips (variation logic)
func TestAudioSourceSet_GetNext_MultipleClips(t *testing.T) {
	audioContext := getSharedAudioContext()

	// Create multiple streams
	streams := []io.ReadSeeker{
		bytes.NewReader(createMockWAVData(44100, 1.0)),
		bytes.NewReader(createMockWAVData(44100, 1.0)),
		bytes.NewReader(createMockWAVData(44100, 1.0)),
	}

	audioSet, err := NewAudioSourceSet(audioContext, streams)
	assert.NoError(t, err)
	assert.NotNil(t, audioSet)
	defer audioSet.Close()

	// Test that we get different players over multiple calls
	players := make([]*audio.Player, 10)
	for i := 0; i < 10; i++ {
		players[i] = audioSet.GetNext()
		assert.NotNil(t, players[i])
	}

	// Should have some variation (not all the same)
	allSame := true
	for i := 1; i < len(players); i++ {
		if players[i] != players[0] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "GetNext should provide variation in audio clips")
}

// Test SetVolume method
func TestAudioSourceSet_SetVolume(t *testing.T) {
	audioContext := getSharedAudioContext()

	stream := bytes.NewReader(createMockWAVData(44100, 1.0))
	audioSet, err := NewAudioSourceSet(audioContext, []io.ReadSeeker{stream})
	assert.NoError(t, err)
	assert.NotNil(t, audioSet)
	defer audioSet.Close()

	// Set volume should not panic
	audioSet.SetVolume(0.5)
	audioSet.SetVolume(0.0)
	audioSet.SetVolume(1.0)
}

// Test Close method
func TestAudioSourceSet_Close(t *testing.T) {
	audioContext := getSharedAudioContext()

	stream := bytes.NewReader(createMockWAVData(44100, 1.0))
	audioSet, err := NewAudioSourceSet(audioContext, []io.ReadSeeker{stream})
	assert.NoError(t, err)
	assert.NotNil(t, audioSet)

	// Should not panic
	audioSet.Close()

	// GetNext should return nil after close
	player := audioSet.GetNext()
	assert.Nil(t, player)

	// Should be marked as empty
	assert.True(t, audioSet.IsEmpty())
}

// Test IsEmpty method
func TestAudioSourceSet_IsEmpty(t *testing.T) {
	audioContext := getSharedAudioContext()

	// Empty set
	emptySet, _ := NewAudioSourceSet(audioContext, []io.ReadSeeker{})
	assert.True(t, emptySet == nil || emptySet.IsEmpty())

	// Non-empty set
	stream := bytes.NewReader(createMockWAVData(44100, 1.0))
	audioSet, err := NewAudioSourceSet(audioContext, []io.ReadSeeker{stream})
	assert.NoError(t, err)
	assert.False(t, audioSet.IsEmpty())

	// After close should be empty
	audioSet.Close()
	assert.True(t, audioSet.IsEmpty())
}

// Helper function to create mock WAV data for testing
func createMockWAVData(sampleRate int, durationSeconds float64) []byte {
	// Create a very simple WAV-like header and data
	// This is a minimal implementation for testing purposes
	numSamples := int(float64(sampleRate) * durationSeconds)

	// Simple WAV header (44 bytes) + PCM data
	header := []byte{
		// RIFF chunk
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x00, 0x00, 0x00, 0x00, // File size (to be filled)
		0x57, 0x41, 0x56, 0x45, // "WAVE"

		// Format chunk
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // Format chunk size (16)
		0x01, 0x00, // Audio format (PCM)
		0x01, 0x00, // Number of channels (1)
		0x44, 0xAC, 0x00, 0x00, // Sample rate (44100)
		0x88, 0x58, 0x01, 0x00, // Byte rate
		0x02, 0x00, // Block align
		0x10, 0x00, // Bits per sample (16)

		// Data chunk
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x00, 0x00, 0x00, // Data size (to be filled)
	}

	// Create simple sine wave PCM data
	data := make([]byte, numSamples*2) // 16-bit samples
	for i := 0; i < numSamples; i++ {
		// Simple sine wave pattern (very basic)
		sample := int16(3276) // Low amplitude (32767 * 0.1)
		if i%100 < 50 {
			sample = -sample
		}

		// Little-endian 16-bit
		data[i*2] = byte(sample & 0xFF)
		data[i*2+1] = byte((sample >> 8) & 0xFF)
	}

	// Update file sizes in header
	totalSize := len(header) + len(data) - 8
	header[4] = byte(totalSize & 0xFF)
	header[5] = byte((totalSize >> 8) & 0xFF)
	header[6] = byte((totalSize >> 16) & 0xFF)
	header[7] = byte((totalSize >> 24) & 0xFF)

	dataSize := len(data)
	header[40] = byte(dataSize & 0xFF)
	header[41] = byte((dataSize >> 8) & 0xFF)
	header[42] = byte((dataSize >> 16) & 0xFF)
	header[43] = byte((dataSize >> 24) & 0xFF)

	// Combine header and data
	result := make([]byte, len(header)+len(data))
	copy(result[:len(header)], header)
	copy(result[len(header):], data)

	return result
}
