package widgets

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestReagentPurchaseDialogDisplaysSuppliedApprovalAndPaidOverflow(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	quote := testReagentShopState().Offers[0]
	dialog := NewReagentPurchaseDialog(ReagentPurchaseDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Quote:    quote,
		Preview:  true,
	})
	approval := quote.ApprovalViews[0]
	body := dialog.GetBodyText()

	assert.Equal(t, uint16(1), dialog.GetBundleCount())
	assert.Equal(t, approval, dialog.GetApprovalView())
	assert.True(t, dialog.IsPreview())
	assert.Contains(t, dialog.headerOutput.GetOutputStr(false), "PREVIEW - Purchase Ginseng")
	assert.Contains(t, body, "Bundles (before theft)")
	assert.Contains(t, body, "10 Ginseng per bundle")
	assert.Contains(t, body, "Requested quantity: 10")
	assert.Contains(t, body, "Inventory: 10 -> 20")
	assert.Contains(t, body, "Maximum receipt: 10")
	assert.Contains(t, body, "Price per bundle: 20 gold")
	assert.Contains(t, body, "Total price: 20 gold")
	assert.Contains(t, body, "Gold: 200 -> 180")
	assert.NotContains(t, body, "PAID OVERFLOW")

	dialog.setBundleCount(quote.MaxBundles)
	maximum := quote.ApprovalViews[len(quote.ApprovalViews)-1]
	body = dialog.GetBodyText()
	assert.Equal(t, maximum, dialog.GetApprovalView())
	assert.True(t, maximum.PaidOverflow)
	assert.Contains(t, body, "Requested quantity: 90")
	assert.Contains(t, body, "Inventory: 10 -> 99")
	assert.Contains(t, body, "Maximum receipt: 89")
	assert.Contains(t, body, "Total price: 180 gold")
	assert.Contains(t, body, "Gold: 200 -> 20")
	assert.Contains(t, compactReagentText(body), "1 reagent will not fit.")
	assert.Contains(t, compactReagentText(body), "Full bundle prices apply.")
}

func TestReagentPurchaseDialogKeyboardBoundsAndSingleStableRequest(t *testing.T) {
	keyboard := newWidgetTestKeyboard(0)
	quote := testReagentShopState().Offers[0]
	dialog := NewReagentPurchaseDialog(ReagentPurchaseDialogConfig{Keyboard: keyboard, Quote: quote})

	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyLeft)
	assert.Equal(t, uint16(1), dialog.GetBundleCount())
	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyDown)
	assert.Equal(t, uint16(2), dialog.GetBundleCount())
	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyUp)
	assert.Equal(t, uint16(1), dialog.GetBundleCount())
	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyRight)
	assert.Equal(t, uint16(2), dialog.GetBundleCount())
	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyM)
	assert.Equal(t, quote.MaxBundles, dialog.GetBundleCount())
	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyRight)
	assert.Equal(t, quote.MaxBundles, dialog.GetBundleCount())

	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyNumpadEnter)
	assert.True(t, dialog.IsDismissed())
	assert.False(t, dialog.IsCancelled())
	request, ok := dialog.ConsumePendingPurchaseRequest()
	require.True(t, ok)
	assert.Equal(t, game_state.ReagentPurchaseRequest{
		Reagent:     references.Ginseng,
		BundleCount: quote.MaxBundles,
	}, request)
	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyEnter)
	_, duplicate := dialog.ConsumePendingPurchaseRequest()
	assert.False(t, duplicate)
}

func TestReagentPurchaseDialogEscapeAndStrictQuoteContracts(t *testing.T) {
	quote := testReagentShopState().Offers[0]
	valid := ReagentPurchaseDialogConfig{Keyboard: newWidgetTestKeyboard(0), Quote: quote}
	tests := []struct {
		name   string
		mutate func(*ReagentPurchaseDialogConfig)
	}{
		{name: "keyboard", mutate: func(config *ReagentPurchaseDialogConfig) { config.Keyboard = nil }},
		{name: "blocked", mutate: func(config *ReagentPurchaseDialogConfig) {
			config.Quote = testReagentShopState().Offers[1]
		}},
		{name: "approval", mutate: func(config *ReagentPurchaseDialogConfig) {
			config.Quote.ApprovalViews[0].ActualReceipt++
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			config.Quote = cloneReagentOfferQuote(valid.Quote)
			test.mutate(&config)
			require.Panics(t, func() { NewReagentPurchaseDialog(config) })
		})
	}

	keyboard := newWidgetTestKeyboard(0)
	dialog := NewReagentPurchaseDialog(ReagentPurchaseDialogConfig{Keyboard: keyboard, Quote: quote})
	pressReagentPurchaseKey(dialog, keyboard, ebiten.KeyEscape)
	assert.True(t, dialog.IsDismissed())
	assert.True(t, dialog.IsCancelled())
	assert.False(t, dialog.HasPendingPurchaseRequest())
}

func TestReagentPurchaseDialogIsCenteredOpaqueAndResolutionSafe(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	shop := newTestReagentShopDialog(newWidgetTestKeyboard(0), testReagentShopState())
	dialog := NewReagentPurchaseDialog(ReagentPurchaseDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Quote:    testReagentShopState().Offers[0],
	})
	dialog.setBundleCount(2)
	initial := dialog.GetBounds()
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)

	assert.True(t, initial.In(shop.GetShell().GetBounds()))
	assert.InDelta(t, 640, initial.Min.X+initial.Dx()/2, 1)
	assert.InDelta(t, 360, initial.Min.Y+initial.Dy()/2, 1)
	assert.Equal(t,
		"[Left/Right] Quantity   [M] Max\n[Enter] Confirm   [Esc] Back",
		dialog.GetFooterGuidance(),
	)
	requireKeyedGuidance(t, dialog.GetFooterGuidance())
	assert.Equal(t, reagentPurchaseFooterLines, dialog.footerOutput.GetPanelContentLineCount())
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlKeyColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlHintColor)
	headerBackground := image.Pt(
		dialog.frame.headerBounds.Min.X+dialog.frame.pixelUnit()*4,
		dialog.frame.headerBounds.Min.Y,
	)
	requireColorRGBA(t, DialogRuleColor, screen.At(headerBackground.X, headerBackground.Y))

	setResponsiveDialogTestResolution(t, 1920, 1080)
	dialog.ensureLayout()
	assert.Greater(t, dialog.GetBounds().Dx(), initial.Dx())
	assert.Greater(t, dialog.GetBounds().Dy(), initial.Dy())
	assert.Equal(t, uint16(2), dialog.GetBundleCount())
	assert.Contains(t, dialog.GetBodyText(), "Requested quantity: 20")
	assert.NotContains(t, strings.ToLower(dialog.GetFooterGuidance()), "mouse")
}

func pressReagentPurchaseKey(dialog *ReagentPurchaseDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}

func TestReagentPurchaseDialogPhysicalMaximumRetainsPaidOverflowView(t *testing.T) {
	keys := newPhysicalQuantityKeys()
	quote := testReagentShopState().Offers[0]
	dialog := NewReagentPurchaseDialog(ReagentPurchaseDialogConfig{Keyboard: keys.keyboard, Quote: quote})
	keys.held[ebiten.KeyM] = true
	keys.tick(dialog)
	maximum := quote.ApprovalViews[len(quote.ApprovalViews)-1]
	require.Equal(t, maximum, dialog.GetApprovalView())
	require.True(t, maximum.PaidOverflow)
	keys.release(dialog)
	keys.held[ebiten.KeyRight] = true
	for range 60 {
		keys.tick(dialog)
	}
	require.Equal(t, maximum, dialog.GetApprovalView(), "held input cannot create an unsupplied approval view")
	keys.release(dialog)
	keys.held[ebiten.KeyEnter] = true
	keys.tick(dialog)
	request, ok := dialog.ConsumePendingPurchaseRequest()
	require.True(t, ok)
	require.Equal(t, game_state.ReagentPurchaseRequest{Reagent: quote.Reagent, BundleCount: quote.MaxBundles}, request)
}
