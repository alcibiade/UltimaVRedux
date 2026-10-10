package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadStartupConfigurationRetainsPreferencesWhenOriginalDataIsMissing(t *testing.T) {
	home := isolatedConfigurationHome(t)
	directory := filepath.Join(home, ".ultima_v_redux")
	require.NoError(t, os.Mkdir(directory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "config.yaml"), []byte(
		"DataFilePath: /missing/ultima-v\nAudio:\n  MusicVolume: 2\n",
	), 0o600))

	result := LoadStartupConfiguration()
	require.NotNil(t, result.Configuration)
	assert.NoError(t, result.ConfigurationError)
	require.Error(t, result.OriginalDataError)
	assert.Equal(t, 2, result.Configuration.AudioConfiguration.MusicVolume)
	assert.Equal(t, "/missing/ultima-v", result.Configuration.SavedConfigData.DataFilePath)
	assert.Empty(t, result.Configuration.RawDataOvl)
}

func TestLoadStartupConfigurationSurvivesMalformedSettings(t *testing.T) {
	home := isolatedConfigurationHome(t)
	directory := filepath.Join(home, ".ultima_v_redux")
	require.NoError(t, os.Mkdir(directory, 0o700))
	file := filepath.Join(directory, "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte("invalid: ["), 0o600))

	result := LoadStartupConfiguration()
	require.NotNil(t, result.Configuration)
	require.Error(t, result.ConfigurationError)
	assert.NoError(t, result.OriginalDataError)
	assert.Equal(t, "./OLD", result.Configuration.SavedConfigData.DataFilePath)
	path, err := result.Configuration.ConfigurationFilePath()
	require.NoError(t, err)
	assert.Equal(t, file, path)

	err = result.Configuration.PersistDataFilePath(filepath.Join(home, "Documents", "UltimaV"))
	require.ErrorContains(t, err, "rereading configuration")
	contents, readErr := os.ReadFile(file)
	require.NoError(t, readErr)
	assert.Equal(t, []byte("invalid: ["), contents)
}

func TestLoadStartupConfigurationSurvivesInvalidTypedSettings(t *testing.T) {
	home := isolatedConfigurationHome(t)
	directory := filepath.Join(home, ".ultima_v_redux")
	require.NoError(t, os.Mkdir(directory, 0o700))
	file := filepath.Join(directory, "config.yaml")
	require.NoError(t, os.WriteFile(file, []byte("Resolution: [1, 2]\n"), 0o600))

	result := LoadStartupConfiguration()

	require.NotNil(t, result.Configuration)
	require.ErrorContains(t, result.ConfigurationError, "decoding configuration")
	assert.NoError(t, result.OriginalDataError)
	assert.Equal(t, "./OLD", result.Configuration.SavedConfigData.DataFilePath)
	path, err := result.Configuration.ConfigurationFilePath()
	require.NoError(t, err)
	assert.Equal(t, file, path)
}

func TestPersistDataFilePathPreservesActiveDataAndFreshExternalEdits(t *testing.T) {
	configuration, file := configurationFixture(t, "DataFilePath: old/path\nUnknown: initial\n")
	activePath := configuration.SavedConfigData.DataFilePath
	activeData := append([]byte(nil), configuration.RawDataOvl...)
	selected := filepath.Join(t.TempDir(), "Documents", "UltimaV")
	require.NoError(t, os.WriteFile(file, []byte("DataFilePath: old/path\nUnknown: externally-edited\n"), 0o600))

	require.NoError(t, configuration.PersistDataFilePath(selected))
	assert.Equal(t, activePath, configuration.SavedConfigData.DataFilePath)
	assert.True(t, slices.Equal(activeData, configuration.RawDataOvl))
	assert.Equal(t, selected, configuration.PersistedDataFilePath())
	source := readConfigurationFixture(t, file)
	assert.Equal(t, selected, source.GetString("DataFilePath"))
	assert.Equal(t, "externally-edited", source.GetString("Unknown"))

	configuration.AudioConfiguration.MusicVolume = 8
	require.NoError(t, configuration.SaveAudioConfiguration())
	source = readConfigurationFixture(t, file)
	assert.Equal(t, selected, source.GetString("DataFilePath"))
	assert.Equal(t, "externally-edited", source.GetString("Unknown"))
}

func TestSuccessfulLegacyImportHistoryIsSeparateFromDataSelection(t *testing.T) {
	configuration, file := configurationFixture(t, "DataFilePath: shipped/path\n")
	importDirectory := filepath.Join(t.TempDir(), "legacy saves")
	require.NoError(t, configuration.PersistLastLegacyImportDirectory(importDirectory))

	assert.Equal(t, importDirectory, configuration.LastLegacyImportDirectory())
	assert.Equal(t, "shipped/path", readConfigurationFixture(t, file).GetString("DataFilePath"))
}

func TestSelectingActiveDataAgainClearsPendingNextLaunchPath(t *testing.T) {
	configuration, file := configurationFixture(t, "DataFilePath: shipped/path\n")
	active := configuration.SavedConfigData.DataFilePath
	pending := filepath.Join(t.TempDir(), "pending")
	require.NoError(t, configuration.PersistDataFilePath(pending))
	assert.Equal(t, pending, configuration.PersistedDataFilePath())

	require.NoError(t, configuration.PersistDataFilePath(active))

	assert.Equal(t, active, configuration.SavedConfigData.DataFilePath)
	assert.Equal(t, active, configuration.PersistedDataFilePath())
	assert.Equal(t, active, readConfigurationFixture(t, file).GetString("DataFilePath"))
}

func TestRefreshPersistedConfigurationDoesNotHotApplyExternalPreferences(t *testing.T) {
	configuration, file := configurationFixture(t, "DataFilePath: shipped/path\nAudio:\n  MusicVolume: 2\n")
	activePath := configuration.SavedConfigData.DataFilePath
	activeMusic := configuration.AudioConfiguration.MusicVolume
	pending := filepath.Join(t.TempDir(), "next launch")
	require.NoError(t, os.WriteFile(file, []byte(
		"DataFilePath: "+pending+"\nAudio:\n  MusicVolume: 9\n",
	), 0o600))

	require.NoError(t, configuration.RefreshPersistedConfiguration())

	assert.Equal(t, pending, configuration.PersistedDataFilePath())
	assert.Equal(t, activePath, configuration.SavedConfigData.DataFilePath)
	assert.Equal(t, activeMusic, configuration.AudioConfiguration.MusicVolume)
}

func TestSaveConfigurationRecreatesFileDeletedAfterStartup(t *testing.T) {
	configuration, file := configurationFixture(t, "DataFilePath: shipped/path\nUnknown: retained\n")
	require.NoError(t, os.Remove(file))
	configuration.AudioConfiguration.MusicVolume = 8

	require.NoError(t, configuration.SaveAudioConfiguration())

	source := readConfigurationFixture(t, file)
	assert.Equal(t, "shipped/path", source.GetString("DataFilePath"))
	assert.Equal(t, "retained", source.GetString("Unknown"))
	assert.Equal(t, 8, source.GetInt("Audio.MusicVolume"))
}
