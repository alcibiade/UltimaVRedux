package widgets

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// quantityInput shares plain horizontal repeat and focus ownership, never quantity
// bounds or transaction rules. Optional keys retain their existing tap behavior;
// they never pass through the list helper's Shift+Up/Down paging normalization.
type quantityInput struct {
	navigation
	keyboard *input.Keyboard
	tapKeys  []ebiten.Key
}

func newQuantityInput(keyboard *input.Keyboard, held func(ebiten.Key) bool, tapKeys ...ebiten.Key) quantityInput {
	state := quantityInput{
		navigation: newNavigation(held),
		keyboard:   keyboard,
		tapKeys:    tapKeys,
	}
	state.onDialogFocusChanged(true)

	return state
}

// DialogStack must notify cover, return, and removal even when the dialog receives
// no Update calls. For window-focus loss, the scene suspends the shared Keyboard;
// its restore suppresses physical holds before selector updates resume. Missing
// updates alone cannot establish a key release. Suppression also protects the next
// input owner from a dismissed modal's hold.
func (q *quantityInput) onDialogFocusChanged(focused bool) {
	q.navigation.onDialogFocusChanged(focused)
	q.suppressQuantityKeys()
	q.keyboard.SuppressUntilRelease(ebiten.KeyEscape)
}

func (q *quantityInput) suppressQuantityKeys() {
	q.keyboard.SuppressUntilRelease(ebiten.KeyLeft)
	q.keyboard.SuppressUntilRelease(ebiten.KeyRight)
	for _, key := range q.tapKeys {
		q.keyboard.SuppressUntilRelease(key)
	}
	q.suppressQuantityConfirmation()
}

func (q *quantityInput) suppressQuantityConfirmation() {
	q.keyboard.SuppressUntilRelease(ebiten.KeyEnter)
	q.keyboard.SuppressUntilRelease(ebiten.KeyNumpadEnter)
}

// pollQuantityInput owns throttled holds as well as accepted steps. Conflicting
// quantity keys are discarded until release, rather than replaying the loser.
// Confirmation pressed during quantity adjustment likewise needs a fresh press.
func (q *quantityInput) pollQuantityInput() (*ebiten.Key, bool) {
	heldCount := 0
	for _, key := range []ebiten.Key{ebiten.KeyLeft, ebiten.KeyRight} {
		if q.keyState(key) {
			heldCount++
		}
	}
	for _, key := range q.tapKeys {
		if q.keyState(key) {
			heldCount++
		}
	}
	if heldCount > 1 || heldCount > 0 && q.quantityModifierHeld() {
		q.Reset()
		q.suppressQuantityKeys()

		return nil, true
	}
	key, owned := q.PollWithKeyState(q.keyboard, q.keyState, ebiten.KeyLeft, ebiten.KeyRight)
	if owned {
		q.suppressQuantityConfirmation()

		return key, true
	}
	for _, tap := range q.tapKeys {
		if q.keyboard.IsKeyPressedAndRegister(tap) {
			q.Reset()
			q.suppressQuantityConfirmation()

			return &tap, true
		}
	}
	if heldCount > 0 {
		q.suppressQuantityConfirmation()
	}

	return nil, heldCount > 0
}

func (q *quantityInput) quantityModifierHeld() bool {
	for _, key := range [...]ebiten.Key{
		ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
		ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight,
	} {
		if q.keyState(key) {
			return true
		}
	}

	return false
}
