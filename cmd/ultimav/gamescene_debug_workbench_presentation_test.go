package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestExecutePresentationOverlayCommandsUseExplicitState(t *testing.T) {
	scene := &GameScene{
		gameState:       &game_state.GameState{},
		overlayRenderer: NewOverlayRenderer(),
	}

	for _, testCase := range []struct {
		commandID string
		path      string
		assertSet func()
	}{
		{debugging.CommandLightingOverlaySet, "presentation.overlays.lighting", func() { assert.True(t, scene.gameState.DebugOptions.LightOverlay) }},
		{debugging.CommandSearchOverlaySet, "presentation.overlays.search", func() { assert.True(t, scene.gameState.DebugOptions.SearchOverlay) }},
		{debugging.CommandMoongateOverlaySet, "presentation.overlays.moongate", func() { assert.True(t, scene.overlayRenderer.IsMoonstoneOverlayEnabled()) }},
	} {
		result := scene.ExecuteDebugCommand(debugging.Request{SessionID: "test", ID: testCase.commandID, CommandID: testCase.commandID,
			Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}})
		require.True(t, result.Success, "code=%s message=%s", result.ErrorCode, result.ErrorMessage)
		assert.Equal(t, debugging.ExecutionPresentation, result.EffectiveExecutionMode)
		require.Len(t, result.Changes, 1)
		assert.Equal(t, testCase.path, result.Changes[0].Path)
		testCase.assertSet()
	}
}

func TestExecuteCombatPresentationSpeedPersistsFactor(t *testing.T) {
	gameConfig, configFile := initializeDialogConfigFile(t, "combat_speed_workbench")
	// Persistent preferences have an application owner, available outside combat
	// and before gameplay. Exercise catalog validation and that registry dispatch.
	game := &Game{
		currentScene:         &GameScene{gameConfig: gameConfig},
		debugCatalog:         debugging.NewInitialCatalog(),
		debugCommandRegistry: initialDebugCommandRegistry,
	}
	game.sceneGeneration.Store(1)
	require.False(t, gameConfig.ConfigurationEditing())

	request := debugging.Request{SessionID: "test", ID: "speed", CommandID: debugging.CommandCombatPresentationSpeedSet,
		Arguments: map[string]json.RawMessage{"percent": json.RawMessage(`200`)}}
	result := game.executeDebugCommand(request)

	require.True(t, result.Success, "code=%s message=%s", result.ErrorCode, result.ErrorMessage)
	assert.Equal(t, debugging.ExecutionPresentation, result.EffectiveExecutionMode)
	require.NotNil(t, result.Invocation)
	assert.JSONEq(t, `200`, string(result.Invocation.Arguments["percent"]))
	assert.JSONEq(t, `{"percent":200}`, string(result.Data))
	assert.Equal(t, 2.0, gameConfig.GetCombatSpeedFactor())
	persisted := viper.New()
	persisted.SetConfigFile(configFile)
	require.NoError(t, persisted.ReadInConfig())
	assert.Equal(t, 2.0, persisted.GetFloat64("Gameplay.Combat.SpeedFactor"))
	beforeFile, err := os.ReadFile(configFile)
	require.NoError(t, err)

	request.ID = "invalid-speed"
	request.Arguments["percent"] = json.RawMessage(`75`)
	result = game.executeDebugCommand(request)
	assert.False(t, result.Success)
	assert.Equal(t, "invalid_request", result.ErrorCode, "message=%s", result.ErrorMessage)
	assert.Empty(t, result.Changes)
	assert.Equal(t, 2.0, gameConfig.GetCombatSpeedFactor())

	release := gameConfig.BeginConfigurationEditing()
	t.Cleanup(release)
	request.ID = "blocked-speed"
	request.Arguments["percent"] = json.RawMessage(`300`)
	result = game.executeDebugCommand(request)
	assert.False(t, result.Success)
	assert.Equal(t, "configuration_editing", result.ErrorCode, "message=%s", result.ErrorMessage)
	assert.Contains(t, result.ErrorMessage, "Apply or Cancel")
	assert.Empty(t, result.Changes)
	assert.Equal(t, 2.0, gameConfig.GetCombatSpeedFactor())
	afterFile, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Equal(t, beforeFile, afterFile)

	release()
	request.ID = "unblocked-speed"
	result = game.executeDebugCommand(request)
	require.True(t, result.Success, "code=%s message=%s", result.ErrorCode, result.ErrorMessage)
	assert.Equal(t, 3.0, gameConfig.GetCombatSpeedFactor())
	require.NoError(t, persisted.ReadInConfig())
	assert.Equal(t, 3.0, persisted.GetFloat64("Gameplay.Combat.SpeedFactor"))
}
