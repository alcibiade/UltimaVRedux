package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func atlasFleetEntries(state *GameState) []journal.AtlasEntry {
	return journal.FilterAtlasEntries(state.CurrentAtlasDossier().Entries, journal.AtlasFilters{Places: journal.AtlasVesselPlaces})
}

func TestAtlasFleetCommittedUseNoBackfillAndStableIdentity(t *testing.T) {
	state, _ := newExitFidelityState(t, references.Position{X: 50, Y: 50})
	setAllExitTerrain(state, indexes.Grass)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	units := state.CurrentNPCAIController.GetNpcs()
	boat := units.InsertVehicle(*map_units.NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, state.MapState.PlayerLocation.Position, 0))
	require.NotNil(t, boat)
	boat.GetVehicleDetails().SetSkiffQuantity(0)
	require.Equal(t, BoardVehicleResultNoSkiffs, state.BoardVehicle(*boat), "warning is still committed boarding")
	parked, result := state.exitVehicle()
	require.Equal(t, exitVehicleSuccess, result)
	require.Zero(t, parked.GetVehicleDetails().VesselID)
	require.Zero(t, state.CaptureAtlasFleet().LastID)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.DateTime.Hour = 12
	beforeDate, beforeCalls := state.DateTime, state.GetRNG().Calls()
	require.Empty(t, atlasFleetEntries(state), "acquisition and projection cannot backfill")
	require.Equal(t, BoardVehicleResultNoSkiffs, state.BoardVehicle(*parked))
	id := state.PartyVehicle.GetVehicleDetails().VesselID
	require.NotZero(t, id)
	require.Empty(t, atlasFleetEntries(state), "occupied boat is not a live Avatar bypass")
	for x := 51; x < 54; x++ {
		state.MapState.PlayerLocation.Position.X = references.Coordinate(x)
		setAllExitTerrain(state, indexes.Grass)
		parked, result = state.exitVehicle()
		require.Equal(t, exitVehicleSuccess, result)
		require.Equal(t, id, parked.GetVehicleDetails().VesselID)
		entries := atlasFleetEntries(state)
		require.Len(t, entries, 1)
		position, ok := entries[0].PlotPosition()
		require.True(t, ok)
		require.Equal(t, x, position.X)
		require.Equal(t, "Frigate 1", entries[0].Name)
		require.False(t, entries[0].Survey.Measured)
		require.Equal(t, BoardVehicleResultNoSkiffs, state.BoardVehicle(*parked))
	}
	parked, result = state.exitVehicle()
	require.Equal(t, exitVehicleSuccess, result)
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
	require.Empty(t, atlasFleetEntries(state))
	for _, entry := range state.CurrentAtlasDossier().Entries {
		require.Zero(t, entry.Vessel.ID)
	}
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.MapState.PlayerLocation.Location = references.Deceit
	state.CurrentNPCAIController = nil
	require.Len(t, atlasFleetEntries(state), 1, "surface authority survives a different active controller")
	require.Equal(t, beforeDate, state.DateTime)
	require.Equal(t, beforeCalls, state.GetRNG().Calls())
	require.Equal(t, uint64(1), state.CaptureAtlasFleet().LastID)
}

func TestAtlasFleetSkiffLaunchAbsorptionAndSinking(t *testing.T) {
	state, _ := newExitFidelityState(t, references.Position{X: 50, Y: 50})
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	setAllExitTerrain(state, indexes.Water1)
	// Historical occupied boats acquire no fabricated history on hydration.
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, state.MapState.PlayerLocation.Position, 0)
	require.Zero(t, state.CaptureAtlasFleet().LastID)
	parked, result := state.exitVehicle()
	require.Equal(t, exitVehicleSuccess, result)
	shipID := parked.GetVehicleDetails().VesselID
	skiffID := state.PartyVehicle.GetVehicleDetails().VesselID
	require.NotZero(t, shipID)
	require.NotZero(t, skiffID)
	require.NotEqual(t, shipID, skiffID)
	require.Len(t, atlasFleetEntries(state), 1)
	require.Equal(t, BoardVehicleResultSuccess, state.BoardVehicle(*parked))
	require.Empty(t, atlasFleetEntries(state), "absorbed skiff leaves no parked ghost")
	require.Equal(t, shipID, state.PartyVehicle.GetVehicleDetails().VesselID)
	state.PartyVehicle.GetVehicleDetails().SetHullHP(0)
	hit := state.ResolveSourceHit()
	require.Equal(t, SourceHitEscapedBySkiff, hit.Outcome)
	escapeID := state.PartyVehicle.GetVehicleDetails().VesselID
	require.Greater(t, escapeID, skiffID)
	setAllExitTerrain(state, indexes.Grass)
	escape, result := state.exitVehicle()
	require.Equal(t, exitVehicleSuccess, result)
	entries := atlasFleetEntries(state)
	require.Len(t, entries, 1)
	require.Equal(t, escapeID, entries[0].Vessel.ID)
	require.Equal(t, "skiff", entries[0].Kind)
	// Slot reuse cannot inherit the removed vessel identity.
	require.True(t, state.CurrentNPCAIController.GetNpcs().RemoveNPCAtPositionAndFloor(escape.Pos(), escape.Floor()))
	fresh := state.CurrentNPCAIController.GetNpcs().InsertVehicle(*map_units.NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, escape.Pos(), 0))
	require.Zero(t, fresh.GetVehicleDetails().VesselID)
	require.Empty(t, atlasFleetEntries(state))
}

func TestAtlasFleetFailedTransitionsAndNonSurfaceExclusion(t *testing.T) {
	state, _ := newExitFidelityState(t, references.Position{X: 50, Y: 50})
	state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.FrigateVehicle, state.MapState.PlayerLocation.Position, 0)
	state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailRaised)
	_, result := state.exitVehicle()
	require.Equal(t, exitVehicleUnderSail, result)
	require.Zero(t, state.CaptureAtlasFleet().LastID)
	state.PartyVehicle.GetVehicleDetails().SetSailState(map_units.SailFurled)
	state.PartyVehicle.GetVehicleDetails().SetSkiffQuantity(0)
	setAllExitTerrain(state, indexes.Water1)
	_, result = state.exitVehicle()
	require.Equal(t, exitVehicleNoSkiffs, result)
	require.Zero(t, state.CaptureAtlasFleet().LastID)
	setAllExitTerrain(state, indexes.Grass)
	parked, result := state.exitVehicle()
	require.Equal(t, exitVehicleSuccess, result)
	parked.SetFloor(-1)
	require.Empty(t, atlasFleetEntries(state))
	parked.SetFloor(0)
	parked.GetVehicleDetails().MarkPendingDelivery()
	require.Empty(t, atlasFleetEntries(state))
	parked.GetVehicleDetails().ClearPendingDelivery()
	require.Len(t, atlasFleetEntries(state), 1)
	parked.GetVehicleDetails().SetSkiffQuantity(MaxFrigateSkiffs)
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.SkiffVehicle, state.MapState.PlayerLocation.Position, 0)
	before := state.CaptureAtlasFleet()
	require.Equal(t, BoardVehicleResultSkiffCapacity, state.BoardVehicle(*parked))
	require.Equal(t, before, state.CaptureAtlasFleet())
	require.Zero(t, state.PartyVehicle.GetVehicleDetails().VesselID)
	require.Len(t, atlasFleetEntries(state), 1)
}
