package references

// ArmsDealerStock is one original arms dealer's seven displayed equipment rows.
type ArmsDealerStock struct {
	Items []Equipment
}

// Original arms catalogs in MerchantDefinition.CatalogRow order; ref:MERCH-0004.
// Prices remain authoritative in equipmentDefinitions and are read with GetBasePrice.
//
//nolint:gochecknoglobals // Canonical original arms rows are immutable after initialization.
var armsStockByCatalogRow = [9][7]Equipment{
	{Dagger, Sling, Bow, Arrows, Crossbow, Quarrels, MagicBow},
	{FlamingOil, Mace, SpikedCollar, ThrowingAxe, SpikedHelm, SpikedShield, MorningStar},
	{LeatherHelm, ClothArmour, LeatherArmour, Club, Spear, SilverSword, MagicAxe},
	{IronHelm, SmallShield, RingMail, ShortSword, LongSword, Mace, TwoHHammer},
	{TwoHAxe, TwoHSword, Halberd, IronHelm, LargeShield, ScaleMail, PlateMail},
	{ChainCoif, MagicShield, ChainMail, PlateMail, LongSword, SilverSword, RingProtection},
	{LeatherHelm, LeatherArmour, Dagger, MainGauche, ShortSword, FlamingOil, RingInvisibility},
	{MagicShield, TwoHAxe, MagicBow, Arrows, TwoHHammer, RingRegeneration, AmuletOfTurning},
	{ChainCoif, ChainMail, Crossbow, Quarrels, Halberd, ThrowingAxe, MorningStar},
}

// ArmsStock returns the canonical stock for one active Arms merchant. The
// returned slice is detached from the private row table.
func (r *MerchantReferences) ArmsStock(merchantID MerchantID) (ArmsDealerStock, bool) {
	if r == nil {
		return ArmsDealerStock{}, false
	}

	merchant, ok := r.Definition(merchantID)
	if !ok || merchant.Family != MerchantFamilyArms ||
		merchant.CatalogRow < 0 || merchant.CatalogRow >= len(armsStockByCatalogRow) {
		return ArmsDealerStock{}, false
	}

	row := armsStockByCatalogRow[merchant.CatalogRow]

	return ArmsDealerStock{Items: append([]Equipment(nil), row[:]...)}, true
}
