package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

func TestReduxModificationsStatusAndMutationPersist(t *testing.T) {
	configuration, configFile := newReduxModificationsTestConfiguration(t)

	status, handled := executeApplicationDebugCommand(debugging.Request{
		SessionID: "test", ID: "status", CommandID: debugging.CommandReduxModificationsStatus,
	}, configuration)
	require.True(t, handled)
	require.True(t, status.Success)
	assert.Equal(t, debugging.ExecutionQuery, status.EffectiveExecutionMode)
	assert.JSONEq(t, `{
		"combat":{"allowRepeatedMonsterDivision":false},
		"dungeons":{"persistHiddenDungeonDiscoveries":true,"showSubduedUnlitDungeons":true},
		"presentation":{"smoothSurfaceMovement":false},
		"navigation":{"flexibleSextantUse":false}
	}`, string(status.Data))

	result, handled := executeApplicationDebugCommand(debugging.Request{
		SessionID: "test", ID: "enable", CommandID: debugging.CommandReduxCombatRepeatedMonsterDivisionSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	}, configuration)
	require.True(t, handled)
	require.True(t, result.Success)
	assert.Equal(t, debugging.ExecutionLiveSafe, result.EffectiveExecutionMode)
	assert.True(t, configuration.GameplayConfiguration.AllowRepeatedMonsterDivision)
	require.Len(t, result.Changes, 1)
	assert.Equal(t, repeatedMonsterDivisionChangePath, result.Changes[0].Path)
	assert.JSONEq(t, `false`, string(result.Changes[0].Before))
	assert.JSONEq(t, `true`, string(result.Changes[0].After))

	persisted := viper.New()
	persisted.SetConfigFile(configFile)
	require.NoError(t, persisted.ReadInConfig())
	assert.True(t, persisted.GetBool("Gameplay.Combat.AllowRepeatedMonsterDivision"))
}

func TestReduxModificationsNoOpEmitsNoChange(t *testing.T) {
	configuration, _ := newReduxModificationsTestConfiguration(t)

	result, handled := executeApplicationDebugCommand(debugging.Request{
		SessionID: "test", ID: "keep-disabled", CommandID: debugging.CommandReduxCombatRepeatedMonsterDivisionSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`false`)},
	}, configuration)

	require.True(t, handled)
	require.True(t, result.Success)
	assert.Empty(t, result.Changes)
}

func TestReduxDungeonModificationsPersistTypedChanges(t *testing.T) {
	configuration, configFile := newReduxModificationsTestConfiguration(t)
	for _, test := range []struct {
		commandID string
		path      string
		get       func(*config.GameplayConfiguration) bool
	}{
		{
			commandID: debugging.CommandReduxDungeonsPersistHiddenDiscoveriesSet,
			path:      persistHiddenDungeonDiscoveriesChangePath,
			get: func(gameplay *config.GameplayConfiguration) bool {
				return gameplay.PersistHiddenDungeonDiscoveries
			},
		},
		{
			commandID: debugging.CommandReduxDungeonsSubduedUnlitVisibilitySet,
			path:      showSubduedUnlitDungeonsChangePath,
			get: func(gameplay *config.GameplayConfiguration) bool {
				return gameplay.ShowSubduedUnlitDungeons
			},
		},
	} {
		result, handled := executeApplicationDebugCommand(debugging.Request{
			SessionID: "test", ID: test.commandID, CommandID: test.commandID,
			Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`false`)},
		}, configuration)

		require.True(t, handled)
		require.True(t, result.Success, result.ErrorMessage)
		assert.Equal(t, debugging.ExecutionLiveSafe, result.EffectiveExecutionMode)
		assert.False(t, test.get(configuration.GameplayConfiguration))
		require.Len(t, result.Changes, 1)
		assert.Equal(t, test.path, result.Changes[0].Path)
		assert.JSONEq(t, `true`, string(result.Changes[0].Before))
		assert.JSONEq(t, `false`, string(result.Changes[0].After))
	}

	persisted := viper.New()
	persisted.SetConfigFile(configFile)
	require.NoError(t, persisted.ReadInConfig())
	assert.False(t, persisted.GetBool("Gameplay.Dungeons.PersistHiddenDiscoveries"))
	assert.False(t, persisted.GetBool("Gameplay.Dungeons.ShowSubduedUnlitDungeons"))
}

func TestReduxSmoothSurfaceMovementPersistsAndRollsBack(t *testing.T) {
	configuration, configFile := newReduxModificationsTestConfiguration(t)
	gameplay := configuration.GameplayConfiguration
	request := debugging.Request{
		SessionID: "test", ID: "smooth", CommandID: debugging.CommandReduxPresentationSmoothSurfaceMovementSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	}
	for _, enabled := range []bool{true, true, false} {
		before := gameplay.SmoothSurfaceMovement
		encoded, err := json.Marshal(enabled)
		require.NoError(t, err)
		request.Arguments["enabled"] = encoded
		result, handled := executeApplicationDebugCommand(request, configuration)
		require.True(t, handled)
		require.True(t, result.Success, result.ErrorMessage)
		assert.Equal(t, debugging.ExecutionLiveSafe, result.EffectiveExecutionMode)
		assert.Same(t, gameplay, configuration.GameplayConfiguration)
		assert.Equal(t, enabled, gameplay.SmoothSurfaceMovement)
		var status debugging.ReduxModificationsStatusData
		require.NoError(t, json.Unmarshal(result.Data, &status))
		assert.Equal(t, enabled, status.Presentation.SmoothSurfaceMovement)
		if before == enabled {
			assert.Empty(t, result.Changes)
		} else {
			require.Len(t, result.Changes, 1)
			assert.Equal(t, smoothSurfaceMovementChangePath, result.Changes[0].Path)
			assert.JSONEq(t, string(encoded), string(result.Changes[0].After))
		}
		persisted := viper.New()
		persisted.SetConfigFile(configFile)
		require.NoError(t, persisted.ReadInConfig())
		assert.Equal(t, enabled, persisted.GetBool("Gameplay.Presentation.SmoothSurfaceMovement"))
	}
	require.NoError(t, os.Remove(configFile))
	require.NoError(t, os.Mkdir(configFile, 0o700))
	request.Arguments["enabled"] = json.RawMessage(`true`)
	result, handled := executeApplicationDebugCommand(request, configuration)
	require.True(t, handled)
	assert.False(t, result.Success)
	assert.Empty(t, result.Changes)
	assert.False(t, gameplay.SmoothSurfaceMovement)
	assert.Same(t, gameplay, configuration.GameplayConfiguration)
}

func TestReduxFlexibleSextantUsePersistsAndRollsBack(t *testing.T) {
	configuration, configFile := newReduxModificationsTestConfiguration(t)
	gameplay := configuration.GameplayConfiguration
	request := debugging.Request{
		SessionID: "test", ID: "sextant", CommandID: debugging.CommandReduxNavigationFlexibleSextantUseSet,
		Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)},
	}
	for _, enabled := range []bool{true, true, false} {
		before := gameplay.FlexibleSextantUse
		encoded, err := json.Marshal(enabled)
		require.NoError(t, err)
		request.Arguments["enabled"] = encoded
		result, handled := executeApplicationDebugCommand(request, configuration)
		require.True(t, handled)
		require.True(t, result.Success, result.ErrorMessage)
		assert.Equal(t, debugging.ExecutionLiveSafe, result.EffectiveExecutionMode)
		assert.Same(t, gameplay, configuration.GameplayConfiguration)
		assert.Equal(t, enabled, gameplay.FlexibleSextantUse)
		var status debugging.ReduxModificationsStatusData
		require.NoError(t, json.Unmarshal(result.Data, &status))
		assert.Equal(t, enabled, status.Navigation.FlexibleSextantUse)
		if before == enabled {
			assert.Empty(t, result.Changes)
		} else {
			require.Len(t, result.Changes, 1)
			assert.Equal(t, flexibleSextantUseChangePath, result.Changes[0].Path)
			assert.JSONEq(t, string(encoded), string(result.Changes[0].After))
			beforeJSON, err := json.Marshal(before)
			require.NoError(t, err)
			assert.JSONEq(t, string(beforeJSON), string(result.Changes[0].Before))
		}
		persisted := viper.New()
		persisted.SetConfigFile(configFile)
		require.NoError(t, persisted.ReadInConfig())
		assert.Equal(t, enabled, persisted.GetBool("Gameplay.Navigation.FlexibleSextantUse"))
	}
	require.NoError(t, os.Remove(configFile))
	require.NoError(t, os.Mkdir(configFile, 0o700))
	request.Arguments["enabled"] = json.RawMessage(`true`)
	result, handled := executeApplicationDebugCommand(request, configuration)
	require.True(t, handled)
	assert.False(t, result.Success)
	assert.Equal(t, "gameplay_configuration_save_failed", result.ErrorCode)
	assert.Empty(t, result.Changes)
	assert.False(t, gameplay.FlexibleSextantUse)
	assert.False(t, newReduxModificationsStatusData(configuration).Navigation.FlexibleSextantUse)
	assert.Same(t, gameplay, configuration.GameplayConfiguration)
}

func newReduxModificationsTestConfiguration(t *testing.T) (*config.UltimaVConfiguration, string) {
	t.Helper()

	return initializeDialogConfigFile(t, "redux_modifications")
}
