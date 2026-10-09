package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestBeginMerchantSessionResolvesAllEightFamilies(t *testing.T) {
	tests := []struct {
		merchantID references.MerchantID
		family     references.MerchantFamily
	}{
		{references.MerchantArmsBritain, references.MerchantFamilyArms},
		{references.MerchantTavernBritain, references.MerchantFamilyTavern},
		{references.MerchantHorseTrinsic, references.MerchantFamilyHorse},
		{references.MerchantShipJhelom, references.MerchantFamilyShip},
		{references.MerchantReagentMoonglow, references.MerchantFamilyReagent},
		{references.MerchantGuildNewMagincia, references.MerchantFamilyGuild},
		{references.MerchantHealerMinoc, references.MerchantFamilyHealer},
		{references.MerchantInnBritain, references.MerchantFamilyInn},
	}

	for _, test := range tests {
		t.Run(string(test.merchantID), func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, test.merchantID)

			session, err := state.BeginMerchantSession(npc)

			require.NoError(t, err)
			assert.Equal(t, test.merchantID, session.MerchantID())
			assert.Equal(t, test.family, session.Family())
			assert.Equal(t, npc.GetKeySpriteIndex(), session.MerchantSpriteIndex())
			assert.Equal(t, 0, session.PricingCharacterIndex())
			assert.Equal(t, state.PartyState.GetCharacter(0).GetNameAsString(), session.PricingCharacterName())
			assert.Equal(t, test.merchantID, session.Merchant().ID)
		})
	}
}

func TestCanonicalMerchantDefinitionsResolveActualNPCReferences(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).Build()
	definitions := state.GameReferences.MerchantReferences.Definitions()
	require.Len(t, definitions, 46)

	for _, definition := range definitions {
		t.Run(string(definition.ID), func(t *testing.T) {
			location := state.GameReferences.LocationReferences.GetLocationReference(definition.Location)
			require.NotNil(t, location)
			assert.Equal(t, definition.Location, location.Location)
			npcReferences := location.GetNPCReferences()
			require.NotNil(t, npcReferences)
			require.Less(t, definition.NPCIndex, len(*npcReferences))
			npcReference := (*npcReferences)[definition.NPCIndex]
			assert.Equal(t, definition.Location, npcReference.Location)
			assert.Equal(t, byte(definition.Family), npcReference.DialogNumber)

			resolved, err := state.GameReferences.MerchantReferences.Resolve(
				references.DialogNumber(npcReference.DialogNumber),
				npcReference.Location,
				definition.NPCIndex,
			)
			require.NoError(t, err)
			assert.Equal(t, definition, resolved)
		})
	}
}

func TestBeginMerchantSessionByIDUsesExactLoadedMerchantAndProductionValidation(t *testing.T) {
	t.Run("correct loaded ID", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)

		session, err := state.BeginMerchantSessionByID(references.MerchantGuildNewMagincia)

		require.NoError(t, err)
		assert.Equal(t, references.MerchantGuildNewMagincia, session.MerchantID())
		assert.Equal(t, npc.GetKeySpriteIndex(), session.MerchantSpriteIndex())
	})

	t.Run("wrong town", func(t *testing.T) {
		state, _ := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
		state.MapState.PlayerLocation.Location = references.Yew

		_, err := state.BeginMerchantSessionByID(references.MerchantGuildNewMagincia)

		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assert.ErrorContains(t, err, "does not match current location")
	})

	t.Run("unimplemented family", func(t *testing.T) {
		state, _ := newOpenMerchantTestState(t, references.MerchantArmsBritain)

		session, err := state.BeginMerchantSessionByID(references.MerchantArmsBritain)

		require.NoError(t, err)
		assert.Equal(t, references.MerchantFamilyArms, session.Family())
	})

	t.Run("detached merchant is not substituted for loaded NPC", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
		detached := map_units.NewNPCFriendly(npc.NPCReference, npc.MapUnitDetails().NPCNum)
		detached.PlaceAtScheduleSlot(npc.ArrivedScheduleSlot())
		state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)

		_, err := state.BeginMerchantSession(detached)
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assert.ErrorContains(t, err, "not managed by the current controller")

		_, err = state.BeginMerchantSessionByID(references.MerchantGuildNewMagincia)

		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assert.ErrorContains(t, err, "not loaded by the current controller")
	})
}

func TestBeginMerchantSessionRejectsUnsupportedAndMismatchedNPCs(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	unsupportedReference := npc.NPCReference
	unsupportedReference.DialogNumber = 0x89
	unsupported := map_units.NewNPCFriendly(unsupportedReference, npc.MapUnitDetails().NPCNum)
	unsupported.PlaceAtScheduleSlot(1)
	installMerchantTestNPC(t, state, unsupported)

	_, err := state.BeginMerchantSession(unsupported)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "unsupported merchant dialog code")

	mismatch := map_units.NewNPCFriendly(npc.NPCReference, npc.MapUnitDetails().NPCNum+1)
	mismatch.PlaceAtScheduleSlot(1)
	installMerchantTestNPC(t, state, mismatch)
	_, err = state.BeginMerchantSession(mismatch)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "NPC index mismatch")

	foreign := map_units.NewNPCFriendly(npc.NPCReference, npc.MapUnitDetails().NPCNum)
	foreign.NPCReference.Location = references.Yew
	foreign.PlaceAtScheduleSlot(1)
	installMerchantTestNPC(t, state, foreign)
	_, err = state.BeginMerchantSession(foreign)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "does not match current location")
}

func TestBeginMerchantSessionRejectsDetachedHiddenEmptyAndWrongFloorNPCs(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	detached := map_units.NewNPCFriendly(npc.NPCReference, npc.MapUnitDetails().NPCNum)
	detached.PlaceAtScheduleSlot(1)

	_, err := state.BeginMerchantSession(detached)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "not managed by the current controller")

	npc.SetVisible(false)
	_, err = state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "empty or hidden")
	npc.SetVisible(true)

	npc.SetFloor(state.MapState.PlayerLocation.Floor + 1)
	_, err = state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "does not match current floor")

	emptyReference := npc.NPCReference
	emptyReference.Schedule.X = [3]byte{}
	emptyReference.Schedule.Y = [3]byte{}
	empty := map_units.NewNPCFriendly(emptyReference, npc.MapUnitDetails().NPCNum)
	empty.SetVisible(true)
	empty.SetFloor(state.MapState.PlayerLocation.Floor)
	installMerchantTestNPC(t, state, empty)
	_, err = state.BeginMerchantSession(empty)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "empty or hidden")
}

func TestMerchantSessionRevalidationRejectsControllerReplacement(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	replacement := map_units.NewNPCFriendly(npc.NPCReference, npc.MapUnitDetails().NPCNum)
	replacement.PlaceAtScheduleSlot(npc.ArrivedScheduleSlot())
	replacement.SetVisible(true)
	state.CurrentNPCAIController = &SimpleTestController{
		mapUnits: map_units.MapUnits{replacement},
	}

	_, err = state.GetGuildShopState(session)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assert.ErrorContains(t, err, "not managed by the current controller")

	_, err = state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
}

func TestMerchantSessionRevalidationRejectsHiddenAndWrongFloorNPCs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*GameState, *map_units.NPCFriendly)
	}{
		{
			name: "hidden",
			mutate: func(_ *GameState, npc *map_units.NPCFriendly) {
				npc.SetVisible(false)
			},
		},
		{
			name: "wrong floor",
			mutate: func(state *GameState, npc *map_units.NPCFriendly) {
				npc.SetFloor(state.MapState.PlayerLocation.Floor + 1)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			test.mutate(state, npc)

			_, err = state.GetGuildShopState(session)
			require.ErrorIs(t, err, ErrInvalidMerchantSession)
		})
	}
}

func TestEndMerchantSessionInvalidatesOnlyCurrentToken(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	first, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	current, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	state.EndMerchantSession(first)
	_, err = state.GetGuildShopState(current)
	require.NoError(t, err, "ending a superseded token must not invalidate the current token")

	state.EndMerchantSession(current)
	_, err = state.GetGuildShopState(current)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
}

func TestBeginMerchantSessionRequiresArrivedAndDesiredMerchantSlots(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	npc.PlaceAtScheduleSlot(0)

	_, err := state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrMerchantClosed, "approaching work remains closed")

	npc.PlaceAtScheduleSlot(1)
	setMerchantDesiredSlot(t, state, npc, 1)
	_, err = state.BeginMerchantSession(npc)
	require.NoError(t, err, "arrived at work during the work slot is open")

	setMerchantDesiredSlot(t, state, npc, 0)
	_, err = state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrMerchantClosed, "departing work closes immediately")
}

func TestBeginMerchantSessionCapturesSelectedEligibleBuyerAndFallsBack(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	selected := state.PartyState.GetCharacter(1)
	selected.PartyMap = party_state.PartyMapActive
	selected.SetStatus(party_state.Poisoned)
	selected.Intelligence = 27
	require.True(t, state.SetActivePlayer(1))

	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	assert.Equal(t, 1, session.PricingCharacterIndex())

	selected.SetStatus(party_state.Sleep)
	session, err = state.BeginMerchantSession(npc)
	require.NoError(t, err)
	assert.Equal(t, 0, session.PricingCharacterIndex(), "ineligible selection falls back to first eligible active character")

	state.PartyState.GetCharacter(0).SetStatus(party_state.Dead)
	_, err = state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrMerchantBuyerUnavailable)
}

func TestBeginMerchantSessionAppliesOnlyHorsebackRefusalWithHorseDealerException(t *testing.T) {
	state, armsNPC := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	setPartyTestVehicle(state, references.HorseVehicle)
	_, err := state.BeginMerchantSession(armsNPC)
	require.ErrorIs(t, err, ErrMerchantHorsebackRefusal)

	horseState, horseNPC := newOpenMerchantTestState(t, references.MerchantHorseTrinsic)
	setPartyTestVehicle(horseState, references.HorseVehicle)
	_, err = horseState.BeginMerchantSession(horseNPC)
	require.NoError(t, err)

	setPartyTestVehicle(state, references.CarpetVehicle)
	_, err = state.BeginMerchantSession(armsNPC)
	require.NoError(t, err, "unrelated vehicles are not part of the original rider refusal")
}

func TestBeginDebugMerchantSessionBypassesScheduleAndHorseOnly(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	npc.PlaceAtScheduleSlot(0)
	setMerchantDesiredSlot(t, state, npc, 0)
	setPartyTestVehicle(state, references.HorseVehicle)
	beforeSequence := state.merchantSessionSequence

	_, err := state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrMerchantClosed)
	assert.Equal(t, beforeSequence, state.merchantSessionSequence)

	session, err := state.BeginDebugMerchantSessionByID(references.MerchantGuildNewMagincia)
	require.NoError(t, err)
	assert.Equal(t, beforeSequence+1, state.merchantSessionSequence)
	_, err = state.GetGuildShopState(session)
	require.NoError(t, err, "the issued debug policy must survive repeated transaction validation")

	npc.SetFloor(state.MapState.PlayerLocation.Floor + 1)
	_, err = state.GetGuildShopState(session)
	require.ErrorIs(t, err, ErrInvalidMerchantSession, "debug launch must retain canonical floor validation")
	npc.SetFloor(state.MapState.PlayerLocation.Floor)
	npc.NPCReference.DialogNumber = byte(references.MerchantFamilyArms)
	_, err = state.GetGuildShopState(session)
	require.ErrorIs(t, err, ErrInvalidMerchantSession, "debug launch must retain canonical merchant validation")

	npc.NPCReference.DialogNumber = byte(references.MerchantFamilyGuild)
	state.EndMerchantSession(session)
	_, err = state.BeginMerchantSession(npc)
	require.ErrorIs(t, err, ErrMerchantClosed, "production entry behavior must remain enforced")
}

func TestBeginDebugMerchantSessionConsumesLoadedTalk3BeforeIssuingSession(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	slot := npc.ArrivedScheduleSlot()
	npc.NPCReference.Schedule.Ai[slot] = byte(references.Talk3)

	session, err := state.BeginDebugMerchantSessionByID(references.MerchantGuildNewMagincia)

	require.NoError(t, err)
	assert.Equal(t, references.AiType(references.Wander3), references.AiType(npc.NPCReference.Schedule.Ai[slot]))
	consumedSlot, consumed := npc.MapUnitDetails().ConsumedTalk3ScheduleSlot()
	assert.True(t, consumed)
	assert.Equal(t, slot, consumedSlot)
	state.EndMerchantSession(session)
}

func TestReviewDebugMerchantSessionByIDDoesNotIssueToken(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	npc.PlaceAtScheduleSlot(0)
	setMerchantDesiredSlot(t, state, npc, 0)
	setPartyTestVehicle(state, references.HorseVehicle)
	beforeSequence := state.merchantSessionSequence

	review, err := state.ReviewDebugMerchantSessionByID(references.MerchantGuildNewMagincia)

	require.NoError(t, err)
	assert.Equal(t, references.MerchantGuildNewMagincia, review.Merchant.ID)
	assert.Equal(t, npc.GetKeySpriteIndex(), review.MerchantSpriteIndex)
	assert.Equal(t, npc.Floor(), review.NPCFloor)
	assert.Equal(t, beforeSequence, state.merchantSessionSequence)
}
