package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newLoyaltyTestMonster(position references.Position, seed int) *CombatMonster {
	reference := &references.EnemyReference{
		Dexterity: 10,
		HitPoints: 10,
	}

	return NewCombatMonster(references.MonsterOrc, reference, position, newFixedRNG(seed))
}

func TestCombatMonsterManager_LoyaltyQueries(t *testing.T) {
	manager := NewCombatMonsterManager()
	hostile := newLoyaltyTestMonster(references.Position{X: 1, Y: 1}, 1)
	partyLoyal := newLoyaltyTestMonster(references.Position{X: 2, Y: 2}, 2)
	partyLoyal.SetLoyalty(CombatMonsterLoyaltyParty)
	deadHostile := newLoyaltyTestMonster(references.Position{X: 3, Y: 3}, 3)
	deadHostile.SetCurrentHP(0)

	manager.AddMonster(hostile)
	manager.AddMonster(partyLoyal)
	manager.AddMonster(deadHostile)

	assert.True(t, hostile.IsHostile(), "normal monsters default to monster loyalty")
	assert.True(t, partyLoyal.IsPartyLoyal(), "summoned or charmed monsters can be marked party-loyal")
	assert.Equal(t, 1, manager.CountActiveHostile())
	assert.Equal(t, 1, manager.CountActivePartyLoyal())
	assert.Equal(t, []*CombatMonster{hostile}, manager.GetActiveHostileMonsters())
	assert.Equal(t, []*CombatMonster{partyLoyal}, manager.GetActivePartyLoyalMonsters())
	assert.Equal(t, 2, manager.CountActive(), "broad active count remains unfiltered")
}

func TestCombatMonsterManager_StackedPositionQueriesRetainStableOrder(t *testing.T) {
	manager := NewCombatMonsterManager()
	position := references.Position{X: 4, Y: 5}
	first := newLoyaltyTestMonster(position, 1)
	elsewhere := newLoyaltyTestMonster(references.Position{X: 5, Y: 5}, 2)
	second := newLoyaltyTestMonster(position, 3)

	firstIndex := manager.AddMonster(first)
	manager.AddMonster(elsewhere)
	secondIndex := manager.AddMonster(second)

	assert.Equal(t, []*CombatMonster{first, second}, manager.GetMonstersAtPosition(position))
	assert.Equal(t, []int{firstIndex, secondIndex}, manager.GetMonsterIndicesAtPosition(position))
	monster, index := manager.GetMonsterIndexAtPosition(position)
	require.Same(t, first, monster, "existing singular query should retain first-match behavior")
	assert.Equal(t, firstIndex, index)
}
