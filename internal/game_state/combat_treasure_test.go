package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatTreasureRollBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		treasureValue     int
		dropRoll          int
		trapRoll          int
		wantDrop          bool
		wantLockedTrapped bool
	}{
		{name: "zero treasure", treasureValue: 0, dropRoll: 1, trapRoll: 1},
		{name: "drop roll above value", treasureValue: 10, dropRoll: 11, trapRoll: 1},
		{name: "drop roll equals value", treasureValue: 10, dropRoll: 10, trapRoll: 10, wantDrop: true},
		{
			name:              "second roll below value traps",
			treasureValue:     10,
			dropRoll:          10,
			trapRoll:          9,
			wantDrop:          true,
			wantLockedTrapped: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			drop, lockedAndTrapped := combatTreasureRoll(test.treasureValue, test.dropRoll, test.trapRoll)

			assert.Equal(t, test.wantDrop, drop)
			assert.Equal(t, test.wantLockedTrapped, lockedAndTrapped)
		})
	}
}

func TestCombatDeathObjectEligibility(t *testing.T) {
	t.Parallel()

	enemy := references.EnemyReference{ //nolint:exhaustruct // Only treasure eligibility fields are relevant.
		HitPoints:      10,
		TreasureNumber: 10,
		EnemyAbilities: map[references.EnemyAbility]bool{},
	}
	monster := NewCombatMonster(
		references.MonsterOrc,
		&enemy,
		references.Position{},
		newFixedRNG(1),
	)

	assert.True(t, combatDeathObjectEligible(monster, &references.Tile{Index: indexes.Grass}))   //nolint:exhaustruct
	assert.False(t, combatDeathObjectEligible(monster, &references.Tile{Index: indexes.Water1})) //nolint:exhaustruct
	assert.False(t,
		combatDeathObjectEligible(monster, &references.Tile{Index: indexes.BrickWallArchway})) //nolint:exhaustruct

	enemy.EnemyAbilities[references.NoCorpse] = true
	assert.False(t, combatDeathObjectEligible(monster, &references.Tile{Index: indexes.Grass})) //nolint:exhaustruct
}

func TestRollCombatTreasureConsumesOriginalNumberOfRolls(t *testing.T) {
	t.Parallel()

	t.Run("failed drop consumes one roll", func(t *testing.T) {
		t.Parallel()

		rollCount := 0
		rollDice := func() int {
			rollCount++

			return 30
		}

		drop, trapped := rollCombatTreasure(1, rollDice)

		assert.False(t, drop)
		assert.False(t, trapped)
		assert.Equal(t, 1, rollCount)
	})

	t.Run("zero treasure still consumes corpse roll", func(t *testing.T) {
		t.Parallel()

		rollCount := 0
		rollDice := func() int {
			rollCount++

			return 1
		}

		drop, trapped := rollCombatTreasure(0, rollDice)

		assert.False(t, drop)
		assert.False(t, trapped)
		assert.Equal(t, 1, rollCount)
	})

	t.Run("successful drop consumes independent trap roll", func(t *testing.T) {
		t.Parallel()

		rolls := []int{10, 9}
		rollIndex := 0
		rollDice := func() int {
			roll := rolls[rollIndex]
			rollIndex++

			return roll
		}

		drop, trapped := rollCombatTreasure(10, rollDice)

		assert.True(t, drop)
		assert.True(t, trapped)
		assert.Equal(t, 2, rollIndex)
	})
}
