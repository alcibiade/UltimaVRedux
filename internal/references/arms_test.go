package references

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArmsStockContainsAllNineCanonicalRowsInCatalogOrder(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	expected := []struct {
		merchantID MerchantID
		items      [7]Equipment
	}{
		{MerchantArmsBritain, [7]Equipment{Dagger, Sling, Bow, Arrows, Crossbow, Quarrels, MagicBow}},
		{MerchantArmsJhelom, [7]Equipment{FlamingOil, Mace, SpikedCollar, ThrowingAxe, SpikedHelm, SpikedShield, MorningStar}},
		{MerchantArmsYew, [7]Equipment{LeatherHelm, ClothArmour, LeatherArmour, Club, Spear, SilverSword, MagicAxe}},
		{MerchantArmsMinoc, [7]Equipment{IronHelm, SmallShield, RingMail, ShortSword, LongSword, Mace, TwoHHammer}},
		{MerchantArmsTrinsic, [7]Equipment{TwoHAxe, TwoHSword, Halberd, IronHelm, LargeShield, ScaleMail, PlateMail}},
		{MerchantArmsCastleBritannia, [7]Equipment{ChainCoif, MagicShield, ChainMail, PlateMail, LongSword, SilverSword, RingProtection}},
		{MerchantArmsBuccaneersDen, [7]Equipment{LeatherHelm, LeatherArmour, Dagger, MainGauche, ShortSword, FlamingOil, RingInvisibility}},
		{MerchantArmsBordermarch, [7]Equipment{MagicShield, TwoHAxe, MagicBow, Arrows, TwoHHammer, RingRegeneration, AmuletOfTurning}},
		{MerchantArmsSerpentsHold, [7]Equipment{ChainCoif, ChainMail, Crossbow, Quarrels, Halberd, ThrowingAxe, MorningStar}},
	}
	require.Len(t, armsStockByCatalogRow, len(expected))

	for row, test := range expected {
		merchant, ok := refs.DefinitionForCatalog(MerchantFamilyArms, row)
		require.True(t, ok, "row %d", row)
		assert.Equal(t, test.merchantID, merchant.ID)

		stock, ok := refs.ArmsStock(test.merchantID)
		require.True(t, ok, test.merchantID)
		require.Len(t, stock.Items, 7)
		assert.Equal(t, test.items[:], stock.Items)
	}
}

func TestArmsStockReturnsDefensiveCopiesAndRejectsOtherDefinitions(t *testing.T) {
	t.Parallel()

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	stock, ok := refs.ArmsStock(MerchantArmsBritain)
	require.True(t, ok)
	stock.Items[0] = MysticSword

	again, ok := refs.ArmsStock(MerchantArmsBritain)
	require.True(t, ok)
	assert.Equal(t, Dagger, again.Items[0])

	_, ok = refs.ArmsStock(MerchantGuildPaws)
	assert.False(t, ok)
	_, ok = refs.ArmsStock(MerchantID("arms.unknown"))
	assert.False(t, ok)
	_, ok = (&MerchantReferences{}).ArmsStock(MerchantArmsBritain)
	assert.False(t, ok)
	_, ok = (*MerchantReferences)(nil).ArmsStock(MerchantArmsBritain)
	assert.False(t, ok)
}

func TestArmsStockDoesNotDuplicateEquipmentPrices(t *testing.T) {
	t.Parallel()

	typeOfStock := reflect.TypeOf(ArmsDealerStock{})
	require.Equal(t, 1, typeOfStock.NumField())
	assert.Equal(t, "Items", typeOfStock.Field(0).Name)

	refs, err := newMerchantReferenceIndex(originalMerchantDefinitions, nil)
	require.NoError(t, err)
	stock, ok := refs.ArmsStock(MerchantArmsBritain)
	require.True(t, ok)
	assert.Equal(t, 1, stock.Items[0].GetBasePrice())
	assert.Equal(t, 800, stock.Items[6].GetBasePrice())
}
