package main

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// Drain callback, deferred-sequence, animation and finalizer stages without
// charging the fixture any frame time. Elapsed values measure presentation
// budgets on the injected clock, never CPU/wall time or interactive readability.
func drainCombatPacingAt(scene *GameScene, elapsedMs int64) {
	scene.gameState.ElapsedMs = elapsedMs
	for range 16 {
		scene.gameState.EventQueueManager.Update()
	}
}

func TestScheduledBatPacingFixtureHeldKeyRelease(t *testing.T) {
	for _, polluted := range []bool{true, false} {
		t.Run(fmt.Sprintf("foreignShift=%t", polluted), func(t *testing.T) {
			scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterBat, 6)
			state := scene.gameState
			t.Cleanup(state.EventQueueManager.Shutdown)
			held := true
			if polluted {
				// Reproduce host contamination without reading or changing OS input.
				scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool {
					return held && key == ebiten.KeyShiftLeft
				})
			}
			state.SetRandomSeed(42)
			state.ElapsedMs = 900
			scene.startScheduledMonsterTurn(step)
			drainCombatPacingAt(scene, 900)
			drainCombatPacingAt(scene, 975)
			require.True(t, scene.combatMovement.Active)
			drainCombatPacingAt(scene, 1150)
			require.False(t, scene.combatMovement.Active)
			require.Equal(t, polluted, callbacks.InputBlocked)
			require.Equal(t, polluted, state.EventQueueManager.HasWorkflow("monster_turn_delay"))
			require.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
			if polluted {
				require.Zero(t, callbacks.UnblockInputCalls, "the real workflow release predicate must retain the hold")
			}
			beforeRelease := snapshotBlockedCombatState(t, state)
			held = false
			drainCombatPacingAt(scene, 1150)
			require.False(t, callbacks.InputBlocked)
			require.False(t, state.EventQueueManager.HasWorkflow("monster_turn_delay"))
			require.Equal(t, 1, callbacks.UnblockInputCalls)
			require.Equal(t, beforeRelease, snapshotBlockedCombatState(t, state), "release must not resolve another action")
			scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
			require.True(t, scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter))
			require.False(t, scene.keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter))
		})
	}
}

func TestScheduledBatPacingProductionQueueBoundaries(t *testing.T) {
	for _, test := range []struct {
		name        string
		kind        references.MonsterType
		factor      float64
		anticipate  int64
		move        int64
		changeSpeed bool
		fleeing     bool
	}{
		{"bat quarter", references.MonsterBat, 0.25, 300, 700, false, false},
		{"bat normal", references.MonsterBat, 1, 75, 175, false, false},
		{"mongbat double", references.MonsterMongbat, 2, 38, 88, false, false},
		{"mongbat quadruple", references.MonsterMongbat, 4, 19, 44, false, false},
		{"capture before first queue update", references.MonsterBat, 1, 75, 175, true, false},
		{"ordinary", references.MonsterOrc, 1, 250, 650, false, false},
		{"slime", references.MonsterSlime, 1, 250, 650, false, false},
		{"fleeing bat", references.MonsterBat, 1, 75, 175, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene, callbacks, step := newScheduledProjectileTestScene(t, test.kind, 6)
			state := scene.gameState
			state.CombatScene.MonsterManager.GetMonster(step.Actor.EntityIndex).SetFleeing(test.fleeing)
			state.SetRandomSeed(42)
			scene.gameConfig = &config.UltimaVConfiguration{
				DebugConfiguration: &config.DebugConfiguration{CombatMovementSpeedFactor: test.factor},
			}
			const startMs int64 = 900
			state.ElapsedMs = startMs
			before := snapshotBlockedCombatState(t, state)
			from := state.CombatScene.GetCombatantPosition(step.Actor)
			scene.startScheduledMonsterTurn(step)
			if test.changeSpeed {
				require.True(t, scene.gameConfig.DebugConfiguration.SetCombatMovementSpeedFactor(4))
			}
			for _, age := range []int64{0, test.anticipate / 2, test.anticipate - 1} {
				drainCombatPacingAt(scene, startMs+age)
				current := snapshotBlockedCombatState(t, state)
				current.ElapsedMs = before.ElapsedMs
				assert.Equal(t, before, current, "anticipation cannot resolve AI or consume RNG")
				require.True(t, callbacks.InputBlocked)
				require.False(t, scene.combatMovement.Active)
			}
			drainCombatPacingAt(scene, startMs+test.anticipate)
			require.True(t, scene.combatMovement.Active)
			to := state.CombatScene.GetCombatantPosition(step.Actor)
			require.NotEqual(t, from, to)
			assert.Equal(t, from, scene.combatMovement.From)
			assert.Equal(t, to, scene.combatMovement.To)
			assert.Zero(t, scene.combatMovement.Progress)
			require.True(t, state.CombatScene.HasScheduledCombatStep())
			_, err := state.ResolveScheduledCombatStep(step.Token)
			require.ErrorIs(t, err, game_state.ErrCombatPresentationDuplicate)
			rngAfterMove := state.GetRNG().Calls()
			actorAfterMove, found := state.CombatScene.TurnQueue.GetEntrySnapshot(step.Actor)
			require.True(t, found)
			for _, age := range []int64{test.move / 2, test.move - 1} {
				drainCombatPacingAt(scene, startMs+test.anticipate+age)
				require.True(t, callbacks.InputBlocked)
				require.True(t, scene.combatMovement.Active)
				assert.InDelta(t, float64(age)/float64(test.move), scene.combatMovement.Progress, 0.00001)
				assert.Equal(t, to, state.CombatScene.GetCombatantPosition(step.Actor))
				assert.Equal(t, step.Actor, state.CombatScene.GetCurrentTurnIdentity())
				actorNow, exists := state.CombatScene.TurnQueue.GetEntrySnapshot(step.Actor)
				require.True(t, exists)
				assert.Equal(t, actorAfterMove, actorNow)
				assert.Equal(t, rngAfterMove, state.GetRNG().Calls())
			}
			drainCombatPacingAt(scene, startMs+test.anticipate+test.move)
			assert.False(t, scene.combatMovement.Active)
			assert.False(t, callbacks.InputBlocked)
			assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
			assert.False(t, state.EventQueueManager.HasWorkflow("monster_turn_delay"))
			assert.Equal(t, 1, callbacks.BlockInputCalls)
			assert.Equal(t, 1, callbacks.UnblockInputCalls)
			after := snapshotBlockedCombatState(t, state)
			assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
			assert.Equal(t, after, snapshotBlockedCombatState(t, state))
		})
	}
}

func TestScheduledBatPacingIdleAndBlockedCompleteAtAnticipation(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		t.Run(fmt.Sprintf("blocked=%t", blocked), func(t *testing.T) {
			scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterBat, 6)
			state := scene.gameState
			monster := state.CombatScene.MonsterManager.GetMonster(step.Actor.EntityIndex)
			from := monster.Position()
			if blocked {
				// Occupancy blocks flyers too, without changing the actor's species.
				for _, position := range []references.Position{
					{X: from.X - 1, Y: from.Y}, {X: from.X + 1, Y: from.Y},
					{X: from.X, Y: from.Y - 1}, {X: from.X, Y: from.Y + 1},
				} {
					reference := (*state.GameReferences.EnemyReferences)[references.MonsterBat]
					state.CombatScene.AddMonsterToQueue(game_state.NewCombatMonster(
						references.MonsterBat, &reference, position, state.AsRandomProvider()))
				}
			} else {
				monster.EnemyRef().DoNotMove = true
			}
			state.SetRandomSeed(42)
			state.ElapsedMs = 900
			scene.startScheduledMonsterTurn(step)
			drainCombatPacingAt(scene, 900)
			drainCombatPacingAt(scene, 974)
			require.True(t, callbacks.InputBlocked)
			drainCombatPacingAt(scene, 975)
			assert.Equal(t, from, monster.Position())
			assert.False(t, scene.combatMovement.Active)
			assert.False(t, callbacks.InputBlocked)
			assert.False(t, state.EventQueueManager.HasWorkflow("monster_turn_delay"))
			assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
		})
	}
}

func TestScheduledBatPacingTargetedAttackRetainsAnticipation(t *testing.T) {
	scene, callbacks, first := newScheduledProjectileTestScene(t, references.MonsterBat, 6)
	scene.gameConfig = &config.UltimaVConfiguration{
		DebugConfiguration: &config.DebugConfiguration{CombatMovementSpeedFactor: 4},
	}
	state := scene.gameState
	_, err := state.ResolveScheduledCombatStep(first.Token)
	require.NoError(t, err)
	monster := state.CombatScene.MonsterManager.GetMonster(first.Actor.EntityIndex)
	position := monster.Position()
	state.CombatScene.SetPartyMemberPosition(0, references.Position{X: position.X + 1, Y: position.Y})
	require.NoError(t, state.CompleteScheduledCombatStep(first.Token))
	step := nextSignatureMonsterStep(t, scene, first.Actor.EntityIndex)
	require.True(t, step.HasTarget)
	callbacks.Reset()
	state.ElapsedMs = 900
	scene.startScheduledMonsterTurn(step)
	rngBefore := state.GetRNG().Calls()
	for _, age := range []int64{0, 75, 999} {
		drainCombatPacingAt(scene, 900+age)
		require.True(t, callbacks.InputBlocked)
		assert.Empty(t, callbacks.CombatEvents)
		assert.Equal(t, rngBefore, state.GetRNG().Calls())
	}
	drainCombatPacingAt(scene, 1900)
	assert.NotEmpty(t, callbacks.CombatEvents)
	assert.Equal(t, position, monster.Position())
	assert.False(t, scene.combatMovement.Active)
	assert.False(t, callbacks.InputBlocked)
	assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
}

func TestScheduledBatPacingCancellationAndEncounterExit(t *testing.T) {
	for _, resolve := range []bool{false, true} {
		for _, exit := range []bool{false, true} {
			t.Run(fmt.Sprintf("resolved=%t/exit=%t", resolve, exit), func(t *testing.T) {
				scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterBat, 6)
				state := scene.gameState
				state.ElapsedMs = 900
				scene.startScheduledMonsterTurn(step)
				drainCombatPacingAt(scene, 900)
				if resolve {
					drainCombatPacingAt(scene, 1000)
					require.True(t, scene.combatMovement.Active)
				}
				rngBeforeCancel := state.GetRNG().Calls()
				if exit {
					state.TransitionFromCombatMap()
				}
				state.EventQueueManager.Shutdown()
				assert.False(t, scene.combatMovement.Active)
				assert.Nil(t, scene.activeProjectile)
				assert.Empty(t, scene.combatSignatureEffects)
				assert.False(t, callbacks.InputBlocked)
				assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
				assert.Equal(t, 1, callbacks.UnblockInputCalls)
				if exit {
					assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationInactive)
				} else if resolve {
					assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
				} else {
					assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationUnresolved)
					assert.Equal(t, rngBeforeCancel, state.GetRNG().Calls())
				}
				rngAfterCancel := state.GetRNG().Calls()
				state.EventQueueManager.Shutdown()
				drainCombatPacingAt(scene, 5000)
				assert.Equal(t, rngAfterCancel, state.GetRNG().Calls(), "cleanup is idempotent")
				assert.Equal(t, 1, callbacks.UnblockInputCalls, "input release is exact-once")
			})
		}
	}
}

func TestScheduledBatPacingTeleportUsesExecutionResult(t *testing.T) {
	scene, callbacks, step := newScheduledProjectileTestScene(t, references.MonsterBat, 6)
	state := scene.gameState
	monster := state.CombatScene.MonsterManager.GetMonster(step.Actor.EntityIndex)
	// Synthetic capability isolates the result contract from species selection:
	// even an accelerated actor must never interpolate a teleport as walking.
	monster.EnemyRef().EnemyAbilities = map[references.EnemyAbility]bool{references.Teleport: true}
	seed := combatPacingTeleportSeed(t)
	state.SetRandomSeed(seed)
	state.ElapsedMs = 900
	scene.startScheduledMonsterTurn(step)
	drainCombatPacingAt(scene, 900)
	drainCombatPacingAt(scene, 974)
	require.Zero(t, state.GetRNG().Calls(), "planning and anticipation never preroll teleport")
	drainCombatPacingAt(scene, 975)
	require.Equal(t, references.Position{X: 5, Y: 5}, monster.Position())
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, game_state.CombatEventTeleport, callbacks.CombatEvents[0].Kind)
	assert.False(t, scene.combatMovement.Active)
	assert.False(t, callbacks.InputBlocked)
	assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
}

func combatPacingTeleportSeed(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 100000; seed++ {
		rng := deterministic_random.New(seed)
		if rng.Intn(4) != 3 && rng.Intn(16) == 5 && rng.Intn(16) == 5 {
			return seed
		}
	}
	t.Fatal("no seed found for the fixed teleport destination")

	return 0
}

func TestCombatPacingPlayerAndControlledBatRetainMovementDuration(t *testing.T) {
	for _, controlled := range []bool{false, true} {
		t.Run(fmt.Sprintf("controlled=%t", controlled), func(t *testing.T) {
			scene, callbacks := newCombatUseTestScene(t)
			state := scene.gameState
			scene.gameConfig = &config.UltimaVConfiguration{
				DebugConfiguration: &config.DebugConfiguration{CombatMovementSpeedFactor: 2},
			}
			actor := state.CombatScene.GetCurrentTurnIdentity()
			from, to := references.Position{X: 2, Y: 3}, references.Position{X: 3, Y: 3}
			reference := (*state.GameReferences.EnemyReferences)[references.MonsterBat]
			monster := game_state.NewCombatMonster(references.MonsterBat, &reference, to, state.AsRandomProvider())
			monster.SetLoyalty(game_state.CombatMonsterLoyaltyParty)
			monsterIndex := state.CombatScene.AddMonsterToQueue(monster)
			hostileReference := (*state.GameReferences.EnemyReferences)[references.MonsterOrc]
			state.CombatScene.AddMonsterToQueue(game_state.NewCombatMonster(
				references.MonsterOrc, &hostileReference, references.Position{X: 9, Y: 0}, state.AsRandomProvider()))
			token := state.FinishCombatTurnDeferred()
			state.ElapsedMs = 900
			if controlled {
				scene.startControlledMonsterCombatMovementWorkflow(monsterIndex, from, to, token)
			} else {
				scene.startPlayerCombatMovementWorkflow(actor, from, to, token)
			}
			drainCombatPacingAt(scene, 900)
			drainCombatPacingAt(scene, 1066)
			require.True(t, callbacks.InputBlocked, "333ms at 2x rounds to 167ms")
			require.True(t, scene.combatMovement.Active)
			drainCombatPacingAt(scene, 1067)
			assert.False(t, callbacks.InputBlocked)
			assert.False(t, scene.combatMovement.Active)
			assert.ErrorIs(t, state.CompleteScheduledCombatStep(token), game_state.ErrCombatPresentationDuplicate)
		})
	}
}
