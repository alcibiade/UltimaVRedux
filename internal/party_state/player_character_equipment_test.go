package party_state

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestGetEquippedItem verifies slot-based equipment retrieval
func TestGetEquippedItem(t *testing.T) {
	pc := &PlayerCharacter{
		Helmet: byte(references.IronHelm),
		Armor:  byte(references.PlateMail),
		Weapon: byte(references.LongSword),
		Shield: byte(references.LargeShield),
		Ring:   byte(references.RingProtection),
		Amulet: byte(references.AmuletOfTurning),
	}

	tests := []struct {
		slot     references.EquipmentSlot
		expected references.Equipment
	}{
		{references.SlotHelm, references.IronHelm},
		{references.SlotArmour, references.PlateMail},
		{references.SlotWeapon, references.LongSword},
		{references.SlotShield, references.LargeShield},
		{references.SlotRing, references.RingProtection},
		{references.SlotAmulet, references.AmuletOfTurning},
	}

	for _, tc := range tests {
		t.Run(tc.slot.String(), func(t *testing.T) {
			got := pc.GetEquippedItem(tc.slot)
			if got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

// TestSetEquippedItem verifies setting equipment in slots
func TestSetEquippedItem(t *testing.T) {
	pc := &PlayerCharacter{}

	pc.SetEquippedItem(references.SlotHelm, references.SpikedHelm)
	pc.SetEquippedItem(references.SlotArmour, references.ChainMail)
	pc.SetEquippedItem(references.SlotWeapon, references.Mace)
	pc.SetEquippedItem(references.SlotShield, references.SmallShield)
	pc.SetEquippedItem(references.SlotRing, references.RingInvisibility)
	pc.SetEquippedItem(references.SlotAmulet, references.SpikedCollar)

	if pc.Helmet != byte(references.SpikedHelm) {
		t.Errorf("Helmet: expected %v, got %v", references.SpikedHelm, pc.Helmet)
	}
	if pc.Armor != byte(references.ChainMail) {
		t.Errorf("Armor: expected %v, got %v", references.ChainMail, pc.Armor)
	}
	if pc.Weapon != byte(references.Mace) {
		t.Errorf("Weapon: expected %v, got %v", references.Mace, pc.Weapon)
	}
	if pc.Shield != byte(references.SmallShield) {
		t.Errorf("Shield: expected %v, got %v", references.SmallShield, pc.Shield)
	}
	if pc.Ring != byte(references.RingInvisibility) {
		t.Errorf("Ring: expected %v, got %v", references.RingInvisibility, pc.Ring)
	}
	if pc.Amulet != byte(references.SpikedCollar) {
		t.Errorf("Amulet: expected %v, got %v", references.SpikedCollar, pc.Amulet)
	}
}

// TestUnequipSlot verifies unequipping returns previous item
func TestUnequipSlot(t *testing.T) {
	pc := &PlayerCharacter{
		Helmet: byte(references.NoEquipment),
		Armor:  byte(references.NoEquipment),
		Weapon: byte(references.LongSword),
		Shield: byte(references.NoEquipment),
		Ring:   byte(references.NoEquipment),
		Amulet: byte(references.NoEquipment),
	}

	prev := pc.UnequipSlot(references.SlotWeapon)
	if prev != references.LongSword {
		t.Errorf("expected previous item %v, got %v", references.LongSword, prev)
	}
	if pc.Weapon != byte(references.NoEquipment) {
		t.Errorf("weapon slot should be empty after unequip")
	}

	// Unequip empty slot
	prev = pc.UnequipSlot(references.SlotHelm)
	if prev != references.NoEquipment {
		t.Errorf("expected NoEquipment from empty slot, got %v", prev)
	}
}

// TestIsSlotEmpty verifies empty slot detection
func TestIsSlotEmpty(t *testing.T) {
	pc := &PlayerCharacter{
		Helmet: byte(references.NoEquipment),
		Armor:  byte(references.NoEquipment),
		Weapon: byte(references.Dagger),
		Shield: byte(references.NoEquipment),
		Ring:   byte(references.NoEquipment),
		Amulet: byte(references.NoEquipment),
	}

	if pc.IsSlotEmpty(references.SlotWeapon) {
		t.Error("weapon slot should NOT be empty")
	}
	if !pc.IsSlotEmpty(references.SlotHelm) {
		t.Error("helm slot SHOULD be empty")
	}
	if !pc.IsSlotEmpty(references.SlotShield) {
		t.Error("shield slot SHOULD be empty")
	}
}

// TestHasTwoHandedWeaponEquipped verifies 2H weapon detection
func TestHasTwoHandedWeaponEquipped(t *testing.T) {
	tests := []struct {
		name     string
		weapon   references.Equipment
		expected bool
	}{
		{"No weapon", references.NoEquipment, false},
		{"One-handed sword", references.LongSword, false},
		{"Dagger", references.Dagger, false},
		{"Halberd (2H)", references.Halberd, true},
		{"Two-handed sword", references.TwoHSword, true},
		{"Bow (2H)", references.Bow, true},
		{"Crossbow (2H)", references.Crossbow, true},
		{"Magic Bow (2H)", references.MagicBow, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pc := &PlayerCharacter{Weapon: byte(tc.weapon)}
			got := pc.HasTwoHandedWeaponEquipped()
			if got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

// TestCanEquipItem_StrengthRequirement verifies STR checks
func TestCanEquipItem_StrengthRequirement(t *testing.T) {
	tests := []struct {
		name     string
		strength byte
		item     references.Equipment
		canEquip bool
	}{
		{"Halberd needs 18 STR - weak", 15, references.Halberd, false},
		{"Halberd needs 18 STR - exact", 18, references.Halberd, true},
		{"Halberd needs 18 STR - strong", 30, references.Halberd, true},
		{"Dagger needs 1 STR - too weak", 0, references.Dagger, false},
		{"Dagger needs 1 STR - exact", 1, references.Dagger, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize with empty slots (255 = NoEquipment)
			pc := &PlayerCharacter{
				Strength: tc.strength,
				Helmet:   byte(references.NoEquipment),
				Armor:    byte(references.NoEquipment),
				Weapon:   byte(references.NoEquipment),
				Shield:   byte(references.NoEquipment),
				Ring:     byte(references.NoEquipment),
				Amulet:   byte(references.NoEquipment),
			}
			canEquip, reason := pc.CanEquipItem(tc.item)
			if canEquip != tc.canEquip {
				t.Errorf("expected canEquip=%v, got %v (reason: %s)", tc.canEquip, canEquip, reason)
			}
			if !tc.canEquip && reason == "" {
				t.Error("expected failure reason when cannot equip")
			}
		})
	}
}

// TestCanEquipItem_TwoHandedConflicts verifies 2H weapon restrictions
func TestCanEquipItem_TwoHandedConflicts(t *testing.T) {
	t.Run("Cannot equip 2H weapon if shield equipped", func(t *testing.T) {
		pc := &PlayerCharacter{
			Strength: 30,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.NoEquipment),
			Shield:   byte(references.LargeShield),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.Halberd)
		if canEquip {
			t.Error("should NOT be able to equip 2H weapon with shield")
		}
		if reason == "" {
			t.Error("expected failure reason")
		}
	})

	t.Run("Can replace weapon with 2H weapon (caller handles unequip)", func(t *testing.T) {
		// Note: CanEquipItem now allows slot replacement - caller is responsible
		// for unequipping the old item first. This is how the Ready dialog works.
		pc := &PlayerCharacter{
			Strength: 30,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.Dagger),
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.Halberd)
		if !canEquip {
			t.Errorf("should be able to replace weapon with 2H weapon: %s", reason)
		}
	})

	t.Run("Can equip 2H weapon with both hands free", func(t *testing.T) {
		pc := &PlayerCharacter{
			Strength: 30,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.NoEquipment),
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.Halberd)
		if !canEquip {
			t.Errorf("should be able to equip 2H weapon: %s", reason)
		}
	})
}

// TestCanEquipItem_ShieldWithTwoHanded verifies shield+2H restrictions
func TestCanEquipItem_ShieldWithTwoHanded(t *testing.T) {
	t.Run("Cannot equip shield with 2H weapon", func(t *testing.T) {
		pc := &PlayerCharacter{
			Strength: 30,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.Halberd),
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.SmallShield)
		if canEquip {
			t.Error("should NOT be able to equip shield with 2H weapon")
		}
		if reason == "" {
			t.Error("expected failure reason")
		}
	})

	t.Run("Can equip shield with 1H weapon", func(t *testing.T) {
		pc := &PlayerCharacter{
			Strength: 30,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.LongSword),
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.SmallShield)
		if !canEquip {
			t.Errorf("should be able to equip shield with 1H weapon: %s", reason)
		}
	})
}

// TestGetTotalAttack verifies attack value summation
func TestGetTotalAttack(t *testing.T) {
	pc := &PlayerCharacter{
		Helmet: byte(references.SpikedHelm), // Attack: 4 (from atkval)
		Armor:  byte(references.NoEquipment),
		Weapon: byte(references.LongSword),    // Attack: 15 (0x0f)
		Shield: byte(references.SpikedShield), // Attack: 6 (0x06)
		Ring:   byte(references.NoEquipment),
		Amulet: byte(references.NoEquipment),
	}

	total := pc.GetTotalAttack()
	// SpikedHelm=4, LongSword=15, SpikedShield=6 = 25
	expected := 4 + 15 + 6
	if total != expected {
		t.Errorf("expected total attack %d, got %d", expected, total)
	}
}

// TestGetTotalDefense verifies defense value summation
func TestGetTotalDefense(t *testing.T) {
	pc := &PlayerCharacter{
		Helmet: byte(references.IronHelm),  // Defense: 3
		Armor:  byte(references.PlateMail), // Defense: 7 (0x07)
		Weapon: byte(references.NoEquipment),
		Shield: byte(references.LargeShield), // Defense: 3
		Ring:   byte(references.NoEquipment),
		Amulet: byte(references.NoEquipment),
	}

	total := pc.GetTotalDefense()
	// IronHelm=3, PlateMail=7, LargeShield=3 = 13
	expected := 3 + 7 + 3
	if total != expected {
		t.Errorf("expected total defense %d, got %d", expected, total)
	}
}

// TestGetWeaponRange verifies weapon range calculation
func TestGetWeaponRange(t *testing.T) {
	tests := []struct {
		name     string
		weapon   references.Equipment
		expected int
	}{
		{"No weapon (unarmed)", references.NoEquipment, 1},
		{"Long Sword (melee)", references.LongSword, 1},
		{"Bow (range 7)", references.Bow, 7},
		{"Crossbow (range 8)", references.Crossbow, 8},
		{"Magic Bow (range 15)", references.MagicBow, 15},
		{"Throwing Axe (range 4)", references.ThrowingAxe, 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pc := &PlayerCharacter{Weapon: byte(tc.weapon)}
			got := pc.GetWeaponRange()
			if got != tc.expected {
				t.Errorf("expected range %d, got %d", tc.expected, got)
			}
		})
	}
}

// TestEquipmentSlotEdgeCases verifies edge case handling
func TestEquipmentSlotEdgeCases(t *testing.T) {
	t.Run("NoEquipment value in slot", func(t *testing.T) {
		pc := &PlayerCharacter{
			Weapon: byte(references.NoEquipment),
		}
		if !pc.IsSlotEmpty(references.SlotWeapon) {
			t.Error("NoEquipment value should be considered empty")
		}
	})

	t.Run("Cannot equip NoEquipment", func(t *testing.T) {
		pc := &PlayerCharacter{Strength: 30}
		canEquip, _ := pc.CanEquipItem(references.NoEquipment)
		if canEquip {
			t.Error("should not be able to equip NoEquipment")
		}
	})

	t.Run("Invalid slot returns NoEquipment", func(t *testing.T) {
		pc := &PlayerCharacter{}
		got := pc.GetEquippedItem(references.SlotNone)
		if got != references.NoEquipment {
			t.Errorf("expected NoEquipment for SlotNone, got %v", got)
		}
	})
}

// TestGetTotalEquippedStrRequirement verifies cumulative STR requirement calculation
func TestGetTotalEquippedStrRequirement(t *testing.T) {
	t.Run("Empty character has 0 STR requirement", func(t *testing.T) {
		pc := &PlayerCharacter{
			Helmet: byte(references.NoEquipment),
			Armor:  byte(references.NoEquipment),
			Weapon: byte(references.NoEquipment),
			Shield: byte(references.NoEquipment),
			Ring:   byte(references.NoEquipment),
			Amulet: byte(references.NoEquipment),
		}
		total := pc.GetTotalEquippedStrRequirement()
		if total != 0 {
			t.Errorf("expected 0 STR requirement for empty character, got %d", total)
		}
	})

	t.Run("Single item STR requirement", func(t *testing.T) {
		pc := &PlayerCharacter{
			Helmet: byte(references.NoEquipment),
			Armor:  byte(references.NoEquipment),
			Weapon: byte(references.LongSword), // STR 9
			Shield: byte(references.NoEquipment),
			Ring:   byte(references.NoEquipment),
			Amulet: byte(references.NoEquipment),
		}
		total := pc.GetTotalEquippedStrRequirement()
		if total != 9 {
			t.Errorf("expected STR 9 for LongSword, got %d", total)
		}
	})

	t.Run("Multiple items sum correctly", func(t *testing.T) {
		// ChainCoif (1) = STR 1, PlateMail (14) = STR 12, LongSword (30) = STR 9, LargeShield (5) = STR 3
		// Total: 1 + 12 + 9 + 3 = 25
		pc := &PlayerCharacter{
			Helmet: byte(references.ChainCoif),
			Armor:  byte(references.PlateMail),
			Weapon: byte(references.LongSword),
			Shield: byte(references.LargeShield),
			Ring:   byte(references.NoEquipment),
			Amulet: byte(references.NoEquipment),
		}
		total := pc.GetTotalEquippedStrRequirement()
		expected := 1 + 12 + 9 + 3
		if total != expected {
			t.Errorf("expected STR %d, got %d", expected, total)
		}
	})

	t.Run("Rings and amulets have 0 STR requirement", func(t *testing.T) {
		pc := &PlayerCharacter{
			Helmet: byte(references.NoEquipment),
			Armor:  byte(references.NoEquipment),
			Weapon: byte(references.NoEquipment),
			Shield: byte(references.NoEquipment),
			Ring:   byte(references.RingProtection),
			Amulet: byte(references.Ankh),
		}
		total := pc.GetTotalEquippedStrRequirement()
		if total != 0 {
			t.Errorf("expected 0 STR for rings/amulets, got %d", total)
		}
	})
}

// TestCanEquipItem_CumulativeStrength verifies cumulative STR check (original U5 behavior)
// Equipped strength costs plus the new item's cost must not exceed player STR; ref:EQUIP-0011.
func TestCanEquipItem_CumulativeStrength(t *testing.T) {
	t.Run("Cumulative STR blocks heavy equipment", func(t *testing.T) {
		// STR 20, equipped: PlateMail (12) + LongSword (9) = 21 total already > 20
		// Even trying to equip LeatherHelm (0) should work since 21+0=21 still > 20... wait no.
		// Character already over limit. Let me recalculate:
		// PlateMail = 12, LongSword = 9 -> total = 21
		// Character STR = 20
		// Adding LeatherHelm (STR 0): 21 + 0 = 21 > 20 -> FAIL
		pc := &PlayerCharacter{
			Strength: 20,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.PlateMail), // STR 12
			Weapon:   byte(references.LongSword), // STR 9
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		// Try to equip ChainCoif (STR 1): 21 + 1 = 22 > 20 -> should FAIL
		canEquip, reason := pc.CanEquipItem(references.ChainCoif)
		if canEquip {
			t.Error("should NOT be able to equip - cumulative STR (22) exceeds player STR (20)")
		}
		if reason == "" {
			t.Error("expected failure reason")
		}
		// Verify the error message includes numbers for clarity
		if !containsNumber(reason, 22) || !containsNumber(reason, 20) {
			t.Errorf("expected reason to include total needed (22) and player STR (20), got: %s", reason)
		}
	})

	t.Run("Slot replacement subtracts old item STR", func(t *testing.T) {
		// STR 15, equipped: LongSword (STR 9)
		// Current load = 9
		// Replacing with ShortSword (STR 5): 9 - 9 + 5 = 5 <= 15 -> should PASS
		pc := &PlayerCharacter{
			Strength: 15,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.LongSword), // STR 9
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.ShortSword) // STR 5
		if !canEquip {
			t.Errorf("should be able to replace weapon (old STR subtracted): %s", reason)
		}
	})

	t.Run("Slot replacement still checks new total", func(t *testing.T) {
		// STR 15, equipped: LongSword (STR 9)
		// Replacing with TwoHHammer (STR 16): 9 - 9 + 16 = 16 > 15 -> should FAIL
		pc := &PlayerCharacter{
			Strength: 15,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.LongSword), // STR 9
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.TwoHHammer) // STR 16
		if canEquip {
			t.Error("should NOT be able to replace - new item STR (16) exceeds player STR (15)")
		}
		if reason == "" {
			t.Error("expected failure reason")
		}
	})

	t.Run("Exact STR match passes", func(t *testing.T) {
		// STR 21, equipped: PlateMail (12) + LongSword (9) = 21 = exactly at limit
		// Adding LeatherHelm (STR 0): 21 + 0 = 21 <= 21 -> should PASS
		pc := &PlayerCharacter{
			Strength: 21,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.PlateMail), // STR 12
			Weapon:   byte(references.LongSword), // STR 9
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		// LeatherHelm has STR 0
		canEquip, reason := pc.CanEquipItem(references.LeatherHelm)
		if !canEquip {
			t.Errorf("should be able to equip at exact STR limit: %s", reason)
		}
	})

	t.Run("Empty character can equip heavy items within STR", func(t *testing.T) {
		// STR 18, empty character
		// Equip Halberd (STR 18): 0 + 18 = 18 <= 18 -> should PASS
		pc := &PlayerCharacter{
			Strength: 18,
			Helmet:   byte(references.NoEquipment),
			Armor:    byte(references.NoEquipment),
			Weapon:   byte(references.NoEquipment),
			Shield:   byte(references.NoEquipment),
			Ring:     byte(references.NoEquipment),
			Amulet:   byte(references.NoEquipment),
		}
		canEquip, reason := pc.CanEquipItem(references.Halberd) // STR 18
		if !canEquip {
			t.Errorf("should be able to equip Halberd with STR 18: %s", reason)
		}
	})
}

// containsNumber checks if a string contains a specific number
func containsNumber(s string, n int) bool {
	return strings.Contains(s, fmt.Sprintf("%d", n))
}
