package game_state

// CastUseCommitSequence lets a blocking Cast/Use owner distinguish commitment
// from merely opening a selector. It is transient orchestration state, not save data.
func (g *GameState) CastUseCommitSequence() uint64 {
	return g.castUseCommitSequence
}

// CommitCastUseAttempt records a validated final confirmation before its effect
// (and any gameplay RNG). Resource-owning operations call it after validation.
// REDUX DEVIATION: original prepared-spell rules and original item-use rules charge before
// prerequisite checks or target input. Redux makes rejection and cancellation
// free, but a confirmed valid attempt is paid even when its effect fails.
// See ref:SPELL-0001.
func (g *GameState) CommitCastUseAttempt() {
	g.castUseCommitSequence++
	if cycle := g.activeDungeonCommand; cycle != nil && cycle.deferredCharge {
		g.chargeDungeonCommandCycle(cycle)
	}
}
