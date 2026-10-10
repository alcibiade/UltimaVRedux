package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// StartSkullKeyWorkflow initiates the skull key unlock workflow with event queue.
// Skull Keys unlock magic doors; consumption follows ref:INV-0003.
// REDUX DEVIATION: CMD-0003: original Use pays before unlocking. Redux prompts
// first, then pays for a confirmed direction even when no door is unlocked.
func (g *GameScene) StartSkullKeyWorkflow(commit *game_state.SkullKeyUseCommit) {
	g.startSkullKeyWorkflow(commit, nil)
}

func (g *GameScene) startSkullKeyWorkflow(commit *game_state.SkullKeyUseCommit, complete func()) {
	commitSequence := g.gameState.CastUseCommitSequence()
	canUnlock := false
	g.gameState.EventQueueManager.Workflow("skull_key_workflow").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.gameState.SystemCallbacks.Message.AddRowStr("Skull Key")
			canUnlock = g.gameState.ReviewSkullKeyUse().Result == game_state.SkullKeyUseReady
			if !canUnlock {
				g.gameState.SystemCallbacks.UI.ShowActionFeedback("Not here!", true)
			}
		}).If(func() bool { return canUnlock }, func(q *event_queue.SequentialEventBuilder) {
		q.ThenSequence(g.thenShowCastUseDirectionPrompt(
			"Direction-",
			func(dir references.Direction) {
				if g.gameState.CommitSkullKeyUse(commit) != game_state.SkullKeyUseConsumed {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("You have none!", true)

					return
				}
				review := g.gameState.ReviewSkullKeyUnlock(commit, dir)
				result := g.gameState.CommitSkullKeyUnlockForSurfaceUse(review)
				if result != game_state.SkullKeyUnlockSuccess {
					g.gameState.SystemCallbacks.UI.ShowActionFeedback("No effect!", true)

					return
				}
				g.playKapowAtPosition(review.Target)

				g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundUnlock)

				g.gameState.SystemCallbacks.UI.ShowActionFeedback("Unlocked!", true)
			},
			func() {},
		))
	}).Finally(

		func() {
			g.clearCastUseDirectionPrompt()
			g.gameState.CancelSkullKeyUnlock(commit)
			g.keyboard.ResetReleaseDebouncing()
			if complete != nil {
				complete()
			} else if g.gameState.CastUseCommitSequence() != commitSequence {
				g.gameState.AdvanceGameplayMinutesAndProcessProvisions(1)
			}
		}).
		Start()
}
