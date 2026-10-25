package widgets

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestShipPurchaseDialogDisplaysVesselDockSkiffsAndPayment(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	state := testShipShopState()
	intent := testShipPurchaseIntent(state, state.FrigateQuote, true)
	dialog := NewShipPurchaseDialog(ShipPurchaseDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Intent:   intent,
	})
	title := compactShipText(dialog.GetTitleText())
	body := compactShipText(dialog.GetBodyText())

	require.Equal(t, "PREVIEW - Purchase Frigate", title)
	require.Contains(t, body, "Vessel: Frigate")
	require.Contains(t, body, "Action: Dispatch new frigate")
	require.Contains(t, body, "Dock: Jhelom Harbour")
	require.Contains(t, body, "Included skiffs: 2")
	require.Contains(t, body, "Price: 600 gold")
	require.Contains(t, body, "Party gold: 1000 -> 400")
	require.NotContains(t, body, "Escort after purchase")
	require.Equal(t, "[Enter] Confirm   [Esc] Back", dialog.GetFooterGuidance())
	require.True(t, dialog.IsPreview())

	visible := strings.ToLower(strings.Join([]string{title, body, dialog.GetFooterGuidance()}, " "))
	for _, forbidden := range shipDialogForbiddenDetails() {
		require.NotContains(t, visible, forbidden)
	}
}

func TestShipPurchaseDialogLoadSkiffAndUnaffordableRemainConfirmable(t *testing.T) {
	state := testPendingFrigateShipShopState(true)
	dialog := NewShipPurchaseDialog(ShipPurchaseDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Intent:   testShipPurchaseIntent(state, state.SkiffQuote, false),
	})
	body := compactShipText(dialog.GetBodyText())
	require.Contains(t, body, "Action: Load skiffs aboard frigate")
	require.Equal(t, "Purchase Skiffs", compactShipText(dialog.GetTitleText()))
	require.Contains(t, body, "Skiffs aboard: 2/12")
	require.Contains(t, body, "Quantity: < 1 >")
	require.Contains(t, body, "If completed: 3/12")
	require.Contains(t, body, "Unit price: 200 gold")
	require.Contains(t, body, "Total: 200 gold")
	require.NotContains(t, body, "Escort after purchase")
	require.Equal(t, "[Left/Right] Quantity   [M] Max\n[Enter] Confirm   [Esc] Back",
		dialog.GetFooterGuidance())
	pressShipPurchaseKey(dialog, dialog.keyboard, ebiten.KeyRight)
	pressShipPurchaseKey(dialog, dialog.keyboard, ebiten.KeyRight)
	require.Equal(t, 3, dialog.GetQuantity())
	body = compactShipText(dialog.GetBodyText())
	require.Contains(t, body, "If completed: 5/12")
	require.Contains(t, body, "Total: 600 gold")
	pressShipPurchaseKey(dialog, dialog.keyboard, ebiten.KeyM)
	require.Equal(t, 5, dialog.GetQuantity(), "maximum is bounded by the party's current gold")

	poor := testShipShopState()
	setShipShopGold(&poor, 100)
	intent := testShipPurchaseIntent(poor, poor.FrigateQuote, false)
	keyboard := newWidgetTestKeyboard(0)
	dialog = NewShipPurchaseDialog(ShipPurchaseDialogConfig{Keyboard: keyboard, Intent: intent})
	body = compactShipText(dialog.GetBodyText())
	require.Contains(t, body, "Party gold: 100")
	require.Contains(t, body, "Need 600 gold.")
	require.NotContains(t, body, "100 ->")

	pressShipPurchaseKey(dialog, keyboard, ebiten.KeyEnter)
	require.True(t, dialog.IsDismissed())
	request, ok := dialog.ConsumePendingPurchaseRequest()
	require.True(t, ok)
	require.Equal(t, intent.Request, request)
}

func TestShipPurchaseDialogConfirmCancelAndSingleRequestLifecycle(t *testing.T) {
	intent := testShipPurchaseIntent(testShipShopState(), testShipShopState().SkiffQuote, false)

	backKeyboard := newWidgetTestKeyboard(0)
	back := NewShipPurchaseDialog(ShipPurchaseDialogConfig{Keyboard: backKeyboard, Intent: intent})
	pressShipPurchaseKey(back, backKeyboard, ebiten.KeyEscape)
	require.True(t, back.IsDismissed())
	require.True(t, back.IsCancelled())
	require.False(t, back.HasPendingPurchaseRequest())

	confirmKeyboard := newWidgetTestKeyboard(0)
	confirmed := NewShipPurchaseDialog(ShipPurchaseDialogConfig{Keyboard: confirmKeyboard, Intent: intent})
	pressShipPurchaseKey(confirmed, confirmKeyboard, ebiten.KeyNumpadEnter)
	require.True(t, confirmed.IsDismissed())
	require.False(t, confirmed.IsCancelled())
	request, ok := confirmed.ConsumePendingPurchaseRequest()
	require.True(t, ok)
	require.Equal(t, intent.Request, request)
	pressShipPurchaseKey(confirmed, confirmKeyboard, ebiten.KeyEnter)
	_, duplicate := confirmed.ConsumePendingPurchaseRequest()
	require.False(t, duplicate)
}

func TestShipPurchaseDialogSemanticBackingAndResizePreserveContent(t *testing.T) {
	setMerchantShellTestResolution(t, 320, 180)
	state := testPendingFrigateShipShopState(true)
	intent := testShipPurchaseIntent(state, state.SkiffQuote, true)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewShipPurchaseDialog(ShipPurchaseDialogConfig{
		Keyboard: keyboard,
		Intent:   intent,
	})
	pressShipPurchaseKey(dialog, keyboard, ebiten.KeyRight)
	pressShipPurchaseKey(dialog, keyboard, ebiten.KeyRight)
	lowBounds := dialog.GetBounds()
	lowBody := compactShipText(dialog.GetBodyText())

	setResponsiveDialogTestResolution(t, 1280, 720)
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)
	require.Greater(t, dialog.GetBounds().Dx(), lowBounds.Dx())
	require.Greater(t, dialog.GetBounds().Dy(), lowBounds.Dy())
	require.Equal(t, lowBody, compactShipText(dialog.GetBodyText()))
	require.Equal(t, intent, dialog.GetIntent())
	require.Equal(t, 3, dialog.GetQuantity())
	requireImageRegionContainsColor(t, screen, dialog.GetBodyBounds(), MerchantDialogBackgroundColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlKeyColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlHintColor)
	pressShipPurchaseKey(dialog, keyboard, ebiten.KeyEnter)
	request, ok := dialog.ConsumePendingPurchaseRequest()
	require.True(t, ok)
	require.Equal(t, 3, request.Quantity)
}

func TestShipPurchaseDialogStrictDetachedIntentContracts(t *testing.T) {
	state := testShipShopState()
	valid := ShipPurchaseDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		Intent:   testShipPurchaseIntent(state, state.FrigateQuote, false),
	}
	tests := []struct {
		name   string
		mutate func(*ShipPurchaseDialogConfig)
	}{
		{name: "keyboard", mutate: func(config *ShipPurchaseDialogConfig) { config.Keyboard = nil }},
		{name: "blocked", mutate: func(config *ShipPurchaseDialogConfig) {
			config.Intent.Quote.BlockReason = game_state.ShipPurchaseBlockBerthOccupied
		}},
		{name: "request price", mutate: func(config *ShipPurchaseDialogConfig) {
			config.Intent.Request.ReviewedPrice++
		}},
		{name: "request action", mutate: func(config *ShipPurchaseDialogConfig) {
			config.Intent.Request.Action = game_state.ShipPurchaseCreateStandaloneSkiff
		}},
		{name: "request fingerprint", mutate: func(config *ShipPurchaseDialogConfig) {
			config.Intent.Request.ReviewedFingerprint = "different"
		}},
		{name: "gold consequence", mutate: func(config *ShipPurchaseDialogConfig) {
			config.Intent.Quote.GoldAfterPayment++
		}},
		{name: "harbour", mutate: func(config *ShipPurchaseDialogConfig) {
			config.Intent.HarbourName = ""
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			require.Panics(t, func() { NewShipPurchaseDialog(config) })
		})
	}
}

func testShipPurchaseIntent(
	state game_state.ShipShopState,
	quote game_state.ShipPurchaseQuote,
	preview bool,
) ShipPurchaseIntent {
	return ShipPurchaseIntent{
		Quote:       quote,
		Request:     shipPurchaseRequestFromState(state, quote),
		HarbourName: state.Travel.HarbourName,
		Travel:      state.Travel,
		Preview:     preview,
	}
}

func pressShipPurchaseKey(dialog *ShipPurchaseDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}

func TestShipPurchaseDialogPhysicalFixedVesselsKeepReviewedRequest(t *testing.T) {
	state := testShipShopState()
	setShipShopGold(&state, 100)
	for _, quote := range []game_state.ShipPurchaseQuote{state.FrigateQuote, state.SkiffQuote} {
		t.Run(shipActionLabel(quote.Action), func(t *testing.T) {
			keys := newPhysicalQuantityKeys()
			intent := testShipPurchaseIntent(state, quote, false)
			dialog := NewShipPurchaseDialog(ShipPurchaseDialogConfig{Keyboard: keys.keyboard, Intent: intent})
			require.False(t, dialog.GetApprovalView().Affordable)
			for _, key := range []ebiten.Key{ebiten.KeyRight, ebiten.KeyLeft, ebiten.KeyM} {
				keys.held[key] = true
				for range 60 {
					keys.tick(dialog)
				}
				require.Equal(t, 1, dialog.GetQuantity())
				keys.release(dialog)
			}
			keys.held[ebiten.KeyEnter] = true
			keys.tick(dialog)
			request, ok := dialog.ConsumePendingPurchaseRequest()
			require.True(t, ok)
			require.Equal(t, intent.Request, request, "fixed unaffordable attempts still reach authority")
		})
	}
}
