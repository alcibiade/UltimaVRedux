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

func TestDungeonUsePotionCancellationPreservesInventoryTimeAndRNG(t *testing.T) {
	scene, _ := newDungeonViewTestScene(t)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.Potions.Set(references.Green, 1)
	expectedTime := state.DateTime

	selectOnlyDungeonUseItem(t, scene)
	require.Equal(t, uint16(1), state.PartyState.Inventory.Potions.Get(references.Green))
	require.False(t, state.IsAtDungeonCommandBoundary())
	dialog := waitForTopModal[*widgets.CharacterSelectionDialog](t, scene)
	rngCalls := state.GetRNG().Calls()
	dialog.HandleEscape()
	drainSceneWorkflow(t, scene, "potion_workflow", DialogPollingRateMs)

	assert.Equal(t, rngCalls, state.GetRNG().Calls())
	assert.True(t, state.PartyState.GetCharacter(0).IsGood())
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Equal(t, expectedTime, state.DateTime)
}

func TestDungeonUseImmediateScrollsConsumeAndApply(t *testing.T) {
	tests := []struct {
		name       string
		scroll     references.Scroll
		effectType game_state.DurationEffectType
	}{
		{name: "Protection", scroll: references.ScrollInSanct, effectType: game_state.DurationEffectProtection},
		{name: "Negate Magic", scroll: references.ScrollInAn, effectType: game_state.DurationEffectNegateMagic},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scene, _ := newDungeonViewTestScene(t)
			state := scene.gameState
			clearUsableItemsForDungeonView(state)
			state.PartyState.Inventory.Scrolls.Set(test.scroll, 1)
			expectedTime := state.DateTime
			expectedTime.Advance(1)

			selectOnlyDungeonUseItem(t, scene)

			assert.Zero(t, state.PartyState.Inventory.Scrolls.Get(test.scroll))
			assert.Equal(t, test.effectType, state.ActiveDurationEffect.EffectType)
			assert.Greater(t, state.ActiveDurationEffect.TurnsRemaining, 0)
			assert.True(t, state.IsAtDungeonCommandBoundary())
			assert.Equal(t, expectedTime, state.DateTime)
		})
	}
}

func TestDungeonUseLightAndTimeStopCompleteAfterAsyncEffect(t *testing.T) {
	tests := []struct {
		name     string
		scroll   references.Scroll
		workflow string
		assert   func(*testing.T, *game_state.GameState)
	}{
		{
			name: "Light", scroll: references.ScrollVasLor, workflow: "light_scroll",
			assert: func(t *testing.T, state *game_state.GameState) {
				assert.Greater(t, state.MagicLightMinutesRemaining, 0)
			},
		},
		{
			name: "Time Stop", scroll: references.ScrollAnTym, workflow: "timestop_scroll",
			assert: func(t *testing.T, state *game_state.GameState) {
				assert.Equal(t, game_state.DurationEffectTimeStop, state.ActiveDurationEffect.EffectType)
				assert.Greater(t, state.ActiveDurationEffect.TurnsRemaining, 0)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scene, _ := newDungeonViewTestScene(t)
			state := scene.gameState
			clearUsableItemsForDungeonView(state)
			state.PartyState.Inventory.Scrolls.Set(test.scroll, 1)
			expectedTime := state.DateTime
			expectedTime.Advance(1)

			selectOnlyDungeonUseItem(t, scene)
			assert.Zero(t, state.PartyState.Inventory.Scrolls.Get(test.scroll))
			assert.False(t, state.IsAtDungeonCommandBoundary())
			drainDungeonEffectWorkflow(t, scene, test.workflow)

			test.assert(t, state)
			assert.True(t, state.IsAtDungeonCommandBoundary())
			assert.Equal(t, expectedTime, state.DateTime)
		})
	}
}

func TestDungeonUseRejectedDoomTimeStopRetainsTokenUntilFeedbackExpires(t *testing.T) {
	scene, callbacks := newDungeonViewTestSceneAt(t, references.Doom)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.Scrolls.Set(references.ScrollAnTym, 1)
	state.ActiveDurationEffect = game_state.DurationEffect{
		EffectType: game_state.DurationEffectProtection, TurnsRemaining: 3,
	}
	expectedTime := state.DateTime
	expectedTime.Advance(1)
	callbacks.Reset()

	selectOnlyDungeonUseItem(t, scene)
	overlay := waitForTimedOverlay(t, scene)

	require.Zero(t, state.PartyState.Inventory.Scrolls.Get(references.ScrollAnTym))
	assert.Equal(t, "No effect!", overlay.GetMessage())
	assert.Equal(t, []game_state.SoundEffect{game_state.SoundScrollInvoke, game_state.SoundSpellFailure}, callbacks.SoundEffectsPlayed)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	assert.True(t, callbacks.InputBlocked)
	assert.Equal(t, 3, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, expectedTime, state.DateTime)

	remainingMs := scene.timedOverlayDismissMs - state.ElapsedMs
	require.Greater(t, remainingMs, int64(1))
	state.ElapsedMs += remainingMs - 1
	state.EventQueueManager.Update()
	scene.updateTimedOverlay()
	assert.Same(t, overlay, scene.timedOverlayDialog)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	assert.True(t, callbacks.InputBlocked)

	state.ElapsedMs++
	for range 30 {
		state.EventQueueManager.Update()
		scene.updateTimedOverlay()
		if state.IsAtDungeonCommandBoundary() && !callbacks.InputBlocked {
			break
		}
		state.ElapsedMs += DialogPollingRateMs
	}

	require.True(t, state.IsAtDungeonCommandBoundary())
	assert.False(t, callbacks.InputBlocked)
	assert.Nil(t, scene.timedOverlayDialog)
	assert.Zero(t, scene.dialogStack.Count())
	assert.Equal(t, 2, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	assert.Equal(t, callbacks.BlockInputCalls, callbacks.UnblockInputCalls)

	completedTime := state.DateTime
	for range 3 {
		state.EventQueueManager.Update()
	}
	assert.Equal(t, completedTime, state.DateTime, "feedback completion must consume the dungeon token once")

	expectedTime.Advance(1)
	scene.dungeonMapInputHandler(ebiten.KeyP)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Equal(t, expectedTime, state.DateTime)
	assert.Equal(t, []int{1, 1}, callbacks.TimeAdvanced())
}

func TestDungeonUseLightShutdownRestoresInputAndCompletesTokenOnce(t *testing.T) {
	scene, callbacks := newDungeonViewTestScene(t)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.Scrolls.Set(references.ScrollVasLor, 1)
	state.ActiveDurationEffect = game_state.DurationEffect{
		EffectType: game_state.DurationEffectProtection, TurnsRemaining: 3,
	}
	expectedTime := state.DateTime
	expectedTime.Advance(1)
	callbacks.Reset()

	selectOnlyDungeonUseItem(t, scene)
	require.True(t, state.EventQueueManager.HasWorkflow("light_scroll"))
	require.False(t, state.IsAtDungeonCommandBoundary())
	require.True(t, callbacks.InputBlocked)

	require.NotPanics(t, state.EventQueueManager.Shutdown)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Greater(t, state.MagicLightMinutesRemaining, 0)
	assert.False(t, callbacks.InputBlocked)
	assert.Equal(t, expectedTime, state.DateTime)
	assert.Equal(t, 2, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	assert.Equal(t, callbacks.BlockInputCalls, callbacks.UnblockInputCalls)
	assert.Zero(t, scene.dialogStack.Count())

	require.NotPanics(t, state.EventQueueManager.Shutdown)
	assert.Equal(t, expectedTime, state.DateTime)
	assert.Equal(t, 2, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
}

func TestDungeonUseTimeStopShutdownAppliesConsumedEffect(t *testing.T) {
	scene, callbacks := newDungeonViewTestScene(t)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.Scrolls.Set(references.ScrollAnTym, 1)
	expectedTime := state.DateTime
	expectedTime.Advance(1)
	callbacks.Reset()

	selectOnlyDungeonUseItem(t, scene)
	require.True(t, state.EventQueueManager.HasWorkflow("timestop_scroll"))
	require.False(t, state.IsAtDungeonCommandBoundary())

	require.NotPanics(t, state.EventQueueManager.Shutdown)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Equal(t, game_state.DurationEffectTimeStop, state.ActiveDurationEffect.EffectType)
	assert.Greater(t, state.ActiveDurationEffect.TurnsRemaining, 0)
	assert.Equal(t, expectedTime, state.DateTime)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	assert.False(t, callbacks.InputBlocked)
}

func TestDungeonUseResurrectionCancellationPreservesInventoryAndTime(t *testing.T) {
	scene, _ := newDungeonViewTestScene(t)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.Scrolls.Set(references.ScrollInManiCorp, 1)
	expectedTime := state.DateTime

	selectOnlyDungeonUseItem(t, scene)
	require.Equal(t, uint16(1), state.PartyState.Inventory.Scrolls.Get(references.ScrollInManiCorp))
	dialog := waitForTopModal[*widgets.CharacterSelectionDialog](t, scene)
	dialog.HandleEscape()
	drainSceneWorkflow(t, scene, "resurrection_scroll", DialogPollingRateMs)

	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Equal(t, expectedTime, state.DateTime)
}

func TestDungeonUsePersistentItemsRouteThroughProductionInput(t *testing.T) {
	tests := []struct {
		name   string
		item   references.QuestItem
		active func(*game_state.GameState) bool
	}{
		{name: "Amulet", item: references.Amulet, active: (*game_state.GameState).IsAmuletActive},
		{name: "Crown", item: references.Crown, active: (*game_state.GameState).IsCrownActive},
		{name: "Black Badge", item: references.BlackBadge, active: (*game_state.GameState).IsBadgeActive},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scene, _ := newDungeonViewTestScene(t)
			state := scene.gameState
			clearUsableItemsForDungeonView(state)
			state.PartyState.Inventory.QuestItems.Set(test.item, 1)
			expectedTime := state.DateTime
			expectedTime.Advance(1)

			selectOnlyDungeonUseItem(t, scene)

			assert.True(t, test.active(state))
			assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(test.item))
			assert.True(t, state.IsAtDungeonCommandBoundary())
			assert.Equal(t, expectedTime, state.DateTime)
		})
	}
}

func TestDungeonUsePocketWatchCompletesAfterDismissal(t *testing.T) {
	scene, callbacks := newDungeonViewTestScene(t)
	state := scene.gameState
	clearUsableItemsForDungeonView(state)
	state.PartyState.Inventory.QuestItems.Set(references.PocketWatch, 1)
	state.SystemCallbacks.Flow.Observation.ShowPocketWatchTime = scene.ShowPocketWatchTime
	expectedTime := state.DateTime
	expectedTime.Advance(1)
	callbacks.Reset()

	selectOnlyDungeonUseItem(t, scene)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	assert.True(t, callbacks.InputBlocked)
	drainDungeonEffectWorkflow(t, scene, "pocket_watch")

	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.False(t, callbacks.InputBlocked)
	assert.Equal(t, expectedTime, state.DateTime)
	assert.Equal(t, callbacks.BlockInputCalls, callbacks.UnblockInputCalls)
	assert.Zero(t, scene.dialogStack.Count())

	expectedTime.Advance(1)
	scene.dungeonMapInputHandler(ebiten.KeyP)
	assert.True(t, state.IsAtDungeonCommandBoundary())
	assert.Equal(t, expectedTime, state.DateTime)
}

func TestDungeonUseSourceRejectionsAndConsumption(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		grant        func(*game_state.GameState)
		quantity     func(*game_state.GameState) uint16
		wantQuantity uint16
	}{
		{name: "Carpet", message: "Not here!", grant: grantDungeonQuestItem(references.Carpet), quantity: dungeonQuestItemQuantity(references.Carpet), wantQuantity: 1},
		{name: "Skull Key", message: "Not here!", grant: func(state *game_state.GameState) { state.PartyState.Inventory.Provisions.SkullKeys.Set(1) }, quantity: func(state *game_state.GameState) uint16 { return state.PartyState.Inventory.Provisions.SkullKeys.Get() }, wantQuantity: 1},
		{name: "Moonstone", message: "Moonstone cannot be buried here!", grant: func(state *game_state.GameState) { state.PartyState.Inventory.Moonstones.Set(references.NewMoon, 1) }, quantity: func(state *game_state.GameState) uint16 {
			return state.PartyState.Inventory.Moonstones.Get(references.NewMoon)
		}, wantQuantity: 1},
		{name: "Shard", message: "No effect!", grant: grantDungeonQuestItem(references.ShardOfFalsehood), quantity: dungeonQuestItemQuantity(references.ShardOfFalsehood), wantQuantity: 1},
		{name: "Spyglass", message: "Not here!", grant: grantDungeonQuestItem(references.Spyglass), quantity: dungeonQuestItemQuantity(references.Spyglass), wantQuantity: 1},
		{name: "Sextant", message: "Only outdoors!", grant: grantDungeonQuestItem(references.Sextant), quantity: dungeonQuestItemQuantity(references.Sextant), wantQuantity: 1},
		{name: "Wooden Box", message: "How?", grant: grantDungeonQuestItem(references.WoodenBox), quantity: dungeonQuestItemQuantity(references.WoodenBox), wantQuantity: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scene, callbacks := newDungeonViewTestScene(t)
			state := scene.gameState
			clearUsableItemsForDungeonView(state)
			test.grant(state)
			expectedTime := state.DateTime
			callbacks.Reset()

			scene.dungeonMapInputHandler(ebiten.KeyU)
			dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
			calls := state.GetRNG().Calls()
			dialog.HandleEnter()
			require.False(t, dialog.IsDismissed())
			require.Equal(t, calls, state.GetRNG().Calls())
			assert.Equal(t, test.wantQuantity, test.quantity(state))
			assert.Empty(t, callbacks.Messages, "unavailable entries never dispatch their old effect handler")
			dialog.Dismiss()
			drainSceneWorkflow(t, scene, "dungeon_use_item_workflow", DialogPollingRateMs)
			assert.True(t, state.IsAtDungeonCommandBoundary())
			assert.Equal(t, expectedTime, state.DateTime)
		})
	}
}

func selectOnlyDungeonUseItem(t *testing.T, scene *GameScene) {
	t.Helper()
	scene.dungeonMapInputHandler(ebiten.KeyU)
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	dialog.HandleEnter()
	drainSceneWorkflow(t, scene, "dungeon_use_item_workflow", DialogPollingRateMs)
}

func drainDungeonEffectWorkflow(t *testing.T, scene *GameScene, name string) {
	t.Helper()
	for range 100 {
		if dialog, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.OverlayDialog); ok {
			dialog.Dismiss()
		}
		scene.gameState.ElapsedMs += 100
		scene.gameState.EventQueueManager.Update()
		if !scene.gameState.EventQueueManager.HasWorkflow(name) {
			return
		}
	}
	t.Fatalf("dungeon effect workflow %q did not complete", name)
}

func grantDungeonQuestItem(item references.QuestItem) func(*game_state.GameState) {
	return func(state *game_state.GameState) {
		state.PartyState.Inventory.QuestItems.Set(item, 1)
	}
}

func dungeonQuestItemQuantity(item references.QuestItem) func(*game_state.GameState) uint16 {
	return func(state *game_state.GameState) uint16 {
		return state.PartyState.Inventory.QuestItems.Get(item)
	}
}
