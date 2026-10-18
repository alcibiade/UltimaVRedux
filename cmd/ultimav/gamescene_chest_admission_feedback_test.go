package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/stretchr/testify/require"
)

func TestChestAdmissionClearsAllInformationalFeedbackPhases(t *testing.T) {
	for limit := range 4 {
		for _, age := range []int64{0, widgets.ActionFeedbackFadeInMs + 1, widgets.ActionFeedbackDisplayMs + 1} {
			t.Run(fmt.Sprintf("age-%d/cap-%d", age, limit), func(t *testing.T) {
				scene := newCombatReportTestScene(t)
				state := scene.gameState
				wireFeedbackOutputForTest(scene)
				setActionPopupLimitForTest(scene, limit)
				scene.addRowStr("No trap found.")
				scene.ShowSearchFeedback("No trap found.")
				scene.ShowActionFeedback("A wooden chest.", true)
				scene.ShowActionFeedback("Not unlocked.", true)
				state.ElapsedMs += age
				scene.updateTransientActionFeedback()
				cards := append([]*transientActionFeedback(nil), scene.transientActionFeedback...)
				require.Len(t, cards, limit)
				history := scene.output.JournalHistory()
				underlay := &countingDialog{}
				scene.dialogStack.PushModalDialog(underlay)
				token := state.FinishCombatActionDeferred()
				scene.startCombatChestRevealWorkflow(game_state.CombatOpenResult{
					Position:    references.Position{X: 5, Y: 10},
					RevealItems: []game_state.CombatLootReveal{{Sprite: indexes.ItemGem, Description: "a gem!"}},
				}, token)
				require.Len(t, scene.transientActionFeedback, limit, "starting a queued workflow is not admission")
				state.EventQueueManager.Update()
				require.IsType(t, &widgets.CombatLootDialog{}, scene.dialogStack.PeekTopModalDialog())
				require.Empty(t, scene.transientActionFeedback)
				for _, card := range cards {
					require.False(t, scene.dialogStack.ContainsWidgetInstance(card.dialog))
					require.False(t, card.dialog.IsVisible())
				}
				require.True(t, scene.dialogStack.ContainsWidgetInstance(underlay), "unrelated workflow-owned choices remain")
				require.Contains(t, scene.output.JournalHistory(), history)
				setActionPopupLimitForTest(scene, 3)
				scene.updateTransientActionFeedback()
				require.Empty(t, scene.transientActionFeedback, "raising the cap during contents cannot revive old cards")
				scene.ShowActionFeedback("later feedback", true)
				later := scene.transientActionFeedback[0]
				state.EventQueueManager.Shutdown()
				require.True(t, scene.dialogStack.ContainsWidgetInstance(underlay))
				require.True(t, scene.dialogStack.ContainsWidgetInstance(later.dialog), "finalization is not another broad purge")
				require.Equal(t, 1, strings.Count(scene.output.JournalHistory(), history))
				require.Equal(t, 1, strings.Count(scene.output.JournalHistory(), "later feedback"))
				require.Nil(t, scene.combatLootPreview)
				scene.clearTransientActionFeedback()
			})
		}
	}
}

func TestChestJimmyOffAllowsEmptyCaptureAndContentsAcknowledgement(t *testing.T) {
	for limit := range 4 {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			scene := newCombatReportTestScene(t)
			state := scene.gameState
			wireFeedbackOutputForTest(scene)
			setActionPopupLimitForTest(scene, limit)
			actor := state.CombatScene.GetActiveCombatantIndex()
			state.CombatScene.SetPartyMemberPosition(actor, references.Position{X: 5, Y: 9})
			state.PartyState.GetCharacter(actor).Dexterity = 99
			state.PartyState.GetCharacter(actor).Intelligence = 99
			state.PartyState.Inventory.Provisions.Keys.Set(2)
			position := references.Position{X: 5, Y: 10}
			setCombatConcealmentTerrain(state, position, indexes.BrickFloor)
			require.True(t, state.CombatScene.AddObject(game_state.NewCombatChest(position, 30, true)))
			require.True(t, state.ActionSearchCombatMap(references.Down))
			searchHistory := scene.output.JournalHistory()
			require.Equal(t, "Thou dost find a complex trap!", searchHistory)
			scene.combatMapJimmySecondary(references.Down)
			require.Len(t, scene.combatChestFeedback, min(limit, 1), "Off legitimately captures no handles")
			require.Equal(t, 1, strings.Count(scene.output.JournalHistory(), "Trap disarmed!"))
			require.Equal(t, 1, strings.Count(scene.output.JournalHistory(), searchHistory))
			require.EqualValues(t, 2, state.PartyState.Inventory.Provisions.Keys.Get())
			setActionPopupLimitForTest(scene, 0)
			scene.updateTransientActionFeedback()
			require.Empty(t, scene.combatChestFeedback)
			require.Zero(t, scene.dialogStack.Count())
			state.SetRandomSeed(624)
			require.True(t, scene.combatMapOpenSecondary(references.Down))
			state.EventQueueManager.Update()
			require.IsType(t, &widgets.CombatLootDialog{}, scene.dialogStack.PeekTopModalDialog())
			require.True(t, state.EventQueueManager.HasInputBlockingWorkflows())
			require.Empty(t, scene.transientActionFeedback)
			setActionPopupLimitForTest(scene, 3)
			finishChestFeedbackSummaryForTest(t, scene)
			require.Empty(t, scene.transientActionFeedback)
			require.Empty(t, scene.combatChestFeedback)
			require.Nil(t, scene.combatLootPreview)
			require.False(t, state.EventQueueManager.HasWorkflow("combat_chest_reveal_workflow"))
			require.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
			require.Equal(t, 1, strings.Count(scene.output.JournalHistory(), "Found:"))
			require.Equal(t, 1, strings.Count(scene.output.JournalHistory(), "Trap disarmed!"))
		})
	}
}
