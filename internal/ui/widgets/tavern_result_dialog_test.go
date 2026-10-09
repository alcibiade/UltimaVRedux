package widgets

import (
	stdcolor "image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	pkgcolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func TestTavernResultDialogRendersSemanticPurchaseSummary(t *testing.T) {
	setMerchantShellTestResolution(t, 320, 180)
	view := TavernResultView{
		Title: "Meal Served", Item: "Mutton for 3", Detail: "The party shares a hot meal.",
		ChargeLabel: "Meal", Charge: 12,
		ShowFood: true, FoodBefore: 40, FoodAfter: 43,
		ShowGold: true, GoldBefore: 100, GoldAfter: 83,
		Theft: 5, Note: "The table is cleared.", Tone: TavernResultSuccess,
	}
	dialog := NewTavernResultDialog(TavernResultDialogConfig{
		Keyboard:     newWidgetTestKeyboard(0),
		View:         view,
		IsKeyPressed: func(ebiten.Key) bool { return false },
	})

	body := compactTavernText(dialog.GetBodyText())
	lowBounds := dialog.GetBounds()
	require.Contains(t, body, "Mutton for 3")
	require.Contains(t, body, "FOOD 40 -> 43 (+3)")
	require.Contains(t, body, "MEAL 12 GOLD")
	require.Contains(t, body, "GOLD 100 -> 83 (-17)")
	require.Contains(t, body, "Faulinei took 5 additional gold.")
	require.Equal(t, tavernResultFooter, dialog.GetFooterGuidance())

	setResponsiveDialogTestResolution(t, 1280, 720)
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Dispose)
	require.NotPanics(t, func() { dialog.Draw(screen) })
	require.Greater(t, dialog.GetBounds().Dx(), lowBounds.Dx())
	require.Equal(t, body, compactTavernText(dialog.GetBodyText()))
	require.Equal(t, view, dialog.GetView())
	require.True(t, dialog.IsVisible())
	requireImageRegionContainsColor(t, screen, dialog.GetBodyBounds(), MerchantDialogBackgroundColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, pkgcolor.EGAYellow)
	requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, pkgcolor.EGALightGray)
	requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, pkgcolor.EGACyan)
	requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, pkgcolor.EGAWhite)
	requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, pkgcolor.EGALightGreen)
	requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, pkgcolor.EGARed)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlKeyColor)
	requireImageRegionContainsColor(t, screen, dialog.frame.footerText, DialogControlHintColor)
}

func TestTavernResultDialogUsesToneForTerminalStatus(t *testing.T) {
	setMerchantShellTestResolution(t, 1280, 720)
	for _, test := range []struct {
		name  string
		tone  TavernResultTone
		color stdcolor.Color
	}{
		{name: "success", tone: TavernResultSuccess, color: pkgcolor.EGALightGreen},
		{name: "caution", tone: TavernResultCaution, color: pkgcolor.EGAYellow},
		{name: "failure", tone: TavernResultFailure, color: pkgcolor.EGARed},
	} {
		t.Run(test.name, func(t *testing.T) {
			dialog := NewTavernResultDialog(TavernResultDialogConfig{
				Keyboard: newWidgetTestKeyboard(0),
				View: TavernResultView{
					Title: "Provision Result", Item: "Provisions", Note: "Final authority result.", Tone: test.tone,
				},
				IsKeyPressed: func(ebiten.Key) bool { return false },
			})
			screen := ebiten.NewImage(1280, 720)
			t.Cleanup(screen.Dispose)
			dialog.Draw(screen)
			requireImageRegionContainsColor(t, screen, dialog.frame.headerText, test.color)
			requireImageRegionContainsColor(t, screen, dialog.frame.bodyText, test.color)
		})
	}
}

func TestTavernResultDialogWaitsForReleaseAndExplicitAcknowledgement(t *testing.T) {
	held := true
	keyboard := newWidgetTestKeyboard(0)
	dialog := NewTavernResultDialog(TavernResultDialogConfig{
		Keyboard: keyboard,
		View:     TavernResultView{Title: "Meal Result", Item: "Mutton", Tone: TavernResultSuccess},
		IsKeyPressed: func(key ebiten.Key) bool {
			return held && key == ebiten.KeyEnter
		},
	})

	dialog.Update()
	require.True(t, dialog.IsVisible())
	require.False(t, dialog.IsComplete())
	held = false
	dialog.Update()
	require.True(t, dialog.IsVisible(), "release frame must not acknowledge the result")
	keyboard.SimulateKeyPress(ebiten.KeyEnter)
	dialog.Update()
	require.True(t, dialog.IsDismissed())
	require.True(t, dialog.IsComplete())
	require.False(t, dialog.IsVisible())
	dialog.ShowWithoutAnimation()
	require.False(t, dialog.IsVisible(), "acknowledged results cannot reopen")
}

func TestTavernResultDialogRejectsIncompleteConfiguration(t *testing.T) {
	require.Panics(t, func() {
		NewTavernResultDialog(TavernResultDialogConfig{
			Keyboard: newWidgetTestKeyboard(0), View: TavernResultView{Item: "Mutton", Tone: TavernResultSuccess},
		})
	})
	require.Panics(t, func() {
		NewTavernResultDialog(TavernResultDialogConfig{
			Keyboard: newWidgetTestKeyboard(0),
			View:     TavernResultView{Title: "Meal Result", Item: "Mutton", Tone: TavernResultTone(99)},
		})
	})
}
