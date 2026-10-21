package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestCalculateArmsSellPriceUsesOriginalIntegerOrdering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		basePrice    int
		intelligence int
		want         int
	}{
		{basePrice: 1, intelligence: 1, want: 1},
		{basePrice: 7, intelligence: 10, want: 3},
		{basePrice: 15, intelligence: 10, want: 5},
		{basePrice: 250, intelligence: 1, want: 8},
		{basePrice: 250, intelligence: 10, want: 76},
		{basePrice: 250, intelligence: 30, want: 226},
	}

	for _, test := range tests {
		assert.Equal(t, test.want, CalculateArmsSellPrice(test.basePrice, test.intelligence))
	}
}

func TestGetArmsShopStateReturnsCanonicalBuyAndCarriedSellQuotes(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Equipment.Set(references.LeatherHelm, 2)
	state.PartyState.Inventory.Equipment.Set(references.JewelShield, 1)
	state.PartyState.Inventory.Equipment.Set(references.Dagger, 3)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 7)
	state.PartyState.GetCharacter(0).SetEquippedItem(references.SlotWeapon, references.LongSword)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	shop, err := state.GetArmsShopState(session)
	require.NoError(t, err)

	assert.Equal(t, references.MerchantArmsBritain, shop.Merchant.ID)
	assert.Equal(t, 0, shop.PricingCharacterIndex)
	assert.Equal(t, 10, shop.PricingIntelligence)
	assert.Equal(t, uint16(1000), shop.Gold)
	assert.Equal(t, [7]references.Equipment{
		references.Dagger,
		references.Sling,
		references.Bow,
		references.Arrows,
		references.Crossbow,
		references.Quarrels,
		references.MagicBow,
	}, [7]references.Equipment{
		shop.BuyQuotes[0].Equipment,
		shop.BuyQuotes[1].Equipment,
		shop.BuyQuotes[2].Equipment,
		shop.BuyQuotes[3].Equipment,
		shop.BuyQuotes[4].Equipment,
		shop.BuyQuotes[5].Equipment,
		shop.BuyQuotes[6].Equipment,
	})
	for _, quote := range shop.BuyQuotes {
		assert.Equal(t, quote.Equipment.GetBasePrice(), quote.BasePrice)
		assert.Equal(t, CalculateMerchantBuyPrice(quote.BasePrice, 10), quote.UnitPrice)
	}
	assert.True(t, shop.BuyQuotes[3].AmmoRefill)
	assert.Equal(t, uint16(1), shop.BuyQuotes[3].MaxQuantity)
	assert.True(t, shop.BuyQuotes[5].AmmoRefill)
	require.Len(t, shop.SellQuotes, 4)
	assert.Equal(t, references.LeatherHelm, shop.SellQuotes[0].Equipment)
	assert.Equal(t, uint16(2), shop.SellQuotes[0].QuantityBefore)
	assert.Equal(t, references.JewelShield, shop.SellQuotes[1].Equipment)
	assert.False(t, shop.SellQuotes[1].Sellable)
	assert.Equal(t, references.Dagger, shop.SellQuotes[2].Equipment)
	assert.Equal(t, uint16(3), shop.SellQuotes[2].QuantityBefore)
	assert.Equal(t, references.Arrows, shop.SellQuotes[3].Equipment)
	assert.False(t, shop.SellQuotes[3].Sellable)
	assert.NotContains(t, []references.Equipment{
		shop.SellQuotes[0].Equipment,
		shop.SellQuotes[1].Equipment,
		shop.SellQuotes[2].Equipment,
		shop.SellQuotes[3].Equipment,
	}, references.LongSword, "equipped-only items are not carried inventory")
}

func TestArmsQuotesReportCurrentActivePartyReadiedQuantity(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Equipment.Set(references.Dagger, 2)

	state.PartyState.GetCharacter(0).SetEquippedItem(references.SlotWeapon, references.Dagger)
	secondActive := state.PartyState.GetCharacter(1)
	secondActive.PartyMap = party_state.PartyMapActive
	secondActive.SetStatus(party_state.Good)
	secondActive.SetEquippedItem(references.SlotWeapon, references.Dagger)

	inactiveInPartySlots := state.PartyState.GetCharacter(2)
	inactiveInPartySlots.SetEquippedItem(references.SlotWeapon, references.Dagger)
	activeOutsidePartySlots := state.PartyState.GetCharacter(party_state.MaxActiveParty)
	activeOutsidePartySlots.PartyMap = party_state.PartyMapActive
	activeOutsidePartySlots.SetEquippedItem(references.SlotWeapon, references.Dagger)

	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	buyQuote, err := state.QuoteArmsBuy(session, references.Dagger)
	require.NoError(t, err)
	assert.Equal(t, uint16(2), buyQuote.ReadiedQuantity)
	sellQuote, err := state.QuoteArmsSell(session, references.Dagger)
	require.NoError(t, err)
	assert.Equal(t, uint16(2), sellQuote.ReadiedQuantity)
	assert.Equal(t, uint16(2), sellQuote.QuantityBefore, "readied items do not alter carried quantity")
	assert.Equal(t, uint16(2), sellQuote.MaxQuantity)
	assert.True(t, sellQuote.Sellable)

	shop, err := state.GetArmsShopState(session)
	require.NoError(t, err)
	assert.Equal(t, uint16(2), shop.BuyQuotes[0].ReadiedQuantity)
	require.Len(t, shop.SellQuotes, 1)
	assert.Equal(t, uint16(2), shop.SellQuotes[0].ReadiedQuantity)

	review, err := state.ReviewArmsShopByMerchantID(references.MerchantArmsBritain)
	require.NoError(t, err)
	assert.Equal(t, uint16(2), review.State.BuyQuotes[0].ReadiedQuantity)
	assert.Equal(t, uint16(2), review.State.SellQuotes[0].ReadiedQuantity)

	secondActive.SetEquippedItem(references.SlotWeapon, references.Sling)
	buyQuote, err = state.QuoteArmsBuy(session, references.Dagger)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), buyQuote.ReadiedQuantity)
	shop, err = state.GetArmsShopState(session)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), shop.BuyQuotes[0].ReadiedQuantity)
	assert.Equal(t, uint16(1), shop.SellQuotes[0].ReadiedQuantity)
}

func TestArmsQuotesCountBothHandsOfDualWieldedWeapons(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Equipment.Set(references.Dagger, 1)
	for index := range 4 {
		member := state.PartyState.GetCharacter(index)
		member.PartyMap = party_state.PartyMapActive
		member.SetStatus(party_state.Good)
		member.SetEquippedItem(references.SlotWeapon, references.Dagger)
		member.SetEquippedItem(references.SlotShield, references.Dagger)
	}

	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	buyQuote, err := state.QuoteArmsBuy(session, references.Dagger)
	require.NoError(t, err)
	assert.Equal(t, uint16(8), buyQuote.ReadiedQuantity)
	sellQuote, err := state.QuoteArmsSell(session, references.Dagger)
	require.NoError(t, err)
	assert.Equal(t, uint16(8), sellQuote.ReadiedQuantity)
}

func TestReviewArmsShopByMerchantIDMatchesProductionWithoutMutation(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBordermarch)
	state.SetRandomSeed(8675309)
	state.PartyState.Inventory.Gold.Set(777)
	state.PartyState.Inventory.Equipment.Set(references.LongSword, 4)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.SetupEnvironmentalSystemCallbacks()
	beforeLocation := state.MapState.PlayerLocation
	beforeDate := state.DateTime
	beforeRNGCalls := state.GetRNG().Calls()
	beforeSequence := state.merchantSessionSequence
	beforeController := state.CurrentNPCAIController

	review, err := state.ReviewArmsShopByMerchantID(references.MerchantArmsBordermarch)

	require.NoError(t, err)
	assert.Equal(t, references.MerchantArmsBordermarch, review.State.Merchant.ID)
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
	production, err := state.GetArmsShopState(session)
	require.NoError(t, err)
	assert.Equal(t, production, review.State)
}

func TestReviewArmsShopRejectsUnknownAndNonArmsMerchants(t *testing.T) {
	state, _ := newOpenMerchantTestState(t, references.MerchantArmsBritain)

	_, err := state.ReviewArmsShopByMerchantID(references.MerchantGuildPaws)
	require.ErrorIs(t, err, ErrInvalidArmsRequest)
	_, err = state.ReviewArmsShopByMerchantID(references.MerchantID("arms.unknown"))
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
}

func TestBuyArmsEquipmentCommitsOrdinaryAggregateBatchOnce(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Equipment.Set(references.Bow, 2)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() {
		refreshes++
		assert.Equal(t, uint16(619), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, uint16(5), state.PartyState.Inventory.Equipment.Get(references.Bow))
	}
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	beforeRNGCalls := state.GetRNG().Calls()

	result, err := state.BuyArmsEquipment(session, ArmsBuyRequest{
		Equipment: references.Bow,
		Quantity:  3,
	})

	require.NoError(t, err)
	assert.Equal(t, ArmsBuyResult{
		Equipment:        references.Bow,
		BasePrice:        75,
		UnitPrice:        127,
		Quantity:         3,
		TotalCost:        381,
		GoldBefore:       1000,
		GoldAfterPayment: 619,
		GoldAfterTheft:   619,
		QuantityBefore:   2,
		QuantityAfter:    5,
		ActualReceipt:    3,
	}, result)
	assert.Equal(t, beforeRNGCalls, state.GetRNG().Calls())
	assert.Equal(t, 1, refreshes)
}

func TestBuyArmsEquipmentRefillsAmmunitionAndRefusesFullStack(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Equipment.Set(references.Arrows, 5)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	result, err := state.BuyArmsEquipment(session, ArmsBuyRequest{
		Equipment: references.Arrows,
		Quantity:  1,
	})

	require.NoError(t, err)
	assert.True(t, result.AmmoRefill)
	assert.Equal(t, 17, result.UnitPrice)
	assert.Equal(t, 17, result.TotalCost)
	assert.Equal(t, uint16(5), result.QuantityBefore)
	assert.Equal(t, uint16(99), result.QuantityAfter)
	assert.Equal(t, uint16(94), result.ActualReceipt)
	assert.Equal(t, uint16(983), result.GoldAfterTheft)
	assert.Equal(t, 1, refreshes)

	beforeRNGCalls := state.GetRNG().Calls()
	_, err = state.BuyArmsEquipment(session, ArmsBuyRequest{
		Equipment: references.Arrows,
		Quantity:  1,
	})
	require.ErrorIs(t, err, ErrArmsCapacity)
	assert.Equal(t, uint16(983), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint16(99), state.PartyState.Inventory.Equipment.Get(references.Arrows))
	assert.Equal(t, beforeRNGCalls, state.GetRNG().Calls())
	assert.Equal(t, 1, refreshes)
}

func TestBuyArmsEquipmentAppliesOneFaulineiCheckAfterAggregatePayment(t *testing.T) {
	const seed = 240211

	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(1000)
	state.PartyState.Inventory.Equipment.Set(references.Dagger, 4)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() {
		refreshes++
		assert.Equal(t, uint16(9), state.PartyState.Inventory.Equipment.Get(references.Dagger))
	}
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	expectedRNG := beradrng.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)

	result, err := state.BuyArmsEquipment(session, ArmsBuyRequest{
		Equipment: references.Dagger,
		Quantity:  5,
	})

	require.NoError(t, err)
	assert.Equal(t, 5, result.TotalCost)
	assert.Equal(t, uint16(995), result.GoldAfterPayment)
	assert.Equal(t, expectedTheft, result.Theft)
	assert.Equal(t, uint16(995)-expectedTheft, result.GoldAfterTheft)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Equal(t, 1, refreshes)
}

func TestBuyArmsEquipmentRejectsInvalidCapacityAndFundsWithoutEffects(t *testing.T) {
	tests := []struct {
		name      string
		equipment references.Equipment
		quantity  uint16
		gold      uint16
		owned     uint16
		wantError error
	}{
		{name: "outside dealer stock", equipment: references.LongSword, quantity: 1, gold: 9999, wantError: ErrInvalidArmsRequest},
		{name: "zero quantity", equipment: references.Dagger, quantity: 0, gold: 9999, wantError: ErrInvalidArmsRequest},
		{name: "ammunition aggregate", equipment: references.Arrows, quantity: 2, gold: 9999, wantError: ErrInvalidArmsRequest},
		{name: "capacity before funds", equipment: references.Dagger, quantity: 1, gold: 0, owned: 99, wantError: ErrArmsCapacity},
		{name: "over capacity", equipment: references.Dagger, quantity: 2, gold: 9999, owned: 98, wantError: ErrArmsCapacity},
		{name: "insufficient funds", equipment: references.Bow, quantity: 1, gold: 126, wantError: ErrArmsInsufficientFunds},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
			state.SetRandomSeed(211)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(test.gold)
			state.PartyState.Inventory.Equipment.Set(test.equipment, test.owned)
			refreshes := 0
			state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			_, err = state.BuyArmsEquipment(session, ArmsBuyRequest{
				Equipment: test.equipment,
				Quantity:  test.quantity,
			})

			require.ErrorIs(t, err, test.wantError)
			assert.Equal(t, test.gold, state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, test.owned, state.PartyState.Inventory.Equipment.Get(test.equipment))
			assert.Zero(t, state.GetRNG().Calls())
			assert.Zero(t, refreshes)
		})
	}
}

func TestArmsBuyRequotesCurrentCapacityAndGold(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(9999)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	quote, err := state.QuoteArmsBuy(session, references.Dagger)
	require.NoError(t, err)
	assert.Equal(t, uint16(99), quote.MaxQuantity)

	state.PartyState.Inventory.Equipment.Set(references.Dagger, 99)
	state.PartyState.Inventory.Gold.Set(0)
	_, err = state.BuyArmsEquipment(session, ArmsBuyRequest{Equipment: references.Dagger, Quantity: 1})
	require.ErrorIs(t, err, ErrArmsCapacity)
}

func TestArmsOperationsRejectStaleFamilyBuyerAndController(t *testing.T) {
	t.Run("wrong family", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantGuildPaws)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		_, err = state.GetArmsShopState(session)
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		_, err = state.QuoteArmsBuy(session, references.Dagger)
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		_, err = state.QuoteArmsSell(session, references.Dagger)
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
	})

	t.Run("superseded session", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
		state.PartyState.Inventory.Gold.Set(100)
		first, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		_, err = state.BeginMerchantSession(npc)
		require.NoError(t, err)

		_, err = state.BuyArmsEquipment(first, ArmsBuyRequest{Equipment: references.Dagger, Quantity: 1})
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assert.Equal(t, uint16(100), state.PartyState.Inventory.Gold.Get())
	})

	t.Run("buyer changed", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
		state.PartyState.Inventory.Gold.Set(100)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		state.PartyState.GetCharacter(session.PricingCharacterIndex()).SetStatus(party_state.Dead)

		_, err = state.BuyArmsEquipment(session, ArmsBuyRequest{Equipment: references.Dagger, Quantity: 1})
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assert.Equal(t, uint16(100), state.PartyState.Inventory.Gold.Get())
	})

	t.Run("controller changed", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
		state.PartyState.Inventory.Equipment.Set(references.LongSword, 1)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		state.CurrentNPCAIController = nil

		_, err = state.SellArmsEquipment(session, ArmsSellRequest{Equipment: references.LongSword, Quantity: 1})
		require.ErrorIs(t, err, ErrInvalidMerchantSession)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.LongSword))
	})
}

func TestQuoteArmsSellAllowsCarriedEquipmentOutsideDealerStock(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(100)
	state.PartyState.Inventory.Equipment.Set(references.LongSword, 3)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteArmsSell(session, references.LongSword)

	require.NoError(t, err)
	assert.True(t, quote.Sellable)
	assert.Equal(t, 70, quote.BasePrice)
	assert.Equal(t, 22, quote.UnitOffer)
	assert.Equal(t, uint16(3), quote.MaxQuantity)
}

func TestSellArmsEquipmentRejectsAmmunitionAndEveryZeroPriceSpecial(t *testing.T) {
	tests := []struct {
		equipment references.Equipment
		wantError error
	}{
		{references.Arrows, ErrArmsUnsellableAmmunition},
		{references.Quarrels, ErrArmsUnsellableAmmunition},
		{references.JewelShield, ErrArmsZeroPrice},
		{references.MysticArmour, ErrArmsZeroPrice},
		{references.ChaosSword, ErrArmsZeroPrice},
		{references.GlassSword, ErrArmsZeroPrice},
		{references.JeweledSword, ErrArmsZeroPrice},
		{references.MysticSword, ErrArmsZeroPrice},
		{references.Ankh, ErrArmsZeroPrice},
	}

	for _, test := range tests {
		t.Run(test.equipment.String(), func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
			state.SetRandomSeed(211)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(500)
			state.PartyState.Inventory.Equipment.Set(test.equipment, 1)
			refreshes := 0
			state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			quote, err := state.QuoteArmsSell(session, test.equipment)
			require.NoError(t, err)
			assert.False(t, quote.Sellable)
			assert.Zero(t, quote.MaxQuantity)
			_, err = state.SellArmsEquipment(session, ArmsSellRequest{Equipment: test.equipment, Quantity: 1})

			require.ErrorIs(t, err, test.wantError)
			assert.Equal(t, uint16(500), state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(test.equipment))
			assert.Zero(t, state.GetRNG().Calls())
			assert.Zero(t, refreshes)
		})
	}
}

func TestSellArmsEquipmentCommitsAggregateWithoutFaulinei(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.SetRandomSeed(211)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(100)
	state.PartyState.Inventory.Equipment.Set(references.LongSword, 4)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() {
		refreshes++
		assert.Equal(t, uint16(166), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.LongSword))
	}
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	result, err := state.SellArmsEquipment(session, ArmsSellRequest{
		Equipment: references.LongSword,
		Quantity:  3,
	})

	require.NoError(t, err)
	assert.Equal(t, ArmsSellResult{
		Equipment:      references.LongSword,
		BasePrice:      70,
		UnitOffer:      22,
		Quantity:       3,
		TotalOffer:     66,
		GoldBefore:     100,
		GoldAfter:      166,
		QuantityBefore: 4,
		QuantityAfter:  1,
	}, result)
	assert.Zero(t, state.GetRNG().Calls())
	assert.Equal(t, 1, refreshes)
}

func TestSellArmsEquipmentEnforcesGoldSafeQuantityCap(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Gold.Set(9950)
	state.PartyState.Inventory.Equipment.Set(references.LongSword, 5)
	refreshes := 0
	state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteArmsSell(session, references.LongSword)
	require.NoError(t, err)
	assert.Equal(t, uint16(2), quote.MaxQuantity)
	_, err = state.SellArmsEquipment(session, ArmsSellRequest{Equipment: references.LongSword, Quantity: 3})
	require.ErrorIs(t, err, ErrArmsGoldCapacity)
	assert.Equal(t, uint16(9950), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint16(5), state.PartyState.Inventory.Equipment.Get(references.LongSword))
	assert.Zero(t, refreshes)

	result, err := state.SellArmsEquipment(session, ArmsSellRequest{Equipment: references.LongSword, Quantity: 2})
	require.NoError(t, err)
	assert.Equal(t, 44, result.TotalOffer)
	assert.Equal(t, uint16(9994), result.GoldAfter)
	assert.Equal(t, uint16(3), result.QuantityAfter)
	assert.Equal(t, 1, refreshes)
}

func TestSellArmsEquipmentRefusesZeroGoldCapacityAndStaleQuantityWithoutEffects(t *testing.T) {
	tests := []struct {
		name      string
		gold      uint16
		carried   uint16
		quantity  uint16
		wantError error
	}{
		{name: "zero quantity", gold: 0, carried: 2, quantity: 0, wantError: ErrInvalidArmsRequest},
		{name: "over carried", gold: 0, carried: 2, quantity: 3, wantError: ErrInvalidArmsRequest},
		{name: "zero gold-safe maximum", gold: 9990, carried: 2, quantity: 1, wantError: ErrArmsGoldCapacity},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
			state.SetRandomSeed(211)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(test.gold)
			state.PartyState.Inventory.Equipment.Set(references.LongSword, test.carried)
			refreshes := 0
			state.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			_, err = state.SellArmsEquipment(session, ArmsSellRequest{
				Equipment: references.LongSword,
				Quantity:  test.quantity,
			})

			require.ErrorIs(t, err, test.wantError)
			assert.Equal(t, test.gold, state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, test.carried, state.PartyState.Inventory.Equipment.Get(references.LongSword))
			assert.Zero(t, state.GetRNG().Calls())
			assert.Zero(t, refreshes)
		})
	}
}

func TestArmsSellRequotesCurrentCarriedQuantity(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantArmsBritain)
	state.PartyState.Inventory.Equipment.Set(references.LongSword, 5)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	quote, err := state.QuoteArmsSell(session, references.LongSword)
	require.NoError(t, err)
	assert.Equal(t, uint16(5), quote.MaxQuantity)

	state.PartyState.Inventory.Equipment.Set(references.LongSword, 1)
	_, err = state.SellArmsEquipment(session, ArmsSellRequest{Equipment: references.LongSword, Quantity: 2})
	require.ErrorIs(t, err, ErrInvalidArmsRequest)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.LongSword))
}
