package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

func TestCombatFieldDispelEffectsUseEventTargetForEveryProducer(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	origin := scene.gameState.CombatScene.GetCombatantPosition(
		scene.gameState.CombatScene.GetCurrentTurnIdentity(),
	)
	firstTarget := references.Position{X: origin.X, Y: origin.Y + 1}
	secondTarget := references.Position{X: origin.X + 1, Y: origin.Y}
	require.True(t, scene.gameState.CombatScene.AddField(
		game_state.NewCombatField(game_state.CombatFieldFire, firstTarget),
	))
	require.True(t, scene.gameState.CombatScene.AddField(
		game_state.NewCombatField(game_state.CombatFieldSleep, secondTarget),
	))
	publishEvent := scene.gameState.SystemCallbacks.Combat.PublishEvent
	publishConclusion := scene.gameState.SystemCallbacks.Combat.PublishConclusion
	scene.gameState.SystemCallbacks.Combat = game_state.NewCombatCallbacks(func(event game_state.CombatEvent) {
		publishEvent(event)
		scene.recordCombatEvent(event)
	}, publishConclusion)
	randomCallsBefore := scene.gameState.GetRNG().Calls()

	require.True(t, scene.gameState.CastCombatAnGrav(
		scene.gameState.CombatScene.GetActiveCombatantIndex(),
		references.Down,
	))
	require.True(t, scene.gameState.UseCombatSceptre(references.Right))

	require.Len(t, scene.combatFieldDispels, 2)
	assert.Equal(t, firstTarget, scene.combatFieldDispels[0].Position)
	assert.Equal(t, secondTarget, scene.combatFieldDispels[1].Position)
	assert.Equal(t, randomCallsBefore, scene.gameState.GetRNG().Calls())
}

func TestCombatSceptreEffectsExpireIndependentlyOnGameplayClock(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	scene.gameState.ElapsedMs = 100
	scene.showCombatSceptreActorEffect()
	scene.showCombatFieldDispelEffect(game_state.CombatEvent{ //nolint:exhaustruct // Focused timing fixture.
		To: references.Position{X: 4, Y: 5},
	})
	scene.gameState.ElapsedMs += 100
	scene.showCombatFieldDispelEffect(game_state.CombatEvent{ //nolint:exhaustruct // Focused timing fixture.
		To: references.Position{X: 6, Y: 5},
	})

	scene.gameState.ElapsedMs = 100 + combatFieldDispelDurationMs
	scene.updateCombatSceptreEffects()

	assert.Empty(t, scene.combatSceptreEffects)
	require.Len(t, scene.combatFieldDispels, 1)
	assert.Equal(t, references.Position{X: 6, Y: 5}, scene.combatFieldDispels[0].Position)

	scene.gameState.ElapsedMs += 100
	scene.updateCombatSceptreEffects()
	assert.Empty(t, scene.combatFieldDispels)
}

func TestCombatSceptreEffectsClearOutsideCombat(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	scene.showCombatSceptreActorEffect()
	scene.showCombatFieldDispelEffect(game_state.CombatEvent{ //nolint:exhaustruct // Focused cleanup fixture.
		To: references.Position{X: 4, Y: 5},
	})

	scene.gameState.TransitionFromCombatMap()
	scene.updateCombatSceptreEffects()

	assert.Empty(t, scene.combatSceptreEffects)
	assert.Empty(t, scene.combatFieldDispels)
}

func TestCombatSceptreCueProgression(t *testing.T) {
	assert.Zero(t, combatSceptreEffectProgress(100, 100, combatSceptreActorDurationMs))
	assert.InDelta(t, 0.5, combatSceptreEffectProgress(
		100,
		100+combatSceptreActorDurationMs/2,
		combatSceptreActorDurationMs,
	), 0.001)
	assert.Equal(t, float32(combatSceptreActorInset), combatSceptreActorCueInset(0.5))
	assert.Zero(t, combatSceptreActorCueInset(0))
	assert.Zero(t, combatFieldDispelCueInset(0))
	assert.Equal(t, float32(combatFieldDispelInset), combatFieldDispelCueInset(1))
	assert.True(t, combatSceptreEffectActive(100, 100+combatSceptreActorDurationMs-1, combatSceptreActorDurationMs))
	assert.False(t, combatSceptreEffectActive(100, 100+combatSceptreActorDurationMs, combatSceptreActorDurationMs))
}

func TestCombatSceptreCuesAlignToCombatCellInterior(t *testing.T) {
	const inset = float32(3)

	left, top, right, bottom := combatCornerCueSourceBounds(
		combatSceptreCueOrigin,
		sprites.TileSize,
		inset,
	)

	assert.Equal(t, 4.0, left)
	assert.Equal(t, 4.0, top)
	assert.Equal(t, 14.0, right)
	assert.Equal(t, 14.0, bottom)
}
