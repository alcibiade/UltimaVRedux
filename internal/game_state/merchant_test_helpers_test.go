package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newOpenMerchantTestState(
	t *testing.T,
	merchantID references.MerchantID,
) (*GameState, *map_units.NPCFriendly) {
	t.Helper()

	state, _ := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		Build()
	definition, ok := state.GameReferences.MerchantReferences.Definition(merchantID)
	require.True(t, ok, merchantID)
	state.MapState.PlayerLocation.Location = definition.Location

	referenceIndex := (int(definition.Location)-1)*party_state.NPcsPerLocation + definition.NPCIndex
	require.GreaterOrEqual(t, referenceIndex, 0)
	require.Less(t, referenceIndex, len(state.GameReferences.NPCReferences.Npcs))
	npcReference := state.GameReferences.NPCReferences.Npcs[referenceIndex]
	require.Equal(t, byte(definition.Family), npcReference.DialogNumber)
	npc := map_units.NewNPCFriendly(npcReference, definition.NPCIndex)
	npc.PlaceAtScheduleSlot(1)
	npc.SetVisible(true)
	state.MapState.PlayerLocation.Floor = npc.Floor()
	installMerchantTestNPC(t, state, npc)
	setMerchantDesiredSlot(t, state, npc, 1)

	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapUnrecruited
		character.SetStatus(party_state.Dead)
	}
	avatar := state.PartyState.GetCharacter(0)
	avatar.PartyMap = party_state.PartyMapActive
	avatar.SetStatus(party_state.Good)
	avatar.Intelligence = 10
	state.activePlayerIndex = -1
	state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()

	return state, npc
}

func installMerchantTestNPC(t *testing.T, state *GameState, npc *map_units.NPCFriendly) {
	t.Helper()
	require.NotNil(t, state.CurrentNPCAIController)
	require.NotNil(t, state.CurrentNPCAIController.GetNpcs())
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{npc})
}

func setMerchantDesiredSlot(
	t *testing.T,
	state *GameState,
	npc *map_units.NPCFriendly,
	wanted references.NPCScheduleSlot,
) {
	t.Helper()

	for hour := range 24 {
		candidate := state.DateTime
		candidate.Hour = byte(hour)
		if npc.NPCReference.Schedule.DesiredSlot(candidate) == wanted {
			state.DateTime = candidate

			return
		}
	}
	t.Fatalf("merchant schedule has no hour for desired slot %d", wanted)
}

func setPartyTestVehicle(state *GameState, vehicle references.VehicleType) {
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(
		vehicle,
		references.Position{},
		0,
	)
}
