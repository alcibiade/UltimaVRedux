package widgets

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// pulseProgress returns a value from 0.0 to 1.0 based on animation phase.
// The phase is typically animationTime * animationSpeed.
func pulseProgress(phase float64) float64 {
	return (math.Sin(phase*2*math.Pi) + 1) / 2
}

// FocusIndicatorStyle defines different visual styles for focus indicators
type FocusIndicatorStyle int

const (
	FocusStyleBorder    FocusIndicatorStyle = iota // Solid border around focused element
	FocusStyleGlow                                 // Glowing border effect
	FocusStyleUnderline                            // Underline for text-based elements
	FocusStyleHighlight                            // Background highlight
)

// FocusIndicatorConfig configures the appearance of focus indicators
type FocusIndicatorConfig struct {
	// Visual style
	Style FocusIndicatorStyle

	// Colors
	PrimaryColor   color.RGBA // Main focus color
	SecondaryColor color.RGBA // Secondary color for gradients/glows

	// Border settings (for border and glow styles)
	BorderWidth   float64 // Width of focus border
	GlowIntensity float64 // Intensity of glow effect (0.0-1.0)

	// Animation settings
	AnimationSpeed float64 // Speed of focus animations (pulses/second)
	PulseEffect    bool    // Whether to animate with pulsing

	// Positioning
	Padding float64 // Extra space around focused element
}

// DefaultFocusConfig provides standard focus indicator settings
var DefaultFocusConfig = FocusIndicatorConfig{
	Style:          FocusStyleBorder,
	PrimaryColor:   pkgcolor.FocusIndicatorBlue.RGBA(),     // Cornflower blue
	SecondaryColor: pkgcolor.FocusIndicatorSemiBlue.RGBA(), // Semi-transparent
	BorderWidth:    2.0,
	GlowIntensity:  0.7,
	AnimationSpeed: 2.0,
	PulseEffect:    true,
	Padding:        2.0,
}

// AccessibilityFocusConfig provides high-contrast focus indicators for accessibility
var AccessibilityFocusConfig = FocusIndicatorConfig{
	Style:          FocusStyleBorder,
	PrimaryColor:   pkgcolor.FocusHighContrastYellow.RGBA(), // High contrast yellow
	SecondaryColor: pkgcolor.FocusBlackOutline.RGBA(),       // Black outline
	BorderWidth:    3.0,
	GlowIntensity:  1.0,
	AnimationSpeed: 1.5,
	PulseEffect:    false, // No animation for accessibility
	Padding:        3.0,
}

// FocusIndicator manages visual feedback for focused UI elements
type FocusIndicator struct {
	config        FocusIndicatorConfig
	isVisible     bool
	animationTime float64
	placement     sprites.PercentBasedPlacement

	// Pre-allocated drawing resources
	borderImage    *ebiten.Image
	glowImage      *ebiten.Image
	highlightImage *ebiten.Image
}

// NewFocusIndicator creates a new focus indicator with the given configuration
func NewFocusIndicator(config FocusIndicatorConfig) *FocusIndicator {
	// Validate and apply defaults
	if config.AnimationSpeed <= 0 {
		config.AnimationSpeed = DefaultFocusConfig.AnimationSpeed
	}
	if config.BorderWidth <= 0 {
		config.BorderWidth = DefaultFocusConfig.BorderWidth
	}
	if config.GlowIntensity < 0 {
		config.GlowIntensity = 0
	} else if config.GlowIntensity > 1 {
		config.GlowIntensity = 1
	}
	if config.Padding < 0 {
		config.Padding = 0
	}

	return &FocusIndicator{
		config:    config,
		isVisible: false,
	}
}

// NewDefaultFocusIndicator creates a focus indicator with default settings
func NewDefaultFocusIndicator() *FocusIndicator {
	return NewFocusIndicator(DefaultFocusConfig)
}

// NewAccessibilityFocusIndicator creates a focus indicator optimized for accessibility
func NewAccessibilityFocusIndicator() *FocusIndicator {
	return NewFocusIndicator(AccessibilityFocusConfig)
}

// SetFocusTarget sets the target element for focus indication
func (fi *FocusIndicator) SetFocusTarget(placement sprites.PercentBasedPlacement) {
	// Expand placement by padding
	fi.placement = sprites.PercentBasedPlacement{
		StartPercentX: placement.StartPercentX - fi.config.Padding/100.0,
		EndPercentX:   placement.EndPercentX + fi.config.Padding/100.0,
		StartPercentY: placement.StartPercentY - fi.config.Padding/100.0,
		EndPercentY:   placement.EndPercentY + fi.config.Padding/100.0,
	}

	// Clamp to screen bounds
	if fi.placement.StartPercentX < 0 {
		fi.placement.StartPercentX = 0
	}
	if fi.placement.StartPercentY < 0 {
		fi.placement.StartPercentY = 0
	}
	if fi.placement.EndPercentX > 1 {
		fi.placement.EndPercentX = 1
	}
	if fi.placement.EndPercentY > 1 {
		fi.placement.EndPercentY = 1
	}
}

// Show makes the focus indicator visible
func (fi *FocusIndicator) Show() {
	fi.isVisible = true
	fi.animationTime = 0
}

// Hide makes the focus indicator invisible
func (fi *FocusIndicator) Hide() {
	fi.isVisible = false
}

// IsVisible returns whether the focus indicator is currently visible
func (fi *FocusIndicator) IsVisible() bool {
	return fi.isVisible
}

// Update advances the focus indicator animation
func (fi *FocusIndicator) Update(elapsedMs uint64) {
	if !fi.isVisible {
		return
	}

	if fi.config.PulseEffect {
		fi.animationTime += float64(elapsedMs) / 1000.0 // Convert to seconds
	}
}

// Draw renders the focus indicator
func (fi *FocusIndicator) Draw(screen *ebiten.Image, dm *display.DisplayManager) {
	if !fi.isVisible {
		return
	}

	switch fi.config.Style {
	case FocusStyleBorder:
		fi.drawBorder(screen, dm)
	case FocusStyleGlow:
		fi.drawGlow(screen, dm)
	case FocusStyleUnderline:
		fi.drawUnderline(screen, dm)
	case FocusStyleHighlight:
		fi.drawHighlight(screen, dm)
	}
}

// drawBorder draws a solid border focus indicator
func (fi *FocusIndicator) drawBorder(screen *ebiten.Image, dm *display.DisplayManager) {
	screenW, screenH := dm.GetCurrentSize()

	// Calculate pixel coordinates
	x1 := int(fi.placement.StartPercentX * float64(screenW))
	y1 := int(fi.placement.StartPercentY * float64(screenH))
	x2 := int(fi.placement.EndPercentX * float64(screenW))
	y2 := int(fi.placement.EndPercentY * float64(screenH))

	borderWidth := int(fi.config.BorderWidth)

	// Calculate animation intensity
	intensity := 1.0
	if fi.config.PulseEffect {
		// Pulse between 0.6 and 1.0 intensity
		intensity = 0.6 + 0.4*pulseProgress(fi.animationTime*fi.config.AnimationSpeed)
	}

	// Apply intensity to color
	focusColor := fi.config.PrimaryColor
	focusColor.A = uint8(float64(focusColor.A) * intensity)

	// Create border image if needed
	if fi.borderImage == nil {
		fi.borderImage = ebiten.NewImage(1, 1)
	}
	fi.borderImage.Fill(focusColor)

	opts := &ebiten.DrawImageOptions{}

	// Draw top border
	opts.GeoM.Reset()
	opts.GeoM.Scale(float64(x2-x1), float64(borderWidth))
	opts.GeoM.Translate(float64(x1), float64(y1))
	screen.DrawImage(fi.borderImage, opts)

	// Draw bottom border
	opts.GeoM.Reset()
	opts.GeoM.Scale(float64(x2-x1), float64(borderWidth))
	opts.GeoM.Translate(float64(x1), float64(y2-borderWidth))
	screen.DrawImage(fi.borderImage, opts)

	// Draw left border
	opts.GeoM.Reset()
	opts.GeoM.Scale(float64(borderWidth), float64(y2-y1))
	opts.GeoM.Translate(float64(x1), float64(y1))
	screen.DrawImage(fi.borderImage, opts)

	// Draw right border
	opts.GeoM.Reset()
	opts.GeoM.Scale(float64(borderWidth), float64(y2-y1))
	opts.GeoM.Translate(float64(x2-borderWidth), float64(y1))
	screen.DrawImage(fi.borderImage, opts)
}

// drawGlow draws a glowing border focus indicator
func (fi *FocusIndicator) drawGlow(screen *ebiten.Image, dm *display.DisplayManager) {
	screenW, screenH := dm.GetCurrentSize()

	// Calculate pixel coordinates
	x1 := int(fi.placement.StartPercentX * float64(screenW))
	y1 := int(fi.placement.StartPercentY * float64(screenH))
	x2 := int(fi.placement.EndPercentX * float64(screenW))
	y2 := int(fi.placement.EndPercentY * float64(screenH))

	glowSize := int(fi.config.BorderWidth * 3) // Glow extends beyond border

	// Calculate animation intensity
	intensity := fi.config.GlowIntensity
	if fi.config.PulseEffect {
		intensity = intensity * (0.7 + 0.3*pulseProgress(fi.animationTime*fi.config.AnimationSpeed))
	}

	// Draw multiple layers for glow effect
	for layer := glowSize; layer > 0; layer-- {
		alpha := uint8(float64(fi.config.SecondaryColor.A) * intensity * float64(layer) / float64(glowSize))

		glowColor := fi.config.SecondaryColor
		glowColor.A = alpha

		// Create glow image if needed
		if fi.glowImage == nil {
			fi.glowImage = ebiten.NewImage(1, 1)
		}
		fi.glowImage.Fill(glowColor)

		opts := &ebiten.DrawImageOptions{}

		// Draw glow layers expanding outward
		expandX := layer
		expandY := layer

		// Top
		opts.GeoM.Reset()
		opts.GeoM.Scale(float64(x2-x1+2*expandX), float64(layer))
		opts.GeoM.Translate(float64(x1-expandX), float64(y1-expandY))
		screen.DrawImage(fi.glowImage, opts)

		// Bottom
		opts.GeoM.Reset()
		opts.GeoM.Scale(float64(x2-x1+2*expandX), float64(layer))
		opts.GeoM.Translate(float64(x1-expandX), float64(y2+expandY-layer))
		screen.DrawImage(fi.glowImage, opts)

		// Left
		opts.GeoM.Reset()
		opts.GeoM.Scale(float64(layer), float64(y2-y1))
		opts.GeoM.Translate(float64(x1-expandX), float64(y1))
		screen.DrawImage(fi.glowImage, opts)

		// Right
		opts.GeoM.Reset()
		opts.GeoM.Scale(float64(layer), float64(y2-y1))
		opts.GeoM.Translate(float64(x2+expandX-layer), float64(y1))
		screen.DrawImage(fi.glowImage, opts)
	}

	// Draw solid border on top
	fi.drawBorder(screen, dm)
}

// drawUnderline draws an underline focus indicator
func (fi *FocusIndicator) drawUnderline(screen *ebiten.Image, dm *display.DisplayManager) {
	screenW, screenH := dm.GetCurrentSize()

	// Calculate pixel coordinates for underline
	x1 := int(fi.placement.StartPercentX * float64(screenW))
	x2 := int(fi.placement.EndPercentX * float64(screenW))
	y := int(fi.placement.EndPercentY * float64(screenH))

	height := int(fi.config.BorderWidth)

	// Calculate animation intensity
	intensity := 1.0
	if fi.config.PulseEffect {
		intensity = 0.7 + 0.3*pulseProgress(fi.animationTime*fi.config.AnimationSpeed)
	}

	// Apply intensity to color
	focusColor := fi.config.PrimaryColor
	focusColor.A = uint8(float64(focusColor.A) * intensity)

	// Create border image if needed
	if fi.borderImage == nil {
		fi.borderImage = ebiten.NewImage(1, 1)
	}
	fi.borderImage.Fill(focusColor)

	// Draw underline
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(float64(x2-x1), float64(height))
	opts.GeoM.Translate(float64(x1), float64(y))
	screen.DrawImage(fi.borderImage, opts)
}

// drawHighlight draws a background highlight focus indicator
func (fi *FocusIndicator) drawHighlight(screen *ebiten.Image, dm *display.DisplayManager) {
	screenW, screenH := dm.GetCurrentSize()

	// Calculate pixel coordinates
	x1 := int(fi.placement.StartPercentX * float64(screenW))
	y1 := int(fi.placement.StartPercentY * float64(screenH))
	x2 := int(fi.placement.EndPercentX * float64(screenW))
	y2 := int(fi.placement.EndPercentY * float64(screenH))

	// Calculate animation intensity
	intensity := 0.3 // Base highlight intensity
	if fi.config.PulseEffect {
		intensity = 0.2 + 0.2*pulseProgress(fi.animationTime*fi.config.AnimationSpeed)
	}

	// Apply intensity to color
	highlightColor := fi.config.SecondaryColor
	highlightColor.A = uint8(float64(fi.config.SecondaryColor.A) * intensity)

	// Create highlight image if needed
	if fi.highlightImage == nil {
		fi.highlightImage = ebiten.NewImage(1, 1)
	}
	fi.highlightImage.Fill(highlightColor)

	// Draw highlight background
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(float64(x2-x1), float64(y2-y1))
	opts.GeoM.Translate(float64(x1), float64(y1))
	screen.DrawImage(fi.highlightImage, opts)
}

// FocusManager manages focus state across multiple UI elements
type FocusManager struct {
	focusIndicator *FocusIndicator
	focusableItems []FocusableItem
	currentFocus   int
	enabled        bool
}

// FocusableItem represents an item that can receive focus
type FocusableItem interface {
	GetPlacement() sprites.PercentBasedPlacement
	CanReceiveFocus() bool
	OnFocusGained()
	OnFocusLost()
}

// NewFocusManager creates a new focus management system
func NewFocusManager(indicator *FocusIndicator) *FocusManager {
	if indicator == nil {
		indicator = NewDefaultFocusIndicator()
	}

	return &FocusManager{
		focusIndicator: indicator,
		currentFocus:   -1,
		enabled:        true,
	}
}

// AddFocusableItem adds an item to the focus management system (test helper)
func (fm *FocusManager) AddFocusableItem(item FocusableItem) {
	fm.focusableItems = append(fm.focusableItems, item)
}

// RemoveFocusableItem removes an item from the focus management system (test helper)
func (fm *FocusManager) RemoveFocusableItem(item FocusableItem) {
	for i, existing := range fm.focusableItems {
		if existing == item {
			fm.focusableItems = append(fm.focusableItems[:i], fm.focusableItems[i+1:]...)
			if fm.currentFocus == i {
				fm.currentFocus = -1
				fm.focusIndicator.Hide()
			} else if fm.currentFocus > i {
				fm.currentFocus--
			}

			break
		}
	}
}

// FocusNext moves focus to the next focusable item (test helper)
func (fm *FocusManager) FocusNext() {
	if !fm.enabled || len(fm.focusableItems) == 0 {
		return
	}

	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableItems) {
		fm.focusableItems[fm.currentFocus].OnFocusLost()
	}

	startIndex := fm.currentFocus + 1
	for i := range fm.focusableItems {
		index := (startIndex + i) % len(fm.focusableItems)
		if fm.focusableItems[index].CanReceiveFocus() {
			fm.currentFocus = index
			fm.focusableItems[index].OnFocusGained()
			fm.focusIndicator.SetFocusTarget(fm.focusableItems[index].GetPlacement())
			fm.focusIndicator.Show()

			return
		}
	}

	fm.currentFocus = -1
	fm.focusIndicator.Hide()
}

// FocusPrevious moves focus to the previous focusable item (test helper)
func (fm *FocusManager) FocusPrevious() {
	if !fm.enabled || len(fm.focusableItems) == 0 {
		return
	}

	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableItems) {
		fm.focusableItems[fm.currentFocus].OnFocusLost()
	}

	startIndex := fm.currentFocus - 1
	if startIndex < 0 {
		startIndex = len(fm.focusableItems) - 1
	}

	for i := range fm.focusableItems {
		index := startIndex - i
		if index < 0 {
			index += len(fm.focusableItems)
		}

		if fm.focusableItems[index].CanReceiveFocus() {
			fm.currentFocus = index
			fm.focusableItems[index].OnFocusGained()
			fm.focusIndicator.SetFocusTarget(fm.focusableItems[index].GetPlacement())
			fm.focusIndicator.Show()

			return
		}
	}

	fm.currentFocus = -1
	fm.focusIndicator.Hide()
}

// GetCurrentFocus returns the currently focused item, or nil if none (test helper)
func (fm *FocusManager) GetCurrentFocus() FocusableItem {
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableItems) {
		return fm.focusableItems[fm.currentFocus]
	}

	return nil
}

// ClearFocus removes focus from all items
func (fm *FocusManager) ClearFocus() {
	if fm.currentFocus >= 0 && fm.currentFocus < len(fm.focusableItems) {
		fm.focusableItems[fm.currentFocus].OnFocusLost()
	}
	fm.currentFocus = -1
	fm.focusIndicator.Hide()
}

// SetEnabled enables or disables focus management
func (fm *FocusManager) SetEnabled(enabled bool) {
	fm.enabled = enabled
	if !enabled {
		fm.ClearFocus()
	}
}

// IsEnabled returns whether focus management is enabled
func (fm *FocusManager) IsEnabled() bool {
	return fm.enabled
}

// Update updates the focus indicator animation
func (fm *FocusManager) Update(elapsedMs uint64) {
	fm.focusIndicator.Update(elapsedMs)
}

// Draw renders the focus indicator
func (fm *FocusManager) Draw(screen *ebiten.Image, dm *display.DisplayManager) {
	fm.focusIndicator.Draw(screen, dm)
}

// Global focus manager for convenience
var globalFocusManager *FocusManager

// GetGlobalFocusManager returns the global focus manager instance
func GetGlobalFocusManager() *FocusManager {
	if globalFocusManager == nil {
		globalFocusManager = NewFocusManager(nil)
	}

	return globalFocusManager
}

// SetGlobalFocusManager sets the global focus manager instance
func SetGlobalFocusManager(manager *FocusManager) {
	globalFocusManager = manager
}
