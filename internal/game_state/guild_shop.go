package game_state

import (
	"errors"
	"fmt"
	"math"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ErrInvalidGuildPurchase indicates an unsupported offer or invalid trusted quote.
var ErrInvalidGuildPurchase = errors.New("invalid guild purchase")

// GuildShopState is a trusted snapshot of one guild session's current offers.
type GuildShopState struct {
	Merchant              references.MerchantDefinition
	PricingCharacterIndex int
	PricingCharacterName  string
	PricingIntelligence   int
	Gold                  uint16
	Offers                [3]GuildOfferQuote
}

// GuildShopReview is a read-only production quote snapshot with the canonical merchant sprite identity.
type GuildShopReview = MerchantShopReview[GuildShopState]

// GuildOfferQuote describes one canonical guild offer against current inventory and gold.
type GuildOfferQuote struct {
	Provision      references.Provision
	BundleQuantity uint16
	BasePrice      int
	PricePerBundle int
	Gold           uint16
	QuantityBefore uint16
	MaxBundles     uint16
}

// GuildPurchaseRequest identifies a positive number of bundles from the trusted catalog.
type GuildPurchaseRequest struct {
	Provision   references.Provision
	BundleCount uint16
}

// GuildPurchaseResult reports one complete authoritative batch purchase.
type GuildPurchaseResult struct {
	Provision       references.Provision
	BasePrice       int
	PricePerBundle  int
	BundleQuantity  uint16
	BundleCount     uint16
	OfferedQuantity uint16
	TotalCost       int
	GoldBefore      uint16
	GoldAfter       uint16
	Theft           uint16
	QuantityBefore  uint16
	QuantityAfter   uint16
	ActualReceipt   uint16
}

// GetGuildShopState returns all three trusted quotes in Keys, Gems, Torches order.
func (g *GameState) GetGuildShopState(session MerchantSession) (GuildShopState, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyGuild)
	if err != nil {
		return GuildShopState{}, err
	}

	return g.buildGuildShopState(merchant, buyer, session.buyerIndex)
}

// ReviewGuildShopByMerchantID builds the same production quotes without
// issuing a session, touching callbacks, or consulting mutable loaded NPC state.
func (g *GameState) ReviewGuildShopByMerchantID(id references.MerchantID) (GuildShopReview, error) {
	return reviewMerchantShop(g, id, references.MerchantFamilyGuild, ErrInvalidGuildPurchase, "a Guild merchant", g.buildGuildShopState)
}

func (g *GameState) buildGuildShopState(
	merchant references.MerchantDefinition,
	buyer *party_state.PlayerCharacter,
	buyerIndex int,
) (GuildShopState, error) {
	offers, ok := g.GameReferences.MerchantReferences.GuildOffers(merchant.ID)
	if !ok {
		return GuildShopState{}, fmt.Errorf(
			"%w: no catalog for merchant %q",
			ErrInvalidGuildPurchase,
			merchant.ID,
		)
	}

	state := GuildShopState{
		Merchant:              merchant,
		PricingCharacterIndex: buyerIndex,
		PricingCharacterName:  buyer.GetNameAsString(),
		PricingIntelligence:   int(buyer.Intelligence),
		Gold:                  g.PartyState.Inventory.Gold.Get(),
	}
	for index, offer := range offers {
		quote, quoteErr := g.quoteGuildOffer(offer, buyer)
		if quoteErr != nil {
			return GuildShopState{}, quoteErr
		}
		state.Offers[index] = quote
	}

	return state, nil
}

// QuoteGuildOffer returns one trusted quote selected only by provision identity.
func (g *GameState) QuoteGuildOffer(
	session MerchantSession,
	provision references.Provision,
) (GuildOfferQuote, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyGuild)
	if err != nil {
		return GuildOfferQuote{}, err
	}
	offer, err := g.guildOffer(merchant.ID, provision)
	if err != nil {
		return GuildOfferQuote{}, err
	}

	return g.quoteGuildOffer(offer, buyer)
}

// PurchaseGuildOffer validates and commits one canonical batch purchase.
func (g *GameState) PurchaseGuildOffer(
	session MerchantSession,
	request GuildPurchaseRequest,
) (GuildPurchaseResult, error) {
	quote, err := g.QuoteGuildOffer(session, request.Provision)
	if err != nil {
		return GuildPurchaseResult{}, err
	}
	if request.BundleCount == 0 || request.BundleCount > quote.MaxBundles {
		return GuildPurchaseResult{}, fmt.Errorf(
			"%w: bundle count %d must be positive and not exceed current maximum %d",
			ErrInvalidGuildPurchase,
			request.BundleCount,
			quote.MaxBundles,
		)
	}

	offeredQuantity := quote.BundleQuantity * request.BundleCount
	totalCost := quote.PricePerBundle * int(request.BundleCount)
	result := GuildPurchaseResult{
		Provision:       quote.Provision,
		BasePrice:       quote.BasePrice,
		PricePerBundle:  quote.PricePerBundle,
		BundleQuantity:  quote.BundleQuantity,
		BundleCount:     request.BundleCount,
		OfferedQuantity: offeredQuantity,
		TotalCost:       totalCost,
		GoldBefore:      quote.Gold,
		GoldAfter:       quote.Gold,
		Theft:           0,
		QuantityBefore:  quote.QuantityBefore,
		QuantityAfter:   quote.QuantityBefore,
		ActualReceipt:   0,
	}

	// REDUX DEVIATION: Guild bundle purchase pays and steals per fixed bundle.
	// The default option batches those boundaries for compact keyboard shopping.
	// See ref:MERCH-0006.
	bundles := request.BundleCount
	if !g.GameConfig.GameplayConfiguration.BatchMerchantTheft {
		bundles = 1
	}
	quantity := g.guildProvisionQuantity(quote.Provision)
	result.BundleCount = 0
	result.TotalCost = 0
	for result.BundleCount < request.BundleCount {
		price := uint16(quote.PricePerBundle * int(bundles))
		if !g.PartyState.Inventory.Gold.DecrementBy(price) {
			break
		}
		result.Theft += g.ApplyFaulineiShopTheft()
		quantity.IncrementBy(quote.BundleQuantity * bundles)
		result.BundleCount += bundles
		result.TotalCost += int(price)
	}
	result.OfferedQuantity = quote.BundleQuantity * result.BundleCount
	result.GoldAfter = g.PartyState.Inventory.Gold.Get()
	result.QuantityAfter = quantity.Get()
	result.ActualReceipt = result.QuantityAfter - result.QuantityBefore

	return result, nil
}

func (g *GameState) guildOffer(
	merchantID references.MerchantID,
	provision references.Provision,
) (references.GuildOffer, error) {
	offers, ok := g.GameReferences.MerchantReferences.GuildOffers(merchantID)
	if !ok {
		return references.GuildOffer{}, fmt.Errorf(
			"%w: no catalog for merchant %q",
			ErrInvalidGuildPurchase,
			merchantID,
		)
	}
	for _, offer := range offers {
		if offer.Provision == provision {
			return offer, nil
		}
	}

	return references.GuildOffer{}, fmt.Errorf(
		"%w: unsupported guild provision %d",
		ErrInvalidGuildPurchase,
		provision,
	)
}

func (g *GameState) quoteGuildOffer(
	offer references.GuildOffer,
	buyer *party_state.PlayerCharacter,
) (GuildOfferQuote, error) {
	chargedPrice := CalculateMerchantBuyPrice(offer.BasePrice, int(buyer.Intelligence))
	if chargedPrice <= 0 || chargedPrice > math.MaxUint16 {
		return GuildOfferQuote{}, fmt.Errorf(
			"%w: calculated price %d is outside supported gold range",
			ErrInvalidGuildPurchase,
			chargedPrice,
		)
	}
	if offer.BundleQuantity == 0 {
		return GuildOfferQuote{}, fmt.Errorf(
			"%w: bundle quantity must be positive",
			ErrInvalidGuildPurchase,
		)
	}
	quantity := g.guildProvisionQuantity(offer.Provision)
	quantityBefore := quantity.Get()
	gold := g.PartyState.Inventory.Gold.Get()
	usefulBundles := 0
	if quantityBefore < party_state.ItemQuantitySmallMax {
		remaining := int(party_state.ItemQuantitySmallMax - quantityBefore)
		usefulBundles = (remaining + int(offer.BundleQuantity) - 1) / int(offer.BundleQuantity)
	}
	affordableBundles := int(gold) / chargedPrice

	return GuildOfferQuote{
		Provision:      offer.Provision,
		BundleQuantity: offer.BundleQuantity,
		BasePrice:      offer.BasePrice,
		PricePerBundle: chargedPrice,
		Gold:           gold,
		QuantityBefore: quantityBefore,
		MaxBundles:     uint16(min(usefulBundles, affordableBundles)),
	}, nil
}

func (g *GameState) guildProvisionQuantity(provision references.Provision) *party_state.ItemQuantitySmall {
	switch provision {
	case references.Key:
		return &g.PartyState.Inventory.Provisions.Keys
	case references.Gem:
		return &g.PartyState.Inventory.Provisions.Gems
	case references.Torches:
		return &g.PartyState.Inventory.Provisions.Torches
	case references.NoProvision, references.SkullKeys, references.Food, references.Gold:
		panic("unsupported guild provision reached inventory mutation")
	}

	panic("unknown guild provision reached inventory mutation")
}
