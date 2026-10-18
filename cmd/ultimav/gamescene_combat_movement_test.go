package main

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
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

func TestFastScheduledMonsterPacingSpeciesAndPartyExclusions(t *testing.T) {
	for _, kind := range []references.MonsterType{
		references.MonsterBat, references.MonsterMongbat, references.MonsterOrc, references.MonsterSlime,
	} {
		t.Run(kind.Name(), func(t *testing.T) {
			scene, _, step := newScheduledProjectileTestScene(t, kind, 6)
			pacing, fast := scene.fastScheduledMonsterPacing(step.Actor)
			wantFast := kind == references.MonsterBat || kind == references.MonsterMongbat
			assert.Equal(t, wantFast, fast)
			if wantFast {
				assert.Equal(t, combatMonsterPacing{anticipationMs: 75, movementMs: 175}, pacing)
			}
			// A possessed PC can share the bat's entity index, but never its profile.
			partyActor := game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: step.Actor.EntityIndex}
			scene.gameState.CombatScene.PartyPossessed[partyActor.EntityIndex] = true
			_, fast = scene.fastScheduledMonsterPacing(partyActor)
			assert.False(t, fast)
		})
	}
}

func TestFastScheduledMonsterPacingScalingAndFallback(t *testing.T) {
	scene, _, step := newScheduledProjectileTestScene(t, references.MonsterBat, 6)
	for _, test := range []struct {
		name   string
		factor float64
		want   combatMonsterPacing
	}{
		{"half", 0.5, combatMonsterPacing{150, 350}},
		{"normal", 1, combatMonsterPacing{75, 175}},
		{"double", 2, combatMonsterPacing{38, 88}},
		{"quadruple", 4, combatMonsterPacing{19, 44}},
		{"unset", 0, combatMonsterPacing{75, 175}},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene.gameConfig = &config.UltimaVConfiguration{
				GameplayConfiguration: &config.GameplayConfiguration{CombatSpeedFactor: test.factor},
			}
			pacing, fast := scene.fastScheduledMonsterPacing(step.Actor)
			require.True(t, fast)
			assert.Equal(t, test.want, pacing)
		})
	}
	for _, invalid := range []float64{-1, 0.24, 4.01, math.NaN(), math.Inf(1)} {
		scene.gameConfig.GameplayConfiguration.CombatSpeedFactor = invalid
		assert.Panics(t, func() { scene.fastScheduledMonsterPacing(step.Actor) })
	}
	for _, configuration := range []*config.UltimaVConfiguration{nil, {DebugConfiguration: nil}} {
		scene.gameConfig = configuration
		pacing, fast := scene.fastScheduledMonsterPacing(step.Actor)
		require.True(t, fast)
		assert.Equal(t, combatMonsterPacing{75, 175}, pacing)
	}
	assert.Equal(t, uint64(1), newCombatPresentationTiming(4).duration(1), "rounding retains the minimum duration")
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
		{factor: 0.5, playerDuration: 666, enemyDuration: 1300},
		{factor: 1.0, playerDuration: 333, enemyDuration: 650},
		{factor: 2.0, playerDuration: 167, enemyDuration: 325},
		{factor: 4.0, playerDuration: 83, enemyDuration: 163},
	}

	for _, test := range tests {
		timing := newCombatPresentationTiming(test.factor)
		assert.Equal(t, test.playerDuration,
			timing.duration(combatPlayerMovementDurationMs))
		assert.Equal(t, test.enemyDuration,
			timing.duration(combatEnemyMovementDurationMs))
	}
}
