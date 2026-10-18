package widgets

import (
	stdcolor "image/color"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/text"
)

const moonstoneGlyphBase = 0x30

// appendMoonPhaseGlyph shares only the phase identity and Runic segment, leaving
// layout, scale and semantic foreground selection to each consuming widget.
func appendMoonPhaseGlyph(
	line *text.LineBuilder, phase datetime.MoonPhase, foreground stdcolor.Color,
) *text.LineBuilder {
	if phase < datetime.MoonPhaseNew || phase > datetime.MoonPhaseCrescentWaning {
		panic("invalid moonstone phase")
	}

	return line.RunicWithColor(string(rune(moonstoneGlyphBase+phase)), foreground)
}
