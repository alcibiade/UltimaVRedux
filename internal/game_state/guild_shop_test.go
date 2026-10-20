package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestCalculateMerchantBuyPriceUsesOriginalIntegerOrdering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		basePrice    int
		intelligence int
		want         int
	}{
		{basePrice: 190, intelligence: 10, want: 323},
		{basePrice: 255, intelligence: 10, want: 433},
		{basePrice: 12, intelligence: 10, want: 20},
		{basePrice: 160, intelligence: 10, want: 272},
		{basePrice: 200, intelligence: 10, want: 340},
		{basePrice: 11, intelligence: 10, want: 18},
		{basePrice: 185, intelligence: 10, want: 314},
		{basePrice: 225, intelligence: 10, want: 382},
		{basePrice: 25, intelligence: 10, want: 42},
		{basePrice: 255, intelligence: 1, want: 502},
		{basePrice: 255, intelligence: 30, want: 280},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, CalculateMerchantBuyPrice(test.basePrice, test.intelligence))
	}
}

func TestGetGuildShopStateReturnsTrustedCurrentQuotes(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildPaws)
	state.PartyState.Inventory.Gold.Set(300)
	state.PartyState.Inventory.Provisions.Keys.Set(98)
	state.PartyState.Inventory.Provisions.Gems.Set(10)
	state.PartyState.Inventory.Provisions.Torches.Set(0)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	shop, err := state.GetGuildShopState(session)
	require.NoError(t, err)

	assert.Equal(t, references.MerchantGuildPaws, shop.Merchant.ID)
	assert.Equal(t, 0, shop.PricingCharacterIndex)
	assert.Equal(t, 10, shop.PricingIntelligence)
	assert.Equal(t, uint16(300), shop.Gold)
	assert.Equal(t, [3]references.Provision{references.Key, references.Gem, references.Torches}, [3]references.Provision{
		shop.Offers[0].Provision,
		shop.Offers[1].Provision,
		shop.Offers[2].Provision,
	})
	assert.Equal(t, GuildOfferQuote{
		Provision:      references.Key,
		BundleQuantity: 3,
		BasePrice:      160,
		PricePerBundle: 272,
		Gold:           300,
		QuantityBefore: 98,
		MaxBundles:     1,
	}, shop.Offers[0])
	assert.Zero(t, shop.Offers[1].MaxBundles)
	assert.Equal(t, uint16(16), shop.Offers[2].MaxBundles)
}

func TestReviewGuildShopByMerchantIDMatchesProductionQuotesWithoutMutation(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildPaws)
	state.SetRandomSeed(8675309)
	state.PartyState.Inventory.Gold.Set(777)
	state.PartyState.Inventory.Provisions.Keys.Set(12)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.SetupEnvironmentalSystemCallbacks()
	beforeLocation := state.MapState.PlayerLocation
	beforeDate := state.DateTime
	beforeRNGCalls := state.GetRNG().Calls()
	beforeSequence := state.merchantSessionSequence
	beforeController := state.CurrentNPCAIController

	review, err := state.ReviewGuildShopByMerchantID(references.MerchantGuildPaws)

	require.NoError(t, err)
	assert.Equal(t, references.MerchantGuildPaws, review.State.Merchant.ID)
	assert.Equal(t, npc.GetKeySpriteIndex(), review.MerchantSpriteIndex)
	assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeRNGCalls, state.GetRNG().Calls())
	assert.Equal(t, beforeSequence, state.merchantSessionSequence)
	assert.Same(t, beforeController, state.CurrentNPCAIController)
	assert.Zero(t, callbacks.MessageCallCount)
	assert.Zero(t, callbacks.AudioCallCount)
	assert.Zero(t, callbacks.VisualCallCount)
	assert.Zero(t, callbacks.ScreenCallCount)
	assert.Zero(t, callbacks.FlowCallCount)
	assert.Zero(t, callbacks.TalkCallCount)

	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	production, err := state.GetGuildShopState(session)
	require.NoError(t, err)
	assert.Equal(t, production, review.State)
}

func TestPurchaseGuildOfferCommitsMultiBundleOrderAndRefreshesOnce(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	state.PartyState.Inventory.Gold.Set(2000)
	state.PartyState.Inventory.Provisions.Keys.Set(4)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() {
		refreshes++
		assert.Equal(t, uint16(1031), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, uint16(13), state.PartyState.Inventory.Provisions.Keys.Get())
	}
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	beforeTime := state.DateTime
	beforeCalls := state.GetRNG().Calls()

	result, err := state.PurchaseGuildOffer(session, GuildPurchaseRequest{
		Provision:   references.Key,
		BundleCount: 3,
	})

	require.NoError(t, err)
	assert.Equal(t, GuildPurchaseResult{
		Provision:       references.Key,
		BasePrice:       190,
		PricePerBundle:  323,
		BundleQuantity:  3,
		BundleCount:     3,
		OfferedQuantity: 9,
		TotalCost:       969,
		GoldBefore:      2000,
		GoldAfter:       1031,
		Theft:           0,
		QuantityBefore:  4,
		QuantityAfter:   13,
		ActualReceipt:   9,
	}, result)
	assert.Equal(t, uint16(1031), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint16(13), state.PartyState.Inventory.Provisions.Keys.Get())
	assert.Equal(t, beforeCalls, state.GetRNG().Calls())
	assert.Equal(t, beforeTime, state.DateTime)
	assert.Equal(t, 1, refreshes)
}

func TestPurchaseGuildOfferAppliesAutomaticFaulineiTheftExactlyOnce(t *testing.T) {
	const seed = 240211

	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Provisions.Torches.Set(5)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	expectedRNG := beradrng.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)

	result, err := state.PurchaseGuildOffer(session, GuildPurchaseRequest{
		Provision:   references.Torches,
		BundleCount: 4,
	})

	require.NoError(t, err)
	assert.Equal(t, expectedTheft, result.Theft)
	assert.Equal(t, uint16(4), result.BundleCount)
	assert.Equal(t, uint16(20), result.OfferedQuantity)
	assert.Equal(t, 80, result.TotalCost)
	assert.Equal(t, uint16(1000-result.TotalCost)-expectedTheft, result.GoldAfter)
	assert.Equal(t, uint16(25), result.QuantityAfter)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Equal(t, 1, refreshes)
}

func TestPurchaseGuildOfferYellSummonedFaulineiDoesNotSteal(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	state.SetRandomSeed(211)
	state.ShadowlordState.Summoned = true
	state.ShadowlordState.LastCalled = references.Faulinei
	state.PartyState.Inventory.Gold.Set(1000)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	result, err := state.PurchaseGuildOffer(session, GuildPurchaseRequest{
		Provision:   references.Gem,
		BundleCount: 2,
	})

	require.NoError(t, err)
	assert.Zero(t, result.Theft)
	assert.Zero(t, state.GetRNG().Calls())
}

func TestPurchaseGuildOfferRejectsZeroAndOverMaximumWithoutEffects(t *testing.T) {
	tests := []struct {
		name        string
		bundleCount uint16
	}{
		{name: "zero", bundleCount: 0},
		{name: "over maximum", bundleCount: 4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
			state.SetRandomSeed(211)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(1000)
			state.PartyState.Inventory.Provisions.Keys.Set(7)
			refreshes := 0
			state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			_, err = state.PurchaseGuildOffer(session, GuildPurchaseRequest{
				Provision:   references.Key,
				BundleCount: test.bundleCount,
			})

			require.ErrorIs(t, err, ErrInvalidGuildPurchase)
			assert.Equal(t, uint16(1000), state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, uint16(7), state.PartyState.Inventory.Provisions.Keys.Get())
			assert.Zero(t, state.GetRNG().Calls())
			assert.Zero(t, refreshes)
		})
	}
}

func TestQuoteGuildOfferCapsMaximumBundlesByAffordability(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildPaws)
	state.PartyState.Inventory.Gold.Set(815)
	state.PartyState.Inventory.Provisions.Keys.Set(0)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteGuildOffer(session, references.Key)

	require.NoError(t, err)
	assert.Equal(t, 272, quote.PricePerBundle)
	assert.Equal(t, uint16(2), quote.MaxBundles)
}

func TestPurchaseGuildOfferAllowsPartialFinalUsefulBundleAndSaturates(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	state.PartyState.Inventory.Gold.Set(5000)
	state.PartyState.Inventory.Provisions.Keys.Set(94)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	quote, err := state.QuoteGuildOffer(session, references.Key)
	require.NoError(t, err)
	assert.Equal(t, uint16(2), quote.MaxBundles)

	result, err := state.PurchaseGuildOffer(session, GuildPurchaseRequest{
		Provision:   references.Key,
		BundleCount: 2,
	})

	require.NoError(t, err)
	assert.Equal(t, uint16(2), result.BundleCount)
	assert.Equal(t, uint16(6), result.OfferedQuantity)
	assert.Equal(t, 646, result.TotalCost)
	assert.Equal(t, uint16(94), result.QuantityBefore)
	assert.Equal(t, uint16(99), result.QuantityAfter)
	assert.Equal(t, uint16(5), result.ActualReceipt)
	assert.Equal(t, uint16(4354), result.GoldAfter)
	assert.Equal(t, 1, refreshes)
}

func TestQuoteGuildOfferHasZeroMaximumAtCapacity(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	state.PartyState.Inventory.Gold.Set(5000)
	state.PartyState.Inventory.Provisions.Keys.Set(99)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteGuildOffer(session, references.Key)

	require.NoError(t, err)
	assert.Zero(t, quote.MaxBundles)
}

func TestPurchaseGuildOfferPostPaymentZeroGoldStillCallsTheftOnce(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	state.SetRandomSeed(211)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	quote, err := state.QuoteGuildOffer(session, references.Key)
	require.NoError(t, err)
	state.PartyState.Inventory.Gold.Set(uint16(quote.PricePerBundle))

	result, err := state.PurchaseGuildOffer(session, GuildPurchaseRequest{
		Provision:   references.Key,
		BundleCount: 1,
	})

	require.NoError(t, err)
	assert.Zero(t, result.GoldAfter)
	assert.Zero(t, result.Theft)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestGuildOperationsRejectInvalidAndStaleSessionsBeforeMutation(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Provisions.Keys.Set(5)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
	assertUnchanged := func() {
		assert.Equal(t, uint16(1000), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, uint16(5), state.PartyState.Inventory.Provisions.Keys.Get())
		assert.Zero(t, state.GetRNG().Calls())
		assert.Zero(t, refreshes)
	}

	request := GuildPurchaseRequest{Provision: references.Key, BundleCount: 1}
	_, err := state.PurchaseGuildOffer(MerchantSession{}, request)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assertUnchanged()

	first, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.PurchaseGuildOffer(first, request)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assertUnchanged()

	current, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	state.PartyState.GetCharacter(current.PricingCharacterIndex()).SetStatus(party_state.Dead)
	_, err = state.PurchaseGuildOffer(current, request)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	assertUnchanged()
}

func TestGuildOperationsRejectWrongFamilyAndUnsupportedProvision(t *testing.T) {
	armsState, armsNPC := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	armsSession, err := armsState.BeginMerchantSession(armsNPC)
	require.NoError(t, err)
	_, err = armsState.GetGuildShopState(armsSession)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)

	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.QuoteGuildOffer(session, references.Food)
	require.ErrorIs(t, err, ErrInvalidGuildPurchase)
}

func TestGuildSessionKeepsCapturedBuyerWhenActiveSelectionChanges(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantGuildNewMagincia)
	avatar := state.PartyState.GetCharacter(0)
	avatar.Intelligence = 1
	selected := state.PartyState.GetCharacter(1)
	selected.PartyMap = party_state.PartyMapActive
	selected.SetStatus(party_state.Poisoned)
	selected.Intelligence = 30
	require.True(t, state.SetActivePlayer(1))
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	require.True(t, state.SetActivePlayer(0))

	quote, err := state.QuoteGuildOffer(session, references.Key)

	require.NoError(t, err)
	assert.Equal(t, 1, session.PricingCharacterIndex())
	assert.Equal(t, CalculateMerchantBuyPrice(190, 30), quote.PricePerBundle)
}
