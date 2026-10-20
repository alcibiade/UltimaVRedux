package sprites

import (
	"fmt"
	"image/color"
	"time"

	graphics "github.com/bradhannah/berad-sdk/modules/berad-graphics"
	graphicsbackend "github.com/bradhannah/berad-sdk/modules/berad-graphics/backend/ebitengine"
	"github.com/hajimehoshi/ebiten/v2"
)

// OverlayAlgorithm distinguishes independent additive channels from base-sprite opacity.
type OverlayAlgorithm int

// Additive alpha modulation deliberately does not pulse the added RGB contribution.
const (
	OverlayAdditiveAlphaPulse OverlayAlgorithm = iota
	OverlayAdditiveColour
	OverlayPulsingAlpha
)

// OverlayEffect retains the game's palette and timing preset.
type OverlayEffect struct {
	Algorithm           OverlayAlgorithm
	Color               color.RGBA
	Alpha               float32
	PeriodMs            int64
	PulseLow, PulseHigh float32
	BlendMode           ebiten.Blend
}

// OverlayPipeline composes game-selected treatments; the SDK owns their image mechanics.
type OverlayPipeline struct{ Effects []OverlayEffect }

// IsEmpty reports whether the preset contains effects.
func (pipeline OverlayPipeline) IsEmpty() bool { return len(pipeline.Effects) == 0 }

// ApplyOverlayWithMask preserves placement, palette and the established additive-channel appearance.
func (pipeline OverlayPipeline) ApplyOverlayWithMask(
	target *ebiten.Image, placement *ebiten.DrawImageOptions, elapsedMs int64, opacity float32, mask *ebiten.Image,
) {
	if pipeline.IsEmpty() || opacity <= 0 {
		return
	}
	if mask == nil {
		mask = getGlowTexture()
	}
	for _, effect := range pipeline.Effects {
		effect.apply(target, placement, mask, elapsedMs, opacity)
	}
}

// ApplyOverlayAtPosition retains the game's cosmetic wall-clock/spatial phase choice.
func (pipeline OverlayPipeline) ApplyOverlayAtPosition(
	target *ebiten.Image, placement *ebiten.DrawImageOptions, x, y int, opacity float32, mask *ebiten.Image,
) {
	pipeline.ApplyOverlayWithMask(target, placement, cosmeticOverlayTime(x, y), opacity, mask)
}

func cosmeticOverlayTime(x, y int) int64 {
	const packedRow, phasePrime = 256, 17

	return time.Now().UnixMilli() + int64(x*packedRow+y)*phasePrime // ALLOWED: presentation-only phase, never gameplay.
}

func (effect OverlayEffect) apply(
	target *ebiten.Image, placement *ebiten.DrawImageOptions, mask *ebiten.Image, elapsedMs int64, opacity float32,
) {
	alpha := effect.Alpha
	switch effect.Algorithm {
	case OverlayAdditiveAlphaPulse:
		alpha = effect.calculatePulsingAlpha(elapsedMs)
	case OverlayAdditiveColour:
	case OverlayPulsingAlpha:
		return
	}
	options := &ebiten.DrawImageOptions{GeoM: placement.GeoM, Blend: effect.BlendMode}
	if err := graphicsbackend.DrawAdditiveChannels(target, mask, effect.Color.R, effect.Color.G, effect.Color.B,
		alpha*opacity, *options); err != nil {
		panic(fmt.Sprintf("draw game additive overlay: %v", err))
	}
}

func (effect OverlayEffect) calculatePulsingAlpha(elapsedMs int64) float32 {
	const defaultPeriod = 1500
	period := effect.PeriodMs
	if period <= 0 {
		period = defaultPeriod
	}
	sine, err := graphics.Pulse(elapsedMs, period, -1, 1)
	if err != nil {
		panic(fmt.Sprintf("sample game overlay phase: %v", err))
	}
	// Retain float32 preset interpolation; only the explicit-time sine mechanism moved.
	const centerDivisor = 2
	alphaRange := effect.PulseHigh - effect.PulseLow
	center := (effect.PulseHigh + effect.PulseLow) / centerDivisor

	return center + (alphaRange/centerDivisor)*float32(sine)
}

// GetPulsingAlpha supplies a multiplier for the caller's ordinary ScaleAlpha draw.
func (pipeline OverlayPipeline) GetPulsingAlpha(elapsedMs int64) float32 {
	alpha := float32(1)
	for _, effect := range pipeline.Effects {
		if effect.Algorithm == OverlayPulsingAlpha {
			alpha *= effect.calculatePulsingAlpha(elapsedMs)
		}
	}

	return alpha
}

// GetPulsingAlphaAtPosition retains cosmetic spatial de-synchronization in the host.
func (pipeline OverlayPipeline) GetPulsingAlphaAtPosition(x, y int) float32 {
	return pipeline.GetPulsingAlpha(cosmeticOverlayTime(x, y))
}

// OverlayBuilder composes game treatment presets without exposing a generic tile API.
type OverlayBuilder struct{ pipeline OverlayPipeline }

// Overlay begins an empty game preset.
func Overlay() *OverlayBuilder { return new(OverlayBuilder) }

// WithAdditiveAlphaPulse preserves constant added RGB with an independently cycling alpha channel.
func (builder *OverlayBuilder) WithAdditiveAlphaPulse(
	paint color.RGBA, period int64, low, high float32,
) *OverlayBuilder {
	builder.pipeline.Effects = append(builder.pipeline.Effects, OverlayEffect{
		Algorithm: OverlayAdditiveAlphaPulse, Color: paint, Alpha: 0, PeriodMs: period, PulseLow: low, PulseHigh: high,
		BlendMode: ebiten.BlendLighter,
	})

	return builder
}

// WithAdditiveColour supplies a constant RGB/alpha additive layer.
func (builder *OverlayBuilder) WithAdditiveColour(paint color.RGBA, alpha float32) *OverlayBuilder {
	builder.pipeline.Effects = append(builder.pipeline.Effects, OverlayEffect{
		Algorithm: OverlayAdditiveColour, Color: paint, Alpha: alpha, PeriodMs: 0, PulseLow: 0, PulseHigh: 0,
		BlendMode: ebiten.BlendLighter,
	})

	return builder
}

// WithPulsingAlpha selects base-sprite opacity modulation, not a separate drawn overlay.
func (builder *OverlayBuilder) WithPulsingAlpha(period int64, low, high float32) *OverlayBuilder {
	var effect OverlayEffect
	effect.Algorithm, effect.PeriodMs, effect.PulseLow, effect.PulseHigh = OverlayPulsingAlpha, period, low, high
	builder.pipeline.Effects = append(builder.pipeline.Effects, effect)

	return builder
}

// Build detaches the preset's effect list from later builder changes.
func (builder *OverlayBuilder) Build() OverlayPipeline {
	return OverlayPipeline{Effects: append([]OverlayEffect(nil), builder.pipeline.Effects...)}
}

//nolint:gochecknoglobals // Application-owned immutable fallback and bounded host-serialized mask resources.
var (
	glowTexture         *ebiten.Image
	spriteMaskProcessor = newGraphicsProcessor(DefaultDynamicCacheSize)
)

func getGlowTexture() *ebiten.Image {
	if glowTexture == nil {
		glowTexture = ebiten.NewImage(TileSize, TileSize)
		glowTexture.Fill(color.White)
	}

	return glowTexture
}

// GetSpriteMask borrows a binary occupancy mask from the bounded SDK resource store.
func GetSpriteMask(source *ebiten.Image) *ebiten.Image {
	if source == nil {
		return nil
	}
	mask, err := spriteMaskProcessor.Silhouette(source, 0)
	if err != nil {
		panic(fmt.Sprintf("prepare game silhouette: %v", err))
	}

	return mask
}
