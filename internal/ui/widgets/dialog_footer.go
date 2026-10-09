package widgets

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/hajimehoshi/ebiten/v2"
)

// DrawKeyboardFooter uses the shared key/description colours and measured
// modifier-label fitting for scene-owned dialogs as well as reusable widgets.
func DrawKeyboardFooter(
	screen *ebiten.Image, content image.Rectangle, guidance string, capacity int, output *text.Output,
) {
	drawModifierFooter(screen, content, guidance, capacity, output)
}

// FitDialogLine fits a status label without splitting UTF-8 or losing its prefix
// to a wrapped one-row Output. An ellipsis marks any omitted tail.
func FitDialogLine(value string, font *text.UltimaFont, width int) string {
	return fitJournalLine(value, font, width)
}
