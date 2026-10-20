package audio

import (
	"errors"
	"math"
)

// AudioConfiguration manages audio settings and volume controls with validation.
//
//nolint:revive // Preserve the configuration type used by scene and settings adapters.
type AudioConfiguration struct {
	// Volume controls (0.0 to 1.0)
	SoundEffectVolume float64
	MusicVolume       float64
	AmbientVolume     float64
}

// NewAudioConfiguration creates a new configuration with default settings
func NewAudioConfiguration() (*AudioConfiguration, error) {
	const defaultSoundEffects, defaultMusic, defaultAmbient = 6, 5, 4

	return &AudioConfiguration{
		// Default volume settings - values are pre-converted from slider scale via ConvertSliderToVolume
		SoundEffectVolume: ConvertSliderToVolume(defaultSoundEffects),
		MusicVolume:       ConvertSliderToVolume(defaultMusic),
		AmbientVolume:     ConvertSliderToVolume(defaultAmbient),
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

// Clone creates a deep copy of the configuration
func (ac *AudioConfiguration) Clone() *AudioConfiguration {
	return &AudioConfiguration{
		SoundEffectVolume: ac.SoundEffectVolume,
		MusicVolume:       ac.MusicVolume,
		AmbientVolume:     ac.AmbientVolume,
	}
}

var (
	errVolumeRange     = errors.New("volume must be between 0.0 and 1.0")
	errVolumeNonFinite = errors.New("volume must be a valid number")
)

// validateVolumeRange ensures volume is in valid 0.0 to 1.0 range
func validateVolumeRange(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return errVolumeRange
	}

	if math.IsNaN(volume) || math.IsInf(volume, 0) {
		return errVolumeNonFinite
	}

	return nil
}

// sliderToVolumeMap maps 0-10 slider values to optimized volume levels (max 0.75)
//
//nolint:gochecknoglobals // Read-only authored preference curve shared by game and editor monitoring.
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
