package game_state

import (
	"errors"
	"fmt"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var (
	// ErrInvalidHorsePurchase indicates stale or unsupported reviewed input.
	ErrInvalidHorsePurchase = errors.New("invalid horse purchase")
	// ErrHorseStableClosed indicates that capacity or adjacent placement is unavailable.
	ErrHorseStableClosed = errors.New("horse stable is closed")
	// ErrHorseInsufficientFunds indicates that the reviewed horse cannot currently be afforded.
	ErrHorseInsufficientFunds = errors.New("insufficient funds for horse")
)

// HorseShopBlockStatus identifies current delivery availability.
type HorseShopBlockStatus byte

const (
	// HorseShopBlockNone indicates that the reviewed placement can accept a horse.
	HorseShopBlockNone HorseShopBlockStatus = iota
	// HorseShopBlockAvailabilityDeferred indicates that remote Preview cannot inspect the destination map.
	HorseShopBlockAvailabilityDeferred
	// HorseShopBlockMapUnitCapacity indicates that the current small map has no free map-unit slot.
	HorseShopBlockMapUnitCapacity
	// HorseShopBlockNoLegalPlacement indicates that no adjacent legal unoccupied tile exists.
	HorseShopBlockNoLegalPlacement
)

// IsBlocked reports whether a horse cannot be committed from this review.
func (s HorseShopBlockStatus) IsBlocked() bool { return s != HorseShopBlockNone }

// HorsePlacementReview is the detached internal delivery fingerprint reviewed before approval.
type HorsePlacementReview struct {
	Position    references.Position
	Floor       references.FloorNumber
	Terrain     indexes.SpriteIndex
	BlockStatus HorseShopBlockStatus
}

// HorsePurchaseQuote describes the canonical horse price and reviewed delivery state.
type HorsePurchaseQuote struct {
	BasePrice        int
	Price            int
	GoldBefore       uint16
	GoldAfterPayment uint16
	Affordable       bool
	Placement        HorsePlacementReview
}

// HorseShopState is a trusted detached snapshot of one horse dealer.
type HorseShopState struct {
	Merchant              references.MerchantDefinition
	PricingCharacterIndex int
	PricingCharacterName  string
	PricingIntelligence   int
	Gold                  uint16
	Quote                 HorsePurchaseQuote
}

// HorseShopReview is a read-only production snapshot with canonical sprite identity.
type HorseShopReview struct {
	State               HorseShopState
	MerchantSpriteIndex indexes.SpriteIndex
}

// HorsePurchaseRequest carries comparison-only review identity. GameState
// derives the merchant, buyer, price, gold, and placement again before committing.
type HorsePurchaseRequest struct {
	ReviewedMerchantID            references.MerchantID
	ReviewedPricingCharacterIndex int
	ReviewedPrice                 int
	ReviewedGoldBefore            uint16
	ReviewedPlacement             HorsePlacementReview
}

// HorsePurchaseResult reports one complete authoritative horse purchase.
type HorsePurchaseResult struct {
	MerchantID            references.MerchantID
	PricingCharacterIndex int
	BasePrice             int
	Price                 int
	GoldBefore            uint16
	GoldAfterPayment      uint16
	Theft                 uint16
	GoldAfterTheft        uint16
	VehicleType           references.VehicleType
	Position              references.Position
	Floor                 references.FloorNumber
	Direction             references.Direction
	OwnedByNPC            bool
}

// GetHorseShopState returns the current canonical quote after checking delivery first.
func (g *GameState) GetHorseShopState(session MerchantSession) (HorseShopState, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyHorse)
	if err != nil {
		return HorseShopState{}, err
	}

	placement := g.reviewHorsePlacement()

	return g.buildHorseShopState(merchant, buyer, session.buyerIndex, placement)
}

// ReviewHorseShopByMerchantID builds a production price without issuing a
// session, invoking callbacks, consuming RNG, advancing time, or mutating map state.
func (g *GameState) ReviewHorseShopByMerchantID(id references.MerchantID) (HorseShopReview, error) {
	if g == nil || g.GameReferences == nil || g.GameReferences.MerchantReferences == nil ||
		g.GameReferences.NPCReferences == nil || g.GameReferences.LocationReferences == nil {
		return HorseShopReview{}, fmt.Errorf("%w: merchant references are unavailable", ErrInvalidMerchantSession)
	}
	merchant, ok := g.GameReferences.MerchantReferences.Definition(id)
	if !ok {
		return HorseShopReview{}, fmt.Errorf("%w: unknown merchant ID %q", ErrInvalidMerchantSession, id)
	}
	if merchant.Family != references.MerchantFamilyHorse {
		return HorseShopReview{}, fmt.Errorf("%w: merchant %q is not a horse dealer", ErrInvalidHorsePurchase, id)
	}
	if merchant.Location.GetMapType() != references.SmallMapType ||
		g.GameReferences.LocationReferences.GetLocationReference(merchant.Location) == nil {
		return HorseShopReview{}, fmt.Errorf(
			"%w: merchant %q has no canonical small-map location",
			ErrInvalidMerchantSession,
			id,
		)
	}
	npcs := g.GameReferences.NPCReferences.GetNPCReferencesByLocation(merchant.Location)
	if npcs == nil || merchant.NPCIndex < 0 || merchant.NPCIndex >= len(*npcs) {
		return HorseShopReview{}, fmt.Errorf(
			"%w: merchant %q has no canonical NPC reference",
			ErrInvalidMerchantSession,
			id,
		)
	}
	npcReference := (*npcs)[merchant.NPCIndex]
	resolved, err := g.GameReferences.MerchantReferences.Resolve(
		references.DialogNumber(npcReference.DialogNumber),
		npcReference.Location,
		merchant.NPCIndex,
	)
	if err != nil || resolved != merchant {
		return HorseShopReview{}, fmt.Errorf(
			"%w: canonical NPC reference does not match merchant %q",
			ErrInvalidMerchantSession,
			id,
		)
	}
	buyer, buyerIndex := g.merchantPricingCharacter()
	if buyer == nil {
		return HorseShopReview{}, ErrMerchantBuyerUnavailable
	}

	placement := HorsePlacementReview{BlockStatus: HorseShopBlockAvailabilityDeferred}
	if g.canReviewLoadedHorsePlacement(merchant) {
		placement = g.reviewHorsePlacement()
	}
	state, err := g.buildHorseShopState(merchant, buyer, buyerIndex, placement)
	if err != nil {
		return HorseShopReview{}, err
	}

	return HorseShopReview{State: state, MerchantSpriteIndex: npcReference.GetSpriteIndex()}, nil
}

// QuoteHorsePurchase returns the current canonical price and delivery fingerprint.
func (g *GameState) QuoteHorsePurchase(session MerchantSession) (HorsePurchaseQuote, error) {
	state, err := g.GetHorseShopState(session)
	if err != nil {
		return HorsePurchaseQuote{}, err
	}

	return state.Quote, nil
}

// PurchaseHorse revalidates and commits one original horse transaction.
func (g *GameState) PurchaseHorse(
	session MerchantSession,
	request HorsePurchaseRequest,
) (HorsePurchaseResult, error) {
	state, err := g.GetHorseShopState(session)
	if err != nil {
		return HorsePurchaseResult{}, err
	}
	if state.Quote.Placement.BlockStatus.IsBlocked() {
		return HorsePurchaseResult{}, fmt.Errorf(
			"%w: delivery status %d",
			ErrHorseStableClosed,
			state.Quote.Placement.BlockStatus,
		)
	}
	if request.ReviewedPlacement.BlockStatus != HorseShopBlockNone {
		return HorsePurchaseResult{}, fmt.Errorf(
			"%w: reviewed delivery status %d is not purchasable",
			ErrInvalidHorsePurchase,
			request.ReviewedPlacement.BlockStatus,
		)
	}
	if request.ReviewedPlacement != state.Quote.Placement {
		return HorsePurchaseResult{}, fmt.Errorf(
			"%w: reviewed horse placement changed",
			ErrHorseStableClosed,
		)
	}
	if request.ReviewedMerchantID != state.Merchant.ID ||
		request.ReviewedPricingCharacterIndex != state.PricingCharacterIndex ||
		request.ReviewedPrice != state.Quote.Price ||
		request.ReviewedGoldBefore != state.Quote.GoldBefore {
		return HorsePurchaseResult{}, fmt.Errorf(
			"%w: reviewed merchant, buyer, price, or gold changed",
			ErrInvalidHorsePurchase,
		)
	}
	if !state.Quote.Affordable {
		return HorsePurchaseResult{}, fmt.Errorf(
			"%w: need %d gold, have %d",
			ErrHorseInsufficientFunds,
			state.Quote.Price,
			state.Gold,
		)
	}

	result := HorsePurchaseResult{
		MerchantID:            state.Merchant.ID,
		PricingCharacterIndex: state.PricingCharacterIndex,
		BasePrice:             state.Quote.BasePrice,
		Price:                 state.Quote.Price,
		GoldBefore:            state.Gold,
		GoldAfterPayment:      state.Gold,
		GoldAfterTheft:        state.Gold,
		VehicleType:           references.HorseVehicle,
		Position:              state.Quote.Placement.Position,
		Floor:                 state.Quote.Placement.Floor,
	}
	price := uint16(state.Quote.Price) //nolint:gosec // The quote validates the price against uint16 gold.
	if !g.PartyState.Inventory.Gold.DecrementBy(price) {
		panic("validated horse purchase could not deduct gold")
	}
	result.GoldAfterPayment = g.PartyState.Inventory.Gold.Get()
	result.Theft = g.ApplyFaulineiShopTheft()
	result.GoldAfterTheft = g.PartyState.Inventory.Gold.Get()

	units := g.CurrentNPCAIController.GetNpcs()
	horse := map_units.NewNPCFriendlyVehiceNewRef(
		references.HorseVehicle,
		state.Quote.Placement.Position,
		state.Quote.Placement.Floor,
	)
	if !units.AddVehicle(*horse) {
		panic("preflighted horse insertion failed")
	}
	inserted := units.GetMapUnitAtPositionAndFloorOrNil(result.Position, result.Floor)
	if inserted == nil {
		panic("inserted horse is missing from reviewed placement")
	}
	insertedHorse, ok := (*inserted).(*map_units.NPCFriendly)
	if !ok || insertedHorse.GetVehicleDetails().VehicleType != references.HorseVehicle {
		panic("reviewed horse placement contains unexpected map unit after insertion")
	}
	result.Direction = insertedHorse.GetVehicleDetails().CurrentDirection()
	result.OwnedByNPC = insertedHorse.GetVehicleDetails().IsOwnedByNPC
	g.SystemCallbacks.Screen.RefreshInventoryDisplay()

	return result, nil
}

func (g *GameState) buildHorseShopState(
	merchant references.MerchantDefinition,
	buyer *party_state.PlayerCharacter,
	buyerIndex int,
	placement HorsePlacementReview,
) (HorseShopState, error) {
	offer, ok := g.GameReferences.MerchantReferences.HorseOffer(merchant.ID)
	if !ok {
		return HorseShopState{}, fmt.Errorf("%w: no horse offer for merchant %q", ErrInvalidHorsePurchase, merchant.ID)
	}
	price := CalculateMerchantBuyPrice(offer.BasePrice, int(buyer.Intelligence))
	if offer.BasePrice <= 0 || price <= 0 || price > math.MaxUint16 {
		return HorseShopState{}, fmt.Errorf("%w: calculated horse price %d is invalid", ErrInvalidHorsePurchase, price)
	}
	gold := g.PartyState.Inventory.Gold.Get()
	affordable := int(gold) >= price
	goldAfterPayment := gold
	if affordable {
		goldAfterPayment = gold - uint16(price) //nolint:gosec // Affordability and range checks make conversion safe.
	}

	return HorseShopState{
		Merchant:              merchant,
		PricingCharacterIndex: buyerIndex,
		PricingCharacterName:  buyer.GetNameAsString(),
		PricingIntelligence:   int(buyer.Intelligence),
		Gold:                  gold,
		Quote: HorsePurchaseQuote{
			BasePrice:        offer.BasePrice,
			Price:            price,
			GoldBefore:       gold,
			GoldAfterPayment: goldAfterPayment,
			Affordable:       affordable,
			Placement:        placement,
		},
	}, nil
}

func (g *GameState) canReviewLoadedHorsePlacement(merchant references.MerchantDefinition) bool {
	if g.MapState.PlayerLocation.Location != merchant.Location ||
		g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType ||
		g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return false
	}

	for _, mapUnit := range *g.CurrentNPCAIController.GetNpcs() {
		npc, ok := mapUnit.(*map_units.NPCFriendly)
		if !ok || !npc.IsVisible() || npc.Floor() != g.MapState.PlayerLocation.Floor {
			continue
		}
		if npc.MapUnitDetails().NPCNum == merchant.NPCIndex &&
			npc.NPCReference.Location == merchant.Location &&
			npc.NPCReference.DialogNumber == byte(references.MerchantFamilyHorse) {
			return true
		}
	}

	return false
}

func (g *GameState) reviewHorsePlacement() HorsePlacementReview {
	if g == nil || g.MapState.PlayerLocation.Location.GetMapType() != references.SmallMapType ||
		g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return HorsePlacementReview{BlockStatus: HorseShopBlockNoLegalPlacement}
	}
	units := g.CurrentNPCAIController.GetNpcs()
	if len(*units) >= map_units.MaximumNpcsPerMap {
		return HorsePlacementReview{BlockStatus: HorseShopBlockMapUnitCapacity}
	}
	layeredMap := g.MapState.LayeredMaps.GetLayeredMap(
		references.SmallMapType,
		g.MapState.PlayerLocation.Floor,
	)
	if layeredMap == nil {
		return HorsePlacementReview{BlockStatus: HorseShopBlockNoLegalPlacement}
	}

	playerPosition := g.MapState.PlayerLocation.Position
	floor := g.MapState.PlayerLocation.Floor
	for _, offset := range [...]references.Position{
		{X: 0, Y: 1},
		{X: 0, Y: -1},
		{X: 1, Y: 0},
		{X: -1, Y: 0},
	} {
		position := references.Position{X: playerPosition.X + offset.X, Y: playerPosition.Y + offset.Y}
		tile := layeredMap.GetTileTopMapOnlyTile(&position)
		if tile == nil || !isHorseShopTerrain(tile.Index) ||
			units.GetMapUnitAtPositionAndFloorOrNil(position, floor) != nil {
			continue
		}

		return HorsePlacementReview{
			Position:    position,
			Floor:       floor,
			Terrain:     tile.Index,
			BlockStatus: HorseShopBlockNone,
		}
	}

	return HorsePlacementReview{BlockStatus: HorseShopBlockNoLegalPlacement}
}

func isHorseShopTerrain(tile indexes.SpriteIndex) bool {
	return tile == indexes.BrickFloor || tile == indexes.MetalFloor || tile == indexes.Grass
}
