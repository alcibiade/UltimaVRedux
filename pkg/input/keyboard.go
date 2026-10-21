// Package input adapts keyboard state into debounced gameplay and UI input.
package input

import (
	"strconv"
	"time"

	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
	beradinput "github.com/bradhannah/berad-sdk/modules/berad-input"
	"github.com/hajimehoshi/ebiten/v2"
)

// Keyboard tracks key consumption, repeat timing, and release gates across input owners.
type Keyboard struct {
	navigationEpoch                   uint64
	millisecondDelayBetweenKeyPresses int
	cadence                           beradinput.Cadence
	lastKeyPressed                    ebiten.Key
	keys                              *beradinput.Keyboard[ebiten.Key]
	forceRegisterNextKeyPressed       bool
	simulatedKeyPressed               ebiten.Key
	hasSimulatedKeyPress              bool
	keyPressed                        func(ebiten.Key) bool
	elapsedMilliseconds               func() int64
	postAcceptDelayMs                 int
}

// NewKeyboard creates a keyboard using physical key state and the default adapter clock.
func NewKeyboard(millisecondDelayBetweenKeyPresses int) *Keyboard {
	var keyboard Keyboard
	keyboard.millisecondDelayBetweenKeyPresses = millisecondDelayBetweenKeyPresses
	keyboard.keyPressed = ebiten.IsKeyPressed
	keyboard.elapsedMilliseconds = func() int64 { return time.Now().UnixMilli() }
	keyboard.keys = beradinput.NewKeyboard(controlKeys(), keyboard.physicalKeyPressed)

	return &keyboard
}

// TryToRegisterKeyPress registers a keypress if it is allowed according to the time passed since
// the previous key press AND if the key hasn't been consumed already while held
func (k *Keyboard) TryToRegisterKeyPress(key ebiten.Key) bool {
	if k.keys.Suspended() || k.IsSuppressedUntilRelease(key) {
		return false
	}
	if k.consumeSimulatedKey(key) {
		return true
	}

	return k.registerPhysicalKey(key)
}

func (k *Keyboard) registerPhysicalKey(key ebiten.Key) bool {
	if !k.keys.Ready(key, true) ||
		!k.acceptCadence(key, k.elapsedMilliseconds(), int64(k.millisecondDelayBetweenKeyPresses)) {
		return false
	}
	k.keys.Consume(key)

	return true
}

func (k *Keyboard) acceptCadence(key ebiten.Key, now, interval int64) bool {
	if !k.cadence.Accept(now, interval, key != k.lastKeyPressed || k.forceRegisterNextKeyPressed) {
		return false
	}
	k.forceRegisterNextKeyPressed = false
	k.lastKeyPressed = key

	return true
}

// TryToRegisterRepeatableKeyPress registers a keypress that can repeat while held (e.g., movement keys).
// Returns true if keypress is allowed according to time-based debouncing and transition state.
// Unlike TryToRegisterKeyPress, this DOES NOT use release-based consumption, allowing
// continuous input while the key is held down.
// Use this for movement/navigation keys that should repeat when held.
func (k *Keyboard) TryToRegisterRepeatableKeyPress(key ebiten.Key) bool {
	if k.keys.Suspended() || k.IsSuppressedUntilRelease(key) {
		return false
	}
	if k.consumeSimulatedKey(key) {
		return true
	}

	return k.keys.Ready(key, false) &&
		k.acceptCadence(key, k.elapsedMilliseconds(), int64(k.millisecondDelayBetweenKeyPresses))
}

// GetMsSinceLastKeyPress returns elapsed adapter milliseconds since the last accepted key.
func (k *Keyboard) GetMsSinceLastKeyPress() int {
	return int(k.cadence.Elapsed(k.elapsedMilliseconds()))
}

// GetBoundKeyPressed returns the first pressed bound key without consuming it.
func (k *Keyboard) GetBoundKeyPressed(boundKeys *[]ebiten.Key) *ebiten.Key {
	return k.GetRegisteredKeyPressed(beradinput.NewBindings((*boundKeys)...))
}

// GetRegisteredKeyPressed inspects a copied binding order without consuming it.
// Ordinary consumption is deliberately not a filter: the first held key retains
// priority even when its subsequent registration will be rejected.
func (k *Keyboard) GetRegisteredKeyPressed(bindings beradinput.Bindings[ebiten.Key]) *ebiten.Key {
	if k.hasSimulatedKeyPress {
		if key, found := bindings.FirstHeld(func(key ebiten.Key) bool { return key == k.simulatedKeyPressed }, nil); found {
			return &key
		}
	}
	if key, found := bindings.FirstHeld(k.isKeyPressed, func(key ebiten.Key) bool {
		return !k.IsSuppressedUntilRelease(key)
	}); found {
		return &key
	}

	return nil
}

func (k *Keyboard) consumeSimulatedKey(key ebiten.Key) bool {
	if k.hasSimulatedKeyPress && k.simulatedKeyPressed == key {
		k.hasSimulatedKeyPress = false

		return true
	}

	return false
}

func (k *Keyboard) isKeyPressed(key ebiten.Key) bool {
	if k == nil {
		return k.physicalKeyPressed(key)
	}

	return k.keys.Held(key)
}

func (k *Keyboard) physicalKeyPressed(key ebiten.Key) bool {
	// Caps Lock is lock state, never a gameplay hold or acknowledgement. Some
	// backends report it continuously while enabled; it must not own release
	// gates, focus-return suppression, samples, or all-key prompts.
	if key == ebiten.KeyCapsLock {
		return false
	}
	if k == nil {
		return ebiten.IsKeyPressed(key)
	}
	if k.keyPressed != nil {
		return k.keyPressed(key)
	}

	return ebiten.IsKeyPressed(key)
}

// IsKeyHeld reads key state through the same adapter used for modifiers and
// registered keys. Suspended polling masks it; it never consumes a key.
func (k *Keyboard) IsKeyHeld(key ebiten.Key) bool {
	return k.isKeyPressed(key)
}

// IsBoundKeyPressed reports whether a bound key is pressed and available to its owner.
func (k *Keyboard) IsBoundKeyPressed(boundKeys *[]ebiten.Key) bool {
	return k.GetBoundKeyPressed(boundKeys) != nil
}

// SetLastKeyPressedNowPlusMs postpones repeat timing by the supplied milliseconds.
func (k *Keyboard) SetLastKeyPressedNowPlusMs(ms int) {
	k.cadence.PostponeUntil(k.elapsedMilliseconds() + int64(ms))
}

// SetForceWaitAnyKey gates key acceptance for the supplied milliseconds.
func (k *Keyboard) SetForceWaitAnyKey(ms int) {
	k.cadence.BlockUntil(k.elapsedMilliseconds() + int64(ms))
}

// SetAllowKeyPressImmediately bypasses repeat delay after transition and release gates pass.
func (k *Keyboard) SetAllowKeyPressImmediately() {
	k.forceRegisterNextKeyPressed = true
}

// SimulateKeyPress simulates a key press for testing purposes
// This bypasses the normal key press timing restrictions
func (k *Keyboard) SimulateKeyPress(key ebiten.Key) {
	if k.keys.Suspended() {
		return
	}
	k.simulatedKeyPressed = key
	k.hasSimulatedKeyPress = true
}

// IsKeyPressedGetKey returns the key when physically pressed, without consuming it.
func (k *Keyboard) IsKeyPressedGetKey(key ebiten.Key) *ebiten.Key {
	if k.isKeyPressed(key) {
		return &key
	}

	return nil
}

// GetKeyLetter formats alphanumeric keys, including numeric keypad digits.
func (k *Keyboard) GetKeyLetter(key ebiten.Key) string {
	keyInt := int(key)
	if keyInt >= int(ebiten.KeyA) && keyInt <= int(ebiten.KeyZ) {
		return key.String()
	}
	if keyInt >= int(ebiten.Key0) && keyInt <= int(ebiten.Key9) {
		res := strconv.Itoa(keyInt - int(ebiten.Key0))

		return res
	}
	if keyInt >= int(ebiten.KeyDigit0) && keyInt <= int(ebiten.KeyDigit9) {
		res := strconv.Itoa(keyInt - int(ebiten.KeyDigit0))

		return res
	}
	if keyInt >= int(ebiten.KeyNumpad0) && keyInt <= int(ebiten.KeyNumpad9) {
		res := strconv.Itoa(keyInt - int(ebiten.KeyNumpad0))

		return res
	}

	return key.String()
}

// GetAlphaNumericPressed returns a pressed alphanumeric key and its display text.
func (k *Keyboard) GetAlphaNumericPressed() (*ebiten.Key, string) {
	if k.hasSimulatedKeyPress && ((k.simulatedKeyPressed >= ebiten.KeyA && k.simulatedKeyPressed <= ebiten.KeyZ) ||
		(k.simulatedKeyPressed >= ebiten.Key0 && k.simulatedKeyPressed <= ebiten.Key9)) {
		key := k.simulatedKeyPressed
		k.hasSimulatedKeyPress = false
		k.forceRegisterNextKeyPressed = true

		return &key, k.GetKeyLetter(key)
	}

	var key *ebiten.Key

	for i := int(ebiten.KeyA); i <= int(ebiten.KeyZ); i++ {
		key = k.IsKeyPressedGetKey(ebiten.Key(i))
		if key != nil {
			return key, k.GetKeyLetter(*key)
		}
	}
	for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
		key = k.IsKeyPressedGetKey(ebiten.Key(i))
		if key != nil {
			return key, k.GetKeyLetter(*key)
		}
	}

	return nil, ""
}

// IsAnyKeyPressed checks if any key is currently pressed
func (k *Keyboard) IsAnyKeyPressed() bool {
	// Check commonly used keys for "any key" functionality
	// Alphanumeric keys
	for i := int(ebiten.KeyA); i <= int(ebiten.KeyZ); i++ {
		if k.isKeyPressed(ebiten.Key(i)) {
			return true
		}
	}
	for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
		if k.isKeyPressed(ebiten.Key(i)) {
			return true
		}
	}

	// Common control keys
	controlKeys := []ebiten.Key{
		ebiten.KeySpace, ebiten.KeyEnter, ebiten.KeyEscape,
		ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight,
		ebiten.KeyTab, ebiten.KeyBackspace, ebiten.KeyDelete,
	}

	for _, key := range controlKeys {
		if k.isKeyPressed(key) {
			return true
		}
	}

	return false
}

// Modifier represents keyboard modifiers (Ctrl, Shift, Alt, Meta/Command).
type Modifier int

// Modifier values identify the supported keyboard modifier families.
const (
	ModifierCtrl Modifier = iota
	ModifierShift
	ModifierAlt
	ModifierMeta
)

// PrepareForTransition records currently pressed keys for release debouncing
// Use this when transitioning to a new UI state (e.g., dialog dismissal, KeyWait)
// to prevent key bleeding from the previous input
func (k *Keyboard) PrepareForTransition(postAcceptDelayMs int) {
	k.navigationEpoch++
	k.postAcceptDelayMs = postAcceptDelayMs
	k.keys.PrepareTransition()
}

// TryAcceptKey combines time-based and state-based release debouncing
// Returns true if the key should be processed, false if it should be ignored
//
// This method implements two-phase checking:
// Phase 1 (State-based): If PrepareForTransition was called, wait for key release
// Phase 2 (Time-based): Standard repeat delay checking
func (k *Keyboard) TryAcceptKey(key ebiten.Key) bool {
	if k.keys.Suspended() || k.IsSuppressedUntilRelease(key) {
		return false
	}
	currentTime := k.elapsedMilliseconds()

	if !k.keys.Ready(key, false) ||
		!k.acceptCadence(key, currentTime, int64(k.millisecondDelayBetweenKeyPresses)) {
		return false
	}
	// Only this legacy route applies the pending post-accept delay. Dummy-key
	// callers intentionally retain its timing side effects after release.
	if k.postAcceptDelayMs > 0 {
		k.cadence.BlockUntil(currentTime + int64(k.postAcceptDelayMs))
		k.postAcceptDelayMs = 0
	}

	return true
}

// IsWaitingForRelease returns true if keyboard is waiting for keys to be released
// Useful for UI feedback (e.g., showing "Release keys..." message)
func (k *Keyboard) IsWaitingForRelease() bool {
	return k.keys.WaitingForRelease()
}

// ResetReleaseDebouncing clears all release debounce state
// Use when closing dialog or resetting UI
func (k *Keyboard) ResetReleaseDebouncing() {
	k.keys.ResetTransition()
	k.postAcceptDelayMs = 0
}

// ClearFrameConsumption cleans up released keys from consumption tracking
// Should be called once per frame by the main game loop at the START of Update()
// Only clears keys that are no longer physically pressed
func (k *Keyboard) ClearFrameConsumption() {
	k.keys.ObserveReleases()
}

// IsModifierPressed checks if a modifier key is currently pressed
func (k *Keyboard) IsModifierPressed(modifier Modifier) bool {
	switch modifier {
	case ModifierCtrl:
		return k.isKeyPressed(ebiten.KeyControl) ||
			k.isKeyPressed(ebiten.KeyControlLeft) ||
			k.isKeyPressed(ebiten.KeyControlRight)
	case ModifierShift:
		return k.isKeyPressed(ebiten.KeyShift) || k.isKeyPressed(ebiten.KeyShiftLeft) || k.isKeyPressed(ebiten.KeyShiftRight)
	case ModifierAlt:
		return k.isKeyPressed(ebiten.KeyAlt) || k.isKeyPressed(ebiten.KeyAltLeft) || k.isKeyPressed(ebiten.KeyAltRight)
	case ModifierMeta:
		return k.isKeyPressed(ebiten.KeyMeta) || k.isKeyPressed(ebiten.KeyMetaLeft) || k.isKeyPressed(ebiten.KeyMetaRight)
	default:
		return false
	}
}

// IsModifierComboPressed checks if a modifier+key combination is currently pressed
// Returns true only if both the modifier and the specific key are pressed
func (k *Keyboard) IsModifierComboPressed(modifier Modifier, key ebiten.Key) bool {
	if !k.IsModifierPressed(modifier) {
		return false
	}

	return k.isKeyPressed(key)
}

// IsKeyPressedAndRegister is a convenience method combining key press detection with debouncing
// Returns true if the key is pressed AND the debounce time has elapsed
// Simplifies the common pattern: if IsKeyPressed && TryToRegisterKeyPress
func (k *Keyboard) IsKeyPressedAndRegister(key ebiten.Key) bool {
	if k.hasSimulatedKeyPress && k.simulatedKeyPressed == key {
		k.hasSimulatedKeyPress = false

		return true
	}
	if k.isKeyPressed(key) {
		return k.TryToRegisterKeyPress(key)
	}

	return false
}

// IsTextKeyPressedAndRegister accepts each released-and-pressed text key without
// the movement repeat delay. Transition and held-key consumption still apply.
func (k *Keyboard) IsTextKeyPressedAndRegister(key ebiten.Key) bool {
	previous := k.forceRegisterNextKeyPressed
	k.forceRegisterNextKeyPressed = true
	registered := k.IsKeyPressedAndRegister(key)
	if registered {
		k.forceRegisterNextKeyPressed = false
	} else {
		k.forceRegisterNextKeyPressed = previous
	}

	return registered
}

// IsKeyPressedAndRegisterRepeatable combines injected/simulated key detection
// with movement-style repeat registration.
func (k *Keyboard) IsKeyPressedAndRegisterRepeatable(key ebiten.Key) bool {
	if k.hasSimulatedKeyPress && k.simulatedKeyPressed == key {
		k.hasSimulatedKeyPress = false

		return true
	}
	if k.isKeyPressed(key) {
		return k.TryToRegisterRepeatableKeyPress(key)
	}

	return false
}

// IsNumberKeyPressedAndRegister checks if any number key (0-9) is pressed.
// Returns the digit (0-9) if pressed and registered, or -1 if no number key is pressed.
// This simplifies the common pattern of checking all number keys in sequence.
func (k *Keyboard) IsNumberKeyPressedAndRegister() int {
	numberKeys := []struct {
		key   ebiten.Key
		digit int
	}{
		{ebiten.Key0, 0}, {ebiten.Key1, 1}, {ebiten.Key2, 2},
		{ebiten.Key3, 3}, {ebiten.Key4, 4}, {ebiten.Key5, 5},
		{ebiten.Key6, 6}, {ebiten.Key7, 7}, {ebiten.Key8, 8},
		{ebiten.Key9, 9},
	}

	for _, nk := range numberKeys {
		if k.IsKeyPressedAndRegister(nk.key) {
			return nk.digit
		}
	}

	return -1
}

// GetPressedArrowKey returns the currently pressed arrow key, or nil if none pressed
// Uses priority ordering to determine which key to return when multiple are pressed:
// Down (highest priority) > Up > Right > Left (lowest priority)
// This matches the priority used in original game movement logic
func (k *Keyboard) GetPressedArrowKey() *ebiten.Key {
	for _, key := range []ebiten.Key{
		ebiten.KeyArrowDown,
		ebiten.KeyArrowUp,
		ebiten.KeyArrowRight,
		ebiten.KeyArrowLeft,
	} {
		if (k.hasSimulatedKeyPress && k.simulatedKeyPressed == key) || k.isKeyPressed(key) {
			return &key
		}
	}

	return nil
}

// GetPressedKey returns the currently pressed key, prioritizing arrow keys
// Returns nil if no key is pressed
// Priority: Arrow keys (Down > Up > Right > Left) > Alpha (A-Z) > Numeric (0-9) > Control keys
//
// Used by event queue WhileAwaitingInput() to detect which key stopped sailing
// Matches original Ultima V key detection priority for automatic sailing
//
// Returns a framework-independent semantic key for use by core packages.
func (k *Keyboard) GetPressedKey() *semantic_input.Key {
	// Highest priority: Arrow keys (for sailing direction changes)
	if arrow := k.GetPressedArrowKey(); arrow != nil {
		key := semanticKey(*arrow)

		return &key
	}

	// Second priority: Alphanumeric keys (A-Z)
	for i := int(ebiten.KeyA); i <= int(ebiten.KeyZ); i++ {
		if k.isKeyPressed(ebiten.Key(i)) {
			key := semanticKey(ebiten.Key(i))

			return &key
		}
	}

	// Third priority: Numeric keys (0-9)
	for i := int(ebiten.Key0); i <= int(ebiten.Key9); i++ {
		if k.isKeyPressed(ebiten.Key(i)) {
			key := semanticKey(ebiten.Key(i))

			return &key
		}
	}

	// Fourth priority: Common control keys
	controlKeys := []ebiten.Key{
		ebiten.KeySpace,
		ebiten.KeyEnter,
		ebiten.KeyEscape,
		ebiten.KeyTab,
		ebiten.KeyBackspace,
	}

	for _, ctrlKey := range controlKeys {
		if k.isKeyPressed(ctrlKey) {
			key := semanticKey(ctrlKey)

			return &key
		}
	}

	return nil
}

func semanticKey(key ebiten.Key) semantic_input.Key {
	//nolint:exhaustive // Core contracts distinguish only the semantic keys listed below.
	switch key {
	case ebiten.KeyArrowUp:
		return semantic_input.KeyUp
	case ebiten.KeyArrowDown:
		return semantic_input.KeyDown
	case ebiten.KeyArrowLeft:
		return semantic_input.KeyLeft
	case ebiten.KeyArrowRight:
		return semantic_input.KeyRight
	case ebiten.KeySpace:
		return semantic_input.KeySpace
	case ebiten.KeyEnter:
		return semantic_input.KeyEnter
	default:
		return semantic_input.KeyOther
	}
}

// controlKeys includes unbound physical controls for activation/focus barriers.
// Caps Lock exclusion is applied by the host source, not by the generic SDK.
func controlKeys() []ebiten.Key {
	keys := make([]ebiten.Key, 0, int(ebiten.KeyMax-ebiten.KeyA)+1)
	for key := ebiten.KeyA; key <= ebiten.KeyMax; key++ {
		keys = append(keys, key)
	}

	return keys
}
