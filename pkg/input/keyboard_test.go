package input

import (
	"slices"
	"testing"

	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func newKeyboardTestFixture(delay int) *Keyboard {
	return NewKeyboardWithKeyStateSource(delay, func(ebiten.Key) bool { return false })
}

func TestGetAlphaNumericPressedPhysicalKeys(t *testing.T) {
	for _, test := range []struct {
		name string
		held []ebiten.Key
		want ebiten.Key
		text string
	}{
		{name: "none"},
		{name: "digit nine", held: []ebiten.Key{ebiten.Key9}, want: ebiten.Key9, text: "9"},
		{name: "last letter", held: []ebiten.Key{ebiten.KeyZ}, want: ebiten.KeyZ, text: "Z"},
		{name: "numpad is not alphanumeric input", held: []ebiten.Key{ebiten.KeyNumpad9}},
		{name: "letter before digit", held: []ebiten.Key{ebiten.Key9, ebiten.KeyZ}, want: ebiten.KeyZ, text: "Z"},
		{name: "alphabetic order", held: []ebiten.Key{ebiten.KeyZ, ebiten.KeyA}, want: ebiten.KeyA, text: "A"},
		{name: "digit order", held: []ebiten.Key{ebiten.Key9, ebiten.Key0}, want: ebiten.Key0, text: "0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return slices.Contains(test.held, key) })
			key, text := keyboard.GetAlphaNumericPressed()
			require.Equal(t, test.text, text)
			if test.text == "" {
				require.Nil(t, key)
				return
			}
			require.NotNil(t, key)
			require.Equal(t, test.want, *key)
		})
	}
}

func TestGetPressedArrowKeyUsesInjectedSourceAndPreservesPriority(t *testing.T) {
	tests := []struct {
		name    string
		pressed map[ebiten.Key]bool
		wantKey ebiten.Key
	}{
		{name: "down", pressed: map[ebiten.Key]bool{ebiten.KeyArrowDown: true}, wantKey: ebiten.KeyArrowDown},
		{name: "up", pressed: map[ebiten.Key]bool{ebiten.KeyArrowUp: true}, wantKey: ebiten.KeyArrowUp},
		{name: "right", pressed: map[ebiten.Key]bool{ebiten.KeyArrowRight: true}, wantKey: ebiten.KeyArrowRight},
		{name: "left", pressed: map[ebiten.Key]bool{ebiten.KeyArrowLeft: true}, wantKey: ebiten.KeyArrowLeft},
		{name: "up before right", pressed: map[ebiten.Key]bool{ebiten.KeyArrowUp: true, ebiten.KeyArrowRight: true}, wantKey: ebiten.KeyArrowUp},
		{name: "right before left", pressed: map[ebiten.Key]bool{ebiten.KeyArrowRight: true, ebiten.KeyArrowLeft: true}, wantKey: ebiten.KeyArrowRight},
		{
			name: "multiple keys use original priority",
			pressed: map[ebiten.Key]bool{
				ebiten.KeyArrowDown:  true,
				ebiten.KeyArrowUp:    true,
				ebiten.KeyArrowRight: true,
				ebiten.KeyArrowLeft:  true,
			},
			wantKey: ebiten.KeyArrowDown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return test.pressed[key] })

			pressed := keyboard.GetPressedArrowKey()
			require.NotNil(t, pressed)
			require.Equal(t, test.wantKey, *pressed)
		})
	}
	require.Nil(t, newKeyboardTestFixture(0).GetPressedArrowKey())
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

func TestGetPressedKeyPhysicalPriorityAndSemanticMapping(t *testing.T) {
	tests := []struct {
		name string
		held []ebiten.Key
		want semantic_input.Key
	}{
		{name: "up", held: []ebiten.Key{ebiten.KeyArrowUp}, want: semantic_input.KeyUp},
		{name: "down", held: []ebiten.Key{ebiten.KeyArrowDown}, want: semantic_input.KeyDown},
		{name: "left", held: []ebiten.Key{ebiten.KeyArrowLeft}, want: semantic_input.KeyLeft},
		{name: "right", held: []ebiten.Key{ebiten.KeyArrowRight}, want: semantic_input.KeyRight},
		{name: "arrow before letter and control", held: []ebiten.Key{ebiten.KeyZ, ebiten.KeySpace, ebiten.KeyArrowRight}, want: semantic_input.KeyRight},
		{name: "letter before control", held: []ebiten.Key{ebiten.KeySpace, ebiten.KeyZ}, want: semantic_input.KeyOther},
		{name: "digit nine before control", held: []ebiten.Key{ebiten.KeySpace, ebiten.Key9}, want: semantic_input.KeyOther},
		{name: "space before enter", held: []ebiten.Key{ebiten.KeyEnter, ebiten.KeySpace}, want: semantic_input.KeySpace},
		{name: "enter before escape", held: []ebiten.Key{ebiten.KeyEscape, ebiten.KeyEnter}, want: semantic_input.KeyEnter},
		{name: "escape", held: []ebiten.Key{ebiten.KeyEscape}, want: semantic_input.KeyOther},
		{name: "tab", held: []ebiten.Key{ebiten.KeyTab}, want: semantic_input.KeyOther},
		{name: "backspace", held: []ebiten.Key{ebiten.KeyBackspace}, want: semantic_input.KeyOther},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return slices.Contains(test.held, key) })
			key := keyboard.GetPressedKey()
			require.NotNil(t, key)
			require.Equal(t, test.want, *key)
		})
	}
	keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return key == ebiten.KeyNumpad9 })
	require.Nil(t, keyboard.GetPressedKey(), "keypad digits are not sailing interruption keys")
	require.Nil(t, newKeyboardTestFixture(0).GetPressedKey())
}

func TestIsNumberKeyPressedAndRegisterPhysicalKeys(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyNumpad9: true, ebiten.KeyZ: true}
	keyboard := NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held[key] })
	keyboard.SetElapsedMillisecondsSource(func() int64 { return 0 })
	require.Equal(t, -1, keyboard.IsNumberKeyPressedAndRegister(), "only top-row digits qualify")
	held[ebiten.Key9], held[ebiten.Key0] = true, true
	require.Equal(t, 0, keyboard.IsNumberKeyPressedAndRegister(), "lower digits have priority")
	require.Equal(t, 9, keyboard.IsNumberKeyPressedAndRegister(), "consumed digits do not hide another fresh digit")
	require.Equal(t, -1, keyboard.IsNumberKeyPressedAndRegister(), "held digits are consumed once")
	clear(held)
	keyboard.ClearFrameConsumption()
	require.Equal(t, -1, keyboard.IsNumberKeyPressedAndRegister())
	held[ebiten.Key9] = true
	keyboard.SetAllowKeyPressImmediately()
	require.Equal(t, 9, keyboard.IsNumberKeyPressedAndRegister(), "release rearms the physical digit")
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
	keyboard.keys.Consume(ebiten.KeyEnter)
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
