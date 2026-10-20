package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func addContinuationEnemy(t *testing.T, state *GameState, monster references.MonsterType, pos references.Position, slot int) *map_units.NPCEnemy {
	t.Helper()
	enemy := addOverworldTestEnemy(t, state, monster, pos)
	enemy.MapUnitDetails().NPCNum = slot

	return enemy
}

func TestOutdoorActorContinuationOrdersConsecutiveBattlesBeforeFireball(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		for _, reason := range []CombatEndReason{CombatVictory, CombatEvacuated, CombatDefeat} {
			t.Run(fmt.Sprintf("floor%d/reason%d", floor, reason), func(t *testing.T) {
				state, mock := newOutdoorFireballFixture(t, floor, references.HorseVehicle)
				fillOutdoorFireballTerrain(state, indexes.Grass)
				first := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
				second := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 100, Y: 101}, 20)
				fireball := addContinuationEnemy(t, state, references.MonsterDragon, references.Position{X: 98, Y: 99}, 10)
				mover := addContinuationEnemy(t, state, references.MonsterOrc, references.Position{X: 95, Y: 100}, 1)
				before := state.MapState.PlayerLocation
				state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 10}
				state.quicknessWorldActorSkip, state.outdoorMountedActorSkip = true, true
				seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
					rng.Intn(8)             // First primary DEX.
					rng.Intn(8)             // Second primary DEX.
					return rng.Intn(8) == 0 // Later fireball opportunity.
				})
				state.SetRandomSeed(seed)
				oracle := beradrng.New(seed)
				state.processNonCombatWorldActorPhase()
				assert.Zero(t, state.GetRNG().Calls(), "nothing below the first contact is evaluated")
				require.Same(t, first, state.pendingOverworldEncounter)
				require.NoError(t, state.ResolvePendingOverworldContact())
				require.NotNil(t, state.CombatScene)
				oracle.Intn(8)
				assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())
				assert.False(t, state.HasPendingOverworldContact(), "retained phase cannot intercept tactical input")
				require.NoError(t, state.ResolvePendingOverworldContact())
				state.processNonCombatWorldActorPhase()
				assert.Equal(t, oracle.Calls(), state.GetRNG().Calls(), "reentrant drains cannot repeat a suspended phase")
				assert.Empty(t, state.pirateAttackQueue)
				assert.Equal(t, uint16(100), state.PartyState.GetCharacter(0).GetCurrentHp())
				state.DebugOptions.MonsterGen = true // Any repeated generation would consume extra RNG.
				// Retain the phase's source slot, not a fresh lookup through a mutable
				// actor record after combat/map-unit removal.
				first.MapUnitDetails().NPCNum = 2

				state.EndCombat(reason)

				assert.Equal(t, before, state.MapState.PlayerLocation)
				require.Same(t, second, state.pendingOverworldEncounter)
				assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())
				assert.Empty(t, state.pirateAttackQueue, "fireball is below the second synchronous combat")
				assert.NotContains(t, *state.GetCurrentLargeMapNPCAIController().GetNpcs(), first)
				assert.Contains(t, *state.GetCurrentLargeMapNPCAIController().GetNpcs(), second)
				require.NoError(t, state.ResolvePendingOverworldContact())
				require.NotNil(t, state.CombatScene)
				assert.Same(t, second, state.CombatScene.OverworldOrigin.SourceEnemy)
				oracle.Intn(8)
				assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())

				state.EndCombat(reason)

				oracle.Intn(8)
				wantHP := uint16(100 - oracle.Intn(8) - 1)
				assert.Equal(t, wantHP, state.PartyState.GetCharacter(0).GetCurrentHp())
				require.Len(t, state.pirateAttackQueue, 1)
				assert.Equal(t, fireball.Pos(), state.pirateAttackQueue[0].result.Source)
				assert.Equal(t, before.Position, state.pirateAttackQueue[0].result.Target)
				assert.Equal(t, before, state.MapState.PlayerLocation)
				assert.NotContains(t, *state.GetCurrentLargeMapNPCAIController().GetNpcs(), second)
				assert.Equal(t, references.Position{X: 95, Y: 100}, mover.Pos(), "attacked suppresses all later movement")
				assert.False(t, state.quicknessWorldActorSkip)
				assert.False(t, state.outdoorMountedActorSkip)
				assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())
				drainGameplayWorkflows(t, state)
				require.NoError(t, state.ResolvePendingOverworldContact())
				assert.Equal(t, oracle.Calls(), state.GetRNG().Calls(), "presentation and repeated drain cannot run actors again")
				assert.Equal(t, oracle.Intn(1000000), state.GetRNG().Intn(1000000))
				assert.Equal(t, 2, countMessage(mock.Messages, "Attacked!"))
				assert.False(t, mock.InputBlocked)
				assert.False(t, state.HasPendingOverworldContact())
			})
		}
	}
}

func TestOutdoorActorContinuationWaitsForCombatExitWorkflow(t *testing.T) {
	state, mock := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
	first := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
	second := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 99, Y: 100}, 20)
	state.largeMapProcessTurnEffects()
	require.NoError(t, state.ResolvePendingOverworldContact())
	calls := state.GetRNG().Calls()
	state.EventQueueManager.Workflow("outdoor_test_combat_exit").Priority(event_queue.PriorityNormal).Sequential().BlockInput().
		First(func() { state.EndCombat(CombatVictory) }).Wait(100).Start()
	state.EventQueueManager.Update()
	require.Nil(t, state.CombatScene)
	assert.True(t, mock.InputBlocked)
	assert.True(t, state.HasPendingOverworldContact())
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Nil(t, state.pendingOverworldEncounter)
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.NotContains(t, *state.GetCurrentLargeMapNPCAIController().GetNpcs(), first)
	drainGameplayWorkflows(t, state)
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	assert.Same(t, second, state.CombatScene.OverworldOrigin.SourceEnemy)
	assert.False(t, mock.InputBlocked)
	assert.False(t, state.HasPendingOverworldContact())
}

func TestOutdoorActorContinuationStopsOnDeadPartyAndRawTransition(t *testing.T) {
	for _, mode := range []string{"dead defeat", "dead victory", "raw transition", "terminal"} {
		t.Run(mode, func(t *testing.T) {
			state, mock := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
			addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
			second := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 99, Y: 100}, 20)
			state.largeMapProcessTurnEffects()
			require.NoError(t, state.ResolvePendingOverworldContact())
			state.activeSlowMovement = &slowMovementState{remainingActorPhases: 1, extraMinutes: 2}
			state.activeShipRepair = &shipRepairState{awaitingCombat: true, completeSurfaceCycle: true}
			calls, before := state.GetRNG().Calls(), state.DateTime
			deaths := 0
			state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deaths++ }
			switch mode {
			case "dead defeat", "dead victory":
				state.PartyState.GetCharacter(0).SetHp(0)
				state.PartyState.GetCharacter(0).SetStatus(party_state.Dead)
				reason := CombatDefeat
				if mode == "dead victory" {
					reason = CombatVictory
				}
				state.EndCombat(reason)
				assert.Equal(t, 1, deaths)
			case "raw transition":
				state.TransitionFromCombatMap()
			case "terminal":
				state.finaleHandoffPending = true
				state.EndCombat(CombatVictory)
			}
			require.NoError(t, state.ResolvePendingOverworldContact())
			assert.Nil(t, state.outdoorActorContinuation)
			assert.Nil(t, state.activeSlowMovement)
			assert.Nil(t, state.activeShipRepair)
			assert.False(t, state.HasPendingOverworldContact())
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.Equal(t, before, state.DateTime)
			assert.Contains(t, *state.GetCurrentLargeMapNPCAIController().GetNpcs(), second)
			assert.False(t, mock.InputBlocked)
		})
	}
}

func TestOutdoorActorContinuationShutdownBeforeAndAfterCombatEntry(t *testing.T) {
	for _, entered := range []bool{false, true} {
		t.Run(fmt.Sprintf("entered%v", entered), func(t *testing.T) {
			state, mock := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
			addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
			addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 99, Y: 100}, 20)
			state.largeMapProcessTurnEffects()
			if entered {
				require.NoError(t, state.ResolvePendingOverworldContact())
			}
			calls := state.GetRNG().Calls()
			state.EventQueueManager.Shutdown()
			if entered {
				state.EndCombat(CombatVictory)
			}
			require.NoError(t, state.ResolvePendingOverworldContact())
			assert.Nil(t, state.outdoorActorContinuation)
			assert.False(t, state.HasPendingOverworldContact())
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.False(t, mock.InputBlocked)
		})
	}
}

func TestExplicitOutdoorAttackDoesNotInventActorContinuation(t *testing.T) {
	state, _ := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
	first := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 99, Y: 100}, 20)
	require.True(t, state.QueueOverworldEncounterAt(first.Pos()))
	state.processNonCombatWorldActorPhase()
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	assert.Nil(t, state.outdoorActorContinuation)
	calls := state.GetRNG().Calls()
	state.EndCombat(CombatVictory)
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Nil(t, state.CombatScene)
	assert.False(t, state.HasPendingOverworldContact())
	assert.Equal(t, calls, state.GetRNG().Calls())
}

func TestShipRepairOutdoorContinuationCompletesOneCycleAcrossTwoBattles(t *testing.T) {
	for _, reason := range []CombatEndReason{CombatVictory, CombatEvacuated, CombatDefeat} {
		t.Run(fmt.Sprintf("reason%d", reason), func(t *testing.T) {
			assertShipRepairContinuationAcrossBattles(t, reason)
		})
	}
}

func assertShipRepairContinuationAcrossBattles(t *testing.T, reason CombatEndReason) {
	t.Helper()
	state, mock := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
	state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailFurled)
	state.PartyVehicle.GetVehicleDetails().SetHullHP(50)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 99, Y: 100}, 20)
	before := state.DateTime
	require.True(t, state.ActionHoleUpRepairShip())
	state.CompleteSurfaceCommandCycle(references.LargeMapType)
	drainGameplayWorkflows(t, state)
	for battle := range 2 {
		require.NoError(t, state.ResolvePendingOverworldContact())
		require.NotNil(t, state.CombatScene)
		assert.Equal(t, before, state.DateTime)
		state.EndCombat(reason)
		if battle == 0 {
			assert.Zero(t, state.activeShipRepair.completedCycles)
			assert.False(t, state.EventQueueManager.HasWorkflow(shipRepairWorkflowName))
		}
	}
	drainGameplayWorkflows(t, state)
	want := before
	for range 5 {
		want.Advance(5)
	}
	want.Advance(2)
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, 5, countMessage(mock.Messages, "*hammering*"))
	assert.Nil(t, state.activeShipRepair)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.False(t, state.HasPendingOverworldContact())
	assert.False(t, mock.InputBlocked)
}

func TestOutdoorActorContinuationLivingPhysicalRetreatCompletesSlowMovementOnce(t *testing.T) {
	state := newSlowMovementFixture(t, indexes.Forest3)
	position := state.MapState.PlayerLocation.Position
	source := addContinuationEnemy(t, state, references.MonsterSeaSerpent,
		*references.Right.GetNewPositionInDirection(&position), 31)
	addContinuationEnemy(t, state, references.MonsterSandtrap,
		*references.Left.GetNewPositionInDirection(&position), 20)
	before := state.DateTime
	state.CompleteSurfaceMovementCycle(references.LargeMapType, true)
	require.NotNil(t, state.CombatScene)
	require.Same(t, source, state.CombatScene.OverworldOrigin.SourceEnemy)
	assert.Equal(t, before, state.DateTime)
	assert.Equal(t, 1, state.activeSlowMovement.remainingActorPhases)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}
	state.CombatScene.PartyPositions[0] = references.Position{X: 0, Y: 5}
	state.SetRandomSeed(123)
	oracle := beradrng.New(123)

	require.Equal(t, CombatMoveFled, state.MoveCombatCharacter(references.Left))
	assert.Positive(t, state.CombatScene.MonsterManager.CountActiveHostile())
	state.AdvanceCombatTurn() // Real whatsleft() produces CombatDefeat for the living retreat.

	require.Nil(t, state.CombatScene)
	assert.False(t, state.PartyState.IsAllPartyDead())
	assert.NotContains(t, *state.GetCurrentLargeMapNPCAIController().GetNpcs(), source)
	require.NotNil(t, state.activeSlowMovement)
	assert.Equal(t, 1, state.activeSlowMovement.remainingActorPhases)
	assert.Equal(t, before, state.DateTime)
	wantHP := uint16(100 - oracle.Intn(8) - 1)
	assert.Equal(t, wantHP, state.PartyState.GetAvatar().GetCurrentHp(), "lower sandtrap finishes the interrupted phase")
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining, "only the physical flee action has completed")

	require.NoError(t, state.ResolvePendingOverworldContact())

	wantHP -= uint16(oracle.Intn(8) + 1) // Remaining extra phase before check_slow's time cost.
	wantHP -= uint16(oracle.Intn(8) + 1) // Ordinary endplayer phase.
	assert.Equal(t, wantHP, state.PartyState.GetAvatar().GetCurrentHp())
	want := before
	want.Advance(4)
	want.Advance(2)
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, 3, state.ActiveDurationEffect.TurnsRemaining, "one ordinary player tail after the slow step")
	assert.Equal(t, oracle.Calls(), state.GetRNG().Calls())
	assert.False(t, state.HasPendingOverworldContact())
	assert.Nil(t, state.activeSlowMovement)
	assert.Nil(t, state.outdoorActorContinuation)
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, oracle.Calls(), state.GetRNG().Calls(), "a second drain cannot repeat actors or player work")
}

func TestOutdoorWhirlpoolComposedFinalizerCancelsPhaseOnHandoff(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprintf("transportCommitted%v", committed), func(t *testing.T) {
			assertOutdoorWhirlpoolHandoffCleanup(t, committed)
		})
	}
}

func assertOutdoorWhirlpoolHandoffCleanup(t *testing.T, committed bool) {
	t.Helper()
	state, mock := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
	state.PartyVehicle.GetVehicleDetails().SetHullHP(99)
	prepareOutdoorWhirlpoolDestination(t, state, true)
	pos := references.Position{X: 101, Y: 100}
	require.True(t, state.spawnWhirlpoolAt(pos))
	whirlpool := state.GetWhirlpoolAt(pos)
	whirlpool.MapUnitDetails().NPCNum = 20
	phase := state.suspendOutdoorActorPhase(whirlpool)
	state.RemoveWhirlpoolAt(pos)
	state.activeSlowMovement = &slowMovementState{remainingActorPhases: 1, extraMinutes: 2}
	state.activeShipRepair = &shipRepairState{awaitingCombat: true}
	name := outdoorWhirlpoolWorkflowName(20)
	handedOff := false
	owner := state.EventQueueManager.Workflow(name).Priority(event_queue.PriorityHigh)
	owner.Sequential().BlockInput().
		First(func() {}).ThenSequence(func(sequence *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		// Exercise runtime sequence construction and normal owner termination,
		// without Shutdown's separate GameState cleanup hiding a lost finalizer.
		state.buildOutdoorWhirlpoolTransportationQueue(sequence, phase)
		return sequence
	}).Then(func() { t.Error("cancelled transport must not resume its actor cursor") }).Start()
	for updates := 0; ; updates++ {
		require.Less(t, updates, 50)
		state.ElapsedMs += 100
		state.EventQueueManager.Update()
		if len(mock.MapFadeLevels) > 0 && mock.MapFadeLevels[len(mock.MapFadeLevels)-1] > 0 &&
			(state.MapState.PlayerLocation.Floor == -1) == committed {
			break
		}
	}
	require.Same(t, phase, state.outdoorActorContinuation)
	assert.True(t, mock.InputBlocked)
	assert.Empty(t, mock.StopRequests)
	// Priority preemption pauses the owner; it must not drop the phase early.
	state.EventQueueManager.Workflow("outdoor_whirlpool_preemption").Priority(event_queue.PriorityCritical).Sequential().
		First(func() {}).Wait(100).Start()
	state.EventQueueManager.Update()
	assert.Same(t, phase, state.outdoorActorContinuation)
	// Schedule the ordinary termination request on the existing owner while
	// its animation is suspended. Do not restart the queue or shut the manager down.
	owner.Sequential().HandoffTo("outdoor_whirlpool_cancel_successor", event_queue.PriorityHigh,
		func(successor *event_queue.SequentialEventBuilder) {
			successor.First(func() { handedOff = true })
		})
	drainGameplayWorkflows(t, state)
	assert.True(t, handedOff, "the manager stays usable after ordinary cancellation")
	assert.Nil(t, state.outdoorActorContinuation)
	assert.Nil(t, state.activeSlowMovement)
	assert.Nil(t, state.activeShipRepair)
	assert.False(t, state.HasPendingOverworldContact())
	assert.Empty(t, mock.ProjectilesStarted, "lower destination actors must not run")
	wantCalls := uint64(0)
	if committed {
		wantCalls = 1
	}
	assert.Equal(t, wantCalls, state.GetRNG().Calls(), "only a committed transport hit consumes RNG")
	require.Len(t, mock.StopRequests, 1)
	assert.Equal(t, SoundWhirlpoolVortex, mock.StopRequests[0].Effect)
	assert.Zero(t, mock.MapFadeLevels[len(mock.MapFadeLevels)-1])
	assert.False(t, mock.InputBlocked)
}

func TestSlowMovementOutdoorContinuationFinishesCurrentScanBeforeNextPhase(t *testing.T) {
	state := newSlowMovementFixture(t, indexes.Forest3)
	position := state.MapState.PlayerLocation.Position
	first := addContinuationEnemy(t, state, references.MonsterSeaSerpent,
		*references.Right.GetNewPositionInDirection(&position), 31)
	second := addContinuationEnemy(t, state, references.MonsterSeaSerpent,
		*references.Left.GetNewPositionInDirection(&position), 20)
	before := state.DateTime
	state.CompleteSurfaceMovementCycle(references.LargeMapType, true)
	for _, source := range []*map_units.NPCEnemy{first, second} {
		require.NoError(t, state.ResolvePendingOverworldContact())
		require.NotNil(t, state.CombatScene)
		assert.Same(t, source, state.CombatScene.OverworldOrigin.SourceEnemy)
		assert.Equal(t, 1, state.activeSlowMovement.remainingActorPhases)
		assert.Equal(t, before, state.DateTime)
		state.EndCombat(CombatEvacuated)
	}
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Nil(t, state.activeSlowMovement)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.Equal(t, byte(6), state.DateTime.Minute)
	assert.Equal(t, uint32(2), state.DateTime.Turn)
	assert.False(t, state.HasPendingOverworldContact())
}

func TestOutdoorActorContinuationFatalFireballOwnsSingleDeathFeedback(t *testing.T) {
	state, mock := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 98, Y: 100}, 20)
	state.PartyState.GetCharacter(0).SetHp(1)
	seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
		rng.Intn(8)
		return rng.Intn(8) == 0
	})
	state.SetRandomSeed(seed)
	deaths := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deaths++ }
	state.largeMapProcessTurnEffects()
	require.NoError(t, state.ResolvePendingOverworldContact())
	state.EndCombat(CombatVictory)
	assert.True(t, state.PartyState.IsAllPartyDead())
	assert.Zero(t, deaths, "fatal lower-slot shot owns its presentation and recovery")
	drainGameplayWorkflows(t, state)
	assert.Equal(t, 1, deaths)
	assert.False(t, mock.InputBlocked)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.False(t, state.HasPendingOverworldContact())
}

func TestOutdoorActorContinuationWhirlpoolThenCombatUsesRestoredWorld(t *testing.T) {
	state, mock := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
	state.PartyVehicle.GetVehicleDetails().SetHullHP(99)
	lower, _ := prepareOutdoorWhirlpoolDestination(t, state, true)
	before, controller := state.MapState.PlayerLocation, state.CurrentNPCAIController
	state.MapState.PlayerLocation.Floor = -1
	state.MapState.PlayerLocation.Position = references.Position{X: 34, Y: 18}
	state.CurrentNPCAIController = state.GetCurrentLargeMapNPCAIController()
	contact := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 35, Y: 18}, 10)
	state.MapState.PlayerLocation, state.CurrentNPCAIController = before, controller
	pos := references.Position{X: 101, Y: 100}
	require.True(t, state.spawnWhirlpoolAt(pos))
	state.GetWhirlpoolAt(pos).MapUnitDetails().NPCNum = 20
	state.largeMapProcessTurnEffects()
	drainGameplayWorkflows(t, state)
	assert.Equal(t, references.FloorNumber(-1), state.MapState.PlayerLocation.Floor)
	assert.Same(t, contact, state.pendingOverworldEncounter)
	assert.Empty(t, mock.ProjectilesStarted, "combat suspends the transported phase before its lower pirate")
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	assert.Same(t, contact, state.CombatScene.OverworldOrigin.SourceEnemy)
	assert.Equal(t, references.UNDERWORLD, state.CombatScene.OverworldOrigin.World)
	state.EndCombat(CombatVictory)
	require.Len(t, state.pirateAttackQueue, 1)
	assert.Equal(t, lower.Pos(), state.pirateAttackQueue[0].result.Source)
	assert.Equal(t, references.Position{X: 34, Y: 18}, state.pirateAttackQueue[0].result.Target)
	drainGameplayWorkflows(t, state)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.False(t, mock.InputBlocked)
	assert.False(t, state.HasPendingOverworldContact())
}

func TestOutdoorActorContinuationRejectsReplacedPendingSource(t *testing.T) {
	state, _ := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
	first := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 99, Y: 100}, 20)
	state.largeMapProcessTurnEffects()
	require.Same(t, first, state.pendingOverworldEncounter)
	require.True(t, state.GetCurrentLargeMapNPCAIController().GetNpcs().RemoveEnemy(first))
	replacement := addContinuationEnemy(t, state, references.MonsterSeaSerpent, first.Pos(), 31)
	calls := state.GetRNG().Calls()
	require.ErrorIs(t, state.ResolvePendingOverworldContact(), ErrInvalidOverworldEncounter)
	assert.Nil(t, state.CombatScene)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.False(t, state.HasPendingOverworldContact())
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Contains(t, *state.GetCurrentLargeMapNPCAIController().GetNpcs(), replacement)
}

func TestOutdoorActorContinuationShutdownDuringCombatExitDoesNotResume(t *testing.T) {
	state, mock := newOutdoorFireballFixture(t, 0, references.NoPartyVehicle)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 99, Y: 100}, 20)
	state.largeMapProcessTurnEffects()
	require.NoError(t, state.ResolvePendingOverworldContact())
	calls := state.GetRNG().Calls()
	state.EventQueueManager.Workflow("outdoor_test_cancel_exit").Priority(event_queue.PriorityNormal).Sequential().BlockInput().
		First(func() { state.EndCombat(CombatEvacuated) }).Wait(100).Start()
	state.EventQueueManager.Update()
	require.Nil(t, state.CombatScene)
	require.NotNil(t, state.outdoorActorContinuation)
	state.EventQueueManager.Shutdown()
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Nil(t, state.outdoorActorContinuation)
	assert.False(t, state.HasPendingOverworldContact())
	assert.False(t, mock.InputBlocked)
	assert.Equal(t, calls, state.GetRNG().Calls())
}

func TestShipRepairOutdoorContinuationWaitsForLowerSlotFireball(t *testing.T) {
	state, mock := newOutdoorFireballFixture(t, 0, references.FrigateVehicle)
	state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailFurled)
	state.PartyVehicle.GetVehicleDetails().SetHullHP(50)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 101, Y: 100}, 31)
	fireball := addContinuationEnemy(t, state, references.MonsterDragon, references.Position{X: 98, Y: 99}, 20)
	seed := outdoorFireballSeed(t, func(rng *beradrng.Source) bool {
		rng.Intn(8)
		return rng.Intn(8) == 0
	})
	state.SetRandomSeed(seed)
	before := state.DateTime
	require.True(t, state.ActionHoleUpRepairShip())
	state.CompleteSurfaceCommandCycle(references.LargeMapType)
	drainGameplayWorkflows(t, state)
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	state.EndCombat(CombatVictory)
	assert.Equal(t, before, state.DateTime)
	assert.Zero(t, state.activeShipRepair.completedCycles)
	require.True(t, state.outdoorActorContinuation.scanned)
	assert.True(t, state.ShouldDeferOverworldContact())
	assert.False(t, state.EventQueueManager.HasWorkflow(shipRepairWorkflowName))
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Zero(t, state.activeShipRepair.completedCycles)
	drainGameplayWorkflows(t, state)
	assert.True(t, state.HasPendingOverworldContact(), "the input drain must still finish the interrupted repair cycle")
	require.True(t, state.GetCurrentLargeMapNPCAIController().GetNpcs().RemoveEnemy(fireball))
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Equal(t, 1, state.activeShipRepair.completedCycles)
	drainGameplayWorkflows(t, state)
	want := before
	for range 5 {
		want.Advance(5)
	}
	want.Advance(2)
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, 5, countMessage(mock.Messages, "*hammering*"))
	assert.Len(t, mock.ProjectilesStarted, 1)
	assert.False(t, mock.InputBlocked)
	assert.Nil(t, state.activeShipRepair)
	assert.False(t, state.HasPendingOverworldContact())
}
