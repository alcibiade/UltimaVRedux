// Package sprites provides tile visual effect system with fluent builder API.
package sprites

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ============================================================================
// OUTLINE COLOR TYPES
// ============================================================================

// OutlineColorType represents different outline color categories for visual effects.
type OutlineColorType int

const (
	// NoOutline indicates no outline should be rendered
	NoOutline OutlineColorType = iota
	// YellowOutline for the Avatar (player character)
	YellowOutline
	// WhiteOutline for friendly NPCs
	WhiteOutline
	// RedOutline for hostile enemies
	RedOutline
	// BlueOutline for vehicles and special objects
	BlueOutline
	// GreenOutline for quest-related NPCs or items
	GreenOutline
	// PurpleOutline for magical entities
	PurpleOutline
	// BlackOutline for enhanced visibility
	BlackOutline
	// CombatActivePCOutline identifies the party member currently taking a turn.
	CombatActivePCOutline
	// CombatInactivePCOutline identifies party members waiting for their turn.
	CombatInactivePCOutline
	// CombatActiveMonsterOutline identifies the monster currently taking a turn.
	CombatActiveMonsterOutline
	// CombatThreatenedPCOutline identifies a PC targeted by an incoming attack.
	CombatThreatenedPCOutline
)

// GetOutlineColor returns the actual color for a given outline type.
func (oct OutlineColorType) GetOutlineColor() color.RGBA {
	switch oct {
	case NoOutline:
		return ucolor.EGATransparent.ToRGBA()
	case YellowOutline:
		return ucolor.EGAYellow.ToRGBA()
	case WhiteOutline:
		return ucolor.EGAWhite.ToRGBA()
	case RedOutline:
		return ucolor.EGARed.ToRGBA()
	case BlueOutline:
		return ucolor.EGALightBlue.ToRGBA()
	case GreenOutline:
		return ucolor.EGALightGreen.ToRGBA()
	case PurpleOutline:
		return ucolor.EGALightMagenta.ToRGBA()
	case BlackOutline:
		return ucolor.EGABlack.ToRGBA()
	case CombatActivePCOutline:
		return ucolor.CombatActivePCOutline.ToRGBA()
	case CombatInactivePCOutline:
		return ucolor.CombatInactivePCOutline.ToRGBA()
	case CombatActiveMonsterOutline:
		return ucolor.CombatActiveMonsterOutline.ToRGBA()
	case CombatThreatenedPCOutline:
		return ucolor.CombatThreatenedPCOutline.ToRGBA()
	default:
		return ucolor.EGATransparent.ToRGBA()
	}
}

// ============================================================================
// EFFECT ALGORITHM TYPES
// ============================================================================

// EffectAlgorithm defines the visual effect algorithm to apply.
type EffectAlgorithm int

const (
	// EffectVerticalRise simulates flames rising upward with horizontal jitter.
	EffectVerticalRise EffectAlgorithm = iota

	// EffectWaveDisplace applies sine-wave horizontal displacement (for lava).
	EffectWaveDisplace

	// EffectVerticalScroll scrolls pixels vertically with wrap-around (for water).
	// Creates authentic Ultima V water animation where pixels scroll downward and wrap.
	EffectVerticalScroll

	// EffectTransparencyOverlay overlays a tile on an animated background.
	// Blue/water-colored pixels become transparent, revealing animated water beneath.
	// The background tile is specified via AddTransparencyOverlay(backgroundTile).
	EffectTransparencyOverlay

	// EffectOutline adds a colored outline around non-transparent pixels.
	// Uses 4-neighbor edge detection to find sprite boundaries.
	EffectOutline

	// EffectOpacity applies alpha transparency to the entire sprite.
	EffectOpacity

	// EffectTint applies a color overlay/tint to the sprite.
	EffectTint

	// EffectGlow applies a brightness modulation for glowing effects.
	EffectGlow

	// EffectPulse modulates selected pixels' opacity through a deterministic cycle.
	EffectPulse

	// EffectBrightnessPulse modulates selected pixels' brightness while preserving opacity.
	EffectBrightnessPulse

	// EffectBorderWrap wraps a 16x16 sprite in an 18x18 image with 1-pixel transparent border.
	// This allows outlines to completely wrap around sprites that have pixels at the edges.
	// Must be the first effect in the pipeline if used (changes dimensions).
	EffectBorderWrap
)

// PreferredFrameCount returns the optimal frame count for this algorithm.
// Water-based algorithms use 16 frames for smooth scrolling,
// while fire/displacement effects use 8 frames.
func (a EffectAlgorithm) PreferredFrameCount() int {
	switch a {
	case EffectVerticalScroll, EffectTransparencyOverlay:
		return 16 // Water uses 16 frames for 1-pixel-per-frame scrolling
	default:
		return 8 // Fire/displacement effects use 8 frames
	}
}

// ============================================================================
// COLOR FILTER TYPES
// ============================================================================

// ColorFilter defines which pixels the effect applies to.
type ColorFilter int

const (
	// FilterAll applies the effect to all non-transparent pixels.
	FilterAll ColorFilter = iota

	// FilterRedOnly applies the effect only to EGA red pixels (170, 0, 0).
	FilterRedOnly

	// FilterYellowOnly applies the effect only to EGA yellow pixels (255, 255, 85).
	FilterYellowOnly

	// FilterFire applies the effect to both EGA red colors used in fire sprites:
	// EGA Red (170, 0, 0) and EGA Light Red (255, 85, 85).
	FilterFire

	// Blue flame
	FilterBlueFire

	// FilterPoison applies to the green pixels used by poison fields.
	FilterPoison

	// FilterSleep applies to the magenta pixels used by sleep fields.
	FilterSleep

	// FilterProtection applies to the blue pixels used by protection fields.
	FilterProtection
)

// ============================================================================
// POSITION OFFSET MODE
// ============================================================================

// PositionOffsetMode defines how position affects frame selection.
type PositionOffsetMode int

const (
	// PositionOffsetDefault uses posHash % frameCount for position offset.
	// Different tiles of the same type animate out of sync based on their position hash.
	PositionOffsetDefault PositionOffsetMode = iota

	// PositionOffsetEvenX applies an additional offset for tiles at even X coordinates.
	// This creates the authentic Ultima V water wave pattern where adjacent columns
	// are offset by a fixed amount (typically +6 frames).
	PositionOffsetEvenX
)

// ============================================================================
// TILE EFFECT PIPELINE
// ============================================================================

// TileEffect represents a single effect in the pipeline.
// Effects can have per-effect parameters configured via functional options.
type TileEffect struct {
	Algorithm  EffectAlgorithm
	Filter     ColorFilter
	Background indexes.SpriteIndex // Background tile for TransparencyOverlay (-1 for none)

	// Per-effect parameters (configured via functional options)
	Bounds       image.Rectangle  // Region to apply effect (empty = full tile)
	IntervalMs   int              // Animation interval for this effect (0 = use pipeline default)
	OutlineColor OutlineColorType // For EffectOutline
	Intensity    float32          // For EffectOutline (0.0-1.0), EffectGlow
	Opacity      float64          // For EffectOpacity (0.0-1.0)
	TintColor    color.RGBA       // For EffectTint
}

// EffectOption configures an individual effect's parameters.
type EffectOption func(*TileEffect)

// WithEffectBounds sets the pixel region for this effect.
// Coordinates are inclusive (x1, y1, x2, y2).
func WithEffectBounds(x1, y1, x2, y2 int) EffectOption {
	return func(e *TileEffect) {
		e.Bounds = image.Rect(x1, y1, x2+1, y2+1)
	}
}

// WithEffectInterval sets the animation interval for this effect in milliseconds.
func WithEffectInterval(ms int) EffectOption {
	return func(e *TileEffect) {
		e.IntervalMs = ms
	}
}

// WithIntensity sets the intensity for outline or glow effects (0.0-1.0).
func WithIntensity(intensity float32) EffectOption {
	return func(e *TileEffect) {
		e.Intensity = intensity
	}
}

// TileEffectPipeline defines the complete visual effect configuration for a tile.
// Effects are applied in order, with each effect operating on the output of the previous.
type TileEffectPipeline struct {
	Background     indexes.SpriteIndex // Sprite to composite behind (-1 for none)
	Bounds         image.Rectangle     // Region to apply effects (empty = full tile) - DEPRECATED: use per-effect bounds
	Effects        []TileEffect        // Effects to apply in order
	IntervalMs     int                 // Frame change interval (0 = use default)
	OffsetMode     PositionOffsetMode  // How position affects frame selection
	EvenXOffset    int                 // Offset applied for even X coordinates (only used with PositionOffsetEvenX)
	FrameCount     int                 // Number of frames (0 = auto-detect from algorithms)
	BaseBackground indexes.SpriteIndex // Background for water scroll effects (e.g., WaterCoast for mixed tiles)
}

// IsEmpty returns true if this pipeline has no effects configured.
func (p TileEffectPipeline) IsEmpty() bool {
	return len(p.Effects) == 0
}

// ============================================================================
// FLUENT BUILDER
// ============================================================================

const (
	// defaultEffectIntervalMs is the default frame change interval for effects.
	defaultEffectIntervalMs = 150

	// defaultFireFrameCount is the number of pre-generated frame variants for fire effects.
	defaultFireFrameCount = 8

	// pulsePhaseDivisor maps eight frames to a complete sine-wave cycle.
	pulsePhaseDivisor = 4

	// pulseWaveRange normalizes a sine wave from [-1, 1] to [0, 1].
	pulseWaveRange = 2

	// pulseMinimumOpacity and pulseOpacityAmplitude make magical fields visibly billow.
	pulseMinimumOpacity   = 0.15
	pulseOpacityAmplitude = 0.80

	// brightnessPulseMinimum and brightnessPulseAmplitude keep barriers visibly energized without flashing.
	brightnessPulseMinimum   = 0.70
	brightnessPulseAmplitude = 0.30

	// defaultWaterEvenXOffset is the frame offset for even X coordinates in water animation.
	defaultWaterEvenXOffset = 6
)

// TileEffectBuilder provides a fluent API for defining tile effects.
type TileEffectBuilder struct {
	pipeline TileEffectPipeline
}

// Effect starts a new tile effect definition.
func Effect() *TileEffectBuilder {
	return &TileEffectBuilder{
		pipeline: TileEffectPipeline{
			Background:     -1,
			BaseBackground: -1,
			Effects:        make([]TileEffect, 0),
			OffsetMode:     PositionOffsetDefault,
		},
	}
}

// WithBackground sets the sprite to composite behind this tile.
func (b *TileEffectBuilder) WithBackground(sprite indexes.SpriteIndex) *TileEffectBuilder {
	b.pipeline.Background = sprite

	return b
}

// WithBounds sets the pixel region to apply effects.
// All coordinates are inclusive (x1, y1, x2, y2).
// For example, WithBounds(6, 1, 12, 4) affects pixels from (6,1) to (12,4) inclusive.
func (b *TileEffectBuilder) WithBounds(x1, y1, x2, y2 int) *TileEffectBuilder {
	// Convert inclusive coordinates to Go's exclusive Rectangle convention
	b.pipeline.Bounds = image.Rect(x1, y1, x2+1, y2+1)

	return b
}

// WithInterval sets the frame change interval in milliseconds.
// If not set, defaults to 150ms.
func (b *TileEffectBuilder) WithInterval(ms int) *TileEffectBuilder {
	b.pipeline.IntervalMs = ms

	return b
}

// WithEvenXOffset enables position-based offset for water wave patterns.
// Tiles at even X coordinates will use an additional frame offset (default: 6).
// This creates the authentic Ultima V water animation pattern.
func (b *TileEffectBuilder) WithEvenXOffset(offset int) *TileEffectBuilder {
	b.pipeline.OffsetMode = PositionOffsetEvenX
	b.pipeline.EvenXOffset = offset

	return b
}

// AddVerticalRise adds the vertical rise effect (flames rising with horizontal jitter).
// Optional EffectOption parameters: WithEffectBounds, WithEffectInterval.
func (b *TileEffectBuilder) AddVerticalRise(filter ColorFilter, opts ...EffectOption) *TileEffectBuilder {
	effect := TileEffect{
		Algorithm:  EffectVerticalRise,
		Filter:     filter,
		Background: -1,
		Intensity:  1.0, // default intensity
	}
	for _, opt := range opts {
		opt(&effect)
	}
	b.pipeline.Effects = append(b.pipeline.Effects, effect)

	return b
}

// AddWaveDisplace adds the wave displacement effect (sine-wave horizontal movement).
// Optional EffectOption parameters: WithEffectBounds, WithEffectInterval.
func (b *TileEffectBuilder) AddWaveDisplace(filter ColorFilter, opts ...EffectOption) *TileEffectBuilder {
	effect := TileEffect{
		Algorithm:  EffectWaveDisplace,
		Filter:     filter,
		Background: -1,
		Intensity:  1.0,
	}
	for _, opt := range opts {
		opt(&effect)
	}
	b.pipeline.Effects = append(b.pipeline.Effects, effect)

	return b
}

// AddVerticalScroll adds vertical pixel scrolling with wrap-around (for water).
// Creates authentic Ultima V water animation where pixels scroll downward and wrap.
func (b *TileEffectBuilder) AddVerticalScroll() *TileEffectBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, TileEffect{
		Algorithm:  EffectVerticalScroll,
		Filter:     FilterAll,
		Background: -1,
	})

	return b
}

// AddTransparencyOverlay adds transparency-based overlay animation.
// Blue/water-colored pixels become transparent, revealing the animated background beneath.
// The background parameter specifies which tile to use as the animated background
// (typically WaterCoast for water stream tiles).
func (b *TileEffectBuilder) AddTransparencyOverlay(background indexes.SpriteIndex) *TileEffectBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, TileEffect{
		Algorithm:  EffectTransparencyOverlay,
		Filter:     FilterAll,
		Background: background,
	})

	return b
}

// AddOutline adds an outline effect around non-transparent pixels.
// Uses 4-neighbor edge detection to find sprite boundaries.
// Optional EffectOption: WithIntensity (default 1.0).
func (b *TileEffectBuilder) AddOutline(outlineColor OutlineColorType, opts ...EffectOption) *TileEffectBuilder {
	effect := TileEffect{
		Algorithm:    EffectOutline,
		Filter:       FilterAll,
		Background:   -1,
		OutlineColor: outlineColor,
		Intensity:    1.0, // default full intensity
	}
	for _, opt := range opts {
		opt(&effect)
	}
	b.pipeline.Effects = append(b.pipeline.Effects, effect)

	return b
}

// AddOpacity adds an opacity/transparency effect.
// opacity should be 0.0 (fully transparent) to 1.0 (fully opaque).
func (b *TileEffectBuilder) AddOpacity(opacity float64) *TileEffectBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, TileEffect{
		Algorithm:  EffectOpacity,
		Filter:     FilterAll,
		Background: -1,
		Opacity:    opacity,
	})

	return b
}

// AddTint adds a color tint overlay effect.
func (b *TileEffectBuilder) AddTint(tintColor color.RGBA) *TileEffectBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, TileEffect{
		Algorithm:  EffectTint,
		Filter:     FilterAll,
		Background: -1,
		TintColor:  tintColor,
	})

	return b
}

// AddGlow adds a brightness/glow effect.
// Optional EffectOption: WithIntensity (default 0.8).
func (b *TileEffectBuilder) AddGlow(opts ...EffectOption) *TileEffectBuilder {
	effect := TileEffect{
		Algorithm:  EffectGlow,
		Filter:     FilterAll,
		Background: -1,
		Intensity:  0.8, // default glow intensity
	}
	for _, opt := range opts {
		opt(&effect)
	}
	b.pipeline.Effects = append(b.pipeline.Effects, effect)

	return b
}

// AddPulse adds a deterministic opacity pulse to pixels selected by filter.
func (b *TileEffectBuilder) AddPulse(filter ColorFilter, opts ...EffectOption) *TileEffectBuilder {
	//nolint:exhaustruct // Remaining fields are optional for a pulse effect.
	effect := TileEffect{
		Algorithm:  EffectPulse,
		Filter:     filter,
		Background: -1,
	}
	for _, opt := range opts {
		opt(&effect)
	}
	b.pipeline.Effects = append(b.pipeline.Effects, effect)

	return b
}

// AddBrightnessPulse adds a deterministic brightness pulse without changing pixel opacity.
func (b *TileEffectBuilder) AddBrightnessPulse(filter ColorFilter, opts ...EffectOption) *TileEffectBuilder {
	//nolint:exhaustruct // Remaining fields are optional for a brightness pulse effect.
	effect := TileEffect{
		Algorithm:  EffectBrightnessPulse,
		Filter:     filter,
		Background: -1,
	}
	for _, opt := range opts {
		opt(&effect)
	}
	b.pipeline.Effects = append(b.pipeline.Effects, effect)

	return b
}

// AddBorder wraps a 16x16 sprite in an 18x18 image with 1-pixel transparent border.
// This allows outlines to completely wrap around sprites that have pixels at the edges.
// Should typically be the first effect in the pipeline (changes dimensions from 16x16 to 18x18).
func (b *TileEffectBuilder) AddBorder() *TileEffectBuilder {
	b.pipeline.Effects = append(b.pipeline.Effects, TileEffect{
		Algorithm:  EffectBorderWrap,
		Filter:     FilterAll,
		Background: -1,
	})

	return b
}

// Build validates and returns the completed pipeline.
// Panics if the pipeline configuration is invalid.
func (b *TileEffectBuilder) Build() TileEffectPipeline {
	b.validate()

	return b.pipeline
}

// validate panics if the pipeline configuration is invalid.
func (b *TileEffectBuilder) validate() {
	if len(b.pipeline.Effects) == 0 {
		panic("TileEffectPipeline: must have at least one effect (use AddVerticalRise, AddWaveDisplace, etc.)")
	}

	bounds := b.pipeline.Bounds
	if !bounds.Empty() {
		if bounds.Min.X < 0 || bounds.Min.Y < 0 {
			panic(fmt.Sprintf("TileEffectPipeline: bounds top-left cannot be negative: %v", bounds.Min))
		}
		if bounds.Max.X > TileSize || bounds.Max.Y > TileSize {
			panic(fmt.Sprintf("TileEffectPipeline: bounds exceed tile size (%d): max=%v", TileSize, bounds.Max))
		}
	}
}

// ============================================================================
// TILE EFFECT PROCESSOR
// ============================================================================

// TileEffectProcessor generates and caches pre-computed effect frames for tiles.
// Also supports dynamic effects (outline, tint, etc.) with LRU caching.
type TileEffectProcessor struct {
	spriteSheet *SpriteSheet

	// Pre-generated effect frames for each registered tile (16x16).
	// Map from SpriteIndex to slice of frame variants (variable length per tile).
	effectFrames map[indexes.SpriteIndex][]*ebiten.Image

	// Pre-generated 18x18 bordered frames for direct use (skips cache entirely).
	// For tiles with PositionOffsetEvenX, this contains [0..frameCount-1] for odd X,
	// and [frameCount..2*frameCount-1] for even X.
	borderedFrames map[indexes.SpriteIndex][]*ebiten.Image

	// Frame count per tile (needed for modulo operations)
	frameCountPerTile map[indexes.SpriteIndex]int

	// Dynamic effect cache for on-demand effects (outline, tint, glow, opacity)
	dynamicCache *EffectCache

	initialized bool
}

// NewTileEffectProcessor creates a new tile effect processor.
// Note: Initialization is deferred until first use (lazy init) because
// ebiten.Image.ReadPixels() cannot be called before ebiten.RunGame() starts.
func NewTileEffectProcessor(spriteSheet *SpriteSheet) *TileEffectProcessor {
	return &TileEffectProcessor{
		spriteSheet:       spriteSheet,
		effectFrames:      make(map[indexes.SpriteIndex][]*ebiten.Image),
		borderedFrames:    make(map[indexes.SpriteIndex][]*ebiten.Image),
		frameCountPerTile: make(map[indexes.SpriteIndex]int),
		dynamicCache:      NewEffectCache(DefaultDynamicCacheSize),
		initialized:       false,
	}
}

// Initialize pre-generates all effect frames for registered tiles.
// Must be called after ebiten.RunGame() starts (ReadPixels requirement).
// Processes tiles in dependency order: pure water/lava tiles first (no dependencies),
// then mixed tiles that depend on water backgrounds.
// Also pre-generates 18x18 bordered versions for direct use (skips cache overhead).
func (p *TileEffectProcessor) Initialize() {
	if p.initialized {
		return
	}

	// Phase 1: Generate frames for tiles WITHOUT transparency overlay dependencies
	// These are pure water, lava, and flame tiles that don't depend on other tiles
	for spriteIndex, pipeline := range TileEffectRegistry {
		if !pipelineHasTransparencyOverlay(pipeline) {
			frameCount := p.getFrameCountForPipeline(pipeline)
			p.frameCountPerTile[spriteIndex] = frameCount
			p.effectFrames[spriteIndex] = p.generateFrames(spriteIndex, pipeline, frameCount)
		}
	}

	// Phase 2: Generate frames for tiles WITH transparency overlay dependencies
	// These depend on the background tiles (typically WaterCoast) being already generated
	for spriteIndex, pipeline := range TileEffectRegistry {
		if pipelineHasTransparencyOverlay(pipeline) {
			frameCount := p.getFrameCountForPipeline(pipeline)
			p.frameCountPerTile[spriteIndex] = frameCount
			p.effectFrames[spriteIndex] = p.generateFrames(spriteIndex, pipeline, frameCount)
		}
	}

	// Phase 3: Pre-generate 18x18 bordered versions for all tiles
	// This eliminates per-frame WrapSpriteWith18x18Border() calls
	for spriteIndex, pipeline := range TileEffectRegistry {
		frames := p.effectFrames[spriteIndex]
		frameCount := p.frameCountPerTile[spriteIndex]

		if pipeline.OffsetMode == PositionOffsetEvenX {
			// For even/odd X offset tiles, we need 2x frames:
			// [0..frameCount-1] = odd X positions (base frames)
			// [frameCount..2*frameCount-1] = even X positions (offset frames)
			borderedFrames := make([]*ebiten.Image, frameCount*2)

			offset := pipeline.EvenXOffset
			if offset == 0 {
				offset = defaultWaterEvenXOffset
			}

			for i := range frameCount {
				// Odd X: direct frame mapping
				borderedFrames[i] = WrapSpriteWith18x18Border(frames[i])

				// Even X: apply offset
				evenFrameIndex := (i + offset) % frameCount
				borderedFrames[frameCount+i] = WrapSpriteWith18x18Border(frames[evenFrameIndex])
			}
			p.borderedFrames[spriteIndex] = borderedFrames
		} else {
			// For default offset mode, just wrap each frame
			borderedFrames := make([]*ebiten.Image, len(frames))
			for i, frame := range frames {
				borderedFrames[i] = WrapSpriteWith18x18Border(frame)
			}
			p.borderedFrames[spriteIndex] = borderedFrames
		}
	}

	p.initialized = true
}

// pipelineHasTransparencyOverlay returns true if the pipeline contains a TransparencyOverlay effect.
func pipelineHasTransparencyOverlay(pipeline TileEffectPipeline) bool {
	for _, effect := range pipeline.Effects {
		if effect.Algorithm == EffectTransparencyOverlay {
			return true
		}
	}

	return false
}

// safeReadPixels wraps ebiten.Image.ReadPixels with panic recovery.
// Returns an error if ReadPixels panics (e.g., before game starts in tests).
func (p *TileEffectProcessor) safeReadPixels(img *ebiten.Image, pixels []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ReadPixels failed: %v", r)
		}
	}()
	img.ReadPixels(pixels)

	return nil
}

// getFrameCountForPipeline determines the appropriate frame count based on the pipeline's effects.
// Uses the maximum preferred frame count among all effects in the pipeline.
func (p *TileEffectProcessor) getFrameCountForPipeline(pipeline TileEffectPipeline) int {
	// If explicitly set, use that
	if pipeline.FrameCount > 0 {
		return pipeline.FrameCount
	}

	// Find maximum preferred frame count among all effects
	maxFrames := defaultFireFrameCount
	for _, effect := range pipeline.Effects {
		preferred := effect.Algorithm.PreferredFrameCount()
		if preferred > maxFrames {
			maxFrames = preferred
		}
	}

	return maxFrames
}

// GetEffectSprite returns the appropriate effect frame for a tile.
// Returns nil if the sprite has no registered effects.
func (p *TileEffectProcessor) GetEffectSprite(
	spriteIndex indexes.SpriteIndex,
	posHash int32,
	elapsedMs int64,
) *ebiten.Image {
	if !p.initialized {
		p.Initialize()
	}

	frames, exists := p.effectFrames[spriteIndex]
	if !exists {
		return nil
	}

	frameCount := p.frameCountPerTile[spriteIndex]
	if frameCount == 0 || len(frames) == 0 {
		return nil
	}

	pipeline := TileEffectRegistry[spriteIndex]
	interval := pipeline.IntervalMs
	if interval <= 0 {
		interval = defaultEffectIntervalMs
	}

	// Calculate frame index based on time
	timeFrame := positiveModulo(elapsedMs/int64(interval), frameCount)

	// Apply position offset based on mode
	var frameIndex int
	switch pipeline.OffsetMode {
	case PositionOffsetEvenX:
		// Extract X coordinate from posHash (Formula: X*256 + Y)
		x := int(posHash / 256)
		if x%2 == 0 {
			// Even X coordinate - add offset
			offset := pipeline.EvenXOffset
			if offset == 0 {
				offset = defaultWaterEvenXOffset
			}
			frameIndex = positiveModulo(int64(timeFrame+offset), frameCount)
		} else {
			// Odd X coordinate - use base frame
			frameIndex = timeFrame
		}
	default:
		// Default: use posHash for position variation
		posOffset := positiveModulo(int64(posHash), frameCount)
		frameIndex = positiveModulo(int64(timeFrame+posOffset), frameCount)
	}

	return frames[frameIndex]
}

// HasEffect returns true if this sprite has registered visual effects.
// This can be called before initialization - it checks the registry directly.
func (p *TileEffectProcessor) HasEffect(spriteIndex indexes.SpriteIndex) bool {
	_, exists := TileEffectRegistry[spriteIndex]

	return exists
}

// GetBorderedEffectSprite returns the pre-generated 18x18 bordered frame directly.
// This bypasses the cache entirely since all frames are pre-generated at initialization.
// Returns nil if the sprite has no registered effects.
func (p *TileEffectProcessor) GetBorderedEffectSprite(
	spriteIndex indexes.SpriteIndex,
	posHash int32,
	elapsedMs int64,
) *ebiten.Image {
	if !p.initialized {
		p.Initialize()
	}

	frames, exists := p.borderedFrames[spriteIndex]
	if !exists || len(frames) == 0 {
		return nil
	}

	pipeline := TileEffectRegistry[spriteIndex]
	frameCount := p.frameCountPerTile[spriteIndex]
	if frameCount == 0 {
		return nil
	}

	interval := pipeline.IntervalMs
	if interval <= 0 {
		interval = defaultEffectIntervalMs
	}

	// Calculate time-based frame
	timeFrame := positiveModulo(elapsedMs/int64(interval), frameCount)

	// For PositionOffsetEvenX tiles, bordered frames are organized as:
	// [0..frameCount-1] = odd X positions
	// [frameCount..2*frameCount-1] = even X positions
	switch pipeline.OffsetMode {
	case PositionOffsetEvenX:
		x := int(posHash / 256)
		if x%2 == 0 {
			// Even X: use second half of frames array
			return frames[frameCount+timeFrame]
		}
		// Odd X: use first half
		return frames[timeFrame]
	default:
		// Default mode: use posHash for variation
		posOffset := positiveModulo(int64(posHash), frameCount)
		frameIndex := positiveModulo(int64(timeFrame+posOffset), frameCount)

		return frames[frameIndex]
	}
}

// GetFrameIndex calculates the frame index for a sprite based on position and time.
// This is used for cache key alignment - the cache key should use the frame index
// rather than raw time to avoid misalignment between cache quantization and frame boundaries.
// Returns -1 if the sprite has no registered effects.
func (p *TileEffectProcessor) GetFrameIndex(
	spriteIndex indexes.SpriteIndex,
	posHash int32,
	elapsedMs int64,
) int {
	pipeline, exists := TileEffectRegistry[spriteIndex]
	if !exists {
		return -1
	}

	frameCount := p.getFrameCountForPipeline(pipeline)
	if frameCount == 0 {
		return 0
	}

	interval := pipeline.IntervalMs
	if interval <= 0 {
		interval = defaultEffectIntervalMs
	}

	// Calculate frame index based on time
	timeFrame := positiveModulo(elapsedMs/int64(interval), frameCount)

	// Apply position offset based on mode
	switch pipeline.OffsetMode {
	case PositionOffsetEvenX:
		// Extract X coordinate from posHash (Formula: X*256 + Y)
		x := int(posHash / 256)
		if x%2 == 0 {
			// Even X coordinate - add offset
			offset := pipeline.EvenXOffset
			if offset == 0 {
				offset = defaultWaterEvenXOffset
			}

			return positiveModulo(int64(timeFrame+offset), frameCount)
		}
		// Odd X coordinate - use base frame
		return timeFrame
	default:
		// Default: use posHash for position variation
		posOffset := positiveModulo(int64(posHash), frameCount)

		return positiveModulo(int64(timeFrame+posOffset), frameCount)
	}
}

// ============================================================================
// DYNAMIC EFFECT APPLICATION
// ============================================================================

// ApplyDynamic applies effects from a pipeline to an arbitrary sprite.
// Uses LRU cache to avoid regenerating identical effect combinations.
// This is used for dynamic effects like entity outlines, tints, and glows.
//
// Parameters:
//   - sprite: The input sprite image to apply effects to
//   - pipeline: The effect pipeline to apply (built with Effect().AddOutline(...).Build())
//
// Returns the processed sprite image. If sprite is nil, returns nil.
func (p *TileEffectProcessor) ApplyDynamic(
	sprite *ebiten.Image,
	pipeline TileEffectPipeline,
) *ebiten.Image {
	if sprite == nil {
		return nil
	}

	// Calculate cache key from sprite pointer and pipeline hash
	key := p.calculateDynamicCacheKey(sprite, pipeline)

	// Check cache
	if cached := p.dynamicCache.Get(key); cached != nil {
		return cached
	}

	// Generate the effect
	result := p.applyPipelineToSprite(sprite, pipeline)

	// Cache result
	p.dynamicCache.Put(key, result)

	return result
}

// ApplyDynamicSimple is a convenience method for applying a single outline effect.
// Equivalent to ApplyDynamic with Effect().AddOutline(outlineColor).Build().
func (p *TileEffectProcessor) ApplyDynamicSimple(
	sprite *ebiten.Image,
	outlineColor OutlineColorType,
) *ebiten.Image {
	if outlineColor == NoOutline {
		return sprite
	}
	pipeline := Effect().AddOutline(outlineColor).Build()

	return p.ApplyDynamic(sprite, pipeline)
}

// ApplyDynamicSimpleWithIntensity applies a cached outline at a caller-provided
// intensity. Callers should quantize animated values to keep cache cardinality bounded.
func (p *TileEffectProcessor) ApplyDynamicSimpleWithIntensity(
	sprite *ebiten.Image,
	outlineColor OutlineColorType,
	intensity float32,
) *ebiten.Image {
	if outlineColor == NoOutline {
		return sprite
	}
	pipeline := Effect().AddOutline(outlineColor, WithIntensity(intensity)).Build()

	return p.ApplyDynamic(sprite, pipeline)
}

// ApplyDynamicMarchingOutline applies a cached black-and-white checker outline.
// Alternating between its two phases creates a color-independent marching effect.
func (p *TileEffectProcessor) ApplyDynamicMarchingOutline(
	sprite *ebiten.Image,
	phase int,
) *ebiten.Image {
	if sprite == nil {
		return nil
	}

	const marchingOutlineCacheTag = uint64(0x4d41524348494e47)
	normalizedPhase := phase & 1
	cachePipeline := Effect().AddOutline(
		WhiteOutline,
		WithIntensity(float32(normalizedPhase)),
	).Build()
	key := p.calculateDynamicCacheKey(sprite, cachePipeline) ^ marchingOutlineCacheTag
	if cached := p.dynamicCache.Get(key); cached != nil {
		return cached
	}

	bounds := sprite.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, width*height*marchingOutlineRGBAChannels)
	if err := p.safeReadPixels(sprite, pixels); err != nil {
		return sprite
	}
	output := make([]byte, len(pixels))
	copy(output, pixels)
	applyMarchingOutlineDynamic(pixels, output, width, height, normalizedPhase)

	result := ebiten.NewImage(width, height)
	result.WritePixels(output)
	p.dynamicCache.Put(key, result)

	return result
}

const marchingOutlineRGBAChannels = 4

func applyMarchingOutlineDynamic(
	input, output []byte,
	width, height, phase int,
) {
	for rowIndex := range height {
		for columnIndex := range width {
			if isMarchingOutlineEdge(input, width, height, columnIndex, rowIndex) {
				paintMarchingOutlineNeighbors(
					input, output, width, height, columnIndex, rowIndex, phase,
				)
			}
		}
	}
}

func isMarchingOutlineEdge(input []byte, width, height, columnIndex, rowIndex int) bool {
	if marchingOutlineAlphaAt(input, width, height, columnIndex, rowIndex) == 0 {
		return false
	}

	return marchingOutlineAlphaAt(input, width, height, columnIndex, rowIndex-1) == 0 ||
		marchingOutlineAlphaAt(input, width, height, columnIndex, rowIndex+1) == 0 ||
		marchingOutlineAlphaAt(input, width, height, columnIndex-1, rowIndex) == 0 ||
		marchingOutlineAlphaAt(input, width, height, columnIndex+1, rowIndex) == 0
}

func marchingOutlineAlphaAt(input []byte, width, height, columnIndex, rowIndex int) byte {
	if columnIndex < 0 || columnIndex >= width || rowIndex < 0 || rowIndex >= height {
		return 0
	}

	return input[(rowIndex*width+columnIndex)*marchingOutlineRGBAChannels+3]
}

func paintMarchingOutlineNeighbors(
	input, output []byte,
	width, height, columnIndex, rowIndex, phase int,
) {
	offsets := [...][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}
	for _, offset := range offsets {
		neighborColumn := columnIndex + offset[0]
		neighborRow := rowIndex + offset[1]
		if marchingOutlineAlphaAt(input, width, height, neighborColumn, neighborRow) != 0 ||
			neighborColumn < 0 || neighborColumn >= width || neighborRow < 0 || neighborRow >= height {
			continue
		}

		pixelIndex := (neighborRow*width + neighborColumn) * marchingOutlineRGBAChannels
		outlineColor := ucolor.EGABlack
		if (neighborColumn+neighborRow+phase)&1 == 0 {
			outlineColor = ucolor.EGAWhite
		}
		output[pixelIndex] = outlineColor.R
		output[pixelIndex+1] = outlineColor.G
		output[pixelIndex+2] = outlineColor.B
		output[pixelIndex+3] = outlineColor.A
	}
}

// ApplyDynamicOutlineRGBA applies an outline with a custom RGBA color and intensity.
// This is useful for status effects and other cases where the color is not a predefined OutlineColorType.
// The intensity controls how prominent the outline is (0.0-1.0).
// Note: Returns original sprite if ReadPixels fails (e.g., in tests before game starts).
func (p *TileEffectProcessor) ApplyDynamicOutlineRGBA(
	sprite *ebiten.Image,
	outlineColor color.Color,
	intensity float32,
) *ebiten.Image {
	if sprite == nil || outlineColor == nil {
		return sprite
	}

	// Convert color.Color to color.RGBA
	r, g, b, a := outlineColor.RGBA()
	rgba := color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}

	// Apply intensity to color
	rgba.R = uint8(float32(rgba.R) * intensity)
	rgba.G = uint8(float32(rgba.G) * intensity)
	rgba.B = uint8(float32(rgba.B) * intensity)
	rgba.A = uint8(float32(rgba.A) * intensity)

	// Read input pixels - may fail before game starts (in tests)
	bounds := sprite.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, width*height*4)
	if err := p.safeReadPixels(sprite, pixels); err != nil {
		return sprite // Return original if ReadPixels fails (e.g., in tests)
	}

	// Create output buffer
	output := make([]byte, len(pixels))
	copy(output, pixels)

	// Detect edges and apply outline with custom color
	p.applyOutlineRGBADynamic(pixels, output, width, height, rgba)

	// Create result image
	result := ebiten.NewImage(width, height)
	result.WritePixels(output)

	return result
}

// applyOutlineRGBADynamic applies outline with a specific RGBA color (no OutlineColorType lookup).
func (p *TileEffectProcessor) applyOutlineRGBADynamic(
	input, output []byte,
	width, height int,
	outlineRGBA color.RGBA,
) {
	getAlpha := func(x, y int) byte {
		if x < 0 || x >= width || y < 0 || y >= height {
			return 0
		}

		return input[(y*width+x)*4+3]
	}

	// Detect edges
	edges := make([]bool, width*height)
	for y := range height {
		for x := range width {
			if getAlpha(x, y) == 0 {
				continue
			}
			if getAlpha(x, y-1) == 0 || getAlpha(x, y+1) == 0 ||
				getAlpha(x-1, y) == 0 || getAlpha(x+1, y) == 0 {
				edges[y*width+x] = true
			}
		}
	}

	// Draw outline
	offsets := [][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for y := range height {
		for x := range width {
			if !edges[y*width+x] {
				continue
			}
			for _, off := range offsets {
				nx, ny := x+off[0], y+off[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				idx := (ny*width + nx) * 4
				if output[idx+3] == 0 {
					output[idx] = outlineRGBA.R
					output[idx+1] = outlineRGBA.G
					output[idx+2] = outlineRGBA.B
					output[idx+3] = outlineRGBA.A
				}
			}
		}
	}
}

// calculateDynamicCacheKey creates a unique hash for sprite + pipeline combination.
// Uses FNV-1a hash for speed.
func (p *TileEffectProcessor) calculateDynamicCacheKey(
	sprite *ebiten.Image,
	pipeline TileEffectPipeline,
) uint64 {
	// FNV-1a hash
	const fnvPrime = 1099511628211
	const fnvOffset = 14695981039346656037

	hash := uint64(fnvOffset)

	// Hash sprite pointer (unique per image)
	spritePtr := uint64(uintptr(unsafe.Pointer(sprite)))
	hash ^= spritePtr
	hash *= fnvPrime

	// Hash number of effects
	hash ^= uint64(len(pipeline.Effects))
	hash *= fnvPrime

	// Hash each effect
	for _, effect := range pipeline.Effects {
		hash ^= uint64(effect.Algorithm)
		hash *= fnvPrime
		hash ^= uint64(effect.OutlineColor)
		hash *= fnvPrime
		hash ^= uint64(effect.Intensity * 1000) // Scale for precision
		hash *= fnvPrime
		hash ^= uint64(effect.Opacity * 1000)
		hash *= fnvPrime
		hash ^= uint64(effect.TintColor.R)<<24 | uint64(effect.TintColor.G)<<16 |
			uint64(effect.TintColor.B)<<8 | uint64(effect.TintColor.A)
		hash *= fnvPrime
	}

	return hash
}

// applyPipelineToSprite applies all effects in a pipeline to an arbitrary sprite.
func (p *TileEffectProcessor) applyPipelineToSprite(
	sprite *ebiten.Image,
	pipeline TileEffectPipeline,
) *ebiten.Image {
	if sprite == nil || len(pipeline.Effects) == 0 {
		return sprite
	}

	currentSprite := sprite
	currentEffects := pipeline.Effects

	// Handle EffectBorderWrap first if present (changes dimensions)
	if len(currentEffects) > 0 && currentEffects[0].Algorithm == EffectBorderWrap {
		currentSprite = WrapSpriteWith18x18Border(currentSprite)
		currentEffects = currentEffects[1:] // remaining effects
	}

	// If no more effects after border wrap, return the wrapped sprite
	if len(currentEffects) == 0 {
		return currentSprite
	}

	bounds := currentSprite.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// Read input pixels
	pixels := make([]byte, width*height*4)
	if err := p.safeReadPixels(currentSprite, pixels); err != nil {
		return currentSprite // Return current sprite if ReadPixels fails (e.g., in tests)
	}

	// Apply remaining effects in order
	for _, effect := range currentEffects {
		pixels = p.applyDynamicEffect(pixels, width, height, effect)
	}

	// Create result image
	result := ebiten.NewImage(width, height)
	result.WritePixels(pixels)

	return result
}

// applyDynamicEffect applies a single effect to pixel data of arbitrary size.
// This is similar to applyEffect but works with non-tile-sized images.
func (p *TileEffectProcessor) applyDynamicEffect(
	pixels []byte,
	width, height int,
	effect TileEffect,
) []byte {
	output := make([]byte, len(pixels))
	copy(output, pixels)

	switch effect.Algorithm {
	case EffectOutline:
		p.applyOutlineDynamic(pixels, output, width, height, effect.OutlineColor, effect.Intensity)
	case EffectOpacity:
		p.applyOpacityDynamic(pixels, output, effect.Opacity)
	case EffectTint:
		p.applyTintDynamic(pixels, output, effect.TintColor)
	case EffectGlow:
		p.applyGlowDynamic(pixels, output, effect.Intensity)
	}

	return output
}

// applyOutlineDynamic applies outline effect to arbitrary-sized images.
func (p *TileEffectProcessor) applyOutlineDynamic(
	input, output []byte,
	width, height int,
	outlineColor OutlineColorType,
	intensity float32,
) {
	if outlineColor == NoOutline || intensity <= 0 {
		return
	}

	rgba := outlineColor.GetOutlineColor()
	r := uint8(float32(rgba.R) * intensity)
	g := uint8(float32(rgba.G) * intensity)
	b := uint8(float32(rgba.B) * intensity)
	a := uint8(float32(rgba.A) * intensity)

	getAlpha := func(x, y int) byte {
		if x < 0 || x >= width || y < 0 || y >= height {
			return 0
		}

		return input[(y*width+x)*4+3]
	}

	// Detect edges
	edges := make([]bool, width*height)
	for y := range height {
		for x := range width {
			if getAlpha(x, y) == 0 {
				continue
			}
			if getAlpha(x, y-1) == 0 || getAlpha(x, y+1) == 0 ||
				getAlpha(x-1, y) == 0 || getAlpha(x+1, y) == 0 {
				edges[y*width+x] = true
			}
		}
	}

	// Draw outline
	offsets := [][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for y := range height {
		for x := range width {
			if !edges[y*width+x] {
				continue
			}
			for _, off := range offsets {
				nx, ny := x+off[0], y+off[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				idx := (ny*width + nx) * 4
				if output[idx+3] == 0 {
					output[idx] = r
					output[idx+1] = g
					output[idx+2] = b
					output[idx+3] = a
				}
			}
		}
	}
}

// applyOpacityDynamic applies opacity effect to arbitrary-sized images.
func (p *TileEffectProcessor) applyOpacityDynamic(
	input, output []byte,
	opacity float64,
) {
	if opacity >= 1.0 {
		return
	}
	opacityScale := float32(opacity)
	for i := 0; i < len(input); i += 4 {
		output[i+3] = uint8(float32(input[i+3]) * opacityScale)
	}
}

// applyTintDynamic applies tint effect to arbitrary-sized images.
func (p *TileEffectProcessor) applyTintDynamic(
	input, output []byte,
	tintColor color.RGBA,
) {
	rFactor := float32(tintColor.R) / 255.0
	gFactor := float32(tintColor.G) / 255.0
	bFactor := float32(tintColor.B) / 255.0

	for i := 0; i < len(input); i += 4 {
		if input[i+3] == 0 {
			continue
		}
		output[i] = uint8(float32(input[i]) * rFactor)
		output[i+1] = uint8(float32(input[i+1]) * gFactor)
		output[i+2] = uint8(float32(input[i+2]) * bFactor)
	}
}

// applyGlowDynamic applies glow effect to arbitrary-sized images.
func (p *TileEffectProcessor) applyGlowDynamic(
	input, output []byte,
	intensity float32,
) {
	if intensity <= 0 {
		return
	}
	for i := 0; i < len(input); i += 4 {
		if input[i+3] == 0 {
			continue
		}
		r := int(float32(input[i]) * intensity)
		g := int(float32(input[i+1]) * intensity)
		b := int(float32(input[i+2]) * intensity)
		if r > 255 {
			r = 255
		}
		if g > 255 {
			g = 255
		}
		if b > 255 {
			b = 255
		}
		output[i] = uint8(r)
		output[i+1] = uint8(g)
		output[i+2] = uint8(b)
	}
}

// generateFrames creates all frame variants for a tile.
func (p *TileEffectProcessor) generateFrames(
	spriteIndex indexes.SpriteIndex,
	pipeline TileEffectPipeline,
	frameCount int,
) []*ebiten.Image {
	frames := make([]*ebiten.Image, frameCount)

	baseTile := p.spriteSheet.getOriginal16x16Sprite(spriteIndex)

	for frame := range frameCount {
		frames[frame] = p.generateSingleFrame(baseTile, frame, pipeline, frameCount)
	}

	return frames
}

// generateSingleFrame creates a single frame variant.
func (p *TileEffectProcessor) generateSingleFrame(
	baseTile *ebiten.Image,
	frameIndex int,
	pipeline TileEffectPipeline,
	frameCount int,
) *ebiten.Image {
	if baseTile == nil {
		return nil
	}

	// Create output image
	result := ebiten.NewImage(TileSize, TileSize)

	// Step 1: Draw background if defined
	if pipeline.Background >= 0 {
		bgTile := p.spriteSheet.getOriginal16x16Sprite(pipeline.Background)
		result.DrawImage(bgTile, nil)
	}

	// Step 2: Read base tile pixels
	pixels := make([]byte, TileSize*TileSize*4)
	baseTile.ReadPixels(pixels)

	// Step 3: Apply effects in order (pipeline)
	processedPixels := make([]byte, TileSize*TileSize*4)
	copy(processedPixels, pixels)

	for _, effect := range pipeline.Effects {
		// Apply effect, using output of previous as input
		processedPixels = p.applyEffect(processedPixels, frameIndex, pipeline.Bounds, effect, frameCount)
	}

	// Step 4: Create temp image and composite over background
	tempImage := ebiten.NewImage(TileSize, TileSize)
	tempImage.WritePixels(processedPixels)
	result.DrawImage(tempImage, nil)

	return result
}

// applyEffect applies a single effect to the pixel buffer.
func (p *TileEffectProcessor) applyEffect(
	pixels []byte,
	frameIndex int,
	bounds image.Rectangle,
	effect TileEffect,
	frameCount int,
) []byte {
	output := make([]byte, len(pixels))
	copy(output, pixels)

	// Use per-effect bounds if set, otherwise use pipeline-level bounds
	effectBounds := effect.Bounds
	if effectBounds.Empty() {
		effectBounds = bounds
	}

	switch effect.Algorithm {
	case EffectVerticalRise:
		p.applyVerticalRise(pixels, output, frameIndex, effectBounds, effect.Filter)
	case EffectWaveDisplace:
		p.applyWaveDisplace(pixels, output, frameIndex, effectBounds, effect.Filter)
	case EffectVerticalScroll:
		p.applyVerticalScroll(pixels, output, frameIndex, frameCount)
	case EffectTransparencyOverlay:
		p.applyTransparencyOverlay(pixels, output, frameIndex, effect.Background)
	case EffectOutline:
		p.applyOutline(pixels, output, effect.OutlineColor, effect.Intensity)
	case EffectOpacity:
		p.applyOpacity(pixels, output, effect.Opacity)
	case EffectTint:
		p.applyTint(pixels, output, effect.TintColor)
	case EffectGlow:
		p.applyGlow(pixels, output, effect.Intensity)
	case EffectPulse:
		p.applyPulse(pixels, output, frameIndex, effect.Filter)
	case EffectBrightnessPulse:
		p.applyBrightnessPulse(pixels, output, frameIndex, effect.Filter)
	}

	return output
}

// ============================================================================
// EFFECT ALGORITHM IMPLEMENTATIONS
// ============================================================================

// applyVerticalRise implements the vertical rise effect with horizontal jitter.
// Simulates flames rising upward by borrowing pixels from below with random horizontal displacement.
func (p *TileEffectProcessor) applyVerticalRise(
	input, output []byte,
	frameIndex int,
	bounds image.Rectangle,
	filter ColorFilter,
) {
	hasBounds := !bounds.Empty()

	for y := range TileSize {
		for x := range TileSize {
			if hasBounds && !image.Pt(x, y).In(bounds) {
				continue
			}

			pixelIndex := (y*TileSize + x) * 4
			if input[pixelIndex+3] == 0 {
				continue
			}

			// Check color filter
			if !p.passesFilter(input, pixelIndex, filter) {
				continue
			}

			seed := (x*31 + y*17 + frameIndex*23) % 100

			sourceX, sourceY := x, y

			// Vertical rise: 30% chance to borrow from row below
			if seed < 30 {
				sourceY = y + 1
			}

			// Horizontal jitter
			horizSeed := (seed * 7) % 100
			if horizSeed < 20 {
				sourceX = x - 1
			} else if horizSeed < 40 {
				sourceX = x + 1
			}

			// Clamp to tile bounds first
			sourceX = clampInt(sourceX, 0, TileSize-1)
			sourceY = clampInt(sourceY, 0, TileSize-1)

			// Then clamp to effect bounds if specified
			if hasBounds {
				sourceX = clampInt(sourceX, bounds.Min.X, bounds.Max.X-1)
				sourceY = clampInt(sourceY, bounds.Min.Y, bounds.Max.Y-1)
			}

			sourceIndex := (sourceY*TileSize + sourceX) * 4
			for c := range 4 {
				output[pixelIndex+c] = input[sourceIndex+c]
			}
		}
	}
}

// applyPulse fades selected pixels in and out without changing their color or position.
func (p *TileEffectProcessor) applyPulse(input, output []byte, frameIndex int, filter ColorFilter) {
	pulse := pulseMinimumOpacity + pulseOpacityAmplitude*
		(math.Sin(float64(frameIndex)*math.Pi/pulsePhaseDivisor)+1)/pulseWaveRange
	for pixelIndex := 0; pixelIndex < len(input); pixelIndex += 4 {
		if input[pixelIndex+3] == 0 || !p.passesFilter(input, pixelIndex, filter) {
			continue
		}
		output[pixelIndex+3] = uint8(float64(input[pixelIndex+3]) * pulse)
	}
}

// applyBrightnessPulse changes selected pixels' brightness without fragmenting the sprite silhouette.
func (p *TileEffectProcessor) applyBrightnessPulse(input, output []byte, frameIndex int, filter ColorFilter) {
	brightness := brightnessPulseMinimum + brightnessPulseAmplitude*
		(math.Sin(float64(frameIndex)*math.Pi/pulsePhaseDivisor)+1)/pulseWaveRange
	for pixelIndex := 0; pixelIndex < len(input); pixelIndex += 4 {
		if input[pixelIndex+3] == 0 || !p.passesFilter(input, pixelIndex, filter) {
			continue
		}
		for colorIndex := range 3 {
			output[pixelIndex+colorIndex] = uint8(float64(input[pixelIndex+colorIndex]) * brightness)
		}
	}
}

// applyWaveDisplace implements sine-wave horizontal displacement.
// Each row shifts together based on a wave pattern for realistic lava/liquid wavering.
func (p *TileEffectProcessor) applyWaveDisplace(
	input, output []byte,
	frameIndex int,
	bounds image.Rectangle,
	filter ColorFilter,
) {
	hasBounds := !bounds.Empty()

	for y := range TileSize {
		// Calculate wave displacement for this row.
		// Different rows have different phase offsets for natural wave appearance.
		wavePhase := float64(frameIndex)*0.8 + float64(y)*0.5
		displacement := int(math.Sin(wavePhase) * 1.1) // -1, 0, or +1

		for x := range TileSize {
			if hasBounds && !image.Pt(x, y).In(bounds) {
				continue
			}

			pixelIndex := (y*TileSize + x) * 4
			if input[pixelIndex+3] == 0 {
				continue
			}

			// Check color filter
			if !p.passesFilter(input, pixelIndex, filter) {
				continue
			}

			sourceX := x + displacement

			// Clamp to tile bounds first
			sourceX = clampInt(sourceX, 0, TileSize-1)

			// Then clamp to effect bounds if specified
			if hasBounds {
				sourceX = clampInt(sourceX, bounds.Min.X, bounds.Max.X-1)
			}

			sourceIndex := (y*TileSize + sourceX) * 4
			for c := range 4 {
				output[pixelIndex+c] = input[sourceIndex+c]
			}
		}
	}
}

// applyVerticalScroll implements vertical pixel scrolling with wrap-around.
// Creates authentic Ultima V water animation where pixels scroll downward.
// Each frame shifts pixels down by frameIndex positions, wrapping at edges.
func (p *TileEffectProcessor) applyVerticalScroll(
	input, output []byte,
	frameIndex int,
	frameCount int,
) {
	// scrollOffset represents how many pixels to shift downward
	scrollOffset := frameIndex % frameCount

	for y := range TileSize {
		// Calculate source Y with wrap-around
		// For downward scrolling: destination Y gets source from Y - offset (with wrap)
		sourceY := (y - scrollOffset + TileSize) % TileSize

		for x := range TileSize {
			destIndex := (y*TileSize + x) * 4
			sourceIndex := (sourceY*TileSize + x) * 4

			// Copy all 4 channels (RGBA)
			for c := range 4 {
				output[destIndex+c] = input[sourceIndex+c]
			}
		}
	}
}

// applyTransparencyOverlay creates a transparency mask for blue/water pixels.
// Blue pixels become transparent, revealing animated water background beneath.
// This implements authentic mixed land/water tile animation by:
// 1. Generating the scrolled background frame for this frameIndex
// 2. For each pixel: if it's blue/water-colored -> use background pixel, otherwise keep original
func (p *TileEffectProcessor) applyTransparencyOverlay(
	input, output []byte,
	frameIndex int,
	backgroundTile indexes.SpriteIndex,
) {
	// Get scrolled background frame for this frameIndex
	// The background tile (typically WaterCoast) provides the animated water
	backgroundFrames, exists := p.effectFrames[backgroundTile]
	if !exists || len(backgroundFrames) == 0 {
		// Background not available - just copy input as-is
		copy(output, input)

		return
	}

	// Get the correct background frame
	bgFrameIndex := frameIndex % len(backgroundFrames)
	bgFrame := backgroundFrames[bgFrameIndex]
	if bgFrame == nil {
		copy(output, input)

		return
	}

	// Read background pixels
	bgPixels := make([]byte, TileSize*TileSize*4)
	bgFrame.ReadPixels(bgPixels)

	// Process each pixel
	for y := range TileSize {
		for x := range TileSize {
			pixelIndex := (y*TileSize + x) * 4
			r := input[pixelIndex]
			g := input[pixelIndex+1]
			b := input[pixelIndex+2]
			a := input[pixelIndex+3]

			if isBlueWaterPixel(r, g, b, a) {
				// Blue/water pixel - use background (animated water)
				output[pixelIndex] = bgPixels[pixelIndex]
				output[pixelIndex+1] = bgPixels[pixelIndex+1]
				output[pixelIndex+2] = bgPixels[pixelIndex+2]
				output[pixelIndex+3] = bgPixels[pixelIndex+3]
			} else {
				// Non-water pixel - keep original
				output[pixelIndex] = r
				output[pixelIndex+1] = g
				output[pixelIndex+2] = b
				output[pixelIndex+3] = a
			}
		}
	}
}

// isBlueWaterPixel determines if a pixel represents water color that should be animated.
// Used by EffectTransparencyOverlay to identify which pixels to replace with animated water.
// Uses EGA palette matching to identify water-colored pixels.
func isBlueWaterPixel(r, g, b, a uint8) bool {
	const (
		alphaTransparencyThreshold = 128 // Alpha below this is considered transparent
		grayscaleThreshold         = 30  // Color difference below this is grayscale
	)

	// Skip already transparent pixels - treat them as water too
	if a < alphaTransparencyThreshold {
		return true
	}

	// Exclude grayscale colors (bridge structures, stone, etc.)
	// If R, G, and B are very close to each other, it's likely grayscale (not water)
	maxVal := max(r, g, b)
	minVal := min(r, g, b)
	if maxVal-minVal < grayscaleThreshold {
		return false // Don't animate grayscale colors
	}

	// Match EGA water colors with tolerance
	return ucolor.EGABlue.MatchesWithTolerance(r, g, b, colorTolerance) ||
		ucolor.EGACyan.MatchesWithTolerance(r, g, b, colorTolerance) ||
		ucolor.EGALightBlue.MatchesWithTolerance(r, g, b, colorTolerance) ||
		ucolor.EGALightCyan.MatchesWithTolerance(r, g, b, colorTolerance)
}

// ============================================================================
// COLOR FILTER HELPERS
// ============================================================================

const colorTolerance = 10

// passesFilter returns true if the pixel at the given index passes the color filter.
func (p *TileEffectProcessor) passesFilter(pixels []byte, pixelIndex int, filter ColorFilter) bool {
	if filter == FilterAll {
		return true
	}

	r := pixels[pixelIndex]
	g := pixels[pixelIndex+1]
	b := pixels[pixelIndex+2]

	switch filter {
	case FilterRedOnly:
		return isEGARed(r, g, b)
	case FilterYellowOnly:
		return isEGAYellow(r, g, b)
	case FilterFire:
		return isEGARed(r, g, b) || isEGALightRed(r, g, b)
	case FilterBlueFire:
		return isEGABlue(r, g, b) || isEGALightBlue(r, g, b)
	case FilterPoison:
		return isEGAGreen(r, g, b) || isEGALightGreen(r, g, b)
	case FilterSleep:
		return isEGAMagenta(r, g, b) || isEGALightMagenta(r, g, b)
	case FilterProtection:
		return isEGABlue(r, g, b) || isEGALightBlue(r, g, b)
	default:
		return true
	}
}

func isEGALightBlue(r, g, b uint8) bool {
	return ucolor.EGALightBlue.MatchesWithTolerance(r, g, b, colorTolerance)
}

func isEGAGreen(r, g, b uint8) bool {
	return ucolor.EGAGreen.MatchesWithTolerance(r, g, b, colorTolerance)
}

func isEGALightGreen(r, g, b uint8) bool {
	return ucolor.EGALightGreen.MatchesWithTolerance(r, g, b, colorTolerance)
}

func isEGAMagenta(r, g, b uint8) bool {
	return ucolor.EGAMagenta.MatchesWithTolerance(r, g, b, colorTolerance)
}

func isEGALightMagenta(r, g, b uint8) bool {
	return ucolor.EGALightMagenta.MatchesWithTolerance(r, g, b, colorTolerance)
}

func isEGABlue(r, g, b uint8) bool {
	return ucolor.EGABlue.MatchesWithTolerance(r, g, b, colorTolerance)
}

// isEGARed checks if a color matches EGA Red (170, 0, 0) within tolerance.
func isEGARed(r, g, b uint8) bool {
	return ucolor.EGARed.MatchesWithTolerance(r, g, b, colorTolerance)
}

// isEGALightRed checks if a color matches EGA Light Red (255, 85, 85) within tolerance.
func isEGALightRed(r, g, b uint8) bool {
	return ucolor.EGALightRed.MatchesWithTolerance(r, g, b, colorTolerance)
}

// isEGAYellow checks if a color matches EGA Yellow (255, 255, 85) within tolerance.
func isEGAYellow(r, g, b uint8) bool {
	return ucolor.EGAYellow.MatchesWithTolerance(r, g, b, colorTolerance)
}

// ============================================================================
// MATH HELPERS
// ============================================================================

// clampInt constrains a value to [min, max].
func clampInt(val, minVal, maxVal int) int {
	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}

	return val
}

// ============================================================================
// DYNAMIC EFFECT ALGORITHM IMPLEMENTATIONS
// ============================================================================

// applyOutline adds a colored outline around non-transparent pixels.
// Uses 4-neighbor edge detection to find sprite edges.
func (p *TileEffectProcessor) applyOutline(
	input, output []byte,
	outlineColor OutlineColorType,
	intensity float32,
) {
	if outlineColor == NoOutline || intensity <= 0 {
		return
	}

	// Get the actual RGBA color
	rgba := outlineColor.GetOutlineColor()

	// Apply intensity
	r := uint8(float32(rgba.R) * intensity)
	g := uint8(float32(rgba.G) * intensity)
	b := uint8(float32(rgba.B) * intensity)
	a := uint8(float32(rgba.A) * intensity)

	// First pass: detect edges (non-transparent pixels with transparent neighbors)
	edges := make([]bool, TileSize*TileSize)

	getAlpha := func(x, y int) byte {
		if x < 0 || x >= TileSize || y < 0 || y >= TileSize {
			return 0
		}

		return input[(y*TileSize+x)*4+3]
	}

	for y := range TileSize {
		for x := range TileSize {
			alpha := getAlpha(x, y)
			if alpha == 0 {
				continue
			}

			// Check 4 neighbors
			if getAlpha(x, y-1) == 0 || getAlpha(x, y+1) == 0 ||
				getAlpha(x-1, y) == 0 || getAlpha(x+1, y) == 0 {
				edges[y*TileSize+x] = true
			}
		}
	}

	// Second pass: draw outline pixels at offset positions around edges
	offsets := [][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for y := range TileSize {
		for x := range TileSize {
			if !edges[y*TileSize+x] {
				continue
			}

			for _, off := range offsets {
				nx, ny := x+off[0], y+off[1]
				if nx < 0 || nx >= TileSize || ny < 0 || ny >= TileSize {
					continue
				}

				idx := (ny*TileSize + nx) * 4
				// Only draw outline on transparent pixels
				if output[idx+3] == 0 {
					output[idx] = r
					output[idx+1] = g
					output[idx+2] = b
					output[idx+3] = a
				}
			}
		}
	}
}

// applyOpacity applies alpha transparency to the sprite.
func (p *TileEffectProcessor) applyOpacity(
	input, output []byte,
	opacity float64,
) {
	if opacity >= 1.0 {
		return // No change needed
	}

	opacityScale := float32(opacity)
	for i := 0; i < len(input); i += 4 {
		// Scale alpha channel
		alpha := float32(input[i+3])
		output[i+3] = uint8(alpha * opacityScale)
	}
}

// applyTint applies a color tint overlay to the sprite.
func (p *TileEffectProcessor) applyTint(
	input, output []byte,
	tintColor color.RGBA,
) {
	// Calculate tint factors (normalize to 0.0-1.0)
	rFactor := float32(tintColor.R) / 255.0
	gFactor := float32(tintColor.G) / 255.0
	bFactor := float32(tintColor.B) / 255.0

	for i := 0; i < len(input); i += 4 {
		if input[i+3] == 0 {
			continue // Skip transparent pixels
		}

		// Multiply RGB by tint factors
		output[i] = uint8(float32(input[i]) * rFactor)
		output[i+1] = uint8(float32(input[i+1]) * gFactor)
		output[i+2] = uint8(float32(input[i+2]) * bFactor)
		// Alpha unchanged
	}
}

// applyGlow applies a brightness modulation for glowing effects.
func (p *TileEffectProcessor) applyGlow(
	input, output []byte,
	intensity float32,
) {
	if intensity <= 0 {
		return
	}

	// Brighten pixels by the intensity factor
	for i := 0; i < len(input); i += 4 {
		if input[i+3] == 0 {
			continue // Skip transparent pixels
		}

		// Scale RGB channels (clamp to 255)
		r := int(float32(input[i]) * intensity)
		g := int(float32(input[i+1]) * intensity)
		b := int(float32(input[i+2]) * intensity)

		if r > 255 {
			r = 255
		}
		if g > 255 {
			g = 255
		}
		if b > 255 {
			b = 255
		}

		output[i] = uint8(r)
		output[i+1] = uint8(g)
		output[i+2] = uint8(b)
		// Alpha unchanged
	}
}
