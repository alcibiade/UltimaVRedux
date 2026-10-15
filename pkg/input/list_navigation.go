package input

import (
	"time"

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
	epoch       uint64
	blockedKeys []ebiten.Key
	key         ebiten.Key
	active      bool
	remaining   time.Duration
}

// Reset discards a previous hold, for example when a nested modal takes focus.
func (n *ListNavigation) Reset() {
	var reset ListNavigation
	*n = reset
}

// BlockUntilRelease prevents a hold crossing a modal focus boundary.
func (n *ListNavigation) BlockUntilRelease(pressed func(ebiten.Key) bool) {
	n.Reset()
	for _, key := range [...]ebiten.Key{
		ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyLeft, ebiten.KeyRight,
		ebiten.KeyPageUp, ebiten.KeyPageDown, ebiten.KeyHome, ebiten.KeyEnd,
	} {
		if pressed(key) {
			n.blockedKeys = append(n.blockedKeys, key)
		}
	}
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
		n.active = false
		n.epoch = keyboard.navigationEpoch
	}
	if len(keys) == 0 {
		keys = []ebiten.Key{ebiten.KeyUp, ebiten.KeyDown, ebiten.KeyPageUp, ebiten.KeyPageDown}
	}
	for _, key := range keys {
		if keyboard.consumeSimulatedKey(key) {
			n.start(key)
			keyboard.consumedKeysUntilRelease[key] = true
			resolved := listNavigationKey(key, pressed)

			return &resolved, true
		}
	}
	key, held, conflicting := listHeldKey(pressed, keys)
	if held && keyboard.IsSuppressedUntilRelease(key) {
		n.Reset()

		return nil, true
	}
	for _, blocked := range n.blockedKeys {
		if pressed(blocked) {
			return nil, true
		}
	}
	n.blockedKeys = nil
	if !held || conflicting {
		n.Reset()

		return nil, held
	}
	// Preserve the keyboard's existing transition gate, but do not inherit its
	// gameplay repeat delay or force-next flag. Fresh list taps are immediate.
	if !keyboard.listNavigationReady() {
		n.Reset()

		return nil, true
	}
	if !n.holdReady(key, keyboard.consumedKeysUntilRelease[key]) {
		return nil, true
	}
	keyboard.consumedKeysUntilRelease[key] = true
	resolved := listNavigationKey(key, pressed)

	return &resolved, true
}

// holdReady accepts a fresh hold immediately, then advances repeats by one update tick.
func (n *ListNavigation) holdReady(key ebiten.Key, consumed bool) bool {
	if !n.active || n.key != key || !consumed {
		n.start(key)

		return true
	}
	ticks := ebiten.TPS()
	if ticks <= 0 {
		ticks = ebiten.DefaultTPS
	}
	n.remaining -= time.Second / time.Duration(ticks)
	if n.remaining > time.Microsecond {
		return false
	}
	n.remaining += listRepeatDelay

	return true
}

func (n *ListNavigation) start(key ebiten.Key) {
	n.blockedKeys = nil
	n.key, n.active, n.remaining = key, true, listInitialDelay
}

func listHeldKey(pressed func(ebiten.Key) bool, keys []ebiten.Key) (ebiten.Key, bool, bool) {
	var selected ebiten.Key
	held := false
	for _, key := range keys {
		if !pressed(key) {
			continue
		}
		if held {
			return selected, true, true
		}
		selected, held = key, true
	}

	return selected, held, false
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

func (k *Keyboard) listNavigationReady() bool {
	if k.inputPollingSuspended {
		return false
	}
	if k.waitingForRelease {
		if !k.areAllKeysReleased() {
			return false
		}
		k.waitingForRelease = false
		k.keysToRelease = nil
	}

	return k.elapsedMilliseconds() >= k.timeToAllowNextKeypress
}
