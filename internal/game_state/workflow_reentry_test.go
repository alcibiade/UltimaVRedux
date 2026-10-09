package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestHoleUpRejectsLiveGameplayWorkflowBeforeFeedback(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	reserveTestWorkflow(state, "unrelated_workflow")
	callbacks.Reset()

	assert.False(t, state.ActionHoleUpLargeMap())
	assert.False(t, state.ActionHoleUpLargeMapWithDebug(1, NoGuardAssigned, false, false))
	assert.Empty(t, callbacks.Messages)
}

func TestShipRepairDuplicateEntryDoesNotMutateOrReserveAgain(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	frigate.GetVehicleDetails().SetSailState(map_units.SailFurled)
	state.PartyVehicle = *frigate

	require.True(t, state.ActionHoleUpRepairShip())
	messages := len(callbacks.Messages)
	rngCalls := state.GetRNG().Calls()
	require.NotPanics(t, state.StartShipRepairWorkflow)
	assert.False(t, state.ActionHoleUpRepairShip())
	assert.Equal(t, messages, len(callbacks.Messages))
	assert.Equal(t, rngCalls, state.GetRNG().Calls())
	assert.Equal(t, 1, callbacks.BlockInputCalls)
}

func TestWorkflowEntryGuardsRunBeforeCommandMutation(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		invoke       func(*GameState) bool
		directStart  func(*GameState)
	}{
		{
			name:         "climbing fall",
			workflowName: climbingFallWorkflowName,
			invoke: func(state *GameState) bool {
				return state.ActionKlimbLargeMap(references.Up)
			},
			directStart: func(state *GameState) { state.StartClimbingFallWorkflow([]int{0}) },
		},
		{
			name:         "small cannon",
			workflowName: smallCannonWorkflowName,
			invoke:       func(state *GameState) bool { return state.ActionFireSmallMap() },
			directStart: func(state *GameState) {
				state.StartSmallMapCannonFireWorkflow(
					references.Position{}, references.Position{}, references.Up, false, nil,
				)
			},
		},
		{
			name:         "ship cannon",
			workflowName: shipCannonWorkflowName,
			invoke: func(state *GameState) bool {
				return state.ActionFireLargeMap(references.Left)
			},
			directStart: func(state *GameState) {
				state.StartShipCannonFireWorkflow(ShipCannonResult{Outcome: ShipCannonMissed})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
			reserveTestWorkflow(state, test.workflowName)
			callbacks.Reset()
			position := state.MapState.PlayerLocation.Position
			rngCalls := state.GetRNG().Calls()

			require.NotPanics(t, func() { test.directStart(state) })
			assert.False(t, test.invoke(state))
			assert.Equal(t, position, state.MapState.PlayerLocation.Position)
			assert.Equal(t, rngCalls, state.GetRNG().Calls())
			assert.Empty(t, callbacks.Messages)
		})
	}
}

func reserveTestWorkflow(state *GameState, name string) {
	state.EventQueueManager.Workflow(name).
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).
		Wait(1000).
		Start()
}
