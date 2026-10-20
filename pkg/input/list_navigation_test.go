package input

import (
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestListNavigationShiftAliasesConsumePhysicalArrow(t *testing.T) {
	for _, shift := range []ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyShiftRight} {
		for _, arrow := range []ebiten.Key{ebiten.KeyUp, ebiten.KeyDown} {
			t.Run(shift.String()+arrow.String(), func(t *testing.T) {
				held := map[ebiten.Key]bool{shift: true, arrow: true}
				keyboard := NewKeyboardWithKeyStateSource(1000, func(key ebiten.Key) bool { return held[key] })
				var nav ListNavigation
				want := ebiten.KeyPageDown
				if arrow == ebiten.KeyUp {
					want = ebiten.KeyPageUp
				}
				key, owned := nav.Poll(keyboard)
				require.True(t, owned)
				require.NotNil(t, key)
				require.Equal(t, want, *key)
				require.False(t, keyboard.IsKeyPressedAndRegister(arrow), "plain arrow must already be consumed")
			})
		}
	}
}

func TestListNavigationRepeatUsesUpdateTPS(t *testing.T) {
	for _, test := range []struct {
		tps, initialTicks, repeatTicks int
	}{
		{tps: 60, initialTicks: 18, repeatTicks: 8},
		{tps: 30, initialTicks: 9, repeatTicks: 4},
	} {
		t.Run(strconv.Itoa(test.tps), func(t *testing.T) {
			previousTPS := ebiten.TPS()
			ebiten.SetTPS(test.tps)
			t.Cleanup(func() { ebiten.SetTPS(previousTPS) })
			keyboard := NewKeyboardWithKeyStateSource(1000, func(key ebiten.Key) bool { return key == ebiten.KeyDown })
			keyboard.SetElapsedMillisecondsSource(func() int64 { return 0 })
			var nav ListNavigation
			key, owned := nav.Poll(keyboard)
			require.True(t, owned)
			require.NotNil(t, key, "a fresh list tap ignores the gameplay interval")
			require.Equal(t, ebiten.KeyDown, *key)
			for _, ticks := range []int{test.initialTicks, test.repeatTicks} {
				for range ticks - 1 {
					keyboard.SetAllowKeyPressImmediately()
					key, owned = nav.Poll(keyboard)
					require.True(t, owned, "throttled navigation retains ownership")
					require.Nil(t, key, "force-next must not accelerate list repeats")
				}
				key, owned = nav.Poll(keyboard)
				require.True(t, owned)
				require.NotNil(t, key)
				require.Equal(t, ebiten.KeyDown, *key)
			}
		})
	}
}

func TestListNavigationFreshTapsConflictAndFocus(t *testing.T) {
	held := map[ebiten.Key]bool{}
	keyboard := NewKeyboardWithKeyStateSource(1000, func(key ebiten.Key) bool { return held[key] })
	var nav ListNavigation
	for range 3 {
		held[ebiten.KeyDown] = true
		key, _ := nav.Poll(keyboard)
		require.NotNil(t, key, "fresh taps bypass gameplay delay")
		clear(held)
		key, owned := nav.Poll(keyboard)
		require.Nil(t, key)
		require.False(t, owned)
	}
	held[ebiten.KeyDown], held[ebiten.KeyUp] = true, true
	key, owned := nav.Poll(keyboard)
	require.Nil(t, key)
	require.True(t, owned)
	delete(held, ebiten.KeyUp)
	key, _ = nav.Poll(keyboard)
	require.NotNil(t, key)
	nav.BlockUntilRelease(keyboard.IsKeyHeld)
	keyboard.PrepareForTransition(0)
	for range 60 {
		key, owned = nav.Poll(keyboard)
		require.Nil(t, key)
		require.True(t, owned)
	}
	clear(held)
	keyboard.ClearFrameConsumption()
	nav.Poll(keyboard)
	held[ebiten.KeyDown] = true
	key, _ = nav.Poll(keyboard)
	require.NotNil(t, key)
}

func TestListNavigationPageKeysAndModifierChanges(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyDown: true}
	keyboard := NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return held[key] })
	var nav ListNavigation
	key, _ := nav.Poll(keyboard)
	require.Equal(t, ebiten.KeyDown, *key)
	held[ebiten.KeyShiftRight] = true
	for range 18 {
		key, _ = nav.Poll(keyboard)
	}
	require.NotNil(t, key)
	require.Equal(t, ebiten.KeyPageDown, *key)
	delete(held, ebiten.KeyShiftRight)
	for range 8 {
		key, _ = nav.Poll(keyboard)
	}
	require.NotNil(t, key)
	require.Equal(t, ebiten.KeyDown, *key, "releasing Shift remaps the next repeat without releasing the arrow")
	clear(held)
	nav.Poll(keyboard)
	for _, page := range []ebiten.Key{ebiten.KeyPageUp, ebiten.KeyPageDown} {
		held[page] = true
		key, _ = nav.Poll(keyboard)
		require.NotNil(t, key)
		require.Equal(t, page, *key)
		clear(held)
		nav.Poll(keyboard)
	}
}

func TestListNavigationObservesReleaseBetweenDialogUpdates(t *testing.T) {
	held := map[ebiten.Key]bool{ebiten.KeyDown: true}
	keyboard := NewKeyboardWithKeyStateSource(1000, func(key ebiten.Key) bool { return held[key] })
	var nav ListNavigation
	key, _ := nav.Poll(keyboard)
	require.NotNil(t, key)
	clear(held)
	keyboard.ClearFrameConsumption() // The scene observes release even if this dialog was not updated.
	held[ebiten.KeyDown] = true
	key, _ = nav.Poll(keyboard)
	require.NotNil(t, key, "a new physical tap must not wait for the old hold")
	keyboard.PrepareForTransition(0)
	for range 60 {
		key, owned := nav.Poll(keyboard)
		require.Nil(t, key)
		require.True(t, owned)
	}
	clear(held)
	keyboard.ClearFrameConsumption()
	held[ebiten.KeyDown] = true
	key, _ = nav.Poll(keyboard)
	require.NotNil(t, key)
}
