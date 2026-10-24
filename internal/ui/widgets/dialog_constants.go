package widgets

import (
	"math"
)

// Layout spacing constants (percentage of container/screen)
// These provide semantic names for the commonly used padding/margin values.
const (
	// LayoutSpacingSmall is minimal padding for tight layouts (1%)
	LayoutSpacingSmall = 0.01

	// LayoutSpacingMedium is increased padding for border insets (3%)
	LayoutSpacingMedium = 0.03
)

const (
	classicUILogicalWidth  = 320
	classicUILogicalHeight = 200
)

func getClassicUIPixelUnit(width, height int) int {
	scaleX := float64(max(1, width)) / classicUILogicalWidth
	scaleY := float64(max(1, height)) / classicUILogicalHeight

	return max(1, int(math.Round(min(scaleX, scaleY))))
}

// ClassicUIPixelUnit returns the scaled logical pixel used by classic game UI frames.
func ClassicUIPixelUnit(width, height int) int {
	return getClassicUIPixelUnit(width, height)
}

// Border width scaling constants (higher value = thinner border)
const (
	// BorderWidthStandard is the default for dialogs (601)
	BorderWidthStandard = 601
)

// Modal dialog layout constants (shared by ButtonListModal, CharacterSelectionDialog, NewOrderDialog)
const (
	modalDialogTitleMaxChars = 30 // Maximum characters in title
)

// Dialog sizing defaults - used by all dialog types for consistency.
// These constants provide a single source of truth for dialog dimensions.
//
// Usage pattern:
//   - Use Default* constants for standard dialogs
//   - Use Narrow* for compact confirmations
//   - Override with With*() builder methods when specific sizing needed
const (
	// Character limits per line (private - only used within widgets package)
	dialogDefaultCharsPerLine      = 40 // Standard width for most dialogs
	dialogNarrowCharsPerLine       = 40 // Narrower dialogs for compact messages
	dialogConfirmationCharsPerLine = 47 // Keeps short quoted statements together before the response question
)

// Dialog sizing constants used by prefab factories (public - used by cmd/ultimav)
const (
	DialogConfirmationLines  = 2    // Yes/No confirmations and simple messages
	DialogCampingLines       = 6    // Camping display with multiple status lines
	DialogDefaultVerticalPos = 0.70 // Standard position (70% down from top)
)

// Shared acknowledgement guidance. Keep timed notices footerless and match the
// hint to the keys accepted by the dialog or its owning workflow.
const (
	DialogCloseHint    = "[Enter/Esc] Close"
	DialogContinueHint = "[Enter/Esc] Continue"
	DialogAnyKeyHint   = "[Any key] Continue"
)

// Action feedback popup constants - small, centered popups for command responses.
// These dialogs display prominently in the game view with fade effects,
// using a GameScene-owned elapsed-time lifecycle that does not block input.
const (
	// Action feedback sizing (private - only used within widgets package)
	actionFeedbackTextLines    = 2  // Maximum lines for feedback message
	actionFeedbackCharsPerLine = 25 // Narrow width for centered popup
)

// Action feedback constants used by prefab factories and GameScene (public)
const (
	ActionFeedbackFadeInMs  = 400  // Fade-in duration
	ActionFeedbackFadeOutMs = 300  // Fade-out duration
	ActionFeedbackDisplayMs = 2000 // Consumed age when fade-out begins (400ms reveal + 1600ms hold)
)

// Keyboard transition timing constants - used for debouncing across dialog transitions.
// These prevent keys pressed during one dialog from bleeding into the next.
const (
	// KeyboardTransitionDelayMs is the post-accept delay for PrepareForTransition() calls.
	// Used when transitioning between dialogs to prevent key bleeding.
	KeyboardTransitionDelayMs = 100

	// DialogDismissalDelayMs is the delay for SetForceWaitAnyKey() calls.
	// Used before waiting for any key to dismiss a dialog.
	DialogDismissalDelayMs = 200
)

// ItemSelectionDialog layout constants - responsive master/detail Use picker.
const (
	itemDialogMarginX               = 0.10
	itemDialogMarginY               = 0.10
	itemDialogListRatio             = 0.40
	itemDialogMinimumPadding        = 4
	itemDialogPaddingDivisor        = 64
	itemDialogVerticalRegionDivisor = 3

	// Shared by the older Ready, Mix, and Cast panel layouts.
	itemDialogBorderWidth = BorderWidthStandard
	itemDialogTextPadding = 8
)

// Auto-sizing dialog constants - used by WithAutoSize() for content-based dialog sizing.
// These constrain dialogs to reasonable screen proportions while allowing content to expand.
const (
	// AutoSizeMaxWidthPercent is the default maximum dialog width (80% of screen)
	AutoSizeMaxWidthPercent = 0.80

	// AutoSizeMaxHeightPercent is the default maximum dialog height (60% of screen)
	AutoSizeMaxHeightPercent = 0.60

	// AutoSizeMinCharsPerLine is the minimum characters per line for aesthetics
	AutoSizeMinCharsPerLine = 15

	// AutoSizeMinLines is the minimum number of lines
	AutoSizeMinLines = 1

	// AutoSizeHorizontalPadding is the percentage of screen width used for dialog padding (4%)
	AutoSizeHorizontalPadding = 0.04

	// AutoSizeVerticalPadding is the percentage of screen height used for dialog padding (6%)
	AutoSizeVerticalPadding = 0.06
)
