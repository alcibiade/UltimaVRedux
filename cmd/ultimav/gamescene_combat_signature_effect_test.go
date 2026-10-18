package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func signatureMonster(t *testing.T, scene *GameScene, kind references.MonsterType, position references.Position) (int, *game_state.CombatMonster) {
	t.Helper()
	state := scene.gameState
	ref := &(*state.GameReferences.EnemyReferences)[kind]
	monster := game_state.NewCombatMonster(kind, ref, position, state.AsRandomProvider())
	index := state.CombatScene.AddMonsterToQueue(monster)
	setCombatConcealmentTerrain(state, position, indexes.BrickFloor)

	return index, monster
}

func signatureGateEvent(index int, monster *game_state.CombatMonster) game_state.CombatEvent {
	return game_state.CombatEvent{Kind: game_state.CombatEventGating,
		Created: game_state.CombatSummonedMonster{CombatantID: monster.CombatantID(), Index: index,
			MonsterType: monster.MonsterType(), Position: monster.Position()}}
}

func TestCombatSignaturePublishEventRoutesSuccessCuesWithoutMutation(t *testing.T) {
	scene, mock := newCombatUseTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	state := scene.gameState
	state.SystemCallbacks.Combat = game_state.NewCombatCallbacks(scene.recordCombatEvent, scene.recordCombatConclusion)
	index, monster := signatureMonster(t, scene, references.MonsterDevil, references.Position{X: 5, Y: 5})
	actor := state.CombatScene.GetCurrentTurnIdentity()
	position := state.CombatScene.GetCombatantPosition(actor)
	state.CombatScene.PartyPossessed[actor.EntityIndex] = true
	target := game_state.CombatFeedbackEntity{IsPartyMember: true, Index: actor.EntityIndex, Slot: actor.Slot, Position: position}
	cases := []struct {
		event game_state.CombatEvent
		sound game_state.SoundEffect
	}{
		{signatureGateEvent(index, monster), game_state.SoundMonsterGate},
		{game_state.CombatEvent{Kind: game_state.CombatEventPossession, Target: target, From: monster.Position(), To: position}, game_state.SoundMonsterPossession},
		{game_state.CombatEvent{Kind: game_state.CombatEventTeleport, Attacker: game_state.CombatFeedbackEntity{Index: index},
			From: references.Position{X: 4, Y: 5}, To: monster.Position()}, game_state.SoundMonsterTeleport},
		{game_state.CombatEvent{Kind: game_state.CombatEventCondition, Condition: game_state.CombatConditionDraggedUnder, Target: target}, game_state.SoundCorpserCapture},
		{game_state.CombatEvent{Kind: game_state.CombatEventCondition, Condition: game_state.CombatConditionRegurgitated, Target: target}, game_state.SoundCorpserRelease},
	}
	for _, test := range cases {
		state.CombatScene.PartyUnderground[actor.EntityIndex] = test.event.Kind == game_state.CombatEventCondition &&
			test.event.Condition == game_state.CombatConditionDraggedUnder
		before := snapshotBlockedCombatState(t, state)
		mock.SoundEffectsPlayed = nil
		state.SystemCallbacks.Combat.PublishEvent(test.event)
		require.Len(t, scene.combatSignatureEffects, 1)
		if test.event.Kind == game_state.CombatEventGating {
			assert.Empty(t, mock.SoundEffectsPlayed, "arrival prelude is silent")
			advanceSignatureForTest(scene, state.ElapsedMs+combatGateArrivalMs)
		}
		assert.Equal(t, []game_state.SoundEffect{test.sound}, mock.SoundEffectsPlayed)
		assert.NotContains(t, mock.SoundEffectsPlayed, game_state.SoundSpellFailure)
		assert.False(t, state.EventQueueManager.HasActiveWorkflows(), "event effects cannot claim input")
		effect := scene.combatSignatureEffects[0]
		assert.Equal(t, state.CombatScene, effect.Encounter)
		advanceSignatureForTest(scene, state.ElapsedMs+effect.DurationMs/2)
		require.Len(t, scene.combatSignatureEffects, 1, "show the complete effect")
		before.ElapsedMs = state.ElapsedMs // The test advances only the shared presentation clock.
		assert.Equal(t, before, snapshotBlockedCombatState(t, state))
		advanceSignatureForTest(scene, state.ElapsedMs+effect.DurationMs/2)
		assert.Empty(t, scene.combatSignatureEffects)
		assert.False(t, effect.Dissolve.IsActive())
		assert.Nil(t, effect.Frame)
		assert.Nil(t, effect.BodyFrame)
		assert.Nil(t, effect.HealthFrame)
	}
	state.SystemCallbacks.Combat.PublishEvent(game_state.CombatEvent{Kind: game_state.CombatEventCondition,
		Condition: game_state.CombatConditionUnderground, Target: target})
	assert.Empty(t, scene.combatSignatureEffects, "routine underground turns must not replay capture")
}

func TestCombatSignatureConcurrentEffectsCompactionAndSlotReuse(t *testing.T) {
	scene, _ := newCombatUseTestScene(t)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	index, monster := signatureMonster(t, scene, references.MonsterDevil, references.Position{X: 5, Y: 5})
	scene.recordCombatEvent(signatureGateEvent(index, monster))
	first := scene.combatSignatureEffects[0]
	advanceSignatureForTest(scene, scene.gameState.ElapsedMs+combatGateDurationMs/2)
	secondIndex, secondMonster := signatureMonster(t, scene, references.MonsterDevil, references.Position{X: 6, Y: 5})
	scene.recordCombatEvent(signatureGateEvent(secondIndex, secondMonster))
	require.Len(t, scene.combatSignatureEffects, 2)
	second := scene.combatSignatureEffects[1]
	advanceSignatureForTest(scene, scene.gameState.ElapsedMs+combatGateDurationMs/2)
	require.Len(t, scene.combatSignatureEffects, 1)
	require.Same(t, second, scene.combatSignatureEffects[0])
	assert.False(t, first.Dissolve.IsActive())
	assert.Nil(t, first.Frame)
	assert.Nil(t, first.BodyFrame)
	assert.Nil(t, first.HealthFrame)
	assert.True(t, second.Dissolve.IsActive())
	assert.Nil(t, scene.combatSignatureEffects[:cap(scene.combatSignatureEffects)][1])
	assert.True(t, scene.combatSignatureHidesActor(game_state.CombatTurnIdentity{EntityIndex: secondIndex}))
	// Replace in the same live manager slot: the previous incarnation cannot hide it.
	scene.gameState.CombatScene.ReplaceCombatantWithMonster(game_state.CombatTurnIdentity{EntityIndex: secondIndex},
		game_state.NewCombatMonster(references.MonsterRat, &(*scene.gameState.GameReferences.EnemyReferences)[references.MonsterRat],
			secondMonster.Position(), scene.gameState.AsRandomProvider()))
	scene.gameState.CombatScene.MonsterManager.RemoveMonster(secondIndex)
	assert.False(t, scene.combatSignatureHidesActor(game_state.CombatTurnIdentity{EntityIndex: secondIndex}))
	scene.updateCombatSignatureEffects()
	assert.Empty(t, scene.combatSignatureEffects)
	assert.False(t, second.Dissolve.IsActive())
	assert.Nil(t, second.Frame)
	assert.Nil(t, second.BodyFrame)
	assert.Nil(t, second.HealthFrame)
}

// Regular effect ticks, deliberately separate from tests that simulate a hitch.
func advanceSignatureForTest(scene *GameScene, untilMs int64) {
	for scene.gameState.ElapsedMs < untilMs {
		scene.gameState.ElapsedMs = min(scene.gameState.ElapsedMs+25, untilMs)
		scene.updateCombatSignatureEffects()
	}
}

func TestCombatSignatureCleansUpOnExitReplacementAndDispose(t *testing.T) {
	for _, path := range []string{"exit", "replacement", "dispose"} {
		t.Run(path, func(t *testing.T) {
			scene, _ := newCombatUseTestScene(t)
			scene.spriteSheet = sharedPresentationTestSpriteSheet()
			index, monster := signatureMonster(t, scene, references.MonsterDevil, references.Position{X: 5, Y: 5})
			scene.recordCombatEvent(signatureGateEvent(index, monster))
			effect := scene.combatSignatureEffects[0]
			switch path {
			case "exit":
				scene.gameState.TransitionFromCombatMap()
			case "replacement":
				copyOfCombat := *scene.gameState.CombatScene
				scene.gameState.CombatScene = &copyOfCombat
			case "dispose":
				scene.Dispose()
			}
			scene.updateCombatSignatureEffects()
			assert.Empty(t, scene.combatSignatureEffects)
			assert.False(t, effect.Dissolve.IsActive())
			assert.Nil(t, effect.Frame)
			assert.Nil(t, effect.BodyFrame)
			assert.Nil(t, effect.HealthFrame)
		})
	}
}

func TestCombatSignatureTeleportPhaseBoundaries(t *testing.T) {
	for _, test := range []struct {
		elapsedMs int64
		phase     combatTeleportPhase
		progress  float32
	}{
		{0, combatTeleportDeparture, 0},
		{90, combatTeleportDeparture, 0.5},
		{180, combatTeleportHidden, 0},
		{260, combatTeleportArrival, 0},
		{360, combatTeleportArrival, 0.5},
		{460, combatTeleportArrival, 1},
	} {
		phase, progress := (&combatSignatureEffect{}).blink(test.elapsedMs)
		assert.Equal(t, test.phase, phase)
		assert.InDelta(t, test.progress, progress, 0.001)
	}
}

func TestCombatSignatureProductionDrawingRespectsTerrainAndInvisibility(t *testing.T) {
	scene, screen := newCombatConcealmentTestScene(t)
	index, monster := signatureMonster(t, scene, references.MonsterDevil, references.Position{X: 5, Y: 5})
	scene.recordCombatEvent(signatureGateEvent(index, monster))
	scene.gameState.ElapsedMs += combatGateDurationMs / 2
	scene.updateCombatSignatureEffects()
	scene.drawCombatSignatureEffects(screen)
	assert.Positive(t, opaquePixelCount(screen))
	screen.Clear()
	monster.SetInvisible(true)
	scene.drawCombatSignatureEffects(screen)
	assert.Zero(t, opaquePixelCount(screen))
	monster.SetInvisible(false)
	setCombatConcealmentTerrain(scene.gameState, monster.Position(), indexes.BlackSquare)
	scene.drawCombatSignatureEffects(screen)
	assert.Zero(t, opaquePixelCount(screen))
	scene.clearCombatSignatureEffects()
}
