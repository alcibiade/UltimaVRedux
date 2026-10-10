package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestCombatFieldOverlayRespectsPileWinnerAndTemporaryReveal(t *testing.T) {
	scene, screen := newCombatConcealmentTestScene(t)
	state := scene.gameState
	position := references.Position{X: 5, Y: 5}
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	require.True(t, state.CombatScene.AddField(game_state.NewCombatField(game_state.CombatFieldFire, position)))
	scene.drawCombatFieldOverlay(screen)
	assert.Positive(t, opaquePixelCount(screen))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), scene.combatLootTile(position, indexes.DeadBody))
	// Even a field that normally wins must not cover the temporary discovery.
	scene.combatLootPreview = &combatLootProjection{position: position, sprite: indexes.ItemPotion}
	screen.Clear()
	scene.drawCombatFieldOverlay(screen)
	assert.Zero(t, opaquePixelCount(screen))
	assert.Equal(t, indexes.SpriteIndex(indexes.ItemPotion), scene.combatLootTile(position, indexes.BrickFloor))
	scene.combatLootPreview = nil
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatLooseItem(position, references.Gem, 1, indexes.ItemGem)))
	for _, elapsed := range []int64{0, 999, 1000, 2000, 60000} {
		state.ElapsedMs = elapsed
		screen.Clear()
		scene.drawCombatFieldOverlay(screen)
		assert.Zero(t, opaquePixelCount(screen), "field cannot overwrite ordinary loot")
		assert.Equal(t, indexes.SpriteIndex(indexes.ItemGem), scene.combatLootTile(position, indexes.FireField))
	}
	assert.Equal(t, indexes.SpriteIndex(indexes.ItemGem), scene.combatLootTile(position, indexes.FireField))
}

func TestCombatChestRevealReturnsToStablePileWithoutOpeningNestedChest(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	actor := state.CombatScene.GetActiveCombatantIndex()
	state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
	position := references.Position{X: 5, Y: 10}
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 30, false)))
	state.SetRandomSeed(624)
	result := state.ActionOpenCombatMap(references.Down)
	require.True(t, result.Handled)
	require.Greater(t, len(result.RevealItems), 1)
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 1, true)))
	inventory, calls, objectCount := state.PartyState.Inventory.LockerSnapshot(), state.GetRNG().Calls(), state.CombatScene.ObjectCount()
	token := state.FinishCombatTurnDeferred()
	scene.startCombatChestRevealWorkflow(result, token)
	state.EventQueueManager.Update()
	for index, entry := range result.RevealItems {
		state.ElapsedMs += combatChestRevealDelayMs
		state.EventQueueManager.Update()
		require.NotNil(t, scene.combatLootPreview)
		assert.Equal(t, entry.Sprite, scene.combatLootTile(position, indexes.Chest))
		assert.Equal(t, game_state.CombatEventLootRevealed, scene.combatHUD.history[index+1].Kind)
		assert.Equal(t, entry, scene.combatHUD.history[index+1].LootReveal)
	}
	state.EventQueueManager.Update()
	dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.CombatLootDialog)
	require.True(t, ok)
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.IsDismissed())
	state.ElapsedMs += DialogPollingRateMs
	state.EventQueueManager.Update()
	assert.Nil(t, scene.combatLootPreview)
	assert.Zero(t, scene.dialogStack.Count())
	assert.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Equal(t, objectCount, state.CombatScene.ObjectCount(), "preview cannot open the nested chest")
	for _, elapsed := range []int64{1000, 2000, 60000} {
		state.ElapsedMs = elapsed
		assert.Equal(t, indexes.SpriteIndex(indexes.Chest), scene.combatLootTile(position, indexes.ItemGem))
	}
	state.CombatScene.SetPartyMemberPosition(state.CombatScene.GetActiveCombatantIndex(), references.Position{X: 5, Y: 9})
	assert.Equal(t, "Open it first!", state.GetCombatItem(references.Down).Message)
}
