package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestHealerShopStateUsesAllSevenFixedPriceRows(t *testing.T) {
	tests := []struct {
		merchantID references.MerchantID
		prices     [3]uint16
	}{
		{merchantID: references.MerchantHealerMinoc, prices: [3]uint16{0, 0, 200}},
		{merchantID: references.MerchantHealerTrinsic, prices: [3]uint16{25, 40, 215}},
		{merchantID: references.MerchantHealerSkaraBrae, prices: [3]uint16{30, 45, 225}},
		{merchantID: references.MerchantHealerEastBritanny, prices: [3]uint16{35, 50, 237}},
		{merchantID: references.MerchantHealerCove, prices: [3]uint16{40, 55, 247}},
		{merchantID: references.MerchantHealerLycaeum, prices: [3]uint16{15, 60, 249}},
		{merchantID: references.MerchantHealerEmpathAbbey, prices: [3]uint16{10, 65, 262}},
	}

	for _, test := range tests {
		t.Run(string(test.merchantID), func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, test.merchantID)
			state.PartyState.Inventory.Gold.Set(999)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			shop, err := state.GetHealerShopState(session)

			require.NoError(t, err)
			assert.Equal(t, test.merchantID, shop.Merchant.ID)
			assert.Equal(t, uint16(999), shop.Gold)
			for index, service := range [...]references.HealerService{
				references.HealerCure,
				references.HealerHeal,
				references.HealerResurrect,
			} {
				assert.Equal(t, service, shop.Services[index].Service)
				assert.Equal(t, test.prices[index], shop.Services[index].Charge)
				assert.True(t, shop.Services[index].Affordable)
			}
			if test.merchantID == references.MerchantHealerMinoc {
				assert.Equal(t, uint16(20), shop.Services[0].SourcePrice)
				assert.Equal(t, uint16(35), shop.Services[1].SourcePrice)
				assert.Equal(t, HealerChargeFree, shop.Services[0].ChargeKind)
				assert.Equal(t, HealerChargeFree, shop.Services[1].ChargeKind)
			}
		})
	}
}

func TestHealerShopStateProjectsServiceEligibilityAndResurrection(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantHealerEastBritanny)
	state.PartyState.Karma.Value = 50
	poisoned := configureHealerTestCharacter(state, 0, party_state.Poisoned, 42, 60)
	poisoned.Exp = 400
	poisoned.Level = 3
	configureHealerTestCharacter(state, 1, party_state.Sleep, 18, 60)
	dead := configureHealerTestCharacter(state, 2, party_state.Dead, 0, 150)
	dead.Class = party_state.Wizard
	dead.Intelligence = 20
	dead.SetMp(2)
	dead.Exp = 900
	dead.Level = 5
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	shop, err := state.GetHealerShopState(session)

	require.NoError(t, err)
	require.Len(t, shop.Recipients, 3)
	poisonedState := shop.Recipients[0]
	assert.Equal(t, HealerEligible, poisonedState.Treatments[references.HealerCure].Eligibility)
	assert.Equal(t, party_state.Good, poisonedState.Treatments[references.HealerCure].StatusAfter)
	assert.Equal(t, uint16(42), poisonedState.Treatments[references.HealerCure].HPAfter)
	assert.Equal(t, HealerEligible, poisonedState.Treatments[references.HealerHeal].Eligibility)
	assert.Equal(t, party_state.Poisoned, poisonedState.Treatments[references.HealerHeal].StatusAfter)
	assert.Equal(t, uint16(60), poisonedState.Treatments[references.HealerHeal].HPAfter)

	sleepingState := shop.Recipients[1]
	assert.Equal(t, HealerNotPoisoned, sleepingState.Treatments[references.HealerCure].Eligibility)
	assert.Equal(t, HealerEligible, sleepingState.Treatments[references.HealerHeal].Eligibility)
	assert.Equal(t, party_state.Sleep, sleepingState.Treatments[references.HealerHeal].StatusAfter)
	assert.Equal(t, uint16(60), sleepingState.Treatments[references.HealerHeal].HPAfter)

	deadState := shop.Recipients[2]
	assert.Equal(t, HealerDeadCannotHeal, deadState.Treatments[references.HealerHeal].Eligibility)
	projection := deadState.Treatments[references.HealerResurrect]
	assert.Equal(t, HealerEligible, projection.Eligibility)
	assert.Equal(t, party_state.Good, projection.StatusAfter)
	assert.Equal(t, uint16(450), projection.ExpAfter)
	assert.Equal(t, byte(4), projection.LevelAfter)
	assert.Equal(t, uint16(120), projection.MaxHPAfter)
	assert.Equal(t, uint16(120), projection.HPAfter)
	assert.Equal(t, byte(20), projection.MPAfter)
}

func TestReviewHealerShopMatchesProductionWithoutMutation(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantHealerCove)
	state.SetRandomSeed(208)
	state.PartyState.Inventory.Gold.Set(500)
	configureHealerTestCharacter(state, 0, party_state.Poisoned, 20, 60)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.SetupEnvironmentalSystemCallbacks()
	beforeRNGCalls := state.GetRNG().Calls()
	beforeSequence := state.merchantSessionSequence

	review, err := state.ReviewHealerShopByMerchantID(references.MerchantHealerCove)

	require.NoError(t, err)
	assert.Equal(t, references.MerchantHealerCove, review.State.Merchant.ID)
	assert.Equal(t, npc.GetKeySpriteIndex(), review.MerchantSpriteIndex)
	assert.Equal(t, beforeRNGCalls, state.GetRNG().Calls())
	assert.Equal(t, beforeSequence, state.merchantSessionSequence)
	assert.Zero(t, callbacks.MessageCallCount)
	assert.Zero(t, callbacks.AudioCallCount)
	assert.Zero(t, callbacks.VisualCallCount)
	assert.Zero(t, callbacks.ScreenCallCount)

	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	production, err := state.GetHealerShopState(session)
	require.NoError(t, err)
	assert.Equal(t, production, review.State)
}

func TestPaidHealerCureDeductsThenStealsThenAppliesEffect(t *testing.T) {
	const seed = 208

	state, npc := newOpenMerchantTestState(t, references.MerchantHealerTrinsic)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(100)
	character := configureHealerTestCharacter(state, 0, party_state.Poisoned, 42, 60)
	statsRefreshes, inventoryRefreshes := trackHealerRefreshes(state)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	expectedRNG := deterministic_random.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)

	result, err := state.ProvideHealerService(session, HealerServiceRequest{
		Service:        references.HealerCure,
		CharacterIndex: 0,
	})

	require.NoError(t, err)
	assert.Equal(t, HealerChargePaid, result.ChargeKind)
	assert.Equal(t, uint16(25), result.Charge)
	assert.Equal(t, uint16(75), result.GoldAfterPayment)
	assert.Equal(t, expectedTheft, result.Theft)
	assert.Equal(t, uint16(75)-expectedTheft, result.GoldAfterTheft)
	assert.Equal(t, party_state.Poisoned, result.StatusBefore)
	assert.Equal(t, party_state.Good, result.StatusAfter)
	assert.Equal(t, uint16(42), result.HPAfter)
	assert.Equal(t, party_state.Good, character.GetStatus())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Equal(t, 1, *statsRefreshes)
	assert.Equal(t, 1, *inventoryRefreshes)
}

func TestPaidHealerAllCureCommitsAtomicallyWithOnePaymentAndTheft(t *testing.T) {
	const seed = 208

	state, npc := newOpenMerchantTestState(t, references.MerchantHealerTrinsic)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(100)
	first := configureHealerTestCharacter(state, 0, party_state.Poisoned, 42, 60)
	second := configureHealerTestCharacter(state, 1, party_state.Poisoned, 18, 60)
	statsRefreshes, inventoryRefreshes := trackHealerRefreshes(state)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	expectedRNG := deterministic_random.New(seed)
	expectedTheft := min(uint16(expectedRNG.Intn(64)+1), uint16(50))

	quote, err := state.QuoteHealerServiceForAll(session, references.HealerCure)
	require.NoError(t, err)
	require.Equal(t, 2, quote.Service.RecipientCount)
	require.Equal(t, uint16(50), quote.Service.Charge)
	result, err := state.ProvideHealerServiceToAll(session, references.HealerCure)

	require.NoError(t, err)
	assert.Equal(t, []int{0, 1}, result.CharacterIndexes)
	assert.Equal(t, uint16(50), result.Charge)
	assert.Equal(t, uint16(50), result.GoldAfterPayment)
	assert.Equal(t, expectedTheft, result.Theft)
	assert.Equal(t, uint16(50)-expectedTheft, result.GoldAfterTheft)
	assert.Equal(t, party_state.Good, first.GetStatus())
	assert.Equal(t, party_state.Good, second.GetStatus())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Equal(t, 1, *statsRefreshes)
	assert.Equal(t, 1, *inventoryRefreshes)
}

func TestHealerAllRejectsUnaffordableAggregateWithoutPartialTreatment(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantHealerTrinsic)
	state.SetRandomSeed(208)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(49)
	first := configureHealerTestCharacter(state, 0, party_state.Poisoned, 42, 60)
	second := configureHealerTestCharacter(state, 1, party_state.Poisoned, 18, 60)
	statsRefreshes, inventoryRefreshes := trackHealerRefreshes(state)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	_, err = state.ProvideHealerServiceToAll(session, references.HealerCure)

	require.ErrorIs(t, err, ErrHealerInsufficientFunds)
	assert.Equal(t, uint16(49), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, party_state.Poisoned, first.GetStatus())
	assert.Equal(t, party_state.Poisoned, second.GetStatus())
	assert.Zero(t, state.GetRNG().Calls())
	assert.Zero(t, *statsRefreshes)
	assert.Zero(t, *inventoryRefreshes)
}

func TestSkaraBraeHealerAllUsesCharityWhenNoPatientAffordable(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantHealerSkaraBrae)
	state.SetRandomSeed(208)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(29)
	first := configureHealerTestCharacter(state, 0, party_state.Poisoned, 42, 60)
	second := configureHealerTestCharacter(state, 1, party_state.Poisoned, 18, 60)
	statsRefreshes, inventoryRefreshes := trackHealerRefreshes(state)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteHealerServiceForAll(session, references.HealerCure)
	require.NoError(t, err)
	require.Equal(t, HealerChargeCharitable, quote.Service.ChargeKind)
	require.Zero(t, quote.Service.Charge)
	result, err := state.ProvideHealerServiceToAll(session, references.HealerCure)

	require.NoError(t, err)
	assert.Equal(t, HealerChargeCharitable, result.ChargeKind)
	assert.Equal(t, uint16(29), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, party_state.Good, first.GetStatus())
	assert.Equal(t, party_state.Good, second.GetStatus())
	assert.Zero(t, state.GetRNG().Calls())
	assert.Equal(t, 1, *statsRefreshes)
	assert.Zero(t, *inventoryRefreshes)
}

func TestHealerHealRestoresFullHPWithoutChangingSleepOrPoison(t *testing.T) {
	for _, status := range []party_state.CharacterStatus{party_state.Sleep, party_state.Poisoned, party_state.Charmed} {
		t.Run(string(rune(status)), func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantHealerMinoc)
			state.PartyState.Inventory.Gold.Set(0)
			character := configureHealerTestCharacter(state, 0, status, 18, 60)
			if status != party_state.Poisoned {
				configureHealerTestCharacter(state, 1, party_state.Good, 60, 60)
			}
			statsRefreshes, inventoryRefreshes := trackHealerRefreshes(state)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			result, err := state.ProvideHealerService(session, HealerServiceRequest{
				Service:        references.HealerHeal,
				CharacterIndex: 0,
			})

			require.NoError(t, err)
			assert.Equal(t, HealerChargeFree, result.ChargeKind)
			assert.Zero(t, result.Charge)
			assert.Equal(t, status, character.GetStatus())
			assert.Equal(t, uint16(60), character.GetCurrentHp())
			assert.Zero(t, state.GetRNG().Calls())
			assert.Equal(t, 1, *statsRefreshes)
			assert.Zero(t, *inventoryRefreshes)
		})
	}
}

func TestSkaraBraeProvidesUnaffordableCureAndHealAsCharity(t *testing.T) {
	for _, service := range []references.HealerService{references.HealerCure, references.HealerHeal} {
		t.Run(service.String(), func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantHealerSkaraBrae)
			state.SetRandomSeed(208)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(1)
			status := party_state.Poisoned
			character := configureHealerTestCharacter(state, 0, status, 18, 60)
			statsRefreshes, inventoryRefreshes := trackHealerRefreshes(state)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			result, err := state.ProvideHealerService(session, HealerServiceRequest{
				Service:        service,
				CharacterIndex: 0,
			})

			require.NoError(t, err)
			assert.Equal(t, HealerChargeCharitable, result.ChargeKind)
			assert.Zero(t, result.Charge)
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Gold.Get())
			assert.Zero(t, state.GetRNG().Calls())
			assert.Equal(t, 1, *statsRefreshes)
			assert.Zero(t, *inventoryRefreshes)
			if service == references.HealerCure {
				assert.Equal(t, party_state.Good, character.GetStatus())
				assert.Equal(t, uint16(18), character.GetCurrentHp())
			} else {
				assert.Equal(t, party_state.Poisoned, character.GetStatus())
				assert.Equal(t, uint16(60), character.GetCurrentHp())
			}
		})
	}
}

func TestHealerResurrectionAppliesProgressionBeforeFullRecovery(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantHealerEastBritanny)
	state.PartyState.Karma.Value = 50
	state.PartyState.Inventory.Gold.Set(1000)
	dead := configureHealerTestCharacter(state, 1, party_state.Dead, 0, 150)
	dead.Class = party_state.Wizard
	dead.Intelligence = 20
	dead.SetMp(2)
	dead.Exp = 900
	dead.Level = 5
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteHealerTreatment(session, HealerServiceRequest{
		Service:        references.HealerResurrect,
		CharacterIndex: 1,
	})
	require.NoError(t, err)
	result, err := state.ProvideHealerService(session, HealerServiceRequest{
		Service:        references.HealerResurrect,
		CharacterIndex: 1,
	})

	require.NoError(t, err)
	assert.Equal(t, quote.Projection.StatusAfter, result.StatusAfter)
	assert.Equal(t, quote.Projection.ExpAfter, result.ExpAfter)
	assert.Equal(t, quote.Projection.LevelAfter, result.LevelAfter)
	assert.Equal(t, quote.Projection.MaxHPAfter, result.MaxHPAfter)
	assert.Equal(t, quote.Projection.HPAfter, result.HPAfter)
	assert.Equal(t, quote.Projection.MPAfter, result.MPAfter)
	assert.Equal(t, party_state.Good, dead.GetStatus())
	assert.Equal(t, uint16(450), dead.Exp)
	assert.Equal(t, byte(4), dead.Level)
	assert.Equal(t, uint16(120), dead.GetMaxHp())
	assert.Equal(t, uint16(120), dead.GetCurrentHp())
	assert.Equal(t, byte(20), dead.GetCurrentMp())
}

func TestHealerRejectsInvalidUnneededAndUnaffordableRequestsWithoutEffects(t *testing.T) {
	tests := []struct {
		name      string
		merchant  references.MerchantID
		service   references.HealerService
		status    party_state.CharacterStatus
		hp        uint16
		maxHP     uint16
		gold      uint16
		wantError error
	}{
		{name: "cure good", merchant: references.MerchantHealerTrinsic, service: references.HealerCure, status: party_state.Good, hp: 40, maxHP: 60, gold: 100, wantError: ErrHealerServiceUnneeded},
		{name: "heal full", merchant: references.MerchantHealerTrinsic, service: references.HealerHeal, status: party_state.Good, hp: 60, maxHP: 60, gold: 100, wantError: ErrHealerServiceUnneeded},
		{name: "heal dead", merchant: references.MerchantHealerTrinsic, service: references.HealerHeal, status: party_state.Dead, hp: 0, maxHP: 60, gold: 100, wantError: ErrHealerServiceUnneeded},
		{name: "resurrect living", merchant: references.MerchantHealerTrinsic, service: references.HealerResurrect, status: party_state.Good, hp: 40, maxHP: 60, gold: 1000, wantError: ErrHealerServiceUnneeded},
		{name: "ordinary funds", merchant: references.MerchantHealerTrinsic, service: references.HealerHeal, status: party_state.Good, hp: 40, maxHP: 60, gold: 1, wantError: ErrHealerInsufficientFunds},
		{name: "Skara resurrection is not charity", merchant: references.MerchantHealerSkaraBrae, service: references.HealerResurrect, status: party_state.Dead, hp: 0, maxHP: 60, gold: 1, wantError: ErrHealerInsufficientFunds},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, test.merchant)
			state.SetRandomSeed(208)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(test.gold)
			character := configureHealerTestCharacter(state, 0, test.status, test.hp, test.maxHP)
			if test.status != party_state.Good && test.status != party_state.Poisoned {
				configureHealerTestCharacter(state, 1, party_state.Good, 60, 60)
			}
			statsRefreshes, inventoryRefreshes := trackHealerRefreshes(state)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			beforeStatus := character.GetStatus()
			beforeHP := character.GetCurrentHp()

			_, err = state.ProvideHealerService(session, HealerServiceRequest{
				Service:        test.service,
				CharacterIndex: 0,
			})

			require.ErrorIs(t, err, test.wantError)
			assert.Equal(t, test.gold, state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, beforeStatus, character.GetStatus())
			assert.Equal(t, beforeHP, character.GetCurrentHp())
			assert.Zero(t, state.GetRNG().Calls())
			assert.Zero(t, *statsRefreshes)
			assert.Zero(t, *inventoryRefreshes)
		})
	}
}

func TestHealerOperationsRejectWrongFamilyStaleRecipientAndUnknownService(t *testing.T) {
	armsState, armsNPC := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	armsSession, err := armsState.BeginMerchantSession(armsNPC)
	require.NoError(t, err)
	_, err = armsState.GetHealerShopState(armsSession)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)

	state, npc := newOpenMerchantTestState(t, references.MerchantHealerMinoc)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.QuoteHealerTreatment(session, HealerServiceRequest{
		Service:        references.HealerService(255),
		CharacterIndex: 0,
	})
	require.ErrorIs(t, err, ErrInvalidHealerRequest)
	_, err = state.QuoteHealerTreatment(session, HealerServiceRequest{
		Service:        references.HealerCure,
		CharacterIndex: party_state.MaxActiveParty,
	})
	require.ErrorIs(t, err, ErrInvalidHealerRequest)

	first, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.ProvideHealerService(first, HealerServiceRequest{
		Service:        references.HealerCure,
		CharacterIndex: 0,
	})
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
}

func configureHealerTestCharacter(
	state *GameState,
	index int,
	status party_state.CharacterStatus,
	hp uint16,
	maxHP uint16,
) *party_state.PlayerCharacter {
	character := state.PartyState.GetCharacter(index)
	character.PartyMap = party_state.PartyMapActive
	character.SetMaxHp(maxHP)
	character.SetHp(hp)
	character.SetStatus(status)

	return character
}

func trackHealerRefreshes(state *GameState) (*int, *int) {
	statsRefreshes := 0
	inventoryRefreshes := 0
	state.SystemCallbacks.Screen.UpdateStatsDisplay = func() { statsRefreshes++ }
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { inventoryRefreshes++ }

	return &statsRefreshes, &inventoryRefreshes
}
