package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func (g *GameScene) startDungeonHoleUpWorkflow(token game_state.DungeonCommandCycle) {
	hours := 0
	cancelled := false
	guardIndex := game_state.NoGuardAssigned
	var scene *camping.CampingScene
	var plan *game_state.DungeonCampingPlan
	var guardDialog *widgets.CharacterSelectionDialog
	prepared := false
	complete := func() {
		if guardDialog != nil {
			g.dialogStack.RemoveWidgetInstance(guardDialog)
		}
		g.keyboard.ResetReleaseDebouncing()
		g.gameState.CompleteDungeonCampingCommand(token)
	}

	g.gameState.EventQueueManager.Workflow("dungeon_hole_up").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).
		ThenSequence(g.thenShowDungeonCampDurationInput(func(selectedHours int) {
			hours = selectedHours
		})).
		If(func() bool {
			return hours > 0 && g.gameState.DungeonCampingWatchEligibleCount() > 1
		}, func(queue *event_queue.SequentialEventBuilder) {
			queue.ThenSequence(g.thenShowGuardSelection(func(selectedGuard int, cancel bool) {
				guardIndex = selectedGuard
				cancelled = cancel
			}, &guardDialog))
		}).
		If(func() bool { return hours > 0 && !cancelled }, func(queue *event_queue.SequentialEventBuilder) {
			queue.Then(func() {
				var err error
				scene, plan, err = g.gameState.PrepareDungeonCampingPlan(hours, guardIndex, token)
				if err != nil {
					logging.Error(logging.CategoryGameState, "Unable to prepare dungeon camping: %v", err)
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("Camping could not begin.", true)

					return
				}
				prepared = true
			}).If(func() bool { return prepared }, func(campQueue *event_queue.SequentialEventBuilder) {
				g.gameState.AddDungeonCampingSequenceToBuilder(campQueue, scene, plan, complete)
			})
		}).Finally(complete).
		Start()
}
