// Package text provides text rendering utilities including sign rendering for Ultima V.
package text

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// SignRenderer renders sign text using bitmap fonts with runic/normal character support.
type SignRenderer struct {
	fonts *BitmapFonts
	scale float64
}

// NewSignRenderer creates a renderer for the responsive warm sign treatment.
// Scale determines the size multiplier for the 8x8 pixel characters.
func NewSignRenderer(scale float64) *SignRenderer {
	return &SignRenderer{
		fonts: GetBitmapFonts(),
		scale: scale,
	}
}

// SignRenderResult contains the rendered sign image and metadata.
type SignRenderResult struct {
	Image  *ebiten.Image
	Width  int
	Height int

	// TranslationRows is the semantic inscription from the same parsed cells
	// used to draw a translated sign, without its decorative-only rows.
	TranslationRows [][]ColorSegment
}

// RenderSign renders the raw sign text to an image.
// Handles runic/normal font switching, border macros, and line breaks.
func (sr *SignRenderer) RenderSign(rawText []byte) *SignRenderResult {
	return sr.render(sr.expandAndSplit(rawText), false)
}

func (sr *SignRenderer) render(lines []signLine, translated bool) *SignRenderResult {
	// Calculate dimensions
	maxWidth := 0
	for _, line := range lines {
		lineWidth := len(line.chars)
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
	}

	charWidth := int(ScaledCharWidth(sr.scale))
	charHeight := int(ScaledCharHeight(sr.scale))

	imgWidth := maxWidth * charWidth
	imgHeight := len(lines) * charHeight

	if imgWidth == 0 || imgHeight == 0 {
		imgWidth, imgHeight = 1, 1
	}

	// Create image and render
	img := ebiten.NewImage(imgWidth, imgHeight)

	for lineIdx, line := range lines {
		positionY := float64(lineIdx * charHeight)
		if translated {
			sr.drawTranslatedSignLine(img, line, positionY)

			continue
		}
		x := 0.0

		for _, ch := range line.chars {
			sr.drawSignChar(img, ch, x, positionY)
			x += ScaledCharWidth(sr.scale)
		}
	}

	var translationRows [][]ColorSegment
	if translated {
		translationRows = decodeSignTranslation(lines)
	}

	return &SignRenderResult{
		Image:           img,
		Width:           imgWidth,
		Height:          imgHeight,
		TranslationRows: translationRows,
	}
}

func (sr *SignRenderer) drawSignChar(image *ebiten.Image, char signChar, positionX, positionY float64) {
	font := sr.fonts.IBM
	glyphColor := ucolor.EGAWhite
	if char.isRunic {
		font = sr.fonts.Runic
		glyphColor = ucolor.EGAYellow
	}
	if char.isStructural {
		glyphColor = ucolor.EGABrown
	}
	red, green, blue := ucolor.ToRGBFloats(glyphColor)
	// REDUX DEVIATION: Original sign output uses one active foreground for all sign
	// glyphs (ref:SIGN-0008). Redux uses brown structure, yellow runes, and white IBM lettering
	// to strengthen hierarchy in the responsive modal.
	font.DrawChar(
		image,
		references.GetCharIndex(char.rawByte),
		positionX,
		positionY,
		sr.scale,
		red,
		green,
		blue,
	)
}

// signChar represents a single character with its font type.
type signChar struct {
	rawByte      byte
	isRunic      bool
	isStructural bool
}

// signLine represents a line of sign text.
type signLine struct {
	chars []signChar
}

// signLineWidth is the original Ultima V message window width.
// Signs without explicit newlines rely on auto-wrapping at this width.
const signLineWidth = 16

// expandAndSplit expands border macros and splits text into lines.
// Handles two sign formats:
// 1. Explicit newlines (0x8A/0x0A) - lines end at newline character
// 2. Implicit wrapping - exactly 16 characters per line, auto-wraps
func (sr *SignRenderer) expandAndSplit(rawText []byte) []signLine {
	var lines []signLine
	var currentLine signLine
	wrappedAtBoundary := false

	// Helper to add a character and auto-wrap at 16 chars
	addChar := func(ch signChar) {
		if len(currentLine.chars) == 0 {
			wrappedAtBoundary = false
		}
		currentLine.chars = append(currentLine.chars, ch)
		if len(currentLine.chars) >= signLineWidth {
			lines = append(lines, currentLine)
			currentLine = signLine{chars: nil}
			wrappedAtBoundary = true
		}
	}

	for _, rawByte := range rawText {
		// REDUX DEVIATION: Original sign output paused for a key at 0x0D (ref:SIGN-0004), then
		// resumed at the same cursor. Redux shows the complete sign in one
		// user-dismissed popup, so it omits the intermediate pause.
		if rawByte == references.SignPauseMarker {
			continue
		}

		// Handle newline - both runic (0x0A) and normal/IBM (0x8A) encodings
		// 0x8A is 0x0A with high bit set indicating "normal font"
		if rawByte == '\n' || rawByte == 0x8A {
			if wrappedAtBoundary && len(currentLine.chars) == 0 {
				wrappedAtBoundary = false

				continue
			}
			lines = append(lines, currentLine)
			currentLine = signLine{chars: nil}
			wrappedAtBoundary = false

			continue
		}

		// Handle border macros (0x29-0x31)
		// Expand inline, each char counts toward 16-char line limit
		if references.IsBorderMacro(rawByte) {
			macro := references.GetBorderMacro(rawByte)
			for index := range len(macro) {
				addChar(signChar{
					rawByte:      macro[index],
					isRunic:      true,
					isStructural: true,
				})
			}

			continue
		}

		addChar(literalSignChar(rawByte))
	}

	// Add final line if non-empty
	if len(currentLine.chars) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

func literalSignChar(rawByte byte) signChar {
	// The original horizontal-line aliases always select runic structure.
	if rawByte == 0x26 || rawByte == 0x27 {
		return signChar{rawByte: 'l', isRunic: true, isStructural: true}
	}
	isRunic := references.IsRunicChar(rawByte)

	return signChar{
		rawByte: rawByte, isRunic: isRunic,
		isStructural: isRunic && references.IsSignBorderGlyph(rawByte),
	}
}
