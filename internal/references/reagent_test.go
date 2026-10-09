package references

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReagentOffersContainFiveSourceRowsAndTwentyTwoOffers(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	expected := []struct {
		merchantID MerchantID
		offers     []ReagentOffer
	}{
		{
			merchantID: MerchantReagentMoonglow,
			offers: []ReagentOffer{
				{Reagent: Ginseng, BundleQuantity: 10, BasePrice: 20},
				{Reagent: Garlic, BundleQuantity: 6, BasePrice: 18},
				{Reagent: SpiderSilk, BundleQuantity: 4, BasePrice: 12},
				{Reagent: NightShade, BundleQuantity: 1, BasePrice: 12},
				{Reagent: MandrakeRoot, BundleQuantity: 1, BasePrice: 13},
			},
		},
		{
			merchantID: MerchantReagentYew,
			offers: []ReagentOffer{
				{Reagent: SulfurAsh, BundleQuantity: 12, BasePrice: 12},
				{Reagent: Ginseng, BundleQuantity: 8, BasePrice: 16},
				{Reagent: Garlic, BundleQuantity: 8, BasePrice: 16},
				{Reagent: SpiderSilk, BundleQuantity: 2, BasePrice: 8},
				{Reagent: BloodMoss, BundleQuantity: 4, BasePrice: 20},
			},
		},
		{
			merchantID: MerchantReagentSkaraBrae,
			offers: []ReagentOffer{
				{Reagent: SulfurAsh, BundleQuantity: 14, BasePrice: 14},
				{Reagent: Ginseng, BundleQuantity: 8, BasePrice: 16},
				{Reagent: BloodMoss, BundleQuantity: 6, BasePrice: 30},
				{Reagent: BlackPearl, BundleQuantity: 6, BasePrice: 18},
			},
		},
		{
			merchantID: MerchantReagentCove,
			offers: []ReagentOffer{
				{Reagent: SpiderSilk, BundleQuantity: 2, BasePrice: 6},
				{Reagent: BloodMoss, BundleQuantity: 2, BasePrice: 8},
				{Reagent: BlackPearl, BundleQuantity: 2, BasePrice: 8},
				{Reagent: NightShade, BundleQuantity: 1, BasePrice: 10},
				{Reagent: MandrakeRoot, BundleQuantity: 1, BasePrice: 15},
			},
		},
		{
			merchantID: MerchantReagentLycaeum,
			offers: []ReagentOffer{
				{Reagent: BloodMoss, BundleQuantity: 4, BasePrice: 50},
				{Reagent: NightShade, BundleQuantity: 1, BasePrice: 30},
				{Reagent: MandrakeRoot, BundleQuantity: 1, BasePrice: 40},
			},
		},
	}
	require.Len(t, reagentOffersByCatalogRow, len(expected))

	totalOffers := 0
	for row, test := range expected {
		merchant, ok := refs.DefinitionForCatalog(MerchantFamilyReagent, row)
		require.True(t, ok, "row %d", row)
		assert.Equal(t, test.merchantID, merchant.ID)

		offers, ok := refs.ReagentOffers(test.merchantID)
		require.True(t, ok, test.merchantID)
		assert.Equal(t, test.offers, offers)
		totalOffers += len(offers)
	}
	assert.Equal(t, 22, totalOffers)
}

func TestReagentOffersReturnDefensiveCopiesAndReuseCanonicalNames(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	offers, ok := refs.ReagentOffers(MerchantReagentMoonglow)
	require.True(t, ok)
	offers[0] = ReagentOffer{Reagent: SulfurAsh, BundleQuantity: 99, BasePrice: 1}

	again, ok := refs.ReagentOffers(MerchantReagentMoonglow)
	require.True(t, ok)
	assert.Equal(t, ReagentOffer{Reagent: Ginseng, BundleQuantity: 10, BasePrice: 20}, again[0])

	inventoryRefs := NewInventoryItemsReferences()
	wantNames := []string{"Ginseng", "Garlic", "Spider Silk", "Night Shade", "Mandrake Root"}
	for index, offer := range again {
		assert.Equal(t, wantNames[index], inventoryRefs.Reagent[offer.Reagent].ItemName)
	}

	offerType := reflect.TypeOf(ReagentOffer{})
	require.Equal(t, 3, offerType.NumField(), "catalog rows must not duplicate reagent names")
	assert.Equal(t, "Reagent", offerType.Field(0).Name)
	assert.Equal(t, "BundleQuantity", offerType.Field(1).Name)
	assert.Equal(t, "BasePrice", offerType.Field(2).Name)

	_, ok = refs.ReagentOffers(MerchantGuildPaws)
	assert.False(t, ok)
	_, ok = refs.ReagentOffers(MerchantID("reagent.unknown"))
	assert.False(t, ok)
	_, ok = (&MerchantReferences{}).ReagentOffers(MerchantReagentMoonglow)
	assert.False(t, ok)
	_, ok = (*MerchantReferences)(nil).ReagentOffers(MerchantReagentMoonglow)
	assert.False(t, ok)
}
