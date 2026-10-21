package game_state

import (
	"errors"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type debugVehicleTestRNG struct{}

func (debugVehicleTestRNG) Intn(int) int { return 0 }

func newDebugVehicleTestState() (*GameState, *ecosystemTestController) {
	controller := &ecosystemTestController{units: make(map_units.MapUnits, 0)}
	state := &GameState{
		CurrentNPCAIController: controller,
		PartyVehicle:           map_units.NewNPCFriendlyVehiceNoVehicle(),
		MapState: map_state.MapState{
			PlayerLocation: references.PlayerLocation{
				Location: references.BritanniaUnderworld,
				Position: references.Position{X: 10, Y: 11},
				Floor:    references.MainFloor,
			},
		},
		DateTime:  datetime.UltimaDate{Day: 4, Hour: 5, Minute: 6, Turn: 7},
		ElapsedMs: 1234,
	}
	state.EventQueueManager = event_queue.NewEventQueueManager(func() int64 { return 0 }, nil)

	return state, controller
}

func newDebugVehicleLargeMapController(state *GameState) *ai.NPCAIControllerLargeMap {
	return ai.NewNPCAIControllerLargeMap(ai.NewNPCAIControllerLargeMapInput{
		World:          references.OVERWORLD,
		MapState:       &state.MapState,
		DebugOptions:   &state.DebugOptions,
		TheOdds:        &state.TheOdds,
		DateTime:       &state.DateTime,
		RandomProvider: debugVehicleTestRNG{},
	})
}

func TestSpawnDebugVehicleUsesDestinationMapAndDoesNotAdvanceTime(t *testing.T) {
	state, currentController := newDebugVehicleTestState()
	dockController := newDebugVehicleLargeMapController(state)
	dockPosition := references.Position{X: 44, Y: 55}
	state.GameReferences = &references.GameReferences{
		DockReferences: &references.DockReferences{Docks: []references.DockReference{
			{Location: references.Jhelom, Position: dockPosition},
		}},
	}
	state.LargeMapNPCAIController = map[references.World]*ai.NPCAIControllerLargeMap{
		references.OVERWORLD: dockController,
	}
	dateBefore, elapsedBefore := state.DateTime, state.ElapsedMs

	result, err := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
		VehicleType:  references.FrigateVehicle,
		Placement:    DebugVehicleSpawnAtDock,
		DockLocation: references.Jhelom,
	})
	require.NoError(t, err)

	assert.Empty(t, currentController.units, "dock insertion must not use the active controller")
	assert.Equal(t, 0, result.Before.UnitCount)
	assert.Equal(t, 1, result.After.UnitCount)
	require.NotNil(t, result.After.Vehicle)
	assert.Equal(t, references.FrigateVehicle, result.After.Vehicle.VehicleType)
	assert.Equal(t, 2, result.After.Vehicle.SkiffQuantity)
	assert.False(t, result.After.Vehicle.PendingDelivery, "debug dock spawns must be immediately active")
	assert.Equal(t, dockPosition, result.After.Position)
	assert.Equal(t, references.BritanniaUnderworld, result.After.Location)
	assert.Equal(t, references.Jhelom, result.After.DockLocation)
	assert.Equal(t, dateBefore, state.DateTime)
	assert.Equal(t, elapsedBefore, state.ElapsedMs)
}

func TestSpawnDebugVehicleAtAvatarIsFloorAware(t *testing.T) {
	state, controller := newDebugVehicleTestState()
	otherFloor := *map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		state.MapState.PlayerLocation.Position,
		references.Basement,
	)
	require.True(t, controller.units.AddVehicle(otherFloor))

	result, err := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
		VehicleType: references.SkiffVehicle,
		Placement:   DebugVehicleSpawnAtAvatar,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Before.UnitCount)
	assert.Nil(t, result.Before.Vehicle)
	assert.Equal(t, 2, result.After.UnitCount)
	require.NotNil(t, result.After.Vehicle)
	assert.Equal(t, references.MainFloor, result.After.Floor)
}

func TestSpawnDebugVehicleFailureIsAtomic(t *testing.T) {
	state, controller := newDebugVehicleTestState()
	occupant := map_units.NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, 1)
	occupant.SetPos(state.MapState.PlayerLocation.Position)
	occupant.SetFloor(state.MapState.PlayerLocation.Floor)
	controller.units = append(controller.units, occupant)

	result, err := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
		VehicleType: references.FrigateVehicle,
		Placement:   DebugVehicleSpawnAtAvatar,
	})

	assert.ErrorIs(t, err, ErrDebugVehiclePositionOccupied)
	assert.Equal(t, result.Before, result.After)
	require.Len(t, controller.units, 1)
	assert.Same(t, occupant, controller.units[0])
}

func TestForceBoardDebugVehicleReleasesCurrentVehicleAndPreservesFields(t *testing.T) {
	state, controller := newDebugVehicleTestState()
	dateBefore, elapsedBefore := state.DateTime, state.ElapsedMs
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	frigate.GetVehicleDetails().SetHullHP(37)
	frigate.GetVehicleDetails().SetSkiffQuantity(5)
	frigate.GetVehicleDetails().SetSailState(map_units.SailRaised)
	state.PartyVehicle = *frigate
	state.isSailing = true

	result, err := state.ForceBoardDebugVehicle(DebugForceBoardRequest{VehicleType: references.HorseVehicle})
	require.NoError(t, err)

	assert.Equal(t, references.FrigateVehicle, result.PartyBefore.VehicleType)
	assert.Equal(t, references.HorseVehicle, result.PartyAfter.VehicleType)
	assert.False(t, state.IsSailing())
	require.Len(t, controller.units, 1)
	released := controller.units[0].(*map_units.NPCFriendly)
	assert.Equal(t, 37, released.GetVehicleDetails().GetHullHP())
	assert.Equal(t, 5, released.GetVehicleDetails().GetSkiffQuantity())
	assert.Equal(t, map_units.SailRaised, released.GetVehicleDetails().GetSailState())
	assert.Equal(t, state.MapState.PlayerLocation.Position, released.Pos())
	assert.Equal(t, state.MapState.PlayerLocation.Floor, released.Floor())
	assert.Equal(t, dateBefore, state.DateTime)
	assert.Equal(t, elapsedBefore, state.ElapsedMs)
}

func TestForceBoardDebugVehicleAttachesCurrentSkiffToFrigate(t *testing.T) {
	state, controller := newDebugVehicleTestState()
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)

	result, err := state.ForceBoardDebugVehicle(DebugForceBoardRequest{VehicleType: references.FrigateVehicle})
	require.NoError(t, err)

	assert.Empty(t, controller.units, "attached skiff must not remain as a map vehicle")
	assert.Equal(t, references.FrigateVehicle, result.PartyAfter.VehicleType)
	assert.Equal(t, 3, result.PartyAfter.SkiffQuantity, "new frigate's two skiffs plus the attached party skiff")
}

func TestForceBoardDebugVehicleSupportsEveryBoardableType(t *testing.T) {
	for _, vehicleType := range []references.VehicleType{
		references.HorseVehicle,
		references.CarpetVehicle,
		references.SkiffVehicle,
		references.FrigateVehicle,
	} {
		t.Run(vehicleType.GetLowerCaseName(), func(t *testing.T) {
			state, _ := newDebugVehicleTestState()

			result, err := state.ForceBoardDebugVehicle(DebugForceBoardRequest{VehicleType: vehicleType})

			require.NoError(t, err)
			assert.Equal(t, vehicleType, result.PartyAfter.VehicleType)
			assert.Equal(t, references.Vehicle, state.PartyVehicle.NPCReference.GetNPCType())
			assert.NotZero(t, state.PartyVehicle.GetVehicleDetails().GetBoardedSpriteIndex())
			if vehicleType == references.FrigateVehicle {
				assert.True(t, state.PartyVehicle.GetVehicleDetails().HasAtLeastOneSkiff())
			}
		})
	}
}

func TestForceBoardDebugVehicleFailureDoesNotReplaceCurrentVehicle(t *testing.T) {
	state, controller := newDebugVehicleTestState()
	horse := map_units.NewNPCFriendlyVehiceNewRef(
		references.HorseVehicle,
		state.MapState.PlayerLocation.Position,
		state.MapState.PlayerLocation.Floor,
	)
	state.PartyVehicle = *horse
	controller.units = make(map_units.MapUnits, map_units.MaximumNpcsPerMap)
	for index := range controller.units {
		controller.units[index] = map_units.NewNPCFriendly(references.NPCReference{NpcType: references.NoStatedNpc}, index)
	}

	result, err := state.ForceBoardDebugVehicle(DebugForceBoardRequest{VehicleType: references.CarpetVehicle})

	assert.ErrorIs(t, err, ErrDebugVehicleMapFull)
	assert.Equal(t, result.PartyBefore, result.PartyAfter)
	assert.Equal(t, references.HorseVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Len(t, controller.units, map_units.MaximumNpcsPerMap)
}

func TestDebugVehicleRequestsRejectInvalidContexts(t *testing.T) {
	tests := []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
	}{
		{name: "dungeon", location: references.Deceit, floor: 0},
		{name: "combat", location: references.CombatLocation, floor: 0},
		{name: "camping", location: references.CampingLocation, floor: 0},
		{name: "invalid overworld floor", location: references.BritanniaUnderworld, floor: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, controller := newDebugVehicleTestState()
			state.MapState.PlayerLocation.Location = test.location
			state.MapState.PlayerLocation.Floor = test.floor

			_, spawnErr := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
				VehicleType: references.SkiffVehicle,
				Placement:   DebugVehicleSpawnAtAvatar,
			})
			_, boardErr := state.ForceBoardDebugVehicle(DebugForceBoardRequest{VehicleType: references.SkiffVehicle})
			state.GameReferences = &references.GameReferences{DockReferences: &references.DockReferences{}}
			_, dockErr := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
				VehicleType: references.SkiffVehicle, Placement: DebugVehicleSpawnAtDock, DockLocation: references.Jhelom,
			})

			assert.True(t,
				errors.Is(spawnErr, ErrDebugVehicleUnsupportedContext) || errors.Is(spawnErr, ErrDebugVehicleInvalidLocation),
				"SpawnDebugVehicle() error = %v", spawnErr,
			)
			assert.True(t,
				errors.Is(boardErr, ErrDebugVehicleUnsupportedContext) || errors.Is(boardErr, ErrDebugVehicleInvalidLocation),
				"ForceBoardDebugVehicle() error = %v", boardErr,
			)
			assert.True(t,
				errors.Is(dockErr, ErrDebugVehicleUnsupportedContext) || errors.Is(dockErr, ErrDebugVehicleInvalidLocation),
				"SpawnDebugVehicle(dock) error = %v", dockErr,
			)
			assert.Empty(t, controller.units)
			assert.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
		})
	}
}

func TestDebugVehicleRequestsRejectUnsupportedTypesAndDock(t *testing.T) {
	state, _ := newDebugVehicleTestState()

	_, spawnTypeErr := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
		VehicleType: references.HorseVehicle,
		Placement:   DebugVehicleSpawnAtAvatar,
	})
	_, boardTypeErr := state.ForceBoardDebugVehicle(DebugForceBoardRequest{VehicleType: references.NoPartyVehicle})
	assert.ErrorIs(t, spawnTypeErr, ErrDebugVehicleUnsupportedType)
	assert.ErrorIs(t, boardTypeErr, ErrDebugVehicleUnsupportedType)

	state.GameReferences = &references.GameReferences{
		DockReferences: &references.DockReferences{Docks: []references.DockReference{
			{Location: references.Jhelom, Position: references.Position{X: 1, Y: 2}},
		}},
	}
	_, dockErr := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
		VehicleType:  references.FrigateVehicle,
		Placement:    DebugVehicleSpawnAtDock,
		DockLocation: references.Britain,
	})
	assert.ErrorIs(t, dockErr, ErrDebugVehicleInvalidLocation)
}

func TestDebugVehicleRequestsRejectMismatchedLargeMapController(t *testing.T) {
	state, _ := newDebugVehicleTestState()
	state.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD)
	state.CurrentNPCAIController = newDebugVehicleLargeMapController(state)

	_, spawnErr := state.SpawnDebugVehicle(DebugVehicleSpawnRequest{
		VehicleType: references.SkiffVehicle,
		Placement:   DebugVehicleSpawnAtAvatar,
	})
	_, boardErr := state.ForceBoardDebugVehicle(DebugForceBoardRequest{VehicleType: references.SkiffVehicle})

	assert.ErrorIs(t, spawnErr, ErrDebugVehicleMapUnavailable)
	assert.ErrorIs(t, boardErr, ErrDebugVehicleMapUnavailable)
}
