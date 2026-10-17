package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestCapsLockDoesNotOwnControlInput(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyCapsLock: true}
	keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	keyboard.PrepareForTransition(0)
	keyboard.ClearFrameConsumption()
	require.False(t, keyboard.IsWaitingForRelease())
	require.False(t, keyboard.IsKeyHeld(ebiten.KeyCapsLock))
	require.False(t, (*Keyboard)(nil).IsKeyHeld(ebiten.KeyCapsLock), "nil fallback must not poll the host lock key")
	require.False(t, keyboard.HasUnprotectedHeldKey())
	require.False(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyCapsLock))
	var allKeys []ebiten.Key
	for key := ebiten.KeyA; key <= ebiten.KeyMax; key++ {
		allKeys = append(allKeys, key)
	}
	require.Nil(t, keyboard.GetBoundKeyPressed(&allKeys), "lock state cannot acknowledge an all-key prompt")
	sample := keyboard.CaptureKeySample()
	require.False(t, sample.Held[ebiten.KeyCapsLock])
	require.False(t, sample.Pressed[ebiten.KeyCapsLock])
	require.False(t, sample.HasModifiers())
}

func TestCapsLockRetainsRealKeyReleaseProtection(t *testing.T) {
	for _, shift := range []ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyShiftRight} {
		t.Run(shift.String(), func(t *testing.T) {
			held := map[ebiten.Key]bool{ebiten.KeyCapsLock: true, ebiten.KeyEnter: true, shift: true}
			keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
			keyboard.PrepareForTransition(0)
			require.False(t, keyboard.TryAcceptKey(ebiten.KeyEnter))
			delete(held, ebiten.KeyEnter)
			keyboard.ClearFrameConsumption()
			require.True(t, keyboard.IsWaitingForRelease(), "physical Shift must still release")
			delete(held, shift)
			keyboard.ClearFrameConsumption()
			require.False(t, keyboard.IsWaitingForRelease(), "Caps Lock may remain enabled")
			for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyM} {
				held[key] = true
				keyboard.SetAllowKeyPressImmediately()
				require.True(t, keyboard.IsKeyPressedAndRegister(key))
				require.False(t, keyboard.IsKeyPressedAndRegister(key))
				delete(held, key)
				keyboard.ClearFrameConsumption()
			}
			held[ebiten.KeyDown] = true
			keyboard.SetAllowKeyPressImmediately()
			require.True(t, keyboard.IsKeyPressedAndRegisterRepeatable(ebiten.KeyDown))
		})
	}
}

func TestCapsLockFocusReturnAndNavigation(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyCapsLock: true, ebiten.KeyDown: true}
	keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	restore := keyboard.SuspendInputPolling()
	restore()
	require.False(t, keyboard.IsSuppressedUntilRelease(ebiten.KeyCapsLock))
	require.True(t, keyboard.IsSuppressedUntilRelease(ebiten.KeyDown))
	var navigation ListNavigation
	key, _ := navigation.Poll(keyboard)
	require.Nil(t, key)
	delete(held, ebiten.KeyDown)
	keyboard.ClearFrameConsumption()
	navigation.Poll(keyboard)
	keyboard.PrepareForTransition(0)
	keyboard.ClearFrameConsumption()
	held[ebiten.KeyDown] = true
	key, _ = navigation.Poll(keyboard)
	require.NotNil(t, key)
	require.Equal(t, ebiten.KeyDown, *key)
	for range 18 {
		key, _ = navigation.Poll(keyboard)
	}
	require.NotNil(t, key, "held navigation still repeats")
	delete(held, ebiten.KeyDown)
	keyboard.ClearFrameConsumption()
	held[ebiten.KeyEnter] = true
	require.True(t, keyboard.TryAcceptKey(ebiten.KeyEnter))
}
