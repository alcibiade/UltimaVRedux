package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkbenchQoLCanonicalPreferencesAndDialogConflict(t *testing.T) {
	configuration, _ := newReduxModificationsTestConfiguration(t)
	for _, test := range []struct{ command, parameter, value string }{
		{debugging.CommandCombatPresentationSpeedSet, "percent", "250"},
		{debugging.CommandReduxConfirmQuickSaveSet, "enabled", "false"},
		{debugging.CommandReduxCombatHPBarsSet, "enabled", "false"},
		{debugging.CommandReduxActionPopupsSet, "count", "0"},
	} {
		request := debugging.Request{SessionID: "test", ID: "qol", CommandID: test.command,
			Arguments: map[string]json.RawMessage{test.parameter: json.RawMessage(test.value)}}
		release := configuration.BeginConfigurationEditing()
		result, handled := executeReduxModificationsDebugCommand(request, configuration)
		require.True(t, handled)
		assert.Equal(t, "configuration_editing", result.ErrorCode)
		assert.Empty(t, result.Changes)
		release()
		result, handled = executeReduxModificationsDebugCommand(request, configuration)
		require.True(t, handled)
		require.True(t, result.Success, "%+v", result)
		assert.Len(t, result.Changes, 1)
		result, _ = executeReduxModificationsDebugCommand(request, configuration)
		assert.Empty(t, result.Changes, "no-op changes are omitted")
	}
	assert.Equal(t, 2.5, configuration.GetCombatSpeedFactor())
	assert.False(t, configuration.GetConfirmQuickSave())
	assert.False(t, configuration.GetShowCombatHPBars())
	assert.Zero(t, configuration.GetMaxVisibleActionPopups())
	status := newReduxModificationsStatusData(configuration)
	assert.Equal(t, 250, status.Combat.SpeedPercent)
	assert.False(t, status.ConfigurationEditing)
}

func TestWorkbenchLegacySpeedRequestsRejectAndWriteFailureRollsBack(t *testing.T) {
	configuration, file := newReduxModificationsTestConfiguration(t)
	request := debugging.Request{SessionID: "test", ID: "speed", CommandID: debugging.CommandCombatPresentationSpeedSet,
		Arguments: map[string]json.RawMessage{"percent": json.RawMessage(`75`)}}
	result, handled := executeReduxModificationsDebugCommand(request, configuration)
	require.True(t, handled)
	assert.False(t, result.Success)
	assert.Equal(t, "invalid_argument", result.ErrorCode)
	assert.Equal(t, 1.0, configuration.GetCombatSpeedFactor())
	require.NoError(t, os.Remove(file))
	require.NoError(t, os.Mkdir(file, 0o700))
	request.Arguments["percent"] = json.RawMessage(`200`)
	result, _ = executeReduxModificationsDebugCommand(request, configuration)
	assert.False(t, result.Success)
	assert.Empty(t, result.Changes)
	assert.Equal(t, 1.0, configuration.GetCombatSpeedFactor())
}
