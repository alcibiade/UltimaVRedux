// Package sprites provides tile overlay effect system with fluent builder API.
// Overlays are drawn ON TOP of tiles with special blend modes (additive for glows, etc.).
package sprites

import (
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ============================================================================
// OVERLAY ALGORITHM TYPES
// ============================================================================

// OverlayAlgorithm defines the overlay effect algorithm to apply.
type OverlayAlgorithm int

const (
	// OverlayPulsingGlow creates a pulsing glow effect using sine wave oscillation.
	// The glow pulses between PulseLow and PulseHigh alpha values over PeriodMs.
	// NOTE: This combines glow and pulsing - for separated control, use
	// OverlayStaticGlow + OverlayPulsingAlpha instead.
	OverlayPulsingGlow OverlayAlgorithm = iota

	// OverlayStaticGlow creates a constant glow effect without pulsing.
	// Uses additive blending (BlendLighter) to add light/color on top of sprite.
	// The Alpha field controls glow intensity.
	OverlayStaticGlow

	// OverlayPulsingAlpha modulates the base sprite's alpha over time.
	// This does NOT draw anything - instead use GetPulsingAlpha() to retrieve
	// the current alpha value and apply it to the base sprite before drawing.
	// Oscillates between PulseLow and PulseHigh over PeriodMs using sine wave.
	OverlayPulsingAlpha
)

// ============================================================================
// OVERLAY EFFECT
// ============================================================================

// OverlayEffect represents a single overlay effect in the pipeline.
type OverlayEffect struct {
	Algorithm OverlayAlgorithm
	Color     color.RGBA   // Base glow color
	Alpha     float32      // Static alpha for OverlayStaticGlow (0.0-1.0)
	PeriodMs  int64        // Full sine wave cycle duration (for pulsing effects)
	PulseLow  float32      // Minimum alpha (0.0-1.0) for pulsing effects
	PulseHigh float32      // Maximum alpha (0.0-1.0) for pulsing effects
	BlendMode ebiten.Blend // Blend mode (typically BlendLighter for additive glow)
}

// ============================================================================
// OVERLAY PIPELINE
// ============================================================================

// OverlayPipeline defines the complete overlay effect configuration.
// Effects are applied in order, each drawing on top of the previous.
type OverlayPipeline struct {
	Effects []OverlayEffect
}

// IsEmpty returns true if this pipeline has no effects configured.
func (p OverlayPipeline) IsEmpty() bool {
	return len(p.Effects) == 0
}

// ApplyOverlay draws all overlay effects in order on top of the base sprite.
// Uses a solid white glow texture - for masked overlays, use ApplyOverlayWithMask.
//
// Parameters:
//   - screen: The target image to draw overlays onto
//   - baseOpts: Draw options from the base sprite (provides GeoM transform)
//   - elapsedMs: Current game time for animation calculations
//   - opacity: External opacity multiplier (0.0-1.0) applied to all overlays
func (p OverlayPipeline) ApplyOverlay(
	screen *ebiten.Image,
	baseOpts *ebiten.DrawImageOptions,
	elapsedMs int64,
	opacity float32,
) {
	p.ApplyOverlayWithMask(screen, baseOpts, elapsedMs, opacity, nil)
}

// ApplyOverlayWithMask draws all overlay effects using a custom mask image.
// The mask defines where the glow appears - only pixels with alpha > 0 in the mask
// will show the glow effect. Use a white silhouette for pure glow colors.
//
// Parameters:
//   - screen: The target image to draw overlays onto
//   - baseOpts: Draw options from the base sprite (provides GeoM transform)
//   - elapsedMs: Current game time for animation calculations
//   - opacity: External opacity multiplier (0.0-1.0) applied to all overlays
//   - mask: Custom mask image (nil uses default solid glow texture)
func (p OverlayPipeline) ApplyOverlayWithMask(
	screen *ebiten.Image,
	baseOpts *ebiten.DrawImageOptions,
	elapsedMs int64,
	opacity float32,
	mask *ebiten.Image,
) {
	if p.IsEmpty() || opacity <= 0 {
		return
	}

	// Use provided mask or fall back to solid glow texture
	glowTexture := mask
	if glowTexture == nil {
		glowTexture = getGlowTexture()
	}

	for _, effect := range p.Effects {
		effect.apply(screen, baseOpts, glowTexture, elapsedMs, opacity)
	}
}

// ApplyOverlayAtPosition draws all overlay effects with position-based phase offset.
// Uses system clock for timing since these are purely cosmetic visual effects.
// The position-based offset ensures multiple entities don't pulse in sync.
//
// Parameters:
//   - screen: The target image to draw overlays onto
//   - baseOpts: Draw options from the base sprite (provides GeoM transform)
//   - posX, posY: World coordinates used to calculate phase offset
//   - opacity: External opacity multiplier (0.0-1.0) applied to all overlays
//   - mask: Custom mask image (nil uses default solid glow texture)
func (p OverlayPipeline) ApplyOverlayAtPosition(
	screen *ebiten.Image,
	baseOpts *ebiten.DrawImageOptions,
	posX, posY int,
	opacity float32,
	mask *ebiten.Image,
) {
	if p.IsEmpty() || opacity <= 0 {
		return
	}

	elapsedMs := time.Now().UnixMilli() // ALLOWED: purely cosmetic visual effects - no gameplay impact

	// Calculate position-based phase offset so entities don't pulse in sync
	// Uses prime multiplier (17) for good distribution
	phaseOffset := int64(posX*256+posY) * 17
	adjustedTime := elapsedMs + phaseOffset

	// Use provided mask or fall back to solid glow texture
	glowTexture := mask
	if glowTexture == nil {
		glowTexture = getGlowTexture()
	}

	for _, effect := range p.Effects {
		effect.apply(screen, baseOpts, glowTexture, adjustedTime, opacity)
	}
}

// apply draws a single overlay effect.
func (e OverlayEffect) apply(
	screen *ebiten.Image,
	baseOpts *ebiten.DrawImageOptions,
	glowTexture *ebiten.Image,
	elapsedMs int64,
	opacity float32,
) {
	switch e.Algorithm {
	case OverlayPulsingGlow:
		e.applyPulsingGlow(screen, baseOpts, glowTexture, elapsedMs, opacity)
	case OverlayStaticGlow:
		e.applyStaticGlow(screen, baseOpts, glowTexture, opacity)
	case OverlayPulsingAlpha:
		// OverlayPulsingAlpha doesn't draw anything - it's used via GetPulsingAlpha()
		// to modulate the base sprite's alpha before drawing.
	}
}

// applyPulsingGlow implements the pulsing glow overlay effect.
func (e OverlayEffect) applyPulsingGlow(
	screen *ebiten.Image,
	baseOpts *ebiten.DrawImageOptions,
	glowTexture *ebiten.Image,
	elapsedMs int64,
	opacity float32,
) {
	// Calculate current alpha from sine wave
	// Oscillates between PulseLow and PulseHigh over PeriodMs
	periodMs := e.PeriodMs
	if periodMs <= 0 {
		periodMs = 1500 // Default 1.5 second cycle
	}

	phase := float64(elapsedMs%periodMs) / float64(periodMs) * 2 * math.Pi
	alphaRange := e.PulseHigh - e.PulseLow
	center := (e.PulseHigh + e.PulseLow) / 2
	currentAlpha := center + (alphaRange/2)*float32(math.Sin(phase))

	// Apply external opacity multiplier
	finalAlpha := currentAlpha * opacity

	// Create overlay draw options, copying transform from base sprite
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM = baseOpts.GeoM

	// Set blend mode (additive for glow effects)
	opts.Blend = e.BlendMode

	// Apply color with calculated alpha
	// Normalize RGB to 0.0-1.0 range and apply as color scale
	r := float32(e.Color.R) / 255.0
	g := float32(e.Color.G) / 255.0
	b := float32(e.Color.B) / 255.0

	opts.ColorScale.Scale(r, g, b, finalAlpha)

	// Draw the glow texture
	screen.DrawImage(glowTexture, opts)
}

// applyStaticGlow implements a constant glow overlay effect (no pulsing).
func (e OverlayEffect) applyStaticGlow(
	screen *ebiten.Image,
	baseOpts *ebiten.DrawImageOptions,
	glowTexture *ebiten.Image,
	opacity float32,
) {
	// Apply external opacity multiplier to static alpha
	finalAlpha := e.Alpha * opacity

	// Create overlay draw options, copying transform from base sprite
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM = baseOpts.GeoM

	// Set blend mode (additive for glow effects)
	opts.Blend = e.BlendMode

	// Apply color with static alpha
	// Normalize RGB to 0.0-1.0 range and apply as color scale
	r := float32(e.Color.R) / 255.0
	g := float32(e.Color.G) / 255.0
	b := float32(e.Color.B) / 255.0

	opts.ColorScale.Scale(r, g, b, finalAlpha)

	// Draw the glow texture
	screen.DrawImage(glowTexture, opts)
}

// calculatePulsingAlpha calculates the current alpha value for pulsing effects.
func (e OverlayEffect) calculatePulsingAlpha(elapsedMs int64) float32 {
	periodMs := e.PeriodMs
	if periodMs <= 0 {
		periodMs = 1500 // Default 1.5 second cycle
	}

	phase := float64(elapsedMs%periodMs) / float64(periodMs) * 2 * math.Pi
	alphaRange := e.PulseHigh - e.PulseLow
	center := (e.PulseHigh + e.PulseLow) / 2

	return center + (alphaRange/2)*float32(math.Sin(phase))
}

// GetPulsingAlpha returns the combined pulsing alpha value from all OverlayPulsingAlpha effects.
// This value should be applied to the base sprite's ColorScale.ScaleAlpha() BEFORE drawing.
// Returns 1.0 if no pulsing alpha effects are configured.
//
// Example usage:
//
//	pulsingAlpha := overlay.GetPulsingAlpha(elapsedMs)
//	drawOpts.ColorScale.ScaleAlpha(pulsingAlpha)
//	screen.DrawImage(sprite, &drawOpts)
//	overlay.ApplyOverlay(screen, &drawOpts, elapsedMs, 1.0)  // Static glow on top
func (p OverlayPipeline) GetPulsingAlpha(elapsedMs int64) float32 {
	combined := float32(1.0)

	for _, effect := range p.Effects {
		if effect.Algorithm == OverlayPulsingAlpha {
			combined *= effect.calculatePulsingAlpha(elapsedMs)
		}
	}

	return combined
}

// GetPulsingAlphaAtPosition returns pulsing alpha with position-based phase offset.
// Uses system clock for timing since these are purely cosmetic visual effects.
// The position-based offset ensures multiple entities don't pulse in sync.
func (p OverlayPipeline) GetPulsingAlphaAtPosition(posX, posY int) float32 {
	elapsedMs := time.Now().UnixMilli() // ALLOWED: purely cosmetic visual effects - no gameplay impact

	// Calculate position-based phase offset so entities don't pulse in sync
	// Uses prime multiplier (17) for good distribution
	phaseOffset := int64(posX*256+posY) * 17
	adjustedTime := elapsedMs + phaseOffset

	return p.GetPulsingAlpha(adjustedTime)
}

// ============================================================================
// FLUENT BUILDER
// ============================================================================

// OverlayBuilder provides a fluent API for building overlay pipelines.
type OverlayBuilder struct {
	pipeline OverlayPipeline
}

// Overlay starts a new overlay pipeline definition.
func Overlay() *OverlayBuilder {
	return &OverlayBuilder{
		pipeline: OverlayPipeline{
			Effects: make([]OverlayEffect, 0),
		},
	}
}

// WithPulsingGlow adds a pulsing glow overlay effect.
// The glow oscillates between pulseLow and pulseHigh alpha values over periodMs.
// Uses additive blending (BlendLighter) by default.
func (b *OverlayBuilder) WithPulsingGlow(
	glowColor color.RGBA,
	periodMs int64,
	pulseLow, pulseHigh float32,
) *OverlayBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, OverlayEffect{
		Algorithm: OverlayPulsingGlow,
		Color:     glowColor,
		PeriodMs:  periodMs,
		PulseLow:  pulseLow,
		PulseHigh: pulseHigh,
		BlendMode: ebiten.BlendLighter, // Additive blending for glow
	})

	return b
}

// WithStaticGlow adds a constant glow overlay effect (no pulsing).
// Uses additive blending (BlendLighter) by default for a light-emitting effect.
//
// Parameters:
//   - glowColor: The color of the glow effect
//   - alpha: Glow intensity (0.0-1.0)
func (b *OverlayBuilder) WithStaticGlow(glowColor color.RGBA, alpha float32) *OverlayBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, OverlayEffect{
		Algorithm: OverlayStaticGlow,
		Color:     glowColor,
		Alpha:     alpha,
		BlendMode: ebiten.BlendLighter, // Additive blending for glow
	})

	return b
}

// WithPulsingAlpha adds an alpha modulation effect for the base sprite.
// This effect does NOT draw anything - instead use GetPulsingAlpha() to retrieve
// the current alpha value and apply it to the base sprite before drawing.
//
// Parameters:
//   - periodMs: Full sine wave cycle duration in milliseconds
//   - pulseLow: Minimum alpha (0.0-1.0)
//   - pulseHigh: Maximum alpha (0.0-1.0)
func (b *OverlayBuilder) WithPulsingAlpha(periodMs int64, pulseLow, pulseHigh float32) *OverlayBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, OverlayEffect{
		Algorithm: OverlayPulsingAlpha,
		PeriodMs:  periodMs,
		PulseLow:  pulseLow,
		PulseHigh: pulseHigh,
	})

	return b
}

// WithBlendMode sets the blend mode for the most recently added effect.
// Call this immediately after adding an effect to override the default blend mode.
func (b *OverlayBuilder) WithBlendMode(blend ebiten.Blend) *OverlayBuilder {
	if len(b.pipeline.Effects) > 0 {
		b.pipeline.Effects[len(b.pipeline.Effects)-1].BlendMode = blend
	}

	return b
}

// Build validates and returns the completed pipeline.
func (b *OverlayBuilder) Build() OverlayPipeline {
	return b.pipeline
}

// ============================================================================
// SHARED GLOW TEXTURE (LAZY INITIALIZATION)
// ============================================================================

// glowTexture is a shared solid white texture used for all glow overlays.
// White base allows ColorScale to produce any desired glow color.
var glowTexture *ebiten.Image

// getGlowTexture returns the shared glow texture, creating it if needed.
// The texture is a solid white TileSize x TileSize image.
func getGlowTexture() *ebiten.Image {
	if glowTexture == nil {
		glowTexture = ebiten.NewImage(TileSize, TileSize)
		glowTexture.Fill(color.White)
	}

	return glowTexture
}

// ============================================================================
// SPRITE MASK GENERATION AND CACHING
// ============================================================================

// spriteMaskCache caches generated sprite masks to avoid regeneration each frame.
// Keyed by sprite image pointer since sprites don't change at runtime.
var (
	spriteMaskCache   = make(map[*ebiten.Image]*ebiten.Image)
	spriteMaskCacheMu sync.RWMutex
)

// GetSpriteMask returns a white silhouette mask for the given sprite image.
// The mask has white pixels where the sprite has any non-transparent pixels,
// and transparent pixels where the sprite is transparent.
// Masks are cached for performance - sprites don't change at runtime.
//
// This is useful for overlay effects that should follow the sprite's shape
// rather than being a solid square.
func GetSpriteMask(sprite *ebiten.Image) *ebiten.Image {
	if sprite == nil {
		return nil
	}

	// Check cache first (read lock)
	spriteMaskCacheMu.RLock()
	if mask, ok := spriteMaskCache[sprite]; ok {
		spriteMaskCacheMu.RUnlock()

		return mask
	}
	spriteMaskCacheMu.RUnlock()

	// Generate mask (no lock during generation - it's safe to regenerate)
	mask := generateSpriteMask(sprite)

	// Store in cache (write lock)
	spriteMaskCacheMu.Lock()
	spriteMaskCache[sprite] = mask
	spriteMaskCacheMu.Unlock()

	return mask
}

// generateSpriteMask creates a white silhouette mask from a sprite.
// Any pixel with alpha > 0 becomes white (255,255,255,255).
// Any pixel with alpha == 0 stays transparent (0,0,0,0).
func generateSpriteMask(sprite *ebiten.Image) *ebiten.Image {
	bounds := sprite.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	mask := ebiten.NewImage(width, height)

	// Read all pixels from source sprite
	pixels := make([]byte, width*height*4)
	sprite.ReadPixels(pixels)

	// Create mask pixels - white where source has alpha, transparent otherwise
	maskPixels := make([]byte, width*height*4)
	for i := 0; i < len(pixels); i += 4 {
		alpha := pixels[i+3]
		if alpha > 0 {
			// Source pixel is visible - make mask pixel white
			maskPixels[i] = 255   // R
			maskPixels[i+1] = 255 // G
			maskPixels[i+2] = 255 // B
			maskPixels[i+3] = 255 // A
		}
		// else: leave as 0,0,0,0 (transparent)
	}

	mask.WritePixels(maskPixels)

	return mask
}

// ClearSpriteMaskCache clears the sprite mask cache.
// This is primarily for testing purposes.
func ClearSpriteMaskCache() {
	spriteMaskCacheMu.Lock()
	spriteMaskCache = make(map[*ebiten.Image]*ebiten.Image)
	spriteMaskCacheMu.Unlock()
}

// GetSpriteMaskCacheSize returns the number of cached sprite masks.
// This is primarily for testing and debugging.
func GetSpriteMaskCacheSize() int {
	spriteMaskCacheMu.RLock()
	defer spriteMaskCacheMu.RUnlock()

	return len(spriteMaskCache)
}
