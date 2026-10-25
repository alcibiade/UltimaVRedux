package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestHMSCapeWorkflowCancellationDoesNotMutate(t *testing.T) {
	scene, callbacks := newSurfaceUseOwnershipScene(t)
	state := scene.gameState
	state.PartyState.Inventory.QuestItems.Set(references.HMSCape, 1)
	state.SetRandomSeed(462)
	rngCalls := state.GetRNG().Calls()
	completed := 0
	commit, review := state.BeginHMSCapeUse()
	require.NotNil(t, commit)
	require.Equal(t, game_state.HMSCapeUseReady, review.Result)
	callbacks.Reset()

	scene.startHMSCapeWorkflow(commit, func() { completed++ })
	dialog := waitForTopModal[*widgets.InputBox](t, scene)

	assert.NotContains(t, dialog.Question, "global setting")
	assert.NotContains(t, dialog.Question, "makes them faster")
	assert.Contains(t, dialog.Question, "Put these plans to use? (Y/N)")
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	dialog.Update()
	drainNamedSurfaceUseWorkflow(t, scene, hmsCapeWorkflowName)

	assert.False(t, state.IsHMSCapeActivated())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.HMSCape))
	assert.Equal(t, rngCalls, state.GetRNG().Calls())
	assert.Empty(t, callbacks.SoundEffectsPlayed)
	assert.Equal(t, 1, completed)
	assert.Zero(t, scene.dialogStack.Count())
}

func TestHMSCapeWorkflowConfirmsEnableAndDisable(t *testing.T) {
	scene, callbacks := newSurfaceUseOwnershipScene(t)
	state := scene.gameState
	state.PartyState.Inventory.QuestItems.Set(references.HMSCape, 1)
	callbacks.Reset()

	commit, _ := state.BeginHMSCapeUse()
	require.NotNil(t, commit)
	scene.startHMSCapeWorkflow(commit, func() {})
	dialog := waitForTopModal[*widgets.InputBox](t, scene)
	confirmHMSCapeDialog(t, scene, dialog)
	drainNamedSurfaceUseWorkflow(t, scene, hmsCapeWorkflowName)

	assert.True(t, state.IsHMSCapeActivated())
	assert.Contains(t, callbacks.Messages, "You put the H.M.S. Cape plans to use.")
	assert.Equal(t, []game_state.SoundEffect{game_state.SoundSummon}, callbacks.SoundEffectsPlayed)

	callbacks.Reset()
	commit, _ = state.BeginHMSCapeUse()
	require.NotNil(t, commit)
	scene.startHMSCapeWorkflow(commit, func() {})
	dialog = waitForTopModal[*widgets.InputBox](t, scene)
	assert.Contains(t, dialog.Question, "Put these plans aside? (Y/N)")
	confirmHMSCapeDialog(t, scene, dialog)
	drainNamedSurfaceUseWorkflow(t, scene, hmsCapeWorkflowName)

	assert.False(t, state.IsHMSCapeActivated())
	assert.Contains(t, callbacks.Messages, "You put the H.M.S. Cape plans aside.")
	assert.Equal(t, []game_state.SoundEffect{game_state.SoundSummon}, callbacks.SoundEffectsPlayed)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.HMSCape))
}

func confirmHMSCapeDialog(t *testing.T, scene *GameScene, dialog *widgets.InputBox) {
	t.Helper()
	scene.keyboard.ClearFrameConsumption()
	scene.keyboard.SimulateKeyPress(ebiten.KeyY)
	dialog.Update()
	require.Equal(t, "Y", dialog.GetText())
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
}
