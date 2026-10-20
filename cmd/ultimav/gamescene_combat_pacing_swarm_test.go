package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func newCombatPacingSwarm(t *testing.T, kinds []references.MonsterType) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, callbacks := newCombatUseTestScene(t)
	state := scene.gameState
	combat := state.CombatScene
	combat.MonsterManager.Clear()
	combat.TurnQueue.RemoveAllMonsters()
	raw := &[references.XCombatMapTiles][references.YCombatMapTiles]byte{}
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			raw[x][y] = byte(indexes.BrickFloor)
			setCombatConcealmentTerrain(state, references.Position{
				X: references.Coordinate(x), Y: references.Coordinate(y),
			}, indexes.BrickFloor)
		}
	}
	combat.CombatMapRef.SetRawData(raw)
	for index, active := range combat.PartyActive {
		if active {
			combat.SetPartyMemberPosition(index, references.Position{X: references.Coordinate(index), Y: 10})
		}
	}
	// Two separated ranks leave room for repeated approaches without reaching
	// the party. Real randomized DEX and legal occupancy choose actor order.
	for index := range 16 {
		kind := kinds[index%len(kinds)]
		reference := (*state.GameReferences.EnemyReferences)[kind]
		combat.AddMonsterToQueue(game_state.NewCombatMonster(kind, &reference, references.Position{
			X: references.Coordinate(index % 8), Y: references.Coordinate((index / 8) * 3),
		}, state.AsRandomProvider()))
	}
	state.ActiveDurationEffect = game_state.DurationEffect{
		EffectType: game_state.DurationEffectProtection, TurnsRemaining: 8,
	}
	require.True(t, combat.AddField(game_state.NewCombatField(
		game_state.CombatFieldPoison, references.Position{X: 10, Y: 7})))
	state.SetRandomSeed(9182)
	state.ElapsedMs = 900
	callbacks.Reset()
	state.BeginCombatTurns(game_state.CombatTurnsPresented)

	return scene, callbacks
}

func nextCombatPacingSwarmStep(t *testing.T, scene *GameScene) game_state.CombatPresentationStep {
	t.Helper()
	state := scene.gameState
	for range 128 {
		if step, ok := state.ReviewScheduledCombatStep(); ok {
			require.Equal(t, game_state.CombatPresentationMonsterTurn, step.Kind)
			require.False(t, step.Actor.IsPartyMember)
			require.False(t, step.HasTarget, "approach fixture must remain movement-only")

			return step
		}
		require.False(t, state.CombatScene.HasScheduledCombatStep())
		state.FinishCombatAction()
	}
	t.Fatal("swarm did not reach its next scheduled action")

	return game_state.CombatPresentationStep{}
}

func completeCombatPacingSwarmMove(
	t *testing.T, scene *GameScene, callbacks *game_state.MockSystemCallbacks,
	step game_state.CombatPresentationStep, baseline bool,
) int64 {
	t.Helper()
	startMs := scene.gameState.ElapsedMs
	anticipationMs, movementMs := int64(75), int64(175)
	if baseline {
		anticipationMs, movementMs = 250, 650
		// Inject the former 1x profile into the production workflow. This changes
		// presentation only: the actor remains a bat/mongbat for every AI stage.
		scene.startScheduledMonsterTurnWithPacing(step, combatMonsterPacing{250, 650}, true)
	} else {
		scene.startScheduledMonsterTurn(step)
	}
	drainCombatPacingAt(scene, startMs)
	drainCombatPacingAt(scene, startMs+anticipationMs)
	require.True(t, scene.combatMovement.Active, "each measured action must really move")
	drainCombatPacingAt(scene, startMs+anticipationMs+movementMs-1)
	require.True(t, callbacks.InputBlocked)
	drainCombatPacingAt(scene, startMs+anticipationMs+movementMs)
	require.False(t, callbacks.InputBlocked)
	require.False(t, scene.gameState.EventQueueManager.HasWorkflow("monster_turn_delay"))
	require.ErrorIs(t, scene.gameState.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)

	return scene.gameState.ElapsedMs - startMs
}

func assertCombatPacingBoundaryEquivalent(t *testing.T, baseline, accelerated *GameScene) {
	t.Helper()
	left, right := baseline.gameState, accelerated.gameState
	leftSnapshot := snapshotBlockedCombatState(t, left)
	rightSnapshot := snapshotBlockedCombatState(t, right)
	leftSnapshot.ElapsedMs, rightSnapshot.ElapsedMs = 0, 0
	assert.Equal(t, leftSnapshot, rightSnapshot, "presentation clock is the sole permitted state difference")
	assert.Equal(t, left.ActiveDurationEffect, right.ActiveDurationEffect)
	assert.Equal(t, left.CombatScene.PartyUnderground, right.CombatScene.PartyUnderground)
	for index := range left.PartyState.GetCharacterCount() {
		assert.Equal(t, *left.PartyState.GetCharacter(index), *right.PartyState.GetCharacter(index))
	}
	left.CombatScene.MonsterManager.ForEach(func(index int, monster *game_state.CombatMonster) {
		other := right.CombatScene.MonsterManager.GetMonster(index)
		require.NotNil(t, other)
		assert.Equal(t, monster.IsAsleep(), other.IsAsleep())
		assert.Equal(t, monster.IsInvisible(), other.IsInvisible())
		assert.Equal(t, monster.IsFleeing(), other.IsFleeing())
		assert.Equal(t, monster.Loyalty(), other.Loyalty())
	})
	leftForecast := make([]game_state.CombatTurnForecastEntry, 32)
	rightForecast := make([]game_state.CombatTurnForecastEntry, 32)
	assert.Equal(t, left.CombatScene.TurnQueue.FillForecast(leftForecast), right.CombatScene.TurnQueue.FillForecast(rightForecast))
	assert.Equal(t, leftForecast, rightForecast)
	leftRNG, rightRNG := left.GetRNG(), right.GetRNG()
	assert.Equal(t, leftRNG.Calls(), rightRNG.Calls())
	assert.Equal(t,
		beradrng.Restore(leftRNG.Seed(), leftRNG.Calls()).Intn(10000),
		beradrng.Restore(rightRNG.Seed(), rightRNG.Calls()).Intn(10000),
		"peek the next value without perturbing either gameplay stream")
}

func TestScheduledBatPacingSeededSwarmEquivalenceAndElapsedBudget(t *testing.T) {
	for _, test := range []struct {
		name  string
		kinds []references.MonsterType
	}{
		{"bats", []references.MonsterType{references.MonsterBat}},
		{"mongbats", []references.MonsterType{references.MonsterMongbat}},
		{"mixed", []references.MonsterType{references.MonsterBat, references.MonsterMongbat}},
	} {
		t.Run(test.name, func(t *testing.T) {
			baseline, baselineCallbacks := newCombatPacingSwarm(t, test.kinds)
			accelerated, acceleratedCallbacks := newCombatPacingSwarm(t, test.kinds)
			var baselineMs, acceleratedMs int64
			actors := make(map[game_state.CombatTurnIdentity]int)
			for action := range 32 {
				baselineStep := nextCombatPacingSwarmStep(t, baseline)
				acceleratedStep := nextCombatPacingSwarmStep(t, accelerated)
				require.Equal(t, baselineStep.Actor, acceleratedStep.Actor, "action=%d", action)
				actors[baselineStep.Actor]++
				baselineMs += completeCombatPacingSwarmMove(t, baseline, baselineCallbacks, baselineStep, true)
				acceleratedMs += completeCombatPacingSwarmMove(t, accelerated, acceleratedCallbacks, acceleratedStep, false)
				assertCombatPacingBoundaryEquivalent(t, baseline, accelerated)
				assert.Equal(t, baselineCallbacks.CombatEvents, acceleratedCallbacks.CombatEvents)
				if action == 15 {
					require.Len(t, actors, 16)
					assert.Equal(t, int64(14400), baselineMs)
					assert.Equal(t, int64(4000), acceleratedMs)
					t.Logf("seed=9182 actors=16 actions=16 factor=1 injected-clock baseline=%dms accelerated=%dms reduction=72.22%%", baselineMs, acceleratedMs)
				}
			}
			repeatedActors := 0
			for _, count := range actors {
				if count > 1 {
					repeatedActors++
				}
			}
			assert.Positive(t, repeatedActors, "the second batch must exercise repeated actor travel")
			assert.Equal(t, int64(28800), baselineMs)
			assert.Equal(t, int64(8000), acceleratedMs)
			assert.LessOrEqual(t, acceleratedMs*2, baselineMs, "at least 50%% less injected presentation time")
			t.Logf("seed=9182 actors=16 actions=32 factor=1 injected-clock baseline=%dms accelerated=%dms; manual readability acceptance remains required", baselineMs, acceleratedMs)
		})
	}
}
