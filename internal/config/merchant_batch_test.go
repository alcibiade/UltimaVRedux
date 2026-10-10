package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerchantBatchTheftDefaultsAndSerialization(t *testing.T) {
	configuration, configPath := configurationFixture(t, "")
	assert.True(t, configuration.GameplayConfiguration.BatchMerchantTheft)
	for _, enabled := range []bool{false, true} {
		configuration.GameplayConfiguration.BatchMerchantTheft = enabled
		require.NoError(t, configuration.SaveGameplayConfiguration())
		loaded, loadErr := LoadUltimaVConfiguration(configPath, filepath.Dir(configPath))
		require.NoError(t, loadErr)
		assert.Equal(t, enabled, loaded.GameplayConfiguration.BatchMerchantTheft)
	}
}
