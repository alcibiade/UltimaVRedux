package input

import "github.com/hajimehoshi/ebiten/v2"

// Test support: fixtures and seams that tests in other packages use. The
// dead-code check treats this file as test support (workshop deadcode config).

// NewKeyboardWithKeyStateSource constructs a keyboard backed by a caller-owned
// key-state source. It supports deterministic input adapters and tests without
// changing the keyboard's release-debouncing behavior.
func NewKeyboardWithKeyStateSource(
	millisecondDelayBetweenKeyPresses int,
	keyPressed func(ebiten.Key) bool,
) *Keyboard {
	if keyPressed == nil {
		panic("NewKeyboardWithKeyStateSource: key-state source is required")
	}
	keyboard := NewKeyboard(millisecondDelayBetweenKeyPresses)
	keyboard.keyPressed = keyPressed

	return keyboard
}
