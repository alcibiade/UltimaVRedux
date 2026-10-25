package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDialogNumbers(t *testing.T) {
	t.Run("NoResponse", func(t *testing.T) {
		assert.Equal(t, DialogNumber(0), NoResponse)
		assert.False(t, NoResponse.IsMerchant())
		assert.False(t, NoResponse.IsSpecial())
		assert.Equal(t, "", NoResponse.GetMerchantType())
	})

	t.Run("RegularNPC", func(t *testing.T) {
		// Test a regular NPC dialog number
		regularNPC := DialogNumber(50)
		assert.False(t, regularNPC.IsMerchant())
		assert.False(t, regularNPC.IsSpecial())
		assert.Equal(t, "", regularNPC.GetMerchantType())
	})

	t.Run("MerchantThreshold", func(t *testing.T) {
		assert.Equal(t, DialogNumber(128), MerchantThreshold)
		assert.False(t, MerchantThreshold.IsMerchant())
		assert.True(t, MerchantThreshold.IsSpecial())
		assert.Equal(t, "", MerchantThreshold.GetMerchantType())
	})

	t.Run("WeaponDealer", func(t *testing.T) {
		assert.Equal(t, DialogNumber(129), DialogWeaponDealer)
		assert.True(t, DialogWeaponDealer.IsMerchant())
		assert.True(t, DialogWeaponDealer.IsSpecial())
		assert.Equal(t, "Weapon Dealer", DialogWeaponDealer.GetMerchantType())
	})

	t.Run("Barkeeper", func(t *testing.T) {
		assert.Equal(t, DialogNumber(130), DialogBarkeeper)
		assert.Equal(t, "Barkeeper", DialogBarkeeper.GetMerchantType())
		assert.True(t, DialogBarkeeper.IsMerchant())
	})

	t.Run("AllMerchants", func(t *testing.T) {
		merchants := []struct {
			dialog   DialogNumber
			expected string
		}{
			{DialogWeaponDealer, "Weapon Dealer"},
			{DialogBarkeeper, "Barkeeper"},
			{DialogHorseDealer, "Horse Dealer"},
			{DialogShipDealer, "Ship Dealer"},
			{DialogMagicDealer, "Magic Dealer"},
			{DialogGuildMaster, "Guild Master"},
			{DialogHealer, "Healer"},
			{DialogInnkeeper, "Innkeeper"},
		}

		for _, m := range merchants {
			assert.True(t, m.dialog.IsMerchant(), "Dialog %d should be merchant", m.dialog)
			assert.True(t, m.dialog.IsSpecial(), "Dialog %d should be special", m.dialog)
			assert.Equal(t, m.expected, m.dialog.GetMerchantType(), "Wrong merchant type for dialog %d", m.dialog)
		}
	})

	t.Run("Guard", func(t *testing.T) {
		assert.Equal(t, DialogNumber(255), DialogGuard)
		assert.False(t, DialogGuard.IsMerchant()) // Guards are not merchants
		assert.True(t, DialogGuard.IsSpecial())
		assert.Equal(t, "Guard", DialogGuard.GetMerchantType())
	})

	t.Run("UnsupportedSpecial", func(t *testing.T) {
		unknown := DialogNumber(137)
		assert.False(t, unknown.IsMerchant())
		assert.True(t, unknown.IsSpecial())
		assert.Equal(t, "", unknown.GetMerchantType())
	})
}

func TestDialogNumberMapping(t *testing.T) {
	// Verify the documented mapping between NPCType and DialogNumber
	t.Run("NPCTypeToDialogMapping", func(t *testing.T) {
		// Based on the documentation in dialog_numbers.go
		// The mapping shows that NPCType values (0x81-0x88) correspond to
		// DialogNumber values (129-136)
		// 0x81 = 129 decimal, and 129 dialog = 129, so they're directly related
		// Actually NPCType 0x81 = 129 decimal, which maps to DialogNumber 129

		assert.Equal(t, byte(129), byte(Blacksmith), "Blacksmith NPCType should be 0x81 (129)")
		assert.Equal(t, byte(130), byte(Barkeeper), "Barkeeper NPCType should be 0x82 (130)")
		assert.Equal(t, byte(131), byte(HorseSeller), "HorseSeller NPCType should be 0x83 (131)")
		assert.Equal(t, byte(132), byte(Shipwright), "Shipwright NPCType should be 0x84 (132)")
		assert.Equal(t, byte(133), byte(MagicSeller), "MagicSeller NPCType should be 0x85 (133)")
		assert.Equal(t, byte(134), byte(GuildMaster), "GuildMaster NPCType should be 0x86 (134)")
		assert.Equal(t, byte(135), byte(Healer), "Healer NPCType should be 0x87 (135)")
		assert.Equal(t, byte(136), byte(InnKeeper), "InnKeeper NPCType should be 0x88 (136)")

		// The NPCType values happen to match the DialogNumber values numerically
		assert.Equal(t, DialogNumber(129), DialogWeaponDealer)
		assert.Equal(t, DialogNumber(130), DialogBarkeeper)
		assert.Equal(t, DialogNumber(131), DialogHorseDealer)
		assert.Equal(t, DialogNumber(132), DialogShipDealer)
		assert.Equal(t, DialogNumber(133), DialogMagicDealer)
		assert.Equal(t, DialogNumber(134), DialogGuildMaster)
		assert.Equal(t, DialogNumber(135), DialogHealer)
		assert.Equal(t, DialogNumber(136), DialogInnkeeper)
	})
}
