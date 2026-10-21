package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var (
	// ErrInvalidArmsRequest indicates stale quantity or equipment identity input.
	ErrInvalidArmsRequest = errors.New("invalid arms request")
	// ErrArmsCapacity indicates that carried equipment capacity blocks the purchase.
	ErrArmsCapacity = errors.New("arms equipment capacity reached")
	// ErrArmsInsufficientFunds indicates that current gold cannot cover the purchase.
	ErrArmsInsufficientFunds = errors.New("insufficient funds for arms purchase")
	// ErrArmsUnsellableAmmunition indicates that used ammunition cannot be sold.
	ErrArmsUnsellableAmmunition = errors.New("ammunition cannot be sold")
	// ErrArmsZeroPrice indicates that a zero-base-price special item cannot be sold.
	ErrArmsZeroPrice = errors.New("zero-price equipment cannot be sold")
	// ErrArmsGoldCapacity indicates that no requested sale can fit below 9999 gold.
	ErrArmsGoldCapacity = errors.New("gold capacity prevents arms sale")
)

// ArmsShopState is a trusted snapshot of one Arms session's current quotes.
type ArmsShopState struct {
	Merchant              references.MerchantDefinition
	PricingCharacterIndex int
	PricingCharacterName  string
	PricingIntelligence   int
	Gold                  uint16
	BuyQuotes             [7]ArmsBuyQuote
	SellQuotes            []ArmsSellQuote
}

// ArmsShopReview is a read-only production snapshot with canonical sprite identity.
type ArmsShopReview struct {
	State               ArmsShopState
	MerchantSpriteIndex indexes.SpriteIndex
}

// ArmsBuyQuote describes one dealer-stock item against current gold and capacity.
type ArmsBuyQuote struct {
	Equipment       references.Equipment
	BasePrice       int
	UnitPrice       int
	Gold            uint16
	QuantityBefore  uint16
	ReadiedQuantity uint16 `exhaustruct:"optional"`
	MaxQuantity     uint16
	AmmoRefill      bool
}

// ArmsSellQuote describes one currently carried equipment stack.
type ArmsSellQuote struct {
	Equipment       references.Equipment
	BasePrice       int
	UnitOffer       int
	Gold            uint16
	QuantityBefore  uint16
	ReadiedQuantity uint16 `exhaustruct:"optional"`
	MaxQuantity     uint16
	Sellable        bool
}

// ArmsBuyRequest identifies one positive dealer-stock quantity.
type ArmsBuyRequest struct {
	Equipment references.Equipment
	Quantity  uint16
}

// ArmsSellRequest identifies one positive carried quantity.
type ArmsSellRequest struct {
	Equipment references.Equipment
	Quantity  uint16
}

// ArmsBuyResult reports completed purchase quantity and actual charges, including partial batches.
type ArmsBuyResult struct {
	Equipment        references.Equipment
	BasePrice        int
	UnitPrice        int
	Quantity         uint16
	TotalCost        int
	GoldBefore       uint16
	GoldAfterPayment uint16
	Theft            uint16
	GoldAfterTheft   uint16
	QuantityBefore   uint16
	QuantityAfter    uint16
	ActualReceipt    uint16
	AmmoRefill       bool
}

// ArmsSellResult reports one complete authoritative Arms sale.
type ArmsSellResult struct {
	Equipment      references.Equipment
	BasePrice      int
	UnitOffer      int
	Quantity       uint16
	TotalOffer     int
	GoldBefore     uint16
	GoldAfter      uint16
	QuantityBefore uint16
	QuantityAfter  uint16
}

// CalculateArmsSellPrice applies the original Arms Intelligence modifier with
// the exact original shop rules integer operation ordering.
// See ref:MERCH-0004.
func CalculateArmsSellPrice(basePrice, intelligence int) int {
	return basePrice*(3*intelligence)/100 + 1
}

// GetArmsShopState returns current canonical buy and carried-inventory sell quotes.
func (g *GameState) GetArmsShopState(session MerchantSession) (ArmsShopState, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyArms)
	if err != nil {
		return ArmsShopState{}, err
	}

	return g.buildArmsShopState(merchant, buyer, session.buyerIndex)
}

// ReviewArmsShopByMerchantID builds production quotes without issuing a
// session, invoking callbacks, consuming RNG, or consulting loaded NPC state.
func (g *GameState) ReviewArmsShopByMerchantID(id references.MerchantID) (ArmsShopReview, error) {
	merchant, npcReference, err := g.canonicalMerchant(id, references.MerchantFamilyArms, ErrInvalidArmsRequest, "an Arms merchant")
	if err != nil {
		return ArmsShopReview{}, err
	}
	buyer, buyerIndex := g.merchantPricingCharacter()
	if buyer == nil {
		return ArmsShopReview{}, ErrMerchantBuyerUnavailable
	}
	state, err := g.buildArmsShopState(merchant, buyer, buyerIndex)
	if err != nil {
		return ArmsShopReview{}, err
	}

	return ArmsShopReview{State: state, MerchantSpriteIndex: npcReference.GetSpriteIndex()}, nil
}

func (g *GameState) buildArmsShopState(
	merchant references.MerchantDefinition,
	buyer *party_state.PlayerCharacter,
	buyerIndex int,
) (ArmsShopState, error) {
	stock, ok := g.GameReferences.MerchantReferences.ArmsStock(merchant.ID)
	if !ok {
		return ArmsShopState{}, fmt.Errorf("%w: no Arms stock for merchant %q", ErrInvalidArmsRequest, merchant.ID)
	}
	if len(stock.Items) != 7 {
		return ArmsShopState{}, fmt.Errorf("%w: merchant %q stock has %d rows", ErrInvalidArmsRequest, merchant.ID, len(stock.Items))
	}

	state := ArmsShopState{
		Merchant:              merchant,
		PricingCharacterIndex: buyerIndex,
		PricingCharacterName:  buyer.GetNameAsString(),
		PricingIntelligence:   int(buyer.Intelligence),
		Gold:                  g.PartyState.Inventory.Gold.Get(),
		SellQuotes:            make([]ArmsSellQuote, 0),
	}
	for index, equipment := range stock.Items {
		quote, err := g.quoteArmsBuy(merchant, buyer, equipment)
		if err != nil {
			return ArmsShopState{}, err
		}
		state.BuyQuotes[index] = quote
	}
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		if !g.PartyState.Inventory.Equipment.HasSome(equipment) {
			continue
		}
		quote, err := g.quoteArmsSell(buyer, equipment)
		if err != nil {
			return ArmsShopState{}, err
		}
		state.SellQuotes = append(state.SellQuotes, quote)
	}

	return state, nil
}

// QuoteArmsBuy returns a current trusted quote for one dealer-stock item.
func (g *GameState) QuoteArmsBuy(
	session MerchantSession,
	equipment references.Equipment,
) (ArmsBuyQuote, error) {
	buyer, merchant, err := g.validateMerchantSession(session, references.MerchantFamilyArms)
	if err != nil {
		return ArmsBuyQuote{}, err
	}

	return g.quoteArmsBuy(merchant, buyer, equipment)
}

// QuoteArmsSell returns a current trusted quote for one carried equipment stack.
func (g *GameState) QuoteArmsSell(
	session MerchantSession,
	equipment references.Equipment,
) (ArmsSellQuote, error) {
	buyer, _, err := g.validateMerchantSession(session, references.MerchantFamilyArms)
	if err != nil {
		return ArmsSellQuote{}, err
	}

	return g.quoteArmsSell(buyer, equipment)
}

// BuyArmsEquipment validates and commits one canonical Arms purchase.
func (g *GameState) BuyArmsEquipment(
	session MerchantSession,
	request ArmsBuyRequest,
) (ArmsBuyResult, error) {
	quote, err := g.QuoteArmsBuy(session, request.Equipment)
	if err != nil {
		return ArmsBuyResult{}, err
	}
	if request.Quantity == 0 {
		return ArmsBuyResult{}, fmt.Errorf("%w: purchase quantity must be positive", ErrInvalidArmsRequest)
	}
	if quote.AmmoRefill && request.Quantity != 1 {
		return ArmsBuyResult{}, fmt.Errorf("%w: ammunition refill quantity must be one", ErrInvalidArmsRequest)
	}

	capacity := party_state.ItemQuantitySmallMax - quote.QuantityBefore
	if request.Quantity > capacity {
		return ArmsBuyResult{}, fmt.Errorf(
			"%w: requested %d with capacity %d",
			ErrArmsCapacity,
			request.Quantity,
			capacity,
		)
	}
	totalCost := quote.UnitPrice * int(request.Quantity)
	if totalCost > int(quote.Gold) {
		return ArmsBuyResult{}, fmt.Errorf(
			"%w: need %d gold, have %d",
			ErrArmsInsufficientFunds,
			totalCost,
			quote.Gold,
		)
	}

	result := ArmsBuyResult{
		Equipment:        quote.Equipment,
		BasePrice:        quote.BasePrice,
		UnitPrice:        quote.UnitPrice,
		Quantity:         request.Quantity,
		TotalCost:        totalCost,
		GoldBefore:       quote.Gold,
		GoldAfterPayment: quote.Gold,
		GoldAfterTheft:   quote.Gold,
		QuantityBefore:   quote.QuantityBefore,
		QuantityAfter:    quote.QuantityBefore,
		AmmoRefill:       quote.AmmoRefill,
	}

	// REDUX DEVIATION: arms purchase pays and steals per ordinary item.
	// The default batching option keeps one payment/theft for keyboard convenience.
	// See ref:MERCH-0004.
	units := request.Quantity
	if !g.GameConfig.GameplayConfiguration.BatchMerchantTheft {
		units = 1
	}
	result.Quantity = 0
	result.TotalCost = 0
	for result.Quantity < request.Quantity {
		price := uint16(quote.UnitPrice * int(units))
		if !g.PartyState.Inventory.Gold.DecrementBy(price) {
			break // Earlier theft can make the next original item unaffordable.
		}
		result.GoldAfterPayment = g.PartyState.Inventory.Gold.Get()
		result.Theft += g.ApplyFaulineiShopTheft()
		if quote.AmmoRefill {
			g.PartyState.Inventory.Equipment.Set(quote.Equipment, party_state.ItemQuantitySmallMax)
		} else {
			g.PartyState.Inventory.Equipment.IncrementBy(quote.Equipment, units)
		}
		result.Quantity += units
		result.TotalCost += int(price)
	}
	result.GoldAfterTheft = g.PartyState.Inventory.Gold.Get()
	result.QuantityAfter = g.PartyState.Inventory.Equipment.Get(quote.Equipment)
	result.ActualReceipt = result.QuantityAfter - result.QuantityBefore

	return result, nil
}

// SellArmsEquipment validates and commits one gold-safe carried-equipment sale.
func (g *GameState) SellArmsEquipment(
	session MerchantSession,
	request ArmsSellRequest,
) (ArmsSellResult, error) {
	quote, err := g.QuoteArmsSell(session, request.Equipment)
	if err != nil {
		return ArmsSellResult{}, err
	}
	if request.Quantity == 0 {
		return ArmsSellResult{}, fmt.Errorf("%w: sale quantity must be positive", ErrInvalidArmsRequest)
	}
	if request.Equipment.IsAmmo() {
		return ArmsSellResult{}, fmt.Errorf("%w: %s", ErrArmsUnsellableAmmunition, request.Equipment)
	}
	if quote.BasePrice == 0 {
		return ArmsSellResult{}, fmt.Errorf("%w: %s", ErrArmsZeroPrice, request.Equipment)
	}
	if request.Quantity > quote.QuantityBefore {
		return ArmsSellResult{}, fmt.Errorf(
			"%w: requested %d but only %d carried",
			ErrInvalidArmsRequest,
			request.Quantity,
			quote.QuantityBefore,
		)
	}
	if quote.MaxQuantity == 0 || request.Quantity > quote.MaxQuantity {
		return ArmsSellResult{}, fmt.Errorf(
			"%w: requested %d with gold-safe maximum %d",
			ErrArmsGoldCapacity,
			request.Quantity,
			quote.MaxQuantity,
		)
	}

	totalOffer := quote.UnitOffer * int(request.Quantity)
	result := ArmsSellResult{
		Equipment:      quote.Equipment,
		BasePrice:      quote.BasePrice,
		UnitOffer:      quote.UnitOffer,
		Quantity:       request.Quantity,
		TotalOffer:     totalOffer,
		GoldBefore:     quote.Gold,
		GoldAfter:      quote.Gold,
		QuantityBefore: quote.QuantityBefore,
		QuantityAfter:  quote.QuantityBefore,
	}

	// REDUX DEVIATION: original shop rules sells one item per confirmation and saturates gold before removing it, even when the gain is partial or zero; Redux batches only through the exact gold-safe cap so one aggregate credit and removal are atomic and never discard value.
	// See ref:MERCH-0004.
	g.PartyState.Inventory.Gold.IncrementBy(uint16(totalOffer))
	if !g.PartyState.Inventory.Equipment.DecrementBy(quote.Equipment, request.Quantity) {
		panic("validated Arms sale could not remove equipment")
	}
	result.GoldAfter = g.PartyState.Inventory.Gold.Get()
	result.QuantityAfter = g.PartyState.Inventory.Equipment.Get(quote.Equipment)

	return result, nil
}

func (g *GameState) quoteArmsBuy(
	merchant references.MerchantDefinition,
	buyer *party_state.PlayerCharacter,
	equipment references.Equipment,
) (ArmsBuyQuote, error) {
	stock, ok := g.GameReferences.MerchantReferences.ArmsStock(merchant.ID)
	if !ok {
		return ArmsBuyQuote{}, fmt.Errorf("%w: no Arms stock for merchant %q", ErrInvalidArmsRequest, merchant.ID)
	}
	inStock := false
	for _, candidate := range stock.Items {
		if candidate == equipment {
			inStock = true

			break
		}
	}
	if !inStock {
		return ArmsBuyQuote{}, fmt.Errorf(
			"%w: %s is not sold by merchant %q",
			ErrInvalidArmsRequest,
			equipment,
			merchant.ID,
		)
	}

	basePrice := equipment.GetBasePrice()
	unitPrice := CalculateMerchantBuyPrice(basePrice, int(buyer.Intelligence))
	if basePrice <= 0 || unitPrice <= 0 {
		return ArmsBuyQuote{}, fmt.Errorf("%w: invalid purchase price for %s", ErrInvalidArmsRequest, equipment)
	}
	quantityBefore := g.PartyState.Inventory.Equipment.Get(equipment)
	gold := g.PartyState.Inventory.Gold.Get()
	capacity := int(party_state.ItemQuantitySmallMax - quantityBefore)
	affordable := int(gold) / unitPrice
	maxQuantity := uint16(min(capacity, affordable))
	if equipment.IsAmmo() && maxQuantity > 0 {
		maxQuantity = 1
	}

	return ArmsBuyQuote{
		Equipment:       equipment,
		BasePrice:       basePrice,
		UnitPrice:       unitPrice,
		Gold:            gold,
		QuantityBefore:  quantityBefore,
		ReadiedQuantity: g.readiedArmsQuantity(equipment),
		MaxQuantity:     maxQuantity,
		AmmoRefill:      equipment.IsAmmo(),
	}, nil
}

func (g *GameState) quoteArmsSell(
	buyer *party_state.PlayerCharacter,
	equipment references.Equipment,
) (ArmsSellQuote, error) {
	if equipment < references.LeatherHelm || equipment > references.Ankh {
		return ArmsSellQuote{}, fmt.Errorf("%w: unknown equipment ID %d", ErrInvalidArmsRequest, equipment)
	}
	quantityBefore := g.PartyState.Inventory.Equipment.Get(equipment)
	if quantityBefore == 0 {
		return ArmsSellQuote{}, fmt.Errorf("%w: %s is not carried", ErrInvalidArmsRequest, equipment)
	}

	basePrice := equipment.GetBasePrice()
	gold := g.PartyState.Inventory.Gold.Get()
	quote := ArmsSellQuote{
		Equipment:       equipment,
		BasePrice:       basePrice,
		Gold:            gold,
		QuantityBefore:  quantityBefore,
		ReadiedQuantity: g.readiedArmsQuantity(equipment),
	}
	if equipment.IsAmmo() || basePrice == 0 {
		return quote, nil
	}

	quote.UnitOffer = CalculateArmsSellPrice(basePrice, int(buyer.Intelligence))
	goldRoom := int(party_state.ItemQuantityLargeMax - gold)
	quote.MaxQuantity = uint16(min(int(quantityBefore), goldRoom/quote.UnitOffer))
	quote.Sellable = true

	return quote, nil
}

func (g *GameState) readiedArmsQuantity(equipment references.Equipment) uint16 {
	var quantity uint16
	g.PartyState.ForEachCharacter(func(index int, character *party_state.PlayerCharacter) {
		if index >= party_state.MaxActiveParty || character.PartyMap != party_state.PartyMapActive {
			return
		}
		for _, slot := range [...]references.EquipmentSlot{
			references.SlotHelm,
			references.SlotArmour,
			references.SlotWeapon,
			references.SlotShield,
			references.SlotRing,
			references.SlotAmulet,
		} {
			if character.GetEquippedItem(slot) == equipment {
				quantity++
			}
		}
	})

	return quantity
}
