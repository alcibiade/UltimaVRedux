package widgets

// =====================================
// SIZING STRATEGY
// =====================================

// SizingStrategy defines how the dialog dimensions are calculated.
// Exactly one sizing strategy must be provided to the builder.
type SizingStrategy interface {
	isSizingStrategy()
	// GetConfig returns the resolved sizing configuration
	GetConfig() (lines int, charsPerLine int, autoFit bool)
}

// AutoFitSizing sizes the dialog to fit its content.
type AutoFitSizing struct {
	MaxWidthPercent float64 // 0.0-1.0, maximum width as percentage of screen
	MinCharsPerLine int     // Minimum characters per line for aesthetics
}

func (AutoFitSizing) isSizingStrategy() {}

// GetConfig reports auto-fit with zero lines; MinCharsPerLine is the width floor.
func (s AutoFitSizing) GetConfig() (int, int, bool) {
	// For auto-fit, lines and charsPerLine are calculated from content
	// MinCharsPerLine is used as the floor
	return 0, s.MinCharsPerLine, true
}

// FixedLinesSizing uses explicit line count and characters per line.
type FixedLinesSizing struct {
	Lines        int // Number of text lines
	CharsPerLine int // Characters per line
}

func (FixedLinesSizing) isSizingStrategy() {}

// GetConfig returns the fixed line and width counts with auto-fit off.
func (s FixedLinesSizing) GetConfig() (int, int, bool) {
	return s.Lines, s.CharsPerLine, false
}

// Sizes of the pre-defined presets.
const (
	autoFitDefaultMaxWidthPercent = 0.8
	autoFitDefaultMinCharsPerLine = 15
	compactLines                  = 2
	compactCharsPerLine           = 35
)

// Pre-defined sizing presets.
//
//nolint:gochecknoglobals // Sizing presets are immutable shared UI configuration.
var (
	// SizingAutoFitDefault auto-fits content with 80% max width and 15 char minimum
	SizingAutoFitDefault = AutoFitSizing{
		MaxWidthPercent: autoFitDefaultMaxWidthPercent, MinCharsPerLine: autoFitDefaultMinCharsPerLine,
	}

	// SizingCompact is a small fixed-size dialog (2 lines, 35 chars)
	SizingCompact = FixedLinesSizing{Lines: compactLines, CharsPerLine: compactCharsPerLine}
)

// =====================================
// POSITION
// =====================================

// HorizontalAnchor defines horizontal positioning strategy.
type HorizontalAnchor int

const (
	// HorizontalMapCenter centers the dialog in the map area (37%)
	HorizontalMapCenter HorizontalAnchor = iota
	// HorizontalScreenCenter centers the dialog on the full screen (50%)
	HorizontalScreenCenter
)

// VerticalAnchor defines vertical positioning strategy.
type VerticalAnchor int

const (
	// VerticalTop positions near the top of the screen
	VerticalTop VerticalAnchor = iota
	// VerticalCenter positions at the center of the screen
	VerticalCenter
	// VerticalLower positions at 70% down (typical overlay position)
	VerticalLower
	// VerticalBottom positions near the bottom of the screen
	VerticalBottom
)

// Position defines the dialog's screen position.
type Position struct {
	Horizontal HorizontalAnchor
	Vertical   VerticalAnchor
}

// Screen heights, as fractions from the top, at which each VerticalAnchor places a dialog.
const (
	verticalTopPercent    = 0.15
	verticalCenterPercent = 0.50
	verticalLowerPercent  = 0.70
	verticalBottomPercent = 0.85
)

// GetVerticalPercent returns the vertical position as a percentage (0.0-1.0)
func (p Position) GetVerticalPercent() float64 {
	switch p.Vertical {
	case VerticalTop:
		return verticalTopPercent
	case VerticalCenter:
		return verticalCenterPercent
	case VerticalLower:
		return verticalLowerPercent
	case VerticalBottom:
		return verticalBottomPercent
	default:
		return verticalLowerPercent
	}
}

// GetHorizontalMode returns the DialogHorizontalPosition for MessageDialog compatibility
func (p Position) GetHorizontalMode() DialogHorizontalPosition {
	switch p.Horizontal {
	case HorizontalMapCenter:
		return DialogHorizontalMapOnly
	case HorizontalScreenCenter:
		return DialogHorizontalWholeScreen
	default:
		return DialogHorizontalWholeScreen
	}
}

// Pre-defined position presets.
//
//nolint:gochecknoglobals // Position presets are immutable shared UI configuration.
var (
	// PositionMapCenterLower is the typical overlay position (map area, 70% down)
	PositionMapCenterLower = Position{HorizontalMapCenter, VerticalLower}

	// PositionScreenCenter is centered on the full screen
	PositionScreenCenter = Position{HorizontalScreenCenter, VerticalCenter}

	// PositionMapCenterBottom keeps brief feedback clear of the primary map view.
	PositionMapCenterBottom = Position{HorizontalMapCenter, VerticalBottom}
)

// =====================================
// EFFECTS
// =====================================

// EffectsConfig defines fade-in and fade-out animation timing.
type EffectsConfig struct {
	FadeInMs  int // Milliseconds for fade-in (0 = instant)
	FadeOutMs int // Milliseconds for fade-out (0 = instant)
}

// HasFadeIn returns true if fade-in animation is enabled
func (e EffectsConfig) HasFadeIn() bool {
	return e.FadeInMs > 0
}

// HasFadeOut returns true if fade-out animation is enabled
func (e EffectsConfig) HasFadeOut() bool {
	return e.FadeOutMs > 0
}

// HasAnyEffect returns true if any animation effect is enabled
func (e EffectsConfig) HasAnyEffect() bool {
	return e.HasFadeIn() || e.HasFadeOut()
}

// Pre-defined effect presets.
//
//nolint:gochecknoglobals // Effect presets are immutable shared UI configuration.
var (
	// EffectsNone disables all fade effects (instant show/hide)
	EffectsNone = EffectsConfig{FadeInMs: 0, FadeOutMs: 0}
)

// =====================================
// TEXT ALIGNMENT
// =====================================

// OverlayTextAlignment defines how text is horizontally aligned within lines.
type OverlayTextAlignment int

const (
	// TextAlignLeft aligns text to the left
	TextAlignLeft OverlayTextAlignment = iota
	// TextAlignCenter centers text horizontally
	TextAlignCenter
)

// NOTE: ContentAlignment is defined in message_dialog_core.go and reused here:
// - ContentAlignBottomUp: Terminal-style (newest at bottom)
// - ContentAlignCenter: Vertically centered
// - ContentAlignTop: Top-aligned
