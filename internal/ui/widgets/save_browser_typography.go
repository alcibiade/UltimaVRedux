package widgets

import (
	"image"
	"math"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/text"
)

// Center visible glyphs rather than the font's asymmetric ascent/descent box.
// Output uses the same top-origin layout when rendering its cached text.
func centeredGlyphOutputY(bounds image.Rectangle, output *text.Output, value string) int {
	options := new(etext.LayoutOptions)
	options.LineSpacing = output.GetLineSpacing()
	top, bottom := math.Inf(1), math.Inf(-1)
	for _, glyph := range etext.AppendGlyphs(nil, value, output.Font.TextFace, options) {
		if glyph.Image != nil {
			top = min(top, glyph.Y)
			bottom = max(bottom, glyph.Y+float64(glyph.Image.Bounds().Dy()))
		}
	}
	if math.IsInf(top, 1) {
		return bounds.Min.Y
	}

	return bounds.Min.Y + int(math.Round((float64(bounds.Dy())-bottom-top)/saveSlotBrowserCenterDivisor))
}
