package sprites

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const (
	outlineOnlyCacheTag     = uint64(0x434f4e544f555200)
	outlineOnlyRGBAChannels = 4
	outlineOnlyAlphaChannel = 3
)

// ApplyDynamicOutlineOnly returns a cached colored contour with black backing
// and a transparent body. It preserves the source dimensions and alignment;
// outlines clip at its bounds, so callers should supply bordered sprite frames.
// The source and result are borrowed, immutable images. Nil or disposed input,
// transparent color or unreadable pixels produce nil, NEVER the original body
// as fallback. Disposed sources are rejected even when their contour is cached.
func (p *TileEffectProcessor) ApplyDynamicOutlineOnly(sprite *ebiten.Image, face color.RGBA) *ebiten.Image {
	if sprite == nil || face.A == 0 {
		return nil
	}
	bounds, readable := outlineOnlySourceBounds(sprite)
	if !readable {
		return nil
	}
	// Match the established tagged dynamic-effect cache pattern. Color and
	// source frame participate in the key, separate from body-bearing effects.
	key := p.calculateDynamicCacheKey(sprite, Effect().AddTint(face).Build()) ^ outlineOnlyCacheTag
	if cached := p.dynamicCache.Get(key); cached != nil {
		return cached
	}
	width, height := bounds.Dx(), bounds.Dy()
	pixels := make([]byte, width*height*outlineOnlyRGBAChannels)
	if err := p.safeReadPixels(sprite, pixels); err != nil {
		return nil
	}
	output := p.outlineOnlyPixels(pixels, width, height, face)
	result := ebiten.NewImage(width, height)
	result.WritePixels(output)
	p.dynamicCache.Put(key, result)

	return result
}

// Bounds can reject a retired image before safeReadPixels gets a chance to read
// it. Ebitengine exposes this condition as a string panic, not a typed error.
// Recover only that lifetime condition, around Bounds alone: initialization,
// allocation, contour generation and cache invariant failures must still panic.
//
//nolint:nonamedreturns // Deferred recovery must return a missing source for the disposed-image condition.
func outlineOnlySourceBounds(sprite *ebiten.Image) (bounds image.Rectangle, readable bool) {
	defer func() {
		if failure := recover(); failure != nil {
			message, isString := failure.(string)
			if !isString || message != "ebiten: the image is already disposed" {
				panic(failure)
			}
			bounds, readable = image.Rect(0, 0, 0, 0), false
		}
	}()

	return sprite.Bounds(), true
}

func (p *TileEffectProcessor) outlineOnlyPixels(input []byte, width, height int, face color.RGBA) []byte {
	outlined := append([]byte(nil), input...)
	p.applyOutlineRGBADynamic(input, outlined, width, height, face)
	output := append([]byte(nil), outlined...)
	p.applyOutlineRGBADynamic(outlined, output, width, height, ucolor.EGABlack.ToRGBA())
	// Remove ALL source pixels, including partial alpha. Neither the colored
	// face nor its backing may replace concealed body pixels with opaque fill.
	for pixel := 0; pixel < len(input); pixel += outlineOnlyRGBAChannels {
		if input[pixel+outlineOnlyAlphaChannel] != 0 {
			clear(output[pixel : pixel+outlineOnlyRGBAChannels])
		}
	}

	return output
}
