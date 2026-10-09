package widgets

import (
	"image/color"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// DialogBackgroundStyle defines the background style for dialog borders.
type DialogBackgroundStyle int

const (
	// DialogBackgroundBlack uses opaque black background (for selection dialogs, view dialogs).
	DialogBackgroundBlack DialogBackgroundStyle = iota
	// DialogBackgroundStandard uses semi-transparent DialogBackground color (for message dialogs).
	DialogBackgroundStandard
)

// newDialogBorder creates a border for dialogs with the specified background style.
// Uses BorderWidthStandard for all dialog borders.
func newDialogBorder(placement sprites.PercentBasedPlacement, style DialogBackgroundStyle) *Border {
	var bgColor color.Color

	switch style {
	case DialogBackgroundBlack:
		bgColor = color.Black
	case DialogBackgroundStandard:
		bgColor = pkgcolor.DialogBackground.RGBA()
	default:
		bgColor = color.Black
	}

	return NewBorder(placement, BorderWidthStandard, bgColor)
}

// newButtonBorder creates a border for buttons.
// Uses BorderWidthThick for visual prominence.
func newButtonBorder(placement sprites.PercentBasedPlacement, interiorColor color.Color) *Border {
	return NewBorder(placement, BorderWidthThick, interiorColor)
}
