package widgets

import (
	"image"
	stdcolor "image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/color"
)

// LoadingIndicator provides a visual loading indicator with spinning animation.
// Supports customizable text, colors, and animation speed.
type LoadingIndicator struct {
	// Core properties
	message   string
	isActive  bool
	isVisible bool

	// Animation properties
	animationFrame float64 // Current animation frame (0-360 for spinner)
	animationSpeed float64 // Animation speed in degrees per frame
	animationStyle LoadingAnimationStyle

	// Display properties
	border    *Border
	font      *text.UltimaFont
	output    *text.Output
	placement sprites.PercentBasedPlacement

	// Visual customization
	backgroundColor stdcolor.Color
	spinnerColor    stdcolor.Color
	textColor       stdcolor.Color

	// Layout properties
	spinnerSize float64 // Spinner size as percentage of area

	// Reusable draw options for safety-first performance
	backgroundDrawOp *ebiten.DrawImageOptions
	spinnerDrawOp    *ebiten.DrawImageOptions
	textDrawOp       *ebiten.DrawImageOptions
}

// LoadingAnimationStyle defines different animation styles
type LoadingAnimationStyle int

const (
	LoadingStyleSpinner LoadingAnimationStyle = iota // Rotating spinner
	LoadingStyleDots                                 // Pulsing dots
	LoadingStyleBars                                 // Moving bars
)

// LoadingIndicatorConfig provides configuration for creating a loading indicator
type LoadingIndicatorConfig struct {
	// Screen placement using percentage-based positioning
	Placement sprites.PercentBasedPlacement

	// Loading message to display
	Message string // Text to show alongside the indicator

	// Animation settings
	AnimationStyle LoadingAnimationStyle // Type of loading animation
	AnimationSpeed float64               // Animation speed (degrees per frame for spinner)

	// Visual customization (optional - safe defaults provided)
	BackgroundColor stdcolor.Color // Background color
	SpinnerColor    stdcolor.Color // Spinner/indicator color
	TextColor       stdcolor.Color // Text color
	SpinnerSize     float64        // Spinner size as percentage of area (default 0.3)

	// Display options
	FontSize float64 // Font size for loading text (default 14)
}

const (
	loadingIndicatorDefaultSpinnerSize = 0.3 // 30% of area
	loadingIndicatorDefaultAnimSpeed   = 5.0 // 5 degrees per frame
	loadingIndicatorBorderWidth        = 101
	loadingIndicatorMaxCharsPerLine    = 30
	loadingIndicatorSpinnerSegments    = 12 // Number of segments in spinner
)

// loadingIndicatorDefaultFontSize uses FontSizeSmall (14) from typography system

// NewLoadingIndicator creates a new loading indicator widget with safe defaults and validation.
//
// The loading indicator provides visual feedback during long-running operations
// with customizable animation styles and text messages.
//
// Parameters:
//   - config: Configuration for the loading indicator widget
//
// Returns a configured LoadingIndicator that implements the Widget interface.
func NewLoadingIndicator(config LoadingIndicatorConfig) *LoadingIndicator {
	// Apply safe defaults
	if config.Message == "" {
		config.Message = "Loading..."
	}

	if config.AnimationSpeed <= 0 {
		config.AnimationSpeed = loadingIndicatorDefaultAnimSpeed
	}

	if config.SpinnerSize <= 0 || config.SpinnerSize > 1.0 {
		logging.Warn(logging.CategoryUI, "NewLoadingIndicator invalid SpinnerSize %f, using default %f", config.SpinnerSize, loadingIndicatorDefaultSpinnerSize)
		config.SpinnerSize = loadingIndicatorDefaultSpinnerSize
	}

	if config.FontSize <= 0 {
		config.FontSize = float64(FontSizeSmall) // Default to 14
	}

	// Apply default colors if not specified
	if config.BackgroundColor == nil {
		config.BackgroundColor = color.LoadingBackground.RGBA() // Semi-transparent dark gray
	}
	if config.SpinnerColor == nil {
		config.SpinnerColor = color.LoadingSpinner.RGBA() // Light blue
	}
	if config.TextColor == nil {
		config.TextColor = color.LoadingText.RGBA() // White
	}

	widget := &LoadingIndicator{
		message:         config.Message,
		isActive:        false,
		isVisible:       true,
		animationFrame:  0.0,
		animationSpeed:  config.AnimationSpeed,
		animationStyle:  config.AnimationStyle,
		placement:       config.Placement,
		backgroundColor: config.BackgroundColor,
		spinnerColor:    config.SpinnerColor,
		textColor:       config.TextColor,
		spinnerSize:     config.SpinnerSize,
	}

	// Initialize display components using centralized typography system
	widget.font, widget.output = GetScaledFontWithOutput(FontConfig{
		Size:     StandardFontSize(config.FontSize),
		MaxLines: 3, // Multiple lines for longer messages
		MaxChars: loadingIndicatorMaxCharsPerLine,
	})

	// Create border for the loading indicator
	widget.border = NewBorder(
		widget.placement,
		loadingIndicatorBorderWidth,
		color.Black,
	)

	// Initialize reusable draw options for safety-first performance
	widget.backgroundDrawOp = &ebiten.DrawImageOptions{}
	widget.spinnerDrawOp = &ebiten.DrawImageOptions{}
	widget.textDrawOp = &ebiten.DrawImageOptions{}

	return widget
}

// Update processes animation updates for the loading indicator
func (l *LoadingIndicator) Update() {
	if !l.isVisible || !l.isActive {
		return
	}

	// Update animation frame
	l.animationFrame += l.animationSpeed

	// Keep animation frame in reasonable bounds
	switch l.animationStyle {
	case LoadingStyleSpinner:
		if l.animationFrame >= 360.0 {
			l.animationFrame -= 360.0
		}
	case LoadingStyleDots, LoadingStyleBars:
		if l.animationFrame >= 360.0 {
			l.animationFrame -= 360.0
		}
	}
}

// Draw renders the loading indicator to the screen
func (l *LoadingIndicator) Draw(screen *ebiten.Image) {
	// Error handling: validate screen parameter
	if screen == nil {
		logging.Warn(logging.CategoryUI, "LoadingIndicator.Draw called with nil screen, skipping draw")

		return
	}

	if !l.isVisible {
		return
	}

	// Draw border and background
	l.border.DrawBackground(screen)

	// Calculate content area
	contentRect := sprites.GetRectangleFromPercents(sprites.PercentBasedPlacement{
		StartPercentX: l.placement.StartPercentX + LayoutSpacingStandard,
		EndPercentX:   l.placement.EndPercentX - LayoutSpacingStandard,
		StartPercentY: l.placement.StartPercentY + LayoutSpacingStandard,
		EndPercentY:   l.placement.EndPercentY - LayoutSpacingStandard,
	})

	// Draw background
	l.drawBackground(screen, *contentRect)

	// Draw loading animation if active
	if l.isActive {
		l.drawLoadingAnimation(screen, *contentRect)
	}

	// Draw loading message
	l.drawMessage(screen, *contentRect)

	// Draw border on top
	l.border.DrawBorder(screen)
}

// drawBackground draws the background of the loading indicator
func (l *LoadingIndicator) drawBackground(screen *ebiten.Image, contentRect image.Rectangle) {
	for y := contentRect.Min.Y; y < contentRect.Max.Y; y++ {
		for x := contentRect.Min.X; x < contentRect.Max.X; x++ {
			screen.Set(x, y, l.backgroundColor)
		}
	}
}

// drawLoadingAnimation draws the appropriate loading animation
func (l *LoadingIndicator) drawLoadingAnimation(screen *ebiten.Image, contentRect image.Rectangle) {
	switch l.animationStyle {
	case LoadingStyleSpinner:
		l.drawSpinner(screen, contentRect)
	case LoadingStyleDots:
		l.drawDots(screen, contentRect)
	case LoadingStyleBars:
		l.drawBars(screen, contentRect)
	}
}

// drawSpinner draws a rotating spinner animation
func (l *LoadingIndicator) drawSpinner(screen *ebiten.Image, contentRect image.Rectangle) {
	centerX := contentRect.Min.X + contentRect.Dx()/2
	centerY := contentRect.Min.Y + contentRect.Dy()/3 // Upper third for spinner

	radius := int(float64(min(contentRect.Dx(), contentRect.Dy())) * l.spinnerSize / 2)

	// Draw spinner segments with varying alpha for trail effect
	for i := 0; i < loadingIndicatorSpinnerSegments; i++ {
		angle := l.animationFrame + float64(i)*30.0 // 30 degrees between segments
		angleRad := angle * math.Pi / 180.0

		// Calculate segment position
		segmentX := centerX + int(float64(radius)*math.Cos(angleRad))
		segmentY := centerY + int(float64(radius)*math.Sin(angleRad))

		// Calculate alpha based on segment position (trailing effect)
		alpha := uint8(255 * (1.0 - float64(i)/float64(loadingIndicatorSpinnerSegments)))
		segmentColor := color.From(l.spinnerColor).Alpha(alpha).RGBA()

		// Draw segment as small circle
		segmentRadius := 3
		for dy := -segmentRadius; dy <= segmentRadius; dy++ {
			for dx := -segmentRadius; dx <= segmentRadius; dx++ {
				if dx*dx+dy*dy <= segmentRadius*segmentRadius {
					x, y := segmentX+dx, segmentY+dy
					if x >= contentRect.Min.X && x < contentRect.Max.X &&
						y >= contentRect.Min.Y && y < contentRect.Max.Y {
						screen.Set(x, y, segmentColor)
					}
				}
			}
		}
	}
}

// drawDots draws pulsing dots animation
func (l *LoadingIndicator) drawDots(screen *ebiten.Image, contentRect image.Rectangle) {
	centerY := contentRect.Min.Y + contentRect.Dy()/3
	dotSpacing := contentRect.Dx() / 6
	startX := contentRect.Min.X + dotSpacing

	for i := 0; i < 4; i++ {
		dotX := startX + i*dotSpacing

		// Calculate pulsing alpha based on animation frame and dot position
		phase := l.animationFrame + float64(i)*90.0               // 90 degree phase offset
		pulseValue := (math.Sin(phase*math.Pi/180.0) + 1.0) / 2.0 // 0-1 range
		alpha := uint8(100 + pulseValue*155)                      // 100-255 range

		dotColor := color.From(l.spinnerColor).Alpha(alpha).RGBA()

		// Draw dot
		dotRadius := 4
		for dy := -dotRadius; dy <= dotRadius; dy++ {
			for dx := -dotRadius; dx <= dotRadius; dx++ {
				if dx*dx+dy*dy <= dotRadius*dotRadius {
					x, y := dotX+dx, centerY+dy
					if x >= contentRect.Min.X && x < contentRect.Max.X &&
						y >= contentRect.Min.Y && y < contentRect.Max.Y {
						screen.Set(x, y, dotColor)
					}
				}
			}
		}
	}
}

// drawBars draws moving bars animation
func (l *LoadingIndicator) drawBars(screen *ebiten.Image, contentRect image.Rectangle) {
	centerY := contentRect.Min.Y + contentRect.Dy()/3
	barWidth := 4
	barSpacing := 8
	numBars := contentRect.Dx() / barSpacing
	startX := contentRect.Min.X + (contentRect.Dx()-numBars*barSpacing)/2

	for i := 0; i < numBars; i++ {
		barX := startX + i*barSpacing

		// Calculate bar height based on animation frame and position
		phase := l.animationFrame + float64(i)*30.0
		heightMultiplier := (math.Sin(phase*math.Pi/180.0) + 1.0) / 2.0
		maxBarHeight := int(float64(contentRect.Dy()) * l.spinnerSize)
		barHeight := int(heightMultiplier * float64(maxBarHeight))

		// Draw bar
		barTop := centerY - barHeight/2
		barBottom := centerY + barHeight/2

		for y := barTop; y <= barBottom; y++ {
			for x := barX; x < barX+barWidth; x++ {
				if x >= contentRect.Min.X && x < contentRect.Max.X &&
					y >= contentRect.Min.Y && y < contentRect.Max.Y {
					screen.Set(x, y, l.spinnerColor)
				}
			}
		}
	}
}

// drawMessage draws the loading message text
func (l *LoadingIndicator) drawMessage(screen *ebiten.Image, contentRect image.Rectangle) {
	// Position text in bottom two-thirds of the area
	// Use center point, then convert to top-left for DrawPanelContent
	centerY := contentRect.Min.Y + (2*contentRect.Dy())/3
	centerX := contentRect.Min.X + contentRect.Dx()/2

	// Container width for centered text (use content rect width)
	containerWidth := contentRect.Dx()
	topLeftX := centerX - containerWidth/2

	// Build and render using Output pattern
	l.output.Reset()
	l.output.AppendLine().Text(l.message, l.textColor).WithAlignment(etext.AlignCenter).Done()
	l.output.DrawPanelContent(screen, image.Pt(topLeftX, centerY), containerWidth, etext.AlignCenter)
}

// Start begins the loading animation
func (l *LoadingIndicator) Start() {
	l.isActive = true
}

// Stop ends the loading animation
func (l *LoadingIndicator) Stop() {
	l.isActive = false
}

// IsActive returns whether the loading animation is active
func (l *LoadingIndicator) IsActive() bool {
	return l.isActive
}

// SetMessage updates the loading message
func (l *LoadingIndicator) SetMessage(message string) {
	l.message = message
}

// GetMessage returns the current loading message
func (l *LoadingIndicator) GetMessage() string {
	return l.message
}

// SetVisible sets whether the loading indicator is visible
func (l *LoadingIndicator) SetVisible(visible bool) {
	l.isVisible = visible
}

// IsVisible returns whether the loading indicator is visible
func (l *LoadingIndicator) IsVisible() bool {
	return l.isVisible
}

// SetAnimationSpeed sets the animation speed
func (l *LoadingIndicator) SetAnimationSpeed(speed float64) {
	if speed < 0 {
		logging.Warn(logging.CategoryUI, "LoadingIndicator.SetAnimationSpeed negative speed %f, using absolute value", speed)
		speed = -speed
	}
	l.animationSpeed = speed
}

// GetAnimationSpeed returns the current animation speed (test accessor)
func (l *LoadingIndicator) GetAnimationSpeed() float64 {
	return l.animationSpeed
}

// SetAnimationStyle sets the animation style (test accessor)
func (l *LoadingIndicator) SetAnimationStyle(style LoadingAnimationStyle) {
	l.animationStyle = style
	l.animationFrame = 0.0 // Reset animation
}

// GetAnimationStyle returns the current animation style (test accessor)
func (l *LoadingIndicator) GetAnimationStyle() LoadingAnimationStyle {
	return l.animationStyle
}

// SetColors updates the loading indicator colors
func (l *LoadingIndicator) SetColors(background, spinner, text stdcolor.Color) {
	if background != nil {
		l.backgroundColor = background
	}
	if spinner != nil {
		l.spinnerColor = spinner
	}
	if text != nil {
		l.textColor = text
	}
}

// Reset resets the animation frame to the beginning
func (l *LoadingIndicator) Reset() {
	l.animationFrame = 0.0
}
