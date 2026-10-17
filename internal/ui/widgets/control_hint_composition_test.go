package widgets

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestMerchantFooterColonBracketAndPlainStatusSemantics(t *testing.T) {
	for _, source := range []string{
		"Left/Right: Quantity  A: Sell All\nEnter: Confirm  Esc: Back",
		"[Enter] Confirm  Esc: Back", "Enter or Esc: Leave",
		"Processing: please wait", "Use [Enter\nOr Esc]",
	} {
		var textLines []string
		for _, line := range parseMerchantFooterGuidance(source) {
			var plain strings.Builder
			for _, segment := range line {
				plain.WriteString(segment.Text)
				if strings.Contains(segment.Text, "Quantity") || strings.Contains(segment.Text, "Processing") ||
					strings.Contains(segment.Text, "Confirm") || strings.Contains(segment.Text, "Sell All") {
					require.Equal(t, DialogControlHintColor, segment.Color)
				}
				if strings.TrimSpace(segment.Text) == "Enter" || segment.Text == "[Enter]" || segment.Text == "Left/Right" {
					require.Equal(t, DialogControlKeyColor, segment.Color)
				}
			}
			textLines = append(textLines, plain.String())
		}
		require.Equal(t, source, strings.Join(textLines, "\n"))
	}
}

func TestArmsAndAcknowledgementActualOutputsUseControlColours(t *testing.T) {
	setResponsiveDialogTestResolution(t, 1280, 720)
	quote := game_state.ArmsSellQuote{Equipment: references.Dagger, BasePrice: 50, UnitOffer: 40,
		Gold: 500, QuantityBefore: 5, MaxQuantity: 3, Sellable: true}
	dialog := newTestArmsTradeDialog(newWidgetTestKeyboard(0), ArmsTradeIntent{Mode: ArmsShopSell, SellQuote: quote})
	dialog.populateFooter()
	width, _ := dialog.footerOutput.Font.MeasureString(strings.Split(dialog.GetFooterGuidance(), "\n")[0])
	t.Logf("footer first line width %.1f, pane %d, measured characters %d", width,
		dialog.frame.footerText.Dx(), dialog.footerOutput.Font.CharsForWidth(dialog.frame.footerText.Dx()))
	_, keys := dialog.footerOutput.FirstPanelLineWithColor(0, dialog.footerOutput.GetPanelContentLineCount(), DialogControlKeyColor)
	_, hints := dialog.footerOutput.FirstPanelLineWithColor(0, dialog.footerOutput.GetPanelContentLineCount(), DialogControlHintColor)
	require.True(t, keys)
	require.True(t, hints)
	require.Contains(t, strings.Join(strings.Fields(dialog.footerOutput.GetOutputStr(false)), " "), "Sell All")
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Deallocate)
	dialog.Draw(screen)
	keyPixels, hintPixels := 0, 0
	for y := dialog.frame.footerText.Min.Y; y < dialog.frame.footerText.Max.Y; y++ {
		for x := dialog.frame.footerText.Min.X; x < dialog.frame.footerText.Max.X; x++ {
			red, green, blue, _ := screen.At(x, y).RGBA()
			if red == 65535 && green == red && blue == red {
				keyPixels++
			}
			if red == 43690 && green == red && blue == red {
				hintPixels++
			}
		}
	}
	require.Positive(t, keyPixels)
	require.Positive(t, hintPixels)
	message := NewMessageDialog("Done").WithEffects(false).WithAutoSizeDefault().WithSeparatePromptArea().Build()
	for _, hint := range []string{DialogCloseHint, DialogAnyKeyHint} {
		message.SetAcknowledgementPrompt(hint)
		require.Equal(t, hint, message.promptOutput.GetOutputStr(false))
		_, found := message.promptOutput.FirstPanelLineWithColor(0, message.promptOutput.GetPanelContentLineCount(), DialogControlKeyColor)
		require.True(t, found)
		require.False(t, message.enableInputDismissal)
	}
	message.ClearPrompt()
	require.Empty(t, message.promptOutput.GetOutputStr(false))
}
