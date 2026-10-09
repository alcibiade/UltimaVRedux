package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipDealerRowsAndPricesMatchActiveSourceOrder(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	expected := []struct {
		merchantID MerchantID
		location   Location
		offer      ShipOffer
	}{
		{MerchantShipJhelom, Jhelom, ShipOffer{FrigateBasePrice: 600, SkiffBasePrice: 200}},
		{MerchantShipMinoc, Minoc, ShipOffer{FrigateBasePrice: 753, SkiffBasePrice: 175}},
		{MerchantShipEastBritanny, EastBritanny, ShipOffer{FrigateBasePrice: 650, SkiffBasePrice: 125}},
		{MerchantShipBuccaneersDen, BuccaneersDen, ShipOffer{FrigateBasePrice: 700, SkiffBasePrice: 100}},
	}

	for row, want := range expected {
		definition, ok := refs.DefinitionForCatalog(MerchantFamilyShip, row)
		require.True(t, ok)
		assert.Equal(t, want.merchantID, definition.ID)
		assert.Equal(t, want.location, definition.Location)

		offer, ok := refs.ShipOffer(definition.ID)
		require.True(t, ok)
		assert.Equal(t, want.offer, offer)
	}
}

func TestShipOfferRejectsNonShipAndUnknownMerchant(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	_, ok := refs.ShipOffer(MerchantHorseTrinsic)
	assert.False(t, ok)
	_, ok = refs.ShipOffer(MerchantID("ship.unknown"))
	assert.False(t, ok)

	var nilRefs *MerchantReferences
	_, ok = nilRefs.ShipOffer(MerchantShipJhelom)
	assert.False(t, ok)
}
