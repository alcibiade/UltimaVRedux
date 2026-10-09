package main

import (
	"image"
	"sync"

	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/hajimehoshi/ebiten/v2"
)

const combatFleeGlyphTop = 5

// Both 4x7 outward chevrons fit the padded 18-pixel actor. Moving the old
// negative-X tip inward by one source pixel keeps the whole cue in signature
// frames, where dissolve/masking must apply to the same artwork as the actor.
//
//nolint:gochecknoglobals // Immutable source-pixel artwork, shared for the application lifetime.
var combatFleeGlyph = sync.OnceValue(func() *ebiten.Image {
	const width, height = 18, 7
	art := image.NewRGBA(image.Rect(0, 0, width, height))
	for y, x := range []int{3, 2, 1, 0, 1, 2, 3} {
		art.Set(x, y, ucolor.EGAYellow)
		art.Set(width-1-x, y, ucolor.EGAYellow)
	}

	return ebiten.NewImageFromImage(art)
})

func drawCombatFleeChevrons(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	glyphOptions := *options
	glyphOptions.Filter = ebiten.FilterNearest
	glyphOptions.GeoM.Reset()
	glyphOptions.GeoM.Translate(0, combatFleeGlyphTop)
	glyphOptions.GeoM.Concat(options.GeoM)
	screen.DrawImage(combatFleeGlyph(), &glyphOptions)
}
