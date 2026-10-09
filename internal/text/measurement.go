package text

import (
	"strings"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TextMeasurement holds the measured dimensions of text content
type TextMeasurement struct {
	Lines        int     // Number of lines (after wrapping/newlines)
	MaxLineWidth float64 // Width of longest line in pixels
	MaxLineChars int     // Character count of longest line
	TotalHeight  float64 // Total height including line spacing
}

// MeasureText analyzes text content and returns dimension requirements.
// If maxCharsPerLine > 0, text will be wrapped at word boundaries.
// If maxCharsPerLine == 0, no wrapping is applied (only explicit newlines).
func MeasureText(text string, font *UltimaFont, lineSpacing float64, maxCharsPerLine int) TextMeasurement {
	// Split by explicit newlines first
	explicitLines := strings.Split(text, "\n")
	if len(explicitLines) > 0 && explicitLines[len(explicitLines)-1] == "" {
		explicitLines = explicitLines[:len(explicitLines)-1]
	}

	var allLines []string

	for _, line := range explicitLines {
		// Apply wrapping only when content exceeds the available columns.
		if maxCharsPerLine > 0 && len(line) > maxCharsPerLine {
			wrapped := wrapPlainLine(line, maxCharsPerLine)
			allLines = append(allLines, wrapped...)
		} else {
			allLines = append(allLines, line)
		}
	}

	// Find longest line by characters and measure max width
	var maxWidth float64

	var maxChars int

	for _, line := range allLines {
		if len(line) > maxChars {
			maxChars = len(line)
		}

		width, _ := etext.Measure(line, font.TextFace, 0)
		if width > maxWidth {
			maxWidth = width
		}
	}

	return TextMeasurement{
		Lines:        len(allLines),
		MaxLineWidth: maxWidth,
		MaxLineChars: maxChars,
		TotalHeight:  float64(len(allLines)) * lineSpacing,
	}
}

// MeasureTextUnwrapped measures text without any wrapping.
// Only explicit newlines create new lines.
func MeasureTextUnwrapped(text string, font *UltimaFont, lineSpacing float64) TextMeasurement {
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

// wrapPlainLine splits a plain text line at word boundaries to fit within maxChars.
// Returns a slice of wrapped lines.
func wrapPlainLine(text string, maxChars int) []string {
	var result []string

	remaining := text
	for len(remaining) > 0 {
		splitAt := maxChars
		splitOnSpace := false
		if len(remaining) <= maxChars {
			splitAt = len(remaining)
		} else {
			searchStart := 0
			for searchStart < maxChars && remaining[searchStart] == ' ' {
				searchStart++
			}
			for index := searchStart; index < maxChars; index++ {
				if remaining[index] == ' ' {
					splitAt = index
					splitOnSpace = true
				}
			}
		}

		chunk := remaining[:splitAt]
		remaining = remaining[splitAt:]
		if splitOnSpace {
			remaining = remaining[1:]
		}
		remaining = strings.TrimLeft(remaining, " ")
		if splitAt == maxChars && len(remaining) > 0 {
			chunk += "-"
		}
		result = append(result, chunk)
	}

	return result
}
