package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatAutoActionIgnoresPartyLoyalSummon(t *testing.T) {
	gs, _, hostile := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	activeIndex := gs.CombatScene.GetActiveCombatantIndex()
	activePosition := gs.CombatScene.GetPartyMemberPosition(activeIndex)
	hostile.SetActive(false)
	ally := NewCombatMonster(
		references.MonsterRat,
		mustCombatEnemyReference(t, gs, references.MonsterRat),
		references.Position{X: activePosition.X + 1, Y: activePosition.Y},
		newFixedRNG(1),
	)
	ally.SetLoyalty(CombatMonsterLoyaltyParty)
	gs.CombatScene.AddMonsterToQueue(ally)

	assert.Equal(t, CombatAutoActionUnavailable, gs.PlanCombatAutoAction().Kind)
}

func TestPartyConeSpellsAffectPartyLoyalSummonWithoutChangingAttackTarget(t *testing.T) {
	gs, _, hostile := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	hostile.SetActive(false)
	allyRef := *mustCombatEnemyReference(t, gs, references.MonsterSkeleton)
	allyRef.HitPoints = 100 // Isolate cone damage from death/treasure RNG.
	ally := NewCombatMonster(
		references.MonsterSkeleton,
		&allyRef,
		references.Position{X: 5, Y: 5},
		newFixedRNG(1),
	)
	ally.SetLoyalty(CombatMonsterLoyaltyParty)
	allyIndex := gs.CombatScene.AddMonsterToQueue(ally)
	hpBefore := ally.CurrentHP()
	gs.SetRandomSeed(42)
	// Original spell projection rules have no loyalty gate: Intelligence saving throw, then Flame Wind's damage roll.
	// See ref:SPELL-0005.
	gs.RollOriginalD30()
	damage := gs.GetRNG().Intn(31)
	gs.SetRandomSeed(42)

	assert.True(t, gs.applyCombatInZuAtPosition(ally.Position(), 255))
	assert.True(t, ally.IsAsleep())
	assert.Equal(t, uint64(1), gs.GetRNG().Calls())
	assert.True(t, gs.applyCombatInFlamHurAtPosition(gs.CombatScene.GetActiveCombatantIndex(), ally.Position()))
	assert.Positive(t, damage)
	assert.Equal(t, hpBefore-damage, ally.CurrentHP(), "Flame Wind bypasses undead physical resistance")
	assert.Equal(t, uint64(2), gs.GetRNG().Calls())
	assert.True(t, ally.IsPartyLoyal())
	assert.NotEqual(t, allyIndex, gs.CombatScene.LastAttackTargets[gs.CombatScene.GetActiveCombatantIndex()])
}

func mustCombatEnemyReference(
	t *testing.T,
	gs *GameState,
	monsterType references.MonsterType,
) *references.EnemyReference {
	t.Helper()
	reference, err := gs.enemyReferenceForCombat(monsterType)
	require.NoError(t, err)

	return reference
}
