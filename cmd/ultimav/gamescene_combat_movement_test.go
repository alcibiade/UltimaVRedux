package main

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestCombatMovementOffset_TraversesActualRoundedGridDistance(t *testing.T) {
	g := &GameScene{}
	from := references.Position{X: 4, Y: 5}
	to := references.Position{X: 5, Y: 5}
	g.startCombatMovement(true, 2, 0, from, to)

	fromX, _ := sprites.ConvertPosition16to18(int(from.X), int(from.Y))
	toX, _ := sprites.ConvertPosition16to18(int(to.X), int(to.Y))
	x, y := g.combatMovementOffset(true, 2, 0)
	assert.Equal(t, fromX-toX, x, "animation must begin at the previous rendered tile")
	assert.Zero(t, y)

	g.updateCombatMovementProgress(0.5)
	x, _ = g.combatMovementOffset(true, 2, 0)
	assert.Equal(t, math.Round((fromX-toX)/2), x)

	g.updateCombatMovementProgress(1)
	x, y = g.combatMovementOffset(true, 2, 0)
	assert.Zero(t, x)
	assert.Zero(t, y)
}

func TestCombatMovementOffset_OnlyMovesSelectedEntity(t *testing.T) {
	g := &GameScene{}
	g.startCombatMovement(false, 3, 0,
		references.Position{X: 5, Y: 6},
		references.Position{X: 5, Y: 5},
	)

	x, y := g.combatMovementOffset(true, 3, 0)
	assert.Zero(t, x)
	assert.Zero(t, y)

	x, y = g.combatMovementOffset(false, 2, 0)
	assert.Zero(t, x)
	assert.Zero(t, y)
}

func TestCombatMovementOffset_UsesExactCloneSlot(t *testing.T) {
	g := &GameScene{}
	g.startCombatMovement(
		true, 2, 1, references.Position{X: 4, Y: 4}, references.Position{X: 5, Y: 4},
	)
	g.updateCombatMovementProgress(0.5)

	baseX, baseY := g.combatMovementOffset(true, 2, 0)
	cloneX, _ := g.combatMovementOffset(true, 2, 1)

	assert.Zero(t, baseX)
	assert.Zero(t, baseY)
	assert.NotZero(t, cloneX)
}

func TestCombatMovementDurations_PrioritizePlayerResponsiveness(t *testing.T) {
	assert.Equal(t, uint64(333), combatPlayerMovementDurationMs)
	assert.Equal(t, uint64(650), combatEnemyMovementDurationMs)
	assert.Less(t, combatPlayerMovementDurationMs, combatEnemyMovementDurationMs)
}

func TestScaledCombatMovementDurationUsesConfiguredSpeedFactor(t *testing.T) {
	tests := []struct {
		factor         float64
		playerDuration uint64
		enemyDuration  uint64
	}{
		{factor: 0.25, playerDuration: 1332, enemyDuration: 2600},
		{factor: 1.0, playerDuration: 333, enemyDuration: 650},
		{factor: 2.0, playerDuration: 167, enemyDuration: 325},
		{factor: 4.0, playerDuration: 83, enemyDuration: 163},
	}

	for _, test := range tests {
		debugConfiguration := &config.DebugConfiguration{ //nolint:exhaustruct // Focused speed fixture.
			CombatMovementSpeedFactor: test.factor,
		}
		assert.Equal(t, test.playerDuration,
			scaledCombatMovementDuration(combatPlayerMovementDurationMs, debugConfiguration))
		assert.Equal(t, test.enemyDuration,
			scaledCombatMovementDuration(combatEnemyMovementDurationMs, debugConfiguration))
	}
}
