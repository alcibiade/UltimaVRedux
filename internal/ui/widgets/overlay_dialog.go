package widgets

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/text"
)

// OverlayDialog provides a screen overlay message with explicit configuration.
// It wraps MessageDialog with a cleaner, validation-enforced API that requires
// all configuration to be explicitly set.
//
// Create using NewOverlayDialog() builder:
//
//	dialog := widgets.NewOverlayDialog("Success!").
//	    AsOverlayFeedback().                 // Template sets appearance
//	    WithExternallyManagedDuration(1000). // Explicit duration choice
//	    Build()
type OverlayDialog struct {
	// Underlying dialog implementation
	dialog *MessageDialog

	// Configuration captured from builder
	message                   string
	durationMs                int  // Duration for timed dialogs (0 if user-dismissed)
	isTimed                   bool // true = timed, false = user-dismissed
	durationExternallyManaged bool // true when the owner, rather than Update, controls expiry
	mirrorToOutput            *text.Output
	onDismiss                 func()
}

// =====================================
// WIDGET INTERFACE IMPLEMENTATION
// =====================================

// Draw renders the overlay dialog to the screen.
func (o *OverlayDialog) Draw(screen *ebiten.Image) {
	if o.dialog != nil && o.dialog.visible {
		o.PrepareLayout(screen.Bounds())
		o.dialog.drawPrepared(screen)
	}
}

// PrepareLayout resolves popup bounds and wrapped presentation for the supplied
// logical screen rectangle. It does not rasterize, advance animation or duration,
// consume input, change visibility/focus, mirror text, or invoke callbacks. Like
// Draw, it may rebuild presentation resources when the logical bounds change.
func (o *OverlayDialog) PrepareLayout(bounds image.Rectangle) {
	if o.dialog != nil {
		o.dialog.resolvePopupLayout(bounds)
	}
}

// Update handles per-frame input. Timed dialogs stay visible until their owner
// removes them; Update never expires them.
func (o *OverlayDialog) Update() {
	if o.dialog != nil {
		o.dialog.Update()
	}
}

// =====================================
// VISIBILITY CONTROL
// =====================================

// Show displays the dialog with configured fade effects (if any).
func (o *OverlayDialog) Show() {
	if o.dialog != nil {
		o.dialog.Show()

		// Mirror to output if configured
		if o.mirrorToOutput != nil && o.message != "" {
			o.mirrorToOutput.AddText(o.message)
		}
	}
}

// ShowWithoutAnimation displays the dialog instantly (no fade-in).
func (o *OverlayDialog) ShowWithoutAnimation() {
	if o.dialog != nil {
		o.dialog.ShowWithoutAnimation()

		// Mirror to output if configured
		if o.mirrorToOutput != nil && o.message != "" {
			o.mirrorToOutput.AddText(o.message)
		}
	}
}

// Hide hides the dialog with configured fade effects (if any).
func (o *OverlayDialog) Hide() {
	if o.dialog != nil {
		o.dialog.Hide()
	}
}

// HideWithoutAnimation hides the dialog instantly (no fade-out).
func (o *OverlayDialog) HideWithoutAnimation() {
	if o.dialog != nil {
		o.dialog.HideWithoutAnimation()
	}
}

// IsVisible returns true if the dialog is currently visible.
func (o *OverlayDialog) IsVisible() bool {
	if o.dialog != nil {
		return o.dialog.IsVisible()
	}

	return false
}

// =====================================
// DISMISSAL STATE
// =====================================

// IsDismissed returns true if the dialog has been dismissed.
// Timed dialogs are never dismissed by the dialog itself: their owner removes them.
// For user-dismissed dialogs, this returns true when user presses any key.
func (o *OverlayDialog) IsDismissed() bool {
	if o.isTimed {
		return false
	}

	if o.dialog != nil {
		return o.dialog.IsDismissed()
	}

	return false
}

// Dismiss closes a user-dismissed overlay through its normal callback path.
func (o *OverlayDialog) Dismiss() {
	if o.dialog != nil {
		o.dialog.Dismiss()
	}
}

// =====================================
// CONFIGURATION ACCESSORS (for testing)
// =====================================

// GetMessage returns the dialog message.
func (o *OverlayDialog) GetMessage() string {
	return o.message
}

// GetContentText returns all currently rendered dialog content.
func (o *OverlayDialog) GetContentText() string {
	if o.dialog == nil || o.dialog.output == nil {
		return ""
	}

	return o.dialog.output.GetOutputStr(false)
}

// GetDurationMs returns the configured duration (0 for user-dismissed).
func (o *OverlayDialog) GetDurationMs() int {
	return o.durationMs
}

// IsTimed returns true if this is a timed auto-dismiss dialog.
func (o *OverlayDialog) IsTimed() bool {
	return o.isTimed
}

// IsDurationExternallyManaged reports whether the dialog owner controls expiry.
func (o *OverlayDialog) IsDurationExternallyManaged() bool {
	return o.durationExternallyManaged
}

// IsUserDismissed returns true if this is a user-dismissed dialog.
func (o *OverlayDialog) IsUserDismissed() bool {
	return !o.isTimed
}

// =====================================
// CONTENT MANIPULATION (proxied from MessageDialog)
// =====================================

// AppendLineWithColor adds a colored text line to the dialog content area.
// Useful for adding highlighted text after the initial message (e.g., gold amounts).
func (o *OverlayDialog) AppendLineWithColor(line string, c color.Color) {
	if o.dialog != nil {
		o.dialog.AppendLineWithColor(line, c)
	}
}

// AppendLineBuilder returns a LineBuilder for constructing formatted lines.
// Provides fluent API for multi-styled text (e.g., "You paid " + yellow("100 gold")).
func (o *OverlayDialog) AppendLineBuilder() *text.LineBuilder {
	if o.dialog != nil {
		return o.dialog.AppendLineBuilder()
	}

	return nil
}
