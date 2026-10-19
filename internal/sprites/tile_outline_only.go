package sprites

import (
	"image/color"

	graphics "github.com/bradhannah/berad-sdk/modules/berad-graphics"
	"github.com/hajimehoshi/ebiten/v2"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// ApplyDynamicOutlineOnly chooses game colours and preserves concealment: failed
// source admission returns no contour, never body artwork. SDK borrows remain read-only.
func (processor *TileEffectProcessor) ApplyDynamicOutlineOnly(source *ebiten.Image, face color.RGBA) *ebiten.Image {
	if source == nil || face.A == 0 {
		return nil
	}
	if processor.graphics == nil {
		panic("missing mandatory graphics processor")
	}
	result, err := processor.graphics.Derive(source, graphics.Contour(face, ucolor.EGABlack.ToRGBA()))
	if err != nil {
		return nil
	}

	return result
}
