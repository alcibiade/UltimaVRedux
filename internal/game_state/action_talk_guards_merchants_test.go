package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestCreateTalkDialogWithSpecialNPCs reproduces the issue with special dialog numbers
func TestCreateTalkDialogWithSpecialNPCs(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	t.Run("GuardDialog", func(t *testing.T) {
		// Create a mock guard NPC with DialogGuard (255)
		guardNPC := &map_units.NPCFriendly{}
		guardNPC.NPCReference = references.NPCReference{
			DialogNumber: byte(references.DialogGuard), // 255
		}

		mockCallbacks.Reset()

		// This should trigger the issue - CreateTalkDialog returns nil for guards
		t.Logf("Testing guard dialog creation (DialogNumber: %d)", references.DialogGuard)

		// Use defer to catch potential panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC REPRODUCED in guard dialog: %v", r)
			}
		}()

		dialog := gs.SystemCallbacks.Talk.CreateTalkDialog(guardNPC)

		// Analysis
		t.Logf("CreateTalkDialog returned: %v", dialog)
		t.Logf("Dialog is nil: %t", dialog == nil)

		if dialog == nil {
			t.Logf("✅ EXPECTED: CreateTalkDialog correctly returned nil for guard")

			// Now test what happens when we try to PushDialog with nil
			t.Logf("Testing PushDialog with nil dialog...")
			gs.SystemCallbacks.Talk.PushDialog(dialog) // This might panic!

			t.Logf("✅ PushDialog handled nil gracefully")
		}
	})

	t.Run("MerchantDialog", func(t *testing.T) {
		// Create a mock merchant NPC
		merchantNPC := &map_units.NPCFriendly{}
		merchantNPC.NPCReference = references.NPCReference{
			DialogNumber: byte(references.DialogWeaponDealer), // 129
		}

		mockCallbacks.Reset()

		t.Logf("Testing merchant dialog creation (DialogNumber: %d)", references.DialogWeaponDealer)

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC REPRODUCED in merchant dialog: %v", r)
			}
		}()

		dialog := gs.SystemCallbacks.Talk.CreateTalkDialog(merchantNPC)

		t.Logf("CreateTalkDialog returned: %v", dialog)
		t.Logf("Dialog is nil: %t", dialog == nil)

		if dialog == nil {
			t.Logf("✅ EXPECTED: CreateTalkDialog correctly returned nil for merchant")

			// Test PushDialog with nil
			t.Logf("Testing PushDialog with nil dialog...")
			gs.SystemCallbacks.Talk.PushDialog(dialog) // This might panic!

			t.Logf("✅ PushDialog handled nil gracefully")
		}
	})
}

// TestSpecialDialogNumberIdentification validates our understanding of special dialog numbers
func TestSpecialDialogNumberIdentification(t *testing.T) {
	t.Run("DialogNumbers", func(t *testing.T) {
		// Test guard dialog number
		guardDialog := references.DialogGuard
		assert.True(t, guardDialog.IsSpecial(), "Guards should be special")
		assert.False(t, guardDialog.IsMerchant(), "Guards are not merchants")

		// Test various merchant dialog numbers
		merchants := []references.DialogNumber{
			references.DialogWeaponDealer,
			references.DialogBarkeeper,
			references.DialogHorseDealer,
			references.DialogInnkeeper,
		}

		for _, merchant := range merchants {
			assert.True(t, merchant.IsSpecial(),
				"Merchant %d (%s) should be special", merchant, merchant.GetMerchantType())
			assert.True(t, merchant.IsMerchant(),
				"Dialog %d should be merchant", merchant)
		}

		// Test regular NPC dialog numbers
		regularNPC := references.DialogNumber(50)
		assert.False(t, regularNPC.IsSpecial(), "Regular NPC should not be special")
		assert.False(t, regularNPC.IsMerchant(), "Regular NPC should not be merchant")
	})
}
