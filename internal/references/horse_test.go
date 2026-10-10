package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHorseDealerRowsAndPricesMatchActiveSourceOrder(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	expected := []struct {
		definition MerchantDefinition
		basePrice  int
	}{
		{
			definition: MerchantDefinition{
				MerchantHorseTrinsic,
				MerchantFamilyHorse,
				Trinsic,
				1,
				0,
				"Hettar",
				"Horse & Rider",
			},
			basePrice: 100,
		},
		{
			definition: MerchantDefinition{
				MerchantHorseNorthBritanny,
				MerchantFamilyHorse,
				NorthBritanny,
				6,
				1,
				"Theoan",
				"The Stablehouse",
			},
			basePrice: 130,
		},
		{
			definition: MerchantDefinition{
				MerchantHorsePaws,
				MerchantFamilyHorse,
				Paws,
				4,
				2,
				"Ferru",
				"Wishing Well Horses",
			},
			basePrice: 160,
		},
	}

	for row, want := range expected {
		definition, ok := refs.DefinitionForCatalog(MerchantFamilyHorse, row)
		require.True(t, ok)
		assert.Equal(t, want.definition, definition)

		offer, ok := refs.HorseOffer(definition.ID)
		require.True(t, ok)
		assert.Equal(t, HorseOffer{BasePrice: want.basePrice}, offer)
	}

	_, ok := refs.DefinitionForCatalog(MerchantFamilyHorse, 3)
	assert.False(t, ok, "orphaned Lycaeum source row must remain absent")
	_, err = refs.Resolve(DialogHorseDealer, Lycaeum, 0)
	assert.ErrorContains(t, err, "no active merchant")
}

func TestHorseOfferRejectsNonHorseAndReturnsDetachedValue(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)

	offer, ok := refs.HorseOffer(MerchantHorseTrinsic)
	require.True(t, ok)
	offer.BasePrice = 1

	detached, ok := refs.HorseOffer(MerchantHorseTrinsic)
	require.True(t, ok)
	assert.Equal(t, HorseOffer{BasePrice: 100}, detached)

	_, ok = refs.HorseOffer(MerchantGuildPaws)
	assert.False(t, ok)
	_, ok = refs.HorseOffer(MerchantID("horse.unknown"))
	assert.False(t, ok)

	var nilRefs *MerchantReferences
	_, ok = nilRefs.HorseOffer(MerchantHorseTrinsic)
	assert.False(t, ok)
}
