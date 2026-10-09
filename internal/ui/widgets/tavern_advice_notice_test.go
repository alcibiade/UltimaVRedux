package widgets

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestTavernAdviceNoticeRightPaneLayoutAndResize(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	setMerchantShellTestResolution(t, 640, 400)
	shop := newTestTavernShopDialog(keyboard, testTavernState(references.TavernMenuStoutFruitProvisions), nil)
	message := `Sir Simon says, "That, I cannot help thee with."`
	notice := newTestTavernAdviceNotice(keyboard, shop.GetWorkspaceBounds, message)
	for _, size := range [][2]int{{320, 200}, {640, 400}, {1280, 720}, {1920, 1080}, {2560, 1440}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			setMerchantShellTestResolution(t, size[0], size[1])
			bounds := notice.GetBounds()
			pane := shop.GetWorkspaceBounds()
			require.True(t, bounds.In(pane), "notice %v outside right pane %v", bounds, pane)
			require.InDelta(t, pane.Min.X+pane.Max.X, bounds.Min.X+bounds.Max.X, 1)
			require.InDelta(t, pane.Min.Y+pane.Max.Y, bounds.Min.Y+bounds.Max.Y, 1)
			require.Equal(t, GetScaledFont(shop.GetFontSize()).TextFace.Size, notice.bodyOutput.Font.TextFace.Size)
			require.Equal(t, strings.Fields(message), strings.Fields(notice.bodyOutput.GetOutputStr(false)))
			require.Equal(t, strings.Fields(DialogCloseHint), strings.Fields(notice.footerOutput.GetOutputStr(false)))
			require.True(t, notice.body.In(bounds))
			require.True(t, notice.footer.In(bounds))
			require.LessOrEqual(t, notice.body.Max.Y, notice.footer.Min.Y)
			for _, output := range []*text.Output{notice.bodyOutput, notice.footerOutput} {
				for _, line := range strings.Split(output.GetOutputStr(false), "\n") {
					width, _ := output.Font.MeasureString(strings.TrimSpace(line))
					require.LessOrEqual(t, width, float64(notice.body.Dx()))
				}
			}
			notice.ensureLayout()
			require.False(t, notice.IsDismissed(), "reflow retains modal lifetime")
		})
	}
}

func TestTavernAdviceNoticeKeyboardReleaseAndDismissal(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeyEscape} {
		keyboard := newWidgetTestKeyboard(0)
		shop := newTestTavernShopDialog(keyboard, testTavernState(references.TavernMenuStoutFruitProvisions), nil)
		notice := newTestTavernAdviceNotice(keyboard, shop.GetWorkspaceBounds, "No advice.")
		held := true
		notice.isKeyPressed = func(pressed ebiten.Key) bool { return held && pressed == key }
		notice.keysReady = false
		notice.Update()
		require.False(t, notice.IsDismissed())
		held = false
		notice.Update()
		keyboard.SimulateKeyPress(key)
		notice.Update()
		require.True(t, notice.IsDismissed())
		require.False(t, shop.IsDismissed(), "notice must not end the merchant session")
	}
}

func newTestTavernAdviceNotice(keyboard *input.Keyboard, pane func() image.Rectangle, message string) *TavernAdviceNotice {
	notice := NewTavernAdviceNotice(keyboard, pane, message)
	notice.isKeyPressed = keyboard.IsKeyHeld
	notice.keysReady = tavernResultControlsReleased(notice.isKeyPressed)
	return notice
}
