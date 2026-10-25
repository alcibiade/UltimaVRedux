package game_state

import (
	"errors"
	"fmt"
	"hash"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	// ShipShopMaximumSkiffs aliases the shared deliberate Redux frigate capacity.
	ShipShopMaximumSkiffs = MaxFrigateSkiffs
	// HarbourTravelTimeCostMinutes is applied by the transition workflow, not review authority.
	HarbourTravelTimeCostMinutes = 3
)

var (
	// ErrInvalidShipPurchase indicates stale, tampered, or unsupported reviewed input.
	ErrInvalidShipPurchase = errors.New("invalid ship purchase")
	// ErrShipPurchaseRefused indicates a current non-funds delivery refusal.
	ErrShipPurchaseRefused = errors.New("ship purchase refused")
	// ErrShipInsufficientFunds is the terminal ship-shop failure used by workflow orchestration.
	ErrShipInsufficientFunds = errors.New("insufficient funds for ship purchase")
	// ErrHarbourTravelUnavailable indicates that reviewed harbour travel is no longer safe.
	ErrHarbourTravelUnavailable = errors.New("harbour travel unavailable")
)

// ShipBerthState identifies the exact current use of a canonical outdoor berth.
type ShipBerthState byte

// Canonical berth states distinguish delivery status and unrelated occupancy.
const (
	ShipBerthEmpty ShipBerthState = iota
	ShipBerthPendingFrigate
	ShipBerthPendingSkiff
	ShipBerthMooredFrigate
	ShipBerthMooredSkiff
	ShipBerthObstructed
)

// ShipPurchaseAction identifies the authoritative effect selected by a quote.
type ShipPurchaseAction byte

// Purchase actions describe creation versus loading an existing frigate.
const (
	ShipPurchaseCreateFrigate ShipPurchaseAction = iota
	ShipPurchaseCreateStandaloneSkiff
	ShipPurchaseLoadSkiff
)

// ShipPurchaseBlockReason identifies a known, nonterminal delivery refusal.
type ShipPurchaseBlockReason byte

// Purchase block reasons are known nonterminal delivery refusals.
const (
	ShipPurchaseBlockNone ShipPurchaseBlockReason = iota
	ShipPurchaseBlockPendingOrderElsewhere
	ShipPurchaseBlockBerthOccupied
	ShipPurchaseBlockStandaloneSkiff
	ShipPurchaseBlockFrigateAlreadyPresent
	ShipPurchaseBlockMapCapacity
	ShipPurchaseBlockSkiffLimit
)

// IsBlocked reports whether the reviewed action must be refused.
func (r ShipPurchaseBlockReason) IsBlocked() bool { return r != ShipPurchaseBlockNone }

// ShipBerthReview is a detached description of one canonical outdoor berth.
type ShipBerthReview struct {
	Location       references.Location
	Position       references.Position
	State          ShipBerthState
	VehicleType    references.VehicleType
	Pending        bool
	SkiffCount     int
	MapUnitCount   int
	HasMapCapacity bool
}

// ShipPurchaseQuote describes one canonical offer against current delivery authority.
type ShipPurchaseQuote struct {
	VehicleType        references.VehicleType
	Action             ShipPurchaseAction
	BasePrice          int
	Price              int
	GoldBefore         uint16
	GoldAfterPayment   uint16
	Affordable         bool
	BlockReason        ShipPurchaseBlockReason
	Berth              ShipBerthReview
	RequestFingerprint string
	ApprovalViews      []ShipPurchaseApprovalView
}

// ShipPurchaseApprovalView is one authority-supplied quantity projection.
type ShipPurchaseApprovalView struct {
	Quantity         int
	UnitPrice        int
	TotalPrice       int
	GoldBefore       uint16
	GoldAfterPayment uint16
	Affordable       bool
	SkiffCountBefore int
	SkiffCountAfter  int
}

// ShipShopState is a trusted detached snapshot of one ship dealer.
type ShipShopState struct {
	Merchant              references.MerchantDefinition
	PricingCharacterIndex int
	PricingCharacterName  string
	PricingIntelligence   int
	Gold                  uint16
	Berth                 ShipBerthReview
	Travel                HarbourTravelReview
	FrigateQuote          ShipPurchaseQuote
	SkiffQuote            ShipPurchaseQuote
}

// ShipShopReview is a pure production snapshot with canonical sprite identity.
type ShipShopReview = MerchantShopReview[ShipShopState]

// ShipPurchaseRequest carries comparison-only reviewed authority.
type ShipPurchaseRequest struct {
	ReviewedMerchantID            references.MerchantID
	ReviewedPricingCharacterIndex int
	VehicleType                   references.VehicleType
	Action                        ShipPurchaseAction
	Quantity                      int
	ReviewedPrice                 int
	ReviewedTotalPrice            int
	ReviewedGoldBefore            uint16
	ReviewedFingerprint           string
}

// ShipPurchaseResult reports one complete authoritative ship purchase.
type ShipPurchaseResult struct {
	MerchantID            references.MerchantID
	PricingCharacterIndex int
	VehicleType           references.VehicleType
	Action                ShipPurchaseAction
	Quantity              int
	BasePrice             int
	UnitPrice             int
	Price                 int
	GoldBefore            uint16
	GoldAfterPayment      uint16
	Theft                 uint16
	GoldAfterTheft        uint16
	DockLocation          references.Location
	DockPosition          references.Position
	PendingDelivery       bool
	SkiffCountBefore      int
	SkiffCountAfter       int
	HullHP                int
	Direction             references.Direction
}

type shipBerthAuthority struct {
	review  ShipBerthReview
	vehicle *map_units.NPCFriendly
}

// GetShipShopState returns both canonical offers and the current berth review.
func (g *GameState) GetShipShopState(session MerchantSession) (ShipShopState, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyShip)
	if err != nil {
		return ShipShopState{}, err
	}

	return g.buildShipShopState(merchant, buyer, session.buyerIndex)
}

// ReviewShipShopByMerchantID builds production quotes without issuing a
// session, invoking callbacks, consuming RNG, advancing time, or mutating units.
func (g *GameState) ReviewShipShopByMerchantID(id references.MerchantID) (ShipShopReview, error) {
	return reviewMerchantShop(g, id, references.MerchantFamilyShip, ErrInvalidShipPurchase, "a ship dealer", g.buildShipShopState)
}

// PurchaseShip revalidates and commits one canonical vessel purchase.
func (g *GameState) PurchaseShip(
	session MerchantSession,
	request ShipPurchaseRequest,
) (ShipPurchaseResult, error) {
	state, err := g.GetShipShopState(session)
	if err != nil {
		return ShipPurchaseResult{}, err
	}
	quote, err := shipQuoteForVehicle(state, request.VehicleType)
	if err != nil {
		return ShipPurchaseResult{}, err
	}
	if request.ReviewedMerchantID != state.Merchant.ID ||
		request.ReviewedPricingCharacterIndex != state.PricingCharacterIndex ||
		request.Action != quote.Action || request.ReviewedPrice != quote.Price ||
		request.ReviewedGoldBefore != quote.GoldBefore || request.ReviewedFingerprint == "" ||
		request.ReviewedFingerprint != quote.RequestFingerprint {
		return ShipPurchaseResult{}, fmt.Errorf(
			"%w: reviewed merchant, buyer, vehicle, action, price, gold, berth, or capacity changed",
			ErrInvalidShipPurchase,
		)
	}
	if quote.BlockReason.IsBlocked() {
		return ShipPurchaseResult{}, fmt.Errorf(
			"%w: delivery block reason %d",
			ErrShipPurchaseRefused,
			quote.BlockReason,
		)
	}
	approval, ok := shipPurchaseApproval(quote, request.Quantity)
	if !ok || request.ReviewedTotalPrice != approval.TotalPrice {
		return ShipPurchaseResult{}, fmt.Errorf(
			"%w: quantity %d or reviewed total price %d is invalid",
			ErrInvalidShipPurchase,
			request.Quantity,
			request.ReviewedTotalPrice,
		)
	}
	if !approval.Affordable {
		return ShipPurchaseResult{}, fmt.Errorf(
			"%w: need %d gold, have %d",
			ErrShipInsufficientFunds,
			approval.TotalPrice,
			quote.GoldBefore,
		)
	}
	berth, err := g.reviewShipBerth(state.Merchant.Location)
	if err != nil {
		return ShipPurchaseResult{}, fmt.Errorf("%w: revalidating berth: %w", ErrShipPurchaseRefused, err)
	}
	result := ShipPurchaseResult{
		MerchantID:            state.Merchant.ID,
		PricingCharacterIndex: state.PricingCharacterIndex,
		VehicleType:           quote.VehicleType,
		Action:                quote.Action,
		Quantity:              approval.Quantity,
		BasePrice:             quote.BasePrice,
		UnitPrice:             quote.Price,
		Price:                 approval.TotalPrice,
		GoldBefore:            quote.GoldBefore,
		GoldAfterPayment:      quote.GoldBefore,
		GoldAfterTheft:        quote.GoldBefore,
		DockLocation:          quote.Berth.Location,
		DockPosition:          quote.Berth.Position,
		SkiffCountBefore:      quote.Berth.SkiffCount,
	}
	// REDUX DEVIATION: vessel payment pays and steals per loaded skiff.
	// The default option batches those boundaries for concise keyboard shopping.
	// See ref:MERCH-0008.
	units := approval.Quantity
	if !g.GameConfig.GameplayConfiguration.BatchMerchantTheft {
		units = 1
	}
	result.Quantity = 0
	result.Price = 0
	var delivered *map_units.NPCFriendly
	for result.Quantity < approval.Quantity {
		price := uint16(quote.Price * units)
		if !g.PartyState.Inventory.Gold.DecrementBy(price) {
			break
		}
		result.GoldAfterPayment = g.PartyState.Inventory.Gold.Get()
		result.Theft += g.ApplyFaulineiShopTheft()
		result.GoldAfterTheft = g.PartyState.Inventory.Gold.Get()
		switch quote.Action {
		case ShipPurchaseCreateFrigate, ShipPurchaseCreateStandaloneSkiff:
			delivered, err = g.PlaceVehicleAtBritanniaDock(quote.VehicleType, state.Merchant.Location, true)
			if err != nil {
				panic(fmt.Sprintf("preflighted ship insertion failed: %v", err))
			}
		case ShipPurchaseLoadSkiff:
			if berth.vehicle == nil || berth.vehicle.GetVehicleDetails().VehicleType != references.FrigateVehicle {
				panic("preflighted frigate berth is unavailable")
			}
			berth.vehicle.GetVehicleDetails().SetSkiffQuantity(berth.vehicle.GetVehicleDetails().GetSkiffQuantity() + units)
			delivered = berth.vehicle
		default:
			panic("validated ship purchase has unsupported action")
		}
		result.Quantity += units
		result.Price += int(price)
	}

	details := delivered.GetVehicleDetails()
	result.PendingDelivery = details.IsPendingDelivery()
	result.SkiffCountAfter = details.GetSkiffQuantity()
	result.HullHP = details.GetHullHP()
	result.Direction = details.CurrentDirection()

	return result, nil
}

func (g *GameState) buildShipShopState(
	merchant references.MerchantDefinition,
	buyer *party_state.PlayerCharacter,
	buyerIndex int,
) (ShipShopState, error) {
	offer, ok := g.GameReferences.MerchantReferences.ShipOffer(merchant.ID)
	if !ok {
		return ShipShopState{}, fmt.Errorf("%w: no ship offer for merchant %q", ErrInvalidShipPurchase, merchant.ID)
	}
	berth, err := g.reviewShipBerth(merchant.Location)
	if err != nil {
		return ShipShopState{}, err
	}
	state := ShipShopState{
		Merchant:              merchant,
		PricingCharacterIndex: buyerIndex,
		PricingCharacterName:  buyer.GetNameAsString(),
		PricingIntelligence:   int(buyer.Intelligence),
		Gold:                  g.PartyState.Inventory.Gold.Get(),
		Berth:                 berth.review,
	}
	state.Travel, err = g.reviewHarbourTravel(merchant)
	if err != nil {
		return ShipShopState{}, err
	}
	state.FrigateQuote, err = g.buildShipQuote(state, berth, references.FrigateVehicle, offer.FrigateBasePrice)
	if err != nil {
		return ShipShopState{}, err
	}
	state.SkiffQuote, err = g.buildShipQuote(state, berth, references.SkiffVehicle, offer.SkiffBasePrice)
	if err != nil {
		return ShipShopState{}, err
	}

	return state, nil
}

func (g *GameState) buildShipQuote(
	state ShipShopState,
	berth shipBerthAuthority,
	vehicleType references.VehicleType,
	basePrice int,
) (ShipPurchaseQuote, error) {
	price := CalculateMerchantBuyPrice(basePrice, state.PricingIntelligence)
	if basePrice <= 0 || price <= 0 || price > math.MaxUint16 {
		return ShipPurchaseQuote{}, fmt.Errorf("%w: calculated ship price %d is invalid", ErrInvalidShipPurchase, price)
	}
	action, blockReason := g.shipPurchaseDisposition(vehicleType, berth)
	goldAfterPayment := state.Gold
	affordable := int(state.Gold) >= price
	if affordable {
		goldAfterPayment -= uint16(price)
	}
	quote := ShipPurchaseQuote{
		VehicleType:      vehicleType,
		Action:           action,
		BasePrice:        basePrice,
		Price:            price,
		GoldBefore:       state.Gold,
		GoldAfterPayment: goldAfterPayment,
		Affordable:       affordable,
		BlockReason:      blockReason,
		Berth:            berth.review,
	}
	quote.RequestFingerprint = shipPurchaseFingerprint(state, quote, berth.vehicle)
	quote.ApprovalViews = buildShipPurchaseApprovals(quote)
	if len(quote.ApprovalViews) > 0 {
		quote.Affordable = quote.ApprovalViews[0].Affordable
		quote.GoldAfterPayment = quote.ApprovalViews[0].GoldAfterPayment
	}

	return quote, nil
}

func buildShipPurchaseApprovals(quote ShipPurchaseQuote) []ShipPurchaseApprovalView {
	if quote.BlockReason.IsBlocked() {
		return nil
	}
	maximum := 1
	if quote.Action == ShipPurchaseLoadSkiff {
		remaining := ShipShopMaximumSkiffs - quote.Berth.SkiffCount
		if remaining <= 0 {
			return nil
		}
		affordable := int(quote.GoldBefore) / quote.Price
		maximum = min(remaining, max(1, affordable))
	}
	approvals := make([]ShipPurchaseApprovalView, maximum)
	for index := range approvals {
		quantity := index + 1
		totalPrice := quote.Price * quantity
		affordable := totalPrice <= int(quote.GoldBefore)
		goldAfter := quote.GoldBefore
		if affordable {
			goldAfter -= uint16(totalPrice)
		}
		skiffsAfter := quote.Berth.SkiffCount
		switch quote.Action {
		case ShipPurchaseCreateFrigate:
			skiffsAfter = 2
		case ShipPurchaseCreateStandaloneSkiff:
			skiffsAfter = 0
		case ShipPurchaseLoadSkiff:
			skiffsAfter += quantity
		default:
			panic("buildShipPurchaseApprovals: unknown action")
		}
		approvals[index] = ShipPurchaseApprovalView{
			Quantity:         quantity,
			UnitPrice:        quote.Price,
			TotalPrice:       totalPrice,
			GoldBefore:       quote.GoldBefore,
			GoldAfterPayment: goldAfter,
			Affordable:       affordable,
			SkiffCountBefore: quote.Berth.SkiffCount,
			SkiffCountAfter:  skiffsAfter,
		}
	}

	return approvals
}

func shipPurchaseApproval(quote ShipPurchaseQuote, quantity int) (ShipPurchaseApprovalView, bool) {
	if quantity <= 0 || quantity > len(quote.ApprovalViews) {
		return ShipPurchaseApprovalView{}, false
	}

	return quote.ApprovalViews[quantity-1], true
}

func (g *GameState) shipPurchaseDisposition(
	vehicleType references.VehicleType,
	berth shipBerthAuthority,
) (ShipPurchaseAction, ShipPurchaseBlockReason) {
	// REDUX DEVIATION: original ship-shop rules's occupied-frigate branch quotes 87,000,
	// checks 10,000, and grants nothing. Redux preflights exact berth occupancy
	// and controller capacity, refusing unsafe insertion before payment or theft.
	// See ref:MERCH-0008.
	if vehicleType == references.FrigateVehicle {
		if berth.review.State != ShipBerthEmpty {
			if berth.review.State == ShipBerthPendingFrigate || berth.review.State == ShipBerthMooredFrigate {
				return ShipPurchaseCreateFrigate, ShipPurchaseBlockFrigateAlreadyPresent
			}

			return ShipPurchaseCreateFrigate, ShipPurchaseBlockBerthOccupied
		}
		if g.hasPendingCanonicalDockOrderExcept(berth.review.Location) {
			return ShipPurchaseCreateFrigate, ShipPurchaseBlockPendingOrderElsewhere
		}
		if !berth.review.HasMapCapacity {
			return ShipPurchaseCreateFrigate, ShipPurchaseBlockMapCapacity
		}

		return ShipPurchaseCreateFrigate, ShipPurchaseBlockNone
	}

	switch berth.review.State {
	case ShipBerthEmpty:
		if g.hasPendingCanonicalDockOrderExcept(berth.review.Location) {
			return ShipPurchaseCreateStandaloneSkiff, ShipPurchaseBlockPendingOrderElsewhere
		}
		if !berth.review.HasMapCapacity {
			return ShipPurchaseCreateStandaloneSkiff, ShipPurchaseBlockMapCapacity
		}

		return ShipPurchaseCreateStandaloneSkiff, ShipPurchaseBlockNone
	case ShipBerthPendingFrigate, ShipBerthMooredFrigate:
		// REDUX DEVIATION: original ship-shop rules increment the encoded frigate skiff count
		// without a bound. Redux uses the deliberate 12-skiff gameplay capacity
		// and refuses atomically when the frigate is full.
		// See ref:MERCH-0008.
		if berth.review.SkiffCount >= ShipShopMaximumSkiffs {
			return ShipPurchaseLoadSkiff, ShipPurchaseBlockSkiffLimit
		}
		// REDUX DEVIATION: original ship-shop rules can add only to its pending encoded frigate.
		// Redux also loads a delivered frigate moored at this exact canonical berth.
		// See ref:MERCH-0008.
		return ShipPurchaseLoadSkiff, ShipPurchaseBlockNone
	case ShipBerthPendingSkiff, ShipBerthMooredSkiff:
		return ShipPurchaseCreateStandaloneSkiff, ShipPurchaseBlockStandaloneSkiff
	case ShipBerthObstructed:
		return ShipPurchaseCreateStandaloneSkiff, ShipPurchaseBlockBerthOccupied
	default:
		panic("unknown ship berth state")
	}
}

func (g *GameState) reviewShipBerth(location references.Location) (shipBerthAuthority, error) {
	units, position, err := g.britanniaDockVehicleTarget(location)
	if err != nil {
		return shipBerthAuthority{}, err
	}
	authority := shipBerthAuthority{review: ShipBerthReview{
		Location:       location,
		Position:       position,
		State:          ShipBerthEmpty,
		VehicleType:    references.NoPartyVehicle,
		MapUnitCount:   len(*units),
		HasMapCapacity: units.HasCapacity(),
	}}
	mapUnit := units.GetMapUnitAtPositionAndFloorOrNil(position, references.MainFloor)
	if mapUnit == nil {
		return authority, nil
	}
	vehicle, ok := (*mapUnit).(*map_units.NPCFriendly)
	if !ok || vehicle.NPCReference.GetNPCType() != references.Vehicle {
		authority.review.State = ShipBerthObstructed

		return authority, nil
	}
	authority.vehicle = vehicle
	details := vehicle.GetVehicleDetails()
	authority.review.VehicleType = details.VehicleType
	authority.review.Pending = details.IsPendingDelivery()
	authority.review.SkiffCount = details.GetSkiffQuantity()
	switch details.VehicleType {
	case references.FrigateVehicle:
		if authority.review.Pending {
			authority.review.State = ShipBerthPendingFrigate
		} else {
			authority.review.State = ShipBerthMooredFrigate
		}
	case references.SkiffVehicle:
		if authority.review.Pending {
			authority.review.State = ShipBerthPendingSkiff
		} else {
			authority.review.State = ShipBerthMooredSkiff
		}
	case references.NoPartyVehicle, references.CarpetVehicle, references.HorseVehicle, references.NPC:
		authority.review.State = ShipBerthObstructed
	default:
		panic("unknown vehicle type at canonical berth")
	}

	return authority, nil
}

func (g *GameState) hasPendingCanonicalDockOrderExcept(except references.Location) bool {
	if g.GameReferences == nil || g.GameReferences.DockReferences == nil {
		return false
	}
	controller := g.LargeMapNPCAIController[references.OVERWORLD]
	if controller == nil || controller.GetNpcs() == nil {
		return false
	}
	units := controller.GetNpcs()
	for _, location := range references.GetListOfAllLocationsWithDocks() {
		if location == except {
			continue
		}
		position, ok := g.GameReferences.DockReferences.GetDockPositionChecked(location)
		if !ok {
			continue
		}
		vehicle := units.GetVehicleAtPositionAndFloorOrNil(position, references.MainFloor)
		if vehicle != nil && vehicle.GetVehicleDetails().IsPendingDelivery() {
			return true
		}
	}

	return false
}

func shipQuoteForVehicle(state ShipShopState, vehicleType references.VehicleType) (ShipPurchaseQuote, error) {
	switch vehicleType {
	case references.FrigateVehicle:
		return state.FrigateQuote, nil
	case references.SkiffVehicle:
		return state.SkiffQuote, nil
	case references.NoPartyVehicle, references.CarpetVehicle, references.HorseVehicle, references.NPC:
		return ShipPurchaseQuote{}, fmt.Errorf("%w: unsupported vehicle type %d", ErrInvalidShipPurchase, vehicleType)
	default:
		return ShipPurchaseQuote{}, fmt.Errorf("%w: unsupported vehicle type %d", ErrInvalidShipPurchase, vehicleType)
	}
}

func shipPurchaseFingerprint(
	state ShipShopState,
	quote ShipPurchaseQuote,
	vehicle *map_units.NPCFriendly,
) string {
	return reviewedMutationFingerprint(func(digest hash.Hash) {
		writeFingerprintf(
			digest,
			"merchant=%s;location=%d;buyer=%d;buyerName=%q;int=%d;vehicle=%d;action=%d;base=%d;price=%d;gold=%d;block=%d;dock=%d,%d;state=%d;berthVehicle=%d;pending=%t;skiffs=%d;units=%d;capacity=%t;identity=%p",
			state.Merchant.ID,
			state.Merchant.Location,
			state.PricingCharacterIndex,
			state.PricingCharacterName,
			state.PricingIntelligence,
			quote.VehicleType,
			quote.Action,
			quote.BasePrice,
			quote.Price,
			quote.GoldBefore,
			quote.BlockReason,
			quote.Berth.Position.X,
			quote.Berth.Position.Y,
			quote.Berth.State,
			quote.Berth.VehicleType,
			quote.Berth.Pending,
			quote.Berth.SkiffCount,
			quote.Berth.MapUnitCount,
			quote.Berth.HasMapCapacity,
			vehicle,
		)
	})
}

// HarbourTravelBlockReason identifies why immediate travel to a dealer's berth is unavailable.
type HarbourTravelBlockReason byte

// Harbour travel blocks identify pure preflight failures for workflow display.
const (
	HarbourTravelBlockNone HarbourTravelBlockReason = iota
	HarbourTravelBlockNotOnFoot
	HarbourTravelBlockMapUnavailable
	HarbourTravelBlockNoBerthVehicle
	HarbourTravelBlockNoAdjacentDock
	HarbourTravelBlockArrivalOccupied
)

// IsBlocked reports whether travel cannot currently be performed by its workflow.
func (r HarbourTravelBlockReason) IsBlocked() bool { return r != HarbourTravelBlockNone }

// HarbourTravelReview is a detached, pure transition preflight.
type HarbourTravelReview struct {
	MerchantID      references.MerchantID
	DockLocation    references.Location
	HarbourName     string
	BerthPosition   references.Position
	ArrivalPosition references.Position
	VehicleType     references.VehicleType
	PendingDelivery bool
	TimeCostMinutes int
	BlockReason     HarbourTravelBlockReason
	Fingerprint     string
}

// ReviewHarbourTravel finds an exact cardinal Dock terrain tile beside the
// current dealer's occupied canonical berth without mutating gameplay.
func (g *GameState) ReviewHarbourTravel(session MerchantSession) (HarbourTravelReview, error) {
	_, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyShip)
	if err != nil {
		return HarbourTravelReview{}, err
	}

	return g.reviewHarbourTravel(merchant)
}

// RevalidateHarbourTravel ensures a workflow still owns the exact reviewed travel preflight.
// It deliberately performs no transition and advances no gameplay time.
func (g *GameState) RevalidateHarbourTravel(
	session MerchantSession,
	reviewed HarbourTravelReview,
) error {
	current, err := g.ReviewHarbourTravel(session)
	if err != nil {
		return err
	}
	if reviewed.Fingerprint == "" || reviewed.Fingerprint != current.Fingerprint ||
		current.BlockReason.IsBlocked() {
		return fmt.Errorf(
			"%w: reviewed harbour, berth vehicle, terrain, occupancy, or party vehicle changed",
			ErrHarbourTravelUnavailable,
		)
	}

	return nil
}

// ThenHarbourTravelTransition appends the accepted escort transition to its
// owning workflow after that workflow has ended the merchant session.
func (g *GameState) ThenHarbourTravelTransition(review HarbourTravelReview) event_queue.Sequence {
	if review.Fingerprint == "" || review.BlockReason.IsBlocked() ||
		review.TimeCostMinutes != HarbourTravelTimeCostMinutes {
		panic("ThenHarbourTravelTransition: an available reviewed harbour trip is required")
	}

	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.Animate(EnterFadeOutMs, func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
		}).Then(func() {
			// REDUX DEVIATION: original ship-shop rules records a dock delivery but provides no
			// transport and charges no additional time. Redux adds a three-minute
			// harbourmaster escort so a purchased vessel is immediately reachable.
			// See ref:MERCH-0008.
			g.AdvanceGameplayMinutes(HarbourTravelTimeCostMinutes)
			g.TransitionToLocation(
				references.BritanniaUnderworld,
				references.MainFloor,
				review.ArrivalPosition,
				TransitionFullLocation,
				false,
			)
		}).Animate(EnterFadeInMs, func(progress float32) {
			g.SystemCallbacks.Visual.SetMapFadeLevel(1.0 - float64(progress))
		}).Then(func() {
			g.SystemCallbacks.Visual.SetMapFadeLevel(0)
		})
	}
}

func (g *GameState) reviewHarbourTravel(
	merchant references.MerchantDefinition,
) (HarbourTravelReview, error) {
	berth, err := g.reviewShipBerth(merchant.Location)
	if err != nil {
		return HarbourTravelReview{}, err
	}
	state := HarbourTravelReview{
		MerchantID:      merchant.ID,
		DockLocation:    merchant.Location,
		BerthPosition:   berth.review.Position,
		VehicleType:     berth.review.VehicleType,
		PendingDelivery: berth.review.Pending,
		TimeCostMinutes: HarbourTravelTimeCostMinutes,
	}
	if location := g.GameReferences.LocationReferences.GetLocationReference(merchant.Location); location != nil {
		state.HarbourName = location.FriendlyLocationName + " Harbour"
	}
	switch {
	case g.PartyVehicle.GetVehicleDetails().VehicleType != references.NoPartyVehicle:
		state.BlockReason = HarbourTravelBlockNotOnFoot
	case berth.vehicle == nil ||
		(berth.review.State != ShipBerthPendingFrigate && berth.review.State != ShipBerthPendingSkiff &&
			berth.review.State != ShipBerthMooredFrigate && berth.review.State != ShipBerthMooredSkiff):
		state.BlockReason = HarbourTravelBlockNoBerthVehicle
	default:
		largeMap := g.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, references.MainFloor)
		controller := g.LargeMapNPCAIController[references.OVERWORLD]
		if largeMap == nil || controller == nil || controller.GetNpcs() == nil {
			state.BlockReason = HarbourTravelBlockMapUnavailable
		} else {
			foundDock := false
			foundArrival := false
			for _, offset := range [...]references.Position{
				{X: 0, Y: 1},
				{X: 0, Y: -1},
				{X: 1, Y: 0},
				{X: -1, Y: 0},
			} {
				position := references.Position{X: berth.review.Position.X + offset.X, Y: berth.review.Position.Y + offset.Y}
				if !vehiclePositionInBounds(position, references.XLargeMapTiles, references.YLargeMapTiles) {
					continue
				}
				tile := largeMap.GetTileTopMapOnlyTile(&position)
				if tile == nil || tile.Index != indexes.Dock {
					continue
				}
				foundDock = true
				if controller.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(position, references.MainFloor) != nil {
					continue
				}
				state.ArrivalPosition = position
				state.BlockReason = HarbourTravelBlockNone
				foundArrival = true

				break
			}
			if !foundArrival {
				if foundDock {
					state.BlockReason = HarbourTravelBlockArrivalOccupied
				} else {
					state.BlockReason = HarbourTravelBlockNoAdjacentDock
				}
			}
		}
	}
	state.Fingerprint = harbourTravelFingerprint(state, berth.vehicle, g.PartyVehicle.GetVehicleDetails().VehicleType)

	return state, nil
}

func harbourTravelFingerprint(
	state HarbourTravelReview,
	vehicle *map_units.NPCFriendly,
	partyVehicle references.VehicleType,
) string {
	return reviewedMutationFingerprint(func(digest hash.Hash) {
		writeFingerprintf(
			digest,
			"merchant=%s;location=%d;name=%q;berth=%d,%d;arrival=%d,%d;vehicle=%d;pending=%t;partyVehicle=%d;cost=%d;block=%d;identity=%p",
			state.MerchantID,
			state.DockLocation,
			state.HarbourName,
			state.BerthPosition.X,
			state.BerthPosition.Y,
			state.ArrivalPosition.X,
			state.ArrivalPosition.Y,
			state.VehicleType,
			state.PendingDelivery,
			partyVehicle,
			state.TimeCostMinutes,
			state.BlockReason,
			vehicle,
		)
	})
}
