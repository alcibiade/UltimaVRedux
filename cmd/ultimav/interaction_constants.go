package main

// Dialog and interaction timing constants

const (
	// DialogPollingRateMs is the polling rate for dialog state checks (~60fps)
	DialogPollingRateMs = 16

	// KeyboardDebounceMs is the standard debounce delay after dialog dismissal
	KeyboardDebounceMs = 250

	// KeyboardDebounceShortMs is a short debounce delay for quick transitions
	KeyboardDebounceShortMs = 50

	// ArrestSequenceUnconsciousDelayMs - Time to read "struck unconscious" message
	ArrestSequenceUnconsciousDelayMs = 2500

	// ArrestSequenceFadeDurationMs - Duration for fade in/out animations
	ArrestSequenceFadeDurationMs = 2000

	// ArrestSequenceBlackScreenPauseMs - Pause while screen is black during teleport
	ArrestSequenceBlackScreenPauseMs = 1000

	// SoundEffectPauseMs is the pause duration after playing a sound effect
	SoundEffectPauseMs = 200

	// GemViewFlashIntervalMs is the interval for player position flash in gem view
	// ref:VIEW-0002: original view marker toggled every 4 loop iterations (~250ms at 60fps).
	GemViewFlashIntervalMs = 250

	// WishMaxLength is the maximum character limit for wishing well wishes (original Ultima V)
	WishMaxLength = 12

	// WishInputLines is the number of text lines for wish input dialog
	WishInputLines = 2

	// WishInputColumns is the character width for wish input dialog
	WishInputColumns = 45
)
