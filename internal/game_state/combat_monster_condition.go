package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// applyCombatMonsterCondition replicates monster condition update from original combat damage rules.
// See ref:COMBAT-0005.
func applyCombatMonsterCondition(monster *CombatMonster, rng references.RandomProvider) {
	quarterHP := monster.MaxHP() / 4

	switch {
	case monster.CurrentHP() < quarterHP:
		monster.SetFleeing(true)
	case monster.CurrentHP() < quarterHP*2:
		// The original documents this branch as one chance in 64.
		monster.SetFleeing(rng.Intn(256) > 251)
	default:
		monster.SetFleeing(false)
	}
}

func (g *GameState) resolveSurvivingCombatMonsterHit(
	monster *CombatMonster,
	monsterIdx int,
	normalDamage bool,
) {
	if normalDamage {
		applyCombatMonsterCondition(monster, g.AsRandomProvider())
	}
	g.tryDivideCombatMonster(monster, monsterIdx)
}
