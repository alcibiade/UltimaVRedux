package party_state

import (
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
