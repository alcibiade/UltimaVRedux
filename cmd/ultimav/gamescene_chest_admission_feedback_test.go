package main

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/stretchr/testify/require"
)

func TestChestAdmissionClearsAllInformationalFeedbackPhases(t *testing.T) {
	for _, age := range []int64{0, widgets.ActionFeedbackFadeInMs + 1, widgets.ActionFeedbackDisplayMs + 1} {
		t.Run(fmt.Sprint(age), func(t *testing.T) {
			scene := newCombatReportTestScene(t)
			state := scene.gameState
			_, scene.output = newMainOutput(1280)
			scene.ShowSearchFeedback("No trap found.")
			scene.ShowActionFeedback("A wooden chest.", true)
			scene.ShowActionFeedback("Not unlocked.", true)
			state.ElapsedMs += age
			scene.updateTransientActionFeedback()
			cards := append([]*transientActionFeedback(nil), scene.transientActionFeedback...)
			require.NotEmpty(t, cards)
			history := scene.output.GetOutputStr(false)
			underlay := &countingDialog{}
			scene.dialogStack.PushModalDialog(underlay)
			token := state.FinishCombatActionDeferred()
			scene.startCombatChestRevealWorkflow(game_state.CombatOpenResult{
				Position:    references.Position{X: 5, Y: 10},
				RevealItems: []game_state.CombatLootReveal{{Sprite: indexes.ItemGem, Description: "a gem!"}},
			}, token)
			require.NotEmpty(t, scene.transientActionFeedback, "starting a queued workflow is not admission")
			state.EventQueueManager.Update()
			require.IsType(t, &widgets.CombatLootDialog{}, scene.dialogStack.PeekTopModalDialog())
			require.Empty(t, scene.transientActionFeedback)
			for _, card := range cards {
				require.False(t, scene.dialogStack.ContainsWidgetInstance(card.dialog))
			}
			require.True(t, scene.dialogStack.ContainsWidgetInstance(underlay), "unrelated workflow-owned choices remain")
			require.Contains(t, scene.output.GetOutputStr(false), history)
			state.EventQueueManager.Shutdown()
			require.True(t, scene.dialogStack.ContainsWidgetInstance(underlay))
		})
	}
}
