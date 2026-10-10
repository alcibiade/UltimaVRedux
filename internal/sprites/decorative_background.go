package sprites

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	decorativeHashRowSize       = 256
	decorativeCacheFieldBits    = 16
	decorativeCacheFrameBits    = 32
	decorativeCacheAnimationBit = 63
	decorativeMaximumFrame      = 65535
)

// GetAnimatedSpriteWithBackground preserves a decoration's animation pipeline
// while replacing its visual underlay. Variants use a bounded cache; the global
// registry and ordinary frames are never mutated. Call on the rendering thread.
func (s *SpriteSheet) GetAnimatedSpriteWithBackground(
	sprite, background indexes.SpriteIndex, posHash int32, elapsedMs int64,
) *ebiten.Image {
	if sprite < 0 || int(sprite) >= totalSprites || background < 0 || int(background) >= totalSprites {
		panic("decorative composition requires atlas sprite identities")
	}
	pipeline, animated := TileEffectRegistry[sprite]
	originalBackground := pipeline.Background
	if !animated {
		originalBackground = -1
	}
	frameCount, frame := 1, 0
	if animated && s.TileEffectsEnabled {
		s.tileEffectProcessor.Initialize()
		frameCount = s.tileEffectProcessor.getFrameCountForPipeline(pipeline)
		frame = decorativeAnimationFrame(pipeline, posHash, elapsedMs, frameCount)
	} else {
		pipeline = TileEffectPipeline{
			Background: -1, BaseBackground: -1, Bounds: image.Rect(0, 0, 0, 0), Effects: nil,
			IntervalMs: 0, OffsetMode: PositionOffsetDefault, EvenXOffset: 0, FrameCount: 1,
		}
	}
	pipeline.Background = background
	if frame < 0 || frame > decorativeMaximumFrame {
		panic("decorative animation frame exceeds cache key range")
	}
	// Atlas IDs occupy 9 bits; reserve 16 per field for future frames/tiles.
	key := uint64(sprite) | uint64(background)<<decorativeCacheFieldBits | uint64(frame)<<decorativeCacheFrameBits
	if s.TileEffectsEnabled {
		key |= 1 << decorativeCacheAnimationBit
	}
	if image := s.backgroundCache.Get(key); image != nil {
		return image
	}
	foreground := s.GetRawSprite(sprite)
	if originalBackground >= 0 && originalBackground != background {
		foreground = s.decorationWithoutBakedBackground(sprite, originalBackground)
	}
	generated := s.tileEffectProcessor.generateSingleFrame(foreground, frame, pipeline, frameCount)
	bordered := WrapSpriteWith18x18Border(generated)
	s.backgroundCache.Put(key, bordered)

	return bordered
}

// The atlas retains some original ground pixels (notably at the lamp's foot).
// Remove only opaque pixels that match the registered ground at that exact
// source coordinate. This preserves the decoration and never edits atlas/cache
// sources. The work runs on a variant-cache miss, not on every rendered frame.
func (s *SpriteSheet) decorationWithoutBakedBackground(sprite, background indexes.SpriteIndex) *ebiten.Image {
	const pixelChannels = 4
	pixels := make([]byte, TileSize*TileSize*pixelChannels)
	ground := make([]byte, len(pixels))
	s.GetRawSprite(sprite).ReadPixels(pixels)
	s.GetRawSprite(background).ReadPixels(ground)
	for offset := 0; offset+pixelChannels <= len(pixels) && offset+pixelChannels <= len(ground); offset += pixelChannels {
		if ground[offset+3] != 0 && pixels[offset] == ground[offset] && pixels[offset+1] == ground[offset+1] &&
			pixels[offset+2] == ground[offset+2] && pixels[offset+3] == ground[offset+3] {
			clear(pixels[offset : offset+pixelChannels])
		}
	}
	result := ebiten.NewImage(TileSize, TileSize)
	result.WritePixels(pixels)

	return result
}

func decorativeAnimationFrame(pipeline TileEffectPipeline, posHash int32, elapsedMs int64, frameCount int) int {
	interval := pipeline.IntervalMs
	if interval <= 0 {
		interval = defaultEffectIntervalMs
	}
	offset := int64(posHash)
	if pipeline.OffsetMode == PositionOffsetEvenX {
		offset = 0
		if (posHash/decorativeHashRowSize)%2 == 0 {
			offset = int64(pipeline.EvenXOffset)
			if offset == 0 {
				offset = defaultWaterEvenXOffset
			}
		}
	}

	return positiveModulo(elapsedMs/int64(interval)+offset, frameCount)
}
