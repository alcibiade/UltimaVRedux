package references

// ReagentOffer is one fixed reagent bundle sold by an original reagent shop.
type ReagentOffer struct {
	Reagent        Reagent
	BundleQuantity uint16
	BasePrice      int
}

// Original reagent bundle prices and quantities (ref:MERCH-0005) in
// MerchantDefinition.CatalogRow and reagent display order.
//
//nolint:gochecknoglobals // Canonical original reagent rows are immutable after initialization.
var reagentOffersByCatalogRow = [5][]ReagentOffer{
	{
		{Reagent: Ginseng, BundleQuantity: 10, BasePrice: 20},
		{Reagent: Garlic, BundleQuantity: 6, BasePrice: 18},
		{Reagent: SpiderSilk, BundleQuantity: 4, BasePrice: 12},
		{Reagent: NightShade, BundleQuantity: 1, BasePrice: 12},
		{Reagent: MandrakeRoot, BundleQuantity: 1, BasePrice: 13},
	},
	{
		{Reagent: SulfurAsh, BundleQuantity: 12, BasePrice: 12},
		{Reagent: Ginseng, BundleQuantity: 8, BasePrice: 16},
		{Reagent: Garlic, BundleQuantity: 8, BasePrice: 16},
		{Reagent: SpiderSilk, BundleQuantity: 2, BasePrice: 8},
		{Reagent: BloodMoss, BundleQuantity: 4, BasePrice: 20},
	},
	{
		{Reagent: SulfurAsh, BundleQuantity: 14, BasePrice: 14},
		{Reagent: Ginseng, BundleQuantity: 8, BasePrice: 16},
		{Reagent: BloodMoss, BundleQuantity: 6, BasePrice: 30},
		{Reagent: BlackPearl, BundleQuantity: 6, BasePrice: 18},
	},
	{
		{Reagent: SpiderSilk, BundleQuantity: 2, BasePrice: 6},
		{Reagent: BloodMoss, BundleQuantity: 2, BasePrice: 8},
		{Reagent: BlackPearl, BundleQuantity: 2, BasePrice: 8},
		{Reagent: NightShade, BundleQuantity: 1, BasePrice: 10},
		{Reagent: MandrakeRoot, BundleQuantity: 1, BasePrice: 15},
	},
	{
		{Reagent: BloodMoss, BundleQuantity: 4, BasePrice: 50},
		{Reagent: NightShade, BundleQuantity: 1, BasePrice: 30},
		{Reagent: MandrakeRoot, BundleQuantity: 1, BasePrice: 40},
	},
}

// ReagentOffers returns the canonical source-order offers for one active
// reagent merchant. The returned slice is detached from the private catalog.
func (r *MerchantReferences) ReagentOffers(merchantID MerchantID) ([]ReagentOffer, bool) {
	if r == nil {
		return nil, false
	}

	merchant, ok := r.Definition(merchantID)
	if !ok || merchant.Family != MerchantFamilyReagent ||
		merchant.CatalogRow < 0 || merchant.CatalogRow >= len(reagentOffersByCatalogRow) {
		return nil, false
	}

	return append([]ReagentOffer(nil), reagentOffersByCatalogRow[merchant.CatalogRow]...), true
}
