package references

// HorseOffer is the fixed horse price from one active original dealer row.
type HorseOffer struct {
	BasePrice int
}

// Original horse base prices in MerchantDefinition.CatalogRow order; ref:MERCH-0008.
// The orphaned fourth source row is deliberately absent.
//
//nolint:gochecknoglobals // Canonical original horse rows are immutable after initialization.
var horseOffersByCatalogRow = [3]HorseOffer{
	{BasePrice: 100},
	{BasePrice: 130},
	{BasePrice: 160},
}

// HorseOffer returns the fixed source price for one active horse dealer.
func (r *MerchantReferences) HorseOffer(merchantID MerchantID) (HorseOffer, bool) {
	if r == nil {
		return HorseOffer{}, false
	}

	merchant, ok := r.Definition(merchantID)
	if !ok || merchant.Family != MerchantFamilyHorse ||
		merchant.CatalogRow < 0 || merchant.CatalogRow >= len(horseOffersByCatalogRow) {
		return HorseOffer{}, false
	}

	return horseOffersByCatalogRow[merchant.CatalogRow], true
}
