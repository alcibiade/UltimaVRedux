package audio

import (
	"fmt"
	"math"
)

// PositionalAudioCalculator handles distance-based audio calculations
// Provides realistic audio positioning for 2D game environments
type PositionalAudioCalculator struct {
	// Configuration for audio calculations
	config *AudioConfiguration
}

// NewPositionalAudioCalculator creates a new positional audio calculator
func NewPositionalAudioCalculator() *PositionalAudioCalculator {
	return &PositionalAudioCalculator{}
}

// SetConfiguration updates the configuration used for calculations
func (pac *PositionalAudioCalculator) SetConfiguration(config *AudioConfiguration) {
	pac.config = config
}

// CalculateAudio calculates volume and pan for a sound source relative to player position
// Returns volume (0.0 to 1.0) and pan (-1.0 to 1.0, where -1.0 is left, 1.0 is right)
func (pac *PositionalAudioCalculator) CalculateAudio(playerX, playerY, sourceX, sourceY int) (volume, pan float64) {
	// Use default configuration if none set
	config := pac.config
	if config == nil {
		defaultConfig, err := NewAudioConfiguration()
		if err != nil {
			panic(fmt.Sprintf("constructing built-in audio defaults: %v", err))
		}
		config = defaultConfig
	}

	// Calculate distance between player and sound source
	distance := pac.calculateDistance(playerX, playerY, sourceX, sourceY)

	// Calculate volume based on distance
	volume = pac.calculateVolumeFromDistance(distance, config)

	// Calculate stereo pan based on horizontal position
	pan = pac.calculatePanFromPosition(playerX, sourceX)

	return volume, pan
}

// calculateDistance returns the Euclidean distance between two points
func (pac *PositionalAudioCalculator) calculateDistance(x1, y1, x2, y2 int) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)

	return math.Sqrt(dx*dx + dy*dy)
}

// calculateVolumeFromDistance calculates volume based on distance and configuration
func (pac *PositionalAudioCalculator) calculateVolumeFromDistance(distance float64, config *AudioConfiguration) float64 {
	// If distance is 0 or very small, return full volume
	if distance <= 0.1 {
		return 1.0
	}

	// If distance exceeds maximum, return silence
	if distance >= config.MaxAudioDistance {
		return 0.0
	}

	// Calculate volume using inverse square law with configurable rolloff
	// volume = 1 / (1 + rolloff * distance^2)
	volumeReduction := 1.0 + (config.VolumeRolloffFactor * distance * distance)
	volume := 1.0 / volumeReduction

	// Ensure volume is in valid range
	return math.Max(0.0, math.Min(1.0, volume))
}

// calculatePanFromPosition calculates stereo pan based on horizontal offset
func (pac *PositionalAudioCalculator) calculatePanFromPosition(playerX, sourceX int) float64 {
	// Calculate horizontal offset
	offset := sourceX - playerX

	// Convert to pan value (-1.0 to 1.0)
	// Use a reasonable range for tile-based movement (e.g., 10 tiles = full pan)
	maxPanDistance := 10.0
	pan := float64(offset) / maxPanDistance

	// Clamp to valid range
	return math.Max(-1.0, math.Min(1.0, pan))
}

// IsAudible returns whether a sound at the given position would be audible to the player
func (pac *PositionalAudioCalculator) IsAudible(playerX, playerY, sourceX, sourceY int, minVolume float64) bool {
	volume, _ := pac.CalculateAudio(playerX, playerY, sourceX, sourceY)

	return volume >= minVolume
}
