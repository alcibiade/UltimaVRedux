package game_state

import (
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

func TestSkiffDeepWaterProgrammaticMovementRetainsEveryHit(t *testing.T) {
	state, callbacks := newOutdoorFireballFixture(t, 0, references.SkiffVehicle)
	position := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)
	const seed = uint64(12345)
	state.SetRandomSeed(seed)
	want := deterministic_random.New(seed)
	wantHP := 100
	for range 3 {
		wantHP -= want.Intn(8) + 1
	}
	parentFinished := false
	state.EventQueueManager.Workflow("programmatic_skiff_movement").
		Priority(event_queue.PriorityNormal).Sequential().First(func() {
		for range 3 {
			state.ProcessSkiffDeepWaterDamage(&position)
		}
	}).While(func() bool {
		return state.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName)
	}, func() {}, 1).Then(func() { parentFinished = true }).Start()

	require.NotPanics(t, func() { drainGameplayWorkflows(t, state) })
	assert.True(t, parentFinished, "damage must preempt the normal-priority caller")
	assert.Equal(t, uint16(wantHP), state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Equal(t, want.Calls(), state.GetRNG().Calls())
	assert.Equal(t, 3, countMessage(callbacks.Messages, "Rough seas!"))
	assert.Zero(t, state.pendingSkiffDeepWaterHits)
	assert.False(t, callbacks.InputBlocked)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestSkiffDeepWaterInlineSequenceHasNoPriorityDeadlock(t *testing.T) {
	state, callbacks := newOutdoorFireballFixture(t, 0, references.SkiffVehicle)
	position := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)
	completed := false
	state.EventQueueManager.Workflow("critical_skiff_travel").
		Priority(event_queue.PriorityCritical).Sequential().BlockInput().First(func() {}).
		ThenSequence(state.SkiffDeepWaterDamageSequence(position)).
		Then(func() { completed = true }).Finally(state.ClearSkiffDeepWaterEffects).Start()

	drainGameplayWorkflows(t, state)
	assert.True(t, completed)
	assert.Less(t, state.PartyState.GetCharacter(0).GetCurrentHp(), uint16(100))
	assert.Equal(t, 1, countMessage(callbacks.Messages, "Rough seas!"))
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, state.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName))
}

func TestSkiffDeepWaterQueuedTerminalHitsHandoffOnce(t *testing.T) {
	for _, test := range []struct {
		name        string
		firstLethal bool
	}{
		{name: "first hit kills", firstLethal: true},
		{name: "surviving hit before fatal hit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newOutdoorFireballFixture(t, 0, references.SkiffVehicle)
			position := state.MapState.PlayerLocation.Position
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)
			const seed = uint64(12345)
			state.SetRandomSeed(seed)
			firstDamage := deterministic_random.New(seed).Intn(8) + 1
			member := state.PartyState.GetCharacter(0)
			member.SetHp(uint16(firstDamage + 1))
			hitsBeforeRecovery := 2
			if test.firstLethal {
				member.SetHp(1)
				hitsBeforeRecovery = 1
			}
			deaths, castleMessages := 0, 0
			addMessage := state.SystemCallbacks.Message.AddRowStr
			state.SystemCallbacks.Message.AddRowStr = func(message string) {
				addMessage(message)
				if message == "Rough seas!" && state.MapState.PlayerLocation.Location == references.LordBritishsCastle {
					castleMessages++
				}
			}
			recovered := false
			var recoveryRNG uint64
			state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
				deaths++
				require.Zero(t, state.pendingSkiffDeepWaterHits, "the old batch must close before invoking recovery")
				require.EqualValues(t, hitsBeforeRecovery, state.GetRNG().Calls(), "all preceding live hits commit once")
				state.ProcessSkiffDeepWaterDamage(&position)
				require.Zero(t, state.pendingSkiffDeepWaterHits, "a dead party cannot admit another rough-seas request")
				state.EventQueueManager.Workflow("rough_seas_test_recovery").
					Priority(event_queue.PriorityCritical).Sequential().BlockInput().First(func() {
					recoverRoughSeasTestPartyAtCastle(t, state)
					recoveryRNG = state.GetRNG().Calls()
					recovered = true
				}).Wait(100).Start()
			}
			for range hitsBeforeRecovery + 1 {
				state.ProcessSkiffDeepWaterDamage(&position)
			}
			drainGameplayWorkflows(t, state)
			require.True(t, recovered)
			assert.Equal(t, 1, deaths)
			assert.Equal(t, hitsBeforeRecovery, countMessage(callbacks.Messages, "Rough seas!"))
			assert.Zero(t, castleMessages)
			assert.Equal(t, recoveryRNG, state.GetRNG().Calls(), "old queued hits cannot draw after resurrection")
			assert.Equal(t, uint16(100), member.GetCurrentHp(), "old damage must not reach the revived party")
			assert.Equal(t, references.LordBritishsCastle, state.MapState.PlayerLocation.Location)
			assert.Zero(t, state.pendingSkiffDeepWaterHits)
			assert.False(t, callbacks.InputBlocked)
			assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
			assert.True(t, state.CanAcceptGameplayCommands())
			state.ElapsedMs += 1000
			state.EventQueueManager.Update()
			assert.Equal(t, recoveryRNG, state.GetRNG().Calls())
			assert.Equal(t, uint16(100), member.GetCurrentHp())
			require.True(t, state.ActionPass(), "the recovered party can accept a new manual command")
			assert.Contains(t, callbacks.CommandPrompts, "Pass")
		})
	}
}

func TestSkiffDeepWaterPostRecoveryRequestStartsNewBatch(t *testing.T) {
	state, callbacks := newOutdoorFireballFixture(t, 0, references.SkiffVehicle)
	position := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)
	member := state.PartyState.GetCharacter(0)
	member.SetHp(1)
	newPosition := references.Position{X: position.X + 1, Y: position.Y}
	const recoveredSeed = uint64(54321)
	wantDamage := deterministic_random.New(recoveredSeed).Intn(8) + 1
	var hitLocations []references.PlayerLocation
	addMessage := state.SystemCallbacks.Message.AddRowStr
	state.SystemCallbacks.Message.AddRowStr = func(message string) {
		addMessage(message)
		if message == "Rough seas!" {
			hitLocations = append(hitLocations, state.MapState.PlayerLocation)
		}
	}
	deaths := 0
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		deaths++
		require.Zero(t, state.pendingSkiffDeepWaterHits)
		state.EventQueueManager.Workflow("rough_seas_test_recovery").
			Priority(event_queue.PriorityCritical).Sequential().BlockInput().First(func() {
			recoverRoughSeasTestPartyAtCastle(t, state)
		}).Wait(100).Then(func() {
			// Simulate a newly admitted journey while the old high-priority
			// hazard owner remains preempted by recovery. Its tail must not
			// consume this fresh request as an old-batch decrement or replay.
			require.True(t, state.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName))
			state.TransitionToLocation(references.BritanniaUnderworld, 0, newPosition, TransitionFullLocation, false)
			state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, newPosition, 0)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &newPosition, indexes.Water1)
			state.SetRandomSeed(recoveredSeed)
			state.ProcessSkiffDeepWaterDamage(&newPosition)
			require.Equal(t, 1, state.pendingSkiffDeepWaterHits)
		}).Start()
	}
	state.ProcessSkiffDeepWaterDamage(&position)
	state.ProcessSkiffDeepWaterDamage(&position)
	for update := 0; update < 200 && state.EventQueueManager.HasActiveWorkflows(); update++ {
		state.ElapsedMs += 1000
		state.EventQueueManager.Update()
		require.GreaterOrEqual(t, state.pendingSkiffDeepWaterHits, 0, "old completion cannot consume a fresh batch's count")
	}
	require.False(t, state.EventQueueManager.HasActiveWorkflows())
	assert.Equal(t, 1, deaths)
	assert.Equal(t, uint16(100-wantDamage), member.GetCurrentHp())
	assert.EqualValues(t, 1, state.GetRNG().Calls(), "only the newly admitted journey may damage the recovered party")
	require.Len(t, hitLocations, 2)
	assert.Equal(t, references.BritanniaUnderworld, hitLocations[0].Location)
	assert.Equal(t, position, hitLocations[0].Position)
	assert.Equal(t, references.BritanniaUnderworld, hitLocations[1].Location)
	assert.Equal(t, newPosition, hitLocations[1].Position)
	assert.Zero(t, state.pendingSkiffDeepWaterHits)
	assert.False(t, callbacks.InputBlocked)
	assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
}

func recoverRoughSeasTestPartyAtCastle(t *testing.T, state *GameState) {
	t.Helper()
	require.True(t, state.PartyState.IsAllPartyDead())
	require.True(t, state.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName), "critical recovery must preempt the old hazard owner")
	state.ResurrectCharacterWithFullRecovery(0)
	state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	state.TransitionToLocation(
		state.GetResurrectionLocation(), state.GetResurrectionFloor(), state.GetResurrectionPosition(), TransitionFullLocation, false,
	)
	require.False(t, state.PartyState.IsAllPartyDead())
	require.Equal(t, references.LordBritishsCastle, state.MapState.PlayerLocation.Location)
}

func TestSkiffDeepWaterRequestDuringInputReleaseRetainsDamage(t *testing.T) {
	state, callbacks := newOutdoorFireballFixture(t, 0, references.SkiffVehicle)
	position := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.Water1)
	const seed = uint64(12345)
	state.SetRandomSeed(seed)
	want := deterministic_random.New(seed)
	firstDamage, secondDamage := want.Intn(8)+1, want.Intn(8)+1
	held := true
	state.SystemCallbacks.Visual.IsAnyKeyPressed = func() bool { return held }
	state.ProcessSkiffDeepWaterDamage(&position)
	for range 20 {
		state.ElapsedMs += 1000
		state.EventQueueManager.Update()
	}
	require.True(t, state.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName), "old name remains reserved through key release")
	require.True(t, callbacks.InputBlocked)
	require.Zero(t, state.pendingSkiffDeepWaterHits, "first damage chain has finished")
	require.Equal(t, uint16(100-firstDamage), state.PartyState.GetCharacter(0).GetCurrentHp())

	state.ProcessSkiffDeepWaterDamage(&position)
	require.Equal(t, 1, state.pendingSkiffDeepWaterHits)
	held = false
	drainGameplayWorkflows(t, state)
	assert.Equal(t, uint16(100-firstDamage-secondDamage), state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Equal(t, want.Calls(), state.GetRNG().Calls())
	assert.Equal(t, 2, countMessage(callbacks.Messages, "Rough seas!"))
	assert.Zero(t, state.pendingSkiffDeepWaterHits)
	assert.False(t, callbacks.InputBlocked)
}
