package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestTalkDialogCrashPrevention specifically tests the panic prevention in PushDialog
func TestTalkDialogCrashPrevention(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	t.Run("NilDialogHandling", func(t *testing.T) {
		mockCallbacks.Reset()

		// This should not panic - it's the core fix
		assert.NotPanics(t, func() {
			gs.SystemCallbacks.Talk.PushDialog(nil)
		}, "PushDialog must handle nil dialog without panicking")

		// Verify the call was tracked
		assert.Equal(t, 1, len(mockCallbacks.DialogsPushed), "PushDialog should be called")
		assert.Nil(t, mockCallbacks.DialogsPushed[0], "Pushed dialog should be nil")
	})
}

// TestSpecialNPCIdentificationAccuracy validates our understanding of special NPCs
func TestSpecialNPCIdentificationAccuracy(t *testing.T) {
	specialCases := []struct {
		dialogNum references.DialogNumber
		isSpecial bool
		isGuard   bool
		merchant  bool
		name      string
	}{
		{references.DialogGuard, true, true, false, "Guard"},
		{references.DialogWeaponDealer, true, false, true, "Weapon Dealer"},
		{references.DialogBarkeeper, true, false, true, "Barkeeper"},
		{references.DialogInnkeeper, true, false, true, "Innkeeper"},
		{references.DialogNumber(50), false, false, false, "Regular NPC"},
		{references.NoResponse, false, false, false, "No Response"},
	}

	for _, tc := range specialCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.isSpecial, tc.dialogNum.IsSpecial(),
				"IsSpecial() mismatch for %s", tc.name)
			assert.Equal(t, tc.isGuard, tc.dialogNum == references.DialogGuard,
				"Guard check mismatch for %s", tc.name)
			assert.Equal(t, tc.merchant, tc.dialogNum.IsMerchant(),
				"IsMerchant() mismatch for %s", tc.name)
		})
	}
}
