package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

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
