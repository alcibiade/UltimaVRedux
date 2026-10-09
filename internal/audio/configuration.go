package audio

import (
	"errors"
	"math"
)

// AudioConfiguration manages audio settings and volume controls with validation
type AudioConfiguration struct {
	// Volume controls (0.0 to 1.0)
	SoundEffectVolume float64
	MusicVolume       float64
	AmbientVolume     float64

	// Positional audio settings (used by PositionalAudioCalculator)
	MaxAudioDistance    float64 // Maximum distance before audio is silent
	VolumeRolloffFactor float64 // How quickly volume decreases with distance
}

// NewAudioConfiguration creates a new configuration with default settings
func NewAudioConfiguration() (*AudioConfiguration, error) {
	return &AudioConfiguration{
		// Default volume settings - values are pre-converted from slider scale via ConvertSliderToVolume
		SoundEffectVolume: ConvertSliderToVolume(6), // Slider 6 = 43% amplitude (balanced)
		MusicVolume:       ConvertSliderToVolume(5), // Slider 5 = 30% amplitude (good medium level)
		AmbientVolume:     ConvertSliderToVolume(4), // Slider 4 = 18% amplitude (subtle background)

		// Positional audio defaults
		MaxAudioDistance:    20.0, // Audio silent beyond 20 tiles
		VolumeRolloffFactor: 0.3,  // Gentle rolloff curve
	}, nil
}

// SetSoundEffectVolume sets the sound effect volume with validation
func (ac *AudioConfiguration) SetSoundEffectVolume(volume float64) error {
	if err := validateVolumeRange(volume); err != nil {
		return err
	}

	ac.SoundEffectVolume = volume

	return nil
}

// SetMusicVolume sets the music volume with validation
func (ac *AudioConfiguration) SetMusicVolume(volume float64) error {
	if err := validateVolumeRange(volume); err != nil {
		return err
	}

	ac.MusicVolume = volume

	return nil
}

// SetAmbientVolume sets the ambient volume with validation
func (ac *AudioConfiguration) SetAmbientVolume(volume float64) error {
	if err := validateVolumeRange(volume); err != nil {
		return err
	}

	ac.AmbientVolume = volume

	return nil
}

// GetEffectiveVolume combines the already-converted user preference with a
// continuous request/context/position gain. Only the final output is bounded.
func (ac *AudioConfiguration) GetEffectiveVolume(gain float64) float64 {
	return outputGain(ac.SoundEffectVolume * gain)
}

// GetEffectiveMusicVolume calculates the final volume for background music
func (ac *AudioConfiguration) GetEffectiveMusicVolume() float64 {
	// MusicVolume is already converted from slider scale (via ConvertSliderToVolume in AudioManager)
	return outputGain(ac.MusicVolume)
}

// GetEffectiveSoundEffectVolume calculates the final volume for sound effects without positional audio
func (ac *AudioConfiguration) GetEffectiveSoundEffectVolume() float64 {
	// SoundEffectVolume is already converted from slider scale (via ConvertSliderToVolume in AudioManager)
	return outputGain(ac.SoundEffectVolume)
}

// GetEffectiveAmbientVolume calculates the final volume for ambient sounds
func (ac *AudioConfiguration) GetEffectiveAmbientVolume(positionalVolume ...float64) float64 {
	gain := 1.0
	if len(positionalVolume) > 0 {
		gain = positionalVolume[0]
	}

	return outputGain(ac.AmbientVolume * gain)
}

// outputGain preserves positive calibration boosts; gainPlayer bounds PCM
// samples instead of incorrectly capping the gain multiplier at unity.
func outputGain(gain float64) float64 {
	if math.IsNaN(gain) || math.IsInf(gain, 0) {
		return 0
	}

	return math.Max(0, gain)
}

// Clone creates a deep copy of the configuration
func (ac *AudioConfiguration) Clone() *AudioConfiguration {
	return &AudioConfiguration{
		SoundEffectVolume:   ac.SoundEffectVolume,
		MusicVolume:         ac.MusicVolume,
		AmbientVolume:       ac.AmbientVolume,
		MaxAudioDistance:    ac.MaxAudioDistance,
		VolumeRolloffFactor: ac.VolumeRolloffFactor,
	}
}

// validateVolumeRange ensures volume is in valid 0.0 to 1.0 range
func validateVolumeRange(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return errors.New("volume must be between 0.0 and 1.0")
	}

	if math.IsNaN(volume) || math.IsInf(volume, 0) {
		return errors.New("volume must be a valid number")
	}

	return nil
}

// sliderToVolumeMap maps 0-10 slider values to optimized volume levels (max 0.75)
var sliderToVolumeMap = [11]float64{
	0.0,  // 0: Complete silence
	0.02, // 1: Barely audible (more granular quiet control)
	0.05, // 2: Very quiet but audible
	0.10, // 3: Quiet (more options in quiet range)
	0.18, // 4: Low-medium
	0.30, // 5: Medium (still good perceived volume)
	0.43, // 6: Medium-high
	0.55, // 7: High
	0.65, // 8: Very high
	0.70, // 9: Near maximum
	0.75, // 10: Maximum (25% reduction from 1.0)
}

// ConvertSliderToVolume converts slider value (0-10) to volume amplitude (max 75%)
func ConvertSliderToVolume(sliderValue int) float64 {
	// Bounds checking
	if sliderValue < 0 {
		return 0.0
	}
	if sliderValue >= len(sliderToVolumeMap) {
		return sliderToVolumeMap[len(sliderToVolumeMap)-1]
	}

	return sliderToVolumeMap[sliderValue]
}
