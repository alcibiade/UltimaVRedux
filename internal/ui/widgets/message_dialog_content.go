package widgets

import (
	"image"
	"image/color"
	"math"
	"unicode/utf8"

	"golang.org/x/exp/rand" //nolint:staticcheck // Retain the established injectable dissolve RNG contract.

	"github.com/bradhannah/UltimaVRedux/internal/text"
)

// AppendLine adds a text line to the dialog content area.
func (md *MessageDialog) AppendLine(line string) {
	md.output.AddText(line)
	md.contentDirty = true
}

// AppendLineBuilder returns a LineBuilder for advanced text formatting with colors and styles.
func (md *MessageDialog) AppendLineBuilder() *text.LineBuilder {
	md.contentDirty = true

	return md.output.AppendLine()
}

// AppendLineWithColor adds a colored text line to the dialog content area.
func (md *MessageDialog) AppendLineWithColor(line string, c color.Color) {
	md.output.AddTextWithColor(line, c)
	md.contentDirty = true
}

// AppendToCurrentLine appends text to the current line without adding a new line.
// Used for progressive text effects like "sneaking" → "sneaking." → "sneaking.." → "sneaking..."
// Wraps text.Output.AppendText() which operates on the current row.
func (md *MessageDialog) AppendToCurrentLine(text string) {
	md.output.AppendText(text)
	md.contentDirty = true
}

// Clear removes all text from the dialog content area.
func (md *MessageDialog) Clear() {
	md.output.Clear()
	md.contentDirty = true
}

// SetAcknowledgementPrompt uses the shared guidance styling without taking input
// ownership. Workflow-owned waits reserve the area with WithSeparatePromptArea.
func (md *MessageDialog) SetAcknowledgementPrompt(hint string) {
	if !md.hasPromptOutput() {
		panic("SetAcknowledgementPrompt() requires WithSeparatePromptArea()")
	}
	md.promptColor = DialogControlHintColor
	md.promptOutput.Reset()
	// Keep the complete logical hint; the measured popup layout wraps a styled
	// copy. A tiny content-width source buffer must not overwrite its key prefix.
	md.promptOutput.SetMaxCharsPerLine(max(1, utf8.RuneCountInString(hint)))
	md.promptOutput.SetColor(DialogControlHintColor)
	for _, segments := range parseMerchantFooterGuidance(hint) {
		line := md.promptOutput.AppendLine()
		for _, segment := range segments {
			line.Text(segment.Text, segment.Color)
		}
		line.Done()
	}
	md.contentDirty = true
}

// SetPrompt sets the prompt text in the separate prompt area.
// Panics if WithSeparatePromptArea() was not called during builder configuration.
func (md *MessageDialog) SetPrompt(prompt string) {
	if !md.hasPromptOutput() {
		panic("SetPrompt() requires WithSeparatePromptArea() - enable separate prompt area in builder")
	}
	md.promptOutput.Clear()
	md.promptOutput.AddText(prompt)
	md.contentDirty = true
}

// ClearPrompt clears the prompt text in the separate prompt area.
// Panics if WithSeparatePromptArea() was not called during builder configuration.
func (md *MessageDialog) ClearPrompt() {
	if !md.hasPromptOutput() {
		panic("ClearPrompt() requires WithSeparatePromptArea() - enable separate prompt area in builder")
	}
	md.promptOutput.Clear()
	md.contentDirty = true
}

// SetPromptWithColor sets prompt text with custom color in the separate prompt area.
// Panics if WithSeparatePromptArea() was not called during builder configuration.
func (md *MessageDialog) SetPromptWithColor(prompt string, promptColor color.Color) {
	if !md.hasPromptOutput() {
		panic("SetPromptWithColor() requires WithSeparatePromptArea() - enable separate prompt area in builder")
	}
	md.promptColor = promptColor
	md.promptOutput.Clear()
	md.promptOutput.SetColor(promptColor)
	md.promptOutput.AddText(prompt)
	md.contentDirty = true
}

// SetRNG sets a custom random number generator for dissolve effects (useful for deterministic testing).
func (md *MessageDialog) SetRNG(rng *rand.Rand) {
	md.rng = rng
}

// SetFadeDurations customizes the fade animation timing for both fade-in and fade-out effects.
func (md *MessageDialog) SetFadeDurations(fadeInMs, fadeOutMs int) {
	md.fadeDurationInMs = fadeInMs
	md.fadeDurationOutMs = fadeOutMs
}

// UseAnimationClock opts this dialog into a caller-owned millisecond clock.
// It must be configured before the dialog is first shown.
func (md *MessageDialog) UseAnimationClock(clock func() int64) {
	if clock == nil {
		panic("MessageDialog.UseAnimationClock: clock is required")
	}
	if md.animationStarted {
		panic("MessageDialog.UseAnimationClock: clock must be configured before Show")
	}
	md.animationClock = clock
}

// SetPresentationPosition moves the already-rendered card as one composition.
// clip is expressed in the same logical screen coordinates as x and y.
func (md *MessageDialog) SetPresentationPosition(x, y float64, clip image.Rectangle) {
	if !md.completeCardLayout {
		panic("MessageDialog.SetPresentationPosition: complete card layout is required")
	}
	md.presentationPosition.X = x
	md.presentationPosition.Y = y
	md.presentationClip = clip
}

// PresentationBounds returns the translated complete-card bounds.
func (md *MessageDialog) PresentationBounds() image.Rectangle {
	if !md.completeCardLayout {
		return md.frameBounds
	}
	topLeft := image.Pt(int(math.Round(md.presentationPosition.X)), int(math.Round(md.presentationPosition.Y)))

	return image.Rectangle{Min: topLeft, Max: topLeft.Add(md.bakedBounds.Size())}
}

// DissolveLevel reports the current visual dissolve amount for lifecycle tests.
func (md *MessageDialog) DissolveLevel() float64 {
	return md.dissolveLevel
}

// GetFadeInDuration returns the configured fade-in animation duration in milliseconds.
func (md *MessageDialog) GetFadeInDuration() int {
	return md.fadeDurationInMs
}

// GetFadeOutDuration returns the configured fade-out animation duration in milliseconds.
func (md *MessageDialog) GetFadeOutDuration() int {
	return md.fadeDurationOutMs
}

// IsDismissed returns true if the user dismissed the dialog via ESC/Enter (simple mode only).
func (md *MessageDialog) IsDismissed() bool {
	return md.dismissed
}

// Dismiss closes an input-dismissed message through its normal callback path.
func (md *MessageDialog) Dismiss() {
	md.dismiss()
}

// IsStaticContent returns true if the dialog should not receive mirrored text from game output.
// Static content dialogs display their initial message and ignore addRowStr() calls.
func (md *MessageDialog) IsStaticContent() bool {
	return md.staticContent
}

// IsFinished returns true if the dialog has completed its lifecycle and should be removed from stack.
func (md *MessageDialog) IsFinished() bool {
	return md.dismissed
}

// GetResult returns the dialog result (currently returns dismissal state for ModalDialog interface).
func (md *MessageDialog) GetResult() interface{} {
	return md.dismissed
}

// Refresh updates visual elements on resolution change
func (md *MessageDialog) Refresh() {
	md.initializeVisualElements()
	md.contentDirty = true
}
