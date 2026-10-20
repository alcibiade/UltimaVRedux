package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestCapturedEdgesDoNotStarveReplacementBehindHeldKey(t *testing.T) {
	held := map[ebiten.Key]bool{}
	keyboard := NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held[key] })
	held[ebiten.KeyG] = true
	first := keyboard.CaptureKeySample()
	require.True(t, keyboard.ConsumePressEdge(first, ebiten.KeyG))
	held[ebiten.KeyO] = true
	second := keyboard.CaptureKeySample()
	require.False(t, second.Pressed[ebiten.KeyG])
	require.True(t, keyboard.ConsumePressEdge(second, ebiten.KeyO), "no movement throttle on replacement edges")
	require.False(t, keyboard.ConsumePressEdge(second, ebiten.KeyG))
}

func TestSuppressionSurvivesCancellationAndBlocksRepeatUntilPhysicalRelease(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyRight, ebiten.KeyNumpad6, ebiten.KeySpace} {
		t.Run(key.String(), func(t *testing.T) {
			held := map[ebiten.Key]bool{key: true}
			keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
			keyboard.SuppressUntilRelease(key)
			keyboard.ResetReleaseDebouncing()
			keyboard.SetAllowKeyPressImmediately()
			keyboard.ClearFrameConsumption()
			require.False(t, keyboard.TryToRegisterRepeatableKeyPress(key))
			require.False(t, keyboard.TryAcceptKey(key))
			require.False(t, keyboard.TryToRegisterKeyPress(key))
			require.Nil(t, keyboard.GetBoundKeyPressed(&[]ebiten.Key{key}))
			require.False(t, keyboard.HasUnprotectedHeldKey())
			if key != ebiten.KeyNumpad6 {
				require.True(t, keyboard.IsAnyKeyPressed(), "raw waits still see the physical hold")
			}
			delete(held, key)
			keyboard.CaptureKeySample()
			held[key] = true
			require.True(t, keyboard.TryToRegisterRepeatableKeyPress(key))
		})
	}
}

func TestSuspendedKeyboardMasksInputWithoutManufacturingPhysicalRelease(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyNumpadEnter: true, ebiten.KeyRight: true}
	keyboard := NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	keyboard.SuppressUntilRelease(ebiten.KeyRight)
	keyboard.SimulateKeyPress(ebiten.KeyG)
	restore := keyboard.SuspendInputPolling()
	require.False(t, keyboard.IsAnyKeyPressed())
	require.False(t, keyboard.IsKeyHeld(ebiten.KeyNumpadEnter))
	require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight))
	require.False(t, keyboard.TryToRegisterKeyPress(ebiten.KeyNumpadEnter))
	require.Nil(t, keyboard.GetBoundKeyPressed(&[]ebiten.Key{ebiten.KeyG, ebiten.KeyNumpadEnter}))
	keyboard.ClearFrameConsumption()
	require.True(t, keyboard.IsSuppressedUntilRelease(ebiten.KeyRight), "masked input must not clear a physical hold")
	keyboard.SimulateKeyPress(ebiten.KeyO)
	restore()
	require.False(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyG), "pre-focus synthetic input is discarded")
	require.False(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyO), "suspended synthetic input is discarded")
	require.True(t, keyboard.IsKeyHeld(ebiten.KeyNumpadEnter))
	require.False(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter), "even keys outside gameplay bindings require release")
	clear(held)
	keyboard.ClearFrameConsumption()
	held[ebiten.KeyNumpadEnter] = true
	require.True(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyNumpadEnter))
}
