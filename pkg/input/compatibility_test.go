package input

import (
	"strconv"
	"testing"

	beradinput "github.com/bradhannah/berad-sdk/modules/berad-input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestLegacyKeyboardCadenceAndSetterContracts(t *testing.T) {
	for _, delay := range []int{0, 120, 165, 250} {
		t.Run(strconv.Itoa(delay), func(t *testing.T) {
			now := int64(0)
			keyboard := NewKeyboardWithKeyStateSource(delay, func(ebiten.Key) bool { return true })
			keyboard.SetElapsedMillisecondsSource(func() int64 { return now })
			require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight))
			now += int64(delay)
			require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight))
			now++
			require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight))
			require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyLeft), "different key bypasses shared interval")
			keyboard.SetForceWaitAnyKey(100)
			keyboard.SetAllowKeyPressImmediately()
			now += 99
			require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyUp), "force cannot bypass all-key wait")
			now++
			require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyLeft))
			keyboard.SetLastKeyPressedNowPlusMs(500)
			require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyLeft))
			require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyUp), "postponed last time is not an all-key wait")
			require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyEnter))
			require.Zero(t, keyboard.GetMsSinceLastKeyPress())
			require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyEnter))
		})
	}
	t.Run("clock replacement resets timing but preserves release gates", func(t *testing.T) {
		held := true
		keyboard := NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held && key == ebiten.KeyRight })
		keyboard.SetElapsedMillisecondsSource(func() int64 { return 1000 })
		require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight))
		keyboard.SetForceWaitAnyKey(60_000)
		keyboard.SetLastKeyPressedNowPlusMs(60_000)
		keyboard.PrepareForTransition(0)
		keyboard.SetElapsedMillisecondsSource(func() int64 { return 0 })
		require.Zero(t, keyboard.GetMsSinceLastKeyPress())
		require.True(t, keyboard.IsWaitingForRelease())
		require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight))
		held = false
		keyboard.ClearFrameConsumption()
		held = true
		require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight), "the same key is immediate on the new clock")
	})
}

func TestLegacyDummyAcceptanceRetainsPostDelaySideEffect(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyEnter: true, ebiten.KeyShiftLeft: true}
	now := int64(1000)
	keyboard := NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held[key] })
	keyboard.SetElapsedMillisecondsSource(func() int64 { return now })
	keyboard.PrepareForTransition(50)
	delete(held, ebiten.KeyEnter)
	require.False(t, keyboard.TryAcceptKey(0))
	delete(held, ebiten.KeyShiftLeft)
	require.True(t, keyboard.TryAcceptKey(0), "dummy acceptance services release and updates timing")
	require.False(t, keyboard.IsWaitingForRelease())
	require.Zero(t, keyboard.GetMsSinceLastKeyPress())
	keyboard.SetAllowKeyPressImmediately()
	require.False(t, keyboard.TryToRegisterKeyPress(ebiten.KeyG), "post-accept delay gates other keys too")
	now += 50
	require.True(t, keyboard.TryToRegisterKeyPress(ebiten.KeyG))
	require.False(t, keyboard.TryToRegisterKeyPress(ebiten.KeyG))
	require.True(t, keyboard.TryAcceptKey(ebiten.KeyA), "TryAccept does not claim one-shot consumption")
}

func TestLegacyOrdinaryAcceptanceDoesNotApplyPendingPostDelay(t *testing.T) {
	keyboard := NewKeyboardWithKeyStateSource(165, func(ebiten.Key) bool { return false })
	keyboard.SetElapsedMillisecondsSource(func() int64 { return 1000 })
	keyboard.PrepareForTransition(50)
	require.True(t, keyboard.TryToRegisterKeyPress(ebiten.KeyEnter))
	require.True(t, keyboard.TryToRegisterKeyPress(ebiten.KeyG))
	require.True(t, keyboard.TryAcceptKey(0))
	require.False(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyUp))
}

func TestLegacyBindingFirstHeldDoesNotSkipConsumedKey(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyG: true, ebiten.KeyO: true}
	keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	bindings := beradinput.NewBindings(ebiten.KeyG, ebiten.KeyO)
	require.True(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyG))
	require.Equal(t, ebiten.KeyG, *keyboard.GetRegisteredKeyPressed(bindings))
	require.False(t, keyboard.TryToRegisterKeyPress(ebiten.KeyG))
	keyboard.SuppressUntilRelease(ebiten.KeyG)
	require.Equal(t, ebiten.KeyO, *keyboard.GetRegisteredKeyPressed(bindings))
}

func TestLegacySimulationRoutesKeepDistinctBypasses(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyEnter: true}
	keyboard := NewKeyboardWithKeyStateSource(165, func(key ebiten.Key) bool { return held[key] })
	keyboard.SetElapsedMillisecondsSource(func() int64 { return 1000 })
	keyboard.PrepareForTransition(50)
	keyboard.SetForceWaitAnyKey(60_000)
	keyboard.SuppressUntilRelease(ebiten.KeyEnter)
	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.False(t, keyboard.TryToRegisterKeyPress(ebiten.KeyEnter), "registration checks strong suppression before simulation")
	require.True(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter), "combined historical helper bypasses those gates")
	require.False(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter))
	keyboard.SimulateKeyPress(ebiten.KeyUp)
	require.True(t, keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyUp), "simulation bypasses transition and time gate")
	require.Equal(t, 1000, keyboard.GetMsSinceLastKeyPress(), "simulation does not record cadence")
	keyboard.SimulateKeyPress(ebiten.Key3)
	key, text := keyboard.GetAlphaNumericPressed()
	require.Equal(t, ebiten.Key3, *key)
	require.Equal(t, "3", text)
	require.False(t, keyboard.TryToRegisterKeyPress(*key), "alpha simulation consumes before ordinary registration")
	require.Nil(t, keyboard.GetBoundKeyPressed(&[]ebiten.Key{ebiten.Key3}))
}

func TestActivationBarrierBeforeAnySampleIncludesUnboundKeys(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyNumpadEnter: true, ebiten.KeyShiftRight: true}
	keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	keyboard.PrepareForTransition(0)
	delete(held, ebiten.KeyNumpadEnter)
	keyboard.ClearFrameConsumption()
	require.True(t, keyboard.IsWaitingForRelease())
	delete(held, ebiten.KeyShiftRight)
	keyboard.ClearFrameConsumption()
	held[ebiten.KeyEnter] = true
	require.True(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter))
	require.False(t, keyboard.IsKeyPressedAndRegister(ebiten.KeyEnter), "a second callback cannot accept it")
}
