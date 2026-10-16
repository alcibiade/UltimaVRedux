package game_state

// CombatSoloResetBoundary describes a scheduled enemy or condition-driven skip.
// Claimed distinguishes an installed presentation owner from the gap before scene
// review. Reading this value never claims, resolves or completes the step.
type CombatSoloResetBoundary struct {
	Step    CombatPresentationStep
	Claimed bool
}

// ProjectCombatSoloReset reports only scheduler authority. The scene must also
// validate its workflow and input owner before admitting an early reset. Unlike a
// pending command, recovery does not require an actionable selected character.
func (g *GameState) ProjectCombatSoloReset() (CombatSoloResetBoundary, bool) {
	combat := g.CombatScene
	if combat == nil || !combat.turnsStarted || !combat.presentTurns || combat.TurnQueue == nil ||
		combat.hasPlayerActionOwner() || combat.pendingPresentationStep == nil {
		return CombatSoloResetBoundary{}, false
	}
	pending := combat.pendingPresentationStep
	if _, err := g.validateCombatPresentationToken(pending.step.Token); err != nil {
		return CombatSoloResetBoundary{}, false
	}
	switch pending.step.Kind {
	case CombatPresentationMonsterTurn, CombatPresentationAutoPass:
		return CombatSoloResetBoundary{Step: pending.step, Claimed: pending.reviewed}, true
	case CombatPresentationTurnCompletion:
		return CombatSoloResetBoundary{}, false
	default:
		return CombatSoloResetBoundary{}, false
	}
}

// ResetCombatSoloDuringWait clears only future selection at a live enemy/skip
// boundary. It preserves the current actor, token, target and all action effects.
// Already-all-party mode is accepted without mutation or repeated output.
// REDUX DEVIATION: COMBAT-0003: original numeric selection is documented at the
// player-input boundary; original out-of-turn polling is not established. Redux
// explicitly permits this recovery while waiting so an incapacitated selection
// cannot trap party control. No initiative or completed skip is restored.
func (g *GameState) ResetCombatSoloDuringWait() bool {
	if _, eligible := g.ProjectCombatSoloReset(); !eligible {
		return false
	}
	g.clearCombatSoloSelection()

	return true
}
