package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestCombatYellPromptsBeforeCompletingAction(t *testing.T) {
	scene, _ := newCombatYellTestScene(t)
	identityBefore := scene.gameState.CombatScene.GetCurrentTurnIdentity()

	scene.combatMapInputHandler(ebiten.KeyY)

	assert.True(t, scene.gameState.EventQueueManager.HasWorkflow("combat_yell"))
	assert.Equal(t, identityBefore, scene.gameState.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
}

func TestCombatYellCompletesOneAction(t *testing.T) {
	tests := []struct {
		name    string
		input   *ebiten.Key
		message string
	}{
		{name: "empty input", message: "Nothing"},
		{name: "nonempty input", input: keyPointer(ebiten.KeyN), message: "No effect!"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scene, mock := newCombatYellTestScene(t)

			scene.combatMapInputHandler(ebiten.KeyY)
			dialog := waitForCombatYellDialog(t, scene)
			if test.input != nil {
				scene.keyboard.SetForceWaitAnyKey(0)
				scene.keyboard.SimulateKeyPress(*test.input)
				dialog.Update()
				require.NotEmpty(t, dialog.GetText())
			}
			scene.keyboard.SetAllowKeyPressImmediately()
			scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
			dialog.Update()
			drainCombatYellWorkflow(t, scene)

			assert.Contains(t, mock.Messages, test.message)
			assert.Empty(t, mock.TimeAdvanced())
		})
	}
}

func newCombatYellTestScene(t *testing.T) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, mock := newCombatUseTestScene(t)
	scene.gameState.BeginCombatTurns(game_state.CombatTurnsPresented)

	return scene, mock
}

func waitForCombatYellDialog(t *testing.T, scene *GameScene) *widgets.InputBox {
	t.Helper()
	for range 10 {
		scene.gameState.ElapsedMs += DialogPollingRateMs
		scene.gameState.EventQueueManager.Update()
		if dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.InputBox); ok {
			return dialog
		}
	}
	t.Fatal("combat yell dialog was not shown")

	return nil
}

func drainCombatYellWorkflow(t *testing.T, scene *GameScene) {
	t.Helper()
	for range 10 {
		scene.gameState.ElapsedMs += KeyboardDebounceMs
		scene.gameState.EventQueueManager.Update()
		if !scene.gameState.EventQueueManager.HasWorkflow("combat_yell") {
			return
		}
	}
	t.Fatal("combat Yell workflow did not complete")
}

func keyPointer(key ebiten.Key) *ebiten.Key {
	return &key
}
