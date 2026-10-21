package widgets

import (
	"strings"
)

// Shared dialog sizing constants are defined in dialog_constants.go.

// NewCampingMessageDialog creates MessageDialog configured for camping system.
// This is a convenience constructor for camping-related message displays.
//
// Features:
//   - 6 lines, 40 characters per line (compact display)
//   - Effects enabled (dissolve animations)
//   - Empty initial message (content added via AppendLine)
//
// Returns:
//   - *MessageDialog configured for camping display
//
// Example usage:
//
//	campingDialog := widgets.NewCampingMessageDialog()
//	campingDialog.AppendLine("You rest for the night...")
//	dialogStack.PushModalDialog(campingDialog)
func NewCampingMessageDialog() *MessageDialog {
	return NewMessageDialog("").
		WithTextLines(DialogCampingLines).
		WithCharsPerLine(dialogNarrowCharsPerLine).
		WithEffects(true).
		WithSeparatePromptArea(). // Enable prompt area for ShowPressAnyKeyPrompt()
		WithHorizontalPosition(DialogHorizontalMapOnly).
		WithVerticalPosition(DialogDefaultVerticalPos).
		Build()
}

// NewCampingResultDialog fits a short rest outcome and the workflow-owned
// acknowledgement prompt. Wanderer conversations retain NewCampingMessageDialog.
func NewCampingResultDialog(message string) *MessageDialog {
	return NewMessageDialog(message).
		WithTextLines(DialogConfirmationLines).
		WithCharsPerLine(max(len(message), len(DialogAnyKeyHint))).
		WithEffects(true).WithSeparatePromptArea().
		WithContentTextAlignment(ContentAlignCenter).WithTextHorizontalCenter(true).
		WithHorizontalPosition(DialogHorizontalMapOnly).WithVerticalPosition(DialogDefaultVerticalPos).
		WithStaticContent().Build()
}

// NewActionFeedbackDialog creates a small popup for action command responses.
// Displays command feedback like "Unlocked!", "Key broke!" prominently in the game view.
//
// Features:
//   - Minimum 2-line, 25-character width, growing to retain the complete source
//   - Caller-selected positioning for context-appropriate visibility
//   - Fade effects enabled (400ms fade-in, 300ms fade-out for readable feedback)
//   - No input dismissal (GameScene owns time-based auto-dismissal)
//   - Center-aligned text
//
// Designed for GameScene-owned transient feedback that runs concurrently with gameplay:
//
//	dialog := widgets.NewActionFeedbackDialog("Unlocked!")
//	dialog.Show()  // Triggers fade-in
//	// ... wait ActionFeedbackDisplayMs ...
//	dialog.Hide()  // Triggers fade-out
//
// Parameters:
//   - message: Short feedback text to display (e.g., "Unlocked!", "Key broke!")
//
// Returns:
//   - *MessageDialog configured for action feedback display
func NewActionFeedbackDialog(message string, position Position) *MessageDialog {
	return newSourceSizedFeedbackDialog(message, position, actionFeedbackTextLines, actionFeedbackCharsPerLine)
}

// NewSearchFeedbackDialog sizes transient feedback to retain every Search line.
func NewSearchFeedbackDialog(message string, position Position) *MessageDialog {
	return newSourceSizedFeedbackDialog(message, position, 1, 1)
}

func newSourceSizedFeedbackDialog(message string, position Position, minimumLines, minimumWidth int) *MessageDialog {
	// Retain explicit source rows before the popup's measured wrapping runs.
	lines := strings.Split(message, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	maximumWidth := minimumWidth
	for _, line := range lines {
		// AddText wraps by bytes; retain each whole row until rune-safe measured reflow.
		maximumWidth = max(maximumWidth, len(line))
	}

	return NewMessageDialog(message).
		WithEffects(true).
		WithFadeDurations(ActionFeedbackFadeInMs, ActionFeedbackFadeOutMs).
		WithTextLines(max(minimumLines, len(lines))).
		WithCharsPerLine(maximumWidth).
		WithHorizontalPosition(DialogHorizontalMapOnly).
		WithVerticalPosition(position.GetVerticalPercent()).
		WithContentTextAlignment(ContentAlignCenter).
		WithStaticContent().
		Build()
}
