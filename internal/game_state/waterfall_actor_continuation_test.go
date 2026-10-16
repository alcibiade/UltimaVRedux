package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func openWaterfallActorLanding(state *GameState) {
	for x := references.Coordinate(47); x <= 53; x++ {
		position := references.Position{X: x, Y: 52}
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Grass)
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Grass)
	}
}

func drainWaterfallInstance(t *testing.T, state *GameState) {
	t.Helper()
	advanceWaterfallUntil(t, state, func() bool { return !state.EventQueueManager.HasWorkflow(waterfallWorkflowName) })
}

func TestWaterfallActorsRunBetweenFirstAndSecondFallsExactlyOnce(t *testing.T) {
	state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 3)
	addContinuationEnemy(t, state, references.MonsterSandtrap, references.Position{X: 51, Y: 52}, 31)
	addContinuationEnemy(t, state, references.MonsterSandtrap, references.Position{X: 49, Y: 52}, 20)
	var landings []references.Position
	var completedFalls []int
	state.SystemCallbacks.Screen.UpdateStatsDisplay = func() {
		landings = append(landings, state.MapState.PlayerLocation.Position)
		completedFalls = append(completedFalls, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
		state.DebugOptions.MonsterGen = true // A replayed actor phase would draw generation RNG.
	}
	state.CompleteSurfacePassCycle(references.LargeMapType)
	charged := state.DateTime
	drainWaterfallInstance(t, state)
	expected := deterministic_random.New(123)
	expected.Intn(originalD30SourceValueCount)
	expected.Intn(1)
	hits := expected.Intn(8) + 1 + expected.Intn(8) + 1
	for range 2 {
		expected.Intn(originalD30SourceValueCount)
		expected.Intn(1)
	}
	assert.Equal(t, []references.Position{{X: 50, Y: 52}, {X: 50, Y: 52}}, landings)
	assert.Equal(t, []int{1, 1}, completedFalls)
	assert.Equal(t, references.Position{X: 50, Y: 56}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, uint16(997-hits), state.PartyState.GetAvatar().GetCurrentHp())
	assert.Equal(t, expected.Calls(), state.GetRNG().Calls())
	assert.Equal(t, charged, state.DateTime)
	assert.Equal(t, 7, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, 1, state.beggarKarmaCounter)
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, state.ResumeWaterfallContinuation())
}

func TestWaterfallActorProjectilesDrainBeforeAutomaticContinuation(t *testing.T) {
	state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	openWaterfallActorLanding(state)
	addContinuationEnemy(t, state, references.MonsterDragon, references.Position{X: 48, Y: 52}, 31)
	addContinuationEnemy(t, state, references.MonsterDragon, references.Position{X: 52, Y: 52}, 20)
	seed := outdoorFireballSeed(t, func(random *deterministic_random.Source) bool {
		random.Intn(originalD30SourceValueCount)
		random.Intn(1)
		if random.Intn(8) != 0 {
			return false
		}
		random.Intn(8)

		return random.Intn(8) == 0
	})
	state.SetRandomSeed(seed)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	charged := state.DateTime
	cascade := state.activeWaterfall
	drainWaterfallInstance(t, state)
	require.Same(t, cascade, state.activeWaterfall)
	assert.True(t, cascade.actorTailDone)
	assert.True(t, cascade.awaitingActors)
	require.True(t, state.EventQueueManager.HasWorkflow(pirateAttackWorkflowName))
	assert.Equal(t, references.Position{X: 50, Y: 52}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.True(t, state.ResumeWaterfallContinuation(), "projectile owner must drain before restarting")
	assert.False(t, state.EventQueueManager.HasWorkflow(waterfallWorkflowName))
	assert.ErrorIs(t, state.ValidateSaveSnapshotEligibility(), ErrSaveSnapshotNotRepresentable)
	drainSourceHitWorkflows(t, state)
	require.Len(t, callbacks.ProjectilesStarted, 2)
	assert.Empty(t, state.pirateAttackQueue)
	assert.False(t, callbacks.InputBlocked)
	assert.ErrorIs(t, state.ValidateSaveSnapshotEligibility(), ErrSaveSnapshotNotRepresentable,
		"the logical cascade remains unrepresentable with no workflow or contact left")
	assert.True(t, state.ResumeWaterfallContinuation())
	require.True(t, state.EventQueueManager.HasWorkflow(waterfallWorkflowName))
	assert.Same(t, cascade, state.activeWaterfall)
	assert.True(t, state.ResumeWaterfallContinuation(), "repeated resume cannot acquire the same live name twice")
	drainWaterfallInstance(t, state)
	expected := deterministic_random.New(seed)
	expected.Intn(originalD30SourceValueCount)
	expected.Intn(1)
	expected.Intn(8)
	hits := expected.Intn(8) + 1
	expected.Intn(8)
	hits += expected.Intn(8) + 1
	expected.Intn(originalD30SourceValueCount)
	expected.Intn(1)
	assert.Equal(t, uint16(998-hits), state.PartyState.GetAvatar().GetCurrentHp())
	assert.Equal(t, expected.Calls(), state.GetRNG().Calls())
	assert.Equal(t, charged, state.DateTime)
	assert.Equal(t, 1, state.beggarKarmaCounter)
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, state.ResumeWaterfallContinuation())
	assert.NoError(t, state.ValidateSaveSnapshotEligibility())
}

func TestWaterfallActorCursorSurvivesMultipleBattlesThenFireball(t *testing.T) {
	for _, reason := range []CombatEndReason{CombatVictory, CombatEvacuated, CombatDefeat} {
		t.Run(fmt.Sprintf("return%d", reason), func(t *testing.T) {
			state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
			setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
			openWaterfallActorLanding(state)
			first := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 51, Y: 52}, 31)
			second := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 49, Y: 52}, 20)
			addContinuationEnemy(t, state, references.MonsterDragon, references.Position{X: 48, Y: 52}, 10)
			seed := outdoorFireballSeed(t, func(random *deterministic_random.Source) bool {
				random.Intn(originalD30SourceValueCount)
				random.Intn(1)
				random.Intn(8) // First battle's monster DEX.
				random.Intn(8) // Second battle's monster DEX.

				return random.Intn(8) == 0
			})
			state.SetRandomSeed(seed)
			state.CompleteSurfacePassCycle(references.LargeMapType)
			charged := state.DateTime
			cascade := state.activeWaterfall
			drainWaterfallInstance(t, state)
			require.Same(t, first, state.pendingOverworldEncounter)
			require.Same(t, cascade, state.outdoorActorContinuation.waterfall)
			assert.False(t, callbacks.InputBlocked, "tactical contact must not inherit the waterfall block")
			assert.True(t, state.ResumeWaterfallContinuation())
			assert.False(t, state.EventQueueManager.HasWorkflow(waterfallWorkflowName))
			state.DebugOptions.MonsterGen = true
			for _, enemy := range []*map_units.NPCEnemy{first, second} {
				require.NoError(t, state.ResolvePendingOverworldContact())
				require.NotNil(t, state.CombatScene)
				assert.Same(t, enemy, state.CombatScene.OverworldOrigin.SourceEnemy)
				assert.False(t, state.ResumeWaterfallContinuation(), "owned tactical input is allowed")
				assert.Same(t, cascade, state.activeWaterfall)
				assert.Equal(t, 1, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
				state.EndCombat(reason)
				assert.Equal(t, references.Position{X: 50, Y: 52}, state.MapState.PlayerLocation.Position)
				assert.Equal(t, charged, state.DateTime)
			}
			require.True(t, state.EventQueueManager.HasWorkflow(pirateAttackWorkflowName))
			assert.True(t, state.ResumeWaterfallContinuation())
			drainSourceHitWorkflows(t, state)
			require.NoError(t, state.ResolvePendingOverworldContact())
			assert.False(t, state.HasPendingOverworldContact())
			assert.ErrorIs(t, state.ValidateSaveSnapshotEligibility(), ErrSaveSnapshotNotRepresentable)
			assert.True(t, state.ResumeWaterfallContinuation())
			drainWaterfallInstance(t, state)
			expected := deterministic_random.New(seed)
			expected.Intn(originalD30SourceValueCount)
			expected.Intn(1)
			expected.Intn(8)
			expected.Intn(8)
			expected.Intn(8)
			hit := expected.Intn(8) + 1
			expected.Intn(originalD30SourceValueCount)
			expected.Intn(1)
			assert.Equal(t, uint16(998-hit), state.PartyState.GetAvatar().GetCurrentHp())
			assert.Equal(t, expected.Calls(), state.GetRNG().Calls(), "never replay actor generation, DEX or firing rolls")
			assert.Equal(t, references.Position{X: 50, Y: 54}, state.MapState.PlayerLocation.Position)
			assert.Equal(t, 2, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
			assert.Equal(t, 1, state.beggarKarmaCounter)
			assert.Equal(t, charged, state.DateTime)
			assert.Equal(t, 7, state.ActiveDurationEffect.TurnsRemaining)
			assert.Nil(t, state.activeWaterfall)
			assert.Nil(t, state.outdoorActorContinuation)
			assert.False(t, callbacks.InputBlocked)
		})
	}
}

func TestWaterfallOwnedWhirlpoolRechecksActualDestinationAfterLowerActors(t *testing.T) {
	state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle,
		state.MapState.PlayerLocation.Position, references.MainFloor)
	state.CurrentNPCAIController = state.GetCurrentLargeMapNPCAIController()
	whirlpoolPosition := references.Position{X: 51, Y: 52}
	require.True(t, state.spawnWhirlpoolAt(whirlpoolPosition))
	state.GetWhirlpoolAt(whirlpoolPosition).MapUnitDetails().NPCNum = 20
	destination := references.Position{X: whirlpoolDestinationX, Y: whirlpoolDestinationY}
	underworld := state.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, references.Basement)
	for y := destination.Y; y <= destination.Y+3; y++ {
		position := references.Position{X: destination.X, Y: y}
		underworld.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Grass)
	}
	setWaterfallContinuationChain(state, references.Basement, destination, 1)
	// A lower-slot hit in the destination world must precede the second fall.
	enemy := map_units.NewEnemyNPC((*state.GameReferences.EnemyReferences)[references.MonsterSandtrap], 10)
	enemy.SetPos(references.Position{X: destination.X + 1, Y: destination.Y})
	enemy.SetFloor(references.Basement)
	require.True(t, state.LargeMapNPCAIController[references.UNDERWORLD].GetNpcs().AddEnemy(enemy))
	stored := state.LargeMapNPCAIController[references.UNDERWORLD].GetNpcs().
		GetVisibleEnemyAtPositionAndFloorOrNil(enemy.Pos(), references.Basement)
	require.NotNil(t, stored)
	stored.MapUnitDetails().NPCNum = 10
	var actorLandings []references.PlayerLocation
	state.SystemCallbacks.Screen.UpdateStatsDisplay = func() {
		actorLandings = append(actorLandings, state.MapState.PlayerLocation)
	}
	state.CompleteSurfacePassCycle(references.LargeMapType)
	charged := state.DateTime
	cascade := state.activeWaterfall
	drainWaterfallInstance(t, state)
	require.True(t, state.EventQueueManager.HasWorkflow(outdoorWhirlpoolWorkflowName(20)))
	assert.True(t, state.ResumeWaterfallContinuation())
	drainSourceHitWorkflows(t, state)
	require.NoError(t, state.ResolvePendingOverworldContact())
	assert.Same(t, cascade, state.activeWaterfall)
	assert.Equal(t, references.Basement, cascade.landing.Floor)
	assert.Equal(t, destination, cascade.landing.Position)
	require.NotEmpty(t, actorLandings)
	assert.Equal(t, cascade.landing, actorLandings[len(actorLandings)-1])
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.True(t, state.ResumeWaterfallContinuation())
	drainWaterfallInstance(t, state)
	expected := deterministic_random.New(123)
	expected.Intn(originalD30SourceValueCount)
	expected.Intn(1)
	hits := expected.Intn(8) + 1 + expected.Intn(8) + 1
	expected.Intn(originalD30SourceValueCount)
	expected.Intn(1)
	assert.Equal(t, expected.Calls(), state.GetRNG().Calls())
	assert.Equal(t, uint16(998-hits), state.PartyState.GetAvatar().GetCurrentHp())
	assert.Equal(t, references.Position{X: destination.X, Y: destination.Y + 2}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, charged, state.DateTime)
	assert.Equal(t, 1, state.beggarKarmaCounter)
	assert.Nil(t, state.activeWaterfall)
}

func TestWaterfallInterludeRejectsDuplicatesShutdownAndForeignReplacement(t *testing.T) {
	for _, mode := range []string{"shutdown", "foreign position", "foreign map", "party defeat", "cancel cursor"} {
		t.Run(mode, func(t *testing.T) {
			state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
			setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
			openWaterfallActorLanding(state)
			addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 51, Y: 52}, 31)
			addContinuationEnemy(t, state, references.MonsterSandtrap, references.Position{X: 49, Y: 52}, 20)
			state.CompleteSurfacePassCycle(references.LargeMapType)
			drainWaterfallInstance(t, state)
			cascade := state.activeWaterfall
			require.NotNil(t, cascade)
			assert.False(t, callbacks.InputBlocked)
			assert.ErrorIs(t, state.ValidateSaveSnapshotEligibility(), ErrSaveSnapshotNotRepresentable)
			date, calls := state.DateTime, state.GetRNG().Calls()
			state.CompleteSurfacePassCycle(references.LargeMapType)
			state.CompleteSurfaceMovementCycle(references.LargeMapType, true)
			require.True(t, state.ProcessWaterfallHazard(nil))
			assert.Same(t, cascade, state.activeWaterfall)
			assert.False(t, state.EventQueueManager.HasWorkflow(waterfallWorkflowName))
			assert.Equal(t, date, state.DateTime)
			assert.Equal(t, calls, state.GetRNG().Calls())
			switch mode {
			case "shutdown":
				state.EventQueueManager.Shutdown()
			case "foreign position":
				// Still adjacent to the old source: adjacency alone cannot validate
				// a retained phase after an unrelated relocation.
				state.SetPlayerPosition(references.Position{X: 52, Y: 52})
			case "foreign map":
				state.TransitionToLocation(references.BritanniaUnderworld, references.Basement,
					references.Position{X: 50, Y: 90}, TransitionFullLocation, false)
			case "party defeat":
				state.PartyState.GetAvatar().SubtractHp(1000)
			case "cancel cursor":
				state.cancelOutdoorActorPhase()
			}
			position, hp := state.MapState.PlayerLocation, state.PartyState.GetAvatar().GetCurrentHp()
			require.NoError(t, state.ResolvePendingOverworldContact())
			state.ResumeWaterfallContinuation()
			assert.Nil(t, state.activeWaterfall)
			assert.Nil(t, state.outdoorActorContinuation)
			assert.Nil(t, state.CombatScene)
			assert.Equal(t, position, state.MapState.PlayerLocation)
			assert.Equal(t, hp, state.PartyState.GetAvatar().GetCurrentHp())
			assert.Equal(t, calls, state.GetRNG().Calls(), "replacement cannot run a late contact or lower actor")
			assert.Equal(t, date, state.DateTime)
			assert.False(t, callbacks.InputBlocked)
			assert.False(t, state.IsWaterfallHazardActive())
		})
	}
}

func TestWaterfallHandoffWaitsForPhysicalNameRelease(t *testing.T) {
	state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 51, Y: 52}, 31)
	addContinuationEnemy(t, state, references.MonsterSandtrap, references.Position{X: 49, Y: 52}, 20)
	held := true
	state.SystemCallbacks.Visual.IsAnyKeyPressed = func() bool { return held }
	state.CompleteSurfacePassCycle(references.LargeMapType)
	advanceWaterfallUntil(t, state, func() bool { return state.HasPendingOverworldContact() })
	require.NotNil(t, state.activeWaterfall)
	assert.True(t, state.activeWaterfall.awaitingActors)
	date, calls := state.DateTime, state.GetRNG().Calls()
	for range 5 {
		assert.True(t, state.ResumeWaterfallContinuation())
		state.CompleteSurfacePassCycle(references.LargeMapType)
		state.EventQueueManager.Update()
		assert.True(t, state.EventQueueManager.HasWorkflow(waterfallWorkflowName))
		assert.True(t, callbacks.InputBlocked)
	}
	assert.Equal(t, date, state.DateTime)
	assert.Equal(t, calls, state.GetRNG().Calls())
	held = false
	state.EventQueueManager.Update()
	assert.False(t, state.EventQueueManager.HasWorkflow(waterfallWorkflowName))
	assert.False(t, callbacks.InputBlocked)
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	assert.False(t, state.ResumeWaterfallContinuation())
	calls = state.GetRNG().Calls()
	hp := state.PartyState.GetAvatar().GetCurrentHp()
	state.EventQueueManager.Shutdown()
	assert.Nil(t, state.activeWaterfall)
	state.EndCombat(CombatVictory)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.Equal(t, hp, state.PartyState.GetAvatar().GetCurrentHp(), "late return after shutdown cannot attack from the discarded cursor")
	assert.Equal(t, calls, state.GetRNG().Calls())
}

func TestWaterfallShutdownAfterActorPresentationLeavesNoRetainedOwner(t *testing.T) {
	state, _ := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	require.True(t, state.startWaterfallHazard(func() {
		state.EventQueueManager.Workflow(pirateAttackWorkflowName).Priority(event_queue.PriorityHigh).
			Sequential().BlockInput().First(func() {}).Wait(100).Start()
	}))
	drainWaterfallInstance(t, state)
	drainSourceHitWorkflows(t, state)
	require.NotNil(t, state.activeWaterfall)
	require.False(t, state.EventQueueManager.HasActiveWorkflows())
	assert.ErrorIs(t, state.ValidateSaveSnapshotEligibility(), ErrSaveSnapshotNotRepresentable)
	state.EventQueueManager.Shutdown()
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, state.ResumeWaterfallContinuation())
}

func TestWaterfallOnFootWhirlpoolPresentationFinishesBeforeSecondFall(t *testing.T) {
	state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	state.CurrentNPCAIController = state.GetCurrentLargeMapNPCAIController()
	position := references.Position{X: 51, Y: 52}
	require.True(t, state.spawnWhirlpoolAt(position))
	state.GetWhirlpoolAt(position).MapUnitDetails().NPCNum = 20
	state.CompleteSurfacePassCycle(references.LargeMapType)
	drainWaterfallInstance(t, state)
	assert.Nil(t, state.outdoorActorContinuation, "foot contact has presentation but no suspended cursor")
	require.True(t, state.EventQueueManager.HasWorkflow(outdoorWhirlpoolWorkflowName(20)))
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Equal(t, references.Position{X: 50, Y: 52}, state.MapState.PlayerLocation.Position)
	assert.True(t, state.ResumeWaterfallContinuation())
	assert.False(t, state.EventQueueManager.HasWorkflow(waterfallWorkflowName))
	drainSourceHitWorkflows(t, state)
	assert.True(t, state.ResumeWaterfallContinuation())
	drainWaterfallInstance(t, state)
	assert.Equal(t, 2, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Nil(t, state.activeWaterfall)
}

func TestWaterfallNonblockingBackgroundDoesNotStarveResume(t *testing.T) {
	state, _ := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	state.EventQueueManager.RegisterBackgroundQueue("waterfall_test_background").Priority(event_queue.PriorityLow).
		Sequential().First(func() {}).While(func() bool { return true }, func() {}, 100).Start()
	state.CompleteSurfacePassCycle(references.LargeMapType)
	drainWaterfallInstance(t, state)
	require.NotNil(t, state.activeWaterfall)
	assert.True(t, state.EventQueueManager.HasActiveBackgroundQueues())
	assert.True(t, state.ResumeWaterfallContinuation(), "background liveness is not an interaction owner")
	drainWaterfallInstance(t, state)
	assert.Equal(t, references.Position{X: 50, Y: 54}, state.MapState.PlayerLocation.Position)
	assert.Nil(t, state.activeWaterfall)
	state.EventQueueManager.Shutdown()
}

func TestWaterfallFatalFirstLandingActorCancelsBeforeSecondFall(t *testing.T) {
	state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	state.PartyState.GetAvatar().SetHp(2)
	addContinuationEnemy(t, state, references.MonsterSandtrap, references.Position{X: 51, Y: 52}, 31)
	deaths := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() { deaths++ }
	state.CompleteSurfacePassCycle(references.LargeMapType)
	drainWaterfallInstance(t, state)
	assert.Equal(t, 1, deaths)
	assert.True(t, state.PartyState.IsAllPartyDead())
	assert.Equal(t, references.Position{X: 50, Y: 52}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, state.ResumeWaterfallContinuation())
}

func TestWaterfallForeignEncounterCannotInheritRetainedActorCursor(t *testing.T) {
	state, _ := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	openWaterfallActorLanding(state)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 51, Y: 52}, 31)
	addContinuationEnemy(t, state, references.MonsterSandtrap, references.Position{X: 49, Y: 52}, 20)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	drainWaterfallInstance(t, state)
	replacement := addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 50, Y: 51}, 10)
	require.NoError(t, state.StartOverworldCombatEncounter(replacement))
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, state.HasPendingOverworldContact(), "an old contact cannot starve the replacement's tactical input")
	calls, hp := state.GetRNG().Calls(), state.PartyState.GetAvatar().GetCurrentHp()
	assert.False(t, state.ResumeWaterfallContinuation())
	assert.Nil(t, state.activeWaterfall)
	state.EndCombat(CombatVictory)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.Equal(t, hp, state.PartyState.GetAvatar().GetCurrentHp())
	assert.Equal(t, calls, state.GetRNG().Calls(), "foreign combat return cannot run the suspended lower sandtrap")
	assert.False(t, state.ResumeWaterfallContinuation())
}

func TestWaterfallVisitedCycleBoundSurvivesTacticalHandoff(t *testing.T) {
	start := references.Position{X: 50, Y: 0}
	state, callbacks := newWaterfallContinuationFixture(t, start)
	fallCount := int(references.YLargeMapTiles) / 2
	setWaterfallContinuationChain(state, references.MainFloor, start, fallCount)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 51, Y: 2}, 31)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	cascade := state.activeWaterfall
	charged := state.DateTime
	drainWaterfallInstance(t, state)
	require.Len(t, cascade.visited, 1)
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	state.EndCombat(CombatVictory)
	require.Same(t, cascade, state.activeWaterfall)
	assert.True(t, state.ResumeWaterfallContinuation())
	drainWaterfallInstance(t, state)
	assert.Equal(t, start, state.MapState.PlayerLocation.Position)
	assert.Len(t, cascade.visited, fallCount)
	assert.Equal(t, fallCount, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "The waterfall current circles back. You regain control."))
	assert.Equal(t, charged, state.DateTime)
	assert.Equal(t, uint16(1000-fallCount), state.PartyState.GetAvatar().GetCurrentHp())
	assert.Nil(t, state.activeWaterfall)
	assert.False(t, callbacks.InputBlocked)
}

func TestWaterfallTacticalDefeatDiscardsCascadeAndLowerActors(t *testing.T) {
	state, callbacks := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	openWaterfallActorLanding(state)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 51, Y: 52}, 31)
	addContinuationEnemy(t, state, references.MonsterSandtrap, references.Position{X: 49, Y: 52}, 20)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	drainWaterfallInstance(t, state)
	require.NoError(t, state.ResolvePendingOverworldContact())
	require.NotNil(t, state.CombatScene)
	state.PartyState.GetAvatar().SubtractHp(1000)
	calls := state.GetRNG().Calls()
	state.EndCombat(CombatDefeat)
	assert.Nil(t, state.activeWaterfall)
	assert.Nil(t, state.outdoorActorContinuation)
	assert.False(t, state.ResumeWaterfallContinuation())
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Equal(t, 1, waterfallMessageCount(callbacks, "F-A-L-L-S!!!"))
	assert.False(t, callbacks.InputBlocked)
}

func TestWaterfallStaleOwnerCannotCancelItsReplacement(t *testing.T) {
	state, _ := newWaterfallContinuationFixture(t, references.Position{X: 50, Y: 50})
	setWaterfallContinuationChain(state, references.MainFloor, state.MapState.PlayerLocation.Position, 2)
	addContinuationEnemy(t, state, references.MonsterSeaSerpent, references.Position{X: 51, Y: 52}, 31)
	state.CompleteSurfacePassCycle(references.LargeMapType)
	drainWaterfallInstance(t, state)
	old := state.activeWaterfall
	oldPhase := state.outdoorActorContinuation
	state.cancelOutdoorActorPhase()
	require.Nil(t, state.activeWaterfall)
	require.True(t, state.ProcessWaterfallHazard(nil))
	replacement := state.activeWaterfall
	require.NotSame(t, old, replacement)
	calls, date := state.GetRNG().Calls(), state.DateTime
	state.cancelWaterfallCascade(old)
	state.completeFirstWaterfallActorTail(old)
	state.continueOutdoorActorPhase(oldPhase)
	assert.Same(t, replacement, state.activeWaterfall)
	assert.True(t, state.EventQueueManager.HasOnlyWorkflow(waterfallWorkflowName))
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Equal(t, date, state.DateTime)
	state.EventQueueManager.Shutdown()
}
