package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestCombatArtifactUseTogglesThroughProductionRouteExactlyOnce(t *testing.T) {
	for _, item := range []references.QuestItem{references.Crown, references.Amulet, references.BlackBadge} {
		for _, initiallyActive := range []bool{false, true} {
			scene, callbacks := newStartedCombatPushScene(t)
			expected, expectedCallbacks := newStartedCombatPushScene(t)
			for _, candidate := range []*GameScene{scene, expected} {
				clearCombatUseTestInventory(candidate.gameState)
				clearCombatUseQuestItems(candidate.gameState)
				candidate.gameState.PartyState.Inventory.QuestItems.Set(item, 1)
				if initiallyActive {
					candidate.gameState.UseItem(candidate.gameState.GetUsableItemsForContext(game_state.UsageCombat)[0])
				}
			}
			callbacks.Reset()
			expectedCallbacks.Reset()
			sequence := scene.gameState.CastUseCommitSequence()
			scene.combatMapInputHandler(ebiten.KeyU)
			dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
			dialog.HandleEnter()
			dialog.HandleEnter()
			drainDismissingOverlays(t, scene, "use_item_workflow")
			expected.gameState.UseItem(expected.gameState.GetUsableItemsForContext(game_state.UsageCombat)[0])
			expected.gameState.FinishCombatAction()
			require.Equal(t, sequence+1, scene.gameState.CastUseCommitSequence())
			require.EqualValues(t, 1, scene.gameState.PartyState.Inventory.QuestItems.Get(item))
			require.Equal(t, expected.gameState.CrownState, scene.gameState.CrownState)
			require.Equal(t, expected.gameState.AmuletState, scene.gameState.AmuletState)
			require.Equal(t, expected.gameState.BadgeState, scene.gameState.BadgeState)
			// Workflow clock advancement is presentation only.
			expected.gameState.ElapsedMs = scene.gameState.ElapsedMs
			assertProductionCombatPushCompletion(t, scene, expected.gameState, callbacks, expectedCallbacks)
		}
	}
}

func TestCombatUnavailableOnlyInventoryRemainsInspectableAndFree(t *testing.T) {
	for _, item := range []references.Scroll{references.ScrollInManiCorp, references.ScrollRelHur, references.ScrollInQuasWis} {
		scene, _ := newStartedCombatPushScene(t)
		clearCombatUseTestInventory(scene.gameState)
		clearCombatUseQuestItems(scene.gameState)
		scene.gameState.PartyState.Inventory.Scrolls.Set(item, 2)
		scene.combatMapInputHandler(ebiten.KeyU)
		dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
		before := snapshotBlockedCombatState(t, scene.gameState)
		dialog.HandleEnter()
		require.False(t, dialog.IsDismissed())
		require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
		dialog.Dismiss()
		drainDismissingOverlays(t, scene, "use_item_workflow")
		before.ElapsedMs = scene.gameState.ElapsedMs
		require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	}
}

func TestCombatArtifactUseRejectsChangedActorBeforeCommit(t *testing.T) {
	scene, _ := newStartedCombatPushScene(t)
	clearCombatUseTestInventory(scene.gameState)
	clearCombatUseQuestItems(scene.gameState)
	scene.gameState.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
	scene.StartCombatUseWorkflow()
	dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
	scene.gameState.FinishCombatAction()
	before := snapshotBlockedCombatState(t, scene.gameState)
	dialog.HandleEnter()
	drainDismissingOverlays(t, scene, "use_item_workflow")
	before.ElapsedMs = scene.gameState.ElapsedMs
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
	require.False(t, scene.gameState.IsCrownActive())
}

func TestCombatCrownUseReplacesDurationEffectsAndPreservesIndependentArtifacts(t *testing.T) {
	for _, activate := range []struct {
		name  string
		apply func(*game_state.GameState, int) bool
	}{
		{"protection", (*game_state.GameState).ActivateProtectionEffect},
		{"quickness", (*game_state.GameState).ActivateQuicknessEffect},
		{"confusion", (*game_state.GameState).ActivateConfusionEffect},
		{"negate magic", (*game_state.GameState).ActivateNegateMagicEffect},
		{"time stop", (*game_state.GameState).ActivateTimeStopEffect},
	} {
		t.Run(activate.name, func(t *testing.T) {
			scene, _ := newStartedCombatPushScene(t)
			state := scene.gameState
			clearCombatUseTestInventory(state)
			clearCombatUseQuestItems(state)
			state.PartyState.Inventory.QuestItems.Set(references.Crown, 1)
			require.True(t, state.ToggleAmulet())
			require.True(t, state.ToggleBadge())
			require.True(t, activate.apply(state, 100))
			amulet, badge := state.AmuletState, state.BadgeState
			scene.combatMapInputHandler(ebiten.KeyU)
			waitForTopModal[*widgets.ItemSelectionDialog](t, scene).HandleEnter()
			drainDismissingOverlays(t, scene, "use_item_workflow")
			require.True(t, state.IsCrownActive())
			require.Equal(t, game_state.DurationEffect{}, state.ActiveDurationEffect)
			require.Equal(t, amulet, state.AmuletState)
			require.Equal(t, badge, state.BadgeState)
			require.EqualValues(t, 1, state.PartyState.Inventory.QuestItems.Get(references.Crown))
		})
	}
}

func TestCombatIndependentArtifactUsePreservesCrownAndDuration(t *testing.T) {
	for _, item := range []references.QuestItem{references.Amulet, references.BlackBadge} {
		for _, crown := range []bool{false, true} {
			scene, _ := newStartedCombatPushScene(t)
			state := scene.gameState
			clearCombatUseTestInventory(state)
			clearCombatUseQuestItems(state)
			state.PartyState.Inventory.QuestItems.Set(item, 1)
			if crown {
				require.True(t, state.ToggleCrown())
			} else {
				require.True(t, state.ActivateProtectionEffect(100))
			}
			scene.combatMapInputHandler(ebiten.KeyU)
			waitForTopModal[*widgets.ItemSelectionDialog](t, scene).HandleEnter()
			drainDismissingOverlays(t, scene, "use_item_workflow")
			require.Equal(t, crown, state.IsCrownActive())
			require.Equal(t, !crown, state.HasActiveDurationEffect(game_state.DurationEffectProtection))
			if item == references.Amulet {
				require.True(t, state.IsAmuletActive())
				require.False(t, state.IsBadgeActive())
			} else {
				require.True(t, state.IsBadgeActive())
				require.False(t, state.IsAmuletActive())
			}
		}
	}
}

func TestCombatDurationScrollUseReplacesCrownWithoutRemovingAmuletOrBadge(t *testing.T) {
	scene, _ := newStartedCombatPushScene(t)
	state := scene.gameState
	clearCombatUseTestInventory(state)
	clearCombatUseQuestItems(state)
	state.PartyState.Inventory.Scrolls.Set(references.ScrollInSanct, 1)
	require.True(t, state.ToggleCrown())
	require.True(t, state.ToggleAmulet())
	require.True(t, state.ToggleBadge())
	scene.combatMapInputHandler(ebiten.KeyU)
	waitForTopModal[*widgets.ItemSelectionDialog](t, scene).HandleEnter()
	drainDismissingOverlays(t, scene, "use_item_workflow")
	require.False(t, state.IsCrownActive())
	require.True(t, state.HasActiveDurationEffect(game_state.DurationEffectProtection))
	require.True(t, state.IsAmuletActive())
	require.True(t, state.IsBadgeActive())
	require.Zero(t, state.PartyState.Inventory.Scrolls.Get(references.ScrollInSanct))
}
