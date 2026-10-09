package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestScheduledEttinProjectileStartsAtLaunchTime(t *testing.T) {
	scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterEttin, 4)
	screen := newProjectileRenderTarget(t, scene, 1280, 720, indexes.Grass)
	state := scene.gameState
	state.SetRandomSeed(projectileHitSeed(t))
	state.ElapsedMs = 0
	scene.startScheduledMonsterTurn(step)
	state.EventQueueManager.Update()
	require.Nil(t, scene.activeProjectile)
	hpBefore := state.PartyState.GetCharacter(0).GetCurrentHp()

	// Ordinary ranged decisions use the 250ms seek delay. The old queue charged
	// that pre-launch interval to the new flight; with the old ordinary speed,
	// the 196ms stone flight reached its endpoint before its first draw.
	state.ElapsedMs = game_state.MonsterMoveDelayMs
	state.EventQueueManager.Update()
	require.NotNil(t, scene.activeProjectile)
	flight := scene.activeProjectile
	require.Equal(t, references.MissileRock, flight.projectileType)
	require.Equal(t, flight.fromTileX, flight.currentTileX)
	require.Equal(t, flight.fromTileY, flight.currentTileY)
	require.NotEqual(t, flight.fromTileX, flight.toTileX)
	assert.Equal(t, hpBefore, state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Empty(t, callbacks.CombatEvents)
	assert.True(t, callbacks.InputBlocked)
	rngBefore := state.GetRNG().Calls()
	const durationMs int64 = 320
	for _, age := range []int64{0, 80, 160, 240, durationMs} {
		state.ElapsedMs = game_state.MonsterMoveDelayMs + age
		state.EventQueueManager.Update()
		require.Same(t, flight, scene.activeProjectile)
		assert.InDelta(t, flight.fromTileX+(flight.toTileX-flight.fromTileX)*float64(age)/float64(durationMs), flight.currentTileX, 0.00001)
		assertProjectileDrawAtCurrentPosition(t, scene, screen)
		assert.Equal(t, hpBefore, state.PartyState.GetCharacter(0).GetCurrentHp(), "impact follows the endpoint frame")
		assert.Equal(t, rngBefore, state.GetRNG().Calls(), "flight/draw consumes no gameplay RNG")
		assert.Equal(t, step.Actor, state.CombatScene.GetCurrentTurnIdentity())
		assert.Empty(t, callbacks.CombatEvents)
	}
	for range 8 {
		state.EventQueueManager.Update()
	}
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, game_state.CombatEventRangedAttack, callbacks.CombatEvents[0].Kind)
	assert.Equal(t, references.Position{X: 6, Y: 3}, callbacks.CombatEvents[0].To)
	assert.Equal(t, uint16(15), hpBefore-state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Len(t, callbacks.CombatImpactCalls, 1)
	assert.Nil(t, scene.activeProjectile)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, state.EventQueueManager.HasWorkflow("monster_turn_delay"))
	assert.Equal(t, rngBefore, state.GetRNG().Calls(), "unarmored rock impact and completion consume no additional RNG")
	after := snapshotBlockedCombatState(t, state)
	assert.False(t, state.ResolvePendingMonsterMissile())
	assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
	assert.Equal(t, after, snapshotBlockedCombatState(t, state))
}

func TestScheduledProjectileCancellationResolvesExactlyOnce(t *testing.T) {
	for _, age := range []int64{0, 160, 320} {
		t.Run(fmt.Sprintf("age%d", age), func(t *testing.T) {
			scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterEttin, 4)
			state := scene.gameState
			state.SetRandomSeed(projectileHitSeed(t))
			state.ElapsedMs = 0
			scene.startScheduledMonsterTurn(step)
			state.EventQueueManager.Update()
			state.ElapsedMs = game_state.MonsterMoveDelayMs
			state.EventQueueManager.Update()
			require.NotNil(t, scene.activeProjectile)
			state.ElapsedMs += age
			state.EventQueueManager.Update()
			rngBefore := state.GetRNG().Calls()
			state.EventQueueManager.Shutdown()
			require.Nil(t, scene.activeProjectile)
			require.Len(t, callbacks.CombatEvents, 1)
			assert.Len(t, callbacks.CombatImpactCalls, 1)
			assert.False(t, callbacks.InputBlocked)
			assert.False(t, state.ResolvePendingMonsterMissile())
			assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
			assert.Equal(t, rngBefore, state.GetRNG().Calls())
			after := snapshotBlockedCombatState(t, state)
			state.EventQueueManager.Shutdown()
			state.EventQueueManager.Update()
			assert.Equal(t, after, snapshotBlockedCombatState(t, state))
			assert.Len(t, callbacks.CombatEvents, 1)
		})
	}
}

func newScheduledProjectileTestScene(
	t *testing.T, kind references.MonsterType, distance int,
) (*GameScene, *game_state.MockSystemCallbacks, game_state.CombatPresentationStep) {
	t.Helper()
	scene, callbacks := newCombatUseTestScene(t)
	state := scene.gameState
	state.CombatScene.MonsterManager.Clear()
	state.CombatScene.TurnQueue.RemoveAllMonsters()
	raw := &[references.XCombatMapTiles][references.YCombatMapTiles]byte{}
	for x := range references.XCombatMapTiles {
		for y := range references.YCombatMapTiles {
			raw[x][y] = byte(indexes.BrickFloor)
			setCombatConcealmentTerrain(state, references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}, indexes.BrickFloor)
		}
	}
	state.CombatScene.CombatMapRef.SetRawData(raw)
	for index, active := range state.CombatScene.PartyActive {
		if active {
			state.CombatScene.SetPartyMemberPosition(index, references.Position{X: references.Coordinate(index), Y: 10})
		}
	}
	state.CombatScene.SetPartyMemberPosition(0, references.Position{X: references.Coordinate(2 + distance), Y: 3})
	character := state.PartyState.GetCharacter(0)
	for _, slot := range []references.EquipmentSlot{references.SlotArmour, references.SlotShield, references.SlotHelm, references.SlotAmulet} {
		character.SetEquippedItem(slot, references.NoEquipment)
	}
	reference := (*state.GameReferences.EnemyReferences)[kind]
	monster := game_state.NewCombatMonster(kind, &reference, references.Position{X: 2, Y: 3}, state.AsRandomProvider())
	index := state.CombatScene.AddMonsterToQueue(monster)
	step := nextSignatureMonsterStep(t, scene, index)
	// Ranged decisions are deferred until execution, so this ordinary token
	// uses the seek anticipation delay rather than advertising a certain hit.
	require.False(t, step.HasTarget)
	state.SystemCallbacks.Visual.StartProjectile = scene.startProjectile
	state.SystemCallbacks.Visual.UpdateProjectileProgress = scene.updateProjectileProgress
	state.SystemCallbacks.Visual.ClearProjectile = scene.clearProjectile
	callbacks.Reset()

	return scene, callbacks, step
}

func TestScheduledProjectileResolvedEndpoints(t *testing.T) {
	for _, test := range []struct {
		name          string
		distance      int
		miss, blocked bool
		endpoint      references.Position
	}{
		{name: "short legal shot", distance: 2, endpoint: references.Position{X: 4, Y: 3}},
		{name: "scatter", distance: 4, miss: true, endpoint: references.Position{X: 7, Y: 3}},
		{name: "obstruction", distance: 4, blocked: true, endpoint: references.Position{X: 4, Y: 3}},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterEttin, test.distance)
			screen := newProjectileRenderTarget(t, scene, 1280, 720, indexes.BrickFloor)
			state := scene.gameState
			seed := projectileHitSeed(t)
			if test.miss {
				seed = projectileScatterSeed(t)
			}
			if test.blocked {
				state.CombatScene.CombatMapRef.SetTileNumber(test.endpoint, indexes.StoneBrickWall)
				setCombatConcealmentTerrain(state, test.endpoint, indexes.StoneBrickWall)
			}
			state.SetRandomSeed(seed)
			state.ElapsedMs = 0
			scene.startScheduledMonsterTurn(step)
			state.EventQueueManager.Update()
			state.ElapsedMs = game_state.MonsterMoveDelayMs
			state.EventQueueManager.Update()
			require.NotNil(t, scene.activeProjectile)
			flight := scene.activeProjectile
			require.Equal(t, float64(combatViewportStartColumn)+float64(test.endpoint.X), flight.toTileX)
			require.Equal(t, float64(combatViewportStartRow)+float64(test.endpoint.Y), flight.toTileY)
			rngBefore := state.GetRNG().Calls()
			hpBefore := state.PartyState.GetCharacter(0).GetCurrentHp()
			// Eight pixels per 40ms: 80ms per horizontal tile for a rock.
			duration := int64(test.endpoint.X-2) * 80
			for _, age := range []int64{0, duration / 2, duration} {
				state.ElapsedMs = game_state.MonsterMoveDelayMs + age
				state.EventQueueManager.Update()
				require.Same(t, flight, scene.activeProjectile)
				assertProjectileDrawAtCurrentPosition(t, scene, screen)
				assert.Empty(t, callbacks.CombatEvents)
				assert.Equal(t, hpBefore, state.PartyState.GetCharacter(0).GetCurrentHp())
			}
			for range 8 {
				state.EventQueueManager.Update()
			}
			require.Len(t, callbacks.CombatEvents, 1)
			assert.Equal(t, test.endpoint, callbacks.CombatEvents[0].To)
			assert.Equal(t, test.blocked, callbacks.CombatEvents[0].Diagnostics.ProjectileBlocked)
			if test.miss || test.blocked {
				assert.Equal(t, hpBefore, state.PartyState.GetCharacter(0).GetCurrentHp())
			} else {
				assert.Equal(t, uint16(15), hpBefore-state.PartyState.GetCharacter(0).GetCurrentHp())
			}
			assert.Nil(t, scene.activeProjectile)
			assert.False(t, callbacks.InputBlocked)
			assert.Equal(t, rngBefore, state.GetRNG().Calls())
			assert.False(t, state.ResolvePendingMonsterMissile())
			assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
		})
	}
}

func TestScheduledProjectileEncounterExitDiscardsPendingImpact(t *testing.T) {
	scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterEttin, 4)
	state := scene.gameState
	state.SetRandomSeed(projectileHitSeed(t))
	state.ElapsedMs = 0
	scene.startScheduledMonsterTurn(step)
	state.EventQueueManager.Update()
	state.ElapsedMs = game_state.MonsterMoveDelayMs
	state.EventQueueManager.Update()
	require.NotNil(t, scene.activeProjectile)
	hpBefore := state.PartyState.GetCharacter(0).GetCurrentHp()
	state.EndCombat(game_state.CombatEvacuated)
	state.EventQueueManager.Shutdown()
	state.EventQueueManager.Update()
	assert.Nil(t, scene.activeProjectile)
	assert.False(t, callbacks.InputBlocked)
	assert.Empty(t, callbacks.CombatImpactCalls)
	assert.Equal(t, hpBefore, state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.False(t, state.ResolvePendingMonsterMissile())
}

func projectileScatterSeed(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		rng := deterministic_random.New(seed)
		if rng.Intn(256) < 128 && rng.Intn(61) == 0 && rng.Intn(3) == 2 && rng.Intn(3) == 1 {
			return seed
		}
	}
	t.Fatal("no deterministic eastward missile scatter seed")
	return 0
}

func projectileHitSeed(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		rng := deterministic_random.New(seed)
		if rng.Intn(256) < 128 && rng.Intn(61) >= 58 {
			return seed
		}
	}
	t.Fatal("no deterministic missile-hit seed")

	return 0
}
