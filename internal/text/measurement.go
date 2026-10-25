package text

import (
	"strings"
	"unicode/utf8"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Measurement holds the measured dimensions of text content
type Measurement struct {
	Lines        int     // Number of lines (after wrapping/newlines)
	MaxLineWidth float64 // Width of longest line in pixels
	MaxLineChars int     // Character count of longest line
	TotalHeight  float64 // Total height including line spacing
}

// MeasureText analyzes text content and returns dimension requirements.
// If maxCharsPerLine > 0, text wraps exactly as an Output with that width draws it.
// If maxCharsPerLine == 0, no wrapping is applied (only explicit newlines).
func MeasureText(text string, font *UltimaFont, lineSpacing float64, maxCharsPerLine int) Measurement {
	// Split by explicit newlines first
	explicitLines := strings.Split(text, "\n")
	if len(explicitLines) > 0 && explicitLines[len(explicitLines)-1] == "" {
		explicitLines = explicitLines[:len(explicitLines)-1]
	}

	var allLines []string
	var wrapper Output
	wrapper.maxCharsPerLine = maxCharsPerLine
	for _, line := range explicitLines {
		if maxCharsPerLine <= 0 || utf8.RuneCountInString(line) <= maxCharsPerLine {
			allLines = append(allLines, line)

			continue
		}
		segment := ColorSegment{Text: line, Color: nil, FontType: FontTypeNormal, FontScale: FontScaleNormal}
		for _, row := range wrapper.wrapStyledLine([]ColorSegment{segment}) {
			var joined strings.Builder
			for _, part := range row.Segments {
				joined.WriteString(part.Text)
			}
			allLines = append(allLines, joined.String())
		}
	}

	// Find longest line by characters and measure max width
	var maxWidth float64

	var maxChars int

	for _, line := range allLines {
		maxChars = max(maxChars, utf8.RuneCountInString(line))

		width, _ := etext.Measure(line, font.TextFace, 0)
		if width > maxWidth {
			maxWidth = width
		}
	}

	return Measurement{
		Lines:        len(allLines),
		MaxLineWidth: maxWidth,
		MaxLineChars: maxChars,
		TotalHeight:  float64(len(allLines)) * lineSpacing,
	}
}

// MeasureTextUnwrapped measures text without any wrapping.
// Only explicit newlines create new lines.
func MeasureTextUnwrapped(text string, font *UltimaFont, lineSpacing float64) Measurement {
	return MeasureText(text, font, lineSpacing, 0)
}

// MeasureLineSegments uses the renderer's font, scale and bitmap-glyph advances.
// Use it to budget a single mixed text/icon row before committing its Output line.
func (o *Output) MeasureLineSegments(segments []ColorSegment) float64 {
	width := 0.0
	for _, segment := range segments {
		width += o.getSegmentWidth(segment)
	}

	return width
}
