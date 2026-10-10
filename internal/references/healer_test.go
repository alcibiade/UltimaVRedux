package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealerOffersContainSevenCanonicalRowsInServiceOrder(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	tests := []struct {
		merchantID MerchantID
		prices     [3]uint16
	}{
		{merchantID: MerchantHealerMinoc, prices: [3]uint16{20, 35, 200}},
		{merchantID: MerchantHealerTrinsic, prices: [3]uint16{25, 40, 215}},
		{merchantID: MerchantHealerSkaraBrae, prices: [3]uint16{30, 45, 225}},
		{merchantID: MerchantHealerEastBritanny, prices: [3]uint16{35, 50, 237}},
		{merchantID: MerchantHealerCove, prices: [3]uint16{40, 55, 247}},
		{merchantID: MerchantHealerLycaeum, prices: [3]uint16{15, 60, 249}},
		{merchantID: MerchantHealerEmpathAbbey, prices: [3]uint16{10, 65, 262}},
	}
	require.Len(t, healerOffersByCatalogRow, len(tests))

	for row, test := range tests {
		merchant, ok := refs.DefinitionForCatalog(MerchantFamilyHealer, row)
		require.True(t, ok, "row %d", row)
		assert.Equal(t, test.merchantID, merchant.ID)

		offers, ok := refs.HealerOffers(test.merchantID)
		require.True(t, ok, test.merchantID)
		for index, service := range [...]HealerService{HealerCure, HealerHeal, HealerResurrect} {
			assert.Equal(t, service, offers[index].Service)
			assert.Equal(t, test.prices[index], offers[index].Price)
		}
	}
}

func TestHealerOffersPreserveSourcePricesAndRejectInactiveRows(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	offers, ok := refs.HealerOffers(MerchantHealerMinoc)
	require.True(t, ok)
	assert.Equal(t, uint16(20), offers[0].Price, "GameState, not references, applies Minoc free Cure")
	assert.Equal(t, uint16(35), offers[1].Price, "GameState, not references, applies Minoc free Heal")

	_, ok = refs.DefinitionForCatalog(MerchantFamilyHealer, 7)
	assert.False(t, ok, "the orphan eighth source row must not become an active healer")
	_, ok = refs.HealerOffers(MerchantGuildPaws)
	assert.False(t, ok)
	_, ok = refs.HealerOffers(MerchantID("healer.unknown"))
	assert.False(t, ok)
	_, ok = (*MerchantReferences)(nil).HealerOffers(MerchantHealerMinoc)
	assert.False(t, ok)
}

func TestHealerServiceString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Cure", HealerCure.String())
	assert.Equal(t, "Heal", HealerHeal.String())
	assert.Equal(t, "Resurrect", HealerResurrect.String())
	assert.Equal(t, "Unknown", HealerService(255).String())
}
