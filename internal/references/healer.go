package references

// HealerService identifies one original healer treatment.
type HealerService byte

// Original healer service menu order; ref:MERCH-0007.
const (
	HealerCure HealerService = iota
	HealerHeal
	HealerResurrect
)

// HealerOffer is one fixed-price treatment from a canonical healer source row.
// Location-specific free and charitable care are applied by GameState authority.
type HealerOffer struct {
	Service HealerService
	Price   uint16
}

// Original cure, heal, and resurrection base prices (ref:MERCH-0007) in
// MerchantDefinition.CatalogRow order. The unmapped eighth source row is absent.
//
//nolint:gochecknoglobals // Canonical original healer rows are immutable after initialization.
var healerOffersByCatalogRow = [7][3]HealerOffer{
	{{Service: HealerCure, Price: 20}, {Service: HealerHeal, Price: 35}, {Service: HealerResurrect, Price: 200}},
	{{Service: HealerCure, Price: 25}, {Service: HealerHeal, Price: 40}, {Service: HealerResurrect, Price: 215}},
	{{Service: HealerCure, Price: 30}, {Service: HealerHeal, Price: 45}, {Service: HealerResurrect, Price: 225}},
	{{Service: HealerCure, Price: 35}, {Service: HealerHeal, Price: 50}, {Service: HealerResurrect, Price: 237}},
	{{Service: HealerCure, Price: 40}, {Service: HealerHeal, Price: 55}, {Service: HealerResurrect, Price: 247}},
	{{Service: HealerCure, Price: 15}, {Service: HealerHeal, Price: 60}, {Service: HealerResurrect, Price: 249}},
	{{Service: HealerCure, Price: 10}, {Service: HealerHeal, Price: 65}, {Service: HealerResurrect, Price: 262}},
}

// String returns the player-facing service name.
func (s HealerService) String() string {
	switch s {
	case HealerCure:
		return "Cure"
	case HealerHeal:
		return "Heal"
	case HealerResurrect:
		return "Resurrect"
	default:
		return "Unknown"
	}
}

// HealerOffers returns Cure, Heal, and Resurrect source prices for one active
// healer. The returned array is detached from the private catalog.
func (r *MerchantReferences) HealerOffers(merchantID MerchantID) ([3]HealerOffer, bool) {
	if r == nil {
		return [3]HealerOffer{}, false
	}

	merchant, ok := r.Definition(merchantID)
	if !ok || merchant.Family != MerchantFamilyHealer ||
		merchant.CatalogRow < 0 || merchant.CatalogRow >= len(healerOffersByCatalogRow) {
		return [3]HealerOffer{}, false
	}

	return healerOffersByCatalogRow[merchant.CatalogRow], true
}
