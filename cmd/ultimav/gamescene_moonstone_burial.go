package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// StartMoonstoneBurialWorkflow initiates the moonstone burial workflow with event queue.
// Handles burying a moonstone at the player's current location to create a permanent moongate.
// Moonstone burial admission and persistence follow ref:MOON-0004.
//
// Flow:
//  1. Remove moonstone from inventory
//  2. Update moonstone system with burial location
//  3. Show "Moonstone buried!" confirmation message
//  4. Advance game time
func (g *GameScene) StartMoonstoneBurialWorkflow(moonstone references.Moonstone) {
	g.startMoonstoneBurialWorkflow(moonstone, nil)
}

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
