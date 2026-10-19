package sprites

import (
	"fmt"

	graphicsbackend "github.com/bradhannah/berad-sdk/modules/berad-graphics/backend/ebitengine"
	"github.com/hajimehoshi/ebiten/v2"
)

// DitherMaskImage retains the game's reusable rejection-mask surface contract.
// Coverage/world geometry stays with the caller; the SDK owns CPU/GPU storage.
type DitherMaskImage struct{ surface *graphicsbackend.Surface }

// Resize returns a cleared borrowed RGBA buffer, preserving same-size allocations.
func (mask *DitherMaskImage) Resize(width, height int) []byte {
	if mask.surface == nil {
		mask.surface = new(graphicsbackend.Surface)
	}
	pixels, err := mask.surface.Resize(width, height)
	if err != nil {
		panic(fmt.Sprintf("resize game rejection mask: %v", err))
	}

	return pixels
}

// Upload commits the caller's rejection pixels.
func (mask *DitherMaskImage) Upload() {
	if err := mask.surface.Upload(); err != nil {
		panic(fmt.Sprintf("upload game rejection mask: %v", err))
	}
}

// Draw preserves nearest filtering and SourceOver rejection over the supplied target.
func (mask *DitherMaskImage) Draw(target *ebiten.Image, transform ebiten.GeoM) {
	options := &ebiten.DrawImageOptions{GeoM: transform, Filter: ebiten.FilterNearest}
	if err := mask.surface.Draw(target, *options); err != nil {
		panic(fmt.Sprintf("draw game rejection mask: %v", err))
	}
}

// Dispose releases the owned surface; this game facade remains ready for a new Resize.
func (mask *DitherMaskImage) Dispose() {
	if mask.surface != nil {
		mask.surface.Close()
		mask.surface = nil
	}
}
