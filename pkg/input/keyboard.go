// Package input adapts keyboard state into debounced gameplay and UI input.
package input

import (
	"strconv"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/semantic_input"
	"github.com/hajimehoshi/ebiten/v2"
)

// Keyboard tracks key consumption, repeat timing, and release gates across input owners.
type Keyboard struct {
	navigationEpoch                   uint64
	millisecondDelayBetweenKeyPresses int
	timeOfLastKeyPress                int64
	timeToAllowNextKeypress           int64
	lastKeyPressed                    ebiten.Key
	forceRegisterNextKeyPressed       bool
	simulatedKeyPressed               ebiten.Key
	hasSimulatedKeyPress              bool
	keyPressed                        func(ebiten.Key) bool
	elapsedMilliseconds               func() int64
	previousSample                    KeySample
	suppressedUntilRelease            map[ebiten.Key]bool
	inputPollingSuspended             bool

	// Release-based consumption tracking - prevents multiple handlers from processing same key while it's held
	// Key stays in map while ebiten.IsKeyPressed() returns true; cleared when key is released
	consumedKeysUntilRelease map[ebiten.Key]bool

	// Release debouncing (from KeyDebouncer consolidation)
	waitingForRelease bool
	keysToRelease     []ebiten.Key
	postAcceptDelayMs int
}

// NewKeyboard creates a keyboard using physical key state and the default adapter clock.
func NewKeyboard(millisecondDelayBetweenKeyPresses int) *Keyboard {
	var keyboard Keyboard
	keyboard.millisecondDelayBetweenKeyPresses = millisecondDelayBetweenKeyPresses
	keyboard.consumedKeysUntilRelease = make(map[ebiten.Key]bool)
	keyboard.keyPressed = ebiten.IsKeyPressed
	keyboard.elapsedMilliseconds = func() int64 { return time.Now().UnixMilli() }

	return &keyboard
}

// NewKeyboardWithKeyStateSource constructs a keyboard backed by a caller-owned
// key-state source. It supports deterministic input adapters and tests without
// changing the keyboard's release-debouncing behavior.
func NewKeyboardWithKeyStateSource(
	millisecondDelayBetweenKeyPresses int,
	keyPressed func(ebiten.Key) bool,
) *Keyboard {
	if keyPressed == nil {
		panic("NewKeyboardWithKeyStateSource: key-state source is required")
	}
	keyboard := NewKeyboard(millisecondDelayBetweenKeyPresses)
	keyboard.keyPressed = keyPressed

	return keyboard
}

// TryToRegisterKeyPress registers a keypress if it is allowed according to the time passed since
// the previous key press AND if the key hasn't been consumed already while held
func (k *Keyboard) TryToRegisterKeyPress(key ebiten.Key) bool {
	if k.inputPollingSuspended || k.IsSuppressedUntilRelease(key) {
		return false
	}
	if k.consumeSimulatedKey(key) {
		return true
	}

	// Transition debouncing check (highest priority)
	// Prevents key bleeding across UI state transitions (e.g., dialog dismissal to console)
	if k.waitingForRelease {
		// Check if all tracked keys have been released
		if !k.areAllKeysReleased() {
			return false // Still waiting for keys to release
		}
		// All keys released - clear waiting state
		k.waitingForRelease = false
		k.keysToRelease = nil
	}

	// Release-based consumption check (second priority)
	// Prevents multiple handlers from processing the same key while it's physically held
	if k.consumedKeysUntilRelease[key] {
		return false // Key already processed - still held down
	}

	nowMilli := k.elapsedMilliseconds()

	// if k.forceRegisterNextKeyPressed {
	//	k.forceRegisterNextKeyPressed = false
	//	k.lastKeyPressed = key
	//	k.timeOfLastKeyPress = nowMilli
	//	return true
	//}

	if nowMilli < k.timeToAllowNextKeypress {
		return false
	}

	if key != k.lastKeyPressed || nowMilli-k.timeOfLastKeyPress > int64(k.millisecondDelayBetweenKeyPresses) ||
		k.forceRegisterNextKeyPressed {
		k.consumedKeysUntilRelease[key] = true // Mark key as consumed until released
		k.lastKeyPressed = key
		k.timeOfLastKeyPress = nowMilli
		k.forceRegisterNextKeyPressed = false

		return true
	}

	return false
}

// TryToRegisterRepeatableKeyPress registers a keypress that can repeat while held (e.g., movement keys).
// Returns true if keypress is allowed according to time-based debouncing and transition state.
// Unlike TryToRegisterKeyPress, this DOES NOT use release-based consumption, allowing
// continuous input while the key is held down.
// Use this for movement/navigation keys that should repeat when held.
func (k *Keyboard) TryToRegisterRepeatableKeyPress(key ebiten.Key) bool {
	return k.TryToRegisterRepeatableKeyPressWithRate(key, RepeatRate(k.millisecondDelayBetweenKeyPresses))
}

// TryToRegisterRepeatableKeyPressWithRate is like TryToRegisterRepeatableKeyPress but uses
// a custom repeat rate instead of the default keyboard delay.
// Use RepeatRateSlow, RepeatRateMedium, or RepeatRateFast for common presets.
func (k *Keyboard) TryToRegisterRepeatableKeyPressWithRate(key ebiten.Key, rate RepeatRate) bool {
	if k.inputPollingSuspended || k.IsSuppressedUntilRelease(key) {
		return false
	}
	if k.consumeSimulatedKey(key) {
		return true
	}

	// Transition debouncing check (highest priority)
	// Prevents key bleeding across UI state transitions (e.g., dialog dismissal to console)
	if k.waitingForRelease {
		// Check if all tracked keys have been released
		if !k.areAllKeysReleased() {
			return false // Still waiting for keys to release
		}
		// All keys released - clear waiting state
		k.waitingForRelease = false
		k.keysToRelease = nil
	}

	// Skip release-based consumption check - allow repeat while held
	// This is the key difference from TryToRegisterKeyPress()

	nowMilli := k.elapsedMilliseconds()

	// Time-based debouncing check
	if nowMilli < k.timeToAllowNextKeypress {
		return false
	}

	if key != k.lastKeyPressed || nowMilli-k.timeOfLastKeyPress > int64(rate) || k.forceRegisterNextKeyPressed {
		// Note: We do NOT add to consumedKeysUntilRelease here - that's what allows repeat
		k.lastKeyPressed = key
		k.timeOfLastKeyPress = nowMilli
		k.forceRegisterNextKeyPressed = false

		return true
	}

	return false
}

// GetMsSinceLastKeyPress returns elapsed adapter milliseconds since the last accepted key.
func (k *Keyboard) GetMsSinceLastKeyPress() int {
	return int(k.elapsedMilliseconds() - k.timeOfLastKeyPress)
}

// GetBoundKeyPressed returns the first pressed bound key without consuming it.
func (k *Keyboard) GetBoundKeyPressed(boundKeys *[]ebiten.Key) *ebiten.Key {
	if k.hasSimulatedKeyPress {
		for _, boundKey := range *boundKeys {
			if k.simulatedKeyPressed == boundKey {
				key := k.simulatedKeyPressed

				return &key
			}
		}
	}
	for _, boundKey := range *boundKeys {
		if k.isKeyPressed(boundKey) && !k.IsSuppressedUntilRelease(boundKey) {
			return &boundKey
		}
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
	if k != nil && k.inputPollingSuspended {
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

// SetLastKeyPressedNow restarts repeat timing at the current adapter time.
func (k *Keyboard) SetLastKeyPressedNow() {
	k.timeOfLastKeyPress = k.elapsedMilliseconds()
}

// ForceLastKeyPressed changes the repeat key identity without changing its timestamp.
func (k *Keyboard) ForceLastKeyPressed(key ebiten.Key) {
	k.lastKeyPressed = key
}

// SetLastKeyPressedNowPlusMs postpones repeat timing by the supplied milliseconds.
func (k *Keyboard) SetLastKeyPressedNowPlusMs(ms int) {
	k.timeOfLastKeyPress = k.elapsedMilliseconds() + int64(ms)
}

// SetForceWaitAnyKey gates key acceptance for the supplied milliseconds.
func (k *Keyboard) SetForceWaitAnyKey(ms int) {
	k.timeToAllowNextKeypress = k.elapsedMilliseconds() + int64(ms)
}

// SetAllowKeyPressImmediately bypasses repeat delay after transition and release gates pass.
func (k *Keyboard) SetAllowKeyPressImmediately() {
	// k.timeOfLastKeyPress = 0
	k.forceRegisterNextKeyPressed = true
}

// SimulateKeyPress simulates a key press for testing purposes
// This bypasses the normal key press timing restrictions
func (k *Keyboard) SimulateKeyPress(key ebiten.Key) {
	if k.inputPollingSuspended {
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

// RepeatRate defines keyboard repeat speed presets in milliseconds
type RepeatRate int

// Repeat-rate presets control the delay between accepted held-key repeats.
const (
	RepeatRateSlow   RepeatRate = 200 // Slow repeat rate (200ms between repeats)
	RepeatRateMedium RepeatRate = 120 // Medium repeat rate (120ms between repeats)
	RepeatRateFast   RepeatRate = 120 // Legacy fast preset (120ms, same as Medium); UI lists use ListNavigation.
)

// PrepareForTransition records currently pressed keys for release debouncing
// Use this when transitioning to a new UI state (e.g., dialog dismissal, KeyWait)
// to prevent key bleeding from the previous input
func (k *Keyboard) PrepareForTransition(postAcceptDelayMs int) {
	k.navigationEpoch++
	k.postAcceptDelayMs = postAcceptDelayMs
	k.keysToRelease = k.getCurrentlyPressedKeys()
	k.waitingForRelease = len(k.keysToRelease) > 0

	if !k.waitingForRelease {
		k.keysToRelease = nil
	}
}

// TryAcceptKey combines time-based and state-based release debouncing
// Returns true if the key should be processed, false if it should be ignored
//
// This method implements two-phase checking:
// Phase 1 (State-based): If PrepareForTransition was called, wait for key release
// Phase 2 (Time-based): Standard repeat delay checking
func (k *Keyboard) TryAcceptKey(key ebiten.Key) bool {
	if k.inputPollingSuspended || k.IsSuppressedUntilRelease(key) {
		return false
	}
	currentTime := k.elapsedMilliseconds()

	// Phase 1: State-based release requirement
	if k.waitingForRelease {
		// Check if all tracked keys have been released
		if !k.areAllKeysReleased() {
			return false // Still waiting for keys to release
		}
		// All keys released - clear waiting state
		k.waitingForRelease = false
		k.keysToRelease = nil
	}

	// Phase 2: Time-based repeat control (existing TryToRegisterKeyPress logic)
	if currentTime < k.timeToAllowNextKeypress {
		return false
	}

	if key != k.lastKeyPressed || currentTime-k.timeOfLastKeyPress > int64(k.millisecondDelayBetweenKeyPresses) ||
		k.forceRegisterNextKeyPressed {
		k.lastKeyPressed = key
		k.timeOfLastKeyPress = currentTime
		k.forceRegisterNextKeyPressed = false

		// Apply post-accept delay if configured
		if k.postAcceptDelayMs > 0 {
			k.timeToAllowNextKeypress = currentTime + int64(k.postAcceptDelayMs)
			k.postAcceptDelayMs = 0 // Reset after use
		}

		return true
	}

	return false
}

// IsWaitingForRelease returns true if keyboard is waiting for keys to be released
// Useful for UI feedback (e.g., showing "Release keys..." message)
func (k *Keyboard) IsWaitingForRelease() bool {
	return k.waitingForRelease
}

// ResetReleaseDebouncing clears all release debounce state
// Use when closing dialog or resetting UI
func (k *Keyboard) ResetReleaseDebouncing() {
	k.waitingForRelease = false
	k.keysToRelease = nil
	k.postAcceptDelayMs = 0
}

// ClearFrameConsumption cleans up released keys from consumption tracking
// Should be called once per frame by the main game loop at the START of Update()
// Only clears keys that are no longer physically pressed
func (k *Keyboard) ClearFrameConsumption() {
	if k.inputPollingSuspended {
		return // Masked input is not evidence of a physical release.
	}
	for key := range k.suppressedUntilRelease {
		if !k.isKeyPressed(key) {
			delete(k.suppressedUntilRelease, key)
		}
	}
	if k.waitingForRelease && k.areAllKeysReleased() {
		k.waitingForRelease = false
		k.keysToRelease = nil
	}

	// Clear consumed keys that have been physically released
	for key := range k.consumedKeysUntilRelease {
		if !k.isKeyPressed(key) {
			// Key is no longer held - remove from consumption tracking
			delete(k.consumedKeysUntilRelease, key)
		}
		// Key still held - leave in map to prevent duplicate processing
	}
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

// GetArrowKeyAsDirection returns the pressed arrow key as a Direction enum
// Returns references.NoneDirection if no arrow key is pressed
// Priority ordering: Down > Up > Right > Left
func (k *Keyboard) GetArrowKeyAsDirection() references.Direction {
	arrowKey := k.GetPressedArrowKey()
	if arrowKey == nil {
		return references.NoneDirection
	}

	switch *arrowKey { //nolint:exhaustive // GetPressedArrowKey returns only the four directional keys.
	case ebiten.KeyArrowUp:
		return references.Up
	case ebiten.KeyArrowDown:
		return references.Down
	case ebiten.KeyArrowLeft:
		return references.Left
	case ebiten.KeyArrowRight:
		return references.Right
	default:
		return references.NoneDirection
	}
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

// areAllKeysReleased checks if all keys in the tracking list have been released
// Returns true when all tracked keys are no longer pressed
func (k *Keyboard) areAllKeysReleased() bool {
	if len(k.keysToRelease) == 0 {
		return true // No keys to track
	}

	for _, key := range k.keysToRelease {
		if k.isKeyPressed(key) {
			return false // At least one key still pressed
		}
	}

	return true // All keys released
}

// getCurrentlyPressedKeys returns a slice of all keys currently pressed
// Used for release tracking to detect which keys need to be released
func (k *Keyboard) getCurrentlyPressedKeys() []ebiten.Key {
	const typicalHeldKeyCapacity = 5
	pressed := make([]ebiten.Key, 0, typicalHeldKeyCapacity)
	for key := ebiten.KeyA; key <= ebiten.KeyMax; key++ {
		if k.isKeyPressed(key) {
			pressed = append(pressed, key)
		}
	}

	return pressed
}
