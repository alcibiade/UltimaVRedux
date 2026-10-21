package widgets

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// The graphics harness keeps processing OS events between test operations.
// SimulateKeyPress only supplies one event: it does not isolate modifier/held
// state. Ordinary fixtures therefore have no hardware keys; tests of holding,
// release, repeats or chords use NewKeyboardWithKeyStateSource explicitly.
func newWidgetTestKeyboard(delay int) *input.Keyboard {
	return input.NewKeyboardWithKeyStateSource(delay, noWidgetTestKeyHeld)
}

func noWidgetTestKeyHeld(ebiten.Key) bool { return false }

func noWidgetTestInputChars(dst []rune) []rune { return dst }

func TestWidgetUnscriptedShiftReproducesEmptyNumericChoice(t *testing.T) {
	for _, polluted := range []bool{true, false} {
		keyboard := newWidgetTestKeyboard(-1)
		want := "9"
		if polluted {
			keyboard = input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return key == ebiten.KeyShiftLeft })
			want = ""
		}
		widget := NewTextInput(sprites.PercentBasedPlacement{}, 20, 1, nil, TextInputCallbacks{}, keyboard)
		require.NoError(t, widget.SetInputValidationRegex("^[1-9]$"))
		widget.SetAutoReplaceMode(true)
		for _, key := range []ebiten.Key{ebiten.Key1, ebiten.Key9, ebiten.Key0} {
			keyboard.SimulateKeyPress(key)
			widget.Update()
		}
		require.Equal(t, want, widget.GetText(), "hardware Shift transforms scripted digits into punctuation")
	}
}
