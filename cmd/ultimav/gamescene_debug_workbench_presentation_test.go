package main

import (
	"encoding/json"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestExecutePresentationOverlayCommandsUseExplicitState(t *testing.T) {
	scene := &GameScene{ //nolint:exhaustruct // Focused presentation-command fixture.
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
		require.True(t, result.Success)
		assert.Equal(t, debugging.ExecutionPresentation, result.EffectiveExecutionMode)
		require.Len(t, result.Changes, 1)
		assert.Equal(t, testCase.path, result.Changes[0].Path)
		testCase.assertSet()
	}
}

func TestExecuteCombatPresentationSpeedPersistsFactor(t *testing.T) {
	gameConfig, configFile := initializeDialogConfigFile(t, "combat_speed_workbench")
	scene := &GameScene{gameConfig: gameConfig, gameState: &game_state.GameState{}, overlayRenderer: NewOverlayRenderer()} //nolint:exhaustruct // Focused presentation fixture.

	result := scene.ExecuteDebugCommand(debugging.Request{SessionID: "test", ID: "speed", CommandID: debugging.CommandCombatPresentationSpeedSet,
		Arguments: map[string]json.RawMessage{"percent": json.RawMessage(`200`)}})

	require.True(t, result.Success)
	assert.Equal(t, 2.0, gameConfig.DebugConfiguration.GetCombatMovementSpeedFactor())
	persisted := viper.New()
	persisted.SetConfigFile(configFile)
	require.NoError(t, persisted.ReadInConfig())
	assert.Equal(t, 2.0, persisted.GetFloat64("Debug.CombatMovementSpeedFactor"))
}
