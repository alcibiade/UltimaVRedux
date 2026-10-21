package text

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func decodeSignTranslation(lines []signLine) [][]ColorSegment {
	var rows [][]ColorSegment
	for _, line := range lines {
		var row []ColorSegment
		structural, content := false, false
		for _, cell := range line.chars {
			value, font := translatedSignCell(cell)
			colour := ucolor.EGAWhite
			if cell.isRunic && !cell.isStructural {
				colour = ucolor.EGAYellow
			}
			structural = structural || cell.isStructural
			content = content || font != FontTypeNormal || strings.TrimSpace(value) != ""
			if len(row) > 0 && row[len(row)-1].FontType == font && row[len(row)-1].Color == colour {
				row[len(row)-1].Text += value
			} else {
				row = append(row, ColorSegment{
					Text: value, Color: colour, FontType: font, FontScale: FontScaleNormal,
				})
			}
		}
		if structural && !content {
			continue // Decorative-only rows must not create large empty margins.
		}
		if len(row) == 0 {
			row = []ColorSegment{{
				Text: "", Color: ucolor.EGAWhite, FontType: FontTypeNormal, FontScale: FontScaleNormal,
			}}
		}
		rows = append(rows, trimSignTranslationRow(row))
	}

	return rows
}

func trimSignTranslationRow(row []ColorSegment) []ColorSegment {
	for len(row) > 0 {
		row[0].Text = strings.TrimLeft(row[0].Text, " ")
		if row[0].Text != "" || len(row) == 1 {
			break
		}
		row = row[1:]
	}
	for len(row) > 0 {
		last := len(row) - 1
		row[last].Text = strings.TrimRight(row[last].Text, " ")
		if row[last].Text != "" || last == 0 {
			break
		}
		row = row[:last]
	}

	return row
}

// RenderTranslatedSign replaces only understood runic lettering in the source
// artwork. The original cells still determine every row and the image bounds.
func (sr *SignRenderer) RenderTranslatedSign(rawText []byte) *SignRenderResult {
	return sr.render(sr.expandAndSplit(rawText), true)
}

func translatedRunicCell(cell signChar) (string, bool) {
	value, font := translatedSignCell(cell)

	return value, cell.isRunic && !cell.isStructural && font == FontTypeNormal
}

func (sr *SignRenderer) drawTranslatedSignLine(dst *ebiten.Image, line signLine, positionY float64) {
	for column := 0; column < len(line.chars); {
		value, translate := translatedRunicCell(line.chars[column])
		positionX := float64(column) * ScaledCharWidth(sr.scale)
		if !translate {
			sr.drawSignChar(dst, line.chars[column], positionX, positionY)
			column++

			continue
		}

		// Artwork, IBM characters and unknown runes are immovable boundaries.
		// Spaces within this runic span can accommodate expanded compound runes.
		start := column
		var inscription strings.Builder
		for translate {
			inscription.WriteString(value)
			column++
			if column == len(line.chars) {
				break
			}
			value, translate = translatedRunicCell(line.chars[column])
		}
		sr.drawTranslatedSignSpan(dst, inscription.String(), column-start, positionX, positionY)
	}
}

func (sr *SignRenderer) drawTranslatedSignSpan(
	dst *ebiten.Image, value string, sourceCells int, positionX, positionY float64,
) {
	inscription := strings.Trim(value, " ")
	if inscription == "" {
		return
	}
	leading := len(value) - len(strings.TrimLeft(value, " "))
	trailing := len(value) - len(strings.TrimRight(value, " "))
	cellWidth := ScaledCharWidth(sr.scale)
	spanWidth := float64(sourceCells) * cellWidth

	// REDUX DEVIATION: the original has no English toggle. Compound runes
	// expand around the source inscription's centre, borrowing only its span's
	// blank padding. Dense spans compress horizontally (at worst 2:1), never
	// vertically, reflowing into other rows or omitting letters. Unexpanded
	// spans retain exact source-cell positions and the full bitmap scale.
	// See ref:SIGN-0008.
	scaleX := min(sr.scale, spanWidth/float64(len(inscription)*BitmapCharWidth))
	width := float64(len(inscription)*BitmapCharWidth) * scaleX
	const halves = 2
	centre := float64(leading+sourceCells-trailing) * cellWidth / halves
	positionX += math.Round(max(0, min(centre-width/halves, spanWidth-width)))
	red, green, blue := ucolor.ToRGBFloats(ucolor.EGAYellow)
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	opts.ColorScale.Scale(red, green, blue, 1)
	for index, glyph := range inscription {
		opts.GeoM.Reset()
		opts.GeoM.Scale(scaleX, sr.scale)
		opts.GeoM.Translate(positionX+float64(index*BitmapCharWidth)*scaleX, positionY)
		dst.DrawImage(sr.fonts.IBM.GetCharImage(int(glyph)), opts)
	}
}

//nolint:cyclop // Keep source-font classification and the five compound rune cases together.
func translatedSignCell(cell signChar) (string, FontType) {
	if cell.isStructural {
		return " ", FontTypeNormal // Never join words separated by source artwork.
	}
	char := references.GetCharIndex(cell.rawByte)
	// Sign output indexes either character set with the low seven bits (ref:SIGN-0001). Treat
	// an out-of-range parser result as a contract violation, never truncate it.
	if char < 0 || char > references.SignMaxCharIndex {
		panic("sign character index outside the seven-bit character set")
	}
	glyph := string(rune(char))
	if !cell.isRunic {
		if char < ' ' || char == references.SignMaxCharIndex {
			return glyph, FontTypeIBM
		}

		return glyph, FontTypeNormal
	}
	if char >= 'A' && char <= 'Z' || char == ' ' {
		return glyph, FontTypeNormal
	}
	switch char {
	case '[':
		return "TH", FontTypeNormal
	case '\\':
		return "EE", FontTypeNormal
	case ']':
		return "NG", FontTypeNormal
	case '_':
		return "ST", FontTypeNormal
	case '@':
		return " ", FontTypeNormal
	default:
		return glyph, FontTypeRunic
	}
}
