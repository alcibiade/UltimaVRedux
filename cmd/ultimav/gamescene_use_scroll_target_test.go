package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestResurrectionScrollTargetCancellationPreservesInventory(t *testing.T) {
	scene, _ := newSurfaceUseOwnershipScene(t)
	state := scene.gameState
	state.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, 1)

	scene.startResurrectionScrollWorkflow(nil)
	state.EventQueueManager.Update()
	dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.CharacterSelectionDialog)
	require.True(t, ok)
	dialog.HandleEscape()
	drainNamedSurfaceUseWorkflow(t, scene, "resurrection_scroll")

	assert.Equal(t, uint16(1), state.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp))
}

func TestResurrectionScrollConfirmedLivingTargetConsumesWithoutEffect(t *testing.T) {
	scene, _ := newSurfaceUseOwnershipScene(t)
	state := scene.gameState
	state.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, 1)
	target := state.PartyState.GetCharacter(0)
	require.True(t, target.IsGood())

	scene.startResurrectionScrollWorkflow(nil)
	state.EventQueueManager.Update()
	dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.CharacterSelectionDialog)
	require.True(t, ok)
	require.Equal(t, widgets.CharacterSelectionWarning, dialog.GetCurrentOption().State)
	dialog.HandleEnter()
	for range 20 {
		state.ElapsedMs += DialogPollingRateMs + KeyboardDebounceShortMs
		state.EventQueueManager.Update()
		if state.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp) == 0 {
			break
		}
	}

	assert.Zero(t, state.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp))
	assert.True(t, target.IsGood())
	require.NotPanics(t, state.EventQueueManager.Shutdown)
}

func TestResurrectionScrollShutdownRemovesTargetDialog(t *testing.T) {
	scene, _ := newSurfaceUseOwnershipScene(t)
	state := scene.gameState
	state.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, 1)

	scene.startResurrectionScrollWorkflow(nil)
	state.EventQueueManager.Update()
	require.IsType(t, &widgets.CharacterSelectionDialog{}, scene.dialogStack.PeekTopModalDialog())

	require.NotPanics(t, state.EventQueueManager.Shutdown)
	assert.Zero(t, scene.dialogStack.Count())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp))
}
