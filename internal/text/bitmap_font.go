// Package text provides text rendering utilities including bitmap fonts for sign display.
package text

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

//go:embed assets/runic_font.png
var runicFontPNG []byte

//go:embed assets/ibm_font.png
var ibmFontPNG []byte

const (
	// BitmapCharWidth is the width of each character in the source font (pixels)
	BitmapCharWidth = 8
	// BitmapCharHeight is the height of each character in the source font (pixels)
	BitmapCharHeight = 8
	// BitmapCharsPerRow is the number of characters per row in the sprite sheet
	BitmapCharsPerRow = 16
	// BitmapTotalChars is the total number of characters in the font
	BitmapTotalChars = 128
)

// BitmapFont represents a loaded bitmap font from a PNG sprite sheet.
// Each character is 8×8 pixels in the source, but can be scaled when rendering.
type BitmapFont struct {
	spriteSheet *ebiten.Image
	charImages  [BitmapTotalChars]*ebiten.Image // Pre-extracted character images
}

// BitmapFonts holds the loaded runic and IBM bitmap fonts for sign rendering.
type BitmapFonts struct {
	Runic *BitmapFont
	IBM   *BitmapFont
}

var loadedBitmapFonts *BitmapFonts

// GetBitmapFonts returns the singleton BitmapFonts instance, loading fonts on first call.
func GetBitmapFonts() *BitmapFonts {
	if loadedBitmapFonts == nil {
		loadedBitmapFonts = &BitmapFonts{
			Runic: newBitmapFontFromPNG(runicFontPNG),
			IBM:   newBitmapFontFromPNG(ibmFontPNG),
		}
	}

	return loadedBitmapFonts
}

// newBitmapFontFromPNG creates a BitmapFont from embedded PNG data.
func newBitmapFontFromPNG(pngData []byte) *BitmapFont {
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		panic(fmt.Sprintf("failed to decode embedded font PNG: %v", err))
	}

	bf := &BitmapFont{
		spriteSheet: ebiten.NewImageFromImage(img),
	}

	// Pre-extract all character sub-images for efficient rendering
	for i := 0; i < BitmapTotalChars; i++ {
		x := (i % BitmapCharsPerRow) * BitmapCharWidth
		y := (i / BitmapCharsPerRow) * BitmapCharHeight
		rect := image.Rect(x, y, x+BitmapCharWidth, y+BitmapCharHeight)
		charImage, ok := bf.spriteSheet.SubImage(rect).(*ebiten.Image)
		if !ok {
			panic("Ebitengine font subimage is not an Ebitengine image")
		}
		bf.charImages[i] = charImage
	}

	return bf
}

// GetCharImage returns the sub-image for a given character index (0-127).
// Returns nil for out-of-range indices.
func (bf *BitmapFont) GetCharImage(charIndex int) *ebiten.Image {
	if charIndex < 0 || charIndex >= BitmapTotalChars {
		return nil
	}

	return bf.charImages[charIndex]
}

// DrawChar draws a single character at the specified position with the given scale.
// The character is drawn with the specified color tint.
// Returns the width of the drawn character (scaled).
func (bf *BitmapFont) DrawChar(dst *ebiten.Image, charIndex int, x, y float64, scale float64, colorR, colorG, colorB float32) float64 {
	charImg := bf.GetCharImage(charIndex)
	if charImg == nil {
		return 0
	}

	opts := &ebiten.DrawImageOptions{}
	opts.Filter = ebiten.FilterNearest
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(x, y)

	// Apply color tint using color matrix
	opts.ColorScale.Scale(colorR, colorG, colorB, 1)

	dst.DrawImage(charImg, opts)

	return float64(BitmapCharWidth) * scale
}

// DrawString draws a string at the specified position with the given scale.
// useRunic controls whether high-bit-clear bytes use runic font (true) or IBM font (false for both).
// Returns the total width of the drawn string.
func (bf *BitmapFont) DrawString(dst *ebiten.Image, text string, x, y float64, scale float64, colorR, colorG, colorB float32) float64 {
	currentX := x
	for i := range len(text) {
		charIndex := int(text[i]) & references.SignCharIndexMask // Mask to 7 bits
		width := bf.DrawChar(dst, charIndex, currentX, y, scale, colorR, colorG, colorB)
		currentX += width
	}

	return currentX - x
}

// ScaledCharWidth returns the width of a character at the given scale.
func ScaledCharWidth(scale float64) float64 {
	return float64(BitmapCharWidth) * scale
}

// ScaledCharHeight returns the height of a character at the given scale.
func ScaledCharHeight(scale float64) float64 {
	return float64(BitmapCharHeight) * scale
}

// StringWidthInChars returns the width of a string in character units.
func StringWidthInChars(text string) int {
	return len(text)
}

// StringWidthScaled returns the pixel width of a string at the given scale.
func StringWidthScaled(text string, scale float64) float64 {
	return float64(len(text)) * ScaledCharWidth(scale)
}
