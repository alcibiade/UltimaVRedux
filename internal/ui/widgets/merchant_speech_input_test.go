package widgets

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestMerchantSpeechKeepsWholeUtteranceAndPagesRenderedText(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	var message strings.Builder
	for index := range 80 {
		fmt.Fprintf(&message, "Merchant statement %03d.\n", index)
	}
	message.WriteString("The final words.")
	shell := NewMerchantDialogShell(MerchantDialogShellConfig{Exchange: MerchantDialogExchangeView{Text: message.String()}})
	content := strings.Join(strings.Fields(shell.exchangeOutput.GetOutputStr(false)), " ")
	require.Contains(t, content, "Merchant statement 000.")
	require.Contains(t, content, "The final words.")
	require.Greater(t, shell.exchangePageCount(), 1)
	capacity := shell.exchangePageCapacity()
	require.GreaterOrEqual(t, capacity*shell.exchangePageCount(), shell.exchangeOutput.GetPanelContentLineCount())
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Deallocate)
	body := shell.exchangeText
	body.Max.Y -= int(math.Ceil(shell.exchangeHint.GetTotalHeightForLines(merchantShellSpeechHintLines))) + shell.pixelUnit()
	readPage := func() []byte {
		screen.Clear()
		shell.DrawBackground(screen)
		shell.DrawCommonRegions(screen)
		pixels := make([]byte, body.Dx()*body.Dy()*4)
		screen.SubImage(body).(*ebiten.Image).ReadPixels(pixels)
		return pixels
	}
	first := readPage()
	shell.PageExchange(1)
	require.Equal(t, 1, shell.exchangePage)
	require.False(t, bytes.Equal(first, readPage()), "speech itself, not just its page label, must advance")
	shell.Refresh(shell.header, shell.exchange, shell.footer)
	require.Equal(t, 1, shell.exchangePage, "same utterance refresh preserves reading position")
	shell.PageExchange(10000)
	require.Equal(t, shell.exchangePageCount()-1, shell.exchangePage)
	shell.PageExchange(-10000)
	require.Zero(t, shell.exchangePage)
	shell.Refresh(shell.header, MerchantDialogExchangeView{Text: "A new reply."}, shell.footer)
	require.Zero(t, shell.exchangePage)
	require.Empty(t, shell.exchangeHint.GetOutputStr(false))
}

type merchantSpeechTestDialog interface {
	Widget
	GetShell() *MerchantDialogShell
}

func TestAllMerchantFamiliesReserveShiftArrowsForSpeech(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	factories := map[string]func(*input.Keyboard) merchantSpeechTestDialog{
		"arms": func(k *input.Keyboard) merchantSpeechTestDialog { return newTestArmsShopDialog(k, testArmsShopState()) },
		"healer": func(k *input.Keyboard) merchantSpeechTestDialog {
			return newTestHealerShopDialog(k, testHealerShopState())
		},
		"horse": func(k *input.Keyboard) merchantSpeechTestDialog {
			return newTestHorseShopDialog(k, testHorseShopState())
		},
		"ship": func(k *input.Keyboard) merchantSpeechTestDialog { return newTestShipShopDialog(k, testShipShopState()) },
		"inn":  func(k *input.Keyboard) merchantSpeechTestDialog { return newTestInnShopDialog(k, testInnShopState()) },
		"guild": func(k *input.Keyboard) merchantSpeechTestDialog {
			return newTestGuildShopDialog(k, testGuildShopState())
		},
		"reagent": func(k *input.Keyboard) merchantSpeechTestDialog {
			return newTestReagentShopDialog(k, testReagentShopState())
		},
		"tavern": func(k *input.Keyboard) merchantSpeechTestDialog {
			return newTestTavernShopDialog(k, testTavernState(references.TavernMenuMuttonAleRations), nil)
		},
	}
	for name, factory := range factories {
		for _, shift := range []ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyShiftRight} {
			t.Run(name+shift.String(), func(t *testing.T) {
				held := map[ebiten.Key]bool{}
				keyboard := input.NewKeyboardWithKeyStateSource(-1, func(key ebiten.Key) bool { return held[key] })
				dialog := factory(keyboard)
				dialog.Update()
				shell := dialog.GetShell()
				shell.Refresh(shell.header, MerchantDialogExchangeView{Text: strings.Repeat("A long merchant reply. ", 100)}, shell.footer)
				held[shift], held[ebiten.KeyRight] = true, true
				dialog.Update()
				require.Equal(t, 1, shell.exchangePage)
				for range 20 {
					keyboard.ClearFrameConsumption()
					dialog.Update()
				}
				require.Equal(t, 1, shell.exchangePage, "holding the key must not skip unread pages")
				clear(held)
				keyboard.ClearFrameConsumption()
				dialog.Update()
				held[shift], held[ebiten.KeyLeft] = true, true
				dialog.Update()
				require.Zero(t, shell.exchangePage)
				if arms, ok := dialog.(*ArmsShopDialog); ok {
					require.Equal(t, ArmsShopBuy, arms.GetMode())
				}
				if tavern, ok := dialog.(*TavernShopDialog); ok {
					require.Equal(t, TavernShopOrder, tavern.GetMode())
				}
			})
		}
	}
}

func TestMerchantSpeechHintAndViewportFitSupportedSizes(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}, {3840, 2160}} {
		setMerchantShellTestResolution(t, size.X, size.Y)
		shell := NewMerchantDialogShell(MerchantDialogShellConfig{Exchange: MerchantDialogExchangeView{
			Text: strings.Repeat("One complete utterance. ", 200),
		}})
		for _, line := range strings.Split(shell.exchangeHint.GetOutputStr(false), "\n") {
			width, _ := shell.exchangeHint.Font.MeasureString(line)
			require.LessOrEqual(t, width, float64(shell.exchangeText.Dx()))
		}
		require.LessOrEqual(t, shell.exchangeOutput.GetTotalHeightForLines(shell.exchangePageCapacity())+
			shell.exchangeHint.GetTotalHeightForLines(merchantShellSpeechHintLines)+float64(shell.pixelUnit()), float64(shell.exchangeText.Dy()))
		oldBorder := NewOpaqueBorder(sprites.PercentBasedPlacement{
			StartPercentX: 0.14, EndPercentX: 0.86, StartPercentY: 0.10, EndPercentY: 0.90,
		}, BorderWidthStandard, MerchantDialogBackgroundColor)
		oldInterior := oldBorder.GetInteriorBounds()
		oldPadding := max(merchantShellMinimumPadding, min(oldInterior.Dx(), oldInterior.Dy())/merchantShellPaddingDivisor)
		oldUsable := insetRectangle(oldInterior, oldPadding).Dx() - oldPadding
		oldWorkspace := oldUsable - int(math.Round(float64(oldUsable)*0.27))
		require.GreaterOrEqual(t, shell.GetWorkspaceBounds().Dx(), oldWorkspace,
			"wider speech must not squeeze the trade workspace")
	}
}

func TestMerchantSpeechHasRoomForOrdinaryLongReply(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {1920, 1080}, {3840, 2160}} {
		setMerchantShellTestResolution(t, size.X, size.Y)
		message := "Welcome, traveler. I have weapons and armour for thy journey. Take care to choose what thy companions can carry. " +
			"A fine weapon is of little use without strength to wield it. What wouldst thou buy?"
		shell := NewMerchantDialogShell(MerchantDialogShellConfig{Exchange: MerchantDialogExchangeView{Text: message}})
		require.Equal(t, 1, shell.exchangePageCount(), size.String())
		require.Empty(t, shell.exchangeHint.GetOutputStr(false))
		require.Equal(t, message, strings.Join(strings.Fields(shell.exchangeOutput.GetOutputStr(false)), " "))
	}
}
