package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestHorseShopPricesUseCanonicalRowsAndCapturedBuyer(t *testing.T) {
	tests := []struct {
		merchantID references.MerchantID
		basePrice  int
		wantPrice  int
	}{
		{merchantID: references.MerchantHorseTrinsic, basePrice: 100, wantPrice: 170},
		{merchantID: references.MerchantHorseNorthBritanny, basePrice: 130, wantPrice: 221},
		{merchantID: references.MerchantHorsePaws, basePrice: 160, wantPrice: 272},
	}

	for _, test := range tests {
		t.Run(string(test.merchantID), func(t *testing.T) {
			state, npc := newOpenHorseShopTestState(t, test.merchantID)
			state.PartyState.Inventory.Gold.Set(500)
			setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass})
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			shop, err := state.GetHorseShopState(session)
			require.NoError(t, err)
			assert.Equal(t, test.merchantID, shop.Merchant.ID)
			assert.Equal(t, test.basePrice, shop.Quote.BasePrice)
			assert.Equal(t, test.wantPrice, shop.Quote.Price)
			assert.Equal(t, uint16(500-test.wantPrice), shop.Quote.GoldAfterPayment)
			assert.True(t, shop.Quote.Affordable)
		})
	}

	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass})
	selected := state.PartyState.GetCharacter(1)
	selected.PartyMap = party_state.PartyMapActive
	selected.SetStatus(party_state.Good)
	selected.Intelligence = 30
	require.True(t, state.SetActivePlayer(1))
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	require.True(t, state.SetActivePlayer(0))

	quote, err := state.QuoteHorsePurchase(session)
	require.NoError(t, err)
	assert.Equal(t, 1, session.PricingCharacterIndex())
	assert.Equal(t, CalculateMerchantBuyPrice(100, 30), quote.Price)
}

func TestHorsePlacementUsesExactTerrainAndSouthNorthEastWestOrder(t *testing.T) {
	tests := []struct {
		name      string
		tiles     [4]indexes.SpriteIndex
		wantIndex int
	}{
		{
			name:      "south grass wins when every direction is legal",
			tiles:     [4]indexes.SpriteIndex{indexes.Grass, indexes.MetalFloor, indexes.BrickFloor, indexes.Grass},
			wantIndex: 0,
		},
		{
			name:      "north metal floor",
			tiles:     [4]indexes.SpriteIndex{indexes.NoSprites, indexes.MetalFloor, indexes.BrickFloor, indexes.Grass},
			wantIndex: 1,
		},
		{
			name:      "east brick floor",
			tiles:     [4]indexes.SpriteIndex{indexes.NoSprites, indexes.NoSprites, indexes.BrickFloor, indexes.Grass},
			wantIndex: 2,
		},
		{
			name:      "west grass fallback",
			tiles:     [4]indexes.SpriteIndex{indexes.NoSprites, indexes.NoSprites, indexes.NoSprites, indexes.Grass},
			wantIndex: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
			setHorseCandidateTiles(t, state, test.tiles)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			quote, err := state.QuoteHorsePurchase(session)
			require.NoError(t, err)
			positions := horseCandidatePositions(state.MapState.PlayerLocation.Position)
			assert.Equal(t, HorsePlacementReview{
				Position:    positions[test.wantIndex],
				Floor:       state.MapState.PlayerLocation.Floor,
				Terrain:     test.tiles[test.wantIndex],
				BlockStatus: HorseShopBlockNone,
			}, quote.Placement)
		})
	}
}

func TestHorsePlacementUsesCurrentMutableMapOnlyTerrain(t *testing.T) {
	tests := []struct {
		name          string
		baseTiles     [4]indexes.SpriteIndex
		overrideIndex int
		overrideTile  indexes.SpriteIndex
		wantStatus    HorseShopBlockStatus
		wantIndex     int
		wantTerrain   indexes.SpriteIndex
	}{
		{
			name:          "illegal override blocks legal base",
			baseTiles:     [4]indexes.SpriteIndex{indexes.Grass},
			overrideIndex: 0,
			overrideTile:  indexes.StoneBrickWall,
			wantStatus:    HorseShopBlockNoLegalPlacement,
		},
		{
			name:          "legal override permits illegal base",
			baseTiles:     [4]indexes.SpriteIndex{},
			overrideIndex: 0,
			overrideTile:  indexes.Grass,
			wantStatus:    HorseShopBlockNone,
			wantIndex:     0,
			wantTerrain:   indexes.Grass,
		},
		{
			name:          "legal override replaces legal base identity",
			baseTiles:     [4]indexes.SpriteIndex{indexes.Grass},
			overrideIndex: 0,
			overrideTile:  indexes.MetalFloor,
			wantStatus:    HorseShopBlockNone,
			wantIndex:     0,
			wantTerrain:   indexes.MetalFloor,
		},
		{
			name:          "blocked south override advances to legal north",
			baseTiles:     [4]indexes.SpriteIndex{indexes.Grass, indexes.BrickFloor},
			overrideIndex: 0,
			overrideTile:  indexes.StoneBrickWall,
			wantStatus:    HorseShopBlockNone,
			wantIndex:     1,
			wantTerrain:   indexes.BrickFloor,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
			setHorseCandidateTiles(t, state, test.baseTiles)
			positions := horseCandidatePositions(state.MapState.PlayerLocation.Position)
			layeredMap := state.MapState.LayeredMaps.GetLayeredMap(
				references.SmallMapType,
				state.MapState.PlayerLocation.Floor,
			)
			layeredMap.SetTileByLayer(
				map_state.MapOverrideLayer,
				&positions[test.overrideIndex],
				test.overrideTile,
			)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			quote, err := state.QuoteHorsePurchase(session)
			require.NoError(t, err)
			assert.Equal(t, test.wantStatus, quote.Placement.BlockStatus)
			if test.wantStatus == HorseShopBlockNone {
				assert.Equal(t, positions[test.wantIndex], quote.Placement.Position)
				assert.Equal(t, test.wantTerrain, quote.Placement.Terrain)
			}
		})
	}
}

func TestHorsePlacementRejectsEveryNonWhitelistedTerrain(t *testing.T) {
	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{
		indexes.NoSprites,
		indexes.StoneBrickWall,
		indexes.DryStone,
		indexes.Water1,
	})
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteHorsePurchase(session)
	require.NoError(t, err)
	assert.Equal(t, HorseShopBlockNoLegalPlacement, quote.Placement.BlockStatus)
}

func TestHorsePlacementOccupancyIsCurrentFloorOnly(t *testing.T) {
	tests := []struct {
		name          string
		occupantFloor func(references.FloorNumber) references.FloorNumber
		wantIndex     int
	}{
		{
			name: "other floor is allowed",
			occupantFloor: func(floor references.FloorNumber) references.FloorNumber {
				return floor + 1
			},
			wantIndex: 0,
		},
		{
			name: "same floor is blocked",
			occupantFloor: func(floor references.FloorNumber) references.FloorNumber {
				return floor
			},
			wantIndex: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
			setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass, indexes.Grass})
			positions := horseCandidatePositions(state.MapState.PlayerLocation.Position)
			occupant := map_units.NewNPCFriendlyVehiceNewRef(
				references.CarpetVehicle,
				positions[0],
				test.occupantFloor(state.MapState.PlayerLocation.Floor),
			)
			require.True(t, state.CurrentNPCAIController.GetNpcs().AddVehicle(*occupant))
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			quote, err := state.QuoteHorsePurchase(session)
			require.NoError(t, err)
			assert.Equal(t, positions[test.wantIndex], quote.Placement.Position)
		})
	}
}

func TestIsPassableFindsCurrentFloorHorseAfterOtherFloorNonVehicle(t *testing.T) {
	state, _ := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass})
	position := horseCandidatePositions(state.MapState.PlayerLocation.Position)[0]
	floor := state.MapState.PlayerLocation.Floor
	units := state.CurrentNPCAIController.GetNpcs()
	otherFloorNPC := map_units.NewNPCFriendly(
		references.NPCReference{NpcType: references.NoStatedNpc},
		map_units.MaximumNpcsPerMap-1,
	)
	otherFloorNPC.SetPos(position)
	otherFloorNPC.SetFloor(floor + 1)
	otherFloorNPC.SetVisible(true)
	*units = append(*units, otherFloorNPC)
	currentFloorHorse := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, floor)
	require.True(t, units.AddVehicle(*currentFloorHorse))

	found := units.GetVehicleAtPositionAndFloorOrNil(position, floor)
	require.NotNil(t, found)
	assert.Equal(t, references.HorseVehicle, found.GetVehicleDetails().VehicleType)
	assert.True(t, state.ObjectPresentAt(&position))
	assert.True(t, state.IsPassable(&position), "movement must allow stepping onto the current-floor horse")
}

func TestHorsePlacementChecksMapUnitCapacityBeforeDelivery(t *testing.T) {
	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass, indexes.Grass, indexes.Grass, indexes.Grass})
	fillHorseShopMapUnitCapacity(t, state)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)

	quote, err := state.QuoteHorsePurchase(session)
	require.NoError(t, err)
	assert.Equal(t, HorseShopBlockMapUnitCapacity, quote.Placement.BlockStatus)
}

func TestReviewHorseShopRemotePreviewQuotesPriceAndDefersAvailabilityWithoutMutation(t *testing.T) {
	state, _ := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	state.SetRandomSeed(214)
	state.PartyState.Inventory.Gold.Set(777)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.SetupEnvironmentalSystemCallbacks()
	beforeLocation := state.MapState.PlayerLocation
	beforeDate := state.DateTime
	beforeElapsed := state.ElapsedMs
	beforeCalls := state.GetRNG().Calls()
	beforeSequence := state.merchantSessionSequence
	beforeGold := state.PartyState.Inventory.Gold.Get()
	beforeUnits := horseMapUnitValueSnapshots(state.CurrentNPCAIController.GetNpcs())

	review, err := state.ReviewHorseShopByMerchantID(references.MerchantHorsePaws)

	require.NoError(t, err)
	assert.Equal(t, references.MerchantHorsePaws, review.State.Merchant.ID)
	assert.Equal(t, 160, review.State.Quote.BasePrice)
	assert.Equal(t, CalculateMerchantBuyPrice(160, 10), review.State.Quote.Price)
	assert.Equal(t, HorseShopBlockAvailabilityDeferred, review.State.Quote.Placement.BlockStatus)
	assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeElapsed, state.ElapsedMs)
	assert.Equal(t, beforeCalls, state.GetRNG().Calls())
	assert.Equal(t, beforeSequence, state.merchantSessionSequence)
	assert.Equal(t, beforeGold, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, beforeUnits, horseMapUnitValueSnapshots(state.CurrentNPCAIController.GetNpcs()))
	assert.Zero(t, callbacks.MessageCallCount)
	assert.Zero(t, callbacks.AudioCallCount)
	assert.Zero(t, callbacks.VisualCallCount)
	assert.Zero(t, callbacks.ScreenCallCount)
	assert.Zero(t, callbacks.FlowCallCount)
	assert.Zero(t, callbacks.TalkCallCount)
}

func TestReviewHorseShopChecksPlacementForMatchingLoadedDealer(t *testing.T) {
	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.NoSprites, indexes.MetalFloor})

	review, err := state.ReviewHorseShopByMerchantID(references.MerchantHorseTrinsic)
	require.NoError(t, err)
	assert.Equal(t, HorseShopBlockNone, review.State.Quote.Placement.BlockStatus)

	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	production, err := state.GetHorseShopState(session)
	require.NoError(t, err)
	assert.Equal(t, production, review.State)
}

func TestPurchaseHorseRejectsStaleAndInvalidReviewsWithoutEffects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*testing.T, *GameState, *map_units.NPCFriendly, MerchantSession, HorseShopState)
		request func(HorseShopState) HorsePurchaseRequest
		wantErr error
	}{
		{
			name: "placement changed",
			mutate: func(t *testing.T, state *GameState, _ *map_units.NPCFriendly, _ MerchantSession, shop HorseShopState) {
				t.Helper()
				occupant := map_units.NewNPCFriendlyVehiceNewRef(
					references.CarpetVehicle,
					shop.Quote.Placement.Position,
					shop.Quote.Placement.Floor,
				)
				require.True(t, state.CurrentNPCAIController.GetNpcs().AddVehicle(*occupant))
			},
			wantErr: ErrHorseStableClosed,
		},
		{
			name: "session ended",
			mutate: func(_ *testing.T, state *GameState, _ *map_units.NPCFriendly, session MerchantSession, _ HorseShopState) {
				state.EndMerchantSession(session)
			},
			wantErr: ErrInvalidMerchantSession,
		},
		{
			name: "captured buyer became invalid",
			mutate: func(_ *testing.T, state *GameState, _ *map_units.NPCFriendly, session MerchantSession, _ HorseShopState) {
				state.PartyState.GetCharacter(session.PricingCharacterIndex()).SetStatus(party_state.Dead)
			},
			wantErr: ErrInvalidMerchantSession,
		},
		{
			name: "price changed",
			mutate: func(_ *testing.T, state *GameState, _ *map_units.NPCFriendly, session MerchantSession, _ HorseShopState) {
				state.PartyState.GetCharacter(session.PricingCharacterIndex()).Intelligence++
			},
			wantErr: ErrInvalidHorsePurchase,
		},
		{
			name: "merchant closed",
			mutate: func(_ *testing.T, state *GameState, npc *map_units.NPCFriendly, _ MerchantSession, _ HorseShopState) {
				npc.PlaceAtScheduleSlot(0)
				state.MapState.PlayerLocation.Floor = npc.Floor()
			},
			wantErr: ErrMerchantClosed,
		},
		{
			name: "capacity changed",
			mutate: func(t *testing.T, state *GameState, _ *map_units.NPCFriendly, _ MerchantSession, _ HorseShopState) {
				t.Helper()
				fillHorseShopMapUnitCapacity(t, state)
			},
			wantErr: ErrHorseStableClosed,
		},
		{
			name:   "merchant identity tampered",
			mutate: func(*testing.T, *GameState, *map_units.NPCFriendly, MerchantSession, HorseShopState) {},
			request: func(shop HorseShopState) HorsePurchaseRequest {
				request := horsePurchaseRequest(shop)
				request.ReviewedMerchantID = references.MerchantHorsePaws

				return request
			},
			wantErr: ErrInvalidHorsePurchase,
		},
		{
			name:   "price tampered",
			mutate: func(*testing.T, *GameState, *map_units.NPCFriendly, MerchantSession, HorseShopState) {},
			request: func(shop HorseShopState) HorsePurchaseRequest {
				request := horsePurchaseRequest(shop)
				request.ReviewedPrice--

				return request
			},
			wantErr: ErrInvalidHorsePurchase,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
			state.SetRandomSeed(214)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(1000)
			setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass, indexes.MetalFloor})
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			shop, err := state.GetHorseShopState(session)
			require.NoError(t, err)
			request := horsePurchaseRequest(shop)
			if test.request != nil {
				request = test.request(shop)
			}
			test.mutate(t, state, npc, session, shop)
			goldBefore := state.PartyState.Inventory.Gold.Get()
			unitsBefore := append(map_units.MapUnits(nil), (*state.CurrentNPCAIController.GetNpcs())...)
			rngBefore := state.GetRNG().Calls()

			_, err = state.PurchaseHorse(session, request)

			require.ErrorIs(t, err, test.wantErr)
			assert.Equal(t, goldBefore, state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, unitsBefore, *state.CurrentNPCAIController.GetNpcs())
			assert.Equal(t, rngBefore, state.GetRNG().Calls())
		})
	}
}

func TestPurchaseHorseRejectsAffordableGoldChangeAfterApprovalWithoutEffects(t *testing.T) {
	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	state.SetRandomSeed(214)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(1000)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass})
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetHorseShopState(session)
	require.NoError(t, err)
	request := horsePurchaseRequest(shop)
	state.PartyState.Inventory.Gold.Set(999)
	require.GreaterOrEqual(t, int(state.PartyState.Inventory.Gold.Get()), shop.Quote.Price)
	goldBefore := state.PartyState.Inventory.Gold.Get()
	rngBefore := state.GetRNG().Calls()
	unitsBefore := horseMapUnitValueSnapshots(state.CurrentNPCAIController.GetNpcs())

	_, err = state.PurchaseHorse(session, request)

	require.ErrorIs(t, err, ErrInvalidHorsePurchase)
	assert.Equal(t, goldBefore, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, rngBefore, state.GetRNG().Calls())
	assert.Equal(t, unitsBefore, horseMapUnitValueSnapshots(state.CurrentNPCAIController.GetNpcs()))
	assert.Zero(t, callbacks.MessageCallCount)
	assert.Zero(t, callbacks.AudioCallCount)
	assert.Zero(t, callbacks.VisualCallCount)
	assert.Zero(t, callbacks.ScreenCallCount)
	assert.Zero(t, callbacks.FlowCallCount)
	assert.Zero(t, callbacks.TalkCallCount)
}

func TestPurchaseHorseCommitsPaymentTheftAndInsertion(t *testing.T) {
	const seed = 240214

	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(1000)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{
		indexes.NoSprites,
		indexes.NoSprites,
		indexes.BrickFloor,
	})
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetHorseShopState(session)
	require.NoError(t, err)
	request := horsePurchaseRequest(shop)
	units := state.CurrentNPCAIController.GetNpcs()
	unitsBefore := len(*units)
	expectedRNG := beradrng.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)
	expectedGoldAfterPayment := uint16(1000 - shop.Quote.Price)
	expectedGoldAfterTheft := expectedGoldAfterPayment - expectedTheft

	result, err := state.PurchaseHorse(session, request)

	require.NoError(t, err)
	assert.Equal(t, HorsePurchaseResult{
		MerchantID:            references.MerchantHorseTrinsic,
		PricingCharacterIndex: 0,
		BasePrice:             100,
		Price:                 170,
		GoldBefore:            1000,
		GoldAfterPayment:      expectedGoldAfterPayment,
		Theft:                 expectedTheft,
		GoldAfterTheft:        expectedGoldAfterTheft,
		VehicleType:           references.HorseVehicle,
		Position:              shop.Quote.Placement.Position,
		Floor:                 shop.Quote.Placement.Floor,
		Direction:             references.Right,
		OwnedByNPC:            false,
	}, result)
	assert.Equal(t, expectedGoldAfterTheft, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Len(t, *units, unitsBefore+1)
	assert.Equal(t, 1, countHorseShopVehicles(units))

	horse := units.GetVehicleAtPositionAndFloorOrNil(result.Position, result.Floor)
	require.NotNil(t, horse)
	assert.Equal(t, result.Floor, horse.Floor())
	assert.Equal(t, references.Right, horse.GetVehicleDetails().CurrentDirection())
	assert.False(t, horse.GetVehicleDetails().IsOwnedByNPC)
	state.MapState.PlayerLocation.Position = result.Position
	assert.Equal(t, BoardVehicleResultSuccess, state.BoardVehicle(*horse))
}

func TestPurchaseHorseWithoutFaulineiDoesNotConsumeGameplayRNG(t *testing.T) {
	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	state.SetRandomSeed(214)
	state.ShadowlordState.AutomaticPresence = references.NoShadowlord
	state.PartyState.Inventory.Gold.Set(1000)
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass})
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetHorseShopState(session)
	require.NoError(t, err)

	result, err := state.PurchaseHorse(session, horsePurchaseRequest(shop))

	require.NoError(t, err)
	assert.Zero(t, result.Theft)
	assert.Zero(t, state.GetRNG().Calls())
	assert.Equal(t, uint16(830), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, 1, countHorseShopVehicles(state.CurrentNPCAIController.GetNpcs()))
}

func TestPurchasedHorseNativeSnapshotRoundTrip(t *testing.T) {
	t.Run("unboarded current town horse", func(t *testing.T) {
		state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
		result := purchaseHorseForSnapshotTest(t, state, npc)

		snapshot, err := state.CaptureSaveSnapshot()
		require.NoError(t, err)
		hydrated, err := NewGameStateFromSaveSnapshot(
			snapshot,
			state.GameConfig,
			state.GameReferences,
			xTilesVisibleOnGameScreen,
			yTilesVisibleOnGameScreen,
		)
		require.NoError(t, err)

		horse := hydrated.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
			result.Position,
			result.Floor,
		)
		require.NotNil(t, horse)
		assertPurchasedHorseState(t, horse, result.Position, result.Floor)
		assert.Equal(t, references.NoPartyVehicle, hydrated.PartyVehicle.GetVehicleDetails().VehicleType)
		assert.Equal(t, 1, countHorseShopVehicles(hydrated.CurrentNPCAIController.GetNpcs()))
	})

	t.Run("boarded party horse", func(t *testing.T) {
		state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
		result := purchaseHorseForSnapshotTest(t, state, npc)
		units := state.CurrentNPCAIController.GetNpcs()
		horse := units.GetVehicleAtPositionAndFloorOrNil(result.Position, result.Floor)
		require.NotNil(t, horse)
		state.MapState.PlayerLocation.Position = result.Position
		require.Equal(t, BoardVehicleResultSuccess, state.BoardVehicle(*horse))
		assert.Zero(t, countHorseShopVehicles(units))

		snapshot, err := state.CaptureSaveSnapshot()
		require.NoError(t, err)
		require.NotNil(t, snapshot.PartyVehicle)
		hydrated, err := NewGameStateFromSaveSnapshot(
			snapshot,
			state.GameConfig,
			state.GameReferences,
			xTilesVisibleOnGameScreen,
			yTilesVisibleOnGameScreen,
		)
		require.NoError(t, err)

		partyHorse := hydrated.CurrentPartyVehicleSnapshot()
		assert.Equal(t, result.Position, partyHorse.Position)
		assert.Equal(t, result.Floor, partyHorse.Floor)
		assert.Equal(t, references.HorseVehicle, partyHorse.VehicleType)
		assert.Equal(t, references.Right, partyHorse.CurrentDirection)
		assert.False(t, partyHorse.IsOwnedByNPC)
		assert.Zero(t, countHorseShopVehicles(hydrated.CurrentNPCAIController.GetNpcs()))
	})
}

func TestFullTownRebuildDiscardsUnboardedPurchasedHorse(t *testing.T) {
	state, npc := newOpenHorseShopTestState(t, references.MerchantHorseTrinsic)
	result := purchaseHorseForSnapshotTest(t, state, npc)
	purchased := state.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
		result.Position,
		result.Floor,
	)
	require.NotNil(t, purchased)

	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)

	for _, mapUnit := range *state.CurrentNPCAIController.GetNpcs() {
		assert.NotSame(t, purchased, mapUnit)
	}
	assert.Nil(t, state.CurrentNPCAIController.GetNpcs().GetVehicleAtPositionAndFloorOrNil(
		result.Position,
		result.Floor,
	))
}

func newOpenHorseShopTestState(
	t *testing.T,
	merchantID references.MerchantID,
) (*GameState, *map_units.NPCFriendly) {
	t.Helper()

	state, npc := newOpenMerchantTestState(t, merchantID)
	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	installMerchantTestNPC(t, state, npc)
	state.MapState.PlayerLocation.Position = references.Position{X: 10, Y: 10}
	npc.SetPos(references.Position{X: 2, Y: 2})

	return state, npc
}

func horseCandidatePositions(position references.Position) [4]references.Position {
	return [4]references.Position{
		{X: position.X, Y: position.Y + 1},
		{X: position.X, Y: position.Y - 1},
		{X: position.X + 1, Y: position.Y},
		{X: position.X - 1, Y: position.Y},
	}
}

func setHorseCandidateTiles(
	t *testing.T,
	state *GameState,
	tiles [4]indexes.SpriteIndex,
) {
	t.Helper()

	layeredMap := state.MapState.LayeredMaps.GetLayeredMap(
		references.SmallMapType,
		state.MapState.PlayerLocation.Floor,
	)
	require.NotNil(t, layeredMap)
	positions := horseCandidatePositions(state.MapState.PlayerLocation.Position)
	for index, position := range positions {
		layeredMap.SetTileByLayer(map_state.MapLayer, &position, tiles[index])
		layeredMap.UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	}
}

func fillHorseShopMapUnitCapacity(t *testing.T, state *GameState) {
	t.Helper()

	units := state.CurrentNPCAIController.GetNpcs()
	require.NotNil(t, units)
	for len(*units) < map_units.MaximumNpcsPerMap {
		vehicle := map_units.NewNPCFriendlyVehiceNewRef(
			references.CarpetVehicle,
			references.Position{X: 25, Y: 25},
			state.MapState.PlayerLocation.Floor,
		)
		require.True(t, units.AddVehicle(*vehicle))
	}
}

func horsePurchaseRequest(shop HorseShopState) HorsePurchaseRequest {
	return HorsePurchaseRequest{
		ReviewedMerchantID:            shop.Merchant.ID,
		ReviewedPricingCharacterIndex: shop.PricingCharacterIndex,
		ReviewedPrice:                 shop.Quote.Price,
		ReviewedGoldBefore:            shop.Quote.GoldBefore,
		ReviewedPlacement:             shop.Quote.Placement,
	}
}

type horseMapUnitValueSnapshot struct {
	MapUnitType        map_units.MapUnitType
	Position           references.Position
	Floor              references.FloorNumber
	Visible            bool
	Empty              bool
	KeySprite          indexes.SpriteIndex
	NPCNum             int
	CurrentPath        []references.Position
	PendingInteraction map_units.PendingNPCInteraction
	SkipNextTurn       bool
	OverriddenAI       references.AiType
	ArrivedSchedule    references.NPCScheduleSlot
	ConsumedTalk3      references.NPCScheduleSlot
	HasConsumedTalk3   bool
	Friendly           bool
	FriendlyReference  references.NPCReference
	Vehicle            bool
	VehicleDetails     map_units.VehicleDetailsSnapshot
}

func horseMapUnitValueSnapshots(units *map_units.MapUnits) []horseMapUnitValueSnapshot {
	result := make([]horseMapUnitValueSnapshot, 0, len(*units))
	for _, mapUnit := range *units {
		details := mapUnit.MapUnitDetails()
		consumedTalk3, hasConsumedTalk3 := details.ConsumedTalk3ScheduleSlot()
		snapshot := horseMapUnitValueSnapshot{
			MapUnitType:        mapUnit.GetMapUnitType(),
			Position:           mapUnit.Pos(),
			Floor:              mapUnit.Floor(),
			Visible:            mapUnit.IsVisible(),
			Empty:              mapUnit.IsEmptyMapUnit(),
			KeySprite:          mapUnit.GetKeySpriteIndex(),
			NPCNum:             details.NPCNum,
			CurrentPath:        append([]references.Position(nil), details.CurrentPath...),
			PendingInteraction: details.PendingInteraction,
			SkipNextTurn:       details.SkipNextTurn,
			OverriddenAI:       details.GetOverriddenAiType(),
			ArrivedSchedule:    details.ArrivedScheduleSlot(),
			ConsumedTalk3:      consumedTalk3,
			HasConsumedTalk3:   hasConsumedTalk3,
		}
		if friendly, ok := mapUnit.(*map_units.NPCFriendly); ok {
			snapshot.Friendly = true
			snapshot.FriendlyReference = friendly.NPCReference
			if friendly.NPCReference.GetNPCType() == references.Vehicle {
				snapshot.Vehicle = true
				snapshot.VehicleDetails = friendly.GetVehicleDetails().Snapshot()
			}
		}
		result = append(result, snapshot)
	}

	return result
}

func purchaseHorseForSnapshotTest(
	t *testing.T,
	state *GameState,
	npc *map_units.NPCFriendly,
) HorsePurchaseResult {
	t.Helper()

	state.SetRandomSeed(214)
	state.ShadowlordState.AutomaticPresence = references.NoShadowlord
	state.PartyState.Inventory.Gold.Set(1000)
	avatar := state.PartyState.GetCharacter(0)
	avatar.Level = 1
	avatar.SetMaxHp(30)
	avatar.SetHp(30)
	avatar.SetMp(0)
	for _, controller := range state.LargeMapNPCAIController {
		controller.GetNpcs().ClearAllMapUnits()
	}
	setHorseCandidateTiles(t, state, [4]indexes.SpriteIndex{indexes.Grass})
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetHorseShopState(session)
	require.NoError(t, err)
	result, err := state.PurchaseHorse(session, horsePurchaseRequest(shop))
	require.NoError(t, err)
	state.EndMerchantSession(session)

	return result
}

func assertPurchasedHorseState(
	t *testing.T,
	horse *map_units.NPCFriendly,
	position references.Position,
	floor references.FloorNumber,
) {
	t.Helper()

	require.NotNil(t, horse)
	assert.Equal(t, position, horse.Pos())
	assert.Equal(t, floor, horse.Floor())
	details := horse.GetVehicleDetails()
	assert.Equal(t, references.HorseVehicle, details.VehicleType)
	assert.Equal(t, references.Right, details.CurrentDirection())
	assert.False(t, details.IsOwnedByNPC)
}

func countHorseShopVehicles(units *map_units.MapUnits) int {
	count := 0
	for _, mapUnit := range *units {
		friendly, ok := mapUnit.(*map_units.NPCFriendly)
		if ok && friendly.GetVehicleDetails().VehicleType == references.HorseVehicle {
			count++
		}
	}

	return count
}
