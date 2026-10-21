package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func (g *GameScene) startMoonstoneBurialWorkflow(moonstone references.Moonstone, complete func()) {
	g.gameState.EventQueueManager.Workflow("moonstone_burial").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			if !g.gameState.CommitMoonstoneBurial(moonstone) {
				panic("accepted moonstone burial could not commit")
			}

			if complete == nil {
				g.gameState.AdvanceGameplayMinutes(1)
			}
		}).ThenSequence(

		g.thenShowConfirmationDialog("Moonstone buried!")).Finally(

		func() {
			g.keyboard.ResetReleaseDebouncing()
			if complete != nil {
				complete()
			} else {
				g.gameState.ProcessPendingProvisionHours()
			}
		}).
		Start()
}
