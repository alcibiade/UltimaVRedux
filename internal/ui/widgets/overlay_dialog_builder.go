package widgets

import (
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// OverlayDialogBuilder provides a fluent API for constructing OverlayDialog instances
// with explicit configuration requirements. All appearance and duration settings must
// be explicitly specified - there are no hidden defaults.
//
// Required configuration (Build() panics if missing):
//   - Sizing: WithSizing() or use a template
//   - Position: WithPosition() or use a template
//   - Effects: WithEffects() or use a template
//   - ContentAlignment: WithContentAlignment() or use a template
//   - TextAlignment: WithTextAlignment() or use a template
//   - Duration: Exactly one of WithDuration(), WithExternallyManagedDuration(), or WithUserDismissed()
//
// Example usage:
//
//	// Concise with template
//	dialog := widgets.NewOverlayDialog("Success!").
//	    AsOverlayFeedback().
//	    WithDuration(1000).
//	    Build()
//
//	// Fully explicit
//	dialog := widgets.NewOverlayDialog("Custom").
//	    WithSizing(FixedLinesSizing{Lines: 4, CharsPerLine: 50}).
//	    WithPosition(PositionScreenCenter).
//	    WithEffects(EffectsNone).
//	    WithContentAlignment(ContentAlignCentered).
//	    WithTextAlignment(TextAlignCenter).
//	    WithUserDismissed(keyboard).
//	    Build()
type OverlayDialogBuilder struct {
	// Message content
	message string

	// Required appearance configuration
	sizing       SizingStrategy
	position     Position
	centerX      *float64
	effects      EffectsConfig
	contentAlign ContentAlignment
	textAlign    OverlayTextAlignment

	// Duration - exactly one must be set
	durationMs int             // Set by either timed duration method
	keyboard   *input.Keyboard // Set by WithUserDismissed()

	// Optional configuration
	mirrorToOutput *text.Output // Optional: also write to output
	dialogStack    *DialogStack // Optional: auto-pop on dismiss
	onDismiss      func()       // Optional: callback on dismiss

	// Validation tracking - ensures all required config is set
	sizingSet           bool
	positionSet         bool
	effectsSet          bool
	contentAlignSet     bool
	textAlignSet        bool
	durationSet         bool // True if WithDuration() called
	externalDurationSet bool // True if WithExternallyManagedDuration() called
	userDismissSet      bool // True if WithUserDismissed() called
}

// NewOverlayDialog creates a new OverlayDialogBuilder with the given message.
// All configuration must be explicitly set before calling Build().
func NewOverlayDialog(message string) *OverlayDialogBuilder {
	return &OverlayDialogBuilder{
		message: message,
	}
}

// =====================================
// REQUIRED: Appearance Configuration
// =====================================

// WithSizing sets the dialog sizing strategy.
// Required - Build() panics if not set (unless using a template).
func (b *OverlayDialogBuilder) WithSizing(s SizingStrategy) *OverlayDialogBuilder {
	b.sizing = s
	b.sizingSet = true

	return b
}

// WithPosition sets the dialog screen position.
// Required - Build() panics if not set (unless using a template).
func (b *OverlayDialogBuilder) WithPosition(p Position) *OverlayDialogBuilder {
	b.position = p
	b.centerX = nil
	b.positionSet = true

	return b
}

// WithHorizontalCenter overrides the horizontal anchor with a screen fraction.
// The configured vertical anchor is unchanged; a later WithPosition resets this override.
func (b *OverlayDialogBuilder) WithHorizontalCenter(x float64) *OverlayDialogBuilder {
	if !(x >= 0 && x <= 1) {
		panic("OverlayDialog.WithHorizontalCenter: expected a screen fraction between 0 and 1")
	}
	b.centerX = &x

	return b
}

// WithEffects sets the fade animation configuration.
// Required - Build() panics if not set (unless using a template).
func (b *OverlayDialogBuilder) WithEffects(e EffectsConfig) *OverlayDialogBuilder {
	b.effects = e
	b.effectsSet = true

	return b
}

// WithContentAlignment sets how content is vertically aligned.
// Required - Build() panics if not set (unless using a template).
func (b *OverlayDialogBuilder) WithContentAlignment(a ContentAlignment) *OverlayDialogBuilder {
	b.contentAlign = a
	b.contentAlignSet = true

	return b
}

// WithTextAlignment sets how text is horizontally aligned.
// Required - Build() panics if not set (unless using a template).
func (b *OverlayDialogBuilder) WithTextAlignment(a OverlayTextAlignment) *OverlayDialogBuilder {
	b.textAlign = a
	b.textAlignSet = true

	return b
}

// =====================================
// REQUIRED: Duration (exactly one)
// =====================================

// WithDuration sets timed auto-dismiss behavior.
// The dialog will automatically hide after the specified duration.
// Cannot be combined with WithUserDismissed() - Build() panics if both are set.
func (b *OverlayDialogBuilder) WithDuration(durationMs int) *OverlayDialogBuilder {
	b.durationMs = durationMs
	b.durationSet = true

	return b
}

// WithExternallyManagedDuration configures a timed dialog whose owner controls expiry.
// OverlayDialog.Update will not advance or hide it. This is intended for owners with
// an authoritative clock that also remove the exact dialog instance from DialogStack.
func (b *OverlayDialogBuilder) WithExternallyManagedDuration(durationMs int) *OverlayDialogBuilder {
	b.durationMs = durationMs
	b.externalDurationSet = true

	return b
}

// WithUserDismissed sets user-dismissed behavior.
// The dialog shows the shared close footer and remains visible until Enter or Escape.
// Cannot be combined with either timed duration mode - Build() panics if combined.
func (b *OverlayDialogBuilder) WithUserDismissed(keyboard *input.Keyboard) *OverlayDialogBuilder {
	b.keyboard = keyboard
	b.userDismissSet = true

	return b
}

// =====================================
// OPTIONAL Configuration
// =====================================

// MirrorToOutput also writes the message to the provided text output.
// Optional - does not affect validation.
func (b *OverlayDialogBuilder) MirrorToOutput(output *text.Output) *OverlayDialogBuilder {
	b.mirrorToOutput = output

	return b
}

// AutoPopFromStack automatically pops the dialog from the stack when dismissed.
//
// IMPORTANT: This only works for user-dismissed dialogs (WithUserDismissed).
// For timed dialogs (WithDuration), use explicit cleanup in Finally():
//
//	.Finally(func() { g.dialogStack.PopModalDialog() })
//
// How it works:
//   - When user presses any key, MessageDialog's InputDismissal triggers
//   - The dismiss callback calls dialogStack.PopModalDialog()
//   - This happens automatically via the callback chain
//
// When NOT to use:
//   - Timed dialogs (use Finally() for cleanup)
//   - When you need the dialog to remain for additional operations
//
// Optional - does not affect validation.
func (b *OverlayDialogBuilder) AutoPopFromStack(stack *DialogStack) *OverlayDialogBuilder {
	b.dialogStack = stack

	return b
}

// OnDismiss sets a callback to execute when the dialog is dismissed.
// Optional - does not affect validation.
func (b *OverlayDialogBuilder) OnDismiss(fn func()) *OverlayDialogBuilder {
	b.onDismiss = fn

	return b
}

// =====================================
// BUILD
// =====================================

// Build creates the OverlayDialog with all configured settings.
// Panics if required configuration is missing or invalid.
//
// Validation rules:
//   - All appearance settings must be set (sizing, position, effects, contentAlign, textAlign)
//   - Exactly one duration method must be called
//   - Timed and user-dismissed duration modes cannot be combined
func (b *OverlayDialogBuilder) Build() *OverlayDialog {
	b.validate()

	return b.build()
}

// validate checks that all required configuration is present and valid.
// Panics with a descriptive message if validation fails.
func (b *OverlayDialogBuilder) validate() {
	var missing []string

	// Check required appearance config
	if !b.sizingSet {
		missing = append(missing, "Sizing (call WithSizing() or use a template)")
	}

	if !b.positionSet {
		missing = append(missing, "Position (call WithPosition() or use a template)")
	}

	if !b.effectsSet {
		missing = append(missing, "Effects (call WithEffects() or use a template)")
	}

	if !b.contentAlignSet {
		missing = append(missing, "ContentAlignment (call WithContentAlignment() or use a template)")
	}

	if !b.textAlignSet {
		missing = append(missing, "TextAlignment (call WithTextAlignment() or use a template)")
	}

	// Check duration - exactly one required
	durationModeCount := 0
	for _, configured := range []bool{b.durationSet, b.externalDurationSet, b.userDismissSet} {
		if configured {
			durationModeCount++
		}
	}

	if durationModeCount > 1 {
		panic("OverlayDialog.Build(): cannot combine duration modes - choose exactly one")
	}

	if durationModeCount == 0 {
		missing = append(missing,
			"Duration (call WithDuration(ms), WithExternallyManagedDuration(ms), or WithUserDismissed(keyboard))")
	}

	if len(missing) > 0 {
		panic(fmt.Sprintf("OverlayDialog.Build(): missing required configuration:\n  - %s",
			strings.Join(missing, "\n  - ")))
	}
}

// build creates the OverlayDialog after validation passes.
func (b *OverlayDialogBuilder) build() *OverlayDialog {
	// Convert our config to MessageDialogBuilder settings
	msgBuilder := NewMessageDialog(b.message)

	// Configure sizing
	lines, charsPerLine, autoFit := b.sizing.GetConfig()
	if autoFit {
		// Use auto-sizing with min chars from config
		if autoSizing, ok := b.sizing.(AutoFitSizing); ok {
			msgBuilder = msgBuilder.WithAutoSize(autoSizing.MaxWidthPercent, 0.6)
		} else {
			msgBuilder = msgBuilder.WithAutoSizeDefault()
		}
	} else {
		msgBuilder = msgBuilder.WithTextLines(lines).WithCharsPerLine(charsPerLine)
	}

	// Configure position
	msgBuilder = msgBuilder.WithHorizontalPosition(b.position.GetHorizontalMode())
	msgBuilder = msgBuilder.WithVerticalPosition(b.position.GetVerticalPercent())
	if b.centerX != nil {
		msgBuilder.centerX = b.centerX
		msgBuilder.screenRelative = true
	}

	// Configure effects
	if b.effects.HasAnyEffect() {
		msgBuilder = msgBuilder.WithEffects(true).WithFadeDurations(b.effects.FadeInMs, b.effects.FadeOutMs)
	} else {
		msgBuilder = msgBuilder.WithEffects(false)
	}

	// Configure content alignment
	msgBuilder = msgBuilder.WithContentTextAlignment(b.contentAlign)

	// Configure text alignment
	msgBuilder = msgBuilder.WithTextHorizontalCenter(b.textAlign == TextAlignCenter)

	// Configure static content (overlay dialogs don't accept mirrored text)
	msgBuilder = msgBuilder.WithStaticContent()

	// Configure input dismissal if user-dismissed
	if b.userDismissSet {
		msgBuilder = msgBuilder.WithInputDismissal(b.keyboard)

		if b.dialogStack == nil && b.onDismiss != nil {
			msgBuilder = msgBuilder.WithDismissCallback(b.onDismiss)
		}
	}

	// Build the underlying MessageDialog
	messageDialog := msgBuilder.Build()

	// Create the OverlayDialog wrapper
	overlay := &OverlayDialog{
		dialog:                    messageDialog,
		message:                   b.message,
		durationMs:                b.durationMs,
		isTimed:                   b.durationSet || b.externalDurationSet,
		durationExternallyManaged: b.externalDurationSet,
		mirrorToOutput:            b.mirrorToOutput,
		onDismiss:                 b.onDismiss,
	}
	if b.dialogStack != nil {
		messageDialog.onDismiss = func() {
			b.dialogStack.MustRemoveWidgetInstance(overlay)
			if b.onDismiss != nil {
				b.onDismiss()
			}
		}
	}

	return overlay
}

// =====================================
// GETTER METHODS (for testing)
// =====================================

// IsSizingSet returns true if sizing has been configured.
func (b *OverlayDialogBuilder) IsSizingSet() bool {
	return b.sizingSet
}

// IsPositionSet returns true if position has been configured.
func (b *OverlayDialogBuilder) IsPositionSet() bool {
	return b.positionSet
}

// IsEffectsSet returns true if effects have been configured.
func (b *OverlayDialogBuilder) IsEffectsSet() bool {
	return b.effectsSet
}

// IsContentAlignmentSet returns true if content alignment has been configured.
func (b *OverlayDialogBuilder) IsContentAlignmentSet() bool {
	return b.contentAlignSet
}

// IsTextAlignmentSet returns true if text alignment has been configured.
func (b *OverlayDialogBuilder) IsTextAlignmentSet() bool {
	return b.textAlignSet
}

// IsDurationSet returns true if self-managed timed duration has been configured.
func (b *OverlayDialogBuilder) IsDurationSet() bool {
	return b.durationSet
}

// IsExternalDurationSet returns true if externally managed timing has been configured.
func (b *OverlayDialogBuilder) IsExternalDurationSet() bool {
	return b.externalDurationSet
}

// IsUserDismissedSet returns true if user-dismissed mode has been configured.
func (b *OverlayDialogBuilder) IsUserDismissedSet() bool {
	return b.userDismissSet
}

// GetSizing returns the configured sizing strategy (nil if not set).
func (b *OverlayDialogBuilder) GetSizing() SizingStrategy {
	return b.sizing
}

// GetPosition returns the configured position.
func (b *OverlayDialogBuilder) GetPosition() Position {
	return b.position
}

// GetEffects returns the configured effects.
func (b *OverlayDialogBuilder) GetEffects() EffectsConfig {
	return b.effects
}

// GetDurationMs returns the configured duration in milliseconds.
func (b *OverlayDialogBuilder) GetDurationMs() int {
	return b.durationMs
}
