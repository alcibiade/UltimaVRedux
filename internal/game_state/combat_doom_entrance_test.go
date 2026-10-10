package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestRejectedDoomEntryStartsOneGenericShadowlordForEveryLivingCount(t *testing.T) {
	for living := 1; living <= references.ShadowlordCount; living++ {
		t.Run(fmt.Sprintf("%d_living", living), func(t *testing.T) {
			state, callbacks, beforeLocation, beforeFacing := newDoomEntranceEncounterFixture(t, living)
			beforeShadowlords := *state.ShadowlordState
			state.PartyState.Inventory.QuestItems.Set(references.Sceptre, 1)
			state.PartyState.Inventory.QuestItems.Set(references.ShardOfFalsehood, 1)
			state.PartyState.Inventory.QuestItems.Set(references.ShardOfHatred, 1)
			state.PartyState.Inventory.QuestItems.Set(references.ShardOfCowardice, 1)

			assert.True(t, state.ActionEnterLargeMap())

			require.NotNil(t, state.CombatScene)
			require.NotNil(t, state.CombatScene.DoomEntranceOrigin)
			assert.Nil(t, state.CombatScene.OverworldOrigin)
			assert.Nil(t, state.CombatScene.SmallMapOrigin)
			assert.Nil(t, state.CombatScene.DungeonRoomOrigin)
			assert.Equal(t, int(references.ShadowlordCombatMap), state.CombatScene.CombatMapRef.MapIndex)
			assert.Equal(t, references.BritanniaCombatTerritory, state.CombatScene.CombatMapRef.Territory)
			assert.Equal(t, CombatEntryNorth, state.CombatScene.EntryDirection)
			assert.Equal(t, 1, state.CombatScene.MonsterManager.Count())
			assert.Equal(t, references.MonsterShadowlord,
				state.CombatScene.MonsterManager.GetMonster(0).MonsterType())
			assert.Equal(t, beforeLocation, state.CombatScene.DoomEntranceOrigin.ReturnLocation)
			assert.Equal(t, beforeFacing, state.CombatScene.DoomEntranceOrigin.ReturnVehicle)
			assert.Equal(t, beforeShadowlords, *state.ShadowlordState)
			assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.ShardOfFalsehood))
			assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.ShardOfHatred))
			assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.ShardOfCowardice))
			assert.Zero(t, state.PartyState.Inventory.QuestItems.Get(references.Sceptre))
			assert.True(t, state.CombatScene.DoomEntranceOrigin.SceptreReclaimed)
			assert.Contains(t, callbacks.Messages, "Attacked at entrance!")
			assert.Contains(t, callbacks.Messages, "The Sceptre is reclaimed!")
		})
	}
}

func TestDoomEntryWithNoLivingShadowlordsUsesOrdinaryDungeonWorkflow(t *testing.T) {
	state, _, _, _ := newDoomEntranceEncounterFixture(t, 0)

	assert.True(t, state.ActionEnterLargeMap())

	assert.Nil(t, state.CombatScene)
	assert.True(t, state.EventQueueManager.HasWorkflow("enter_location"))
}

func TestResolveEnterLargeMapDistinguishesDoomEntranceCombat(t *testing.T) {
	state, _, _, _ := newDoomEntranceEncounterFixture(t, 1)

	assert.Equal(t, EnterLargeMapDoomCombat, state.ResolveEnterLargeMap())
	assert.NotNil(t, state.CombatScene)
}

func TestDoomEntranceOriginDoesNotEnableDoomInteriorOverrides(t *testing.T) {
	state, _, _, _ := newDoomEntranceEncounterFixture(t, 3)
	character := state.PartyState.GetCharacter(0)
	character.SetEquippedItem(references.SlotArmour, references.PlateMail)
	character.SetEquippedItem(references.SlotShield, references.LargeShield)
	require.Positive(t, character.GetTotalDefense())
	require.True(t, state.ActionEnterLargeMap())

	assert.False(t, state.CombatScene.isDoomInteriorCombat())
	assert.Equal(t, character.GetTotalDefense(), state.combatPartyDefense(character))
	monster := state.CombatScene.MonsterManager.GetMonster(0)
	monster.SetInvisible(true)
	assert.Equal(t, CombatAimEmpty, state.GetCombatAimDisposition(monster.Position()),
		"generic entrance combat retains ordinary player targeting invisibility")
}

func TestDoomEntranceCombatRejectsDebugExitBypass(t *testing.T) {
	state, _, _, _ := newDoomEntranceEncounterFixture(t, 3)
	require.True(t, state.ActionEnterLargeMap())

	_, err := state.ExitDebugCombat(state.CombatScene.EncounterID)

	assert.ErrorIs(t, err, ErrDebugCombatBusy)
	assert.NotNil(t, state.CombatScene)
}

func TestDoomEntranceCombatRestoresExactOriginAfterVictoryAndEscape(t *testing.T) {
	for _, test := range []struct {
		name    string
		resolve func(*GameState) CombatEndReason
	}{
		{
			name: "victory",
			resolve: func(state *GameState) CombatEndReason {
				state.EndCombat(CombatVictory)

				return CombatVictory
			},
		},
		{
			name: "party escape",
			resolve: func(state *GameState) CombatEndReason {
				require.True(t, state.CombatScene.establishExitRoute(CombatExitSouth))
				for _, partyIndex := range state.ActiveCombatPartyMembers() {
					state.departPCFromCombat(CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex})
				}

				return state.checkCombatEndConditions()
			},
		},
		{
			name: "Shadowlord escape",
			resolve: func(state *GameState) CombatEndReason {
				state.CombatScene.MonsterManager.GetMonster(0).SetActive(false)
				state.CombatScene.TurnQueue.MarkFled(0, false)
				state.checkCombatEndConditions()
				for _, partyIndex := range state.ActiveCombatPartyMembers() {
					state.departPCFromCombat(CombatTurnIdentity{IsPartyMember: true, EntityIndex: partyIndex})
				}

				return state.checkCombatEndConditions()
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks, beforeLocation, beforeFacing := newDoomEntranceEncounterFixture(t, 2)
			beforeShadowlords := *state.ShadowlordState
			require.True(t, state.ActionEnterLargeMap())
			state.PartyVehicle.GetVehicleDetails().SetPartyVehicleDirection(references.Down)

			reason := test.resolve(state)

			assert.Nil(t, state.CombatScene)
			assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
			assert.Equal(t, beforeFacing, state.PartyVehicle.GetVehicleDetails().Snapshot())
			assert.Equal(t, beforeShadowlords, *state.ShadowlordState)
			require.Len(t, callbacks.CombatConclusions, 1)
			assert.Equal(t, reason, callbacks.CombatConclusions[0].Reason)
		})
	}
}

func TestDoomEntranceCombatDefeatUsesOrdinaryPartyWipeFlowAfterRestoration(t *testing.T) {
	state, _, beforeLocation, beforeFacing := newDoomEntranceEncounterFixture(t, 1)
	deathSequences := 0
	var deathLocation references.PlayerLocation
	var deathFacing map_units.VehicleDetailsSnapshot
	state.SystemCallbacks.Flow.Consequences.TriggerDeathSequence = func() {
		deathSequences++
		deathLocation = state.MapState.PlayerLocation
		deathFacing = state.PartyVehicle.GetVehicleDetails().Snapshot()
	}
	require.True(t, state.ActionEnterLargeMap())
	state.PartyState.ForEachCharacter(func(_ int, character *party_state.PlayerCharacter) {
		if character.PartyMap == party_state.PartyMapActive {
			character.SetHp(0)
			character.MarkAsDead()
		}
	})
	state.PartyVehicle.GetVehicleDetails().SetPartyVehicleDirection(references.Down)

	state.EndCombat(CombatDefeat)

	assert.Equal(t, 1, deathSequences)
	assert.Equal(t, beforeLocation, deathLocation)
	assert.Equal(t, beforeFacing, deathFacing)
}

func newDoomEntranceEncounterFixture(
	t *testing.T,
	living int,
) (*GameState, *MockSystemCallbacks, references.PlayerLocation, map_units.VehicleDetailsSnapshot) {
	t.Helper()

	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(references.FloorNumber(references.UNDERWORLD)).
		WithRandomSeed(312).
		WithSystemCallbacks().
		Build()
	state.SetDungeonState(references.Doom, true)
	state.MapState.PlayerLocation.Position = state.GetDungeonEntranceCoordinates()[references.Doom]
	state.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	state.PartyVehicle.GetVehicleDetails().SetPartyVehicleDirection(references.Left)
	state.ShadowlordState.Reset()
	for lord := references.Faulinei; lord <= references.Nosfentor; lord++ {
		state.ShadowlordState.Alive[lord] = int(lord) < living
		state.ShadowlordState.TownAssignments[lord] = references.Location(int(references.Britain) + int(lord))
	}
	state.ShadowlordState.Summoned = living > 0
	state.ShadowlordState.LastCalled = references.NoShadowlord
	if living > 0 {
		state.ShadowlordState.LastCalled = references.Faulinei
	}
	state.ShadowlordState.AutomaticPresence = references.NoShadowlord
	callbacks.Reset()

	return state, callbacks, state.MapState.PlayerLocation, state.PartyVehicle.GetVehicleDetails().Snapshot()
}
