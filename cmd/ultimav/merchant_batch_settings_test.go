package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

func TestMerchantBatchSettingStagesCancelsAndApplies(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "configuration.yaml")
	require.NoError(t, os.WriteFile(filepath.Join(directory, "DATA.OVL"), nil, 0o600))
	require.NoError(t, os.WriteFile(configPath, []byte("Gameplay:\n  Merchants:\n    BatchTheft: true\n"), 0o600))
	configuration, err := config.LoadUltimaVConfiguration(configPath, directory)
	require.NoError(t, err)
	before, err := os.ReadFile(configPath)
	require.NoError(t, err)
	gameplay := configuration.GameplayConfiguration
	placement := gameScreenPercents
	for _, cancel := range []bool{true, false} {
		dialog := newConfigurationDialogForTab(&placement, newPresentationTestKeyboard(), func() {},
			configuration, nil, audioSettingsGameplayTab)
		for range gameplayBatchMerchantTheftIndex {
			pressAudioSettingsKey(dialog, ebiten.KeyDown)
		}
		assert.Equal(t, "X", dialog.gameplayCheckmark(gameplayBatchMerchantTheftIndex))
		pressAudioSettingsKey(dialog, ebiten.KeySpace)
		assert.False(t, dialog.stagedGameplay.BatchMerchantTheft)
		assert.True(t, gameplay.BatchMerchantTheft)
		staged, readErr := os.ReadFile(configPath)
		require.NoError(t, readErr)
		assert.Equal(t, before, staged)
		if cancel {
			pressAudioSettingsKey(dialog, ebiten.KeyEscape)
			assert.True(t, gameplay.BatchMerchantTheft)
			continue
		}
		pressAudioSettingsKey(dialog, ebiten.KeyEnter)
		assert.False(t, dialog.IsVisible())
		assert.False(t, gameplay.BatchMerchantTheft)
		assert.Same(t, gameplay, configuration.GameplayConfiguration)
		persisted, loadErr := config.LoadUltimaVConfiguration(configPath, directory)
		require.NoError(t, loadErr)
		assert.False(t, persisted.GameplayConfiguration.BatchMerchantTheft)
	}
}
