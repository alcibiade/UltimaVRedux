package main

import (
	"slices"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

// combatChestFeedbackScope ties presentation to the encounter and chest selected
// by an authoritative Jimmy action. It is not inferred from message wording.
type combatChestFeedbackScope struct {
	combat      *game_state.CombatState
	encounterID game_state.CombatEncounterID
	chest       game_state.CombatChestIdentity
}

// captureCombatChestJimmyFeedback wraps only the synchronous authoritative action.
// The optional identity is returned only for a successful chest disarm; handled
// retains the command's existing meaning (including handled failures). Feedback
// remains immediate and unawaited, and its normal history output is untouched.
func (g *GameScene) captureCombatChestJimmyFeedback(
	action func() (handled bool, disarmed *game_state.CombatChestIdentity),
) bool {
	combat := g.gameState.CombatScene
	if combat == nil {
		panic("captureCombatChestJimmyFeedback requires combat")
	}
	scope := combatChestFeedbackScope{combat: combat, encounterID: combat.EncounterID}
	before := slices.Clone(g.transientActionFeedback)
	handled, disarmed := action()
	if disarmed == nil {
		return handled
	}
	scope.chest = *disarmed
	if !g.combatChestFeedbackScopeIsCurrent(scope) || !combat.HasChestIdentity(scope.chest) {
		return handled
	}
	for _, feedback := range g.transientActionFeedback {
		if slices.Contains(before, feedback) {
			continue
		}
		if g.combatChestFeedback == nil {
			g.combatChestFeedback = make(map[*transientActionFeedback]combatChestFeedbackScope)
		}
		g.combatChestFeedback[feedback] = scope
	}

	return handled
}

func (g *GameScene) combatChestFeedbackScopeIsCurrent(scope combatChestFeedbackScope) bool {
	if g.gameState == nil || scope.combat == nil || g.gameState.CombatScene != scope.combat ||
		scope.combat.EncounterID != scope.encounterID || scope.chest.ObjectID == 0 {
		return false
	}
	// Open removes the object before its summary is admitted. Only the exact
	// detached reveal snapshot can bridge that gap; a reused slot cannot.
	return scope.combat.HasChestIdentity(scope.chest) ||
		(g.combatLootPreview != nil && g.combatLootPreview.chestScope == scope)
}

// retireCombatChestFeedback runs at summary admission, before the modal push.
// Match only captured instances for this reveal; other cards retain their owner
// and lifetime. Expiry, eviction and external removal may already have retired it.
func (g *GameScene) retireCombatChestFeedback(scope combatChestFeedbackScope) {
	now := g.gameState.ElapsedMs
	g.reconcileTransientActionFeedback(now, true)
	removed := g.retireExternallyRemovedActionFeedback()
	for index := len(g.transientActionFeedback) - 1; index >= 0; index-- {
		feedback := g.transientActionFeedback[index]
		if owner, found := g.combatChestFeedback[feedback]; found && owner == scope {
			g.evictTransientActionFeedback(index)
			removed = true
		}
	}
	if removed {
		g.layoutTransientActionFeedback(now, g.currentTransientFeedbackLayout())
	}
}

// retireStaleCombatChestFeedback also runs while a modal pauses toast lifetime.
// Removed objects and replacement/ended encounters must not retain presentation
// associations, even when identical contents reuse the slot and position.
func (g *GameScene) retireStaleCombatChestFeedback() bool {
	removed := false
	for index := len(g.transientActionFeedback) - 1; index >= 0; index-- {
		feedback := g.transientActionFeedback[index]
		if scope, found := g.combatChestFeedback[feedback]; found && !g.combatChestFeedbackScopeIsCurrent(scope) {
			g.evictTransientActionFeedback(index)
			removed = true
		}
	}

	return removed
}
