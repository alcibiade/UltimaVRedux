package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestReviewTavernShopByMerchantIDCoversAllMenusAndIsPure(t *testing.T) {
	tests := []struct {
		merchantID references.MerchantID
		variant    references.TavernMenuVariant
		meal       string
		drink      string
		provision  bool
	}{
		{references.MerchantTavernMoonglow, references.TavernMenuMuttonAleRations, "Mutton", "Ale", true},
		{references.MerchantTavernBritain, references.TavernMenuMuttonAleRations, "Mutton", "Ale", true},
		{references.MerchantTavernJhelom, references.TavernMenuMuttonAleRations, "Mutton", "Ale", true},
		{references.MerchantTavernYew, references.TavernMenuRumBoar, "Boar", "Rum", false},
		{references.MerchantTavernNewMagincia, references.TavernMenuStoutFruitProvisions, "Fruit", "Stout", true},
		{references.MerchantTavernWestBritanny, references.TavernMenuWineCheese, "Cheese", "Wine", false},
		{references.MerchantTavernPaws, references.TavernMenuMuttonAleRations, "Mutton", "Ale", true},
		{references.MerchantTavernBuccaneersDen, references.TavernMenuRumBoar, "Boar", "Rum", false},
		{references.MerchantTavernLycaeum, references.TavernMenuMuttonAleRations, "Mutton", "Ale", true},
	}

	for _, test := range tests {
		t.Run(string(test.merchantID), func(t *testing.T) {
			state, npc := newOpenMerchantTestState(t, test.merchantID)
			state.SetRandomSeed(108)
			state.PartyState.Inventory.Gold.Set(777)
			state.PartyState.Inventory.Provisions.Food.Set(123)
			callbacks := NewMockSystemCallbacks(t)
			state.SystemCallbacks = callbacks.ToSystemCallbacks()
			state.SetupEnvironmentalSystemCallbacks()
			beforeLocation := state.MapState.PlayerLocation
			beforeDate := state.DateTime
			beforeCalls := state.GetRNG().Calls()
			beforeSequence := state.merchantSessionSequence
			beforeGold := state.PartyState.Inventory.Gold.Get()
			beforeFood := state.PartyState.Inventory.Provisions.Food.Get()

			review, err := state.ReviewTavernShopByMerchantID(test.merchantID)

			require.NoError(t, err)
			assert.Equal(t, test.merchantID, review.State.Merchant.ID)
			assert.Equal(t, test.variant, review.State.Menu.Variant)
			assert.Equal(t, test.meal, review.State.Menu.Meal.Name)
			assert.Equal(t, test.drink, review.State.Menu.Drink.Name)
			assert.Equal(t, test.provision, review.State.Menu.Provision.Available)
			assert.Equal(t, npc.GetKeySpriteIndex(), review.MerchantSpriteIndex)
			assert.False(t, review.State.CustomerEligible)
			if test.variant == references.TavernMenuWineCheese {
				assert.Len(t, review.State.Wines, 6)
			} else {
				assert.Empty(t, review.State.Wines)
				assert.Equal(t, TavernOrdinaryDrinkWineIndex, review.State.Drink.WineIndex)
			}
			assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
			assert.Equal(t, beforeDate, state.DateTime)
			assert.Equal(t, beforeCalls, state.GetRNG().Calls())
			assert.Equal(t, beforeSequence, state.merchantSessionSequence)
			assert.Equal(t, beforeGold, state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, beforeFood, state.PartyState.Inventory.Provisions.Food.Get())
			assert.Zero(t, callbacks.MessageCallCount)
			assert.Zero(t, callbacks.AudioCallCount)
			assert.Zero(t, callbacks.VisualCallCount)
			assert.Zero(t, callbacks.ScreenCallCount)
			assert.Zero(t, callbacks.FlowCallCount)
			assert.Zero(t, callbacks.TalkCallCount)

			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			production, err := state.GetTavernShopState(session)
			require.NoError(t, err)
			assert.Equal(t, production, review.State)
		})
	}
}

func TestTavernDinerCountUsesNonDeadActivePrefixAndIncludesSleep(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		character.PartyMap = party_state.PartyMapActive
		character.SetStatus(party_state.Good)
	}
	state.PartyState.GetCharacter(1).SetStatus(party_state.Sleep)
	state.PartyState.GetCharacter(2).SetStatus(party_state.Dead)
	state.PartyState.GetCharacter(4).PartyMap = party_state.PartyMapAtInn(references.Britain)
	state.PartyState.GetCharacter(5).PartyMap = party_state.PartyMapActive
	state.PartyState.GetCharacter(5).SetStatus(party_state.Good)
	state.PartyState.Inventory.Gold.Set(100)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteTavernMeal(session)

	require.NoError(t, err)
	assert.Equal(t, 3, quote.DinerCount, "active slots 0, 1 (sleeping), and 3 dine; dead 2 and records after inactive 4 do not")
	assert.Equal(t, uint16(12), quote.Charge)
}

func TestPurchaseTavernMealCommitsPaymentTheftFoodAndTransientVisual(t *testing.T) {
	const seed = 240211

	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	second := state.PartyState.GetCharacter(1)
	second.PartyMap = party_state.PartyMapActive
	second.SetStatus(party_state.Sleep)
	state.PartyState.GetCharacter(0).Intelligence = 30
	state.PartyState.Inventory.Gold.Set(100)
	state.PartyState.Inventory.Provisions.Food.Set(40)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.SetRandomSeed(seed)
	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	installMerchantTestNPC(t, state, npc)
	state.MapState.PlayerLocation.Position = references.Position{X: 10, Y: 10}
	north := state.MapState.PlayerLocation.Position.GetPositionUp()
	layeredMap := state.GetLayeredMapByCurrentLocation()
	require.NotNil(t, layeredMap)
	layeredMap.SetTileByLayer(map_state.MapOverrideLayer, north, indexes.TableMiddle)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	quote, err := state.QuoteTavernMeal(session)
	require.NoError(t, err)
	expectedRNG := beradrng.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)

	result, err := state.PurchaseTavernMeal(session, tavernMealRequest(state, quote))

	require.NoError(t, err)
	assert.Equal(t, TavernMealCompleted, result.Outcome)
	assert.Equal(t, 2, result.DinerCount)
	assert.Equal(t, uint16(8), result.Charge, "meal pricing ignores Intelligence")
	assert.Equal(t, uint16(92), result.GoldAfterPayment)
	assert.Equal(t, expectedTheft, result.Theft)
	assert.Equal(t, uint16(92)-expectedTheft, result.GoldAfterTheft)
	assert.Equal(t, uint16(42), result.FoodAfter)
	assert.Equal(t, uint16(2), result.ActualFood)
	assert.True(t, result.CustomerAfter)
	assert.Equal(t, TavernMealTableVisual{Show: true, Position: *north, SpriteIndex: indexes.TableFoodBottom}, result.TransientTableVisual)
	assert.Equal(t, indexes.SpriteIndex(indexes.TableMiddle), layeredMap.GetTileTopMapOnlyTile(north).Index, "meal visual must not mutate the map")
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	shop, err := state.GetTavernShopState(session)
	require.NoError(t, err)
	assert.True(t, shop.CustomerEligible)
}

func TestTavernMealTableVisualFallsBackSouthWithoutMutatingMap(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	installMerchantTestNPC(t, state, npc)
	state.MapState.PlayerLocation.Position = references.Position{X: 10, Y: 10}
	south := state.MapState.PlayerLocation.Position.GetPositionDown()
	layeredMap := state.GetLayeredMapByCurrentLocation()
	require.NotNil(t, layeredMap)
	layeredMap.SetTileByLayer(map_state.MapOverrideLayer, south, indexes.TableMiddle)

	visual := state.tavernMealTableVisual()

	assert.Equal(t, TavernMealTableVisual{Show: true, Position: *south, SpriteIndex: indexes.TableFoodTop}, visual)
	assert.Equal(t, indexes.SpriteIndex(indexes.TableMiddle), layeredMap.GetTileTopMapOnlyTile(south).Index)
}

func TestTavernDrinkQuotesUseOrdinaryDinersAndExactWinePrices(t *testing.T) {
	t.Run("ordinary", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
		for index := 1; index < 3; index++ {
			character := state.PartyState.GetCharacter(index)
			character.PartyMap = party_state.PartyMapActive
			character.SetStatus(party_state.Good)
		}
		state.PartyState.GetCharacter(0).Intelligence = 30
		state.PartyState.Inventory.Gold.Set(100)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		quote, err := state.QuoteTavernDrink(session)

		require.NoError(t, err)
		assert.Equal(t, 3, quote.DinerCount)
		assert.Equal(t, uint16(3), quote.Charge)
		assert.Equal(t, TavernOrdinaryDrinkWineIndex, quote.WineIndex)
	})

	t.Run("wine", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernWestBritanny)
		state.PartyState.GetCharacter(0).Intelligence = 1
		state.PartyState.Inventory.Gold.Set(500)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		quote, err := state.QuoteTavernWine(session, 4)

		require.NoError(t, err)
		assert.Equal(t, references.TavernWine{Name: "Moselle", Price: 275}, quote.Wine)
		assert.Equal(t, uint16(275), quote.Charge)
	})
}

func TestPurchaseTavernWineCommitsSelectedBottleAndOneIntoxicationIncrement(t *testing.T) {
	const seed = 240211
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernWestBritanny)
	state.PartyState.Inventory.Gold.Set(500)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.SetRandomSeed(seed)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	quote, err := state.QuoteTavernWine(session, 4)
	require.NoError(t, err)
	expectedRNG := beradrng.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)

	result, err := state.PurchaseTavernDrink(
		session,
		tavernDrinkRequest(state, quote, TavernDrinkApprovalPurchase),
	)

	require.NoError(t, err)
	assert.Equal(t, TavernDrinkPurchased, result.Outcome)
	assert.Equal(t, references.TavernWine{Name: "Moselle", Price: 275}, result.Wine)
	assert.Equal(t, uint16(225), result.GoldAfterPayment)
	assert.Equal(t, expectedTheft, result.Theft)
	assert.Equal(t, uint16(225)-expectedTheft, result.GoldAfterTheft)
	assert.Equal(t, 0, result.DrunkBefore)
	assert.Equal(t, 1, result.DrunkAfter)
	assert.True(t, result.CustomerAfter)
}

func TestStopTavernDrinkingEstablishesCustomerWithoutOtherEffects(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	state.drunk = 3
	state.smashed = 7
	state.PartyState.Karma.Value = 42
	state.PartyState.Inventory.Gold.Set(100)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.SetRandomSeed(108)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	drink, err := state.QuoteTavernDrink(session)
	require.NoError(t, err)
	assert.True(t, drink.Warning)
	assert.True(t, drink.StopAvailable)
	quote, err := state.QuoteTavernStop(session)
	require.NoError(t, err)

	result, err := state.StopTavernDrinking(session, TavernStopRequest{
		ReviewedMerchantID: quote.MerchantID,
		ReviewedDrunk:      quote.Drunk,
	})

	require.NoError(t, err)
	assert.Equal(t, uint16(100), result.GoldAfter)
	assert.Equal(t, TavernStopAccepted, result.Outcome)
	assert.Equal(t, 3, result.DrunkAfter)
	assert.Equal(t, 7, result.SmashedAfter)
	assert.Equal(t, 42, result.KarmaAfter)
	assert.False(t, result.CustomerBefore)
	assert.True(t, result.CustomerAfter)
	assert.Zero(t, state.GetRNG().Calls())

	state.drunk = 2
	_, err = state.QuoteTavernStop(session)
	require.ErrorIs(t, err, ErrInvalidTavernRequest)
}

func TestWarnedTavernDrinkContinueIsAtomic(t *testing.T) {
	t.Run("successful Continue", func(t *testing.T) {
		const seed = 240211
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
		state.drunk = 3
		state.smashed = 4
		state.PartyState.Karma.Value = 50
		state.PartyState.Inventory.Gold.Set(100)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.SetRandomSeed(seed)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernDrink(session)
		require.NoError(t, err)
		expectedRNG := beradrng.New(seed)
		expectedTheft := uint16(expectedRNG.Intn(64) + 1)
		request := tavernDrinkRequest(state, quote, TavernDrinkApprovalContinue)

		result, err := state.PurchaseTavernDrink(session, request)

		require.NoError(t, err)
		assert.Equal(t, TavernDrinkPurchased, result.Outcome)
		assert.Equal(t, uint16(99), result.GoldAfterPayment)
		assert.Equal(t, expectedTheft, result.Theft)
		assert.Equal(t, uint16(99)-expectedTheft, result.GoldAfterTheft)
		assert.Equal(t, 3, result.DrunkBefore)
		assert.Equal(t, 4, result.DrunkAfter)
		assert.Equal(t, 4, result.SmashedBefore)
		assert.Equal(t, 25, result.SmashedAfter)
		assert.Equal(t, 50, result.KarmaBefore)
		assert.Equal(t, 49, result.KarmaAfter)
		assert.True(t, result.CustomerAfter)
		assert.Equal(t, uint64(1), state.GetRNG().Calls())
	})

	t.Run("cancel", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
		state.drunk = 3
		state.smashed = 4
		state.PartyState.Karma.Value = 50
		state.PartyState.Inventory.Gold.Set(100)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernDrink(session)
		require.NoError(t, err)

		result, err := state.PurchaseTavernDrink(
			session,
			tavernDrinkRequest(state, quote, TavernDrinkApprovalCancel),
		)

		require.NoError(t, err)
		assert.Equal(t, TavernDrinkCancelled, result.Outcome)
		assert.Equal(t, uint16(100), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, 3, state.drunk)
		assert.Equal(t, 4, state.smashed)
		assert.Equal(t, 50, state.PartyState.Karma.Value)
		assert.Zero(t, state.tavernCustomerSessionSequence)
		assert.Zero(t, state.GetRNG().Calls())
	})

	t.Run("unaffordable Continue", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
		state.drunk = 3
		state.smashed = 4
		state.PartyState.Karma.Value = 50
		state.PartyState.Inventory.Gold.Set(0)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernDrink(session)
		require.NoError(t, err)
		assert.False(t, quote.Affordable)

		result, err := state.PurchaseTavernDrink(
			session,
			tavernDrinkRequest(state, quote, TavernDrinkApprovalContinue),
		)

		require.NoError(t, err)
		assert.Equal(t, TavernDrinkInsufficientFunds, result.Outcome)
		assert.Equal(t, 3, state.drunk)
		assert.Equal(t, 4, state.smashed)
		assert.Equal(t, 50, state.PartyState.Karma.Value)
		assert.Zero(t, state.tavernCustomerSessionSequence)
		assert.Zero(t, state.GetRNG().Calls())
	})

	t.Run("warning requires explicit Continue", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
		state.drunk = 3
		state.PartyState.Inventory.Gold.Set(100)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernDrink(session)
		require.NoError(t, err)

		_, err = state.PurchaseTavernDrink(
			session,
			tavernDrinkRequest(state, quote, TavernDrinkApprovalPurchase),
		)

		require.ErrorIs(t, err, ErrInvalidTavernRequest)
		assert.Equal(t, 3, state.drunk)
		assert.Zero(t, state.tavernCustomerSessionSequence)
	})
}

func TestSuccessfulTavernDrinkDoesNotCapOriginalDrunkCounter(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	state.drunk = 9999
	state.PartyState.Inventory.Gold.Set(10)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	quote, err := state.QuoteTavernDrink(session)
	require.NoError(t, err)

	result, err := state.PurchaseTavernDrink(
		session,
		tavernDrinkRequest(state, quote, TavernDrinkApprovalPurchase),
	)

	require.NoError(t, err)
	assert.Equal(t, TavernDrinkPurchased, result.Outcome)
	assert.Equal(t, 10_000, result.DrunkAfter)
}

func TestTavernCustomerEligibilityResetsAndFollowsCurrentSession(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	state.PartyState.Inventory.Gold.Set(100)
	first, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	meal, err := state.QuoteTavernMeal(first)
	require.NoError(t, err)
	_, err = state.PurchaseTavernMeal(first, tavernMealRequest(state, meal))
	require.NoError(t, err)
	shop, err := state.GetTavernShopState(first)
	require.NoError(t, err)
	assert.True(t, shop.CustomerEligible)

	current, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	_, err = state.GetTavernShopState(first)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
	shop, err = state.GetTavernShopState(current)
	require.NoError(t, err)
	assert.False(t, shop.CustomerEligible)

	state.drunk = 3
	stop, err := state.QuoteTavernStop(current)
	require.NoError(t, err)
	_, err = state.StopTavernDrinking(current, TavernStopRequest{
		ReviewedMerchantID: stop.MerchantID,
		ReviewedDrunk:      stop.Drunk,
	})
	require.NoError(t, err)
	state.EndMerchantSession(first)
	shop, err = state.GetTavernShopState(current)
	require.NoError(t, err)
	assert.True(t, shop.CustomerEligible, "ending a superseded token must not affect the current customer")

	state.EndMerchantSession(current)
	assert.Zero(t, state.tavernCustomerSessionSequence)
	_, err = state.GetTavernShopState(current)
	require.ErrorIs(t, err, ErrInvalidMerchantSession)
}

func TestPurchaseTavernProvisionsPreservesPerPackBoundaries(t *testing.T) {
	t.Run("complete uses captured Intelligence and one Faulinei", func(t *testing.T) {
		const seed = 240211
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(300)
		state.PartyState.Inventory.Provisions.Food.Set(0)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.SetRandomSeed(seed)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		state.PartyState.GetCharacter(0).Intelligence = 30
		quote, err := state.QuoteTavernProvisions(session)
		require.NoError(t, err)
		assert.Equal(t, 51, quote.PricePerPack, "session-captured Intelligence must remain authoritative")
		expectedRNG := beradrng.New(seed)
		expectedTheft := uint16(expectedRNG.Intn(64) + 1)

		result, err := state.PurchaseTavernProvisions(session, tavernProvisionRequest(state, quote, 2))

		require.NoError(t, err)
		assert.Equal(t, TavernProvisionCompleted, result.Outcome)
		assert.Equal(t, 2, result.PurchasedPacks)
		assert.Equal(t, uint16(50), result.OfferedFood)
		assert.Equal(t, uint16(50), result.ActualFood)
		assert.Equal(t, 102, result.TotalCost)
		assert.Equal(t, uint16(198), result.GoldAfterPurchases)
		assert.Equal(t, expectedTheft, result.Theft)
		assert.Equal(t, uint16(198)-expectedTheft, result.GoldAfterTheft)
		assert.True(t, result.CustomerAfter)
		assert.Equal(t, uint64(1), state.GetRNG().Calls())
	})

	t.Run("partial retains one pack and skips Faulinei", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(60)
		state.PartyState.Inventory.Provisions.Food.Set(0)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernProvisions(session)
		require.NoError(t, err)

		result, err := state.PurchaseTavernProvisions(session, tavernProvisionRequest(state, quote, 2))

		require.NoError(t, err)
		assert.Equal(t, TavernProvisionPartial, result.Outcome)
		assert.Equal(t, 1, result.PurchasedPacks)
		assert.Equal(t, uint16(25), result.ActualFood)
		assert.Equal(t, 51, result.TotalCost)
		assert.Equal(t, uint16(9), result.GoldAfterPurchases)
		assert.Equal(t, uint16(9), result.GoldAfterTheft)
		assert.Zero(t, result.Theft)
		assert.True(t, result.CustomerAfter)
		assert.Zero(t, state.GetRNG().Calls())
	})

	t.Run("saturation stops after charged partial final pack", func(t *testing.T) {
		const seed = 240211
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(200)
		state.PartyState.Inventory.Provisions.Food.Set(9980)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.SetRandomSeed(seed)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernProvisions(session)
		require.NoError(t, err)

		result, err := state.PurchaseTavernProvisions(session, tavernProvisionRequest(state, quote, 3))

		require.NoError(t, err)
		assert.Equal(t, TavernProvisionSaturated, result.Outcome)
		assert.Equal(t, 1, result.PurchasedPacks)
		assert.Equal(t, uint16(25), result.OfferedFood)
		assert.Equal(t, uint16(19), result.ActualFood)
		assert.Equal(t, 51, result.TotalCost)
		assert.NotZero(t, result.Theft)
		assert.True(t, result.CustomerAfter)
		assert.Equal(t, uint64(1), state.GetRNG().Calls())
	})

	t.Run("low-food charity ejects without theft or customer", func(t *testing.T) {
		const seed = 108
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(0)
		state.PartyState.Inventory.Provisions.Food.Set(1)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.SetRandomSeed(seed)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernProvisions(session)
		require.NoError(t, err)
		expectedRNG := beradrng.New(seed)
		expectedCharity := uint16(expectedRNG.Intn(2) + 1)

		result, err := state.PurchaseTavernProvisions(session, tavernProvisionRequest(state, quote, 2))

		require.NoError(t, err)
		assert.Equal(t, TavernProvisionCharityEjection, result.Outcome)
		assert.Zero(t, result.PurchasedPacks)
		assert.Equal(t, expectedCharity, result.ActualFood)
		assert.Zero(t, result.Theft)
		assert.False(t, result.CustomerAfter)
		assert.Equal(t, uint64(1), state.GetRNG().Calls())
	})

	t.Run("no-need ejection has no effects", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(0)
		state.PartyState.Inventory.Provisions.Food.Set(3)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernProvisions(session)
		require.NoError(t, err)

		result, err := state.PurchaseTavernProvisions(session, tavernProvisionRequest(state, quote, 1))

		require.NoError(t, err)
		assert.Equal(t, TavernProvisionEjection, result.Outcome)
		assert.Equal(t, uint16(3), result.FoodAfter)
		assert.Zero(t, result.ActualFood)
		assert.Zero(t, result.Theft)
		assert.False(t, result.CustomerAfter)
		assert.Zero(t, state.GetRNG().Calls())
	})
}

func TestQuoteTavernProvisionsProjectsOutcomesWithoutMutation(t *testing.T) {
	t.Run("complete and partial affordability", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(60)
		state.PartyState.Inventory.Provisions.Food.Set(0)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.SetRandomSeed(108)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		beforeRNG := state.GetRNG().Calls()

		quote, err := state.QuoteTavernProvisions(session)

		require.NoError(t, err)
		require.True(t, quote.HasValidApprovals())
		require.Equal(t, maximumTavernProvisionPacks, quote.MaximumUsefulPacks)
		require.Len(t, quote.Approvals, maximumTavernProvisionPacks)
		assert.Equal(t, TavernProvisionApproval{
			PackCount: 1, Outcome: TavernProvisionCompleted, PurchasedPacks: 1,
			TotalCost: 51, GoldAfterPurchases: 9, FoodAfter: 25, ActualFood: 25,
		}, quote.Approvals[0])
		assert.Equal(t, TavernProvisionApproval{
			PackCount: 2, Outcome: TavernProvisionPartial, PurchasedPacks: 1,
			TotalCost: 51, GoldAfterPurchases: 9, FoodAfter: 25, ActualFood: 25,
		}, quote.Approvals[1])
		assert.Equal(t, uint16(60), state.PartyState.Inventory.Gold.Get())
		assert.Zero(t, state.PartyState.Inventory.Provisions.Food.Get())
		assert.Equal(t, beforeRNG, state.GetRNG().Calls())

		for _, test := range []struct {
			name   string
			mutate func(*TavernProvisionApproval)
		}{
			{name: "pack count", mutate: func(value *TavernProvisionApproval) { value.PackCount++ }},
			{name: "outcome", mutate: func(value *TavernProvisionApproval) { value.Outcome = TavernProvisionEjection }},
			{name: "purchased packs", mutate: func(value *TavernProvisionApproval) { value.PurchasedPacks++ }},
			{name: "total cost", mutate: func(value *TavernProvisionApproval) { value.TotalCost++ }},
			{name: "gold after", mutate: func(value *TavernProvisionApproval) { value.GoldAfterPurchases++ }},
			{name: "food after", mutate: func(value *TavernProvisionApproval) { value.FoodAfter++ }},
			{name: "actual food", mutate: func(value *TavernProvisionApproval) { value.ActualFood++ }},
			{name: "charity", mutate: func(value *TavernProvisionApproval) { value.CharityPossible = true }},
		} {
			t.Run("rejects tampered "+test.name, func(t *testing.T) {
				tampered := quote
				tampered.Approvals = append([]TavernProvisionApproval(nil), quote.Approvals...)
				test.mutate(&tampered.Approvals[0])
				assert.False(t, tampered.HasValidApprovals())
			})
		}
	})

	t.Run("saturation bounds useful quantity", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(200)
		state.PartyState.Inventory.Provisions.Food.Set(9980)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		quote, err := state.QuoteTavernProvisions(session)

		require.NoError(t, err)
		require.Equal(t, 1, quote.MaximumUsefulPacks)
		require.Len(t, quote.Approvals, 1)
		assert.Equal(t, TavernProvisionApproval{
			PackCount: 1, Outcome: TavernProvisionSaturated, PurchasedPacks: 1,
			TotalCost: 51, GoldAfterPurchases: 149, FoodAfter: 9999, ActualFood: 19,
		}, quote.Approvals[0])
		assert.Equal(t, uint16(200), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, uint16(9980), state.PartyState.Inventory.Provisions.Food.Get())
		assert.Zero(t, state.GetRNG().Calls())
	})

	t.Run("insufficient funds distinguishes charity and ejection", func(t *testing.T) {
		for _, test := range []struct {
			name    string
			food    uint16
			outcome TavernProvisionOutcome
			charity bool
		}{
			{name: "charity possible", food: 1, outcome: TavernProvisionCharityEjection, charity: true},
			{name: "no need", food: 3, outcome: TavernProvisionEjection},
		} {
			t.Run(test.name, func(t *testing.T) {
				state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
				state.PartyState.Inventory.Gold.Set(0)
				state.PartyState.Inventory.Provisions.Food.Set(test.food)
				state.SetRandomSeed(108)
				session, err := state.BeginMerchantSession(npc)
				require.NoError(t, err)

				quote, err := state.QuoteTavernProvisions(session)

				require.NoError(t, err)
				require.NotEmpty(t, quote.Approvals)
				approval := quote.Approvals[0]
				assert.Equal(t, test.outcome, approval.Outcome)
				assert.Equal(t, test.charity, approval.CharityPossible)
				assert.Zero(t, approval.PurchasedPacks)
				assert.Zero(t, approval.TotalCost)
				assert.Equal(t, test.food, approval.FoodAfter)
				assert.Equal(t, test.food, state.PartyState.Inventory.Provisions.Food.Get())
				assert.Zero(t, state.GetRNG().Calls())
			})
		}
	})
}

func TestTavernInformationMatchingLockRepeatAndRNG(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	state.PartyState.Inventory.Gold.Set(200)
	state.PartyState.GetCharacter(0).Intelligence = 30
	state.drunk = 3
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	_, err = state.QuoteTavernInformation(session, "hone")
	require.ErrorIs(t, err, ErrTavernInformationLocked)
	stop, err := state.QuoteTavernStop(session)
	require.NoError(t, err)
	_, err = state.StopTavernDrinking(session, TavernStopRequest{
		ReviewedMerchantID: stop.MerchantID,
		ReviewedDrunk:      stop.Drunk,
	})
	require.NoError(t, err)

	_, err = state.QuoteTavernInformation(session, "")
	require.ErrorIs(t, err, ErrTavernInformationTopicUnknown)
	_, err = state.QuoteTavernInformation(session, "something unknown")
	require.ErrorIs(t, err, ErrTavernInformationTopicUnknown)
	assert.Equal(t, session.sequence, state.tavernCustomerSessionSequence)
	assert.Zero(t, state.GetRNG().Calls())

	quote, err := state.QuoteTavernInformation(session, "tell me honeypot")
	require.NoError(t, err)
	assert.Equal(t, "hone", quote.Topic.Key, "matching does not require a trailing word boundary")
	assert.Equal(t, 50, quote.Price, "information ignores Intelligence")
	first, err := state.PurchaseTavernInformation(session, tavernInformationRequest(state, quote))
	require.NoError(t, err)
	assert.Equal(t, TavernInformationCompleted, first.Outcome)
	assert.Equal(t, "Malik", first.Informant)
	assert.Equal(t, "Moonglow", first.Location)
	assert.Equal(t, uint16(150), first.GoldAfterPayment)
	assert.Zero(t, state.GetRNG().Calls(), "information wording and payment must not consume gameplay RNG")

	quote, err = state.QuoteTavernInformation(session, "hone")
	require.NoError(t, err)
	second, err := state.PurchaseTavernInformation(session, tavernInformationRequest(state, quote))
	require.NoError(t, err)
	assert.Equal(t, TavernInformationCompleted, second.Outcome)
	assert.Equal(t, uint16(100), second.GoldAfterPayment)
	assert.Zero(t, state.GetRNG().Calls())
}

func TestReviewTavernInformationByMerchantIDIsPureAndCustomerEligible(t *testing.T) {
	state, _ := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	state.PartyState.Inventory.Gold.Set(200)
	beforeGold := state.PartyState.Inventory.Gold.Get()
	beforeFood := state.PartyState.Inventory.Provisions.Food.Get()
	beforeCustomerSequence := state.tavernCustomerSessionSequence
	beforeRNG := state.GetRNG().Calls()

	quote, err := state.ReviewTavernInformationByMerchantID(
		references.MerchantTavernBritain,
		"tell me honeypot",
	)

	require.NoError(t, err)
	assert.Equal(t, "hone", quote.Topic.Key)
	assert.Equal(t, 50, quote.Price)
	assert.Equal(t, uint16(150), quote.GoldAfterPayment)
	assert.Equal(t, beforeGold, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, beforeFood, state.PartyState.Inventory.Provisions.Food.Get())
	assert.Equal(t, beforeCustomerSequence, state.tavernCustomerSessionSequence)
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	_, err = state.ReviewTavernInformationByMerchantID(references.MerchantTavernBritain, "unknown")
	require.ErrorIs(t, err, ErrTavernInformationTopicUnknown)
}

func TestTavernInformationInsufficientFundsIsTypedAndSideEffectFree(t *testing.T) {
	state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
	state.PartyState.Inventory.Gold.Set(49)
	state.drunk = 3
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	stop, err := state.QuoteTavernStop(session)
	require.NoError(t, err)
	_, err = state.StopTavernDrinking(session, TavernStopRequest{
		ReviewedMerchantID: stop.MerchantID,
		ReviewedDrunk:      stop.Drunk,
	})
	require.NoError(t, err)
	quote, err := state.QuoteTavernInformation(session, "hone")
	require.NoError(t, err)
	assert.False(t, quote.Affordable)

	result, err := state.PurchaseTavernInformation(session, tavernInformationRequest(state, quote))

	require.NoError(t, err)
	assert.Equal(t, TavernInformationInsufficientFunds, result.Outcome)
	assert.Equal(t, uint16(49), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, session.sequence, state.tavernCustomerSessionSequence)
	assert.Zero(t, state.GetRNG().Calls())
}

func TestTavernTransactionsRejectStaleAndTamperedRequests(t *testing.T) {
	t.Run("meal", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
		state.PartyState.Inventory.Gold.Set(100)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernMeal(session)
		require.NoError(t, err)
		request := tavernMealRequest(state, quote)
		request.ReviewedCharge++

		_, err = state.PurchaseTavernMeal(session, request)

		require.ErrorIs(t, err, ErrInvalidTavernRequest)
		assert.Equal(t, uint16(100), state.PartyState.Inventory.Gold.Get())
		assert.Zero(t, state.GetRNG().Calls())
		assert.Zero(t, state.tavernCustomerSessionSequence)
	})

	t.Run("drink", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernWestBritanny)
		state.PartyState.Inventory.Gold.Set(500)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernWine(session, 0)
		require.NoError(t, err)
		request := tavernDrinkRequest(state, quote, TavernDrinkApprovalPurchase)
		request.ReviewedWine.Price++

		_, err = state.PurchaseTavernDrink(session, request)

		require.ErrorIs(t, err, ErrInvalidTavernRequest)
		assert.Equal(t, uint16(500), state.PartyState.Inventory.Gold.Get())
		assert.Zero(t, state.drunk)
	})

	t.Run("provisions", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernNewMagincia)
		state.PartyState.Inventory.Gold.Set(100)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		quote, err := state.QuoteTavernProvisions(session)
		require.NoError(t, err)
		foodBefore := quote.FoodBefore
		request := tavernProvisionRequest(state, quote, 1)
		request.ReviewedPrice++

		_, err = state.PurchaseTavernProvisions(session, request)

		require.ErrorIs(t, err, ErrInvalidTavernRequest)
		assert.Equal(t, uint16(100), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, foodBefore, state.PartyState.Inventory.Provisions.Food.Get())
	})

	t.Run("information", func(t *testing.T) {
		state, npc := newOpenMerchantTestState(t, references.MerchantTavernBritain)
		state.PartyState.Inventory.Gold.Set(100)
		state.drunk = 3
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		stop, err := state.QuoteTavernStop(session)
		require.NoError(t, err)
		_, err = state.StopTavernDrinking(session, TavernStopRequest{
			ReviewedMerchantID: stop.MerchantID,
			ReviewedDrunk:      stop.Drunk,
		})
		require.NoError(t, err)
		quote, err := state.QuoteTavernInformation(session, "hone")
		require.NoError(t, err)
		request := tavernInformationRequest(state, quote)
		request.ReviewedPrice++

		_, err = state.PurchaseTavernInformation(session, request)

		require.ErrorIs(t, err, ErrInvalidTavernRequest)
		assert.Equal(t, uint16(100), state.PartyState.Inventory.Gold.Get())
		assert.Zero(t, state.GetRNG().Calls())
	})
}

func tavernMealRequest(state *GameState, quote TavernMealQuote) TavernMealRequest {
	return TavernMealRequest{
		ReviewedMerchantID: currentTavernMerchantID(state),
		ReviewedMeal:       quote.Meal,
		ReviewedDinerCount: quote.DinerCount,
		ReviewedCharge:     quote.Charge,
		ReviewedGoldBefore: quote.GoldBefore,
		ReviewedFoodBefore: quote.FoodBefore,
	}
}

func tavernDrinkRequest(
	state *GameState,
	quote TavernDrinkQuote,
	approval TavernDrinkApproval,
) TavernDrinkRequest {
	merchantID := currentTavernMerchantID(state)

	return TavernDrinkRequest{
		ReviewedMerchantID: merchantID,
		ReviewedDrink:      quote.Drink,
		ReviewedWineIndex:  quote.WineIndex,
		ReviewedWine:       quote.Wine,
		ReviewedDinerCount: quote.DinerCount,
		ReviewedCharge:     quote.Charge,
		ReviewedGoldBefore: quote.GoldBefore,
		ReviewedDrunk:      quote.Drunk,
		ReviewedSmashed:    quote.Smashed,
		ReviewedKarma:      quote.Karma,
		Approval:           approval,
	}
}

func tavernProvisionRequest(
	state *GameState,
	quote TavernProvisionQuote,
	packCount int,
) TavernProvisionRequest {
	return TavernProvisionRequest{
		ReviewedMerchantID: currentTavernMerchantID(state),
		ReviewedProvision:  quote.Provision,
		ReviewedPrice:      quote.PricePerPack,
		ReviewedGoldBefore: quote.GoldBefore,
		ReviewedFoodBefore: quote.FoodBefore,
		PackCount:          packCount,
	}
}

func tavernInformationRequest(state *GameState, quote TavernInformationQuote) TavernInformationRequest {
	return TavernInformationRequest{
		ReviewedMerchantID: currentTavernMerchantID(state),
		ReviewedTopic:      quote.Topic,
		ReviewedPrice:      quote.Price,
		ReviewedGoldBefore: quote.GoldBefore,
	}
}

func currentTavernMerchantID(state *GameState) references.MerchantID {
	for _, merchant := range state.GameReferences.MerchantReferences.Definitions() {
		if merchant.Family == references.MerchantFamilyTavern &&
			merchant.Location == state.MapState.PlayerLocation.Location {
			return merchant.ID
		}
	}

	panic("current location has no canonical Tavern")
}
