package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Exercise the actual combat sprite-selection boundary with real atlas images,
// rather than only the frame-index arithmetic. ref:ENEMY-0002 keeps
// Corpsers inactive until an adjacent attack and resets them next awake turn.
func TestCorpserCombatStateSelectsHiddenExposedAndSleepingSprites(t *testing.T) {
	sheet := sharedPresentationTestSpriteSheet()
	scene := &GameScene{spriteSheet: sheet}
	random := deterministic_random.New(646)
	corpser := game_state.NewCombatMonster(references.MonsterCorpser, &references.EnemyReference{
		MonsterType:  references.MonsterCorpser,
		KeyFrameTile: &references.Tile{Index: indexes.Corpser1},
	}, references.Position{X: 5, Y: 5}, random)
	require.True(t, corpser.IsCorpserHidden(), "a newly created combat Corpser is inactive")
	beforeCalls := random.Calls()
	for _, slot := range []int{0, 2} {
		for _, elapsed := range []int64{0, 175, 350, 525} {
			assert.Same(t, sheet.GetSprite(indexes.Corpser1), scene.combatMonsterSprite(corpser, slot, elapsed),
				"initial hidden state stays static regardless of elapsed time or slot")
		}
	}

	corpser.SetCorpserExposed(true)
	require.True(t, corpser.IsCorpserExposed())
	for step, frame := range []indexes.SpriteIndex{indexes.Corpser2, indexes.Corpser3, indexes.Corpser4, indexes.Corpser2} {
		assert.Same(t, sheet.GetSprite(frame), scene.combatMonsterSprite(corpser, 0, int64(step)*175))
	}
	assert.Same(t, sheet.GetSprite(indexes.Corpser4), scene.combatMonsterSprite(corpser, 2, 0),
		"slot offset reaches the actual renderer")
	assert.NotSame(t, scene.combatMonsterSprite(corpser, 0, 0), scene.combatMonsterSprite(corpser, 0, 175))

	corpser.SetAsleep(true)
	for _, elapsed := range []int64{0, 175, 350} {
		assert.Same(t, sheet.GetSprite(indexes.Corpser1), scene.combatMonsterSprite(corpser, 0, elapsed),
			"sleep takes precedence over exposure")
	}
	corpser.SetAsleep(false)
	corpser.SetCorpserExposed(false)
	assert.Same(t, sheet.GetSprite(indexes.Corpser1), scene.combatMonsterSprite(corpser, 0, 350),
		"resetting exposure restores the base sprite")
	assert.NotSame(t, sheet.GetAnimatedSprite(indexes.Corpser1, 0, 0), sheet.GetAnimatedSprite(indexes.Corpser1, 0, 175),
		"the ordinary map animation remains independent of combat hidden state")
	assert.Equal(t, beforeCalls, random.Calls(), "presentation does not consume gameplay randomness")
}

func TestCorpserExposedSpriteCyclesOnlyAttackFrames(t *testing.T) {
	assert.Equal(t, indexes.SpriteIndex(indexes.Corpser2), corpserExposedSpriteIndex(0, 0))
	assert.Equal(t, indexes.SpriteIndex(indexes.Corpser3),
		corpserExposedSpriteIndex(0, corpserExposedFrameDurationMs))
	assert.Equal(t, indexes.SpriteIndex(indexes.Corpser4),
		corpserExposedSpriteIndex(0, 2*corpserExposedFrameDurationMs))
	assert.Equal(t, indexes.SpriteIndex(indexes.Corpser2),
		corpserExposedSpriteIndex(0, 3*corpserExposedFrameDurationMs))
}

func TestCombatConditionPlainTextDescribesCorpserLifecycle(t *testing.T) {
	assert.Equal(t, "Avatar is dragged under",
		combatConditionPlainText(game_state.CombatConditionDraggedUnder, "Avatar"))
	assert.Equal(t, "Avatar struggles underground",
		combatConditionPlainText(game_state.CombatConditionUnderground, "Avatar"))
	assert.Equal(t, "Avatar is regurgitated",
		combatConditionPlainText(game_state.CombatConditionRegurgitated, "Avatar"))
}
