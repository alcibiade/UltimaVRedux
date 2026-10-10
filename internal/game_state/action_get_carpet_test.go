package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestGetLordBritishMagicCarpetRemovesCurrentEntityAndAddsInventory(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithFloor(2).
		WithPlayerAt(14, 18).
		WithSystemCallbacks().
		Build()

	position := references.Position{X: 15, Y: 18}
	mapUnit := state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(position, 2)
	require.NotNil(t, mapUnit)
	carpet, ok := (*mapUnit).(*map_units.NPCFriendly)
	require.True(t, ok)
	require.Equal(t, references.LordBritishMagicCarpetNPC, carpet.MapUnitDetails().NPCNum)
	require.Equal(t, references.CarpetVehicle, carpet.GetVehicleDetails().VehicleType)
	callbacks.Reset()

	require.True(t, state.ActionGetSmallMap(references.Right))

	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(position, 2))
	assert.False(t, state.PartyState.IsNPCDead(
		references.LordBritishsCastle,
		references.LordBritishMagicCarpetNPC,
	))
	assert.Equal(t, []string{"A magic carpet!"}, callbacks.Messages)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
}

func TestGetPlayerDeployedMagicCarpetReturnsItToInventory(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(0).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	units := state.CurrentNPCAIController.GetNpcs()
	units.ClearAllMapUnits()
	position := state.MapState.PlayerLocation.Position
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Grass)
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	commit, review := state.BeginCarpetBoarding()
	require.Equal(t, CarpetBoardingReady, review.Result)
	require.Equal(t, CarpetBoardingSuccess, state.CommitCarpetBoarding(commit))
	require.True(t, state.ActionExit())
	dropped := units.GetVehicleAtPositionAndFloorOrNil(position, 0)
	require.NotNil(t, dropped)
	require.False(t, dropped.GetVehicleDetails().IsOwnedByNPC)
	state.MapState.PlayerLocation.Position = position.Left()
	callbacks.Reset()

	require.True(t, state.ActionGetSmallMap(references.Right))

	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Nil(t, units.GetVehicleAtPositionAndFloorOrNil(position, 0))
	assert.Equal(t, []string{"A magic carpet!"}, callbacks.Messages)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())

	callbacks.Reset()
	assert.False(t, state.ActionGetSmallMap(references.Right))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Equal(t, []string{"Nothing to get!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestGetLargeMapDismountedMagicCarpetReturnsItToInventoryOnce(t *testing.T) {
	for _, direction := range []references.Direction{references.Up, references.Down, references.Left, references.Right} {
		t.Run(direction.GetDirectionCompassName(), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithFloor(0).
				WithPlayerAt(50, 50).
				WithSystemCallbacks().
				Build()
			units := state.CurrentNPCAIController.GetNpcs()
			units.ClearAllMapUnits()
			boardingPosition := state.MapState.PlayerLocation.Position
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &boardingPosition, indexes.Grass)
			state.PartyState.Inventory.QuestItems.Set(references.Carpet, 3)
			commit, review := state.BeginCarpetBoarding()
			require.Equal(t, CarpetBoardingReady, review.Result)
			require.Equal(t, CarpetBoardingSuccess, state.CommitCarpetBoarding(commit))

			// Exit must place the carpet at the destination, not its original boarding position.
			position := references.Position{X: 150, Y: 150}
			state.SetPlayerPosition(position)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.Grass)
			require.True(t, state.ActionExit())
			require.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
			require.NotNil(t, units.GetVehicleAtPositionAndFloorOrNil(position, 0))
			require.Nil(t, units.GetVehicleAtPositionAndFloorOrNil(boardingPosition, 0))
			require.Equal(t, uint16(2), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
			otherCarpet := units.InsertVehicle(*map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, boardingPosition, 0))
			require.NotNil(t, otherCarpet)
			state.SetPlayerPosition(*direction.GetOppositeDirection().GetNewPositionInDirection(&position))
			callbacks.Reset()

			require.True(t, state.ActionGetLargeMap(direction))

			assert.Equal(t, uint16(3), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
			assert.Nil(t, units.GetVehicleAtPositionAndFloorOrNil(position, 0))
			assert.Same(t, otherCarpet, units.GetVehicleAtPositionAndFloorOrNil(boardingPosition, 0))
			assert.Equal(t, []string{"A magic carpet!"}, callbacks.Messages)
			assert.Equal(t, []int{1}, callbacks.TimeAdvanced())

			callbacks.Reset()
			assert.False(t, state.ActionGetLargeMap(direction))
			assert.Equal(t, uint16(3), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
			assert.Equal(t, []string{"Nothing to get!"}, callbacks.Messages)
			assert.Empty(t, callbacks.TimeAdvanced())
		})
	}
}

func TestGetAuthoredWorldCarpetRefusesCollection(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithFloor(0).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	units := state.CurrentNPCAIController.GetNpcs()
	units.ClearAllMapUnits()
	position := references.Position{X: 11, Y: 10}
	carpet := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, position, 0)
	carpet.GetVehicleDetails().IsOwnedByNPC = true
	require.NotNil(t, units.InsertVehicle(*carpet))
	callbacks.Reset()

	assert.False(t, state.ActionGetSmallMap(references.Right))

	assert.Zero(t, state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.NotNil(t, units.GetVehicleAtPositionAndFloorOrNil(position, 0))
	assert.Equal(t, []string{"Nothing to get!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
}
