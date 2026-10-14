package game_state

// CombatUseOwner retains the encounter and exact scheduled actor boundary that
// opened Use. An index alone can refer to a later turn or a replacement encounter.
type CombatUseOwner struct {
	combat   *CombatState
	actor    CombatTurnIdentity
	sequence uint64
}

// CaptureCombatUseOwner captures ownership without changing gameplay or forecasting.
func (g *GameState) CaptureCombatUseOwner() CombatUseOwner {
	if g.CombatScene == nil {
		return CombatUseOwner{}
	}

	return CombatUseOwner{
		combat: g.CombatScene, actor: g.CombatScene.GetCurrentTurnIdentity(), sequence: g.CombatScene.nextPossessionTurn,
	}
}

// isRetainedUseContext also retains exploration: entering combat invalidates its confirmation.
func (g *GameState) isRetainedUseContext(owner CombatUseOwner) bool {
	if owner.combat == nil {
		return g.CombatScene == nil
	}

	return g.IsCurrentCombatUseOwner(owner)
}

// IsCurrentCombatUseOwner rejects stale actor, clone, encounter and turn changes.
func (g *GameState) IsCurrentCombatUseOwner(owner CombatUseOwner) bool {
	return owner.combat != nil && owner.combat == g.CombatScene &&
		owner.actor == g.CombatScene.GetCurrentTurnIdentity() && owner.sequence == g.CombatScene.nextPossessionTurn &&
		owner.actor.IsPartyMember && g.CombatScene.isCombatPCActive(owner.actor) &&
		!g.CombatScene.isCombatPCPossessed(owner.actor)
}
