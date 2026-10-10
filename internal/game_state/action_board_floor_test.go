package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestActionBoardSelectsAndRemovesVehicleOnCurrentFloor(t *testing.T) {
	position := references.Position{X: 12, Y: 34}
	otherFloorHorse := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, 0)
	otherFloorHorse.GetVehicleDetails().IsOwnedByNPC = true
	currentFloorHorse := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, 1)
	otherFloorHorse.SetPos(position)
	currentFloorHorse.SetPos(position)
	controller := &ecosystemTestController{
		units: map_units.MapUnits{otherFloorHorse, currentFloorHorse},
	}
	state, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(1).
		WithPlayerAt(int(position.X), int(position.Y)).
		WithSystemCallbacks().
		Build()
	state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	state.CurrentNPCAIController = controller
	mockCallbacks.Reset()

	require.True(t, state.ActionBoard())
	assert.Equal(t, references.HorseVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Equal(t, references.FloorNumber(1), state.PartyVehicle.Floor())
	assert.False(t, state.PartyVehicle.GetVehicleDetails().IsOwnedByNPC)
	assert.False(t, mockCallbacks.HorseRefusalShown)
	assert.Equal(t, []int{1}, mockCallbacks.TimeAdvanced())

	remainingHorse := controller.units.GetVehicleAtPositionAndFloorOrNil(position, 0)
	require.NotNil(t, remainingHorse)
	assert.Same(t, otherFloorHorse, remainingHorse)
	assert.True(t, remainingHorse.GetVehicleDetails().IsOwnedByNPC)
	assert.Nil(t, controller.units.GetVehicleAtPositionAndFloorOrNil(position, 1))
}

func TestBoardVehiclePanicsWhenCurrentFloorVehicleCannotBeRemoved(t *testing.T) {
	position := references.Position{X: 12, Y: 34}
	vehicle := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, 1)
	state := &GameState{ //nolint:exhaustruct // The invariant check only needs vehicle storage and player location.
		PartyVehicle:           map_units.NewNPCFriendlyVehiceNoVehicle(),
		CurrentNPCAIController: &ecosystemTestController{units: map_units.MapUnits{}},
		MapState: map_state.MapState{ //nolint:exhaustruct // PlayerLocation is the only map state used by boarding.
			PlayerLocation: references.PlayerLocation{Position: position, Floor: 1},
		},
	}

	assert.PanicsWithValue(
		t,
		"BoardVehicle: vehicle removal failed at X=12,Y=34,floor=1",
		func() { state.BoardVehicle(*vehicle) },
	)
}

func TestBoardVehicleRefusesFullFrigateWithoutMutation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.MainFloor).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	units := state.CurrentNPCAIController.GetNpcs()
	units.ClearAllMapUnits()
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	stored := units.InsertVehicle(*frigate)
	require.NotNil(t, stored)
	stored.GetVehicleDetails().SetSkiffQuantity(MaxFrigateSkiffs)
	skiff := map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	state.PartyVehicle = *skiff
	sourceBefore := state.PartyVehicle.GetVehicleDetails().Snapshot()
	targetBefore := stored.GetVehicleDetails().Snapshot()

	result := state.BoardVehicle(*stored)

	assert.Equal(t, BoardVehicleResultSkiffCapacity, result)
	assert.Equal(t, sourceBefore, state.PartyVehicle.GetVehicleDetails().Snapshot())
	assert.Equal(t, targetBefore, stored.GetVehicleDetails().Snapshot())
	require.Len(t, *units, 1)
	assert.Same(t, stored, units.GetVehicleAtPositionAndFloorOrNil(
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	))
	_, err := state.CaptureSaveSnapshot()
	require.NoError(t, err, "capacity refusal must leave representable vehicle state")
}

func TestActionBoardRefusesFullFrigateWithoutMutationOrTime(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.MainFloor).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	units := state.CurrentNPCAIController.GetNpcs()
	units.ClearAllMapUnits()
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	stored := units.InsertVehicle(*frigate)
	require.NotNil(t, stored)
	stored.GetVehicleDetails().SetSkiffQuantity(MaxFrigateSkiffs)
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	sourceBefore := state.PartyVehicle.GetVehicleDetails().Snapshot()
	targetBefore := stored.GetVehicleDetails().Snapshot()

	result := state.ActionBoard()

	assert.False(t, result)
	assert.Contains(t, callbacks.Messages, "No room for another skiff.")
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Equal(t, sourceBefore, state.PartyVehicle.GetVehicleDetails().Snapshot())
	assert.Equal(t, targetBefore, stored.GetVehicleDetails().Snapshot())
	require.Len(t, *units, 1)
	assert.Same(t, stored, units.GetVehicleAtPositionAndFloorOrNil(
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	))
	_, err := state.CaptureSaveSnapshot()
	require.NoError(t, err, "capacity refusal must leave representable vehicle state")
}

func TestActionBoardFrigateTransfersVehicleAndUsesResultingSkiffCount(t *testing.T) {
	tests := []struct {
		name             string
		partyVehicle     references.VehicleType
		initialSkiffs    int
		initialCarpets   uint16
		wantSkiffs       int
		wantCarpets      uint16
		wantSkiffWarning bool
	}{
		{name: "on foot warns for empty frigate", partyVehicle: references.NoPartyVehicle, wantSkiffWarning: true},
		{name: "on foot preserves existing skiffs", partyVehicle: references.NoPartyVehicle, initialSkiffs: 2, wantSkiffs: 2},
		{name: "skiff is carried aboard", partyVehicle: references.SkiffVehicle, initialSkiffs: 1, wantSkiffs: 2},
		{name: "carpet is returned to inventory", partyVehicle: references.CarpetVehicle, initialCarpets: 2, wantCarpets: 3, wantSkiffWarning: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			position := references.Position{X: 50, Y: 50}
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithFloor(references.MainFloor).
				WithPlayerAt(int(position.X), int(position.Y)).
				WithSystemCallbacks().
				Build()
			units := state.CurrentNPCAIController.GetNpcs()
			units.ClearAllMapUnits()
			frigate := map_units.NewNPCFriendlyVehiceNewRef(
				references.FrigateVehicle,
				position,
				state.MapState.PlayerLocation.Floor,
			)
			frigate.GetVehicleDetails().SetHullHP(37)
			frigate.GetVehicleDetails().SetPartyVehicleDirection(references.Left)
			frigate.GetVehicleDetails().SetSkiffQuantity(test.initialSkiffs)
			stored := units.InsertVehicle(*frigate)
			require.NotNil(t, stored)
			wantFrigateState := stored.GetVehicleDetails().Snapshot()
			state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
				test.partyVehicle,
				position,
				state.MapState.PlayerLocation.Floor,
			)
			state.PartyState.Inventory.QuestItems.Set(references.Carpet, test.initialCarpets)

			require.True(t, state.ActionBoard())

			wantFrigateState.SkiffQuantity = test.wantSkiffs
			assert.Equal(t, wantFrigateState, state.PartyVehicle.GetVehicleDetails().Snapshot())
			assert.Equal(t, test.wantCarpets, state.PartyState.Inventory.QuestItems.Get(references.Carpet))
			assert.Nil(t, units.GetVehicleAtPositionAndFloorOrNil(position, state.MapState.PlayerLocation.Floor))
			if test.wantSkiffWarning {
				assert.Contains(t, callbacks.Messages, "WARNING: NO SKIFFS ON BOARD!")
			} else {
				assert.NotContains(t, callbacks.Messages, "WARNING: NO SKIFFS ON BOARD!")
			}
			assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
		})
	}
}
