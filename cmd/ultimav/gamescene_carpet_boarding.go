package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// StartCarpetBoardingWorkflow initiates the magic carpet boarding workflow with event queue.
// Handles using the magic carpet from inventory to board it at the player's current location.
// Carpet boarding admission and ownership follow ref:VEH-0004.
//
// Flow:
//  1. Remove carpet from inventory (QuestItems)
//  2. Play stub sound
//  3. Spawn carpet at player position and board it
//  4. Show "Board carpet!" confirmation message
//  5. Advance game time
func (g *GameScene) StartCarpetBoardingWorkflow(commit *game_state.CarpetBoardingCommit) {
	g.startCarpetBoardingWorkflow(commit, nil)
}

func (g *GameScene) startCarpetBoardingWorkflow(commit *game_state.CarpetBoardingCommit, complete func()) {
	g.gameState.EventQueueManager.Workflow("carpet_boarding").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			var result game_state.CarpetBoardingResult
			if complete == nil {
				result = g.gameState.CommitCarpetBoarding(commit)
			} else {
				result = g.gameState.CommitCarpetBoardingForSurfaceUse(commit)
			}
			if result != game_state.CarpetBoardingSuccess {
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("Cannot place carpet here!", true)

				return
			}
			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundSummon)
		}).Finally(

		func() {
			g.keyboard.ResetReleaseDebouncing()
			if complete != nil {
				complete()
			}
		}).
		Start()
}
