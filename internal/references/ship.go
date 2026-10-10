package references

// MaxFrigateSkiffs is the deliberate Redux gameplay capacity for one frigate.
// REDUX DEVIATION: Original boarding stores materialized frigate skiffs in an unsigned
// byte and enforces no designed cap (ref:VEH-0002). Redux limits the count to 12 for playability.
const MaxFrigateSkiffs = 12

// ShipOffer contains one shipwright's fixed frigate and skiff base prices.
type ShipOffer struct {
	FrigateBasePrice int
	SkiffBasePrice   int
}

// Original ship base prices in MerchantDefinition.CatalogRow order; ref:MERCH-0008.
//
//nolint:gochecknoglobals // Canonical original ship rows are immutable after initialization.
var shipOffersByCatalogRow = [4]ShipOffer{
	{FrigateBasePrice: 600, SkiffBasePrice: 200},
	{FrigateBasePrice: 753, SkiffBasePrice: 175},
	{FrigateBasePrice: 650, SkiffBasePrice: 125},
	{FrigateBasePrice: 700, SkiffBasePrice: 100},
}

// ShipOffer returns the source prices for one active ship dealer.
func (r *MerchantReferences) ShipOffer(merchantID MerchantID) (ShipOffer, bool) {
	if r == nil {
		return ShipOffer{}, false
	}

	merchant, ok := r.Definition(merchantID)
	if !ok || merchant.Family != MerchantFamilyShip ||
		merchant.CatalogRow < 0 || merchant.CatalogRow >= len(shipOffersByCatalogRow) {
		return ShipOffer{}, false
	}

	return shipOffersByCatalogRow[merchant.CatalogRow], true
}
