package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestGetReagentShopStateProvidesCurrentPricesLimitsAndApprovalViews(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
	state.PartyState.Inventory.Gold.Set(500)
	state.PartyState.Inventory.Reagent.Set(references.Ginseng, 20)
	state.PartyState.Inventory.Reagent.Set(references.Garlic, 80)
	state.PartyState.Inventory.Reagent.Set(references.NightShade, 99)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	shop, err := state.GetReagentShopState(session)
	require.NoError(t, err)

	assert.Equal(t, references.MerchantReagentMoonglow, shop.Merchant.ID)
	assert.Equal(t, 0, shop.PricingCharacterIndex)
	assert.Equal(t, 10, shop.PricingIntelligence)
	assert.Equal(t, uint16(500), shop.Gold)
	require.Len(t, shop.Offers, 5)
	assert.Equal(t, []references.Reagent{
		references.Ginseng,
		references.Garlic,
		references.SpiderSilk,
		references.NightShade,
		references.MandrakeRoot,
	}, reagentQuoteIdentities(shop.Offers))
	assert.Equal(t, []string{
		"Ginseng",
		"Garlic",
		"Spider Silk",
		"Night Shade",
		"Mandrake Root",
	}, reagentQuoteNames(shop.Offers))

	ginseng := shop.Offers[0]
	assert.Equal(t, uint16(10), ginseng.BundleQuantity)
	assert.Equal(t, 20, ginseng.BasePrice)
	assert.Equal(t, CalculateMerchantBuyPrice(20, 10), ginseng.PricePerBundle)
	assert.Equal(t, uint16(20), ginseng.QuantityBefore)
	assert.Equal(t, uint16(8), ginseng.UsefulBundles)
	assert.Equal(t, uint16(14), ginseng.AffordableBundles)
	assert.Equal(t, uint16(8), ginseng.MaxBundles)
	assert.False(t, ginseng.Full)
	assert.Equal(t, ReagentOfferBlockNone, ginseng.BlockReason)
	require.Len(t, ginseng.ApprovalViews, int(ginseng.MaxBundles))
	for index, approval := range ginseng.ApprovalViews {
		bundleCount := uint16(index + 1)
		requested := uint16(10) * bundleCount
		quantityAfter := min(uint16(99), uint16(20)+requested)
		actualReceipt := quantityAfter - 20
		assert.Equal(t, references.Ginseng, approval.Reagent)
		assert.Equal(t, "Ginseng", approval.ReagentName)
		assert.Equal(t, bundleCount, approval.BundleCount)
		assert.Equal(t, uint16(8), approval.MaxBundles)
		assert.Equal(t, requested, approval.RequestedQuantity)
		assert.Equal(t, quantityAfter, approval.QuantityAfter)
		assert.Equal(t, actualReceipt, approval.ActualReceipt)
		assert.Equal(t, 34*int(bundleCount), approval.TotalPrice)
		assert.Equal(t, uint16(500-34*int(bundleCount)), approval.GoldAfterPayment)
		assert.Equal(t, actualReceipt < requested, approval.PaidOverflow)
		assert.Equal(t, requested-actualReceipt, approval.OverflowQuantity)
	}
	assert.True(t, ginseng.ApprovalViews[7].PaidOverflow)
	assert.Equal(t, uint16(79), ginseng.ApprovalViews[7].ActualReceipt)

	garlic := shop.Offers[1]
	assert.Equal(t, uint16(4), garlic.UsefulBundles)
	assert.Equal(t, uint16(4), garlic.MaxBundles)
	assert.Equal(t, uint16(24), garlic.ApprovalViews[3].RequestedQuantity)
	assert.Equal(t, uint16(99), garlic.ApprovalViews[3].QuantityAfter)
	assert.Equal(t, uint16(19), garlic.ApprovalViews[3].ActualReceipt)
	assert.True(t, garlic.ApprovalViews[3].PaidOverflow)

	nightShade := shop.Offers[3]
	assert.True(t, nightShade.Full)
	assert.Zero(t, nightShade.UsefulBundles)
	assert.Zero(t, nightShade.MaxBundles)
	assert.Equal(t, ReagentOfferBlockInventoryFull, nightShade.BlockReason)
	assert.True(t, nightShade.BlockReason.IsBlocked())
	assert.Empty(t, nightShade.ApprovalViews)

	shop.Offers[0].ApprovalViews[0].TotalPrice = 1
	fresh, err := state.GetReagentShopState(session)
	require.NoError(t, err)
	assert.Equal(t, 34, fresh.Offers[0].ApprovalViews[0].TotalPrice)
}

func TestQuoteReagentOfferBlocksUnaffordableBundlesBeforeApproval(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
	state.PartyState.Inventory.Gold.Set(21)
	state.PartyState.Inventory.Reagent.Set(references.MandrakeRoot, 0)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteReagentOffer(session, references.MandrakeRoot)
	require.NoError(t, err)

	assert.Equal(t, 22, quote.PricePerBundle)
	assert.Equal(t, uint16(99), quote.UsefulBundles)
	assert.Zero(t, quote.AffordableBundles)
	assert.Zero(t, quote.MaxBundles)
	assert.False(t, quote.Full)
	assert.Equal(t, ReagentOfferBlockUnaffordable, quote.BlockReason)
	assert.Empty(t, quote.ApprovalViews)

	_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
		Reagent:     references.MandrakeRoot,
		BundleCount: 1,
	})
	require.ErrorIs(t, err, ErrInvalidReagentPurchase)
	assert.Equal(t, uint16(21), state.PartyState.Inventory.Gold.Get())
	assert.Zero(t, state.PartyState.Inventory.Reagent.Get(references.MandrakeRoot))
	assert.Zero(t, state.GetRNG().Calls())
}

func TestQuoteReagentOfferSupportsMaximumUsefulBundleBoundary(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentCove)
	state.PartyState.Inventory.Gold.Set(9999)
	state.PartyState.Inventory.Reagent.Set(references.NightShade, 0)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteReagentOffer(session, references.NightShade)
	require.NoError(t, err)

	assert.Equal(t, uint16(99), quote.UsefulBundles)
	assert.Equal(t, uint16(99), quote.MaxBundles)
	require.Len(t, quote.ApprovalViews, 99)
	maximum := quote.ApprovalViews[98]
	assert.Equal(t, uint16(99), maximum.BundleCount)
	assert.Equal(t, uint16(99), maximum.RequestedQuantity)
	assert.Equal(t, uint16(99), maximum.ActualReceipt)
	assert.False(t, maximum.PaidOverflow)
	assert.Zero(t, maximum.OverflowQuantity)
}

func TestReviewReagentShopByMerchantIDIsPureAndDetached(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentCove)
	state.SetRandomSeed(8675309)
	state.PartyState.Inventory.Gold.Set(777)
	state.PartyState.Inventory.Reagent.Set(references.BlackPearl, 12)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.SetupEnvironmentalSystemCallbacks()
	beforeLocation := state.MapState.PlayerLocation
	beforeDate := state.DateTime
	beforeRNGCalls := state.GetRNG().Calls()
	beforeSequence := state.merchantSessionSequence
	beforeController := state.CurrentNPCAIController

	review, err := state.ReviewReagentShopByMerchantID(references.MerchantReagentCove)
	require.NoError(t, err)

	assert.Equal(t, references.MerchantReagentCove, review.State.Merchant.ID)
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
	production, err := state.GetReagentShopState(session)
	require.NoError(t, err)
	assert.Equal(t, production, review.State)

	review.State.Offers[0].ApprovalViews[0].TotalPrice = 1
	again, err := state.ReviewReagentShopByMerchantID(references.MerchantReagentCove)
	require.NoError(t, err)
	assert.NotEqual(t, 1, again.State.Offers[0].ApprovalViews[0].TotalPrice)
}

func TestReagentSessionKeepsCapturedBuyerWhenActiveSelectionChanges(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
	state.PartyState.Inventory.Gold.Set(500)
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

	quote, err := state.QuoteReagentOffer(session, references.Ginseng)
	require.NoError(t, err)

	assert.Equal(t, 1, session.PricingCharacterIndex())
	assert.Equal(t, CalculateMerchantBuyPrice(20, 30), quote.PricePerBundle)
}

func TestPurchaseReagentOfferCommitsPaidSaturationInAuthoritativeOrder(t *testing.T) {
	const seed = 240211

	state, npc := newOpenMerchantTestState(t, references.MerchantReagentCove)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(100)
	state.PartyState.Inventory.Reagent.Set(references.BloodMoss, 94)
	expectedRNG := deterministic_random.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)
	expectedTheft = min(expectedTheft, uint16(61))
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() {
		refreshes++
		assert.Equal(t, uint16(61)-expectedTheft, state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, uint16(99), state.PartyState.Inventory.Reagent.Get(references.BloodMoss))
	}
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	beforeTime := state.DateTime

	result, err := state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
		Reagent:     references.BloodMoss,
		BundleCount: 3,
	})
	require.NoError(t, err)

	assert.Equal(t, ReagentPurchaseResult{
		Reagent:           references.BloodMoss,
		ReagentName:       "Blood Moss",
		BundleQuantity:    2,
		BundleCount:       3,
		RequestedQuantity: 6,
		QuantityBefore:    94,
		QuantityAfter:     99,
		ActualReceipt:     5,
		PricePerBundle:    13,
		TotalPrice:        39,
		GoldBefore:        100,
		GoldAfterPayment:  61,
		Theft:             expectedTheft,
		GoldAfterTheft:    uint16(61) - expectedTheft,
	}, result)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Equal(t, 1, refreshes)
	assert.Equal(t, beforeTime, state.DateTime)
}

func TestPurchaseReagentOfferExactPaymentStillInvokesFaulineiOnce(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentYew)
	state.SetRandomSeed(211)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(20)
	state.PartyState.Inventory.Reagent.Set(references.SulfurAsh, 0)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	result, err := state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
		Reagent:     references.SulfurAsh,
		BundleCount: 1,
	})
	require.NoError(t, err)

	assert.Equal(t, 20, result.TotalPrice)
	assert.Zero(t, result.GoldAfterPayment)
	assert.Zero(t, result.Theft)
	assert.Zero(t, result.GoldAfterTheft)
	assert.Equal(t, uint16(12), result.ActualReceipt)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Equal(t, 1, refreshes)
}

func TestReagentOperationsRejectInvalidAndStaleRequestsWithoutEffects(t *testing.T) {
	t.Run("wrong session family", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantGuildPaws)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		_, err = state.GetReagentShopState(session)
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
	})

	t.Run("unsupported reagent and zero count", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentYew)
		state.SetRandomSeed(211)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent:     references.MandrakeRoot,
			BundleCount: 1,
		})
		require.ErrorIs(t, err, ErrInvalidReagentPurchase)
		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent:     references.Ginseng,
			BundleCount: 0,
		})
		require.ErrorIs(t, err, ErrInvalidReagentPurchase)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 500, 5, refreshes)
	})

	t.Run("gold changed after quote", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
		state.SetRandomSeed(211)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteReagentOffer(session, references.Ginseng)
		require.NoError(t, err)
		assert.NotZero(t, quote.MaxBundles)
		state.PartyState.Inventory.Gold.Set(0)

		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent:     references.Ginseng,
			BundleCount: 1,
		})
		require.ErrorIs(t, err, ErrInvalidReagentPurchase)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 0, 5, refreshes)
	})

	t.Run("quantity changed after quote", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
		state.SetRandomSeed(211)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		_, err = state.QuoteReagentOffer(session, references.Ginseng)
		require.NoError(t, err)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 99)

		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent:     references.Ginseng,
			BundleCount: 1,
		})
		require.ErrorIs(t, err, ErrInvalidReagentPurchase)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 500, 99, refreshes)
	})

	t.Run("superseded session and ineligible buyer", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
		state.SetRandomSeed(211)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		first, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		current, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		request := ReagentPurchaseRequest{Reagent: references.Ginseng, BundleCount: 1}
		_, err = state.PurchaseReagentOffer(first, request)
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		state.PartyState.GetCharacter(current.PricingCharacterIndex()).SetStatus(party_state.Dead)
		_, err = state.PurchaseReagentOffer(current, request)
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 500, 5, refreshes)
	})

	t.Run("wrong floor", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		npc.SetFloor(state.MapState.PlayerLocation.Floor + 1)

		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent: references.Ginseng, BundleCount: 1,
		})
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 500, 5, refreshes)
	})

	t.Run("location changed", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		state.MapState.PlayerLocation.Location = references.Britain

		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent: references.Ginseng, BundleCount: 1,
		})
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 500, 5, refreshes)
	})

	t.Run("schedule closed", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		setMerchantDesiredSlot(t, state, npc, 0)

		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent: references.Ginseng, BundleCount: 1,
		})
		require.ErrorIs(t, err, ErrMerchantClosed)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 500, 5, refreshes)
	})

	t.Run("mounted after quote", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantReagentMoonglow)
		state.PartyState.Inventory.Gold.Set(500)
		state.PartyState.Inventory.Reagent.Set(references.Ginseng, 5)
		refreshes := 0
		state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		setPartyTestVehicle(state, references.HorseVehicle)

		_, err = state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
			Reagent: references.Ginseng, BundleCount: 1,
		})
		require.ErrorIs(t, err, ErrMerchantHorsebackRefusal)
		assertReagentPurchaseUnchanged(t, state, references.Ginseng, 500, 5, refreshes)
	})
}

func TestPurchaseReagentOfferSupportsRepeatedPurchasesInOneSession(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentYew)
	state.PartyState.Inventory.Gold.Set(100)
	state.PartyState.Inventory.Reagent.Set(references.SpiderSilk, 0)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	beforeTime := state.DateTime

	first, err := state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
		Reagent:     references.SpiderSilk,
		BundleCount: 1,
	})
	require.NoError(t, err)
	second, err := state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
		Reagent:     references.SpiderSilk,
		BundleCount: 2,
	})
	require.NoError(t, err)

	assert.Equal(t, uint16(0), first.QuantityBefore)
	assert.Equal(t, uint16(2), first.QuantityAfter)
	assert.Equal(t, uint16(2), second.QuantityBefore)
	assert.Equal(t, uint16(6), second.QuantityAfter)
	assert.Equal(t, 26, second.TotalPrice)
	assert.Equal(t, uint16(61), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, 2, refreshes)
	assert.Zero(t, state.GetRNG().Calls())
	assert.Equal(t, beforeTime, state.DateTime)
}

func TestPurchasedReagentsAreConsumedByExecuteMixing(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantReagentYew)
	state.PartyState.Inventory.Gold.Set(100)
	state.PartyState.Inventory.Reagent.Set(references.SulfurAsh, 0)
	state.PartyState.Inventory.Spells.Set(references.InLor, 0)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	result, err := state.PurchaseReagentOffer(session, ReagentPurchaseRequest{
		Reagent:     references.SulfurAsh,
		BundleCount: 1,
	})
	require.NoError(t, err)
	require.Equal(t, uint16(12), result.ActualReceipt)

	outcome := state.ExecuteMixing(MixRequest{
		Spell:       references.InLor,
		Quantity:    3,
		ReagentMask: references.GetSpellReagents(references.InLor),
	})

	assert.True(t, outcome.Success)
	assert.Equal(t, uint16(9), state.PartyState.Inventory.Reagent.Get(references.SulfurAsh))
	assert.Equal(t, uint16(3), state.PartyState.Inventory.Spells.Get(references.InLor))
}

func reagentQuoteIdentities(quotes []ReagentOfferQuote) []references.Reagent {
	identities := make([]references.Reagent, len(quotes))
	for index, quote := range quotes {
		identities[index] = quote.Reagent
	}

	return identities
}

func reagentQuoteNames(quotes []ReagentOfferQuote) []string {
	names := make([]string, len(quotes))
	for index, quote := range quotes {
		names[index] = quote.ReagentName
	}

	return names
}

func assertReagentPurchaseUnchanged(
	t *testing.T,
	state *GameState,
	reagent references.Reagent,
	wantGold uint16,
	wantQuantity uint16,
	refreshes int,
) {
	t.Helper()
	assert.Equal(t, wantGold, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, wantQuantity, state.PartyState.Inventory.Reagent.Get(reagent))
	assert.Zero(t, state.GetRNG().Calls())
	assert.Zero(t, refreshes)
}
