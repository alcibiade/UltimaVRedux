package text

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sync"

	beradtext "github.com/bradhannah/berad-sdk/modules/berad-text"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/text/language"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

//go:embed assets/runic_font.png
var runicFontPNG []byte

//go:embed assets/ibm_font.png
var ibmFontPNG []byte

// Original bitmap dimensions and byte namespace remain game-owned.
const (
	bitmapColourMax   = 65535
	BitmapCharWidth   = 8
	BitmapCharHeight  = 8
	BitmapCharsPerRow = 16
	BitmapTotalChars  = 128
	bitmapRuneBase    = 0xe000
)

// BitmapFont owns Ultima's byte-to-glyph mapping and compatibility image borrows.
// Generic atlas storage, glyph measurement and rasterization belong to the SDK.
type BitmapFont struct {
	source     *beradtext.Source
	scaled     *beradtext.Font
	charImages [BitmapTotalChars]*ebiten.Image
}

// BitmapFonts contains the game's two original character sets.
type BitmapFonts struct{ Runic, IBM *BitmapFont }

//nolint:gochecknoglobals // The two immutable game assets live for the application.
var loadBitmapFonts = sync.OnceValue(func() *BitmapFonts {
	return &BitmapFonts{Runic: newBitmapFontFromPNG(runicFontPNG), IBM: newBitmapFontFromPNG(ibmFontPNG)}
})

// GetBitmapFonts returns application-lifetime font assets.
func GetBitmapFonts() *BitmapFonts { return loadBitmapFonts() }

func newBitmapFontFromPNG(data []byte) *BitmapFont {
	atlas, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		panic(fmt.Sprintf("decoding embedded game font: %v", err))
	}
	glyphs := make(map[rune]beradtext.Glyph, BitmapTotalChars)
	for index := range BitmapTotalChars {
		x, y := (index%BitmapCharsPerRow)*BitmapCharWidth, (index/BitmapCharsPerRow)*BitmapCharHeight
		glyphs[bitmapRuneBase+rune(index)] = beradtext.Glyph{
			Region: image.Rect(x, y, x+BitmapCharWidth, y+BitmapCharHeight), Advance: BitmapCharWidth,
			BearingX: 0, BearingY: -BitmapCharHeight, Colored: false,
		}
	}
	metrics := beradtext.Metrics{Ascent: BitmapCharHeight, Descent: 0, LineGap: 0}
	source, err := beradtext.NewBitmapSource(atlas, metrics, glyphs)
	if err != nil {
		panic(fmt.Sprintf("configuring embedded game font: %v", err))
	}

	return &BitmapFont{source: source, scaled: nil, charImages: [BitmapTotalChars]*ebiten.Image{}}
}

func (f *BitmapFont) fontForScale(scale float64) *beradtext.Font {
	size := float64(BitmapCharHeight) * scale
	if f.scaled == nil || f.scaled.Size() != size {
		font, err := f.source.NewFont(beradtext.FontOptions{
			Size: size, Language: language.Und, Features: nil, Variations: nil,
		})
		if err != nil {
			panic(fmt.Sprintf("scaling game bitmap font: %v", err))
		}
		f.scaled = font
	}

	return f.scaled
}

func (f *BitmapFont) layout(value string, scale float64) *beradtext.Layout {
	// The original encoding is bytes, not UTF-8. Private-use runes prevent a
	// bitmap control-cell index from becoming an SDK newline or shaping control.
	runes := make([]rune, len(value))
	for i := range len(value) {
		runes[i] = bitmapRuneBase + rune(int(value[i])&references.SignCharIndexMask)
	}

	return newTextLayout(string(runes), f.fontForScale(scale))
}

// GetCharImage returns a game-owned, application-lifetime read-only glyph snapshot.
// Existing icon borrowers must not mutate/dispose it. No SDK cache image escapes.
func (f *BitmapFont) GetCharImage(index int) *ebiten.Image {
	if index < 0 || index >= BitmapTotalChars {
		return nil
	}
	if f.charImages[index] == nil {
		img := ebiten.NewImage(BitmapCharWidth, BitmapCharHeight)
		drawTextLayout(img, f.layout(string([]byte{byte(index)}), 1), &ebiten.DrawImageOptions{}, color.White)
		f.charImages[index] = img
	}

	return f.charImages[index]
}

// DrawChar draws a legacy cell through the SDK and returns its scaled advance.
func (f *BitmapFont) DrawChar(dst *ebiten.Image, index int, x, y, scale float64, r, g, b float32) float64 {
	if index < 0 || index >= BitmapTotalChars {
		return 0
	}

	return f.DrawString(dst, string([]byte{byte(index)}), x, y, scale, r, g, b)
}

// DrawString translates original bytes to SDK glyphs and applies supplied tint.
func (f *BitmapFont) DrawString(
	dst *ebiten.Image, value string, positionX, positionY, scale float64, red, green, blue float32,
) float64 {
	options := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	// Cache native-size glyph pixels, then sample once at final placement. Baking
	// a reduced glyph before a fractional translation changes nearest texel picks.
	options.GeoM.Scale(scale, scale)
	options.GeoM.Translate(positionX, positionY)
	ink := color.RGBA64{
		R: uint16(max(0, min(1, red)) * bitmapColourMax), G: uint16(max(0, min(1, green)) * bitmapColourMax),
		B: uint16(max(0, min(1, blue)) * bitmapColourMax), A: bitmapColourMax,
	}
	layout := f.layout(value, 1)
	drawTextLayout(dst, layout, options, ink)

	return layout.Bounds().Width() * scale
}

// ScaledCharWidth returns the scaled original cell width.
func ScaledCharWidth(scale float64) float64 { return float64(BitmapCharWidth) * scale }

// ScaledCharHeight returns the scaled original cell height.
func ScaledCharHeight(scale float64) float64 { return float64(BitmapCharHeight) * scale }

// StringWidthInChars counts original bytes, not UTF-8 runes.
func StringWidthInChars(value string) int { return len(value) }

// StringWidthScaled measures an original byte sequence in scaled cell pixels.
func StringWidthScaled(value string, scale float64) float64 {
	return float64(len(value)) * ScaledCharWidth(scale)
}
