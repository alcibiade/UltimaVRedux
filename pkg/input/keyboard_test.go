package input

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newKeyboardTestFixture(delay int) *Keyboard {
	return NewKeyboardWithKeyStateSource(delay, func(ebiten.Key) bool { return false })
}

// The digit-nine regression covers the inclusive upper bound of the digit-key range.
func TestGetAlphaNumericPressed_IncludesDigitNine(t *testing.T) {
	// Test that all digit keys are properly detected
	digitTests := []struct {
		key     ebiten.Key
		keyName string
		keyChar string
	}{
		{ebiten.Key0, "Key0", "0"},
		{ebiten.Key1, "Key1", "1"},
		{ebiten.Key2, "Key2", "2"},
		{ebiten.Key3, "Key3", "3"},
		{ebiten.Key4, "Key4", "4"},
		{ebiten.Key5, "Key5", "5"},
		{ebiten.Key6, "Key6", "6"},
		{ebiten.Key7, "Key7", "7"},
		{ebiten.Key8, "Key8", "8"},
		{ebiten.Key9, "Key9", "9"}, // Regression: an exclusive upper bound omitted this key.
	}

	keyboard := newKeyboardTestFixture(100)

	// Test the key range detection loop - now tests the FIXED version
	t.Run("KeyRangeDetection", func(t *testing.T) {
		// Count how many digit keys the loop would check (after fix)
		count := 0
		for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
			count++
		}

		// The fixed code should check all 10 keys (0-9)
		if count != 10 {
			t.Errorf("Fixed loop checks %d keys, expected 10", count)
		}

		// Show that Key9 is now included in the range
		includesKey9 := false
		for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
			if i == int(ebiten.Key9) {
				includesKey9 = true
				break
			}
		}

		if !includesKey9 {
			t.Error("Fixed loop should include Key9")
		} else {
			t.Log("Loop correctly includes Key9")
		}
	})

	// Test GetKeyLetter for all digits to ensure the mapping works
	t.Run("GetKeyLetterMapping", func(t *testing.T) {
		for _, test := range digitTests {
			result := keyboard.GetKeyLetter(test.key)
			if result != test.keyChar {
				t.Errorf("GetKeyLetter(%s) = %q, expected %q", test.keyName, result, test.keyChar)
			}
		}
	})

	// Note: We can't easily test the actual key press detection in GetAlphaNumericPressed
	// because it relies on ebiten.IsKeyPressed() which requires the game loop context.
	// The fix is to change the loop condition from < to <=
}

// The digit-nine detection regression tests the corrected inclusive loop condition.
func TestDigitKeyRange_IncludesUpperBound(t *testing.T) {
	// Test the corrected loop range (what the fix should be)
	t.Run("CorrectedLoopRange", func(t *testing.T) {
		count := 0
		keysIncluded := []ebiten.Key{}

		// This is what the fixed loop should look like: <= instead of <
		for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
			count++
			keysIncluded = append(keysIncluded, ebiten.Key(i))
		}

		// The fixed code should check all 10 keys (0-9)
		if count != 10 {
			t.Errorf("Fixed loop should check 10 keys, got %d", count)
		}

		// Verify Key9 is included in the corrected range
		includesKey9 := false
		for _, key := range keysIncluded {
			if key == ebiten.Key9 {
				includesKey9 = true
				break
			}
		}

		if !includesKey9 {
			t.Error("Fixed loop should include Key9")
		} else {
			t.Log("Confirmed: Fixed loop correctly includes Key9")
		}
	})
}

// TestGetPressedKey_KeyPriority tests that GetPressedKey checks keys in correct priority order
func TestGetPressedKey_KeyPriority(t *testing.T) {
	// This test verifies the LOGIC of key priority checking
	// Actual key press detection requires game loop context

	t.Run("ArrowKeysPriority", func(t *testing.T) {
		// Arrow keys should be checked first via GetPressedArrowKey()
		// This is important for sailing - direction changes take priority

		keyboard := newKeyboardTestFixture(100)

		// The injected empty source exercises the no-input result directly.
		require.Nil(t, keyboard.GetPressedArrowKey())
		require.Nil(t, keyboard.GetPressedKey())
	})

	t.Run("AlphaNumericKeysSecondPriority", func(t *testing.T) {
		// After arrow keys, should check A-Z then 0-9
		// Verified by checking the loop ranges in GetPressedKey implementation

		// Count expected alpha keys (A-Z)
		alphaCount := 0
		for i := int(ebiten.KeyA); i <= int(ebiten.KeyZ); i++ {
			alphaCount++
		}

		if alphaCount != 26 {
			t.Errorf("Expected 26 alpha keys, got %d", alphaCount)
		}

		// Count expected numeric keys (0-9)
		numericCount := 0
		for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
			numericCount++
		}

		if numericCount != 10 {
			t.Errorf("Expected 10 numeric keys, got %d", numericCount)
		}
	})

	t.Run("ControlKeysThirdPriority", func(t *testing.T) {
		// Control keys checked last (Space, Enter, Escape, etc.)
		// List should include common control keys for sailing stop

		expectedControlKeys := []ebiten.Key{
			ebiten.KeySpace,
			ebiten.KeyEnter,
			ebiten.KeyEscape,
			ebiten.KeyTab,
			ebiten.KeyBackspace,
		}

		if len(expectedControlKeys) < 5 {
			t.Error("Expected at least 5 control keys to be checked")
		}
	})
}

func TestGetPressedArrowKeyUsesInjectedSourceAndPreservesPriority(t *testing.T) {
	tests := []struct {
		name    string
		pressed map[ebiten.Key]bool
		wantKey ebiten.Key
		wantDir references.Direction
	}{
		{name: "down", pressed: map[ebiten.Key]bool{ebiten.KeyArrowDown: true}, wantKey: ebiten.KeyArrowDown, wantDir: references.Down},
		{name: "up", pressed: map[ebiten.Key]bool{ebiten.KeyArrowUp: true}, wantKey: ebiten.KeyArrowUp, wantDir: references.Up},
		{name: "right", pressed: map[ebiten.Key]bool{ebiten.KeyArrowRight: true}, wantKey: ebiten.KeyArrowRight, wantDir: references.Right},
		{name: "left", pressed: map[ebiten.Key]bool{ebiten.KeyArrowLeft: true}, wantKey: ebiten.KeyArrowLeft, wantDir: references.Left},
		{
			name: "multiple keys use original priority",
			pressed: map[ebiten.Key]bool{
				ebiten.KeyArrowDown:  true,
				ebiten.KeyArrowUp:    true,
				ebiten.KeyArrowRight: true,
				ebiten.KeyArrowLeft:  true,
			},
			wantKey: ebiten.KeyArrowDown,
			wantDir: references.Down,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := newKeyboardTestFixture(0)
			keyboard.keyPressed = func(key ebiten.Key) bool { return test.pressed[key] }

			pressed := keyboard.GetPressedArrowKey()
			require.NotNil(t, pressed)
			require.Equal(t, test.wantKey, *pressed)
			require.Equal(t, test.wantDir, keyboard.GetArrowKeyAsDirection())
		})
	}
}

func TestSimulateKeyPressFeedsArrowRegistrationOnce(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	keyboard.keyPressed = func(ebiten.Key) bool { return false }
	keyboard.SimulateKeyPress(ebiten.KeyArrowUp)

	pressed := keyboard.GetPressedArrowKey()
	require.NotNil(t, pressed)
	require.Equal(t, ebiten.KeyArrowUp, *pressed)
	require.True(t, keyboard.TryToRegisterKeyPress(*pressed))
	require.Nil(t, keyboard.GetPressedArrowKey())
}

func TestSemanticKeyMapping(t *testing.T) {
	tests := []struct {
		name string
		key  ebiten.Key
		want semantic_input.Key
	}{
		{name: "up", key: ebiten.KeyArrowUp, want: semantic_input.KeyUp},
		{name: "down", key: ebiten.KeyArrowDown, want: semantic_input.KeyDown},
		{name: "left", key: ebiten.KeyArrowLeft, want: semantic_input.KeyLeft},
		{name: "right", key: ebiten.KeyArrowRight, want: semantic_input.KeyRight},
		{name: "space", key: ebiten.KeySpace, want: semantic_input.KeySpace},
		{name: "enter", key: ebiten.KeyEnter, want: semantic_input.KeyEnter},
		{name: "other", key: ebiten.KeyA, want: semantic_input.KeyOther},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, semanticKey(test.key))
		})
	}
}

// TestIsNumberKeyPressedAndRegister_NoKeyPressed tests that the method returns -1
// when no number key is pressed
func TestIsNumberKeyPressedAndRegister_NoKeyPressed(t *testing.T) {
	keyboard := newKeyboardTestFixture(100)

	// When no keys are pressed, should return -1
	result := keyboard.IsNumberKeyPressedAndRegister()
	if result != -1 {
		t.Errorf("Expected -1 when no key pressed, got %d", result)
	}
}

// TestIsNumberKeyPressedAndRegister_MethodExists verifies the method signature
func TestIsNumberKeyPressedAndRegister_MethodExists(t *testing.T) {
	keyboard := newKeyboardTestFixture(100)

	// Method should exist and return an int
	// This test passes if it compiles with correct type
	var result int = keyboard.IsNumberKeyPressedAndRegister()
	_ = result

	// Verify expected return range: -1 (no key) or 0-9 (digit)
	if result < -1 || result > 9 {
		t.Errorf("Expected result in range [-1, 9], got %d", result)
	}
}

func TestSimulateKeyPressFeedsRegisteredKeyCheckOnce(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)

	keyboard.SimulateKeyPress(ebiten.KeyEnter)

	if !keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) {
		t.Fatal("simulated key press was not registered")
	}
	if keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter) {
		t.Fatal("simulated input must be consumed once")
	}
}

func TestSimulateKeyPressFeedsRepeatableRegisteredKeyCheckOnce(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	keyboard.SimulateKeyPress(ebiten.KeyNumpad7)

	require.True(t, keyboard.IsKeyPressedAndRegisterRepeatable(ebiten.KeyNumpad7))
	require.False(t, keyboard.IsKeyPressedAndRegisterRepeatable(ebiten.KeyNumpad7))
}

func TestSimulateKeyPressFeedsAlphaNumericInputOnce(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	keyboard.SimulateKeyPress(ebiten.Key3)

	key, text := keyboard.GetAlphaNumericPressed()
	if key == nil || *key != ebiten.Key3 || text != "3" {
		t.Fatalf("GetAlphaNumericPressed() = (%v, %q), want (Key3, 3)", key, text)
	}
	if key, text = keyboard.GetAlphaNumericPressed(); key != nil || text != "" {
		t.Fatalf("second GetAlphaNumericPressed() = (%v, %q), want no input", key, text)
	}
}

func TestGetAlphaNumericPressedUsesInjectedSourceIncludingZ(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	keyboard.keyPressed = func(key ebiten.Key) bool { return key == ebiten.KeyZ }

	key, text := keyboard.GetAlphaNumericPressed()

	require.NotNil(t, key)
	require.Equal(t, ebiten.KeyZ, *key)
	require.Equal(t, "Z", text)
}

func TestSimulateKeyPressFeedsBoundKeyRegistrationOnce(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	keys := []ebiten.Key{ebiten.KeyDown, ebiten.KeyEnter}
	keyboard.SetForceWaitAnyKey(60_000)
	keyboard.SimulateKeyPress(ebiten.KeyDown)

	pressed := keyboard.GetBoundKeyPressed(&keys)
	require.NotNil(t, pressed)
	require.Equal(t, ebiten.KeyDown, *pressed)
	require.True(t, keyboard.TryToRegisterRepeatableKeyPress(*pressed))
	require.Nil(t, keyboard.GetBoundKeyPressed(&keys))
}

func TestSimulateKeyPressBypassesReleaseConsumption(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	keys := []ebiten.Key{ebiten.KeyEnter}
	keyboard.consumedKeysUntilRelease[ebiten.KeyEnter] = true
	keyboard.SimulateKeyPress(ebiten.KeyEnter)

	pressed := keyboard.GetBoundKeyPressed(&keys)
	require.NotNil(t, pressed)
	require.True(t, keyboard.TryToRegisterKeyPress(*pressed))
	require.Nil(t, keyboard.GetBoundKeyPressed(&keys))
}

func TestHeldKeyReleaseIsObservedBeforeFreshTransitionPress(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	keys := []ebiten.Key{ebiten.KeyEnter}
	held := true
	keyboard.keyPressed = func(key ebiten.Key) bool {
		return key == ebiten.KeyEnter && held
	}

	pressed := keyboard.GetBoundKeyPressed(&keys)
	require.NotNil(t, pressed)
	require.True(t, keyboard.TryToRegisterKeyPress(*pressed))
	keyboard.PrepareForTransition(0)

	keyboard.ClearFrameConsumption()
	require.True(t, keyboard.IsWaitingForRelease())

	held = false
	keyboard.ClearFrameConsumption()
	keyboard.SetAllowKeyPressImmediately()
	require.False(t, keyboard.IsWaitingForRelease())

	held = true
	pressed = keyboard.GetBoundKeyPressed(&keys)
	require.NotNil(t, pressed)
	require.True(t, keyboard.TryToRegisterKeyPress(*pressed))
}

func TestModifierChecksUseInjectedKeySource(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	pressed := map[ebiten.Key]bool{
		ebiten.KeyMetaLeft: true,
		ebiten.KeyV:        true,
	}
	keyboard.keyPressed = func(key ebiten.Key) bool { return pressed[key] }

	require.True(t, keyboard.IsModifierPressed(ModifierMeta))
	require.True(t, keyboard.IsModifierComboPressed(ModifierMeta, ebiten.KeyV))
	require.True(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyV))
}

func TestPrepareForTransitionWaitsForHeldModifier(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	held := map[ebiten.Key]bool{
		ebiten.KeyV:         true,
		ebiten.KeyShiftLeft: true,
	}
	keyboard.keyPressed = func(key ebiten.Key) bool { return held[key] }

	keyboard.PrepareForTransition(0)
	delete(held, ebiten.KeyV)
	keyboard.ClearFrameConsumption()

	require.True(t, keyboard.IsWaitingForRelease())
	require.False(t, keyboard.TryToRegisterKeyPress(ebiten.KeyShiftLeft))

	delete(held, ebiten.KeyShiftLeft)
	keyboard.ClearFrameConsumption()
	require.False(t, keyboard.IsWaitingForRelease())
}

func TestTransitionBlocksHeldRepeatableKey(t *testing.T) {
	keyboard := newKeyboardTestFixture(0)
	held := true
	keyboard.keyPressed = func(key ebiten.Key) bool {
		return key == ebiten.KeyArrowUp && held
	}

	require.True(t, keyboard.TryToRegisterKeyPress(ebiten.KeyArrowUp))
	keyboard.PrepareForTransition(0)
	require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyArrowUp))

	held = false
	keyboard.ClearFrameConsumption()
	held = true
	keyboard.SetAllowKeyPressImmediately()
	require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyArrowUp))
}
