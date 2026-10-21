package game_state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestShipShopCanonicalDealersPricesAndPurePreview(t *testing.T) {
	tests := []struct {
		merchantID  references.MerchantID
		location    references.Location
		position    references.Position
		frigateBase int
		skiffBase   int
	}{
		{references.MerchantShipJhelom, references.Jhelom, references.Position{X: 0x27, Y: 0xdd}, 600, 200},
		{references.MerchantShipMinoc, references.Minoc, references.Position{X: 0x97, Y: 0x15}, 753, 175},
		{references.MerchantShipEastBritanny, references.EastBritanny, references.Position{X: 0x4f, Y: 0x6d}, 650, 125},
		{references.MerchantShipBuccaneersDen, references.BuccaneersDen, references.Position{X: 0x8a, Y: 0x9f}, 700, 100},
	}

	for _, test := range tests {
		t.Run(string(test.merchantID), func(t *testing.T) {
			state, _ := newOpenMerchantTestState(t, references.MerchantShipJhelom)
			clearShipShopOverworldUnits(t, state)
			state.SetRandomSeed(212)
			state.PartyState.Inventory.Gold.Set(5000)
			callbacks := NewMockSystemCallbacks(t)
			state.SystemCallbacks = callbacks.ToSystemCallbacks()
			beforeLocation := state.MapState.PlayerLocation
			beforeDate := state.DateTime
			beforeElapsed := state.ElapsedMs
			beforeRNG := state.GetRNG().Calls()
			beforeSequence := state.merchantSessionSequence

			review, err := state.ReviewShipShopByMerchantID(test.merchantID)

			require.NoError(t, err)
			assert.Equal(t, test.merchantID, review.State.Merchant.ID)
			assert.Equal(t, test.location, review.State.Berth.Location)
			assert.Equal(t, test.position, review.State.Berth.Position)
			assert.Equal(t, ShipBerthEmpty, review.State.Berth.State)
			assert.Equal(t, test.frigateBase, review.State.FrigateQuote.BasePrice)
			assert.Equal(t, CalculateMerchantBuyPrice(test.frigateBase, 10), review.State.FrigateQuote.Price)
			assert.Equal(t, test.skiffBase, review.State.SkiffQuote.BasePrice)
			assert.Equal(t, CalculateMerchantBuyPrice(test.skiffBase, 10), review.State.SkiffQuote.Price)
			assert.Len(t, review.State.FrigateQuote.ApprovalViews, 1)
			assert.Len(t, review.State.SkiffQuote.ApprovalViews, 1, "standalone delivery is always one skiff")
			assert.NotEmpty(t, review.State.FrigateQuote.RequestFingerprint)
			assert.NotEqual(t, review.State.FrigateQuote.RequestFingerprint, review.State.SkiffQuote.RequestFingerprint)
			assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
			assert.Equal(t, beforeDate, state.DateTime)
			assert.Equal(t, beforeElapsed, state.ElapsedMs)
			assert.Equal(t, beforeRNG, state.GetRNG().Calls())
			assert.Equal(t, beforeSequence, state.merchantSessionSequence)
			assert.Zero(t, callbacks.MessageCallCount)
			assert.Zero(t, callbacks.AudioCallCount)
			assert.Zero(t, callbacks.VisualCallCount)
			assert.Zero(t, callbacks.ScreenCallCount)
			assert.Zero(t, callbacks.FlowCallCount)
			assert.Zero(t, callbacks.TalkCallCount)
		})
	}
}

func TestPurchaseShipRefusesExistingStandaloneSkiffAtBerth(t *testing.T) {
	for _, pending := range []bool{true, false} {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		state.PartyState.Inventory.Gold.Set(2000)
		position := shipShopDockPosition(t, state, references.Jhelom)
		addShipShopVehicle(t, state, references.SkiffVehicle, position, pending, 0)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		shop, err := state.GetShipShopState(session)
		require.NoError(t, err)
		before := shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state))

		_, err = state.PurchaseShip(session, shipPurchaseRequest(shop, shop.SkiffQuote))

		require.ErrorIs(t, err, ErrShipPurchaseRefused)
		assert.Equal(t, uint16(2000), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, before, shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state)))
	}
}

func TestShipShopBerthStatesAndSkiffDispositions(t *testing.T) {
	tests := []struct {
		name             string
		setup            func(*testing.T, *GameState, references.Position) *map_units.NPCFriendly
		wantState        ShipBerthState
		wantFrigateBlock ShipPurchaseBlockReason
		wantSkiffAction  ShipPurchaseAction
		wantSkiffBlock   ShipPurchaseBlockReason
		wantSkiffs       int
	}{
		{
			name:             "empty berth creates either vessel",
			wantState:        ShipBerthEmpty,
			wantFrigateBlock: ShipPurchaseBlockNone,
			wantSkiffAction:  ShipPurchaseCreateStandaloneSkiff,
			wantSkiffBlock:   ShipPurchaseBlockNone,
		},
		{
			name: "pending frigate loads skiff",
			setup: func(t *testing.T, state *GameState, position references.Position) *map_units.NPCFriendly {
				return addShipShopVehicle(t, state, references.FrigateVehicle, position, true, 2)
			},
			wantState:        ShipBerthPendingFrigate,
			wantFrigateBlock: ShipPurchaseBlockFrigateAlreadyPresent,
			wantSkiffAction:  ShipPurchaseLoadSkiff,
			wantSkiffBlock:   ShipPurchaseBlockNone,
			wantSkiffs:       2,
		},
		{
			name: "moored frigate loads skiff",
			setup: func(t *testing.T, state *GameState, position references.Position) *map_units.NPCFriendly {
				return addShipShopVehicle(t, state, references.FrigateVehicle, position, false, 7)
			},
			wantState:        ShipBerthMooredFrigate,
			wantFrigateBlock: ShipPurchaseBlockFrigateAlreadyPresent,
			wantSkiffAction:  ShipPurchaseLoadSkiff,
			wantSkiffBlock:   ShipPurchaseBlockNone,
			wantSkiffs:       7,
		},
		{
			name: "pending standalone skiff refuses",
			setup: func(t *testing.T, state *GameState, position references.Position) *map_units.NPCFriendly {
				return addShipShopVehicle(t, state, references.SkiffVehicle, position, true, 0)
			},
			wantState:        ShipBerthPendingSkiff,
			wantFrigateBlock: ShipPurchaseBlockBerthOccupied,
			wantSkiffAction:  ShipPurchaseCreateStandaloneSkiff,
			wantSkiffBlock:   ShipPurchaseBlockStandaloneSkiff,
		},
		{
			name: "moored standalone skiff refuses",
			setup: func(t *testing.T, state *GameState, position references.Position) *map_units.NPCFriendly {
				return addShipShopVehicle(t, state, references.SkiffVehicle, position, false, 0)
			},
			wantState:        ShipBerthMooredSkiff,
			wantFrigateBlock: ShipPurchaseBlockBerthOccupied,
			wantSkiffAction:  ShipPurchaseCreateStandaloneSkiff,
			wantSkiffBlock:   ShipPurchaseBlockStandaloneSkiff,
		},
		{
			name: "non-ship obstruction refuses",
			setup: func(t *testing.T, state *GameState, position references.Position) *map_units.NPCFriendly {
				return addShipShopVehicle(t, state, references.CarpetVehicle, position, false, 0)
			},
			wantState:        ShipBerthObstructed,
			wantFrigateBlock: ShipPurchaseBlockBerthOccupied,
			wantSkiffAction:  ShipPurchaseCreateStandaloneSkiff,
			wantSkiffBlock:   ShipPurchaseBlockBerthOccupied,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
			position := shipShopDockPosition(t, state, references.Jhelom)
			if test.setup != nil {
				test.setup(t, state, position)
			}
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			shop, err := state.GetShipShopState(session)

			require.NoError(t, err)
			assert.Equal(t, test.wantState, shop.Berth.State)
			assert.Equal(t, test.wantSkiffs, shop.Berth.SkiffCount)
			assert.Equal(t, test.wantFrigateBlock, shop.FrigateQuote.BlockReason)
			assert.Equal(t, test.wantSkiffAction, shop.SkiffQuote.Action)
			assert.Equal(t, test.wantSkiffBlock, shop.SkiffQuote.BlockReason)
		})
	}
}

func TestShipShopReduxSkiffLimitAndGlobalPendingSingleton(t *testing.T) {
	t.Run("frigate at Redux capacity refuses both pending and moored", func(t *testing.T) {
		for _, pending := range []bool{false, true} {
			state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
			position := shipShopDockPosition(t, state, references.Jhelom)
			addShipShopVehicle(t, state, references.FrigateVehicle, position, pending, ShipShopMaximumSkiffs)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			quote, err := state.QuoteShipPurchase(session, references.SkiffVehicle)

			require.NoError(t, err)
			assert.Equal(t, ShipPurchaseLoadSkiff, quote.Action)
			assert.Equal(t, ShipPurchaseBlockSkiffLimit, quote.BlockReason)
		}
	})

	t.Run("pending order at another canonical dock blocks new vessels", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		otherDock := shipShopDockPosition(t, state, references.Minoc)
		addShipShopVehicle(t, state, references.SkiffVehicle, otherDock, true, 0)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		shop, err := state.GetShipShopState(session)

		require.NoError(t, err)
		assert.Equal(t, ShipBerthEmpty, shop.Berth.State)
		assert.Equal(t, ShipPurchaseBlockPendingOrderElsewhere, shop.FrigateQuote.BlockReason)
		assert.Equal(t, ShipPurchaseBlockPendingOrderElsewhere, shop.SkiffQuote.BlockReason)
	})

	t.Run("full map blocks insertion but not same-pointer loading", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		fillShipShopOverworldCapacity(t, state)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		shop, err := state.GetShipShopState(session)
		require.NoError(t, err)
		assert.Equal(t, ShipPurchaseBlockMapCapacity, shop.FrigateQuote.BlockReason)
		assert.Equal(t, ShipPurchaseBlockMapCapacity, shop.SkiffQuote.BlockReason)

		clearShipShopOverworldUnits(t, state)
		position := shipShopDockPosition(t, state, references.Jhelom)
		frigate := addShipShopVehicle(t, state, references.FrigateVehicle, position, false, 4)
		fillShipShopOverworldCapacity(t, state)
		shop, err = state.GetShipShopState(session)
		require.NoError(t, err)
		assert.False(t, shop.Berth.HasMapCapacity)
		assert.Equal(t, ShipPurchaseBlockNone, shop.SkiffQuote.BlockReason)

		state.PartyState.Inventory.Gold.Set(1000)
		shop, err = state.GetShipShopState(session)
		require.NoError(t, err)
		beforeCount := len(*shipShopOverworldUnits(t, state))
		result, err := state.PurchaseShip(session, shipPurchaseRequest(shop, shop.SkiffQuote))

		require.NoError(t, err)
		assert.Equal(t, ShipPurchaseLoadSkiff, result.Action)
		assert.Equal(t, 5, result.SkiffCountAfter)
		assert.Equal(t, 5, frigate.GetVehicleDetails().GetSkiffQuantity())
		assert.Len(t, *shipShopOverworldUnits(t, state), beforeCount)
	})
}

func TestShipShopUnaffordableFrigateSkiffRemainsReviewable(t *testing.T) {
	state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
	state.PartyState.Inventory.Gold.Set(1)
	position := shipShopDockPosition(t, state, references.Jhelom)
	frigate := addShipShopVehicle(t, state, references.FrigateVehicle, position, false, 2)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetShipShopState(session)
	require.NoError(t, err)
	require.Len(t, shop.SkiffQuote.ApprovalViews, 1)
	approval := shop.SkiffQuote.ApprovalViews[0]
	assert.Equal(t, 1, approval.Quantity)
	assert.False(t, approval.Affordable)
	before := shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state))

	_, err = state.PurchaseShip(session, shipPurchaseRequest(shop, shop.SkiffQuote))

	require.ErrorIs(t, err, ErrShipInsufficientFunds)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, 2, frigate.GetVehicleDetails().GetSkiffQuantity())
	assert.Equal(t, before, shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state)))
}

func TestPurchaseShipCommitsPaymentTheftAndDelivery(t *testing.T) {
	const seed = 240212

	state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(2000)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetShipShopState(session)
	require.NoError(t, err)
	quote := shop.FrigateQuote
	expectedRNG := beradrng.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)
	expectedAfterPayment := uint16(2000 - quote.Price)
	expectedAfterTheft := expectedAfterPayment - expectedTheft
	units := shipShopOverworldUnits(t, state)
	beforeUnits := len(*units)

	result, err := state.PurchaseShip(session, shipPurchaseRequest(shop, quote))

	require.NoError(t, err)
	assert.Equal(t, ShipPurchaseResult{
		MerchantID:            references.MerchantShipJhelom,
		PricingCharacterIndex: 0,
		VehicleType:           references.FrigateVehicle,
		Action:                ShipPurchaseCreateFrigate,
		Quantity:              1,
		BasePrice:             600,
		UnitPrice:             CalculateMerchantBuyPrice(600, 10),
		Price:                 CalculateMerchantBuyPrice(600, 10),
		GoldBefore:            2000,
		GoldAfterPayment:      expectedAfterPayment,
		Theft:                 expectedTheft,
		GoldAfterTheft:        expectedAfterTheft,
		DockLocation:          references.Jhelom,
		DockPosition:          shop.Berth.Position,
		PendingDelivery:       true,
		SkiffCountBefore:      0,
		SkiffCountAfter:       2,
		HullHP:                map_units.MaxHullHP,
		Direction:             references.Right,
	}, result)
	assert.Equal(t, expectedAfterTheft, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
	assert.Len(t, *units, beforeUnits+1)
	vehicle := units.GetVehicleAtPositionAndFloorOrNil(result.DockPosition, references.MainFloor)
	require.NotNil(t, vehicle)
	assert.True(t, vehicle.GetVehicleDetails().IsPendingDelivery())
	assert.Equal(t, 2, vehicle.GetVehicleDetails().GetSkiffQuantity())
	assert.Equal(t, map_units.SailFurled, vehicle.GetVehicleDetails().GetSailState())
	assert.Equal(t, map_units.MaxHullHP, vehicle.GetVehicleDetails().GetHullHP())
}

func TestPurchaseShipCreatesStandaloneSkiffAndLoadsExactFrigatePointer(t *testing.T) {
	t.Run("standalone skiff starts pending", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipMinoc)
		state.PartyState.Inventory.Gold.Set(1000)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		shop, err := state.GetShipShopState(session)
		require.NoError(t, err)

		result, err := state.PurchaseShip(session, shipPurchaseRequest(shop, shop.SkiffQuote))

		require.NoError(t, err)
		assert.Equal(t, ShipPurchaseCreateStandaloneSkiff, result.Action)
		assert.True(t, result.PendingDelivery)
		assert.Zero(t, result.SkiffCountAfter)
		vehicle := shipShopOverworldUnits(t, state).GetVehicleAtPositionAndFloorOrNil(
			result.DockPosition,
			references.MainFloor,
		)
		require.NotNil(t, vehicle)
		assert.Equal(t, references.SkiffVehicle, vehicle.GetVehicleDetails().VehicleType)
	})

	for _, pending := range []bool{true, false} {
		name := "moored"
		if pending {
			name = "pending"
		}
		t.Run(name+" frigate", func(t *testing.T) {
			state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
			state.PartyState.Inventory.Gold.Set(1000)
			position := shipShopDockPosition(t, state, references.Jhelom)
			frigate := addShipShopVehicle(t, state, references.FrigateVehicle, position, pending, 9)
			units := shipShopOverworldUnits(t, state)
			beforeUnits := len(*units)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			shop, err := state.GetShipShopState(session)
			require.NoError(t, err)

			result, err := state.PurchaseShip(session, shipPurchaseRequest(shop, shop.SkiffQuote))

			require.NoError(t, err)
			assert.Equal(t, ShipPurchaseLoadSkiff, result.Action)
			assert.Equal(t, pending, result.PendingDelivery)
			assert.Equal(t, 9, result.SkiffCountBefore)
			assert.Equal(t, 10, result.SkiffCountAfter)
			assert.Same(t, frigate, units.GetVehicleAtPositionAndFloorOrNil(position, references.MainFloor))
			assert.Len(t, *units, beforeUnits)
		})
	}
}

func TestPurchaseShipLoadsReviewedSkiffQuantityAtomically(t *testing.T) {
	const seed = 21212
	state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
	state.SetRandomSeed(seed)
	state.ShadowlordState.AutomaticPresence = references.Faulinei
	state.PartyState.Inventory.Gold.Set(2000)
	position := shipShopDockPosition(t, state, references.Jhelom)
	frigate := addShipShopVehicle(t, state, references.FrigateVehicle, position, false, 2)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetShipShopState(session)
	require.NoError(t, err)
	require.Len(t, shop.SkiffQuote.ApprovalViews, 5, "quantity is bounded by current gold")
	approval := shop.SkiffQuote.ApprovalViews[2]
	request := shipPurchaseRequest(shop, shop.SkiffQuote)
	request.Quantity = approval.Quantity
	request.ReviewedTotalPrice = approval.TotalPrice
	expectedRNG := beradrng.New(seed)
	expectedTheft := uint16(expectedRNG.Intn(64) + 1)

	result, err := state.PurchaseShip(session, request)

	require.NoError(t, err)
	assert.Equal(t, 3, result.Quantity)
	assert.Equal(t, shop.SkiffQuote.Price, result.UnitPrice)
	assert.Equal(t, shop.SkiffQuote.Price*3, result.Price)
	assert.Equal(t, 2, result.SkiffCountBefore)
	assert.Equal(t, 5, result.SkiffCountAfter)
	assert.Equal(t, 5, frigate.GetVehicleDetails().GetSkiffQuantity())
	assert.Equal(t, uint16(2000-result.Price)-expectedTheft, state.PartyState.Inventory.Gold.Get()) //nolint:gosec // Reviewed total is affordable.
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}

func TestPurchaseShipRejectsQuantityBeyondRemainingCapacityAtomically(t *testing.T) {
	state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
	state.PartyState.Inventory.Gold.Set(2000)
	position := shipShopDockPosition(t, state, references.Jhelom)
	frigate := addShipShopVehicle(t, state, references.FrigateVehicle, position, false, 11)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetShipShopState(session)
	require.NoError(t, err)
	require.Len(t, shop.SkiffQuote.ApprovalViews, 1)
	request := shipPurchaseRequest(shop, shop.SkiffQuote)
	request.Quantity = 2
	request.ReviewedTotalPrice = shop.SkiffQuote.Price * 2

	_, err = state.PurchaseShip(session, request)

	require.ErrorIs(t, err, ErrInvalidShipPurchase)
	assert.Equal(t, uint16(2000), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, 11, frigate.GetVehicleDetails().GetSkiffQuantity())
}

func TestPurchaseShipRejectsFundsRefusalsAndStaleFingerprintsAtomically(t *testing.T) {
	t.Run("insufficient funds is distinct terminal error", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		state.SetRandomSeed(212)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.PartyState.Inventory.Gold.Set(1)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		shop, err := state.GetShipShopState(session)
		require.NoError(t, err)
		beforeUnits := shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state))

		_, err = state.PurchaseShip(session, shipPurchaseRequest(shop, shop.FrigateQuote))

		require.ErrorIs(t, err, ErrShipInsufficientFunds)
		assert.NotErrorIs(t, err, ErrShipPurchaseRefused)
		assert.Equal(t, uint16(1), state.PartyState.Inventory.Gold.Get())
		assert.Zero(t, state.GetRNG().Calls())
		assert.Equal(t, beforeUnits, shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state)))
	})

	t.Run("known non-funds refusal has no effects", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		state.SetRandomSeed(212)
		state.ShadowlordState.AutomaticPresence = references.Faulinei
		state.PartyState.Inventory.Gold.Set(2000)
		position := shipShopDockPosition(t, state, references.Jhelom)
		addShipShopVehicle(t, state, references.SkiffVehicle, position, true, 0)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)
		shop, err := state.GetShipShopState(session)
		require.NoError(t, err)
		beforeUnits := shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state))

		_, err = state.PurchaseShip(session, shipPurchaseRequest(shop, shop.SkiffQuote))

		require.ErrorIs(t, err, ErrShipPurchaseRefused)
		assert.Equal(t, uint16(2000), state.PartyState.Inventory.Gold.Get())
		assert.Zero(t, state.GetRNG().Calls())
		assert.Equal(t, beforeUnits, shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state)))
	})

	staleTests := []struct {
		name   string
		mutate func(*testing.T, *GameState, MerchantSession, ShipShopState)
	}{
		{
			name: "gold",
			mutate: func(_ *testing.T, state *GameState, _ MerchantSession, _ ShipShopState) {
				state.PartyState.Inventory.Gold.Set(1999)
			},
		},
		{
			name: "buyer price",
			mutate: func(_ *testing.T, state *GameState, session MerchantSession, _ ShipShopState) {
				state.PartyState.GetCharacter(session.PricingCharacterIndex()).Intelligence++
			},
		},
		{
			name: "berth identity",
			mutate: func(t *testing.T, state *GameState, _ MerchantSession, shop ShipShopState) {
				units := shipShopOverworldUnits(t, state)
				require.True(t, units.RemoveNPCAtPositionAndFloor(shop.Berth.Position, references.MainFloor))
				addShipShopVehicle(t, state, references.FrigateVehicle, shop.Berth.Position, true, 2)
			},
		},
		{
			name: "skiff count",
			mutate: func(t *testing.T, state *GameState, _ MerchantSession, shop ShipShopState) {
				vehicle := shipShopOverworldUnits(t, state).GetVehicleAtPositionAndFloorOrNil(
					shop.Berth.Position,
					references.MainFloor,
				)
				vehicle.GetVehicleDetails().IncrementSkiffQuantity()
			},
		},
		{
			name: "pending marker",
			mutate: func(t *testing.T, state *GameState, _ MerchantSession, shop ShipShopState) {
				vehicle := shipShopOverworldUnits(t, state).GetVehicleAtPositionAndFloorOrNil(
					shop.Berth.Position,
					references.MainFloor,
				)
				vehicle.GetVehicleDetails().ClearPendingDelivery()
			},
		},
		{
			name: "capacity count",
			mutate: func(t *testing.T, state *GameState, _ MerchantSession, _ ShipShopState) {
				addShipShopVehicle(t, state, references.CarpetVehicle, references.Position{X: 1, Y: 1}, false, 0)
			},
		},
	}

	for _, test := range staleTests {
		t.Run("stale "+test.name, func(t *testing.T) {
			state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
			state.SetRandomSeed(212)
			state.ShadowlordState.AutomaticPresence = references.Faulinei
			state.PartyState.Inventory.Gold.Set(2000)
			position := shipShopDockPosition(t, state, references.Jhelom)
			addShipShopVehicle(t, state, references.FrigateVehicle, position, true, 2)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)
			shop, err := state.GetShipShopState(session)
			require.NoError(t, err)
			request := shipPurchaseRequest(shop, shop.SkiffQuote)
			test.mutate(t, state, session, shop)
			goldBefore := state.PartyState.Inventory.Gold.Get()
			unitsBefore := shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state))
			rngBefore := state.GetRNG().Calls()

			_, err = state.PurchaseShip(session, request)

			require.ErrorIs(t, err, ErrInvalidShipPurchase)
			assert.Equal(t, goldBefore, state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, rngBefore, state.GetRNG().Calls())
			assert.Equal(t, unitsBefore, shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state)))
		})
	}
}

func TestHarbourTravelReviewIsPureUsesEffectiveDockAndRevalidates(t *testing.T) {
	state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
	state.SetRandomSeed(212)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.ActiveDurationEffect = DurationEffect{}
	callbacks.BindGameState(state)
	position := shipShopDockPosition(t, state, references.Jhelom)
	frigate := addShipShopVehicle(t, state, references.FrigateVehicle, position, true, 2)
	largeMap := state.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, references.MainFloor)
	require.NotNil(t, largeMap)
	candidates := shipShopCardinalPositions(position)
	for _, candidate := range candidates {
		largeMap.SetTileByLayer(map_state.MapLayer, &candidate, indexes.Grass)
		largeMap.UnSetTileByLayer(map_state.MapOverrideLayer, &candidate)
	}
	largeMap.SetTileByLayer(map_state.MapLayer, &candidates[0], indexes.Dock)
	largeMap.SetTileByLayer(map_state.MapOverrideLayer, &candidates[0], indexes.StoneBrickWall)
	largeMap.SetTileByLayer(map_state.MapOverrideLayer, &candidates[1], indexes.Dock)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	beforeLocation := state.MapState.PlayerLocation
	beforeDate := state.DateTime
	beforeElapsed := state.ElapsedMs
	beforeRNG := state.GetRNG().Calls()
	beforeUnits := shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state))

	review, err := state.ReviewHarbourTravel(session)

	require.NoError(t, err)
	assert.Equal(t, references.MerchantShipJhelom, review.MerchantID)
	assert.Equal(t, references.Jhelom, review.DockLocation)
	assert.Equal(t, "JHELOM Harbour", review.HarbourName)
	assert.Equal(t, position, review.BerthPosition)
	assert.Equal(t, candidates[1], review.ArrivalPosition, "effective override must win over base terrain")
	assert.Equal(t, references.FrigateVehicle, review.VehicleType)
	assert.True(t, review.PendingDelivery)
	assert.Equal(t, HarbourTravelTimeCostMinutes, review.TimeCostMinutes)
	assert.Equal(t, HarbourTravelBlockNone, review.BlockReason)
	assert.NotEmpty(t, review.Fingerprint)
	require.NoError(t, state.RevalidateHarbourTravel(session, review))
	assert.Equal(t, beforeLocation, state.MapState.PlayerLocation)
	assert.Equal(t, beforeDate, state.DateTime)
	assert.Equal(t, beforeElapsed, state.ElapsedMs)
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	assert.Equal(t, beforeUnits, shipMapUnitValueSnapshots(shipShopOverworldUnits(t, state)))
	assert.Same(t, frigate, shipShopOverworldUnits(t, state).GetVehicleAtPositionAndFloorOrNil(position, references.MainFloor))
	assert.Zero(t, callbacks.MessageCallCount)
	assert.Zero(t, callbacks.AudioCallCount)
	assert.Zero(t, callbacks.VisualCallCount)
	assert.Zero(t, callbacks.ScreenCallCount)
	assert.Zero(t, callbacks.FlowCallCount)
	assert.Zero(t, callbacks.TalkCallCount)

	addShipShopVehicle(t, state, references.CarpetVehicle, review.ArrivalPosition, false, 0)
	err = state.RevalidateHarbourTravel(session, review)
	require.ErrorIs(t, err, ErrHarbourTravelUnavailable)
}

func TestHarbourTravelReviewBlocksWithoutVehicleDockOrOnFootState(t *testing.T) {
	t.Run("no berth vehicle", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		review, err := state.ReviewHarbourTravel(session)

		require.NoError(t, err)
		assert.Equal(t, HarbourTravelBlockNoBerthVehicle, review.BlockReason)
		require.ErrorIs(t, state.RevalidateHarbourTravel(session, review), ErrHarbourTravelUnavailable)
	})

	t.Run("not on foot", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		position := shipShopDockPosition(t, state, references.Jhelom)
		addShipShopVehicle(t, state, references.FrigateVehicle, position, true, 2)
		setPartyTestVehicle(state, references.HorseVehicle)
		session, err := state.beginMerchantSession(npc, true)
		require.NoError(t, err)

		review, err := state.ReviewHarbourTravel(session)

		require.NoError(t, err)
		assert.Equal(t, HarbourTravelBlockNotOnFoot, review.BlockReason)
	})

	t.Run("no exact adjacent Dock terrain", func(t *testing.T) {
		state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
		position := shipShopDockPosition(t, state, references.Jhelom)
		addShipShopVehicle(t, state, references.SkiffVehicle, position, true, 0)
		largeMap := state.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, references.MainFloor)
		for _, candidate := range shipShopCardinalPositions(position) {
			largeMap.SetTileByLayer(map_state.MapOverrideLayer, &candidate, indexes.Grass)
		}
		session, err := state.BeginMerchantSession(npc)
		require.NoError(t, err)

		review, err := state.ReviewHarbourTravel(session)

		require.NoError(t, err)
		assert.Equal(t, HarbourTravelBlockNoAdjacentDock, review.BlockReason)
	})
}

func TestHarbourTravelReviewUsesCanonicalAdjacentDockByDealer(t *testing.T) {
	tests := []struct {
		name       string
		merchantID references.MerchantID
		location   references.Location
		offset     references.Position
	}{
		{name: "Jhelom below", merchantID: references.MerchantShipJhelom, location: references.Jhelom,
			offset: references.Position{X: 0, Y: 1}},
		{name: "Minoc above", merchantID: references.MerchantShipMinoc, location: references.Minoc,
			offset: references.Position{X: 0, Y: -1}},
		{name: "East Brittany above", merchantID: references.MerchantShipEastBritanny, location: references.EastBritanny,
			offset: references.Position{X: 0, Y: -1}},
		{name: "Buccaneer's Den left", merchantID: references.MerchantShipBuccaneersDen, location: references.BuccaneersDen,
			offset: references.Position{X: -1, Y: 0}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, npc := newOpenShipShopTestState(t, test.merchantID)
			berth := shipShopDockPosition(t, state, test.location)
			addShipShopVehicle(t, state, references.FrigateVehicle, berth, true, 2)
			session, err := state.BeginMerchantSession(npc)
			require.NoError(t, err)

			review, err := state.ReviewHarbourTravel(session)

			require.NoError(t, err)
			require.Equal(t, HarbourTravelBlockNone, review.BlockReason)
			assert.Equal(t, references.Position{X: berth.X + test.offset.X, Y: berth.Y + test.offset.Y}, review.ArrivalPosition)
		})
	}
}

func TestShipPurchasePendingDeliverySurvivesCodecHydrationAndActivatesOnce(t *testing.T) {
	state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
	state.SetRandomSeed(212)
	state.PartyState.Inventory.Gold.Set(2000)
	state.PartyState.GetCharacter(0).SetMp(0)
	state.MapState.PlayerLocation.Position = npc.Pos()
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	shop, err := state.GetShipShopState(session)
	require.NoError(t, err)
	purchase, err := state.PurchaseShip(session, shipPurchaseRequest(shop, shop.FrigateQuote))
	require.NoError(t, err)
	travel, err := state.ReviewHarbourTravel(session)
	require.NoError(t, err)
	require.Equal(t, HarbourTravelBlockNone, travel.BlockReason)
	state.EndMerchantSession(session)

	captured, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	encoded, err := save_snapshot.EncodeCanonical(save_snapshot.Envelope{
		Format:          save_snapshot.EnvelopeFormat,
		EnvelopeVersion: save_snapshot.CurrentEnvelopeVersion,
		Metadata: save_snapshot.Metadata{
			Name: "Ship pending delivery lifecycle", CreatedAt: time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC),
		},
		Snapshot: captured,
	})
	require.NoError(t, err)
	decoded, err := save_snapshot.Decode(encoded)
	require.NoError(t, err)
	hydrated, err := NewGameStateFromSaveSnapshot(
		decoded.Snapshot,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	assert.Equal(t, references.Jhelom, hydrated.MapState.PlayerLocation.Location)
	units := hydrated.LargeMapNPCAIController[references.OVERWORLD].GetNpcs()
	require.Len(t, *units, 1)
	delivery := units.GetVehicleAtPositionAndFloorOrNil(purchase.DockPosition, references.MainFloor)
	require.NotNil(t, delivery)
	assert.True(t, delivery.GetVehicleDetails().IsPendingDelivery())

	hydrated.SystemCallbacks = NewMockSystemCallbacks(t).ToSystemCallbacks()
	hydrated.TransitionToLocation(
		references.BritanniaUnderworld,
		references.MainFloor,
		travel.ArrivalPosition,
		TransitionFullLocation,
		false,
	)
	assert.Same(t, delivery, units.GetVehicleAtPositionAndFloorOrNil(purchase.DockPosition, references.MainFloor))
	assert.False(t, delivery.GetVehicleDetails().IsPendingDelivery())
	require.Len(t, *units, 1)

	hydrated.TransitionToLocation(
		references.BritanniaUnderworld,
		references.MainFloor,
		travel.ArrivalPosition,
		TransitionFullLocation,
		false,
	)
	assert.Same(t, delivery, units.GetVehicleAtPositionAndFloorOrNil(purchase.DockPosition, references.MainFloor))
	require.Len(t, *units, 1)
	recaptured, err := hydrated.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Empty(t, save_snapshot.Validate(recaptured))
}

func TestThenHarbourTravelTransitionAdvancesTimeActivatesInPlaceAndLeavesPartyOnFoot(t *testing.T) {
	state, npc := newOpenShipShopTestState(t, references.MerchantShipJhelom)
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	berth := shipShopDockPosition(t, state, references.Jhelom)
	frigate := addShipShopVehicle(t, state, references.FrigateVehicle, berth, true, 2)
	session, err := state.BeginMerchantSession(npc)
	require.NoError(t, err)
	review, err := state.ReviewHarbourTravel(session)
	require.NoError(t, err)
	require.NoError(t, state.RevalidateHarbourTravel(session, review))
	state.EndMerchantSession(session)
	units := shipShopOverworldUnits(t, state)
	unitCount := len(*units)
	beforeTravel := state.DateTime

	state.EventQueueManager.Workflow("harbour_travel_transition_test").
		Priority(event_queue.PriorityNormal).
		Sequential().
		First(func() {}).
		ThenSequence(state.ThenHarbourTravelTransition(review)).
		Start()
	for state.EventQueueManager.HasActiveWorkflows() {
		state.ElapsedMs += 100
		state.EventQueueManager.Update()
	}

	expectedTravel := beforeTravel
	expectedTravel.Advance(HarbourTravelTimeCostMinutes)
	assert.Equal(t, expectedTravel, state.DateTime)
	assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
	assert.Equal(t, references.MainFloor, state.MapState.PlayerLocation.Floor)
	assert.Equal(t, review.ArrivalPosition, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.NoPartyVehicle, state.PartyVehicle.GetVehicleDetails().VehicleType)
	assert.Len(t, *units, unitCount)
	assert.Same(t, frigate, units.GetVehicleAtPositionAndFloorOrNil(berth, references.MainFloor))
	assert.False(t, frigate.GetVehicleDetails().IsPendingDelivery())
	require.NotEmpty(t, callbacks.MapFadeLevels)
	assert.Zero(t, callbacks.MapFadeLevels[len(callbacks.MapFadeLevels)-1])
}

func newOpenShipShopTestState(
	t *testing.T,
	merchantID references.MerchantID,
) (*GameState, *map_units.NPCFriendly) {
	t.Helper()
	state, npc := newOpenMerchantTestState(t, merchantID)
	state.UpdateSmallMap(state.GameReferences.TileReferences, state.GameReferences.LocationReferences)
	installMerchantTestNPC(t, state, npc)
	clearShipShopOverworldUnits(t, state)
	state.PartyState.Inventory.Gold.Set(5000)

	return state, npc
}

func clearShipShopOverworldUnits(t *testing.T, state *GameState) {
	t.Helper()
	units := shipShopOverworldUnits(t, state)
	units.ClearAllMapUnits()
}

func shipShopOverworldUnits(t *testing.T, state *GameState) *map_units.MapUnits {
	t.Helper()
	controller := state.LargeMapNPCAIController[references.OVERWORLD]
	require.NotNil(t, controller)
	require.NotNil(t, controller.GetNpcs())

	return controller.GetNpcs()
}

func shipShopDockPosition(
	t *testing.T,
	state *GameState,
	location references.Location,
) references.Position {
	t.Helper()
	position, ok := state.GameReferences.DockReferences.GetDockPositionChecked(location)
	require.True(t, ok)

	return position
}

func addShipShopVehicle(
	t *testing.T,
	state *GameState,
	vehicleType references.VehicleType,
	position references.Position,
	pending bool,
	skiffs int,
) *map_units.NPCFriendly {
	t.Helper()
	vehicle := map_units.NewNPCFriendlyVehiceNewRef(vehicleType, position, references.MainFloor)
	vehicle.GetVehicleDetails().SetSkiffQuantity(skiffs)
	if pending {
		vehicle.GetVehicleDetails().MarkPendingDelivery()
	}
	stored := shipShopOverworldUnits(t, state).InsertVehicle(*vehicle)
	require.NotNil(t, stored)

	return stored
}

func fillShipShopOverworldCapacity(t *testing.T, state *GameState) {
	t.Helper()
	units := shipShopOverworldUnits(t, state)
	for len(*units) < map_units.MaximumNpcsPerMap {
		index := len(*units)
		position := references.Position{X: references.Coordinate(index + 1), Y: 1}
		vehicle := map_units.NewNPCFriendlyVehiceNewRef(
			references.CarpetVehicle,
			position,
			references.MainFloor,
		)
		require.NotNil(t, units.InsertVehicle(*vehicle))
	}
}

func shipPurchaseRequest(shop ShipShopState, quote ShipPurchaseQuote) ShipPurchaseRequest {
	approval := ShipPurchaseApprovalView{Quantity: 1, TotalPrice: quote.Price}
	if len(quote.ApprovalViews) > 0 {
		approval = quote.ApprovalViews[0]
	}

	return ShipPurchaseRequest{
		ReviewedMerchantID:            shop.Merchant.ID,
		ReviewedPricingCharacterIndex: shop.PricingCharacterIndex,
		VehicleType:                   quote.VehicleType,
		Action:                        quote.Action,
		Quantity:                      approval.Quantity,
		ReviewedPrice:                 quote.Price,
		ReviewedTotalPrice:            approval.TotalPrice,
		ReviewedGoldBefore:            quote.GoldBefore,
		ReviewedFingerprint:           quote.RequestFingerprint,
	}
}

func shipShopCardinalPositions(position references.Position) [4]references.Position {
	return [4]references.Position{
		{X: position.X, Y: position.Y + 1},
		{X: position.X, Y: position.Y - 1},
		{X: position.X + 1, Y: position.Y},
		{X: position.X - 1, Y: position.Y},
	}
}

type shipMapUnitValueSnapshot struct {
	MapUnitType    map_units.MapUnitType
	Position       references.Position
	Floor          references.FloorNumber
	Visible        bool
	Empty          bool
	NPCNum         int
	Vehicle        bool
	VehicleDetails map_units.VehicleDetailsSnapshot
}

func shipMapUnitValueSnapshots(units *map_units.MapUnits) []shipMapUnitValueSnapshot {
	result := make([]shipMapUnitValueSnapshot, 0, len(*units))
	for _, mapUnit := range *units {
		snapshot := shipMapUnitValueSnapshot{
			MapUnitType: mapUnit.GetMapUnitType(),
			Position:    mapUnit.Pos(),
			Floor:       mapUnit.Floor(),
			Visible:     mapUnit.IsVisible(),
			Empty:       mapUnit.IsEmptyMapUnit(),
			NPCNum:      mapUnit.MapUnitDetails().NPCNum,
		}
		if friendly, ok := mapUnit.(*map_units.NPCFriendly); ok &&
			friendly.NPCReference.GetNPCType() == references.Vehicle {
			snapshot.Vehicle = true
			snapshot.VehicleDetails = friendly.GetVehicleDetails().Snapshot()
		}
		result = append(result, snapshot)
	}

	return result
}
