package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/hajimehoshi/ebiten/v2"
)

// captureCombatSoloReset runs before pending verbs and queue advancement. There
// is no retained request: the exact live boundary is validated and changed now.
func (g *GameScene) captureCombatSoloReset() bool {
	state := &g.transientCommandInput
	if g.gameState.CombatScene == nil || !state.sample.Held[ebiten.Key0] {
		return false
	}
	if state.sample.HasModifiers() || !state.sample.Pressed[ebiten.Key0] {
		g.keyboard.SuppressUntilRelease(ebiten.Key0)

		return false
	}
	boundary, eligible := g.gameState.ProjectCombatSoloReset()
	if !eligible || !g.combatSoloResetInputOwned(boundary) {
		if g.inputBlocked || g.gameState.CombatScene.HasScheduledCombatStep() ||
			g.gameState.EventQueueManager.HasInputBlockingWorkflows() {
			// A player action/other blocked owner cannot carry 0 into readiness.
			g.keyboard.SuppressUntilRelease(ebiten.Key0)
		}

		return false
	}
	if !g.keyboard.ConsumePressEdge(state.sample, ebiten.Key0) {
		return false
	}
	g.keyboard.SuppressUntilRelease(ebiten.Key0)
	for _, key := range boundKeysGame {
		if state.sample.Held[key] && (bufferedCombatVerb(key) != "" || isEarlyCombatDirection(key)) {
			// Reset wins this sample. Neither a losing verb nor an early arrow
			// may become ordinary input when this same Update releases the wait.
			g.keyboard.SuppressUntilRelease(key)
		}
	}
	g.clearPendingCombatCommand()
	g.gameState.ResetCombatSoloDuringWait()

	return true
}

func (g *GameScene) combatSoloResetInputOwned(boundary game_state.CombatSoloResetBoundary) bool {
	manager := g.gameState.EventQueueManager
	if !boundary.Claimed {
		// A newly scheduled step may not have reached scene review yet. No
		// unrelated workflow or manual input block may borrow this gap.
		return !g.inputBlocked && !manager.HasActiveWorkflows() && !manager.HasInputBlockingWorkflows()
	}
	switch boundary.Step.Kind {
	case game_state.CombatPresentationMonsterTurn:
		return manager.HasOnlyWorkflow("monster_turn_delay")
	case game_state.CombatPresentationAutoPass:
		return manager.HasOnlyWorkflow("combat_pc_auto_pass")
	case game_state.CombatPresentationTurnCompletion:
		return false
	default:
		return false
	}
}
