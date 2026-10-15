package widgets

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

// nativeTextField is the platform session protocol shared by the save description
// and Tavern subject editors. Selection and composition offsets are UTF-8 bytes.
type nativeTextField interface {
	Focus()
	Blur()
	IsFocused() bool
	HandleInputWithBounds(bounds image.Rectangle) (handled bool, err error)
	Text() string
	TextForRendering() string
	Selection() (startInBytes, endInBytes int)
	CompositionSelection() (startInBytes, endInBytes int, ok bool)
	SetTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int)
}

// Composer requires caller-owned selection/preedit callbacks, not Field's
// session protocol. Keep both editors on the same native focus lifecycle.
//
//nolint:staticcheck // SA1019: Composer is not a drop-in replacement for Field.
func newNativeTextField() *textinput.Field {
	return new(textinput.Field)
}
