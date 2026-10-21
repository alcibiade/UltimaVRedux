package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatVanishEffectUsesTypedEventWithoutGameplayRNG(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	event := combatVanishTestEvent(t, scene, references.Position{X: 4, Y: 5})
	randomCallsBefore := scene.gameState.GetRNG().Calls()

	scene.recordCombatEvent(event)

	require.Len(t, scene.combatVanishEffects, 1)
	effect := &scene.combatVanishEffects[0]
	assert.Equal(t, event.Vanished.CombatantID, effect.CombatantID)
	assert.Equal(t, event.Vanished.Index, effect.Index)
	assert.Equal(t, event.Vanished.Position, effect.Position)
	assert.True(t, effect.Dissolve.IsActive())
	assert.Equal(t, randomCallsBefore, scene.gameState.GetRNG().Calls())

	scene.gameState.ElapsedMs += combatVanishDurationMs / 2
	scene.updateCombatVanishEffects(0)
	assert.InDelta(t, 0.5, effect.Dissolve.GetLevel(), 0.001)
	scene.gameState.ElapsedMs += combatVanishDurationMs / 2
	scene.updateCombatVanishEffects(0)
	assert.Empty(t, scene.combatVanishEffects)
	assert.False(t, effect.Dissolve.IsActive())
}

func TestCombatVanishEffectsRemainConcurrentAndClearWhenCombatEnds(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	first := combatVanishTestEvent(t, scene, references.Position{X: 4, Y: 5})
	second := combatVanishTestEvent(t, scene, references.Position{X: 6, Y: 5})

	scene.recordCombatEvent(first)
	scene.recordCombatEvent(second)

	require.Len(t, scene.combatVanishEffects, 2)
	assert.NotEqual(t, combatVanishSeed(first.Vanished), combatVanishSeed(second.Vanished))
	firstDissolve := &scene.combatVanishEffects[0].Dissolve
	secondDissolve := &scene.combatVanishEffects[1].Dissolve
	scene.gameState.TransitionFromCombatMap()
	scene.updateCombatVanishEffects(0)

	assert.Empty(t, scene.combatVanishEffects)
	assert.False(t, firstDissolve.IsActive())
	assert.False(t, secondDissolve.IsActive())
}

func TestCombatVanishEffectCompactionClearsStaleTailResources(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	first := combatVanishTestEvent(t, scene, references.Position{X: 4, Y: 5})
	second := combatVanishTestEvent(t, scene, references.Position{X: 6, Y: 5})

	scene.recordCombatEvent(first)
	scene.gameState.ElapsedMs += combatVanishDurationMs / 2
	scene.updateCombatVanishEffects(0)
	scene.recordCombatEvent(second)
	require.Len(t, scene.combatVanishEffects, 2)

	scene.gameState.ElapsedMs += combatVanishDurationMs / 2
	scene.updateCombatVanishEffects(0)

	require.Len(t, scene.combatVanishEffects, 1)
	assert.Equal(t, second.Vanished.CombatantID, scene.combatVanishEffects[0].CombatantID)
	assert.True(t, scene.combatVanishEffects[0].Dissolve.IsActive())
	backing := scene.combatVanishEffects[:cap(scene.combatVanishEffects)]
	assert.Zero(t, backing[1])
}

func TestCombatVanishEffectCleansUpOnSceneDisposal(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.recordCombatEvent(combatVanishTestEvent(t, scene, references.Position{X: 4, Y: 5}))
	require.Len(t, scene.combatVanishEffects, 1)
	dissolve := &scene.combatVanishEffects[0].Dissolve

	scene.Dispose()

	assert.Empty(t, scene.combatVanishEffects)
	assert.False(t, dissolve.IsActive())
}

func combatVanishTestEvent(
	t *testing.T,
	scene *GameScene,
	position references.Position,
) game_state.CombatEvent {
	t.Helper()
	index, monster := signatureMonster(t, scene, references.MonsterShadowlord, position)
	monster.SetCurrentHP(0) // Publication follows authoritative death, not invisibility.
	vanished := game_state.CombatVanishedMonster{
		CombatantID: monster.CombatantID(),
		Index:       index,
		MonsterType: references.MonsterShadowlord,
		Position:    position,
	}

	return game_state.CombatEvent{
		Kind:     game_state.CombatEventVanished,
		Attacker: game_state.CombatFeedbackEntity{Index: index, Position: position},
		From:     position,
		To:       position,
		Vanished: vanished,
	}
}
