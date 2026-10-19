package sprites

import (
	"fmt"
	"image"
	"math"

	graphics "github.com/bradhannah/berad-sdk/modules/berad-graphics"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func (processor *TileEffectProcessor) generateFrames(
	sprite indexes.SpriteIndex, pipeline TileEffectPipeline, count int,
) []*ebiten.Image {
	source := processor.spriteSheet.getOriginal16x16Sprite(sprite)
	frames := make([]*ebiten.Image, count)
	for index := range count {
		frames[index] = processor.generateSingleFrame(source, index, pipeline, count)
	}

	return frames
}

// Original underlay choice and dependency lookup remain game-owned. Every pixel
// operation delegates to the supplied-image SDK; no second implementation remains.
func (processor *TileEffectProcessor) generateSingleFrame(
	source *ebiten.Image, frame int, pipeline TileEffectPipeline, count int,
) *ebiten.Image {
	result := ebiten.NewImage(TileSize, TileSize)
	if pipeline.Background >= 0 {
		result.DrawImage(processor.spriteSheet.getOriginal16x16Sprite(pipeline.Background), nil)
	}
	pixels := make([]byte, TileSize*TileSize*tilePixelChannels)
	source.ReadPixels(pixels)
	for _, effect := range pipeline.Effects {
		pixels = processor.applyEffect(pixels, frame, pipeline.Bounds, effect, count)
	}
	foreground := ebiten.NewImage(TileSize, TileSize)
	defer foreground.Deallocate()
	foreground.WritePixels(pixels)
	result.DrawImage(foreground, nil)

	return result
}

func (processor *TileEffectProcessor) applyEffect(
	pixels []byte, frame int, bounds image.Rectangle, effect TileEffect, count int,
) []byte {
	source, err := graphics.NewPixels(TileSize, TileSize, pixels)
	if err != nil {
		panic(fmt.Sprintf("import original tile pixels: %v", err))
	}
	selection := processor.effectSelection(pixels, effect.Filter, bounds)
	var operation graphics.Operation
	switch effect.Algorithm {
	case EffectVerticalRise:
		operation = graphics.Rise(frame, selection)
	case EffectWaveDisplace:
		const amplitude, framePhase, rowPhase = 1.1, .8, .5
		operation = graphics.Wave(amplitude, float64(frame)*framePhase, rowPhase, selection)
	case EffectVerticalScroll:
		operation = graphics.ScrollY(frame % count)
	case EffectTransparencyOverlay:
		return processor.replaceWaterPixels(source, pixels, frame, effect.Background)
	case EffectOutline:
		operation = graphics.Outline(outlineColour(effect.OutlineColor.GetOutlineColor(), effect.Intensity), 0)
	case EffectPulse:
		pulse := pulseMinimumOpacity +
			pulseOpacityAmplitude*(math.Sin(float64(frame)*math.Pi/pulsePhaseDivisor)+1)/pulseWaveRange
		// The selected game appearance deliberately changes alpha alone.
		selection = processor.effectSelection(pixels, effect.Filter, image.Rect(0, 0, 0, 0))
		operation = graphics.AlphaChannelScale(pulse, selection)
	case EffectBrightnessPulse:
		gain := brightnessPulseMinimum +
			brightnessPulseAmplitude*(math.Sin(float64(frame)*math.Pi/pulsePhaseDivisor)+1)/pulseWaveRange
		selection = processor.effectSelection(pixels, effect.Filter, image.Rect(0, 0, 0, 0))
		operation = graphics.RGBGain(gain, selection)
	case EffectBorderWrap:
		panic("border padding is a game presentation operation, not a tile-generation operation")
	}
	result, err := source.Apply(operation)
	if err != nil {
		panic(fmt.Sprintf("generate original tile effect: %v", err))
	}

	return result.Pixels()
}

func (processor *TileEffectProcessor) effectSelection(
	pixels []byte, filter ColorFilter, bounds image.Rectangle,
) graphics.Selection {
	if bounds.Empty() {
		bounds = image.Rect(0, 0, TileSize, TileSize)
	}
	selected := make([]byte, TileSize*TileSize)
	for index := range selected {
		if processor.passesFilter(pixels, index*tilePixelChannels, filter) {
			selected[index] = 1
		}
	}
	mask, err := graphics.NewMask(TileSize, TileSize, selected)
	if err != nil {
		panic(fmt.Sprintf("select original tile palette: %v", err))
	}

	return graphics.Select(bounds, mask)
}

func (processor *TileEffectProcessor) replaceWaterPixels(
	source *graphics.Image, pixels []byte, frame int, background indexes.SpriteIndex,
) []byte {
	frames := processor.effectFrames[background]
	if len(frames) == 0 {
		return pixels
	} // Existing game registry fallback, not an SDK success claim.
	selected := make([]byte, TileSize*TileSize)
	for index := range selected {
		pixel := pixels[index*tilePixelChannels:]
		if isBlueWaterPixel(pixel[0], pixel[1], pixel[2], pixel[3]) {
			selected[index] = 1
		}
	}
	mask, err := graphics.NewMask(TileSize, TileSize, selected)
	if err != nil {
		panic(fmt.Sprintf("select original water pixels: %v", err))
	}
	backgroundPixels := make([]byte, len(pixels))
	frames[cycleIndex(len(frames), int64(frame), 0)].ReadPixels(backgroundPixels)
	replacement, err := graphics.NewPixels(TileSize, TileSize, backgroundPixels)
	if err != nil {
		panic(fmt.Sprintf("import water background: %v", err))
	}
	result, err := source.Apply(graphics.Replace(replacement, graphics.Masked(mask)))
	if err != nil {
		panic(fmt.Sprintf("compose original water tile: %v", err))
	}

	return result.Pixels()
}

const colorTolerance = 10

func isBlueWaterPixel(red, green, blue, alpha uint8) bool {
	const transparentThreshold, grayscaleThreshold = 128, 30
	if alpha < transparentThreshold {
		return true
	}
	if max(red, green, blue)-min(red, green, blue) < grayscaleThreshold {
		return false
	}

	return ucolor.EGABlue.MatchesWithTolerance(red, green, blue, colorTolerance) ||
		ucolor.EGACyan.MatchesWithTolerance(red, green, blue, colorTolerance) ||
		ucolor.EGALightBlue.MatchesWithTolerance(red, green, blue, colorTolerance) ||
		ucolor.EGALightCyan.MatchesWithTolerance(red, green, blue, colorTolerance)
}

func (processor *TileEffectProcessor) passesFilter(pixels []byte, index int, filter ColorFilter) bool {
	red, green, blue := pixels[index], pixels[index+1], pixels[index+2]
	switch filter {
	case FilterAll:
		return true
	case FilterRedOnly:
		return isEGARed(red, green, blue)
	case FilterYellowOnly:
		return isEGAYellow(red, green, blue)
	case FilterFire:
		return isEGARed(red, green, blue) || ucolor.EGALightRed.MatchesWithTolerance(red, green, blue, colorTolerance)
	case FilterBlueFire, FilterProtection:
		return ucolor.EGABlue.MatchesWithTolerance(red, green, blue, colorTolerance) ||
			ucolor.EGALightBlue.MatchesWithTolerance(red, green, blue, colorTolerance)
	case FilterPoison:
		return ucolor.EGAGreen.MatchesWithTolerance(red, green, blue, colorTolerance) ||
			ucolor.EGALightGreen.MatchesWithTolerance(red, green, blue, colorTolerance)
	case FilterSleep:
		return ucolor.EGALightMagenta.MatchesWithTolerance(red, green, blue, colorTolerance) ||
			ucolor.EGAMagenta.MatchesWithTolerance(red, green, blue, colorTolerance)
	default:
		panic("unknown original palette filter")
	}
}

func isEGARed(red, green, blue uint8) bool {
	return ucolor.EGARed.MatchesWithTolerance(red, green, blue, colorTolerance)
}
func isEGAYellow(red, green, blue uint8) bool {
	return ucolor.EGAYellow.MatchesWithTolerance(red, green, blue, colorTolerance)
}
