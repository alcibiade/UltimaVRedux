package references

import (
	"testing"
)

// TestStrengthRequirements verifies equipment strength costs; ref:EQUIP-0001.
func TestStrengthRequirements(t *testing.T) {
	// Original per-item strength costs; ref:EQUIP-0001.
	tests := []struct {
		equipment Equipment
		expected  int
		name      string
	}{
		// Helms
		{LeatherHelm, 0, "Leather Helm"},
		{ChainCoif, 1, "Chain Coif"},
		{IronHelm, 2, "Iron Helm"},
		{SpikedHelm, 3, "Spiked Helm"},
		// Shields
		{SmallShield, 2, "Small Shield"},
		{LargeShield, 3, "Large Shield"},
		{SpikedShield, 4, "Spiked Shield"},
		{MagicShield, 0, "Magic Shield"},
		{JewelShield, 0, "Jewel Shield"},
		// Armor
		{ClothArmour, 0, "Cloth Armour"},
		{LeatherArmour, 2, "Leather Armour"},
		{RingMail, 4, "Ring Mail"},
		{ScaleMail, 6, "Scale Mail"},
		{ChainMail, 10, "Chain Mail"},
		{PlateMail, 12, "Plate Mail"},
		{MysticArmour, 0, "Mystic Armour"},
		// Weapons
		{Dagger, 1, "Dagger"},
		{Sling, 2, "Sling"},
		{Club, 3, "Club"},
		{FlamingOil, 2, "Flaming Oil"},
		{MainGauche, 3, "Main Gauche"},
		{Spear, 4, "Spear"},
		{ThrowingAxe, 6, "Throwing Axe"},
		{ShortSword, 5, "Short Sword"},
		{Mace, 7, "Mace"},
		{MorningStar, 8, "Morning Star"},
		{Bow, 8, "Bow"},
		{Arrows, 0, "Arrows"},
		{Crossbow, 6, "Crossbow"},
		{Quarrels, 0, "Quarrels"},
		{LongSword, 9, "Long Sword"},
		{TwoHHammer, 16, "2H Hammer"},
		{TwoHAxe, 15, "2H Axe"},
		{TwoHSword, 13, "2H Sword"},
		{Halberd, 18, "Halberd"},
		// Magic weapons
		{ChaosSword, 0, "Chaos Sword"},
		{MagicBow, 0, "Magic Bow"},
		{SilverSword, 8, "Silver Sword"},
		{MagicAxe, 0, "Magic Axe"},
		{GlassSword, 5, "Glass Sword"},
		{JeweledSword, 0, "Jeweled Sword"},
		{MysticSword, 0, "Mystic Sword"},
		// Rings & Amulets (all 0)
		{RingInvisibility, 0, "Ring of Invisibility"},
		{RingProtection, 0, "Ring of Protection"},
		{RingRegeneration, 0, "Ring of Regeneration"},
		{AmuletOfTurning, 0, "Amulet of Turning"},
		{SpikedCollar, 0, "Spiked Collar"},
		{Ankh, 0, "Ankh"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.equipment.GetStrengthRequired()
			if got != tc.expected {
				t.Errorf("%s: expected STR %d, got %d", tc.name, tc.expected, got)
			}
		})
	}
}

// TestAttackValues verifies equipment attack ratings; ref:EQUIP-0001.
func TestAttackValues(t *testing.T) {
	tests := []struct {
		equipment Equipment
		expected  int
		name      string
	}{
		// Helms (mostly 0, SpikedHelm has 4)
		{LeatherHelm, 0, "Leather Helm"},
		{SpikedHelm, 4, "Spiked Helm"},
		// Shields (SpikedShield has 6)
		{SmallShield, 0, "Small Shield"},
		{SpikedShield, 6, "Spiked Shield"},
		// Armor (all 0)
		{ClothArmour, 0, "Cloth Armour"},
		{PlateMail, 0, "Plate Mail"},
		// Weapons
		{Dagger, 6, "Dagger"},
		{Sling, 6, "Sling"},
		{Club, 8, "Club"},
		{FlamingOil, 8, "Flaming Oil"},
		{MainGauche, 8, "Main Gauche"},
		{Spear, 10, "Spear"},
		{ThrowingAxe, 10, "Throwing Axe"},
		{ShortSword, 12, "Short Sword"},
		{Mace, 15, "Mace"},
		{MorningStar, 15, "Morning Star"},
		{Bow, 10, "Bow"},
		{Arrows, 1, "Arrows"},
		{Crossbow, 12, "Crossbow"},
		{Quarrels, 1, "Quarrels"},
		{LongSword, 15, "Long Sword"},
		{TwoHHammer, 20, "2H Hammer"},
		{TwoHAxe, 20, "2H Axe"},
		{TwoHSword, 20, "2H Sword"},
		{Halberd, 30, "Halberd"},
		// Magic weapons
		{ChaosSword, 99, "Chaos Sword"},
		{MagicBow, 15, "Magic Bow"},
		{SilverSword, 12, "Silver Sword"},
		{MagicAxe, 20, "Magic Axe"},
		{GlassSword, 99, "Glass Sword"},
		{JeweledSword, 1, "Jeweled Sword"},
		{MysticSword, 30, "Mystic Sword"},
		// Rings & Amulets (all 0)
		{RingProtection, 0, "Ring of Protection"},
		{AmuletOfTurning, 0, "Amulet of Turning"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.equipment.GetAttackValue()
			if got != tc.expected {
				t.Errorf("%s: expected ATK %d, got %d", tc.name, tc.expected, got)
			}
		})
	}
}

// TestDefenseValues verifies equipment defense ratings; ref:EQUIP-0001.
func TestDefenseValues(t *testing.T) {
	tests := []struct {
		equipment Equipment
		expected  int
		name      string
	}{
		// Helms
		{LeatherHelm, 1, "Leather Helm"},
		{ChainCoif, 2, "Chain Coif"},
		{IronHelm, 3, "Iron Helm"},
		{SpikedHelm, 3, "Spiked Helm"},
		// Shields
		{SmallShield, 2, "Small Shield"},
		{LargeShield, 3, "Large Shield"},
		{SpikedShield, 3, "Spiked Shield"},
		{MagicShield, 5, "Magic Shield"},
		{JewelShield, 0, "Jewel Shield"},
		// Armor
		{ClothArmour, 1, "Cloth Armour"},
		{LeatherArmour, 2, "Leather Armour"},
		{RingMail, 3, "Ring Mail"},
		{ScaleMail, 4, "Scale Mail"},
		{ChainMail, 5, "Chain Mail"},
		{PlateMail, 7, "Plate Mail"},
		{MysticArmour, 10, "Mystic Armour"},
		// Weapons (mostly 0, MainGauche has 1)
		{Dagger, 0, "Dagger"},
		{MainGauche, 1, "Main Gauche"},
		{LongSword, 0, "Long Sword"},
		// Magic weapons (Mystic Sword has 1)
		{MysticSword, 1, "Mystic Sword"},
		{ChaosSword, 0, "Chaos Sword"},
		// Rings/Amulets
		{RingInvisibility, 0, "Ring of Invisibility"},
		{RingProtection, 2, "Ring of Protection"},
		{RingRegeneration, 0, "Ring of Regeneration"},
		{AmuletOfTurning, 0, "Amulet of Turning"},
		{SpikedCollar, 2, "Spiked Collar"},
		{Ankh, 0, "Ankh"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.equipment.GetDefenseValue()
			if got != tc.expected {
				t.Errorf("%s: expected DEF %d, got %d", tc.name, tc.expected, got)
			}
		})
	}
}

// TestTwoHandedWeapons verifies the 11 two-handed weapons; ref:EQUIP-0002.
func TestTwoHandedWeapons(t *testing.T) {
	twoHanded := []Equipment{
		Sling,       // 17
		FlamingOil,  // 19
		Bow,         // 26
		Crossbow,    // 28
		TwoHHammer,  // 31
		TwoHAxe,     // 32
		TwoHSword,   // 33
		Halberd,     // 34
		ChaosSword,  // 35
		MagicBow,    // 36
		MysticSword, // 41
	}

	for _, e := range twoHanded {
		if !e.IsTwoHanded() {
			t.Errorf("%v should be two-handed", e)
		}
	}

	// Verify one-handed weapons are NOT two-handed
	oneHanded := []Equipment{
		Dagger, Club, MainGauche, Spear, ThrowingAxe, ShortSword,
		Mace, MorningStar, LongSword, SilverSword, MagicAxe, GlassSword, JeweledSword,
	}

	for _, e := range oneHanded {
		if e.IsTwoHanded() {
			t.Errorf("%v should NOT be two-handed", e)
		}
	}
}

// TestEquipmentSlots verifies slot assignment for all equipment types
func TestEquipmentSlots(t *testing.T) {
	tests := []struct {
		equipment Equipment
		expected  EquipmentSlot
		name      string
	}{
		// Helms
		{LeatherHelm, SlotHelm, "Leather Helm"},
		{ChainCoif, SlotHelm, "Chain Coif"},
		{IronHelm, SlotHelm, "Iron Helm"},
		{SpikedHelm, SlotHelm, "Spiked Helm"},
		// Shields
		{SmallShield, SlotShield, "Small Shield"},
		{LargeShield, SlotShield, "Large Shield"},
		{SpikedShield, SlotShield, "Spiked Shield"},
		{MagicShield, SlotShield, "Magic Shield"},
		{JewelShield, SlotShield, "Jewel Shield"},
		// Armor
		{ClothArmour, SlotArmour, "Cloth Armour"},
		{LeatherArmour, SlotArmour, "Leather Armour"},
		{PlateMail, SlotArmour, "Plate Mail"},
		{MysticArmour, SlotArmour, "Mystic Armour"},
		// Weapons
		{Dagger, SlotWeapon, "Dagger"},
		{LongSword, SlotWeapon, "Long Sword"},
		{Halberd, SlotWeapon, "Halberd"},
		{MysticSword, SlotWeapon, "Mystic Sword"},
		// Ammo (not equippable)
		{Arrows, SlotNone, "Arrows"},
		{Quarrels, SlotNone, "Quarrels"},
		// Rings
		{RingInvisibility, SlotRing, "Ring of Invisibility"},
		{RingProtection, SlotRing, "Ring of Protection"},
		{RingRegeneration, SlotRing, "Ring of Regeneration"},
		// Amulets
		{AmuletOfTurning, SlotAmulet, "Amulet of Turning"},
		{SpikedCollar, SlotAmulet, "Spiked Collar"},
		{Ankh, SlotAmulet, "Ankh"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.equipment.GetSlotType()
			if got != tc.expected {
				t.Errorf("%s: expected slot %v, got %v", tc.name, tc.expected, got)
			}
		})
	}
}

// TestAmmoRequirements verifies weapon -> ammo mappings
func TestAmmoRequirements(t *testing.T) {
	tests := []struct {
		weapon   Equipment
		expected Equipment
		name     string
	}{
		{Bow, Arrows, "Bow requires Arrows"},
		{MagicBow, Arrows, "Magic Bow requires Arrows"},
		{Crossbow, Quarrels, "Crossbow requires Quarrels"},
		{LongSword, NoEquipment, "Long Sword requires no ammo"},
		{Dagger, NoEquipment, "Dagger requires no ammo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.weapon.GetRequiredAmmo()
			if got != tc.expected {
				t.Errorf("%s: expected ammo %v, got %v", tc.name, tc.expected, got)
			}
		})
	}
}

// TestIsAmmo verifies ammo detection
func TestIsAmmo(t *testing.T) {
	if !Arrows.IsAmmo() {
		t.Error("Arrows should be ammo")
	}
	if !Quarrels.IsAmmo() {
		t.Error("Quarrels should be ammo")
	}
	if Bow.IsAmmo() {
		t.Error("Bow should NOT be ammo")
	}
	if LongSword.IsAmmo() {
		t.Error("Long Sword should NOT be ammo")
	}
}

// TestEquipmentAbilities verifies the EquipmentAbility enum and HasAbility method
func TestEquipmentAbilities(t *testing.T) {
	// Test Ring of Invisibility has correct abilities
	if !RingInvisibility.HasAbility(AbilityInvisibility) {
		t.Error("Ring of Invisibility should have AbilityInvisibility")
	}
	if !RingInvisibility.HasAbility(AbilityMayVanish) {
		t.Error("Ring of Invisibility should have AbilityMayVanish")
	}
	if RingInvisibility.HasAbility(AbilityRegeneration) {
		t.Error("Ring of Invisibility should NOT have AbilityRegeneration")
	}

	// Test Ring of Regeneration has correct abilities
	if !RingRegeneration.HasAbility(AbilityRegeneration) {
		t.Error("Ring of Regeneration should have AbilityRegeneration")
	}
	if !RingRegeneration.HasAbility(AbilityMayVanish) {
		t.Error("Ring of Regeneration should have AbilityMayVanish")
	}

	// Test Chaos Sword has charm ability
	if !ChaosSword.HasAbility(AbilityCharmsWielder) {
		t.Error("Chaos Sword should have AbilityCharmsWielder")
	}
	if ChaosSword.HasAbility(AbilityMayVanish) {
		t.Error("Chaos Sword should NOT have AbilityMayVanish")
	}

	// Test Glass Sword shatters
	if !GlassSword.HasAbility(AbilityShatters) {
		t.Error("Glass Sword should have AbilityShatters")
	}

	// Test Jeweled Sword never misses
	if !JeweledSword.HasAbility(AbilityNeverMisses) {
		t.Error("Jeweled Sword should have AbilityNeverMisses")
	}

	// Test Magic Axe returns
	if !MagicAxe.HasAbility(AbilityReturns) {
		t.Error("Magic Axe should have AbilityReturns")
	}

	// Test Amulet of Turning reflects spells
	if !AmuletOfTurning.HasAbility(AbilitySpellReflect) {
		t.Error("Amulet of Turning should have AbilitySpellReflect")
	}

	// Test normal equipment has no abilities
	if LongSword.HasAbility(AbilityNeverMisses) {
		t.Error("Long Sword should NOT have any special abilities")
	}
	abilities := LongSword.GetAbilities()
	if len(abilities) != 0 {
		t.Errorf("Long Sword should have 0 abilities, got %d", len(abilities))
	}

	// Test GetAbilities returns correct count
	ringAbilities := RingInvisibility.GetAbilities()
	if len(ringAbilities) != 2 {
		t.Errorf("Ring of Invisibility should have 2 abilities, got %d", len(ringAbilities))
	}
}

// TestEquipmentAbilityString verifies the String() method for EquipmentAbility
func TestEquipmentAbilityString(t *testing.T) {
	tests := []struct {
		ability  EquipmentAbility
		expected string
	}{
		{AbilityInvisibility, "Invisibility"},
		{AbilityRegeneration, "Regeneration"},
		{AbilitySpellReflect, "Spell Reflect"},
		{AbilityCharmsWielder, "Charms Wielder"},
		{AbilityShatters, "Shatters"},
		{AbilityNeverMisses, "Never Misses"},
		{AbilityReturns, "Returns"},
		{AbilityMayVanish, "May Vanish"},
		{EquipmentAbility(99), "None"},
	}

	for _, tc := range tests {
		if tc.ability.String() != tc.expected {
			t.Errorf("EquipmentAbility(%d).String() = %q, expected %q",
				tc.ability, tc.ability.String(), tc.expected)
		}
	}
}

// TestAttackRange verifies ranged weapon reference distances; ref:EQUIP-0001.
func TestAttackRange(t *testing.T) {
	tests := []struct {
		equipment Equipment
		expected  int
		name      string
	}{
		{Dagger, 3, "Dagger"},
		{Sling, 4, "Sling"},
		{FlamingOil, 4, "Flaming Oil"},
		{Spear, 5, "Spear"},
		{ThrowingAxe, 4, "Throwing Axe"},
		{MorningStar, 2, "Morning Star"},
		{Bow, 7, "Bow"},
		{Crossbow, 8, "Crossbow"},
		{Halberd, 2, "Halberd"},
		{MagicBow, 15, "Magic Bow"},
		// Melee weapons with 0 range (Glass Sword is melee despite instant kill)
		{GlassSword, 0, "Glass Sword"},
		{Club, 0, "Club"},
		{LongSword, 0, "Long Sword"},
		{TwoHSword, 0, "2H Sword"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.equipment.GetAttackRange()
			if got != tc.expected {
				t.Errorf("%s: expected range %d, got %d", tc.name, tc.expected, got)
			}
		})
	}
}

// TestEquipmentSlotString verifies slot display names
func TestEquipmentSlotString(t *testing.T) {
	tests := []struct {
		slot     EquipmentSlot
		expected string
	}{
		{SlotNone, "None"},
		{SlotHelm, "Helm"},
		{SlotArmour, "Armour"},
		{SlotWeapon, "Weapon"},
		{SlotShield, "Shield"},
		{SlotRing, "Ring"},
		{SlotAmulet, "Amulet"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			got := tc.slot.String()
			if got != tc.expected {
				t.Errorf("slot.String(): expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestTypeCheckerMethods verifies the Is* helper methods
func TestTypeCheckerMethods(t *testing.T) {
	if !LongSword.IsWeapon() {
		t.Error("LongSword.IsWeapon() should be true")
	}
	if !PlateMail.IsArmour() {
		t.Error("PlateMail.IsArmour() should be true")
	}
	if !MagicShield.IsShield() {
		t.Error("MagicShield.IsShield() should be true")
	}
	if !IronHelm.IsHelm() {
		t.Error("IronHelm.IsHelm() should be true")
	}
	if !RingProtection.IsRing() {
		t.Error("RingProtection.IsRing() should be true")
	}
	if !AmuletOfTurning.IsAmulet() {
		t.Error("AmuletOfTurning.IsAmulet() should be true")
	}

	// Negative cases
	if LongSword.IsArmour() {
		t.Error("LongSword.IsArmour() should be false")
	}
	if PlateMail.IsWeapon() {
		t.Error("PlateMail.IsWeapon() should be false")
	}
}
