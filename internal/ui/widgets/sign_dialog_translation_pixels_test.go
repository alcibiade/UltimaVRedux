package widgets

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// These readbacks run inside the widgets package's existing native game harness.
// Compare against the public source bitmap, including every transparent pixel.
// This checks actual English letter shapes, scale, colour and cell registration.
func requireTranslatedBitmapGlyph(t *testing.T, pixels *image.RGBA, glyph byte, position image.Point, scaleX, scaleY int) {
	t.Helper()
	glyphImage := text.GetBitmapFonts().IBM.GetCharImage(int(glyph))
	require.NotNil(t, glyphImage)
	font := atlasMarkerPixels(glyphImage)
	origin := font.Bounds().Min
	for y := range text.BitmapCharHeight * scaleY {
		for x := range text.BitmapCharWidth * scaleX {
			source := font.RGBAAt(origin.X+x/scaleX, origin.Y+y/scaleY)
			want := color.RGBA{R: source.R, G: source.G, B: uint8(uint16(source.B) * uint16(ucolor.EGAYellow.ToRGBA().B) / 255), A: source.A}
			require.Equal(t, want, pixels.RGBAAt(position.X+x, position.Y+y), "glyph %c at (%d,%d)", glyph, x, y)
		}
	}
}

func TestSignTranslationEnglishBitmapAlphabetAndSpaceKeepSourceCells(t *testing.T) {
	raw := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ@")
	for _, scale := range []int{1, 3, 5, 10} {
		renderer := text.NewSignRenderer(float64(scale))
		result := renderer.RenderTranslatedSign(raw)
		pixels := atlasMarkerPixels(result.Image)
		result.Image.Deallocate()
		width, height := renderer.CalculateSignDimensions(raw)
		require.Equal(t, image.Rect(0, 0, width, height), pixels.Bounds())
		for index, glyph := range []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ ") {
			position := image.Pt(index%signCanvasColumns*text.BitmapCharWidth*scale, index/signCanvasColumns*text.BitmapCharHeight*scale)
			requireTranslatedBitmapGlyph(t, pixels, glyph, position, scale, scale)
		}
	}
}

func TestSignTranslationCompoundsStayInsideTheirSourceSpan(t *testing.T) {
	for _, test := range []struct {
		name       string
		raw        string
		want       string
		startCell  int
		horizontal int
	}{
		// Full-size English borrows the runic inscription's blank padding.
		{"padded", "g  [\\]_@ABC   g", "THEENGST ABC", 1, 4},
		// All four compounds remain visible without changing line or art height.
		{"dense", "g[\\]_g", "THEENGST", 1, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			renderer := text.NewSignRenderer(4)
			original := renderer.RenderSign([]byte(test.raw))
			defer original.Image.Deallocate()
			translated := renderer.RenderTranslatedSign([]byte(test.raw))
			defer translated.Image.Deallocate()
			require.Equal(t, original.Width, translated.Width)
			require.Equal(t, original.Height, translated.Height)
			require.Equal(t, test.want+"\n", signTranslationRowsText(translated.TranslationRows))
			pixels := atlasMarkerPixels(translated.Image)
			for index, glyph := range []byte(test.want) {
				position := image.Pt(test.startCell*text.BitmapCharWidth*4+index*text.BitmapCharWidth*test.horizontal, 0)
				requireTranslatedBitmapGlyph(t, pixels, glyph, position, test.horizontal, 4)
			}
		})
	}
}

func TestSignTranslationMixedSourceArtworkIBMAndUnknownRunesArePixelExact(t *testing.T) {
	// Literal and macro borders, a no-pause cursor, both newline encodings,
	// unknown runes, and IBM lookalikes that must never be translated.
	raw := []byte{0x2c, 'g', '[', '?', '\\', 'g' | 0x80, '[' | 0x80, 1 | 0x80,
		'_', 'g', 0x8a, ':', 'l', 'n', 'l', ';', '\n', 'A', 0x0d, 'B', 0x26, 0x27, '\n', 0x2d}
	immutable := append([]byte(nil), raw...)
	// Authored cell coordinates, independent of the renderer's private parser.
	fixedCells := []image.Rectangle{
		image.Rect(0, 0, 16, 1), // Macro top border.
		image.Rect(0, 1, 1, 2),  // Left border.
		image.Rect(2, 1, 3, 2),  // Unknown runic symbol.
		image.Rect(4, 1, 7, 2),  // IBM lookalikes and control glyph.
		image.Rect(8, 1, 9, 2),  // Right border.
		image.Rect(0, 2, 5, 3),  // Literal structural glyphs.
		image.Rect(2, 3, 4, 4),  // Original horizontal-line aliases after the pause.
		image.Rect(0, 4, 16, 5), // Macro bottom border.
	}
	for _, scale := range []int{3, 5, 6, 10} {
		renderer := text.NewSignRenderer(float64(scale))
		original := renderer.RenderSign(raw)
		translated := renderer.RenderTranslatedSign(raw)
		before, after := atlasMarkerPixels(original.Image), atlasMarkerPixels(translated.Image)
		original.Image.Deallocate()
		translated.Image.Deallocate()
		require.Equal(t, before.Bounds(), after.Bounds())
		require.Equal(t, image.Rect(0, 0, 16*text.BitmapCharWidth*scale, 5*text.BitmapCharHeight*scale), after.Bounds())
		require.Equal(t, text.DecodeSignTranslation(raw), translated.TranslationRows)
		for _, cells := range fixedCells {
			for y := cells.Min.Y * text.BitmapCharHeight * scale; y < cells.Max.Y*text.BitmapCharHeight*scale; y++ {
				for x := cells.Min.X * text.BitmapCharWidth * scale; x < cells.Max.X*text.BitmapCharWidth*scale; x++ {
					require.Equal(t, before.RGBAAt(x, y), after.RGBAAt(x, y), "fixed region %v at (%d,%d)", cells, x, y)
				}
			}
		}
		// Every changed pixel must belong to yellow inscription ink, never art.
		for y := range before.Bounds().Dy() {
			for x := range before.Bounds().Dx() {
				old, next := before.RGBAAt(x, y), after.RGBAAt(x, y)
				if old != next {
					require.Contains(t, []color.RGBA{{}, ucolor.EGAYellow.ToRGBA()}, old)
					require.Contains(t, []color.RGBA{{}, ucolor.EGAYellow.ToRGBA()}, next)
				}
			}
		}
		require.Equal(t, immutable, raw)
	}
}
