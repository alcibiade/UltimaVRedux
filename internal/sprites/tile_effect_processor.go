package sprites

import (
	"fmt"
	"image/color"
	"math"

	graphics "github.com/bradhannah/berad-sdk/modules/berad-graphics"
	graphicsbackend "github.com/bradhannah/berad-sdk/modules/berad-graphics/backend/ebitengine"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// TileEffectProcessor owns game frame sets and adapts palettes/presets to SDK mechanisms.
type TileEffectProcessor struct {
	spriteSheet    *SpriteSheet
	effectFrames   map[indexes.SpriteIndex][]*ebiten.Image
	borderedFrames map[indexes.SpriteIndex][]*ebiten.Image
	graphics       *graphicsbackend.Processor
	initialized    bool
}

func newGraphicsProcessor(entries int) *graphicsbackend.Processor {
	limits := graphicsbackend.DefaultLimits()
	limits.MaxEntries = entries
	processor, err := graphicsbackend.NewProcessor(limits)
	if err != nil {
		panic(fmt.Sprintf("configure mandatory graphics processor: %v", err))
	}

	return processor
}

// NewTileEffectProcessor creates the game adapter without native readback.
func NewTileEffectProcessor(sheet *SpriteSheet) *TileEffectProcessor {
	return &TileEffectProcessor{spriteSheet: sheet, effectFrames: make(map[indexes.SpriteIndex][]*ebiten.Image),
		borderedFrames: make(map[indexes.SpriteIndex][]*ebiten.Image),
		graphics:       newGraphicsProcessor(DefaultDynamicCacheSize), initialized: false}
}

// Initialize generates original tile dependencies before dependent mixed-water frames.
func (processor *TileEffectProcessor) Initialize() {
	if processor.initialized {
		return
	}
	for _, dependent := range []bool{false, true} {
		for sprite, pipeline := range TileEffectRegistry {
			if pipelineHasTransparencyOverlay(pipeline) != dependent {
				continue
			}
			count := processor.getFrameCountForPipeline(pipeline)
			processor.effectFrames[sprite] = processor.generateFrames(sprite, pipeline, count)
		}
	}
	for sprite, pipeline := range TileEffectRegistry {
		frames := processor.effectFrames[sprite]
		count := len(frames)
		variants := 1
		if pipeline.OffsetMode == PositionOffsetEvenX {
			variants = 2
		}
		bordered := make([]*ebiten.Image, count*variants)
		for index, frame := range frames {
			bordered[index] = WrapSpriteWith18x18Border(frame)
			if variants > 1 {
				selected := cycleIndex(count, int64(index), int64(waterOffset(pipeline)))
				bordered[count+index] = WrapSpriteWith18x18Border(frames[selected])
			}
		}
		processor.borderedFrames[sprite] = bordered
	}
	processor.initialized = true
}

func pipelineHasTransparencyOverlay(pipeline TileEffectPipeline) bool {
	for _, effect := range pipeline.Effects {
		if effect.Algorithm == EffectTransparencyOverlay {
			return true
		}
	}

	return false
}

func (processor *TileEffectProcessor) getFrameCountForPipeline(pipeline TileEffectPipeline) int {
	if pipeline.FrameCount > 0 {
		return pipeline.FrameCount
	}
	count := defaultFireFrameCount
	for _, effect := range pipeline.Effects {
		count = max(count, effect.Algorithm.PreferredFrameCount())
	}

	return count
}

func cycleIndex(count int, tick, offset int64) int {
	index, err := graphics.CycleIndex(count, tick, offset)
	if err != nil {
		panic(fmt.Sprintf("sample game animation: %v", err))
	}

	return index
}

func waterOffset(pipeline TileEffectPipeline) int {
	if pipeline.EvenXOffset == 0 {
		return defaultWaterEvenXOffset
	}

	return pipeline.EvenXOffset
}

func (processor *TileEffectProcessor) frameIndices(
	sprite indexes.SpriteIndex, positionHash int32, elapsedMs int64,
) (int, int) {
	pipeline, exists := TileEffectRegistry[sprite]
	if !exists {
		return -1, -1
	}
	count := processor.getFrameCountForPipeline(pipeline)
	interval := pipeline.IntervalMs
	if interval <= 0 {
		interval = defaultEffectIntervalMs
	}
	tick := elapsedMs / int64(interval)
	phase := cycleIndex(count, tick, 0)
	if pipeline.OffsetMode == PositionOffsetEvenX {
		const positionRowSize = 256
		if (positionHash/positionRowSize)%2 == 0 {
			return cycleIndex(count, tick, int64(waterOffset(pipeline))), count + phase
		}

		return phase, phase
	}
	index := cycleIndex(count, tick, int64(positionHash))

	return index, index
}

// GetFrameIndex reports the source frame selected by the same actual rendering policy.
func (processor *TileEffectProcessor) GetFrameIndex(
	sprite indexes.SpriteIndex, positionHash int32, elapsedMs int64,
) int {
	index, _ := processor.frameIndices(sprite, positionHash, elapsedMs)

	return index
}

// GetEffectSprite returns borrowed game-owned generated artwork.
func (processor *TileEffectProcessor) GetEffectSprite(
	sprite indexes.SpriteIndex, positionHash int32, elapsedMs int64,
) *ebiten.Image {
	processor.Initialize()
	index, _ := processor.frameIndices(sprite, positionHash, elapsedMs)
	frames := processor.effectFrames[sprite]
	if index < 0 || index >= len(frames) {
		return nil
	}

	return frames[index]
}

// GetBorderedEffectSprite preserves the existing alternate-column bordered slots.
func (processor *TileEffectProcessor) GetBorderedEffectSprite(
	sprite indexes.SpriteIndex, positionHash int32, elapsedMs int64,
) *ebiten.Image {
	processor.Initialize()
	_, index := processor.frameIndices(sprite, positionHash, elapsedMs)
	frames := processor.borderedFrames[sprite]
	if index < 0 || index >= len(frames) {
		return nil
	}

	return frames[index]
}

// HasEffect checks the game registry without forcing native initialization.
func (processor *TileEffectProcessor) HasEffect(sprite indexes.SpriteIndex) bool {
	_, exists := TileEffectRegistry[sprite]

	return exists
}

// ApplyDynamic outlines supplied artwork and delegates generation/cache ownership.
// SpriteSheet and WrapSpriteWith18x18Border prepare padding before this stage.
func (processor *TileEffectProcessor) ApplyDynamic(source *ebiten.Image, pipeline TileEffectPipeline) *ebiten.Image {
	if source == nil {
		return nil
	}
	var operations []graphics.Operation
	for _, effect := range pipeline.Effects {
		switch effect.Algorithm {
		case EffectOutline:
			if effect.OutlineColor != NoOutline && effect.Intensity > 0 {
				paint := outlineColour(effect.OutlineColor.GetOutlineColor(), effect.Intensity)
				operations = append(operations, graphics.Outline(paint, 0))
			}
		case EffectVerticalRise, EffectWaveDisplace, EffectVerticalScroll, EffectTransparencyOverlay,
			EffectPulse, EffectBrightnessPulse:
			panic("generated tile operation used as a dynamic game preset")
		}
	}
	if len(operations) == 0 {
		return source
	}

	return processor.derive(source, operations...)
}

func (processor *TileEffectProcessor) derive(source *ebiten.Image, operations ...graphics.Operation) *ebiten.Image {
	if processor.graphics == nil {
		panic("missing mandatory graphics processor")
	}
	result, err := processor.graphics.Derive(source, operations...)
	if err != nil {
		panic(fmt.Sprintf("derive game sprite: %v", err))
	}

	return result
}

// ApplyDynamicSimple applies the selected game outline colour.
func (processor *TileEffectProcessor) ApplyDynamicSimple(source *ebiten.Image, colour OutlineColorType) *ebiten.Image {
	return processor.ApplyDynamicSimpleWithIntensity(source, colour, 1)
}

// ApplyDynamicSimpleWithIntensity uses the supplied, game-quantized outline intensity.
func (processor *TileEffectProcessor) ApplyDynamicSimpleWithIntensity(
	source *ebiten.Image, colour OutlineColorType, intensity float32,
) *ebiten.Image {
	if source == nil || colour == NoOutline {
		return source
	}

	return processor.ApplyDynamic(source, Effect().AddOutline(colour, WithIntensity(intensity)).Build())
}

// ApplyDynamicMarchingOutline preserves black/white phase-selected game contours.
func (processor *TileEffectProcessor) ApplyDynamicMarchingOutline(source *ebiten.Image, phase int) *ebiten.Image {
	if source == nil {
		return nil
	}

	return processor.derive(source, graphics.CheckerOutline(ucolor.EGAWhite.ToRGBA(), ucolor.EGABlack.ToRGBA(), phase))
}

// ApplyDynamicOutlineRGBA returns an owned copy for the existing portrait callers.
func (processor *TileEffectProcessor) ApplyDynamicOutlineRGBA(
	source *ebiten.Image, paint color.Color, intensity float32,
) *ebiten.Image {
	if source == nil || paint == nil {
		return source
	}
	rgba, ok := color.RGBAModel.Convert(paint).(color.RGBA)
	if !ok {
		panic("RGBA colour model returned an incompatible colour")
	}
	derived := processor.derive(source, graphics.Outline(outlineColour(rgba, intensity), 0))
	var padding graphics.Insets
	owned, err := graphicsbackend.CopyRegion(derived, derived.Bounds(), padding)
	if err != nil {
		panic(fmt.Sprintf("copy portrait outline: %v", err))
	}

	return owned
}

func outlineColour(paint color.RGBA, intensity float32) color.RGBA {
	if math.IsNaN(float64(intensity)) || math.IsInf(float64(intensity), 0) || intensity < 0 || intensity > 1 {
		panic("invalid game outline intensity")
	}

	return color.RGBA{R: uint8(float32(paint.R) * intensity), G: uint8(float32(paint.G) * intensity),
		B: uint8(float32(paint.B) * intensity), A: uint8(float32(paint.A) * intensity)}
}
