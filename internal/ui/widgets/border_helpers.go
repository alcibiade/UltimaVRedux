package widgets

import (
	"image/color"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// newDialogBorder creates a standard-width dialog border with an opaque black
// background, as used by selection and view dialogs.
func newDialogBorder(placement sprites.PercentBasedPlacement) *Border {
	return NewBorder(placement, BorderWidthStandard, color.Black)
}
