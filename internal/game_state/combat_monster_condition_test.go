package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatMonsterConditionUsesOriginalHPBandsAndRNG(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		hp        int
		roll      int
		initial   bool
		wantFlee  bool
		wantRolls int
	}{
		{name: "below quarter flees", hp: 4, roll: 0, wantFlee: true, wantRolls: 0},
		{name: "quarter boundary failed roll clears", hp: 5, roll: 251, initial: true, wantFlee: false, wantRolls: 1},
		{name: "middle band successful roll flees", hp: 9, roll: 252, wantFlee: true, wantRolls: 1},
		{name: "half boundary clears", hp: 10, roll: 255, initial: true, wantFlee: false, wantRolls: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			monster := newConditionTestMonster(t)
			monster.SetCurrentHP(test.hp)
			monster.SetFleeing(test.initial)
			rng := newSequenceRNG(test.roll)

			applyCombatMonsterCondition(monster, rng)

			assert.Equal(t, test.wantFlee, monster.IsFleeing())
			assert.Equal(t, test.wantRolls, rng.index)
		})
	}
}

func TestCombatMonsterConditionHandlesSmallMaximumHP(t *testing.T) {
	t.Parallel()

	monster := newConditionTestMonster(t)
	monster.enemyRef.HitPoints = 3
	monster.SetCurrentHP(1)
	monster.SetFleeing(true)
	rng := newSequenceRNG(255)

	applyCombatMonsterCondition(monster, rng)

	assert.False(t, monster.IsFleeing())
	assert.Zero(t, rng.index)
}

func TestSurvivingMonsterHitAppliesConditionOnlyForNormalDamage(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterGazer, references.Down, 42)
	if gs == nil {
		return
	}
	monster.SetCurrentHP(4)

	gs.resolveSurvivingCombatMonsterHit(monster, 0, false)
	assert.False(t, monster.IsFleeing(), "magical damage must not invoke condition")

	gs.resolveSurvivingCombatMonsterHit(monster, 0, true)
	assert.True(t, monster.IsFleeing(), "normal damage must invoke condition")
}

func newConditionTestMonster(t *testing.T) *CombatMonster {
	t.Helper()

	reference := references.EnemyReference{HitPoints: 20} //nolint:exhaustruct // Only HP is relevant to condition().
	monster := NewCombatMonster(references.MonsterGazer, &reference, references.Position{}, newFixedRNG(0))
	require.Equal(t, 20, monster.MaxHP())

	return monster
}
