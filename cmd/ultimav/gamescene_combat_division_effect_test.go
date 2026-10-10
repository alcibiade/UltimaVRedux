package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatDivisionEffectHighlightsOnlyNewChildAfterTether(t *testing.T) {
	g := &GameScene{
		combatDivisionEffects: []CombatDivisionEffect{{
			ParentIndex: 2, ChildIndex: 3, Elapsed: combatDivisionTetherDurationSeconds,
		}},
	}

	phase, highlighted := g.combatDivisionChildOutlinePhase(3)

	assert.True(t, highlighted)
	assert.Equal(t, 0, phase)
	_, parentHighlighted := g.combatDivisionChildOutlinePhase(2)
	assert.False(t, parentHighlighted)
}

func TestCombatDivisionEffectExpiresAndClearsOutsideCombat(t *testing.T) {
	g := &GameScene{
		gameState: &game_state.GameState{},
		combatDivisionEffects: []CombatDivisionEffect{{
			ParentPosition: references.Position{X: 4, Y: 4},
		}},
	}

	g.updateCombatDivisionEffects(0.1)

	assert.Empty(t, g.combatDivisionEffects)
}

func TestCombatDivisionEffectDoesNotHighlightChildDuringTether(t *testing.T) {
	g := &GameScene{
		combatDivisionEffects: []CombatDivisionEffect{{
			ChildIndex: 3, Elapsed: combatDivisionTetherDurationSeconds / 2,
		}},
	}

	_, highlighted := g.combatDivisionChildOutlinePhase(3)

	assert.False(t, highlighted)
}
