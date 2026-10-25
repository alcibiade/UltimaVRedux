package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

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
