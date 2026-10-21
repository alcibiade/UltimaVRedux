package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestFinalizeCombatMonsterDeathSaturatesExperience(t *testing.T) {
	tests := []struct {
		name            string
		startingXP      uint16
		awardExperience bool
	}{
		{name: "below cap", startingXP: 0, awardExperience: true},
		{name: "crossing cap", startingXP: MaxExperienceValue - 1, awardExperience: true},
		{name: "at cap", startingXP: MaxExperienceValue, awardExperience: true},
		{name: "no award", startingXP: 100, awardExperience: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 1)
			attackerIdx := gs.CombatScene.GetActiveCombatantIndex()
			attacker := gs.PartyState.GetCharacter(attackerIdx)
			attacker.Exp = test.startingXP
			monster.SetCurrentHP(0)
			nominalAward := combatCalcExperience(monster.MaxHP())
			require.Greater(t, nominalAward, 1)

			gs.finalizeCombatMonsterDeathWithPolicy(
				monster, 0, attacker, attackerIdx, test.awardExperience, false,
			)
			credited := int(attacker.Exp - test.startingXP)

			switch test.name {
			case "below cap":
				assert.Equal(t, nominalAward, credited)
				assert.Equal(t, uint16(nominalAward), attacker.Exp)
			case "crossing cap":
				assert.Equal(t, 1, credited)
				assert.Equal(t, uint16(MaxExperienceValue), attacker.Exp)
			case "at cap":
				assert.Zero(t, credited)
				assert.Equal(t, uint16(MaxExperienceValue), attacker.Exp)
			case "no award":
				assert.Zero(t, credited)
				assert.Equal(t, test.startingXP, attacker.Exp)
			}
		})
	}
}

func TestCombatSpellKillUsesSaturatingExperienceAward(t *testing.T) {
	gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 1)
	casterIdentity := gs.CombatScene.GetCurrentTurnIdentity()
	caster := gs.PartyState.GetCharacter(casterIdentity.EntityIndex)
	caster.Exp = MaxExperienceValue - 1
	monster.SetCurrentHP(1)
	mock.Reset()

	gs.applyCombatProjectileDamage(
		pendingCombatSpell{
			spell:          references.VasFlam,
			caster:         casterIdentity,
			target:         CombatTurnIdentity{EntityIndex: 0},
			targetPosition: monster.Position(),
			targetSet:      true,
			hit:            CombatHitResult{Outcome: CombatHitNeverMiss, AutomaticHit: true},
		},
		monster,
		CombatDamageResult{Damage: 1},
		1,
	)

	assert.True(t, monster.IsDead())
	assert.Equal(t, uint16(MaxExperienceValue), caster.Exp)
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatEventSpell, mock.CombatEvents[0].Kind)
	assert.True(t, mock.CombatEvents[0].Fatal)
}
