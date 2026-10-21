package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDisplayConfigurationUsesCompleteExplicitValues(t *testing.T) {
	for _, test := range []struct {
		name   string
		mode   Mode
		width  int
		height int
	}{
		{name: "windowed custom", mode: ModeWindowed, width: 1700, height: 950},
		{name: "maximized", mode: ModeMaximized, width: 1920, height: 1080},
		{name: "fullscreen", mode: ModeFullscreen, width: 2560, height: 1440},
	} {
		t.Run(test.name, func(t *testing.T) {
			configuration := displayLoadFixture()
			configuration.source.Set("Display.Mode", test.mode.String())
			configuration.source.Set("Display.WindowWidth", test.width)
			configuration.source.Set("Display.WindowHeight", test.height)
			configuration.source.Set("Display.TargetFPS", 60)
			configuration.source.Set("Display.EnableFpsLimiting", false)
			configuration.source.Set("Resolution", 0)
			configuration.source.Set("FullScreen", test.mode != ModeFullscreen)
			configuration.loadDisplayConfiguration()

			assert.Equal(t, test.mode, configuration.DisplayConfiguration.Mode)
			assert.Equal(t, test.width, configuration.DisplayConfiguration.WindowWidth)
			assert.Equal(t, test.height, configuration.DisplayConfiguration.WindowHeight)
			assert.Equal(t, 60, configuration.DisplayConfiguration.TargetFPS)
			assert.False(t, configuration.DisplayConfiguration.EnableFpsLimiting)
		})
	}
}

func TestLoadDisplayConfigurationMigratesLegacyPresetAndFullscreen(t *testing.T) {
	for _, test := range []struct {
		name       string
		index      int
		fullscreen bool
		mode       Mode
	}{
		{name: "windowed", index: 1, mode: ModeWindowed},
		{name: "fullscreen", index: 3, fullscreen: true, mode: ModeFullscreen},
	} {
		t.Run(test.name, func(t *testing.T) {
			configuration := displayLoadFixture()
			configuration.source.Set("Resolution", test.index)
			configuration.source.Set("FullScreen", test.fullscreen)
			configuration.loadDisplayConfiguration()
			expected := SupportedWindowResolutions()[test.index]

			assert.Equal(t, test.mode, configuration.DisplayConfiguration.Mode)
			assert.Equal(t, expected.X, configuration.DisplayConfiguration.WindowWidth)
			assert.Equal(t, expected.Y, configuration.DisplayConfiguration.WindowHeight)
		})
	}
}

func TestLoadDisplayConfigurationFallsBackForInvalidPersistedValues(t *testing.T) {
	for _, test := range []struct {
		name     string
		settings map[string]any
	}{
		{
			name: "partial explicit",
			settings: map[string]any{
				"Display.Mode": "fullscreen",
			},
		},
		{
			name: "unknown mode",
			settings: map[string]any{
				"Display.Mode":         "borderless",
				"Display.WindowWidth":  1920,
				"Display.WindowHeight": 1080,
			},
		},
		{
			name: "malformed width",
			settings: map[string]any{
				"Display.Mode":         "windowed",
				"Display.WindowWidth":  "1920",
				"Display.WindowHeight": 1080,
			},
		},
		{
			name: "below minimum",
			settings: map[string]any{
				"Display.Mode":         "windowed",
				"Display.WindowWidth":  MinimumWindowUnitsWidth - 1,
				"Display.WindowHeight": MinimumWindowHeight,
			},
		},
		{
			name: "invalid legacy index",
			settings: map[string]any{
				"Resolution": 999,
				"FullScreen": true,
			},
		},
		{
			name: "malformed legacy index",
			settings: map[string]any{
				"Resolution": "2",
				"FullScreen": true,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			configuration := displayLoadFixture()
			for key, value := range test.settings {
				configuration.source.Set(key, value)
			}

			assert.NotPanics(t, configuration.loadDisplayConfiguration)

			assert.Equal(t, ModeWindowed, configuration.DisplayConfiguration.Mode)
			assert.Equal(t, MinimumWindowWidth, configuration.DisplayConfiguration.WindowWidth)
			assert.Equal(t, MinimumWindowHeight, configuration.DisplayConfiguration.WindowHeight)
		})
	}
}

func TestSaveConfigurationSettingsPreservesDisplayWhenLegacyCallerOmitsIt(t *testing.T) {
	configuration, configFile := configurationFixture(t, "existing: true\n")
	displayConfiguration := &DisplayConfiguration{
		Mode:              ModeMaximized,
		WindowWidth:       1700,
		WindowHeight:      950,
		TargetFPS:         60,
		EnableFpsLimiting: true,
	}
	configuration.DisplayConfiguration = displayConfiguration

	require.NoError(t, configuration.SaveConfigurationSettings(ConfigurationSettings{Gameplay: *DefaultGameplayConfiguration()}))
	assert.Equal(t, DisplayConfiguration{
		Mode:              ModeMaximized,
		WindowWidth:       1700,
		WindowHeight:      950,
		TargetFPS:         60,
		EnableFpsLimiting: true,
	}, *displayConfiguration)

	source := readConfigurationFixture(t, configFile)
	assert.Equal(t, "maximized", source.GetString("Display.Mode"))
	assert.Equal(t, 1700, source.GetInt("Display.WindowWidth"))
	assert.Equal(t, 950, source.GetInt("Display.WindowHeight"))
}

func TestUpdateSaveFilePersistsDisplayAtomically(t *testing.T) {
	configuration, configFile := configurationFixture(t, "existing: true\n")
	displayConfiguration := &DisplayConfiguration{
		Mode:              ModeWindowed,
		WindowWidth:       1777,
		WindowHeight:      999,
		TargetFPS:         30,
		EnableFpsLimiting: true,
	}
	configuration.DisplayConfiguration = displayConfiguration

	require.NoError(t, configuration.UpdateSaveFile())
	assert.Same(t, displayConfiguration, configuration.DisplayConfiguration)

	source := readConfigurationFixture(t, configFile)
	assert.Equal(t, "windowed", source.GetString("Display.Mode"))
	assert.Equal(t, 1777, source.GetInt("Display.WindowWidth"))
	assert.Equal(t, 999, source.GetInt("Display.WindowHeight"))
}

func TestSaveConfigurationSettingsRequiresDisplayModel(t *testing.T) {
	configuration := &UltimaVConfiguration{
		AudioConfiguration:    &AudioConfiguration{},
		GameplayConfiguration: &GameplayConfiguration{},
	}

	require.ErrorIs(t, configuration.SaveConfigurationSettings(ConfigurationSettings{}), ErrDisplayConfigNil)
}

func displayLoadFixture() *UltimaVConfiguration {
	return &UltimaVConfiguration{
		source:           viper.New(),
		SavedConfigData:  &UltimaVConfigurationFlags{},
		allWindowConfigs: SupportedWindowResolutions(),
	}
}
