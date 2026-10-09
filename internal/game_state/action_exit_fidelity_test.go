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

func TestNearbyVehicleExitClassifier(t *testing.T) {
	tests := []struct {
		name        string
		tile        indexes.SpriteIndex
		object      references.VehicleType
		objectFloor references.FloorNumber
		raisedSails bool
		player      bool
		npc         bool
		want        bool
	}{
		{name: "effective walkable terrain", tile: indexes.Grass, want: true},
		{name: "impassable terrain", tile: indexes.SmallMountains},
		{name: "carpet object", tile: indexes.SmallMountains, object: references.CarpetVehicle, want: true},
		{name: "horse object", tile: indexes.SmallMountains, object: references.HorseVehicle, want: true},
		{name: "skiff object", tile: indexes.SmallMountains, object: references.SkiffVehicle, want: true},
		{name: "furled frigate object", tile: indexes.SmallMountains, object: references.FrigateVehicle, want: true},
		{name: "raised frigate object", tile: indexes.SmallMountains, object: references.FrigateVehicle, raisedSails: true},
		{name: "player object", tile: indexes.SmallMountains, player: true, want: true},
		{name: "ordinary NPC object", tile: indexes.Grass, npc: true},
		{name: "allowed object on another floor", tile: indexes.SmallMountains, object: references.HorseVehicle, objectFloor: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _ := newExitFidelityState(t, references.Position{X: 50, Y: 50})
			setAllExitTerrain(state, indexes.SmallMountains)
			candidate := state.MapState.PlayerLocation.Position.Up()
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &candidate, test.tile)

			switch {
			case test.player:
				reference := references.NPCReference{ //nolint:exhaustruct // Only identity and fixed placement matter here.
					Schedule: references.CreateNPCScheduledFixedOneLocation(candidate, test.objectFloor),
				}
				reference.SetKeyIndex(indexes.BasicAvatar)
				player := map_units.NewNPCFriendly(reference, 0)
				player.SetPos(candidate)
				player.SetFloor(test.objectFloor)
				*state.CurrentNPCAIController.GetNpcs() = append(*state.CurrentNPCAIController.GetNpcs(), player)
			case test.npc:
				reference := references.NPCReference{ //nolint:exhaustruct // Only identity and fixed placement matter here.
					Schedule: references.CreateNPCScheduledFixedOneLocation(candidate, test.objectFloor),
				}
				reference.SetKeyIndex(indexes.Guard1)
				npc := map_units.NewNPCFriendly(reference, 0)
				npc.SetPos(candidate)
				npc.SetFloor(test.objectFloor)
				*state.CurrentNPCAIController.GetNpcs() = append(*state.CurrentNPCAIController.GetNpcs(), npc)
			case test.object != references.NoPartyVehicle:
				vehicle := map_units.NewNPCFriendlyVehiceNewRef(test.object, candidate, test.objectFloor)
				if test.raisedSails {
					vehicle.GetVehicleDetails().SetSailState(map_units.SailRaised)
				}
				require.NotNil(t, state.CurrentNPCAIController.GetNpcs().InsertVehicle(*vehicle))
			}

			assert.Equal(t, test.want, state.hasNearbyVehicleExit())
		})
	}
}

func TestNearbyVehicleExitClassifierWrapsLargeMap(t *testing.T) {
	state, _ := newExitFidelityState(t, references.Position{X: 255, Y: 50})
	setAllExitTerrain(state, indexes.SmallMountains)
	wrappedRight := references.Position{X: 0, Y: 50}
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &wrappedRight, indexes.Grass)

	assert.True(t, state.hasNearbyVehicleExit())
}

func TestActionExitFrigatePriorityAndStatePreservation(t *testing.T) {
	tests := []struct {
		name             string
		nearbyLand       bool
		skiffs           int
		carpets          uint16
		wantSuccess      bool
		wantPartyVehicle references.VehicleType
		wantStoredSkiffs int
		wantCarpets      uint16
		wantMessage      string
	}{
		{name: "nearby land wins", nearbyLand: true, skiffs: 1, carpets: 1, wantSuccess: true, wantPartyVehicle: references.NoPartyVehicle, wantStoredSkiffs: 1, wantCarpets: 1},
		{name: "skiff wins without land", skiffs: 1, carpets: 1, wantSuccess: true, wantPartyVehicle: references.SkiffVehicle, wantCarpets: 1},
		{name: "carpet follows empty skiffs", carpets: 1, wantSuccess: true, wantPartyVehicle: references.CarpetVehicle},
		{name: "no fallback refuses", wantPartyVehicle: references.FrigateVehicle, wantMessage: "No skiffs on board!"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			position := references.Position{X: 50, Y: 50}
			state, callbacks := newExitFidelityState(t, position)
			setAllExitTerrain(state, indexes.Water1)
			if test.nearbyLand {
				land := position.Up()
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &land, indexes.Grass)
			}
			frigate := map_units.NewNPCFriendlyVehiceNewRef(
				references.FrigateVehicle,
				position,
				state.MapState.PlayerLocation.Floor,
			)
			frigate.GetVehicleDetails().SetHullHP(0)
			frigate.GetVehicleDetails().SetPartyVehicleDirection(references.Left)
			frigate.GetVehicleDetails().SetSkiffQuantity(test.skiffs)
			state.PartyVehicle = *frigate
			state.PartyState.Inventory.QuestItems.Set(references.Carpet, test.carpets)
			before := state.PartyVehicle.GetVehicleDetails().Snapshot()

			assert.Equal(t, test.wantSuccess, state.ActionExit())
			assert.Equal(t, test.wantPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
			assert.Equal(t, test.wantCarpets, state.PartyState.Inventory.QuestItems.Get(references.Carpet))
			if !test.wantSuccess {
				assert.Equal(t, before, state.PartyVehicle.GetVehicleDetails().Snapshot())
				assert.Empty(t, *state.CurrentNPCAIController.GetNpcs())
				assert.Contains(t, callbacks.Messages, test.wantMessage)
				assert.Empty(t, callbacks.TimeAdvanced())

				return
			}

			stored := state.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
				position,
				state.MapState.PlayerLocation.Floor,
			)
			require.NotNil(t, stored)
			wantStored := before
			wantStored.SkiffQuantity = test.wantStoredSkiffs
			assert.Equal(t, wantStored, stored.GetVehicleDetails().Snapshot())
			assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
		})
	}
}

func TestActionExitVehicleSpecificTerrainRules(t *testing.T) {
	tests := []struct {
		name             string
		vehicle          references.VehicleType
		currentTile      indexes.SpriteIndex
		nearbyLand       bool
		sameSquareShip   bool
		wantSuccess      bool
		wantPartyVehicle references.VehicleType
		wantMessage      string
	}{
		{name: "carpet accepts current walkable underlay", vehicle: references.CarpetVehicle, currentTile: indexes.Grass, wantSuccess: true},
		{name: "carpet accepts left bed underlay", vehicle: references.CarpetVehicle, currentTile: indexes.LeftBed, wantSuccess: true},
		{name: "carpet accepts right bed underlay", vehicle: references.CarpetVehicle, currentTile: indexes.RightBed, wantSuccess: true},
		{name: "carpet accepts nearby exit", vehicle: references.CarpetVehicle, currentTile: indexes.Water1, nearbyLand: true, wantSuccess: true},
		{name: "carpet refuses without either exit", vehicle: references.CarpetVehicle, currentTile: indexes.Water1, wantPartyVehicle: references.CarpetVehicle, wantMessage: "No land nearby!"},
		{name: "skiff requires nearby exit", vehicle: references.SkiffVehicle, currentTile: indexes.Grass, wantPartyVehicle: references.SkiffVehicle, wantMessage: "No land nearby!"},
		{name: "skiff accepts ordinary bridge index", vehicle: references.SkiffVehicle, currentTile: indexes.Bridge, nearbyLand: true, wantSuccess: true},
		{name: "skiff rejects horizontal bridge underlay", vehicle: references.SkiffVehicle, currentTile: indexes.TrollBridgeHoriz, nearbyLand: true, wantPartyVehicle: references.SkiffVehicle, wantMessage: "Not here!"},
		{name: "skiff rejects vertical bridge underlay", vehicle: references.SkiffVehicle, currentTile: indexes.TrollBridgeVert, nearbyLand: true, wantPartyVehicle: references.SkiffVehicle, wantMessage: "Not here!"},
		{name: "same square frigate does not block skiff exit", vehicle: references.SkiffVehicle, currentTile: indexes.Water1, nearbyLand: true, sameSquareShip: true, wantSuccess: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			position := references.Position{X: 50, Y: 50}
			state, callbacks := newExitFidelityState(t, position)
			setAllExitTerrain(state, indexes.Water1)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, test.currentTile)
			if test.nearbyLand {
				land := position.Up()
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &land, indexes.Grass)
			}
			if test.sameSquareShip {
				ship := map_units.NewNPCFriendlyVehiceNewRef(
					references.FrigateVehicle,
					position,
					state.MapState.PlayerLocation.Floor,
				)
				require.NotNil(t, state.CurrentNPCAIController.GetNpcs().InsertVehicle(*ship))
			}
			state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
				test.vehicle,
				position,
				state.MapState.PlayerLocation.Floor,
			)

			assert.Equal(t, test.wantSuccess, state.ActionExit())
			assert.Equal(t, test.wantPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
			if test.wantSuccess && test.vehicle == references.CarpetVehicle {
				stored := state.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
					position,
					state.MapState.PlayerLocation.Floor,
				)
				require.NotNil(t, stored)
				assert.Equal(t, references.CarpetVehicle, stored.GetVehicleDetails().VehicleType)
			}
			if test.wantMessage != "" {
				assert.Contains(t, callbacks.Messages, test.wantMessage)
			}
			if test.sameSquareShip {
				assert.Len(t, *state.CurrentNPCAIController.GetNpcs(), 2)
			}
		})
	}
}

func TestActionExitRaisedSailsRefusesBeforeMutation(t *testing.T) {
	position := references.Position{X: 50, Y: 50}
	state, callbacks := newExitFidelityState(t, position)
	setAllExitTerrain(state, indexes.Grass)
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		position,
		state.MapState.PlayerLocation.Floor,
	)
	frigate.GetVehicleDetails().SetHullHP(17)
	frigate.GetVehicleDetails().SetSkiffQuantity(2)
	frigate.GetVehicleDetails().SetSailState(map_units.SailRaised)
	state.PartyVehicle = *frigate
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	before := state.PartyVehicle.GetVehicleDetails().Snapshot()

	assert.False(t, state.ActionExit())
	assert.Equal(t, before, state.PartyVehicle.GetVehicleDetails().Snapshot())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Empty(t, *state.CurrentNPCAIController.GetNpcs())
	assert.Contains(t, callbacks.Messages, "Under sail!")
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestActionExitFullMapDoesNotLoseVehicle(t *testing.T) {
	position := references.Position{X: 50, Y: 50}
	state, callbacks := newExitFidelityState(t, position)
	setAllExitTerrain(state, indexes.Grass)
	units := state.CurrentNPCAIController.GetNpcs()
	for index := range map_units.MaximumNpcsPerMap {
		vehiclePosition := references.Position{X: references.Coordinate(index + 1), Y: 1}
		vehicle := map_units.NewNPCFriendlyVehiceNewRef(
			references.HorseVehicle,
			vehiclePosition,
			state.MapState.PlayerLocation.Floor,
		)
		require.NotNil(t, units.InsertVehicle(*vehicle))
	}
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		references.CarpetVehicle,
		position,
		state.MapState.PlayerLocation.Floor,
	)
	before := state.PartyVehicle.GetVehicleDetails().Snapshot()

	assert.False(t, state.ActionExit())
	assert.Equal(t, before, state.PartyVehicle.GetVehicleDetails().Snapshot())
	assert.Len(t, *units, map_units.MaximumNpcsPerMap)
	assert.Contains(t, callbacks.Messages, "Not here!")
	assert.Empty(t, callbacks.TimeAdvanced())
}

func newExitFidelityState(
	t *testing.T,
	position references.Position,
) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.MainFloor).
		WithPlayerAt(int(position.X), int(position.Y)).
		WithSystemCallbacks().
		Build()
	state.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
	callbacks.Reset()

	return state, callbacks
}

func setAllExitTerrain(state *GameState, tile indexes.SpriteIndex) {
	positions := append(
		[]references.Position{state.MapState.PlayerLocation.Position},
		state.MapState.PlayerLocation.Position.GetFourDirectionsWrapped(
			references.XLargeMapTiles,
			references.YLargeMapTiles,
		)...,
	)
	for _, position := range positions {
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, tile)
	}
}
