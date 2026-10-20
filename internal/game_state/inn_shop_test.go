package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestReviewInnShopIsPureAndExactForAllSixRows(t *testing.T) {
	tests := []struct {
		id       references.MerchantID
		base     int
		capacity int
		sleep    references.PlayerLocation
		wake     references.PlayerLocation
	}{
		{references.MerchantInnBritain, 2, 3, references.PlayerLocation{Location: references.Britain, Position: references.Position{X: 21, Y: 10}, Floor: 0}, references.PlayerLocation{Location: references.Britain, Position: references.Position{X: 22, Y: 10}, Floor: 0}},
		{references.MerchantInnJhelom, 3, 4, references.PlayerLocation{Location: references.Jhelom, Position: references.Position{X: 15, Y: 7}, Floor: 1}, references.PlayerLocation{Location: references.Jhelom, Position: references.Position{X: 16, Y: 7}, Floor: 1}},
		{references.MerchantInnSkaraBrae, 2, 3, references.PlayerLocation{Location: references.SkaraBrae, Position: references.Position{X: 25, Y: 9}, Floor: 0}, references.PlayerLocation{Location: references.SkaraBrae, Position: references.Position{X: 26, Y: 9}, Floor: 0}},
		{references.MerchantInnNorthBritanny, 3, 2, references.PlayerLocation{Location: references.NorthBritanny, Position: references.Position{X: 20, Y: 1}, Floor: 0}, references.PlayerLocation{Location: references.NorthBritanny, Position: references.Position{X: 21, Y: 1}, Floor: 0}},
		{references.MerchantInnPaws, 2, 2, references.PlayerLocation{Location: references.Paws, Position: references.Position{X: 27, Y: 6}, Floor: 0}, references.PlayerLocation{Location: references.Paws, Position: references.Position{X: 28, Y: 6}, Floor: 0}},
		{references.MerchantInnBuccaneersDen, 3, 2, references.PlayerLocation{Location: references.BuccaneersDen, Position: references.Position{X: 7, Y: 26}, Floor: 0}, references.PlayerLocation{Location: references.BuccaneersDen, Position: references.Position{X: 8, Y: 26}, Floor: 0}},
	}
	for _, test := range tests {
		t.Run(string(test.id), func(t *testing.T) {
			state, _ := newOpenMerchantTestState(t, test.id)
			prepareInnTestRoster(state, 3)
			state.PartyState.GetAvatar().Intelligence = 10
			state.PartyState.Inventory.Gold.Set(500)
			state.SetRandomSeed(212)
			beforeRoster := state.CurrentPartyRosterSnapshot()
			beforeDate := state.DateTime
			beforeLocation := state.MapState.PlayerLocation
			beforeSequence := state.merchantSessionSequence
			beforeRNG := state.GetRNG().Calls()

			review, err := state.ReviewInnShopByMerchantID(test.id)

			require.NoError(t, err)
			assert.Equal(t, test.id, review.State.Merchant.ID)
			assert.Equal(t, test.capacity, review.State.RoomCapacity)
			assert.Zero(t, review.State.RoomOccupancy)
			assert.Equal(t, 3, review.State.PartySize)
			assert.Equal(t, party_state.MaxActiveParty, review.State.PartyCapacity)
			assert.Equal(t, test.sleep, review.State.Rest.SleepLocation)
			assert.Equal(t, test.wake, review.State.Rest.WakeLocation)
			assert.Equal(t, uint16(CalculateMerchantBuyPrice(test.base*3, 10)), review.State.Rest.Charge)
			assert.Len(t, review.State.LeaveRows, 3)
			assert.Empty(t, review.State.GuestRows)
			assert.Equal(t, beforeRoster, state.CurrentPartyRosterSnapshot())
			assert.Equal(t, beforeDate, state.DateTime)
			assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
			assert.Equal(t, beforeSequence, state.merchantSessionSequence)
			assert.Equal(t, beforeRNG, state.GetRNG().Calls())
		})
	}
}

func TestInnStateReportsEligibilityWarningsAndGuestBillsInRosterOrder(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 3)
	state.PartyState.GetCharacter(1).SetStatus(party_state.Dead)
	state.PartyState.GetCharacter(1).SetHp(0)
	guestA := state.PartyState.GetCharacter(7)
	guestA.PartyMap = party_state.PartyMapAtInn(references.Britain)
	guestA.MonthsAtInn = 0
	guestA.SetStatus(party_state.Sleep)
	guestB := state.PartyState.GetCharacter(11)
	guestB.PartyMap = party_state.PartyMapAtInn(references.Britain)
	guestB.MonthsAtInn = 2
	guestB.SetStatus(party_state.Dead)
	state.PartyState.Inventory.Gold.Set(30)

	shop, err := state.GetInnShopState(session)

	require.NoError(t, err)
	assert.Equal(t, InnLeaveBlockAvatar, shop.LeaveRows[0].BlockReason)
	assert.Equal(t, InnLeaveBlockDead, shop.LeaveRows[1].BlockReason)
	assert.True(t, shop.LeaveRows[2].Eligible)
	require.Len(t, shop.GuestRows, 2)
	assert.Equal(t, 7, shop.GuestRows[0].RosterIndex)
	assert.Equal(t, 11, shop.GuestRows[1].RosterIndex)
	assert.Equal(t, 1, shop.GuestRows[0].BilledMonths)
	assert.Contains(t, shop.GuestRows[0].Warnings, InnWarningMinimumMonthCharge)
	assert.Equal(t, 2, shop.GuestRows[1].BilledMonths)
	assert.Equal(t, InnPickupBlockDead, shop.GuestRows[1].BlockReason)
	assert.False(t, shop.GuestRows[1].Confirmable)
	wantMonthly := uint16(CalculateMerchantBuyPrice(20, 10))
	assert.Equal(t, wantMonthly, shop.GuestRows[0].MonthlyRate)
	assert.Equal(t, wantMonthly, shop.GuestRows[0].Charge)
	assert.Equal(t, wantMonthly*2, shop.GuestRows[1].Charge)
}

func TestInnSessionRetainsCapturedBuyerAfterStableLeave(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 3)
	state.activePlayerIndex = 1
	state.EndMerchantSession(session)
	var err error
	session, err = state.BeginMerchantSessionByID(references.MerchantInnBritain)
	require.NoError(t, err)
	captured := state.PartyState.GetCharacter(1)
	assert.Equal(t, 10, session.buyerIntelligence)
	captured.Intelligence = 30
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	row := shop.LeaveRows[1]

	result, err := state.LeaveCompanionAtInn(session, innLeaveRequest(shop, row))

	require.NoError(t, err)
	assert.Same(t, captured, state.PartyState.GetCharacter(party_state.NRosterSlots-1))
	assert.Equal(t, party_state.NRosterSlots-1, result.State.PricingCharacterIndex)
	require.Len(t, result.State.GuestRows, 1)
	assert.Equal(t, captured.GetNameAsString(), result.State.PricingCharacterName)
	assert.Equal(t, uint16(CalculateMerchantBuyPrice(20, 10)), result.State.GuestRows[0].MonthlyRate)
	pickupRow, err := state.QuoteInnPickup(session, result.State.GuestRows[0].Identity)
	require.NoError(t, err)
	pickup, err := state.PickUpInnGuest(session, innPickupRequest(result.State, pickupRow))
	require.NoError(t, err)
	assert.Same(t, captured, state.PartyState.GetCharacter(2))
	assert.Equal(t, 2, pickup.State.PricingCharacterIndex)
	assert.Equal(t, 30, int(captured.Intelligence), "roster movement must not overwrite the captured character")
}

func TestInnPickupRoundsOneMonthRateBeforeMultiplyingMonths(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 2)
	state.EndMerchantSession(session)
	state.PartyState.GetAvatar().Intelligence = 12
	var err error
	session, err = state.BeginMerchantSessionByID(references.MerchantInnBritain)
	require.NoError(t, err)
	guest := state.PartyState.GetCharacter(8)
	guest.PartyMap = party_state.PartyMapAtInn(references.Britain)
	guest.MonthsAtInn = 2

	shop, err := state.GetInnShopState(session)

	require.NoError(t, err)
	require.Len(t, shop.GuestRows, 1)
	assert.Equal(t, uint16(32), shop.GuestRows[0].MonthlyRate)
	assert.Equal(t, uint16(64), shop.GuestRows[0].Charge)
	assert.NotEqual(t, uint16(CalculateMerchantBuyPrice(40, 12)), shop.GuestRows[0].Charge,
		"the rounded one-month rate must be multiplied after division")
}

func TestInnCommitRejectsCompleteRosterStalenessAndSupersededSession(t *testing.T) {
	state, npc, session, _ := newInnTestState(t, references.MerchantInnBritain, 2)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	request := innLeaveRequest(shop, shop.LeaveRows[1])
	state.PartyState.GetCharacter(15).ArmorClass++
	before := state.CurrentPartyRosterSnapshot()

	_, err = state.LeaveCompanionAtInn(session, request)

	require.ErrorIs(t, err, ErrInvalidInnRequest)
	assert.Equal(t, before, state.CurrentPartyRosterSnapshot())

	newSession, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.GetInnShopState(session)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	_, err = state.GetInnShopState(newSession)
	require.NoError(t, err)
}

func TestRestAtInnUsesExactPaymentTimeAndMorningOutcomes(t *testing.T) {
	state, _, session, callbacks := newInnTestState(t, references.MerchantInnBritain, 5)
	state.EndMerchantSession(session)
	state.DateTime.Year, state.DateTime.Month, state.DateTime.Day = 1, 13, 28
	state.DateTime.Hour, state.DateTime.Minute = 23, 30
	var err error
	session, err = state.beginMerchantSession(stateMerchantNPC(t, state), true)
	require.NoError(t, err)

	avatar := state.PartyState.GetCharacter(0)
	avatar.Class, avatar.Intelligence = party_state.Avatar, 20
	avatar.SetHp(1)
	avatar.SetMp(0)
	sleeper := state.PartyState.GetCharacter(1)
	sleeper.Class, sleeper.Intelligence = party_state.Wizard, 18
	sleeper.SetStatus(party_state.Sleep)
	sleeper.SetHp(2)
	sleeper.SetMp(0)
	charmed := state.PartyState.GetCharacter(2)
	charmed.Class, charmed.Intelligence = party_state.Bard, 17
	charmed.SetStatus(party_state.Charmed)
	charmed.SetHp(3)
	charmed.SetMp(0)
	poisoned := state.PartyState.GetCharacter(3)
	poisoned.Class, poisoned.Intelligence = party_state.Wizard, 16
	poisoned.SetStatus(party_state.Poisoned)
	poisoned.SetHp(4)
	poisoned.SetMp(1)
	dead := state.PartyState.GetCharacter(4)
	dead.SetHp(0)
	dead.SetStatus(party_state.Dead)
	dead.SetMp(3)
	for index := range 5 {
		state.PartyState.GetCharacter(index).MonthsAtInn = 0
	}
	state.PartyState.Inventory.Provisions.Food.Set(77)
	state.PartyState.Inventory.Gold.Set(500)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 20}
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	const seed = uint64(99)
	expectedRNG := beradrng.New(seed)
	state.SetRandomSeed(seed)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	assert.Positive(t, shop.MonthlyRate)
	request := innRestRequest(shop)
	wantTheft := uint16(expectedRNG.Intn(64) + 1)
	merchant := stateMerchantNPC(t, state)
	merchant.SetPos(references.Position{X: 1, Y: 1})
	merchant.MapUnitDetails().SetCurrentPath([]references.Position{{X: 2, Y: 2}})

	result, err := state.RestAtInn(session, request)

	require.NoError(t, err)
	assert.Equal(t, shop.Rest.Charge, result.Charge)
	assert.Equal(t, uint16(500)-result.Charge, result.GoldAfterPayment)
	assert.Equal(t, wantTheft, result.Theft)
	assert.Equal(t, result.GoldAfterPayment-wantTheft, result.GoldAfterTheft)
	assert.Equal(t, result.GoldAfterTheft, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint16(77), result.FoodAfter)
	assert.Equal(t, uint16(77), state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, shop.Rest.TimeStepsMinutes, result.TimeStepsMinutes)
	assert.Equal(t, shop.Rest.DateTimeAfter, state.DateTime)
	assert.Equal(t, uint16(2), state.DateTime.Year)
	assert.Equal(t, byte(1), state.DateTime.Month)
	assert.Equal(t, byte(6), state.DateTime.Hour)
	assert.Equal(t, references.PlayerLocation{Location: references.Britain, Position: references.Position{X: 22, Y: 10}, Floor: 0}, state.MapState.PlayerLocation)
	assert.Equal(t, DurationEffectNone, state.ActiveDurationEffect.EffectType)
	wantMerchantSlot := merchant.NPCReference.Schedule.DesiredSlot(state.DateTime)
	wantMerchantPosition := merchant.NPCReference.Schedule.GetIndividualNPCBehaviourBySlot(wantMerchantSlot).Position
	assert.Equal(t, wantMerchantPosition, merchant.Pos(), "morning initnpcs must restore the current schedule")
	assert.Empty(t, merchant.MapUnitDetails().CurrentPath)
	assert.Equal(t, party_state.Good, avatar.GetStatus())
	assert.Equal(t, avatar.GetMaxHp(), avatar.GetCurrentHp())
	assert.Equal(t, byte(20), avatar.GetCurrentMp())
	assert.Equal(t, party_state.Good, sleeper.GetStatus())
	assert.Equal(t, byte(18), sleeper.GetCurrentMp())
	assert.Equal(t, party_state.Charmed, charmed.GetStatus())
	assert.Equal(t, byte(8), charmed.GetCurrentMp())
	assert.Equal(t, party_state.Dead, poisoned.GetStatus())
	assert.Zero(t, poisoned.GetCurrentHp())
	assert.Equal(t, byte(16), poisoned.GetCurrentMp(), "Rest restores class MP before poison death")
	assert.Equal(t, party_state.Dead, dead.GetStatus())
	assert.Zero(t, dead.GetCurrentHp())
	assert.Equal(t, byte(3), dead.GetCurrentMp())
	for index := range party_state.NRosterSlots {
		character := state.PartyState.GetCharacter(index)
		if index < 5 {
			assert.Equal(t, byte(1), character.MonthsAtInn, "slot %d", index)
		}
	}
	assert.Equal(t, 1, callbacks.StatsDisplayUpdated)
	foodAfterRest := state.PartyState.Inventory.Provisions.Food.Get()
	state.AdvanceGameplayMinutes(1)
	assert.Equal(t, foodAfterRest, state.PartyState.Inventory.Provisions.Food.Get(), "the final Talk minute must not consume deferred breakfast")
}

func TestRestAtInnRejectsAStaleReviewedClock(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 1)
	state.PartyState.Inventory.Gold.Set(100)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	request := innRestRequest(shop)
	state.DateTime.Advance(1)

	_, err = state.RestAtInn(session, request)

	require.ErrorIs(t, err, ErrInvalidInnRequest)
}

func TestReviewInnShopSupportsVacantNativeRosterSlots(t *testing.T) {
	state, _, _, _ := newInnTestState(t, references.MerchantInnBritain, 1)
	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	roster[0] = state.PartyState.GetAvatar()
	roster[8] = state.PartyState.GetCharacter(8)
	roster[8].PartyMap = party_state.PartyMapAtInn(references.Britain)
	state.PartyState = *party_state.NewPartyStateFromRoster(roster)

	review, err := state.ReviewInnShopByMerchantID(references.MerchantInnBritain)

	require.NoError(t, err)
	assert.Equal(t, 1, review.State.PartySize)
	assert.Equal(t, 1, review.State.RoomOccupancy)
	require.Len(t, review.State.GuestRows, 1)
	assert.Equal(t, 8, review.State.GuestRows[0].RosterIndex)
	detachedRoster := state.CurrentPartyRosterSnapshot()
	assert.Empty(t, detachedRoster.Characters[1].Name)
	assert.Equal(t, 1, detachedRoster.Characters[1].RosterIndex)
}

func TestLeaveCapturedInnBuyerSupportsVacantNativeRosterSlots(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 2)
	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	roster[0] = state.PartyState.GetCharacter(0)
	roster[1] = state.PartyState.GetCharacter(1)
	state.PartyState = *party_state.NewPartyStateFromRoster(roster)
	state.PartyState.Inventory.Gold.Set(999)
	session.buyer = state.PartyState.GetCharacter(1)
	session.buyerIndex = 1
	session.buyerName = roster[1].GetNameAsString()
	session.buyerIntelligence = int(roster[1].Intelligence)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	row := shop.LeaveRows[1]

	result, err := state.LeaveCompanionAtInn(session, innLeaveRequest(shop, row))

	require.NoError(t, err)
	assert.Equal(t, 1, result.State.PartySize)
	assert.Equal(t, 1, result.State.RoomOccupancy)
	assert.Equal(t, party_state.NRosterSlots-1, result.State.PricingCharacterIndex)
	assert.Same(t, session.buyer, state.PartyState.GetCharacter(party_state.NRosterSlots-1))
}

func TestRestAtInnPreservesFoodAndSpellOnRefusal(t *testing.T) {
	state, _, session, callbacks := newInnTestState(t, references.MerchantInnBritain, 2)
	state.PartyState.Inventory.Gold.Set(0)
	state.PartyState.Inventory.Provisions.Food.Set(9)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 30}
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	before := state.CurrentPartyRosterSnapshot()
	beforeDate := state.DateTime
	beforePosition := state.MapState.PlayerLocation

	_, err = state.RestAtInn(session, innRestRequest(shop))

	require.ErrorIs(t, err, ErrInnRestUnavailable)
	assert.Equal(t, before, state.CurrentPartyRosterSnapshot())
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforePosition, state.MapState.PlayerLocation)
	assert.Equal(t, DurationEffectQuickness, state.ActiveDurationEffect.EffectType)
	assert.Equal(t, 30, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, uint16(9), state.PartyState.Inventory.Provisions.Food.Get())
	assert.Zero(t, callbacks.StatsDisplayUpdated)
}

func TestRestAtInnConsumesOneRegenerationRollPerNineMinuteStep(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 1)
	state.EndMerchantSession(session)
	state.DateTime.Hour, state.DateTime.Minute = 20, 0
	var err error
	session, err = state.beginMerchantSession(stateMerchantNPC(t, state), true)
	require.NoError(t, err)
	character := state.PartyState.GetAvatar()
	character.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	character.SetHp(1)
	state.PartyState.Inventory.Gold.Set(100)
	state.ShadowlordState.AutomaticPresence = references.NoShadowlord
	for lord := references.Faulinei; lord < references.ShadowlordCount; lord++ {
		state.ShadowlordState.Kill(lord)
	}
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	ringSteps := len(shop.Rest.TimeStepsMinutes) - 12
	require.Positive(t, ringSteps)
	const seed = uint64(44)
	state.SetRandomSeed(seed)

	result, err := state.RestAtInn(session, innRestRequest(shop))

	require.NoError(t, err)
	assert.Equal(t, uint64(ringSteps), state.GetRNG().Calls())
	expected := beradrng.New(seed)
	successes := 0
	for range ringSteps {
		if expected.Intn(8) == 7 {
			successes++
		}
	}
	assert.Len(t, result.Regeneration, successes)
}

func TestLeaveCompanionAllowsLastLivingMemberAndPreservesStableRoster(t *testing.T) {
	state, _, session, callbacks := newInnTestState(t, references.MerchantInnBritain, 3)
	state.PartyState.GetCharacter(0).SetHp(0)
	state.PartyState.GetCharacter(0).SetStatus(party_state.Dead)
	state.PartyState.GetCharacter(1).SetHp(0)
	state.PartyState.GetCharacter(1).SetStatus(party_state.Dead)
	selected := state.PartyState.GetCharacter(2)
	selected.MonthsAtInn = 8
	state.activePlayerIndex = 2
	state.PartyState.Inventory.Gold.Set(321)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	row := shop.LeaveRows[2]
	require.True(t, row.Eligible)
	assert.Contains(t, row.Warnings, InnWarningLastLivingMember)
	beforeNames := innRosterSnapshotNames(shopRosterCharacters(shop, state))

	result, err := state.LeaveCompanionAtInn(session, innLeaveRequest(shop, row))

	require.NoError(t, err)
	assert.Equal(t, uint16(321), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, -1, state.activePlayerIndex)
	assert.Same(t, selected, state.PartyState.GetCharacter(15))
	assert.Equal(t, party_state.PartyMapAtInn(references.Britain), selected.PartyMap)
	assert.Zero(t, selected.MonthsAtInn)
	wantNames := append(append([]string(nil), beforeNames[:2]...), beforeNames[3:]...)
	wantNames = append(wantNames, beforeNames[2])
	assert.Equal(t, wantNames, currentInnRosterNames(state))
	assert.Equal(t, 2, result.State.PartySize)
	assert.Equal(t, 1, result.State.RoomOccupancy)
	assert.Equal(t, 1, callbacks.StatsDisplayUpdated)
	_, err = state.GetInnShopState(session)
	require.NoError(t, err, "Leave must keep the Inn session usable")
}

func TestLeaveCompanionBlocksAvatarDeadAndNoVacancy(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnPaws, 3)
	state.PartyState.GetCharacter(1).SetHp(0)
	state.PartyState.GetCharacter(1).SetStatus(party_state.Dead)
	state.PartyState.GetCharacter(8).PartyMap = party_state.PartyMapAtInn(references.Paws)
	state.PartyState.GetCharacter(9).PartyMap = party_state.PartyMapAtInn(references.Paws)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)

	assert.Equal(t, InnLeaveBlockAvatar, shop.LeaveRows[0].BlockReason)
	assert.Equal(t, InnLeaveBlockDead, shop.LeaveRows[1].BlockReason)
	assert.Equal(t, InnLeaveBlockNoVacancy, shop.LeaveRows[2].BlockReason)
	assert.Equal(t, InnRestBlockNoVacancy, shop.Rest.BlockReason)
	before := state.CurrentPartyRosterSnapshot()
	_, err = state.LeaveCompanionAtInn(session, innLeaveRequest(shop, shop.LeaveRows[2]))
	require.ErrorIs(t, err, ErrInnLeaveUnavailable)
	assert.Equal(t, before, state.CurrentPartyRosterSnapshot())
}

func TestPickUpInnGuestDeductsThenStealsThenStableInsertsAndRestores(t *testing.T) {
	state, _, session, callbacks := newInnTestState(t, references.MerchantInnBritain, 2)
	guest := state.PartyState.GetCharacter(9)
	guest.PartyMap = party_state.PartyMapAtInn(references.Britain)
	guest.MonthsAtInn = 2
	guest.Class, guest.Intelligence = party_state.Bard, 17
	guest.SetStatus(party_state.Sleep)
	guest.SetHp(3)
	guest.SetMp(1)
	state.PartyState.Inventory.Gold.Set(500)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	const seed = uint64(17)
	expectedRNG := beradrng.New(seed)
	state.SetRandomSeed(seed)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	require.Len(t, shop.GuestRows, 1)
	row := shop.GuestRows[0]
	wantTheft := uint16(expectedRNG.Intn(64) + 1)
	beforeNames := currentInnRosterNames(state)

	result, err := state.PickUpInnGuest(session, innPickupRequest(shop, row))

	require.NoError(t, err)
	assert.Equal(t, InnPickupCompleted, result.Outcome)
	assert.Equal(t, uint16(500)-row.Charge, result.GoldAfterPayment)
	assert.Equal(t, wantTheft, result.Theft)
	assert.Equal(t, result.GoldAfterPayment-wantTheft, result.FinalGold)
	assert.Same(t, guest, state.PartyState.GetCharacter(2))
	assert.Equal(t, party_state.PartyMapActive, guest.PartyMap)
	assert.Equal(t, party_state.Sleep, guest.GetStatus())
	assert.Equal(t, guest.GetMaxHp(), guest.GetCurrentHp())
	assert.Equal(t, byte(8), guest.GetCurrentMp())
	wantNames := append([]string(nil), beforeNames[:2]...)
	wantNames = append(wantNames, beforeNames[9])
	wantNames = append(wantNames, beforeNames[2:9]...)
	wantNames = append(wantNames, beforeNames[10:]...)
	assert.Equal(t, wantNames, currentInnRosterNames(state))
	assert.Empty(t, result.State.GuestRows)
	assert.Equal(t, 1, callbacks.StatsDisplayUpdated)
}

func TestPickUpInnGuestAllowsPoisonButBlocksDeadAndFullParty(t *testing.T) {
	t.Run("poisoned guest dies after payment", func(t *testing.T) {
		state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 2)
		guest := state.PartyState.GetCharacter(8)
		guest.PartyMap = party_state.PartyMapAtInn(references.Britain)
		guest.MonthsAtInn = 1
		guest.SetStatus(party_state.Poisoned)
		guest.SetHp(20)
		guest.SetMp(4)
		state.PartyState.Inventory.Gold.Set(500)
		shop, err := state.GetInnShopState(session)
		require.NoError(t, err)
		row := shop.GuestRows[0]
		assert.True(t, row.Confirmable)
		assert.Contains(t, row.Warnings, InnWarningPoisonDeath)

		result, pickupErr := state.PickUpInnGuest(session, innPickupRequest(shop, row))

		require.NoError(t, pickupErr)
		assert.Equal(t, party_state.Dead, guest.GetStatus())
		assert.Zero(t, guest.GetCurrentHp())
		assert.Equal(t, byte(4), guest.GetCurrentMp(), "poison pickup does not enter the restoration branch")
		assert.Equal(t, party_state.Dead, result.Projection.StatusAfter)
	})

	t.Run("dead guest remains visible but blocked", func(t *testing.T) {
		state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 2)
		guest := state.PartyState.GetCharacter(8)
		guest.PartyMap = party_state.PartyMapAtInn(references.Britain)
		guest.SetHp(0)
		guest.SetStatus(party_state.Dead)
		shop, err := state.GetInnShopState(session)
		require.NoError(t, err)
		row := shop.GuestRows[0]
		assert.Equal(t, InnPickupBlockDead, row.BlockReason)
		assert.False(t, row.Confirmable)
		_, err = state.PickUpInnGuest(session, innPickupRequest(shop, row))
		require.ErrorIs(t, err, ErrInnPickupUnavailable)
		assert.Same(t, guest, state.PartyState.GetCharacter(8))
	})

	t.Run("full party blocks living guest", func(t *testing.T) {
		state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, party_state.MaxActiveParty)
		guest := state.PartyState.GetCharacter(8)
		guest.PartyMap = party_state.PartyMapAtInn(references.Britain)
		shop, err := state.GetInnShopState(session)
		require.NoError(t, err)
		assert.Equal(t, InnPickupBlockPartyFull, shop.GuestRows[0].BlockReason)
	})
}

func TestInsufficientPickupActivatesGuardsExactlyOnceAndMutatesNothingElse(t *testing.T) {
	state, _, session, callbacks := newInnTestState(t, references.MerchantInnBritain, 2)
	guest := state.PartyState.GetCharacter(8)
	guest.PartyMap = party_state.PartyMapAtInn(references.Britain)
	guest.MonthsAtInn = 3
	guest.SetStatus(party_state.Good)
	state.PartyState.Inventory.Gold.Set(1)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.SetCriminalStatus(false)
	state.SetRandomSeed(1)
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	row := shop.GuestRows[0]
	require.False(t, row.Affordable)
	require.True(t, row.Confirmable)
	assert.Contains(t, row.Warnings, InnWarningInsufficientFundsActivatesGuards)
	beforeRoster := state.CurrentPartyRosterSnapshot()
	beforeRNG := state.GetRNG().Calls()

	result, err := state.PickUpInnGuest(session, innPickupRequest(shop, row))

	require.NoError(t, err)
	assert.Equal(t, InnPickupRefusedWithGuards, result.Outcome)
	assert.Equal(t, row.Charge, result.ListedCharge)
	assert.Zero(t, result.Theft)
	assert.Equal(t, uint16(1), result.FinalGold)
	assert.Equal(t, row.RosterIndex, result.CharacterIndexAfter)
	assert.Equal(t, guest.GetStatus(), result.Projection.StatusAfter)
	assert.Equal(t, guest.GetCurrentHp(), result.Projection.HPAfter)
	assert.Equal(t, beforeRoster, state.CurrentPartyRosterSnapshot())
	assert.Equal(t, beforeRoster, result.RosterAfter)
	assert.Equal(t, byte(3), guest.MonthsAtInn)
	assert.Same(t, guest, state.PartyState.GetCharacter(8))
	assert.False(t, state.IsCriminal())
	assert.Equal(t, 1, result.GuardActivation.ActiveNPCCount)
	assert.Equal(t, 1, result.GuardActivation.CivilianRollCount)
	assert.Equal(t, beforeRNG+1, state.GetRNG().Calls(), "one direct guard activation must process the one loaded merchant once")
	assert.Zero(t, callbacks.StatsDisplayUpdated)
}

func TestInnRosterTransferSurvivesNativeSnapshotRoundTrip(t *testing.T) {
	state, _, session, _ := newInnTestState(t, references.MerchantInnBritain, 3)
	selected := state.PartyState.GetCharacter(1)
	selected.ArmorClass = 77
	shop, err := state.GetInnShopState(session)
	require.NoError(t, err)
	_, err = state.LeaveCompanionAtInn(session, innLeaveRequest(shop, shop.LeaveRows[1]))
	require.NoError(t, err)
	want := state.CurrentPartyRosterSnapshot()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	snapshot.MapUnits = nil
	snapshot.MarkMapUnitsForReconstruction()
	restored, err := NewGameStateFromSaveSnapshot(
		snapshot, state.GameConfig, state.GameReferences, xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)

	assert.Equal(t, want, restored.CurrentPartyRosterSnapshot())
	stored := restored.PartyState.GetCharacter(15)
	assert.Equal(t, selected.GetNameAsString(), stored.GetNameAsString())
	assert.Equal(t, byte(77), stored.ArmorClass)
	assert.Equal(t, party_state.PartyMapAtInn(references.Britain), stored.PartyMap)
}

func newInnTestState(
	t *testing.T,
	merchantID references.MerchantID,
	activeCount int,
) (*GameState, *map_units.NPCFriendly, MerchantSession, *MockSystemCallbacks) {
	t.Helper()
	state, npc := newOpenMerchantTestState(t, merchantID)
	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	installMerchantTestNPC(t, state, npc)
	state.MapState.PlayerLocation.Position = references.Position{X: 15, Y: 15}
	prepareInnTestRoster(state, activeCount)
	state.PartyState.Inventory.Gold.Set(999)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	callbacks.Reset()
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	return state, npc, session, callbacks
}

func prepareInnTestRoster(state *GameState, activeCount int) {
	for index := range party_state.NRosterSlots {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapUnrecruited
		character.MonthsAtInn = 0xff
		character.SetStatus(party_state.Good)
		character.SetMaxHp(90)
		character.SetHp(40)
		character.SetMp(3)
		character.Intelligence = 10
	}
	for index := range activeCount {
		state.PartyState.GetCharacter(index).PartyMap = party_state.PartyMapActive
	}
	state.activePlayerIndex = -1
}

func stateMerchantNPC(t *testing.T, state *GameState) *map_units.NPCFriendly {
	t.Helper()
	for _, unit := range *state.CurrentNPCAIController.GetNpcs() {
		if npc, ok := unit.(*map_units.NPCFriendly); ok {
			return npc
		}
	}
	t.Fatal("merchant NPC is not loaded")

	return nil
}

func innRestRequest(state InnShopState) InnRestRequest {
	return InnRestRequest{
		ReviewedMerchantID: state.Merchant.ID, ReviewedPricingCharacterIndex: state.PricingCharacterIndex,
		ReviewedRosterFingerprint: state.RosterFingerprint, ReviewedCharge: state.Rest.Charge,
		ReviewedGoldBefore: state.Gold, ReviewedDateTimeBefore: state.Rest.DateTimeBefore,
		ReviewedDateTimeAfter: state.Rest.DateTimeAfter,
	}
}

func innLeaveRequest(state InnShopState, row InnLeaveRow) InnLeaveRequest {
	return InnLeaveRequest{
		ReviewedMerchantID: state.Merchant.ID, ReviewedPricingCharacterIndex: state.PricingCharacterIndex,
		ReviewedRosterFingerprint: state.RosterFingerprint, Character: row.Identity,
		ReviewedCharacterIndex: row.RosterIndex,
	}
}

func innPickupRequest(state InnShopState, row InnGuestRow) InnPickupRequest {
	return InnPickupRequest{
		ReviewedMerchantID: state.Merchant.ID, ReviewedPricingCharacterIndex: state.PricingCharacterIndex,
		ReviewedRosterFingerprint: state.RosterFingerprint, Character: row.Identity,
		ReviewedCharacterIndex: row.RosterIndex, ReviewedCharge: row.Charge,
		ReviewedGoldBefore: state.Gold,
	}
}

func currentInnRosterNames(state *GameState) []string {
	names := make([]string, party_state.NRosterSlots)
	for index := range names {
		names[index] = state.PartyState.GetCharacter(index).GetNameAsString()
	}

	return names
}

func shopRosterCharacters(_ InnShopState, state *GameState) []PartyRosterCharacterSnapshot {
	return state.CurrentPartyRosterSnapshot().Characters
}

func innRosterSnapshotNames(characters []PartyRosterCharacterSnapshot) []string {
	names := make([]string, len(characters))
	for index, character := range characters {
		names[index] = character.Name
	}

	return names
}
