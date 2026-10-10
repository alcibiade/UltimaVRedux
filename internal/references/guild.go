package references

// GuildOffer is one fixed provision bundle sold by an original guild shop.
type GuildOffer struct {
	Provision      Provision
	BundleQuantity uint16
	BasePrice      int
}

//nolint:gochecknoglobals // Canonical original guild rows are immutable after initialization.
var guildOffersByMerchantID = map[MerchantID][3]GuildOffer{
	MerchantGuildNewMagincia: {
		{Provision: Key, BundleQuantity: 3, BasePrice: 190},
		{Provision: Gem, BundleQuantity: 4, BasePrice: 255},
		{Provision: Torches, BundleQuantity: 5, BasePrice: 12},
	},
	MerchantGuildPaws: {
		{Provision: Key, BundleQuantity: 3, BasePrice: 160},
		{Provision: Gem, BundleQuantity: 4, BasePrice: 200},
		{Provision: Torches, BundleQuantity: 5, BasePrice: 11},
	},
	MerchantGuildBuccaneersDen: {
		{Provision: Key, BundleQuantity: 3, BasePrice: 185},
		{Provision: Gem, BundleQuantity: 4, BasePrice: 225},
		{Provision: Torches, BundleQuantity: 5, BasePrice: 25},
	},
}

// GuildOffers returns the fixed Keys, Gems, and Torches offers for one active
// guild merchant. The returned array is detached from the private catalog.
func (r *MerchantReferences) GuildOffers(merchantID MerchantID) ([3]GuildOffer, bool) {
	if r == nil {
		return [3]GuildOffer{}, false
	}

	offers, ok := guildOffersByMerchantID[merchantID]

	return offers, ok
}
