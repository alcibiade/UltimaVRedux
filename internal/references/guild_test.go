package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuildOffersContainExactlyTheThreeActiveCatalogs(t *testing.T) {
	t.Parallel()

	require.Len(t, guildOffersByMerchantID, 3)
	tests := []struct {
		merchantID MerchantID
		prices     [3]int
	}{
		{merchantID: MerchantGuildNewMagincia, prices: [3]int{190, 255, 12}},
		{merchantID: MerchantGuildPaws, prices: [3]int{160, 200, 11}},
		{merchantID: MerchantGuildBuccaneersDen, prices: [3]int{185, 225, 25}},
	}
	wantProvisions := [3]Provision{Key, Gem, Torches}
	wantBundles := [3]uint16{3, 4, 5}
	refs := &MerchantReferences{}

	for _, test := range tests {
		offers, ok := refs.GuildOffers(test.merchantID)
		require.True(t, ok, test.merchantID)
		for index, offer := range offers {
			assert.Equal(t, wantProvisions[index], offer.Provision)
			assert.Equal(t, wantBundles[index], offer.BundleQuantity)
			assert.Equal(t, test.prices[index], offer.BasePrice)
		}
	}

	_, ok := refs.GuildOffers(MerchantReagentLycaeum)
	assert.False(t, ok)
	_, ok = refs.GuildOffers(MerchantID("guild.lycaeum"))
	assert.False(t, ok, "the orphan fourth source row must not become an active catalog")
}

func TestGuildOffersReturnsDetachedValues(t *testing.T) {
	t.Parallel()

	refs := &MerchantReferences{}
	offers, ok := refs.GuildOffers(MerchantGuildPaws)
	require.True(t, ok)
	offers[0].BasePrice = 1

	again, ok := refs.GuildOffers(MerchantGuildPaws)
	require.True(t, ok)
	assert.Equal(t, 160, again[0].BasePrice)
	_, ok = (*MerchantReferences)(nil).GuildOffers(MerchantGuildPaws)
	assert.False(t, ok)
}
