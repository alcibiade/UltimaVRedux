package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestAllMerchantTypes validates that all merchant dialog numbers are properly identified
func TestAllMerchantTypes(t *testing.T) {
	t.Run("MerchantDialogNumbers", func(t *testing.T) {
		// Based on original Ultima V merchant dispatch handles cnum-129
		// This means dialog numbers 129-136 are merchants
		// See ref:TALK-0005.
		merchants := []struct {
			dialogNum    references.DialogNumber
			expectedType string
		}{
			{references.DialogWeaponDealer, "Weapon Dealer"}, // 129
			{references.DialogBarkeeper, "Barkeeper"},        // 130
			{references.DialogHorseDealer, "Horse Dealer"},   // 131
			{references.DialogShipDealer, "Ship Dealer"},     // 132
			{references.DialogMagicDealer, "Magic Dealer"},   // 133
			{references.DialogGuildMaster, "Guild Master"},   // 134
			{references.DialogHealer, "Healer"},              // 135
			{references.DialogInnkeeper, "Innkeeper"},        // 136
		}

		for _, merchant := range merchants {
			t.Run(merchant.expectedType, func(t *testing.T) {
				assert.True(t, merchant.dialogNum.IsMerchant(),
					"Dialog %d (%s) should be identified as merchant",
					merchant.dialogNum, merchant.expectedType)
				assert.True(t, merchant.dialogNum.IsSpecial(),
					"Dialog %d (%s) should be identified as special",
					merchant.dialogNum, merchant.expectedType)
				assert.Equal(t, merchant.expectedType, merchant.dialogNum.GetMerchantType(),
					"Dialog %d should return correct merchant type", merchant.dialogNum)
			})
		}
	})
}

// TestSpecialNPCIdentification validates our understanding of special NPCs
func TestSpecialNPCIdentification(t *testing.T) {
	testCases := []struct {
		name         string
		dialogNum    references.DialogNumber
		isSpecial    bool
		isMerchant   bool
		expectedType string
	}{
		{"NoResponse", references.NoResponse, false, false, ""},
		{"RegularNPC_50", references.DialogNumber(50), false, false, ""},
		{"RegularNPC_127", references.DialogNumber(127), false, false, ""},
		{"WeaponDealer", references.DialogWeaponDealer, true, true, "Weapon Dealer"},
		{"Barkeeper", references.DialogBarkeeper, true, true, "Barkeeper"},
		{"HorseDealer", references.DialogHorseDealer, true, true, "Horse Dealer"},
		{"ShipDealer", references.DialogShipDealer, true, true, "Ship Dealer"},
		{"MagicDealer", references.DialogMagicDealer, true, true, "Magic Dealer"},
		{"GuildMaster", references.DialogGuildMaster, true, true, "Guild Master"},
		{"Healer", references.DialogHealer, true, true, "Healer"},
		{"Innkeeper", references.DialogInnkeeper, true, true, "Innkeeper"},
		{"Guard", references.DialogGuard, true, false, "Guard"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isSpecial, tc.dialogNum.IsSpecial(),
				"Dialog %d (%s) IsSpecial should be %t", tc.dialogNum, tc.name, tc.isSpecial)
			assert.Equal(t, tc.isMerchant, tc.dialogNum.IsMerchant(),
				"Dialog %d (%s) IsMerchant should be %t", tc.dialogNum, tc.name, tc.isMerchant)
			assert.Equal(t, tc.expectedType, tc.dialogNum.GetMerchantType(),
				"Dialog %d (%s) GetMerchantType should return '%s'", tc.dialogNum, tc.name, tc.expectedType)
		})
	}
}

// TestCreateTalkDialogComprehensive tests all special dialog number cases
func TestCreateTalkDialogComprehensive(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	testCases := []struct {
		name        string
		dialogNum   references.DialogNumber
		shouldCrash bool
	}{
		{"NoResponse", references.NoResponse, false},
		{"WeaponDealer", references.DialogWeaponDealer, false},
		{"Barkeeper", references.DialogBarkeeper, false},
		{"HorseDealer", references.DialogHorseDealer, false},
		{"ShipDealer", references.DialogShipDealer, false},
		{"MagicDealer", references.DialogMagicDealer, false},
		{"GuildMaster", references.DialogGuildMaster, false},
		{"Healer", references.DialogHealer, false},
		{"Innkeeper", references.DialogInnkeeper, false},
		{"Guard", references.DialogGuard, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock NPC
			npc := &map_units.NPCFriendly{}
			npc.NPCReference = references.NPCReference{
				DialogNumber: byte(tc.dialogNum),
			}

			mockCallbacks.Reset()

			// Test should not panic for any dialog number
			assert.NotPanics(t, func() {
				dialog := gs.SystemCallbacks.Talk.CreateTalkDialog(npc)

				// For special NPCs, dialog should be nil but handled gracefully
				if tc.dialogNum.IsSpecial() || tc.dialogNum == references.NoResponse {
					assert.Nil(t, dialog, "Special NPC should return nil dialog")

					// Test PushDialog with nil - should not crash
					gs.SystemCallbacks.Talk.PushDialog(dialog)
				}

				t.Logf("DialogNumber %d (%s): handled gracefully", tc.dialogNum, tc.name)
			}, "Dialog handling should not panic for %s (DialogNumber: %d)", tc.name, tc.dialogNum)
		})
	}
}
