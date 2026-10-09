package mainscreen

import (
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

const (
	mainScreenReferenceWidth  = 1920
	mainScreenMaxCharsPerLine = 16
)

func newMainScreenOutput(
	screenWidth int,
	size widgets.StandardFontSize,
	maxLines int,
) (*text.UltimaFont, *text.Output) {
	scale := float64(screenWidth) / mainScreenReferenceWidth
	font := text.NewUltimaFont(float64(size) * scale)
	lineSpacing := float64(size) * widgets.MainScreenLineSpacingMultiplier * scale

	return font, text.NewOutput(font, lineSpacing, maxLines, mainScreenMaxCharsPerLine)
}
