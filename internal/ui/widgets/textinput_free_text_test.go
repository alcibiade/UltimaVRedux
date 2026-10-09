package widgets

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestTextInputFreeTextKeyboardPreservesSpacesAndEnter(t *testing.T) {
	held := make(map[ebiten.Key]bool)
	keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
	var submitted []string
	var widget *TextInput
	widget = NewTextInput(sprites.PercentBasedPlacement{}, 20, 45, nil, TextInputCallbacks{
		OnEnter:               func() { submitted = append(submitted, widget.GetText()); widget.ClearText() },
		AmbiguousAutoComplete: func(string) { t.Fatal("free text must not autocomplete") },
	}, keyboard)
	widget.SetFreeTextMode()
	press := func(key ebiten.Key) {
		held[key] = true
		widget.Update()
		delete(held, key)
		keyboard.ClearFrameConsumption()
		widget.Update()
	}
	press(ebiten.KeyTab)
	for _, key := range []ebiten.Key{ebiten.KeyY, ebiten.KeyO, ebiten.KeyU, ebiten.KeyR, ebiten.KeySpace, ebiten.KeyN, ebiten.KeyA, ebiten.KeyM, ebiten.KeyE} {
		press(key)
	}
	press(ebiten.KeyNumpadEnter)
	press(ebiten.KeyEnter)
	require.Equal(t, []string{"YOUR NAME", ""}, submitted)
}

func TestRegexChoiceReplacesValidInputAndRetainsItOnInvalidKey(t *testing.T) {
	for _, test := range []struct {
		pattern string
		keys    []ebiten.Key
		want    string
	}{
		{"(?i)^[yn]?$", []ebiten.Key{ebiten.KeyY, ebiten.KeyN, ebiten.KeyX}, "N"},
		{"(?i)^[yn]?$", []ebiten.Key{ebiten.KeyN, ebiten.KeyY}, "Y"},
		{"^[1-9]$", []ebiten.Key{ebiten.Key1, ebiten.Key9, ebiten.Key0}, "9"},
	} {
		t.Run(test.pattern+test.want, func(t *testing.T) {
			keyboard := newWidgetTestKeyboard(-1)
			widget := NewTextInput(sprites.PercentBasedPlacement{}, 20, 1, nil, TextInputCallbacks{}, keyboard)
			require.NoError(t, widget.SetInputValidationRegex(test.pattern))
			widget.SetAutoReplaceMode(true)
			for _, key := range test.keys {
				keyboard.SimulateKeyPress(key)
				widget.Update()
			}
			require.Equal(t, test.want, widget.GetText())
		})
	}
}
