package widgets

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestSaveDescriptionNativeClearAcceptsEitherASCIIShortcutCase(t *testing.T) {
	for _, letter := range []string{"u", "U"} {
		for _, selected := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/selected=%t", letter, selected), func(t *testing.T) {
				held := map[ebiten.Key]bool{ebiten.KeyCapsLock: true, ebiten.KeyControl: true, ebiten.KeyU: true}
				modal, field := newNativeEditorTest(t, strings.Repeat("x", 80), held)
				if selected {
					modal.setFieldTextAndSelection(modal.GetText(), 30, 10)
				}
				field.nextInput = func() { field.commit(letter) }
				modal.Update()
				require.Empty(t, modal.GetText())
				require.Empty(t, modal.GetValidationMessage())
				require.False(t, modal.IsFinished())
				modal.Update()
				require.Empty(t, modal.GetText(), "held shortcut remains a single edit")
			})
		}
	}
}
