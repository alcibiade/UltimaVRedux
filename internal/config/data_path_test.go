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
	assert.Empty(t, result.Configuration.SavedConfigData.DataFilePath)
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
	assert.Empty(t, result.Configuration.SavedConfigData.DataFilePath)
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

func TestOldLegacyImportHistoryRemainsInertAndPreserved(t *testing.T) {
	configuration, file := configurationFixture(t, "DataFilePath: shipped/path\nLegacyImport:\n  LastSuccessfulDirectory: history-only\n")
	selected := filepath.Join(t.TempDir(), "UltimaV")
	require.NoError(t, configuration.PersistDataFilePath(selected))
	assert.Equal(t, "history-only", readConfigurationFixture(t, file).GetString("LegacyImport.LastSuccessfulDirectory"))
	assert.Equal(t, selected, readConfigurationFixture(t, file).GetString("DataFilePath"))
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

func TestUnsetDataPathNeverLoadsWorkingDirectoryData(t *testing.T) {
	home := isolatedConfigurationHome(t)
	t.Chdir(home)
	require.NoError(t, os.WriteFile("DATA.OVL", make([]byte, startupDataOvlSize), 0o600))
	for _, contents := range []string{"Unknown: retained\n", "DataFilePath: ''\n", "DataFilePath: '  '\n"} {
		file := filepath.Join(home, "preferences.yaml")
		require.NoError(t, os.WriteFile(file, []byte(contents), 0o600))
		_, err := LoadUltimaVConfiguration(file, "")
		require.ErrorIs(t, err, ErrDataFilePathUnset)
	}
	startup := LoadStartupConfiguration()
	require.NoError(t, startup.ConfigurationError)
	require.ErrorIs(t, startup.OriginalDataError, ErrDataFilePathUnset)
	assert.Empty(t, startup.Configuration.SavedConfigData.DataFilePath)
	assert.Empty(t, startup.Configuration.RawDataOvl)
}

func TestStartupPreservesExplicitOldPathWithoutWriting(t *testing.T) {
	home := isolatedConfigurationHome(t)
	t.Chdir(home)
	require.NoError(t, os.Mkdir("OLD", 0o700))
	require.NoError(t, os.WriteFile(filepath.Join("OLD", "DATA.OVL"), make([]byte, startupDataOvlSize), 0o600))
	directory := filepath.Join(home, ".ultima_v_redux")
	require.NoError(t, os.Mkdir(directory, 0o700))
	file := filepath.Join(directory, "config.yaml")
	before := []byte("# explicitly chosen\nDataFilePath: ./OLD\n")
	require.NoError(t, os.WriteFile(file, before, 0o600))
	startup := LoadStartupConfiguration()
	require.NoError(t, startup.ConfigurationError)
	require.NoError(t, startup.OriginalDataError)
	assert.Equal(t, "./OLD", startup.Configuration.SavedConfigData.DataFilePath)
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestInvalidTypedPreferencesCannotBeAdoptedOrSilentlyRepairedBySaving(t *testing.T) {
	for _, invalid := range []string{"Resolution: [1, 2]\n", "Audio:\n  MusicVolume: many\n", "Gameplay:\n  Merchants:\n    BatchTheft: [false]\n"} {
		t.Run(invalid, func(t *testing.T) {
			configuration, file := configurationFixture(t, "DataFilePath: previous\nAudio:\n  MusicVolume: 2\n")
			require.NoError(t, os.WriteFile(file, []byte(invalid), 0o600))
			require.ErrorContains(t, configuration.RefreshPersistedConfiguration(), "decoding configuration")
			assert.Equal(t, "previous", configuration.PersistedDataFilePath())
			require.ErrorContains(t, configuration.PersistDataFilePath(t.TempDir()), "decoding configuration")
			require.ErrorContains(t, configuration.UpdateSaveFile(), "decoding configuration")
			after, err := os.ReadFile(file)
			require.NoError(t, err)
			assert.Equal(t, invalid, string(after))
			require.NoError(t, os.WriteFile(file, []byte("DataFilePath: repaired\nAudio:\n  MusicVolume: 9\n"), 0o600))
			require.NoError(t, configuration.RefreshPersistedConfiguration())
			assert.Equal(t, "repaired", configuration.PersistedDataFilePath())
			assert.Equal(t, 2, configuration.AudioConfiguration.MusicVolume)
		})
	}
}

func TestFinalPreferenceSavePreservesNextLaunchExternalPreferences(t *testing.T) {
	configuration, file := configurationFixture(t, "Audio:\n  MusicVolume: 2\n")
	require.NoError(t, os.WriteFile(file, []byte("Audio:\n  MusicVolume: 9\nUnknown: changed\n"), 0o600))
	require.NoError(t, configuration.RefreshPersistedConfiguration())
	configuration.DisplayConfiguration.WindowWidth = 1700
	require.NoError(t, configuration.UpdateSaveFile())
	source := readConfigurationFixture(t, file)
	assert.Equal(t, 9, source.GetInt("Audio.MusicVolume"))
	assert.Equal(t, 1700, source.GetInt("Display.WindowWidth"))
	assert.Equal(t, "changed", source.GetString("Unknown"))
	assert.Equal(t, 2, configuration.AudioConfiguration.MusicVolume)
}

func TestUnavailableHomeNeverCreatesWorkingDirectorySettings(t *testing.T) {
	working := t.TempDir()
	t.Chdir(working)
	for _, home := range []string{"", "relative", filepath.Join(working, "missing-home")} {
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		startup := LoadStartupConfiguration()
		require.Error(t, startup.ConfigurationError)
		assert.Nil(t, startup.Configuration)
		_, err := os.Stat(filepath.Join(working, ".ultima_v_redux"))
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	// Explicit constructors remain independent of HOME and cannot persist there.
	require.NoError(t, os.WriteFile(filepath.Join(working, "DATA.OVL"), nil, 0o600))
	configuration, err := NewUltimaVConfigurationWithPath(working)
	require.NoError(t, err)
	require.ErrorIs(t, configuration.UpdateSaveFile(), ErrConfigFileUnknown)
}

func TestRecoveryRetainsSelectedAlternateYAMLFile(t *testing.T) {
	home := isolatedConfigurationHome(t)
	directory := filepath.Join(home, ".ultima_v_redux")
	require.NoError(t, os.Mkdir(directory, 0o700))
	file := filepath.Join(directory, "config.yml")
	for _, invalid := range []string{"invalid: [", "Resolution: [1, 2]\n"} {
		require.NoError(t, os.WriteFile(file, []byte(invalid), 0o600))
		startup := LoadStartupConfiguration()
		require.Error(t, startup.ConfigurationError)
		require.NotNil(t, startup.Configuration)
		selected, err := startup.Configuration.ConfigurationFilePath()
		require.NoError(t, err)
		assert.Equal(t, file, selected)
		_, err = os.Stat(filepath.Join(directory, "config.yaml"))
		require.ErrorIs(t, err, os.ErrNotExist)
	}
}
