package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ErrInvalidReagentPurchase indicates an unsupported offer or invalid current bundle count.
var ErrInvalidReagentPurchase = errors.New("invalid reagent purchase")

// ReagentOfferBlockReason identifies why an offer cannot currently be purchased.
type ReagentOfferBlockReason byte

const (
	// ReagentOfferBlockNone indicates that at least one bundle can be purchased.
	ReagentOfferBlockNone ReagentOfferBlockReason = iota
	// ReagentOfferBlockInventoryFull indicates that the reagent quantity is already 99.
	ReagentOfferBlockInventoryFull
	// ReagentOfferBlockUnaffordable indicates that current gold cannot buy one bundle.
	ReagentOfferBlockUnaffordable
)

// IsBlocked reports whether the current offer cannot produce a purchase request.
func (r ReagentOfferBlockReason) IsBlocked() bool { return r != ReagentOfferBlockNone }

// ReagentShopState is a trusted detached snapshot of one reagent-shop session.
type ReagentShopState struct {
	Merchant              references.MerchantDefinition
	PricingCharacterIndex int
	PricingCharacterName  string
	PricingIntelligence   int
	Gold                  uint16
	Offers                []ReagentOfferQuote
}

// ReagentShopReview is a read-only production snapshot with canonical sprite identity.
type ReagentShopReview struct {
	State               ReagentShopState
	MerchantSpriteIndex indexes.SpriteIndex
}

// ReagentOfferQuote describes one canonical bundle against current inventory and gold.
type ReagentOfferQuote struct {
	Reagent           references.Reagent
	ReagentName       string
	BundleQuantity    uint16
	BasePrice         int
	PricePerBundle    int
	Gold              uint16
	QuantityBefore    uint16
	UsefulBundles     uint16
	AffordableBundles uint16
	MaxBundles        uint16
	Full              bool
	BlockReason       ReagentOfferBlockReason
	ApprovalViews     []ReagentPurchaseApprovalView
}

// ReagentPurchaseApprovalView is one complete projection for a selectable bundle count.
type ReagentPurchaseApprovalView struct {
	Reagent           references.Reagent
	ReagentName       string
	BundleQuantity    uint16
	BundleCount       uint16
	MaxBundles        uint16
	RequestedQuantity uint16
	QuantityBefore    uint16
	QuantityAfter     uint16
	ActualReceipt     uint16
	PricePerBundle    int
	TotalPrice        int
	GoldBefore        uint16
	GoldAfterPayment  uint16
	PaidOverflow      bool
	OverflowQuantity  uint16
}

// ReagentPurchaseRequest identifies one positive bundle count by reagent identity only.
type ReagentPurchaseRequest struct {
	Reagent     references.Reagent
	BundleCount uint16
}

// ReagentPurchaseResult reports completed bundles, actual charges and receipts.
// RequestedQuantity retains the original request, even if theft stops the batch.
type ReagentPurchaseResult struct {
	Reagent           references.Reagent
	ReagentName       string
	BundleQuantity    uint16
	BundleCount       uint16
	RequestedQuantity uint16
	QuantityBefore    uint16
	QuantityAfter     uint16
	ActualReceipt     uint16
	PricePerBundle    int
	TotalPrice        int
	GoldBefore        uint16
	GoldAfterPayment  uint16
	Theft             uint16
	GoldAfterTheft    uint16
}

// GetReagentShopState returns all canonical offers with current prices and projections.
func (g *GameState) GetReagentShopState(session MerchantSession) (ReagentShopState, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyReagent)
	if err != nil {
		return ReagentShopState{}, err
	}

	return g.buildReagentShopState(merchant, buyer, session.buyerIndex)
}

// ReviewReagentShopByMerchantID builds production quotes without issuing a
// session, invoking callbacks, consuming RNG, or consulting loaded NPC state.
func (g *GameState) ReviewReagentShopByMerchantID(id references.MerchantID) (ReagentShopReview, error) {
	merchant, npcReference, err := g.canonicalMerchant(id, references.MerchantFamilyReagent, ErrInvalidReagentPurchase, "a reagent merchant")
	if err != nil {
		return ReagentShopReview{}, err
	}
	buyer, buyerIndex := g.merchantPricingCharacter()
	if buyer == nil {
		return ReagentShopReview{}, ErrMerchantBuyerUnavailable
	}
	state, err := g.buildReagentShopState(merchant, buyer, buyerIndex)
	if err != nil {
		return ReagentShopReview{}, err
	}

	return ReagentShopReview{State: state, MerchantSpriteIndex: npcReference.GetSpriteIndex()}, nil
}

func (g *GameState) buildReagentShopState(
	merchant references.MerchantDefinition,
	buyer *party_state.PlayerCharacter,
	buyerIndex int,
) (ReagentShopState, error) {
	offers, ok := g.GameReferences.MerchantReferences.ReagentOffers(merchant.ID)
	if !ok {
		return ReagentShopState{}, fmt.Errorf(
			"%w: no reagent catalog for merchant %q",
			ErrInvalidReagentPurchase,
			merchant.ID,
		)
	}

	state := ReagentShopState{
		Merchant:              merchant,
		PricingCharacterIndex: buyerIndex,
		PricingCharacterName:  buyer.GetNameAsString(),
		PricingIntelligence:   int(buyer.Intelligence),
		Gold:                  g.PartyState.Inventory.Gold.Get(),
		Offers:                make([]ReagentOfferQuote, len(offers)),
	}
	for index, offer := range offers {
		quote, err := g.quoteReagentOffer(offer, buyer)
		if err != nil {
			return ReagentShopState{}, err
		}
		state.Offers[index] = quote
	}

	return state, nil
}

// QuoteReagentOffer returns one current trusted quote selected only by reagent identity.
func (g *GameState) QuoteReagentOffer(
	session MerchantSession,
	reagent references.Reagent,
) (ReagentOfferQuote, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyReagent)
	if err != nil {
		return ReagentOfferQuote{}, err
	}
	offer, err := g.reagentOffer(merchant.ID, reagent)
	if err != nil {
		return ReagentOfferQuote{}, err
	}

	return g.quoteReagentOffer(offer, buyer)
}

// PurchaseReagentOffer re-quotes and commits one canonical aggregate bundle purchase.
func (g *GameState) PurchaseReagentOffer(
	session MerchantSession,
	request ReagentPurchaseRequest,
) (ReagentPurchaseResult, error) {
	quote, err := g.QuoteReagentOffer(session, request.Reagent)
	if err != nil {
		return ReagentPurchaseResult{}, err
	}
	if request.BundleCount == 0 || request.BundleCount > quote.MaxBundles {
		return ReagentPurchaseResult{}, fmt.Errorf(
			"%w: bundle count %d must be positive and not exceed current maximum %d (block reason %d)",
			ErrInvalidReagentPurchase,
			request.BundleCount,
			quote.MaxBundles,
			quote.BlockReason,
		)
	}
	approval := quote.ApprovalViews[request.BundleCount-1]
	result := ReagentPurchaseResult{
		Reagent:           quote.Reagent,
		ReagentName:       quote.ReagentName,
		BundleQuantity:    quote.BundleQuantity,
		BundleCount:       request.BundleCount,
		RequestedQuantity: approval.RequestedQuantity,
		QuantityBefore:    approval.QuantityBefore,
		QuantityAfter:     approval.QuantityBefore,
		ActualReceipt:     0,
		PricePerBundle:    quote.PricePerBundle,
		TotalPrice:        approval.TotalPrice,
		GoldBefore:        approval.GoldBefore,
		GoldAfterPayment:  approval.GoldBefore,
		Theft:             0,
		GoldAfterTheft:    approval.GoldBefore,
	}

	// REDUX DEVIATION: buy reagent pays and steals per bundle, not
	// ingredient. The default option batches those boundaries for convenience.
	// See ref:MERCH-0005.
	bundles := request.BundleCount
	if !g.GameConfig.GameplayConfiguration.BatchMerchantTheft {
		bundles = 1
	}
	result.BundleCount = 0
	result.TotalPrice = 0
	for result.BundleCount < request.BundleCount {
		price := uint16(quote.PricePerBundle * int(bundles)) //nolint:gosec // Initial affordability bounds every payment.
		if !g.PartyState.Inventory.Gold.DecrementBy(price) {
			break
		}
		result.GoldAfterPayment = g.PartyState.Inventory.Gold.Get()
		result.Theft += g.ApplyFaulineiShopTheft()
		g.PartyState.Inventory.Reagent.IncrementBy(quote.Reagent, quote.BundleQuantity*bundles)
		result.BundleCount += bundles
		result.TotalPrice += int(price)
	}
	result.GoldAfterTheft = g.PartyState.Inventory.Gold.Get()
	result.QuantityAfter = g.PartyState.Inventory.Reagent.Get(quote.Reagent)
	result.ActualReceipt = result.QuantityAfter - result.QuantityBefore

	return result, nil
}

func (g *GameState) reagentOffer(
	merchantID references.MerchantID,
	reagent references.Reagent,
) (references.ReagentOffer, error) {
	offers, ok := g.GameReferences.MerchantReferences.ReagentOffers(merchantID)
	if !ok {
		return references.ReagentOffer{}, fmt.Errorf(
			"%w: no reagent catalog for merchant %q",
			ErrInvalidReagentPurchase,
			merchantID,
		)
	}
	for _, offer := range offers {
		if offer.Reagent == reagent {
			return offer, nil
		}
	}

	return references.ReagentOffer{}, fmt.Errorf(
		"%w: reagent %d is not sold by merchant %q",
		ErrInvalidReagentPurchase,
		reagent,
		merchantID,
	)
}

func (g *GameState) quoteReagentOffer(
	offer references.ReagentOffer,
	buyer *party_state.PlayerCharacter,
) (ReagentOfferQuote, error) {
	if g.GameReferences.InventoryItemReferences == nil {
		return ReagentOfferQuote{}, fmt.Errorf("%w: inventory item references are unavailable", ErrInvalidReagentPurchase)
	}
	item, ok := g.GameReferences.InventoryItemReferences.Reagent[offer.Reagent]
	if !ok || item.ItemName == "" {
		return ReagentOfferQuote{}, fmt.Errorf(
			"%w: reagent %d has no canonical inventory name",
			ErrInvalidReagentPurchase,
			offer.Reagent,
		)
	}
	if offer.BundleQuantity == 0 {
		return ReagentOfferQuote{}, fmt.Errorf("%w: bundle quantity must be positive", ErrInvalidReagentPurchase)
	}
	pricePerBundle := CalculateMerchantBuyPrice(offer.BasePrice, int(buyer.Intelligence))
	if offer.BasePrice <= 0 || pricePerBundle <= 0 {
		return ReagentOfferQuote{}, fmt.Errorf(
			"%w: calculated price %d is invalid for reagent %d",
			ErrInvalidReagentPurchase,
			pricePerBundle,
			offer.Reagent,
		)
	}

	quantityBefore := g.PartyState.Inventory.Reagent.Get(offer.Reagent)
	gold := g.PartyState.Inventory.Gold.Get()
	usefulBundles := 0
	if quantityBefore < party_state.ItemQuantitySmallMax {
		remaining := int(party_state.ItemQuantitySmallMax - quantityBefore)
		usefulBundles = (remaining + int(offer.BundleQuantity) - 1) / int(offer.BundleQuantity)
	}
	affordableBundles := int(gold) / pricePerBundle
	maxBundles := min(usefulBundles, affordableBundles)
	blockReason := ReagentOfferBlockNone
	switch {
	case quantityBefore == party_state.ItemQuantitySmallMax:
		blockReason = ReagentOfferBlockInventoryFull
	case affordableBundles == 0:
		// REDUX DEVIATION: buy reagent discovers insufficient gold only after one bundle is confirmed and then ends the buying loop; Redux proactively exposes an affordability-bounded maximum and blocks the request while keeping the shop reviewable.
		// See ref:MERCH-0005.
		blockReason = ReagentOfferBlockUnaffordable
	}

	quote := ReagentOfferQuote{
		Reagent:           offer.Reagent,
		ReagentName:       item.ItemName,
		BundleQuantity:    offer.BundleQuantity,
		BasePrice:         offer.BasePrice,
		PricePerBundle:    pricePerBundle,
		Gold:              gold,
		QuantityBefore:    quantityBefore,
		UsefulBundles:     uint16(usefulBundles),     //nolint:gosec // Capacity to 99 limits useful bundles to 99.
		AffordableBundles: uint16(affordableBundles), //nolint:gosec // Gold and positive source prices bound this below uint16.
		MaxBundles:        uint16(maxBundles),        //nolint:gosec // The useful bound limits the final maximum to 99.
		Full:              quantityBefore == party_state.ItemQuantitySmallMax,
		BlockReason:       blockReason,
		ApprovalViews:     make([]ReagentPurchaseApprovalView, maxBundles),
	}
	for index := range maxBundles {
		bundleCount := uint16(index + 1) //nolint:gosec // MaxBundles is at most 99.
		requestedQuantity := offer.BundleQuantity * bundleCount
		quantityAfter := min(
			party_state.ItemQuantitySmallMax,
			quantityBefore+requestedQuantity,
		)
		totalPrice := pricePerBundle * int(bundleCount)
		actualReceipt := quantityAfter - quantityBefore
		quote.ApprovalViews[index] = ReagentPurchaseApprovalView{
			Reagent:           offer.Reagent,
			ReagentName:       item.ItemName,
			BundleQuantity:    offer.BundleQuantity,
			BundleCount:       bundleCount,
			MaxBundles:        quote.MaxBundles,
			RequestedQuantity: requestedQuantity,
			QuantityBefore:    quantityBefore,
			QuantityAfter:     quantityAfter,
			ActualReceipt:     actualReceipt,
			PricePerBundle:    pricePerBundle,
			TotalPrice:        totalPrice,
			GoldBefore:        gold,
			GoldAfterPayment:  gold - uint16(totalPrice), //nolint:gosec // Bundle count is bounded by affordability.
			PaidOverflow:      actualReceipt < requestedQuantity,
			OverflowQuantity:  requestedQuantity - actualReceipt,
		}
	}

	return quote, nil
}
