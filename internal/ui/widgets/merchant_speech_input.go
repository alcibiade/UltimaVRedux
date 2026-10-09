package widgets

import (
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
)

// Merchant owners reserve Shift+Left/Right for speech, independently of their
// ordinary list/mode controls. The shell remains presentation-only.
func handleMerchantSpeechPaging(shell *MerchantDialogShell, keyboard *input.Keyboard) bool {
	if !keyboard.IsModifierPressed(input.ModifierShift) {
		return false
	}
	left, right := keyboard.IsKeyHeld(ebiten.KeyLeft), keyboard.IsKeyHeld(ebiten.KeyRight)
	if left && right {
		return true
	}
	if keyboard.IsKeyPressedAndRegister(ebiten.KeyLeft) {
		shell.PageExchange(-1)

		return true
	}
	if keyboard.IsKeyPressedAndRegister(ebiten.KeyRight) {
		shell.PageExchange(1)

		return true
	}

	return left || right // Throttled/held paging must not change a shop mode.
}
