package sprites

import (
	"fmt"
	"image"
	"image/color"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// OutlineColorType selects game-owned semantic outline colours.
type OutlineColorType int

// Outline roles are presentation choices, never persisted gameplay identities.
const (
	NoOutline OutlineColorType = iota
	YellowOutline
	WhiteOutline
	RedOutline
	BlueOutline
	GreenOutline
	PurpleOutline
	BlackOutline
	CombatActivePCOutline
	CombatInactivePCOutline
	CombatActiveMonsterOutline
	CombatThreatenedPCOutline
)

// GetOutlineColor resolves the game palette without exporting it to the SDK.
func (kind OutlineColorType) GetOutlineColor() color.RGBA {
	switch kind {
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

// EffectAlgorithm identifies the operations actually used by game presets.
type EffectAlgorithm int

// Unused opacity/tint/glow declarations were retired; RGB gain is SDK-owned.
const (
	EffectVerticalRise EffectAlgorithm = iota
	EffectWaveDisplace
	EffectVerticalScroll
	EffectTransparencyOverlay
	EffectOutline
	EffectPulse
	EffectBrightnessPulse
)

const (
	defaultEffectIntervalMs  = 150
	defaultFireFrameCount    = 8
	waterFrameCount          = 16
	defaultWaterEvenXOffset  = 6
	pulsePhaseDivisor        = 4
	pulseWaveRange           = 2
	pulseMinimumOpacity      = .15
	pulseOpacityAmplitude    = .80
	brightnessPulseMinimum   = .70
	brightnessPulseAmplitude = .30
	tilePixelChannels        = 4
)

// PreferredFrameCount retains the original game families' frame budgets.
func (algorithm EffectAlgorithm) PreferredFrameCount() int {
	switch algorithm {
	case EffectVerticalScroll, EffectTransparencyOverlay:
		return waterFrameCount
	case EffectVerticalRise, EffectWaveDisplace, EffectOutline, EffectPulse, EffectBrightnessPulse:
		return defaultFireFrameCount
	default:
		panic("unknown game effect algorithm")
	}
}

// ColorFilter interprets the game's original artwork palette.
type ColorFilter int

// Palette selections stay in the game; the SDK receives immutable binary masks.
const (
	FilterAll ColorFilter = iota
	FilterRedOnly
	FilterYellowOnly
	FilterFire
	FilterBlueFire
	FilterPoison
	FilterSleep
	FilterProtection
)

// PositionOffsetMode describes the game's authored animation phase convention.
type PositionOffsetMode int

// Spatial phase selection remains separate from SDK frame sampling.
const (
	PositionOffsetDefault PositionOffsetMode = iota
	PositionOffsetEvenX
)

// TileEffect contains only parameters consumed by the game adapter.
type TileEffect struct {
	Algorithm    EffectAlgorithm
	Filter       ColorFilter
	Background   indexes.SpriteIndex
	OutlineColor OutlineColorType
	Intensity    float32
}

// EffectOption customizes an outline's game appearance.
type EffectOption func(*TileEffect)

// WithIntensity selects an outline's colour multiplier.
func WithIntensity(intensity float32) EffectOption {
	return func(effect *TileEffect) { effect.Intensity = intensity }
}

// TileEffectPipeline retains game tile selection, bounds, dependencies and phase policy.
type TileEffectPipeline struct {
	Background  indexes.SpriteIndex
	Bounds      image.Rectangle
	Effects     []TileEffect
	IntervalMs  int
	OffsetMode  PositionOffsetMode
	EvenXOffset int
	FrameCount  int
}

// TileEffectBuilder keeps existing game preset composition outside the SDK.
type TileEffectBuilder struct{ pipeline TileEffectPipeline }

// Effect begins an original-tile preset with no background or selected operations.
func Effect() *TileEffectBuilder {
	builder := new(TileEffectBuilder)
	builder.pipeline.Background = -1
	builder.pipeline.Effects = make([]TileEffect, 0)

	return builder
}

// WithBackground selects the original underlay tile.
func (builder *TileEffectBuilder) WithBackground(sprite indexes.SpriteIndex) *TileEffectBuilder {
	builder.pipeline.Background = sprite

	return builder
}

// WithBounds translates inclusive original-tile coordinates to a rectangle.
func (builder *TileEffectBuilder) WithBounds(x1, y1, x2, y2 int) *TileEffectBuilder {
	builder.pipeline.Bounds = image.Rect(x1, y1, x2+1, y2+1)

	return builder
}

// WithInterval supplies the game frame interval in milliseconds.
func (builder *TileEffectBuilder) WithInterval(milliseconds int) *TileEffectBuilder {
	builder.pipeline.IntervalMs = milliseconds

	return builder
}

// WithEvenXOffset selects the original alternating-column water phase.
func (builder *TileEffectBuilder) WithEvenXOffset(offset int) *TileEffectBuilder {
	builder.pipeline.OffsetMode, builder.pipeline.EvenXOffset = PositionOffsetEvenX, offset

	return builder
}

func (builder *TileEffectBuilder) add(
	algorithm EffectAlgorithm, filter ColorFilter, background indexes.SpriteIndex,
) *TileEffectBuilder {
	builder.pipeline.Effects = append(builder.pipeline.Effects, TileEffect{
		Algorithm: algorithm, Filter: filter, Background: background, OutlineColor: NoOutline, Intensity: 1,
	})

	return builder
}

// AddVerticalRise animates selected original flame pixels.
func (builder *TileEffectBuilder) AddVerticalRise(filter ColorFilter) *TileEffectBuilder {
	return builder.add(EffectVerticalRise, filter, -1)
}

// AddWaveDisplace applies the game's lava displacement parameters.
func (builder *TileEffectBuilder) AddWaveDisplace(filter ColorFilter) *TileEffectBuilder {
	return builder.add(EffectWaveDisplace, filter, -1)
}

// AddVerticalScroll selects one-pixel-per-frame water scrolling.
func (builder *TileEffectBuilder) AddVerticalScroll() *TileEffectBuilder {
	return builder.add(EffectVerticalScroll, FilterAll, -1)
}

// AddTransparencyOverlay replaces original water pixels using another tile's frames.
func (builder *TileEffectBuilder) AddTransparencyOverlay(background indexes.SpriteIndex) *TileEffectBuilder {
	return builder.add(EffectTransparencyOverlay, FilterAll, background)
}

// AddOutline selects game colours; the SDK supplies the contour mechanism.
func (builder *TileEffectBuilder) AddOutline(colour OutlineColorType, options ...EffectOption) *TileEffectBuilder {
	builder.add(EffectOutline, FilterAll, -1)
	effect := &builder.pipeline.Effects[len(builder.pipeline.Effects)-1]
	effect.OutlineColor = colour
	for _, option := range options {
		option(effect)
	}

	return builder
}

// AddPulse preserves the sleep field's independent alpha-channel pulse.
func (builder *TileEffectBuilder) AddPulse(filter ColorFilter) *TileEffectBuilder {
	return builder.add(EffectPulse, filter, -1)
}

// AddBrightnessPulse preserves protection-field RGB modulation with fixed alpha.
func (builder *TileEffectBuilder) AddBrightnessPulse(filter ColorFilter) *TileEffectBuilder {
	return builder.add(EffectBrightnessPulse, filter, -1)
}

// Build validates this game-specific preset; generic SDK operations have their own errors.
func (builder *TileEffectBuilder) Build() TileEffectPipeline {
	if len(builder.pipeline.Effects) == 0 {
		panic("TileEffectPipeline: at least one effect required")
	}
	bounds := builder.pipeline.Bounds
	if !bounds.Empty() && (bounds.Min.X < 0 || bounds.Min.Y < 0 || bounds.Max.X > TileSize || bounds.Max.Y > TileSize) {
		panic(fmt.Sprintf("TileEffectPipeline: bounds outside original tile: %v", bounds))
	}

	return builder.pipeline
}
