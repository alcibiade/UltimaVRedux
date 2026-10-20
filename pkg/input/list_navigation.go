package input

import (
	"time"

	beradinput "github.com/bradhannah/berad-sdk/modules/berad-input"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	listInitialDelay = 300 * time.Millisecond
	listRepeatDelay  = 120 * time.Millisecond
)

// ListNavigation resolves UI chords before consuming their physical keys. Its
// zero value is ready to use. Poll once per focused Update; Reset on focus loss
// or a change of input mode. Timing advances in update ticks, never wall time.
type ListNavigation struct {
	epoch   uint64
	barrier beradinput.ReleaseBarrier[ebiten.Key]
	repeat  beradinput.Repeat[ebiten.Key]
}

// Reset discards a previous hold, for example when a nested modal takes focus.
func (n *ListNavigation) Reset() {
	var reset ListNavigation
	*n = reset
}

// BlockUntilRelease prevents a hold crossing a modal focus boundary.
func (n *ListNavigation) BlockUntilRelease(pressed func(ebiten.Key) bool) {
	n.Reset()
	n.barrier.Capture([]ebiten.Key{
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight,
		ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyHome, ebiten.KeyEnd,
	}, pressed)
}

// Poll returns a normalized navigation key when movement is due. owned remains
// true during throttled holds, so text input cannot consume/reset navigation.
// By default Up, Down, PageUp and PageDown are read. Explicit keys allow readers
// to include Home/End and menus to include Left/Right. Shift+Up/Down always maps
// to PageUp/PageDown; callers without paging deliberately ignore those intents.
func (n *ListNavigation) Poll(k *Keyboard, keys ...ebiten.Key) (*ebiten.Key, bool) {
	return n.PollWithKeyState(k, k.isKeyPressed, keys...)
}

// PollWithKeyState supports widgets whose existing physical-input adapter also
// owns their release gates. Production widgets normally use Poll.
func (n *ListNavigation) PollWithKeyState(
	keyboard *Keyboard, pressed func(ebiten.Key) bool, keys ...ebiten.Key,
) (*ebiten.Key, bool) {
	if n.epoch != keyboard.navigationEpoch {
		n.repeat.Reset()
		n.epoch = keyboard.navigationEpoch
	}
	if len(keys) == 0 {
		keys = []ebiten.Key{ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyPageUp, ebiten.KeyPageDown}
	}
	for _, key := range keys {
		if keyboard.consumeSimulatedKey(key) {
			n.barrier.Reset()
			n.holdReady(key, false)
			keyboard.keys.Consume(key)
			resolved := listNavigationKey(key, pressed)

			return &resolved, true
		}
	}
	selection := beradinput.NewBindings(keys...).SelectSingle(pressed)
	key := selection.Key
	if selection.Held && keyboard.IsSuppressedUntilRelease(key) {
		n.Reset()

		return nil, true
	}
	if !n.barrier.Observe(pressed) {
		return nil, true
	}
	if !selection.Held || selection.Conflicting {
		n.Reset()

		return nil, selection.Held
	}
	// Preserve the keyboard's existing transition gate, but do not inherit its
	// gameplay repeat delay or force-next flag. Fresh list taps are immediate.
	if !keyboard.keys.Ready(key, false) || !keyboard.cadence.Ready(keyboard.elapsedMilliseconds()) {
		n.Reset()

		return nil, true
	}
	if !n.holdReady(key, keyboard.keys.Consumed(key)) {
		return nil, true
	}
	keyboard.keys.Consume(key)
	resolved := listNavigationKey(key, pressed)

	return &resolved, true
}

// holdReady accepts a fresh hold immediately, then advances repeats by one update tick.
func (n *ListNavigation) holdReady(key ebiten.Key, consumed bool) bool {
	ticks := ebiten.TPS()
	if ticks <= 0 {
		ticks = ebiten.DefaultTPS
	}
	result := n.repeat.Poll(beradinput.Selection[ebiten.Key]{Key: key, Held: true, Conflicting: false},
		!consumed, time.Second/time.Duration(ticks), beradinput.RepeatTiming{
			Initial: listInitialDelay, Interval: listRepeatDelay, Tolerance: time.Microsecond,
		})

	return result.Triggered
}

func listNavigationKey(key ebiten.Key, pressed func(ebiten.Key) bool) ebiten.Key {
	shift := pressed(ebiten.KeyShift) || pressed(ebiten.KeyShiftLeft) || pressed(ebiten.KeyShiftRight)
	if shift {
		switch key { //nolint:exhaustive // Only vertical arrows have paging aliases.
		case ebiten.KeyUp:
			return ebiten.KeyPageUp
		case ebiten.KeyDown:
			return ebiten.KeyPageDown
		}
	}

	return key
}
