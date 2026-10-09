package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// dungeonViewDialogPercents uses the full modal canvas so the enhanced legend
// sits beside the source-faithful map when space permits and below it otherwise.
var dungeonViewDialogPercents = sprites.PercentBasedPlacement{ //nolint:gochecknoglobals // Immutable layout contract.
	StartPercentX: 0.03,
	EndPercentX:   0.97,
	StartPercentY: 0.04,
	EndPercentY:   0.96,
}

func (g *GameScene) thenShowDungeonViewDialog(
	model *game_state.DungeonViewModel,
	dialog **widgets.DungeonViewDialog,
) event_queue.Sequence {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Then(func() {
			*dialog = widgets.NewDungeonViewDialog(widgets.DungeonViewDialogConfig{
				Placement: dungeonViewDialogPercents,
				Keyboard:  g.keyboard,
				Model:     *model,
			})
			g.dialogStack.PushModalDialog(*dialog)
		}).While(
			func() bool { return !(*dialog).IsDismissed() },
			func() {},
			DialogPollingRateMs,
		).Then(func() {
			g.dialogStack.RemoveWidgetInstance(*dialog)
			*dialog = nil
		})
	}
}

// StartDungeonViewWorkflow owns the accepted gem command through modal dismissal.
func (g *GameScene) StartDungeonViewWorkflow(
	token game_state.DungeonCommandCycle,
	model game_state.DungeonViewModel,
) {
	var dialog *widgets.DungeonViewDialog
	g.gameState.EventQueueManager.Workflow("dungeon_view_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).
		ThenSequence(g.thenShowDungeonViewDialog(&model, &dialog)).
		Finally(func() {
			if dialog != nil {
				dialog.Dismiss()
				g.dialogStack.RemoveWidgetInstance(dialog)
				dialog = nil
			}
			g.gameState.CompleteDungeonCommandCycle(token, game_state.DungeonFullEndPlayer)
		}).
		Start()
}
