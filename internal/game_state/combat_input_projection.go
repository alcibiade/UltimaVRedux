package game_state

// CombatInputOpportunity binds transient input to an exact live encounter, queue
// entry, and entry-local dispatch generation. It is neither serializable nor an
// action authority. Only the normal command handlers may commit an action.
type CombatInputOpportunity struct {
	owner      *GameState
	combat     *CombatState
	entry      *TurnQueueEntry
	encounter  CombatEncounterID
	generation uint64
	Actor      CombatTurnIdentity
}

// CombatInputProjection is a nonmutating input-owner view. Waiting denotes an
// intervening scheduled enemy/auto-pass step; Ready excludes committed actions.
type CombatInputProjection struct {
	Opportunity CombatInputOpportunity
	Waiting     bool
	Ready       bool
}

// ProjectSoloCombatInput reads the selected entry's actual dispatch generation;
// it does not scan future countdowns, claim a scheduler step, execute turn-start
// rules, plan AI, or consume randomness. Ambiguous clones are outside this scope.
func (g *GameState) ProjectSoloCombatInput() (CombatInputProjection, bool) {
	combat := g.CombatScene
	if combat == nil || !combat.turnsStarted || !combat.presentTurns ||
		!combat.SoloMoveMode || combat.TurnQueue == nil || combat.hasPlayerActionOwner() {
		return CombatInputProjection{}, false
	}
	actor := CombatTurnIdentity{IsPartyMember: true, EntityIndex: combat.SoloMoveCharacterIndex}
	var selected *TurnQueueEntry
	for _, entry := range combat.TurnQueue.entries {
		if !combat.isLivingCombatPCEntry(entry, actor.EntityIndex) {
			continue
		}
		if selected != nil || entry.Slot != 0 {
			return CombatInputProjection{}, false
		}
		selected = entry
	}
	if selected == nil || !selected.Combatant.CanTakeTurn() ||
		combat.isCombatPCSleeping(actor) || combat.isCombatPCUnderground(actor) ||
		g.combatIdentityLoyalty(actor) != CombatMonsterLoyaltyParty {
		return CombatInputProjection{}, false
	}
	projection := CombatInputProjection{Opportunity: CombatInputOpportunity{
		owner: g, combat: combat, entry: selected, encounter: combat.EncounterID,
		generation: selected.dispatchGeneration, Actor: actor,
	}}
	pending := combat.pendingPresentationStep
	if pending == nil {
		projection.Ready = selected.dispatchGeneration != 0 &&
			combat.GetCurrentTurnIdentity() == actor && g.GetActivePlayerIndex() == actor.EntityIndex

		return projection, projection.Ready
	}
	// A skipped selected opportunity or its committed-action presentation is not
	// an intervening wait. Never silently carry a verb into a future opportunity.
	if pending.step.Kind == CombatPresentationTurnCompletion || pending.step.Actor == actor {
		return CombatInputProjection{}, false
	}
	// This is the selected entry's next opportunity, however many other actors
	// intervene. Only its real dispatch advances this generation; enemy removal,
	// changing DEX and temporal suppression cannot invalidate it indirectly.
	projection.Opportunity.generation++
	projection.Waiting = true

	return projection, true
}
