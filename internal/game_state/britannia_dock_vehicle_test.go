package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceVehicleAtBritanniaDockInsertsPendingVehicleAndActivatesInPlace(t *testing.T) {
	t.Parallel()

	state, _ := newDebugVehicleTestState()
	controller := newDebugVehicleLargeMapController(state)
	dockPosition := references.Position{X: 0x27, Y: 0xdd}
	state.GameReferences = &references.GameReferences{
		DockReferences: &references.DockReferences{Docks: []references.DockReference{
			{Location: references.Jhelom, Position: dockPosition},
		}},
	}
	state.LargeMapNPCAIController = map[references.World]*ai.NPCAIControllerLargeMap{
		references.OVERWORLD: controller,
	}

	stored, err := state.PlaceVehicleAtBritanniaDock(
		references.FrigateVehicle,
		references.Jhelom,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, dockPosition, stored.Pos())
	assert.Equal(t, references.MainFloor, stored.Floor())
	assert.Equal(t, map_units.MaxHullHP, stored.GetVehicleDetails().GetHullHP())
	assert.Equal(t, map_units.SailFurled, stored.GetVehicleDetails().GetSailState())
	assert.Equal(t, 2, stored.GetVehicleDetails().GetSkiffQuantity())
	assert.True(t, stored.GetVehicleDetails().IsPendingDelivery())

	state.MapState.PlayerLocation.Location = references.Jhelom
	assert.Nil(t, state.ActivatePendingBritanniaDockVehicle())
	assert.True(t, stored.GetVehicleDetails().IsPendingDelivery())

	state.MapState.PlayerLocation.Location = references.BritanniaUnderworld
	state.MapState.PlayerLocation.Floor = references.MainFloor
	state.TransitionToLocation(
		references.BritanniaUnderworld,
		references.MainFloor,
		state.MapState.PlayerLocation.Position,
		TransitionPositionOnly,
		false,
	)
	activated := controller.GetNpcs().GetVehicleAtPositionAndFloorOrNil(dockPosition, references.MainFloor)
	assert.Same(t, stored, activated)
	assert.False(t, stored.GetVehicleDetails().IsPendingDelivery())
	require.Len(t, *controller.GetNpcs(), 1, "activation must not insert a second vehicle")
}

func TestPlaceVehicleAtBritanniaDockRejectsOccupiedDockAtomically(t *testing.T) {
	t.Parallel()

	state, _ := newDebugVehicleTestState()
	controller := newDebugVehicleLargeMapController(state)
	dockPosition := references.Position{X: 0x97, Y: 0x15}
	state.GameReferences = &references.GameReferences{
		DockReferences: &references.DockReferences{Docks: []references.DockReference{
			{Location: references.Minoc, Position: dockPosition},
		}},
	}
	state.LargeMapNPCAIController = map[references.World]*ai.NPCAIControllerLargeMap{
		references.OVERWORLD: controller,
	}
	occupant := map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		dockPosition,
		references.MainFloor,
	)
	require.True(t, controller.GetNpcs().AddVehicle(*occupant))

	stored, err := state.PlaceVehicleAtBritanniaDock(
		references.FrigateVehicle,
		references.Minoc,
		true,
	)

	assert.ErrorIs(t, err, ErrDockVehiclePositionOccupied)
	assert.Nil(t, stored)
	require.Len(t, *controller.GetNpcs(), 1)
	assert.Same(t, (*controller.GetNpcs())[0], controller.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
		dockPosition,
		references.MainFloor,
	))
}
