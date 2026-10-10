package widgets

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ultimacolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// Button font sizes use typography system - these map to StandardFontSize:
// SmallButton (15) ≈ FontSizeSmall (14) - closest semantic match
// MediumButton (20) = FontSizeLarge (20) - exact match
// LargeButton (30) = FontSizeHuge (30) - exact match

var _ Widget = &Button{}

type ButtonSize int

const (
	SmallButton ButtonSize = iota
	MediumButton
	LargeButton
	TabButton // Taller button for tab panels (50% taller than SmallButton)
)

// Button font sizes - used by getFontSize() for typography system integration
// These are legacy values preserved for visual consistency with existing UI
const (
	smallButtonFontPoint  = 15                              // Custom 15pt (between FontSizeSmall 14 and FontSizeMedium 16)
	mediumButtonFontPoint = StandardFontSize(FontSizeLarge) // 20pt
	largeButtonFontPoint  = StandardFontSize(FontSizeHuge)  // 30pt
	buttonMaxChars        = 20
)

var (
	smallButtonPercentHeight  = 0.05
	smallButtonPercentWidth   = 0.15
	mediumButtonPercentHeight = 0.065
	mediumButtonPercentWidth  = 0.2
	largeButtonPercentHeight  = 0.08
	largeButtonPercentWidth   = 0.3
	tabButtonPercentHeight    = 0.1125 // 125% taller than SmallButton (was 0.075, +50% again)
)

// Line spacing uses 1:1 ratio (same as font size) - handled by typography system default

type ButtonStatus int

const (
	ButtonNormal   ButtonStatus = iota // Default state
	ButtonSelected                     // Button is selected/focused
	ButtonDisabled                     // Button is disabled
)

// Legacy compatibility
const (
	NotSelected = ButtonNormal
	Selected    = ButtonSelected
	Disabled    = ButtonDisabled
)

type Button struct {
	text         string
	callback     func()
	border       *Border
	size         ButtonSize
	centerPoint  sprites.PercentBasedCenterPoint
	pbp          sprites.PercentBasedPlacement
	buttonStatus ButtonStatus

	font   *text.UltimaFont
	Output *text.Output

	// Visual state management
	interiorColor color.Color
	textColor     color.Color

	// State configuration
	enabled bool
}

// NewButton creates a new button widget with the specified text, callback, and layout.
//
// The button handles rendering and border drawing based on the provided size
// and positioning parameters. Its owner invokes the callback when activated.
//
// Parameters:
//   - buttonText: Text displayed on the button
//   - callback: Function called when button is activated
//   - centerPoint: Screen position as percentage-based coordinates
//   - size: ButtonSize enum determining dimensions and appearance
//
// Returns a configured Button that implements the Widget interface.
func NewButton(
	buttonText string,
	callback func(),
	centerPoint sprites.PercentBasedCenterPoint,
	size ButtonSize,
) *Button {
	button := &Button{}
	button.text = buttonText
	button.callback = callback
	button.size = size
	button.enabled = true
	button.buttonStatus = ButtonNormal
	button.interiorColor = ultimacolor.LighterBlackSemi
	button.textColor = ultimacolor.White
	button.setPosition(centerPoint)
	button.initializeBorder(button.interiorColor)

	// Initialize font using typography system
	button.font, button.Output = GetScaledFontWithOutput(FontConfig{
		Size:     button.getFontSize(),
		MaxLines: 1,
		MaxChars: buttonMaxChars,
	})

	button.refreshTextOutput()

	return button
}

func (b *Button) setPosition(centerPoint sprites.PercentBasedCenterPoint) {
	b.centerPoint = centerPoint
	switch b.size {
	case SmallButton, MediumButton, LargeButton:
		b.pbp = sprites.PercentBasedPlacement{
			StartPercentX: centerPoint.X - b.GetWidthPercent()/2,
			EndPercentX:   centerPoint.X + b.GetWidthPercent()/2,
			StartPercentY: centerPoint.Y - b.GetHeightPercent()/2,
			EndPercentY:   centerPoint.Y + b.GetHeightPercent()/2,
		}
	default:
		// UI widget recovery: unknown button size, defaulting to medium for graceful degradation
		logging.Warn(logging.CategoryUI, "unknown button size %v, defaulting to medium", b.size)
		b.size = MediumButton
		b.pbp = sprites.PercentBasedPlacement{
			StartPercentX: centerPoint.X - b.GetWidthPercent()/2,
			EndPercentX:   centerPoint.X + b.GetWidthPercent()/2,
			StartPercentY: centerPoint.Y - b.GetHeightPercent()/2,
			EndPercentY:   centerPoint.Y + b.GetHeightPercent()/2,
		}
	}
	b.initializeBorder(b.interiorColor)
}

func (b *Button) initializeBorder(color color.Color) {
	b.border = newButtonBorder(b.pbp, color)
}

func (b *Button) Draw(screen *ebiten.Image) {
	b.border.Draw(screen)

	// const percentIntoBorder = 0.02
	textRect := sprites.GetRectangleFromPercents(
		b.pbp,
	)
	textRectWidth := textRect.Dx()

	// Measure actual text height for proper vertical centering
	_, textHeight := etext.Measure(b.text, b.font.TextFace, 0)

	// Calculate centered Y position: top + (button_height - text_height) / 2
	// This ensures text is mathematically centered within the button
	centeredY := float64(textRect.Min.Y) + (float64(textRect.Dy())-textHeight)/2

	b.Output.DrawScrollingBuffer(screen, image.Point{
		X: textRect.Min.X,
		Y: int(centeredY),
	}, textRectWidth, false, etext.AlignCenter, etext.AlignCenter)
}

// Update implements Widget; button activation is owner-driven.
func (b *Button) Update() {}

// updateVisualState sets colors and appearance based on current button state
func (b *Button) updateVisualState() {
	switch b.buttonStatus {
	case ButtonNormal:
		b.interiorColor = ultimacolor.LighterBlackSemi
		b.textColor = ultimacolor.White
	case ButtonSelected:
		b.interiorColor = ultimacolor.UltimaBlue
		b.textColor = ultimacolor.White
	case ButtonDisabled:
		b.interiorColor = ultimacolor.LighterBlackSemi
		b.textColor = ultimacolor.ButtonTextDisabledGray.RGBA() // Gray text
	default:
		// Fallback to normal state
		b.interiorColor = ultimacolor.LighterBlackSemi
		b.textColor = ultimacolor.White
	}

	// Update border with new color
	b.initializeBorder(b.interiorColor)
	b.refreshTextOutput()
}

func (b *Button) refreshTextOutput() {
	b.Output.Reset()
	b.Output.AddTextWithColor(b.text, b.textColor)
}

func (b *Button) SetButtonStatus(buttonStatus ButtonStatus) {
	b.buttonStatus = buttonStatus

	// Handle special cases for disabled state
	if buttonStatus == ButtonDisabled {
		b.enabled = false
	} else if !b.enabled && buttonStatus != ButtonDisabled {
		b.enabled = true
	}

	// Update visual appearance
	b.updateVisualState()
}

// SetEnabled enables or disables the button
func (b *Button) SetEnabled(enabled bool) {
	b.enabled = enabled
	if !enabled {
		b.buttonStatus = ButtonDisabled
	} else if b.buttonStatus == ButtonDisabled {
		b.buttonStatus = ButtonNormal
	}
	b.updateVisualState()
}

// IsEnabled returns whether the button is currently enabled
func (b *Button) IsEnabled() bool {
	return b.enabled
}

// GetButtonStatus returns the current button status (test accessor)
func (b *Button) GetButtonStatus() ButtonStatus {
	return b.buttonStatus
}

// SetCallback updates the button's callback function
func (b *Button) SetCallback(callback func()) {
	b.callback = callback
}

// Activate invokes the configured callback when the button is enabled.
func (b *Button) Activate() {
	if b.enabled && b.callback != nil {
		b.callback()
	}
}

// GetText returns the button's text
func (b *Button) GetText() string {
	return b.text
}

// SetText updates the button's text and refreshes the display
func (b *Button) SetText(text string) {
	b.text = text
	b.refreshTextOutput()
}

// SetFontSize replaces the button's scaled font while preserving its text.
func (b *Button) SetFontSize(fontSize StandardFontSize) {
	b.font, b.Output = GetScaledFontWithOutput(FontConfig{
		Size:                  fontSize,
		LineSpacing:           0,
		LineSpacingMultiplier: 0,
		MaxLines:              1,
		MaxChars:              buttonMaxChars,
	})
	b.refreshTextOutput()
}

// FocusableItem interface implementation
// GetPlacement returns the button's screen placement for focus indication
func (b *Button) GetPlacement() sprites.PercentBasedPlacement {
	return b.pbp
}

// SetPlacement overrides button placement for panel-relative layouts.
// Use this when button dimensions should be relative to a container rather than screen.
func (b *Button) SetPlacement(pbp sprites.PercentBasedPlacement) {
	b.pbp = pbp
	b.initializeBorder(b.interiorColor)
}

// CanReceiveFocus returns whether the button can receive focus
func (b *Button) CanReceiveFocus() bool {
	return b.enabled
}

// OnFocusGained is called when the button receives focus
func (b *Button) OnFocusGained() {
	if b.enabled {
		b.buttonStatus = ButtonSelected
		b.updateVisualState()
	}
}

// OnFocusLost is called when the button loses focus
func (b *Button) OnFocusLost() {
	if b.buttonStatus == ButtonSelected {
		b.buttonStatus = ButtonNormal
		b.updateVisualState()
	}
}

func (b *Button) GetHeightPercent() float64 {
	return GetButtonHeightPercent(b.size)
}

func (b *Button) GetWidthPercent() float64 {
	return GetButtonWidthPercent(b.size)
}

func GetButtonWidthPercent(size ButtonSize) float64 {
	switch size {
	case SmallButton:
		return smallButtonPercentWidth
	case MediumButton:
		return mediumButtonPercentWidth
	case LargeButton:
		return largeButtonPercentWidth
	}
	// Note: TabButton width is calculated dynamically in panel-relative layouts

	return -1
}

func GetButtonHeightPercent(size ButtonSize) float64 {
	switch size {
	case SmallButton:
		return smallButtonPercentHeight
	case MediumButton:
		return mediumButtonPercentHeight
	case LargeButton:
		return largeButtonPercentHeight
	case TabButton:
		return tabButtonPercentHeight
	}

	return -1
}

// getFontSize returns the StandardFontSize for this button's size.
// Uses typography system for consistent font handling.
func (b *Button) getFontSize() StandardFontSize {
	switch b.size {
	case SmallButton, TabButton:
		return StandardFontSize(smallButtonFontPoint) // 15 - preserves original size
	case MediumButton:
		return FontSizeLarge // 20 - exact match
	case LargeButton:
		return FontSizeHuge // 30 - exact match
	}

	return FontSizeLarge // Default to medium
}
