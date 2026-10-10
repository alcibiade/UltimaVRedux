package widgets

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func TestShipDispatchDialogShowsRefreshedDeliveryAndAvailableEscort(t *testing.T) {
	state := testPendingFrigateShipShopState(true)
	dialog := NewShipDispatchDialog(ShipDispatchDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		State:    state,
	})
	body := compactShipText(dialog.GetBodyText())

	require.Equal(t, "Vessel Dispatched", compactShipText(dialog.GetTitleText()))
	require.Contains(t, body, "Harbour: Jhelom Harbour")
	require.Contains(t, body, "Berth: Pending arrival")
	require.Contains(t, body, "Vessel: Frigate")
	require.Contains(t, body, "Skiffs aboard: 2/12")
	require.Contains(t, body, "[D] Escort available, 3 minutes")
	require.Equal(t, "[Enter] Return to showroom   [D] Escort",
		dialog.GetFooterGuidance())
	require.Equal(t, state, dialog.GetState())
	require.Equal(t, FontSizeMainScreen, dialog.GetFontSize())

	visible := strings.ToLower(body + " " + dialog.GetFooterGuidance())
	for _, forbidden := range shipDialogForbiddenDetails() {
		require.NotContains(t, visible, forbidden)
	}
}

func TestShipDispatchDialogChoicesAreExclusiveAndTyped(t *testing.T) {
	state := testPendingFrigateShipShopState(true)

	continueKeyboard := newWidgetTestKeyboard(0)
	continueDialog := NewShipDispatchDialog(ShipDispatchDialogConfig{
		Keyboard: continueKeyboard,
		State:    state,
	})
	pressShipDispatchKey(continueDialog, continueKeyboard, ebiten.KeyEnter)
	require.True(t, continueDialog.IsDismissed())
	require.True(t, continueDialog.ShouldContinueShopping())
	require.False(t, continueDialog.HasPendingTravelIntent())

	escapeKeyboard := newWidgetTestKeyboard(0)
	escapeDialog := NewShipDispatchDialog(ShipDispatchDialogConfig{
		Keyboard: escapeKeyboard,
		State:    state,
	})
	pressShipDispatchKey(escapeDialog, escapeKeyboard, ebiten.KeyEscape)
	require.False(t, escapeDialog.IsDismissed())
	require.False(t, escapeDialog.ShouldContinueShopping())
	require.False(t, escapeDialog.HasPendingTravelIntent())

	travelKeyboard := newWidgetTestKeyboard(0)
	travelDialog := NewShipDispatchDialog(ShipDispatchDialogConfig{
		Keyboard: travelKeyboard,
		State:    state,
	})
	pressShipDispatchKey(travelDialog, travelKeyboard, ebiten.KeyD)
	require.True(t, travelDialog.IsDismissed())
	require.False(t, travelDialog.ShouldContinueShopping())
	intent, ok := travelDialog.ConsumePendingTravelIntent()
	require.True(t, ok)
	require.Equal(t, state.Travel, intent.Review)
	require.False(t, intent.Preview)
	pressShipDispatchKey(travelDialog, travelKeyboard, ebiten.KeyD)
	_, duplicate := travelDialog.ConsumePendingTravelIntent()
	require.False(t, duplicate)
}

func TestShipDispatchDialogUnavailableEscortCannotEmitTravel(t *testing.T) {
	state := testPendingFrigateShipShopState(false)
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewShipDispatchDialog(ShipDispatchDialogConfig{Keyboard: keyboard, State: state})

	require.Contains(t, compactShipText(dialog.GetBodyText()), "Escort: unavailable, 3 minutes")
	require.Equal(t, "[Enter] Return to showroom", dialog.GetFooterGuidance())
	pressShipDispatchKey(dialog, keyboard, ebiten.KeyD)
	require.False(t, dialog.IsDismissed())
	require.False(t, dialog.HasPendingTravelIntent())
	pressShipDispatchKey(dialog, keyboard, ebiten.KeyNumpadEnter)
	require.True(t, dialog.ShouldContinueShopping())
}

func TestShipDispatchDialogSemanticBackingResizeAndStrictState(t *testing.T) {
	setMerchantShellTestResolution(t, 320, 180)
	state := testPendingFrigateShipShopState(true)
	dialog := NewShipDispatchDialog(ShipDispatchDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		State:    state,
	})
	lowBounds := dialog.GetBounds()
	lowBody := compactShipText(dialog.GetBodyText())
	setMerchantShellTestResolution(t, 1280, 720)
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	dialog.Draw(screen)
	require.Greater(t, dialog.GetBounds().Dx(), lowBounds.Dx())
	require.Greater(t, dialog.GetBounds().Dy(), lowBounds.Dy())
	require.Equal(t, lowBody, compactShipText(dialog.GetBodyText()))
	requireImageRegionContainsColor(t, screen, dialog.GetBodyBounds(), MerchantDialogBackgroundColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlKeyColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlHintColor)

	require.Panics(t, func() {
		NewShipDispatchDialog(ShipDispatchDialogConfig{Keyboard: newWidgetTestKeyboard(0), State: testShipShopState()})
	})
	require.Panics(t, func() {
		NewShipDispatchDialog(ShipDispatchDialogConfig{Keyboard: nil, State: state})
	})

	skiffState := testPendingSkiffShipShopState()
	skiffDialog := NewShipDispatchDialog(ShipDispatchDialogConfig{
		Keyboard: newWidgetTestKeyboard(0),
		State:    skiffState,
	})
	require.Contains(t, compactShipText(skiffDialog.GetBodyText()), "Skiffs: 1 standalone")
	require.NotContains(t, skiffDialog.GetBodyText(), "/12")
	require.Equal(t, game_state.ShipBerthPendingSkiff, skiffDialog.GetState().Berth.State)
}

func pressShipDispatchKey(dialog *ShipDispatchDialog, keyboard *input.Keyboard, key ebiten.Key) {
	keyboard.SimulateKeyPress(key)
	dialog.Update()
}
