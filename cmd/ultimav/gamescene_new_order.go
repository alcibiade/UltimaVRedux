package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// StartNewOrderWorkflow initiates the party member reordering dialog sequence.
// Two-stage selection: "Swap" → "with [Name]" → confirmation message.
// Party reordering is an exploration command (ref:CMD-0001).
//
// Flow:
//  1. Show NewOrderDialog with two-stage character selection
//  2. Stage 1: "Swap" - select first character (Avatar grayed out)
//  3. Stage 2: "with [Name]" - select second character
//  4. Execute swap via PartyState.SwapMembers()
//  5. Show confirmation: "[Name] and [Name] swapped!"
//
// No time cost for party reordering (skipFinishTurn = true in input handler).
func (g *GameScene) StartNewOrderWorkflow() {
	g.startNewOrderWorkflowWithCompletion(nil)
}

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
