package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInnReferencesContainSixExactCanonicalRows(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	expected := []struct {
		id       MerchantID
		location Location
		owner    string
		shop     string
		base     int
		capacity int
		sleep    PlayerLocation
		wake     PlayerLocation
	}{
		{MerchantInnBritain, Britain, "Donya", "The Wayfarer Inn", 2, 3, PlayerLocation{Britain, Position{21, 10}, 0}, PlayerLocation{Britain, Position{22, 10}, 0}},
		{MerchantInnJhelom, Jhelom, "Gremnor", "The Warrior's Stead", 3, 4, PlayerLocation{Jhelom, Position{15, 7}, 1}, PlayerLocation{Jhelom, Position{16, 7}, 1}},
		{MerchantInnSkaraBrae, SkaraBrae, "Rogi", "The Haunting Inn", 2, 3, PlayerLocation{SkaraBrae, Position{25, 9}, 0}, PlayerLocation{SkaraBrae, Position{26, 9}, 0}},
		{MerchantInnNorthBritanny, NorthBritanny, "Terbor", "Hotel Brittany", 3, 2, PlayerLocation{NorthBritanny, Position{20, 1}, 0}, PlayerLocation{NorthBritanny, Position{21, 1}, 0}},
		{MerchantInnPaws, Paws, "Lorien", "The Smugglers' Inn", 2, 2, PlayerLocation{Paws, Position{27, 6}, 0}, PlayerLocation{Paws, Position{28, 6}, 0}},
		{MerchantInnBuccaneersDen, BuccaneersDen, "Ransack", "The King's Ransom Inn", 3, 2, PlayerLocation{BuccaneersDen, Position{7, 26}, 0}, PlayerLocation{BuccaneersDen, Position{8, 26}, 0}},
	}
	require.Len(t, innsByCatalogRow, len(expected))

	for row, want := range expected {
		merchant, ok := refs.DefinitionForCatalog(MerchantFamilyInn, row)
		require.True(t, ok, "row %d", row)
		assert.Equal(t, want.id, merchant.ID)
		assert.Equal(t, want.location, merchant.Location)
		assert.Equal(t, want.owner, merchant.OwnerName)
		assert.Equal(t, want.shop, merchant.ShopName)

		inn, ok := refs.Inn(want.id)
		require.True(t, ok, want.id)
		assert.Equal(t, InnReference{
			MerchantID: want.id, BasePrice: want.base, GuestCapacity: want.capacity,
			SleepLocation: want.sleep, WakeLocation: want.wake,
		}, inn)
	}
}

func TestInnReferencesRejectNonInnAndUnknownRows(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	_, ok := refs.Inn(MerchantHealerMinoc)
	assert.False(t, ok)
	_, ok = refs.Inn(MerchantID("inn.unknown"))
	assert.False(t, ok)
	_, ok = (*MerchantReferences)(nil).Inn(MerchantInnBritain)
	assert.False(t, ok)
}
