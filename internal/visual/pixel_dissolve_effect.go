// Package visual adapts presentation effects to game-owned targets and clocks.
package visual

import (
	"fmt"
	"image"

	graphics "github.com/bradhannah/berad-sdk/modules/berad-graphics"
	graphicsbackend "github.com/bradhannah/berad-sdk/modules/berad-graphics/backend/ebitengine"
	"github.com/hajimehoshi/ebiten/v2"
)

type randomShuffler interface{ Intn(n int) int }

// DissolveMode retains the game's reveal/conceal presentation vocabulary.
type DissolveMode int

// The game selects mode and presentation RNG; the SDK owns ordering/coverage.
const (
	DissolveModeReveal DissolveMode = iota
	DissolveModeHide
)

// PixelDissolveEffect adapts target scaling and highlight projection to an SDK mask.
type PixelDissolveEffect struct {
	targetWidth, targetHeight, maskWidth, maskHeight int
	pattern                                          *graphics.Dissolve
	surface                                          graphicsbackend.Surface
	pixelData                                        []byte
	scaledMask                                       *ebiten.Image
	maskDirty                                        bool
}

// NewPixelDissolveEffect uses explicit game-chosen dimensions and presentation randomness.
func NewPixelDissolveEffect(
	targetWidth, targetHeight, maskWidth, maskHeight int, mode DissolveMode, random randomShuffler,
) *PixelDissolveEffect {
	if mode != DissolveModeReveal && mode != DissolveModeHide {
		panic("unknown game dissolve mode")
	}
	if targetWidth <= 0 || targetHeight <= 0 {
		panic("invalid game dissolve target")
	}
	pattern, err := graphics.NewDissolve(maskWidth, maskHeight, mode == DissolveModeHide, random)
	if err != nil {
		panic(fmt.Sprintf("configure game dissolve: %v", err))
	}
	effect := new(PixelDissolveEffect)
	effect.targetWidth, effect.targetHeight = targetWidth, targetHeight
	effect.maskWidth, effect.maskHeight = maskWidth, maskHeight
	effect.pattern = pattern
	effect.pixelData, err = effect.surface.Resize(maskWidth, maskHeight)
	if err != nil {
		panic(fmt.Sprintf("allocate game dissolve: %v", err))
	}
	if maskWidth != targetWidth || maskHeight != targetHeight {
		effect.scaledMask = ebiten.NewImage(targetWidth, targetHeight)
	}
	effect.Reset()

	return effect
}

// UpdateProgress delegates reversible ordering and changes only the affected buffer cells.
func (effect *PixelDissolveEffect) UpdateProgress(progress float64) bool {
	complete, changed, err := effect.pattern.UpdatePixels(progress, effect.pixelData)
	if err != nil {
		panic(fmt.Sprintf("update game dissolve: %v", err))
	}
	effect.maskDirty = effect.maskDirty || changed

	return complete
}

// GetMask borrows owner-lifetime artwork at the game-selected target resolution.
func (effect *PixelDissolveEffect) GetMask() *ebiten.Image {
	if effect.maskDirty {
		if err := effect.surface.Upload(); err != nil {
			panic(fmt.Sprintf("upload game dissolve: %v", err))
		}
		effect.maskDirty = false
	}
	if effect.scaledMask == nil {
		return effect.surface.Image()
	}
	effect.scaledMask.Clear()
	options := new(ebiten.DrawImageOptions)
	options.GeoM.Scale(float64(effect.targetWidth)/float64(effect.maskWidth),
		float64(effect.targetHeight)/float64(effect.maskHeight))
	options.Filter = ebiten.FilterNearest
	if err := effect.surface.Draw(effect.scaledMask, *options); err != nil {
		panic(fmt.Sprintf("scale game dissolve: %v", err))
	}

	return effect.scaledMask
}

// Reset restores the original ordering without consuming any more random values.
func (effect *PixelDissolveEffect) Reset() {
	effect.pattern.Reset()
	effect.pattern.WritePixels(effect.pixelData)
	if err := effect.surface.Upload(); err != nil {
		panic(fmt.Sprintf("reset game dissolve: %v", err))
	}
	effect.maskDirty = false
}

// VisitLeadingPixels preserves the game's integer highlight projection and leaves
// source/body clipping with the owning renderer. Querying never advances the effect.
func (effect *PixelDissolveEffect) VisitLeadingPixels(fraction float64, visit func(image.Rectangle)) {
	effect.pattern.VisitLeading(fraction, func(point image.Point) {
		visit(image.Rect(point.X*effect.targetWidth/effect.maskWidth, point.Y*effect.targetHeight/effect.maskHeight,
			(point.X+1)*effect.targetWidth/effect.maskWidth, (point.Y+1)*effect.targetHeight/effect.maskHeight))
	})
}

// Dispose closes only this effect's native/scaled masks, leaving host targets alive.
func (effect *PixelDissolveEffect) Dispose() {
	effect.surface.Close()
	if effect.scaledMask != nil {
		effect.scaledMask.Deallocate()
		effect.scaledMask = nil
	}
	effect.pixelData = nil
}
