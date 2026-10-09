package main

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestCombatSummaryAutomaticDepartureAndReviewDoNotConsumeTurns(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	index := state.CombatScene.GetActiveCombatantIndex()
	position := state.CombatScene.GetPartyMemberPosition(index)
	target := *references.Down.GetNewPositionInDirection(&position)
	state.CombatScene.Victory = true
	require.True(t, state.CombatScene.AddObject(game_state.NewCombatLooseItem(target, references.Gold, 38, indexes.ItemMoney)))
	state.GetCombatItem(references.Down)
	// Eviction of the pickup text must have no effect on authoritative totals.
	for range combatEventHistoryCapacity + 10 {
		scene.combatHUD.addEvent(game_state.CombatEvent{Kind: game_state.CombatEventBattleBegun})
	}
	assert.False(t, scene.showPendingBattleSummary(), "victory and looting do not end the encounter")
	state.EndCombat(game_state.CombatVictory)
	require.Len(t, scene.pendingBattleSummaries, 1)
	require.True(t, scene.showPendingBattleSummary())
	require.NotNil(t, scene.battleSummaryDialog)
	require.Len(t, scene.lastBattleSummary.Collected, 1)
	assert.Equal(t, uint64(38), scene.lastBattleSummary.Collected[0].Quantity)
	assert.Empty(t, scene.pendingBattleSummaries)
	assert.False(t, scene.showPendingBattleSummary())
	date, calls := state.DateTime, state.GetRNG().Calls()
	scene.battleSummaryDialog.Update() // release the opening transition
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	scene.battleSummaryDialog.Update()
	assert.Nil(t, scene.battleSummaryDialog)
	assert.False(t, scene.dialogStack.HasModalDialog())
	require.True(t, scene.handleBattleSummaryShortcut(ebiten.KeyF4))
	require.NotNil(t, scene.battleSummaryDialog)
	scene.battleSummaryDialog.Update()
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	scene.battleSummaryDialog.Update()
	assert.Nil(t, scene.battleSummaryDialog)
	assert.Equal(t, date, state.DateTime)
	assert.Equal(t, calls, state.GetRNG().Calls())
}

func TestCombatSummaryWaitsForOwnerAndPreservesOtherDialogs(t *testing.T) {
	scene := newCombatReportTestScene(t)
	conclusion := game_state.CombatConclusion{Reason: game_state.CombatVictory, Collected: []game_state.CombatCollectedItem{
		{Name: "Gold", Sprite: indexes.ItemMoney, Quantity: 38},
	}}
	scene.recordCombatConclusion(conclusion)
	conclusion.Collected[0].Quantity = 0
	scene.gameState.EventQueueManager.Workflow("summary_owner").Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().First(func() {}).Wait(500).Start()
	assert.False(t, scene.showPendingBattleSummary())
	scene.gameState.EventQueueManager.Shutdown()
	other := &countingDialog{}
	scene.dialogStack.PushModalDialog(other)
	assert.False(t, scene.showPendingBattleSummary())
	scene.dialogStack.MustRemoveWidgetInstance(other)
	require.True(t, scene.showPendingBattleSummary())
	assert.Equal(t, uint64(38), scene.lastBattleSummary.Collected[0].Quantity)
	scene.dialogStack.PushModalDialog(other)
	scene.clearBattleSummaries()
	assert.Same(t, other, scene.dialogStack.PeekTopModalDialog())
	assert.Nil(t, scene.lastBattleSummary)
	scene.clearBattleSummaries()
	assert.Equal(t, 1, scene.dialogStack.Count())
}

func TestCombatSummarySpecialConclusionAndHeldExitKey(t *testing.T) {
	scene := newCombatReportTestScene(t)
	scene.recordCombatConclusion(game_state.CombatConclusion{
		Reason: game_state.CombatVictory, Special: game_state.CombatSpecialConclusionDoomMirror,
	})
	assert.Empty(t, scene.pendingBattleSummaries)
	assert.NotNil(t, scene.combatConclusion)
	pressed := true
	scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return key == ebiten.KeyEscape && pressed })
	scene.recordCombatConclusion(game_state.CombatConclusion{Reason: game_state.CombatVictory})
	require.True(t, scene.showPendingBattleSummary())
	dialog := scene.battleSummaryDialog
	dialog.Update()
	assert.False(t, dialog.IsDismissed())
	pressed = false
	scene.keyboard.ClearFrameConsumption()
	dialog.Update()
	pressed = true
	dialog.Update()
	assert.True(t, dialog.IsDismissed())
	assert.Nil(t, scene.battleSummaryDialog)
}

func TestCombatSummaryProductionUpdateDefersUntilExitWorkflowFinishes(t *testing.T) {
	scene, state, _ := newSmashedInputTestScene(t, 0, 42)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	state.SystemCallbacks.Combat = game_state.NewCombatCallbacks(scene.recordCombatEvent, scene.recordCombatConclusion)
	require.NoError(t, state.TransitionToCombatMap(0, references.BritanniaCombatTerritory, game_state.CombatEntryNorth))
	now := time.Unix(100, 0)
	scene.startTime, scene.now = now, func() time.Time { return now }
	state.EventQueueManager.Workflow("test_departure").Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().First(func() { state.EndCombat(game_state.CombatVictory) }).Wait(500).Start()
	require.NoError(t, scene.Update(nil))
	assert.Nil(t, state.CombatScene)
	assert.Nil(t, scene.battleSummaryDialog)
	now = now.Add(time.Second)
	require.NoError(t, scene.Update(nil))
	require.NotNil(t, scene.battleSummaryDialog)
	date, calls := state.DateTime, state.GetRNG().Calls()
	require.NoError(t, scene.Update(nil)) // release gate
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, scene.Update(nil))
	assert.Nil(t, scene.battleSummaryDialog)
	require.NoError(t, scene.Update(nil))
	scene.keyboard.SimulateKeyPress(ebiten.KeyF4)
	require.NoError(t, scene.Update(nil))
	require.NotNil(t, scene.battleSummaryDialog)
	assert.Equal(t, date, state.DateTime)
	assert.Equal(t, calls, state.GetRNG().Calls())
}

func TestCombatSummaryWaitsForRecovery(t *testing.T) {
	scene := newCombatReportTestScene(t)
	state := scene.gameState
	state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		if character.IsInParty() {
			character.SetHp(0)
			character.MarkAsDead()
		}
	})
	scene.recordCombatConclusion(game_state.CombatConclusion{Reason: game_state.CombatDefeat})
	assert.False(t, scene.showPendingBattleSummary())
	assert.Len(t, scene.pendingBattleSummaries, 1)
	character := state.PartyState.GetCharacter(0)
	character.SetHp(30)
	character.SetStatus(party_state.Good)
	require.True(t, scene.showPendingBattleSummary())
	assert.Equal(t, game_state.CombatDefeat, scene.lastBattleSummary.Reason)
	scene.PrepareForSceneReplacement()
	assert.Nil(t, scene.lastBattleSummary)
	assert.Nil(t, scene.battleSummaryDialog)
	assert.Empty(t, scene.pendingBattleSummaries)
}
