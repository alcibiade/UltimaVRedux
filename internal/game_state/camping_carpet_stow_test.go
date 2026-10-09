package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func mountCampingTestCarpet(state *GameState, carried uint16) {
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, carried)
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle,
		state.MapState.PlayerLocation.Position, state.MapState.PlayerLocation.Floor)
}

func TestCampingCarpetStowsOnlyAtCommitAndCannotOverflow(t *testing.T) {
	for _, stage := range []string{"cancel before start", "full at entry", "full after prompt", "commit"} {
		t.Run(stage, func(t *testing.T) {
			state, _ := newCampingAmbushTestState(t)
			mountCampingTestCarpet(state, 2)
			state.SystemCallbacks.Flow.Rest.RequestCampDuration = func() {}
			if stage == "full at entry" {
				state.PartyState.Inventory.QuestItems.Set(references.Carpet, party_state.ItemQuantitySmallMax)
			}
			accepted := state.ActionHoleUpLargeMap()
			assert.Equal(t, stage != "full at entry", accepted)
			assert.Equal(t, references.CarpetVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType, "prompt never stows")
			if !accepted {
				return
			}
			date, calls := state.DateTime, state.GetRNG().Calls()
			require.True(t, state.ActionHoleUpLargeMapWithDebug(1, NoGuardAssigned, false, false))
			if stage == "cancel before start" {
				state.EventQueueManager.Shutdown()
				assert.Equal(t, uint16(2), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
				assert.Equal(t, references.CarpetVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
				assert.Equal(t, date, state.DateTime)
				assert.Equal(t, calls, state.GetRNG().Calls())
				return
			}
			if stage == "full after prompt" {
				state.PartyState.Inventory.QuestItems.Set(references.Carpet, party_state.ItemQuantitySmallMax)
			}
			state.EventQueueManager.Update()
			if stage == "full after prompt" {
				assert.Equal(t, references.CarpetVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
				assert.Equal(t, uint16(party_state.ItemQuantitySmallMax), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
				assert.Nil(t, state.CampingScene)
				assert.Equal(t, date, state.DateTime)
				assert.Equal(t, calls, state.GetRNG().Calls())
			} else {
				assert.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
				assert.Equal(t, uint16(3), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
			}
			state.EventQueueManager.Shutdown()
		})
	}
}

func TestCampingCarpetSurvivesActualAmbushAndResurrection(t *testing.T) {
	for _, defeat := range []bool{false, true} {
		state, _ := newCampingAmbushTestState(t)
		mountCampingTestCarpet(state, 0)
		origin, originUnits := state.MapState.PlayerLocation, state.CurrentNPCAIController.GetNpcs()
		installAutomaticAnyKey(state)
		require.True(t, state.ActionHoleUpLargeMapWithDebug(2, NoGuardAssigned, false, true))
		runCampingWorkflowToCompletion(t, state)
		require.True(t, state.HasPendingOverworldContact())
		require.NoError(t, state.ResolvePendingOverworldContact())
		require.NotNil(t, state.CombatScene)
		require.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
		require.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
		if defeat {
			for index := range party_state.MaxActiveParty {
				pc := state.PartyState.GetCharacter(index)
				pc.SetHp(0)
				pc.MarkAsDead()
			}
			state.EndCombat(CombatDefeat)
			review, err := state.ReviewDeathResurrection()
			require.NoError(t, err)
			_, err = state.CommitDeathResurrection(review.Fingerprint)
			require.NoError(t, err)
		} else {
			state.EndCombat(CombatVictory)
		}
		assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
		assert.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
		assert.Nil(t, originUnits.GetVehicleAtPositionAndFloorOrNil(origin.Position, origin.Floor), "no parked duplicate at camp")
	}
}
