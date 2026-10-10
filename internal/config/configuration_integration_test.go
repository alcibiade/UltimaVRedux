package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAudioConfigurationLoading tests that config properly loads and initializes audio configuration
func TestAudioConfigurationLoading(t *testing.T) {
	ultimaConfig, _ := configurationFixture(t, "{}\n")

	require.NotNil(t, ultimaConfig.AudioConfiguration, "AudioConfiguration should be initialized")

	// Log actual values for debugging (now integers)
	t.Logf("Loaded values - Music: %d, SFX: %d, Ambient: %d",
		ultimaConfig.AudioConfiguration.MusicVolume,
		ultimaConfig.AudioConfiguration.SoundEffectVolume,
		ultimaConfig.AudioConfiguration.AmbientVolume)

	// Test that values are loaded correctly (whether from config file or defaults)
	// Values should be within valid ranges for slider values (0-10)

	assert.GreaterOrEqual(t, ultimaConfig.AudioConfiguration.MusicVolume, 0, "Music volume should be >= 0")
	assert.LessOrEqual(t, ultimaConfig.AudioConfiguration.MusicVolume, 10, "Music volume should be <= 10")

	assert.GreaterOrEqual(t, ultimaConfig.AudioConfiguration.SoundEffectVolume, 0, "SFX volume should be >= 0")
	assert.LessOrEqual(t, ultimaConfig.AudioConfiguration.SoundEffectVolume, 10, "SFX volume should be <= 10")

	assert.GreaterOrEqual(t, ultimaConfig.AudioConfiguration.AmbientVolume, 0, "Ambient volume should be >= 0")
	assert.LessOrEqual(t, ultimaConfig.AudioConfiguration.AmbientVolume, 10, "Ambient volume should be <= 10")

	t.Log("Audio configuration loading test passed")
}
