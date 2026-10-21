package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func (g *GameScene) startNewOrderWorkflowWithCompletion(complete func()) {
	var dialog *widgets.NewOrderDialog
	g.gameState.EventQueueManager.Workflow("new_order_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).ThenSequence(

		g.thenShowNewOrderDialog(&dialog)).
		Finally(func() {
			if dialog != nil {
				g.dialogStack.RemoveWidgetInstance(dialog)
			}
			if complete != nil {
				complete()
			}
		}).
		Start()
}
